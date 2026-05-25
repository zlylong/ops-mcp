package api

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zlylong/darwin-ops-mcp/backend/internal/app"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/config"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/domain"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/policy"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/storage"
)

// createTestRegistryForAuth is a local alias to avoid conflict with router_test.go.
func createTestRegistryForAuth() *app.Registry {
	reg := app.NewRegistry(
		policy.NewEngine(),
		&mockRecorder{},
		storage.NewExecutionStore(),
		storage.NewApprovalStore(),
		storage.NewUserStore(),
		storage.NewJumpServerStore(),
		domain.EnvDevelopment,
	)
	// Register a read-only low-risk tool for identity-override tests.
	reg.Register(domain.Tool{Name: "test.tool", ReadOnly: true, Risk: domain.RiskLow}, func(ctx context.Context, params map[string]any) (map[string]any, error) {
		return map[string]any{"result": "ok"}, nil
	})
	return reg
}

// TestHandleMCPToolCall_IgnoresCallerRole verifies that the caller's "role"
// field in the JSON-RPC params is completely ignored and the server-side
// identity (from agent API key or user token) is used instead.
func TestHandleMCPToolCall_IgnoresCallerRole(t *testing.T) {
	cfg := config.Config{APIToken: "master-token"}
	reg := createTestRegistryForAuth()

	created, err := reg.CreateAgentAPIKey(domain.AgentAPIKeyCreateRequest{
		Name:  "test-agent",
		Actor: "test-agent-actor",
		Role:  domain.RoleViewer,
	})
	require.NoError(t, err)

	r := NewRouter(cfg, reg, &mockRecorder{}, slog.Default())

	// Caller claims role=admin (privilege escalation attempt)
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "test.tool",
			"arguments": map[string]any{
				"role":   "admin",    // caller-supplied — must be ignored
				"actor":  "attacker", // caller-supplied — must be ignored
				"target": "critical-system",
			},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+created.Secret)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	executions := reg.Executions()
	require.NotEmpty(t, executions)
	last := executions[len(executions)-1]
	assert.Equal(t, "test-agent-actor", last.Actor, "actor must be resolved from agent key, not from caller-supplied value")
	assert.Equal(t, domain.RoleViewer, last.Role, "role must be resolved from agent key, not from caller-supplied 'admin'")
}

// TestHandleMCPToolCall_ApprovedAlwaysFalse verifies that the approved flag
// is always forced to false regardless of what the caller passes.
func TestHandleMCPToolCall_ApprovedAlwaysFalse(t *testing.T) {
	cfg := config.Config{APIToken: "master-token"}
	reg := createTestRegistryForAuth()

	_, err := reg.CreateAgentAPIKey(domain.AgentAPIKeyCreateRequest{
		Name:  "test-agent",
		Actor: "test-agent",
		Role:  domain.RoleAdmin,
	})
	require.NoError(t, err)

	// Register a tool that requires approval so we can observe the approval flow.
	reg.Register(domain.Tool{
		Name:             "approval.test",
		RequiresApproval: true,
		Risk:             domain.RiskHigh,
		ReadOnly:         false,
	}, func(ctx context.Context, params map[string]any) (map[string]any, error) {
		return map[string]any{"done": true}, nil
	})

	r := NewRouter(cfg, reg, &mockRecorder{}, slog.Default())

	// Caller explicitly sets approved=true (bypass attempt)
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "approval.test",
			"arguments": map[string]any{
				"role":     "admin",
				"approved": true, // caller-supplied — must be ignored
				"target":   "sensitive",
			},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer master-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	// Must NOT execute immediately; must create an approval record instead.
	executions := reg.Executions()
	assert.NotEmpty(t, executions, "approved=true from caller must be ignored; tool requires approval")
}

// TestHandleMCPToolCall_UnknownMethod returns method-not-found error.
func TestHandleMCPToolCall_UnknownMethod(t *testing.T) {
	cfg := config.Config{APIToken: "master-token"}
	r := NewRouter(cfg, createTestRegistry(t), &mockRecorder{}, slog.Default())

	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/execute", // wrong method name
		"params":  map[string]any{},
	})
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer master-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp mcpResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, -32601, resp.Error.Code) // method not found
}

// TestHandleMCPToolCall_InvalidJSON returns parse error.
func TestHandleMCPToolCall_InvalidJSON(t *testing.T) {
	cfg := config.Config{APIToken: "master-token"}
	r := NewRouter(cfg, createTestRegistry(t), &mockRecorder{}, slog.Default())

	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader([]byte("not json at all")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer master-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp mcpResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, -32700, resp.Error.Code) // parse error
}

// TestMCPPing returns empty result.
func TestMCPPing(t *testing.T) {
	cfg := config.Config{APIToken: "master-token"}
	r := NewRouter(cfg, createTestRegistry(t), &mockRecorder{}, slog.Default())

	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "ping"})
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer master-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp mcpResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Nil(t, resp.Error)
}

// TestMCPNotificationsInitialized returns 202 Accepted with no body.
func TestMCPNotificationsInitialized(t *testing.T) {
	cfg := config.Config{APIToken: "master-token"}
	r := NewRouter(cfg, createTestRegistry(t), &mockRecorder{}, slog.Default())

	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"})
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer master-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusAccepted, w.Code)
}

// TestMCPToolsCall_MissingName returns invalid params.
func TestMCPToolsCall_MissingName(t *testing.T) {
	cfg := config.Config{APIToken: "master-token"}
	r := NewRouter(cfg, createTestRegistry(t), &mockRecorder{}, slog.Default())

	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params":  map[string]any{"name": "", "arguments": map[string]any{}},
	})
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer master-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp mcpResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, -32602, resp.Error.Code) // invalid params
}

// TestExecuteTool_IgnoresCallerIdentity verifies that executeTool also
// overrides role/actor from the caller's request body using server-side identity.
func TestExecuteTool_IgnoresCallerIdentity(t *testing.T) {
	cfg := config.Config{APIToken: "master-token"}
	reg := createTestRegistryForAuth()

	created, err := reg.CreateAgentAPIKey(domain.AgentAPIKeyCreateRequest{
		Name:  "exec-agent",
		Actor: "exec-agent-actor",
		Role:  domain.RoleViewer,
	})
	require.NoError(t, err)

	r := NewRouter(cfg, reg, &mockRecorder{}, slog.Default())

	body, _ := json.Marshal(map[string]any{
		"actor":  "attacker-impersonator",
		"role":   "admin",
		"target": "critical",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/test.tool/execute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+created.Secret)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	executions := reg.Executions()
	require.NotEmpty(t, executions)
	last := executions[len(executions)-1]
	assert.Equal(t, "exec-agent-actor", last.Actor, "actor must be from agent key, not from HTTP body")
	assert.Equal(t, domain.RoleViewer, last.Role, "role must be from agent key, not from HTTP body 'admin'")
}
