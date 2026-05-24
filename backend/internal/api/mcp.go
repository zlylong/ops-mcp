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

// handleMCPToolCall dispatches an MCP tools/call request.
//
// SECURITY NOTES:
// - role: always resolved server-side from the authenticated identity (agent key
//   or user token). The caller's args.role is IGNORED to prevent privilege escalation.
// - approved: always forced to false here. The only valid approval path is through
//   registry.Approve() which re-dispatches the tool internally with approved=true.
// - actor: likewise taken from the authenticated agent key, not from args.actor.
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
	execReq, actor := mcpExecuteRequest(params.Arguments)

	// Resolve authoritative role and actor from the authenticated identity.
	if agentKey, ok := authenticatedAgent(c); ok {
		execReq.Role = agentKey.Role
		execReq.Actor = agentKey.Actor
		actor = agentKey.Actor
	} else if uid, ok := c.Get(authUserID); ok {
		if user, found := s.registry.Users().Get(uid.(string)); found {
			execReq.Role = user.Role
			execReq.Actor = actor
		}
	}

	// SECURITY: approved is always false; the only valid approval path is
	// registry.Approve() which re-executes the tool internally.
	execReq.Approved = false

	result, status, err := s.registry.Execute(c.Request.Context(), params.Name, execReq)
	payload := gin.H{"httpStatus": status, "result": result}
	if err != nil {
		payload["error"] = err.Error()
	}
	text, _ := json.Marshal(payload)
	c.JSON(http.StatusOK, mcpSuccess(req.ID, gin.H{"content": []gin.H{{"type": "text", "text": string(text)}}, "structuredContent": payload, "isError": status >= 400}))
}

// mcpExecuteRequest extracts tool execution parameters from the MCP arguments map.
// It returns an ExecuteRequest and the raw actor string for further resolution.
//
// NOTE: The role and approved fields returned here are defaults that are ALWAYS
// over-ridden in handleMCPToolCall after server-side identity resolution.
// Callers MUST NOT rely on these fields having any effect.
func mcpExecuteRequest(args map[string]any) (domain.ExecuteRequest, string) {
	if args == nil {
		args = map[string]any{}
	}
	actor := stringArg(args, "actor", "external-agent")
	parameters := map[string]any{}
	if nested, ok := args["parameters"].(map[string]any); ok {
		parameters = nested
	} else {
		for k, v := range args {
			if k == "actor" || k == "role" || k == "target" || k == "approved" {
				continue
			}
			parameters[k] = v
		}
	}
	return domain.ExecuteRequest{
		Actor:      actor,
		Role:       domain.RoleViewer, // OVER-RIDDEN server-side; caller role is ignored
		Target:     stringArg(args, "target", ""),
		Approved:   false, // OVER-RIDDEN server-side; caller approval flag is ignored
		Parameters: parameters,
	}, actor
}

func stringArg(args map[string]any, key, fallback string) string {
	if v, ok := args[key].(string); ok && strings.TrimSpace(v) != "" {
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

// mcpInputSchema returns the JSON Schema for a tool input.
// Note: role and approved are NOT enumerated in the schema because they
// are server-side decisions, not caller-supplied parameters.
func mcpInputSchema(tool domain.Tool) gin.H {
	properties := gin.H{
		"actor":  gin.H{"type": "string", "description": "Actor name for audit records", "default": "external-agent"},
		"target": gin.H{"type": "string", "description": "Human-readable target (e.g. host=192.168.20.166)"},
	}
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
