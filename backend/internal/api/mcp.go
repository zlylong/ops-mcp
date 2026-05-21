package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/domain"
)

const mcpProtocolVersion = "2024-11-05"

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}
type mcpResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *mcpError `json:"error,omitempty"`
}
type mcpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}
type mcpCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

func (s *Server) mcp(c *gin.Context) {
	if c.Request.Method == http.MethodGet {
		c.JSON(http.StatusOK, gin.H{"name": "darwin-ops-mcp", "protocolVersion": mcpProtocolVersion, "endpoint": "/mcp", "methods": []string{"initialize", "tools/list", "tools/call", "ping"}})
		return
	}
	var req mcpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, mcpFailure(nil, -32700, "parse error", err.Error()))
		return
	}
	if req.JSONRPC != "2.0" {
		c.JSON(http.StatusOK, mcpFailure(req.ID, -32600, "invalid request", "jsonrpc must be 2.0"))
		return
	}
	switch req.Method {
	case "initialize":
		c.JSON(http.StatusOK, mcpSuccess(req.ID, gin.H{"protocolVersion": mcpProtocolVersion, "capabilities": gin.H{"tools": gin.H{"listChanged": false}}, "serverInfo": gin.H{"name": "darwin-ops-mcp", "version": "1.0.0"}}))
	case "notifications/initialized":
		c.Status(http.StatusAccepted)
	case "ping":
		c.JSON(http.StatusOK, mcpSuccess(req.ID, gin.H{}))
	case "tools/list":
		c.JSON(http.StatusOK, mcpSuccess(req.ID, gin.H{"tools": s.mcpTools()}))
	case "tools/call":
		s.handleMCPToolCall(c, req)
	default:
		c.JSON(http.StatusOK, mcpFailure(req.ID, -32601, "method not found", req.Method))
	}
}

func (s *Server) handleMCPToolCall(c *gin.Context, req mcpRequest) {
	var params mcpCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		c.JSON(http.StatusOK, mcpFailure(req.ID, -32602, "invalid params", err.Error()))
		return
	}
	params.Name = strings.TrimSpace(params.Name)
	if params.Name == "" {
		c.JSON(http.StatusOK, mcpFailure(req.ID, -32602, "invalid params", "tool name is required"))
		return
	}
	result, status, err := s.registry.Execute(c.Request.Context(), params.Name, mcpExecuteRequest(params.Arguments))
	payload := gin.H{"httpStatus": status, "result": result}
	if err != nil {
		payload["error"] = err.Error()
	}
	text, _ := json.Marshal(payload)
	c.JSON(http.StatusOK, mcpSuccess(req.ID, gin.H{"content": []gin.H{{"type": "text", "text": string(text)}}, "structuredContent": payload, "isError": status >= 400}))
}

func mcpExecuteRequest(args map[string]any) domain.ExecuteRequest {
	if args == nil {
		args = map[string]any{}
	}
	actor := stringArg(args, "actor", "external-agent")
	role := domain.Role(stringArg(args, "role", string(domain.RoleViewer)))
	target := stringArg(args, "target", "")
	approved := boolArg(args, "approved", false)
	parameters := map[string]any{}
	if nested, ok := args["parameters"].(map[string]any); ok {
		parameters = nested
	} else {
		for k, v := range args {
			switch k {
			case "actor", "role", "target", "approved":
				continue
			default:
				parameters[k] = v
			}
		}
	}
	return domain.ExecuteRequest{Actor: actor, Role: role, Target: target, Approved: approved, Parameters: parameters}
}
func stringArg(args map[string]any, key, fallback string) string {
	if v, ok := args[key].(string); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}
func boolArg(args map[string]any, key string, fallback bool) bool {
	if v, ok := args[key].(bool); ok {
		return v
	}
	return fallback
}
func (s *Server) mcpTools() []gin.H {
	tools := s.registry.List()
	out := make([]gin.H, 0, len(tools))
	for _, tool := range tools {
		out = append(out, gin.H{"name": tool.Name, "description": mcpToolDescription(tool), "inputSchema": mcpInputSchema(tool)})
	}
	return out
}
func mcpToolDescription(tool domain.Tool) string {
	parts := []string{tool.Description, "category=" + tool.Category, "risk=" + string(tool.Risk), fmt.Sprintf("readOnly=%t", tool.ReadOnly)}
	if tool.RequiresApproval {
		parts = append(parts, "requiresApproval=true")
	}
	return strings.TrimSpace(strings.Join(parts, "; "))
}
func mcpInputSchema(tool domain.Tool) gin.H {
	properties := gin.H{"actor": gin.H{"type": "string", "description": "Actor name used for audit records", "default": "external-agent"}, "role": gin.H{"type": "string", "description": "Policy role", "enum": []string{"viewer", "operator", "admin"}, "default": "viewer"}, "target": gin.H{"type": "string", "description": "Human-readable target, for example host=192.168.20.166"}, "approved": gin.H{"type": "boolean", "description": "Set true only after an external approval decision", "default": false}}
	required := []string{}
	for name, schema := range tool.InputSchema {
		prop := gin.H{"type": mcpJSONSchemaType(schema.Type), "description": schema.Description}
		if schema.Default != nil {
			prop["default"] = schema.Default
		}
		properties[name] = prop
		if schema.Required {
			required = append(required, name)
		}
	}
	return gin.H{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
}
func mcpJSONSchemaType(paramType string) string {
	switch paramType {
	case "number":
		return "number"
	case "boolean":
		return "boolean"
	default:
		return "string"
	}
}
func mcpSuccess(id any, result any) mcpResponse {
	return mcpResponse{JSONRPC: "2.0", ID: id, Result: result}
}
func mcpFailure(id any, code int, message string, data any) mcpResponse {
	return mcpResponse{JSONRPC: "2.0", ID: id, Error: &mcpError{Code: code, Message: message, Data: data}}
}
