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

	"github.com/zlylong/darwin-ops-mcp/backend/internal/config"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/domain"
)

func TestAgentAPIKeyLifecycle(t *testing.T) {
	cfg := config.Config{APIToken: "master-token"}
	registry := createTestRegistry()
	r := NewRouter(cfg, registry, &mockRecorder{}, slog.Default())

	createBody := map[string]any{
		"name":         "opsagent topic 436",
		"actor":        "opsagent-topic-436",
		"role":         "viewer",
		"reason":       "least-privilege read-only automation",
		"scopes":       []string{"tools:execute", "applications:create"},
		"expiresInHrs": 24,
	}
	body, _ := json.Marshal(createBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent-keys", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer master-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created domain.AgentAPIKeyCreateResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.NotEmpty(t, created.ID)
	assert.NotEmpty(t, created.Secret)
	assert.Contains(t, created.Secret, "domcp_")
	assert.Equal(t, "active", created.Status)
	assert.Equal(t, domain.RoleViewer, created.Role)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/tools", nil)
	req.Header.Set("Authorization", "Bearer "+created.Secret)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/agent-keys", nil)
	req.Header.Set("Authorization", "Bearer "+created.Secret)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/agent-keys", nil)
	req.Header.Set("Authorization", "Bearer master-token")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), created.ID)
	assert.NotContains(t, w.Body.String(), created.Secret)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/agent-keys/"+created.ID+"/revoke", nil)
	req.Header.Set("Authorization", "Bearer master-token")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "revoked")

	req = httptest.NewRequest(http.MethodGet, "/api/v1/tools", nil)
	req.Header.Set("Authorization", "Bearer "+created.Secret)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAgentAPIKeyActorFallback(t *testing.T) {
	cfg := config.Config{APIToken: "master-token"}
	registry := createTestRegistry()
	created, err := registry.CreateAgentAPIKey(domain.AgentAPIKeyCreateRequest{Name: "agent", Actor: "agent-key-actor", Role: domain.RoleViewer})
	require.NoError(t, err)
	r := NewRouter(cfg, registry, &mockRecorder{}, slog.Default())

	executeBody := map[string]any{"role": "viewer", "target": "local", "parameters": map[string]any{}}
	body, _ := json.Marshal(executeBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/test.tool/execute", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+created.Secret)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	executions := registry.Executions()
	require.NotEmpty(t, executions)
	assert.Equal(t, "agent-key-actor", executions[0].Actor)
}

func TestAgentAPIKeyValidation(t *testing.T) {
	cfg := config.Config{APIToken: "master-token"}
	r := NewRouter(cfg, createTestRegistry(), &mockRecorder{}, slog.Default())

	body, _ := json.Marshal(map[string]any{"name": "missing actor", "role": "viewer"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent-keys", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer master-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	body, _ = json.Marshal(map[string]any{"name": "agent", "actor": "agent", "role": "superuser"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/agent-keys", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer master-token")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
