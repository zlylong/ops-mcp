package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gin-gonic/gin"

	"github.com/zlylong/darwin-ops-mcp/backend/internal/config"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/domain"
)

// ------------------------------------------------------------------
// GET /mcp — capability discovery
// ------------------------------------------------------------------

func TestMCP_Get_Discovery(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{APIToken: "test-agent-token"}
	router := NewRouter(cfg, r, &mockRecorder{}, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer test-agent-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "darwin-ops-mcp", resp["name"])
	assert.Equal(t, "2024-11-05", resp["protocolVersion"])
}

func TestMCP_Get_Unauthenticated(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{APIToken: "test-agent-token"}
	router := NewRouter(cfg, r, &mockRecorder{}, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ------------------------------------------------------------------
// POST /mcp — jsonrpc dispatch
// ------------------------------------------------------------------

func TestMCP_POST_Initialize(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{APIToken: "test-agent-token"}
	router := NewRouter(cfg, r, &mockRecorder{}, slog.Default())

	body := mcpJSONBody(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]any{},
	})
	req := httptest.NewRequest(http.MethodPost, "/mcp", body)
	req.Header.Set("Authorization", "Bearer test-agent-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp mcpResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Nil(t, resp.Error)
	require.NotNil(t, resp.Result)
	resultMap := resp.Result.(map[string]any)
	assert.Equal(t, "2024-11-05", resultMap["protocolVersion"])
}

func TestMCP_POST_NotificationsInitialized(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{APIToken: "test-agent-token"}
	router := NewRouter(cfg, r, &mockRecorder{}, slog.Default())

	body := mcpJSONBody(map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
		"params":  map[string]any{},
	})
	req := httptest.NewRequest(http.MethodPost, "/mcp", body)
	req.Header.Set("Authorization", "Bearer test-agent-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusAccepted, w.Code)
}

func TestMCP_POST_Ping(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{APIToken: "test-agent-token"}
	router := NewRouter(cfg, r, &mockRecorder{}, slog.Default())

	body := mcpJSONBody(map[string]any{"jsonrpc": "2.0", "id": 2, "method": "ping"})
	req := httptest.NewRequest(http.MethodPost, "/mcp", body)
	req.Header.Set("Authorization", "Bearer test-agent-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp mcpResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Nil(t, resp.Error)
}

func TestMCP_POST_ToolsList(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{APIToken: "test-agent-token"}
	router := NewRouter(cfg, r, &mockRecorder{}, slog.Default())

	body := mcpJSONBody(map[string]any{"jsonrpc": "2.0", "id": 3, "method": "tools/list"})
	req := httptest.NewRequest(http.MethodPost, "/mcp", body)
	req.Header.Set("Authorization", "Bearer test-agent-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp mcpResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Result)
	resultMap := resp.Result.(map[string]any)
	tools := resultMap["tools"].([]any)
	assert.GreaterOrEqual(t, len(tools), 0)
}

func TestMCP_POST_UnknownMethod(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{APIToken: "test-agent-token"}
	router := NewRouter(cfg, r, &mockRecorder{}, slog.Default())

	body := mcpJSONBody(map[string]any{"jsonrpc": "2.0", "id": 4, "method": "tools/execute"})
	req := httptest.NewRequest(http.MethodPost, "/mcp", body)
	req.Header.Set("Authorization", "Bearer test-agent-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp mcpResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, -32601, resp.Error.Code)
}

func TestMCP_POST_InvalidJSONRPCVersion(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{APIToken: "test-agent-token"}
	router := NewRouter(cfg, r, &mockRecorder{}, slog.Default())

	body := mcpJSONBody(map[string]any{"jsonrpc": "1.5", "id": 5, "method": "ping"})
	req := httptest.NewRequest(http.MethodPost, "/mcp", body)
	req.Header.Set("Authorization", "Bearer test-agent-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp mcpResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, -32600, resp.Error.Code)
	assert.Contains(t, resp.Error.Data, "jsonrpc must be 2.0")
}

func TestMCP_POST_MalformedJSON(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{APIToken: "test-agent-token"}
	router := NewRouter(cfg, r, &mockRecorder{}, slog.Default())

	body := bytes.NewReader([]byte("not json at all"))
	req := httptest.NewRequest(http.MethodPost, "/mcp", body)
	req.Header.Set("Authorization", "Bearer test-agent-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp mcpResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	require.NotNil(t, resp.Error)
	assert.Equal(t, -32700, resp.Error.Code)
}

func TestMCP_POST_InvalidParams_MissingToolName(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{APIToken: "test-agent-token"}
	router := NewRouter(cfg, r, &mockRecorder{}, slog.Default())

	body := mcpJSONBody(map[string]any{
		"jsonrpc": "2.0",
		"id":      6,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "",
			"arguments": map[string]any{},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/mcp", body)
	req.Header.Set("Authorization", "Bearer test-agent-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp mcpResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, -32602, resp.Error.Code)
	assert.Contains(t, resp.Error.Data, "tool name is required")
}

// ------------------------------------------------------------------
// mcpExecuteRequest unit tests
// ------------------------------------------------------------------

func TestMCPTools_InputSchemaTypes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"string", "string"},
		{"number", "number"},
		{"boolean", "boolean"},
		{"unknown", "string"},
	}
	for _, tt := range tests {
		result := mcpJSONSchemaType(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

func TestMCPTools_ToolDescription(t *testing.T) {
	tool := domain.Tool{
		Name:        "ping",
		Description: "Ping a host",
		Category:    "network",
		Risk:        domain.RiskLow,
		ReadOnly:    true,
	}
	desc := mcpToolDescription(tool)
	assert.Contains(t, desc, "Ping a host")
	assert.Contains(t, desc, "category=network")
	assert.Contains(t, desc, "risk=low")
	assert.Contains(t, desc, "readOnly=true")
}

func TestMCPTools_ToolDescription_WithApprovalRequired(t *testing.T) {
	tool := domain.Tool{
		Name:             "execute_command",
		Description:      "Run a shell command",
		Category:         "exec",
		Risk:             domain.RiskHigh,
		ReadOnly:         false,
		RequiresApproval: true,
	}
	desc := mcpToolDescription(tool)
	assert.Contains(t, desc, "requiresApproval=true")
}

func TestMCPTools_StringArg(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]any
		key      string
		fallback string
		expected string
	}{
		{"present", map[string]any{"key": "value"}, "key", "fallback", "value"},
		{"whitespace preserved", map[string]any{"key": "  spaced  "}, "key", "fallback", "  spaced  "},
		{"missing", map[string]any{}, "key", "fallback", "fallback"},
		{"empty string", map[string]any{"key": "   "}, "key", "fallback", "fallback"},
		{"not a string", map[string]any{"key": 123}, "key", "fallback", "fallback"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stringArg(tt.args, tt.key, tt.fallback)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMCPTools_ExecuteRequestParsing(t *testing.T) {
	args := map[string]any{
		"actor":    "test-actor",
		"role":     "admin",
		"target":   "server1",
		"host":     "192.168.1.1",
		"count":    5,
		"approved": true,
	}
	req, actor := mcpExecuteRequest(args)
	assert.Equal(t, "test-actor", actor)
	assert.Equal(t, domain.RoleViewer, req.Role)
	assert.False(t, req.Approved)
	assert.Equal(t, "server1", req.Target)
	assert.Equal(t, 5, req.Parameters["count"])
	assert.Equal(t, "192.168.1.1", req.Parameters["host"])
	_, hasActor := req.Parameters["actor"]
	_, hasRole := req.Parameters["role"]
	_, hasApproved := req.Parameters["approved"]
	assert.False(t, hasActor)
	assert.False(t, hasRole)
	assert.False(t, hasApproved)
}

func TestMCPTools_ExecuteRequestParsing_NestedParameters(t *testing.T) {
	args := map[string]any{
		"actor":      "test-actor",
		"role":       "admin",
		"target":     "server1",
		"approved":   true,
		"parameters": map[string]any{"host": "10.0.0.1", "port": 22, "count": 3},
	}
	req, _ := mcpExecuteRequest(args)
	assert.Equal(t, "10.0.0.1", req.Parameters["host"])
	assert.Equal(t, 22, req.Parameters["port"])
	assert.Equal(t, 3, req.Parameters["count"])
}

func TestMCPTools_ExecuteRequestParsing_NilArgs(t *testing.T) {
	req, actor := mcpExecuteRequest(nil)
	assert.Equal(t, "external-agent", actor)
	assert.Equal(t, domain.RoleViewer, req.Role)
	assert.False(t, req.Approved)
	assert.Empty(t, req.Target)
}

// ------------------------------------------------------------------
// mcpSuccess / mcpFailure helpers
// ------------------------------------------------------------------

func TestMCPTools_MCPSuccess(t *testing.T) {
	resp := mcpSuccess(1, gin.H{"msg": "ok"})
	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Equal(t, 1, resp.ID)
	assert.NotNil(t, resp.Result)
	assert.Nil(t, resp.Error)
}

func TestMCPTools_MCPFailure(t *testing.T) {
	resp := mcpFailure(1, -32600, "bad request", "invalid version")
	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Equal(t, 1, resp.ID)
	assert.Nil(t, resp.Result)
	require.NotNil(t, resp.Error)
	assert.Equal(t, -32600, resp.Error.Code)
	assert.Equal(t, "bad request", resp.Error.Message)
	assert.Equal(t, "invalid version", resp.Error.Data)
}

// ------------------------------------------------------------------
// helper
// ------------------------------------------------------------------

func mcpJSONBody(v any) *bytes.Reader {
	b, _ := json.Marshal(v)
	return bytes.NewReader(b)
}
