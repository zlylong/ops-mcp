package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zlylong/darwin-ops-mcp/backend/internal/config"
)

// ------------------------------------------------------------------
// createAgentAPIKey
// ------------------------------------------------------------------

func TestCreateAgentAPIKey_ValidRequest_ReturnsKey(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{APIToken: "master-token"}
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	body := jsonBody(map[string]any{
		"name":         "opsagent topic 436",
		"actor":        "opsagent-topic-436",
		"role":         "viewer",
		"reason":       "least-privilege read-only automation",
		"scopes":       []string{"tools:execute", "applications:create"},
		"expiresInHrs": 24,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent-keys", body)
	req.Header.Set("Authorization", "Bearer master-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var key map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &key))
	assert.NotEmpty(t, key["id"])
	assert.NotEmpty(t, key["key"])
	assert.Equal(t, "opsagent topic 436", key["name"])
	assert.Equal(t, "opsagent-topic-436", key["actor"])
	assert.Equal(t, "viewer", key["role"])
}

func TestCreateAgentAPIKey_NoMasterToken_ReturnsForbidden(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{APIToken: "master-token"}
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	body := jsonBody(map[string]any{"name": "test", "actor": "test", "role": "viewer", "scopes": []string{}, "expiresInHrs": 24})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent-keys", body)
	req.Header.Set("Authorization", "Bearer wrong-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateAgentAPIKey_EmptyBody_ReturnsBadRequest(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{APIToken: "master-token"}
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent-keys", bytes.NewReader([]byte("{}")))
	req.Header.Set("Authorization", "Bearer master-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	var errResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Contains(t, errResp["error"], "name is required")
}

// ------------------------------------------------------------------
// listAgentAPIKeys
// ------------------------------------------------------------------

func TestListAgentAPIKeys_MasterToken_ReturnsKeys(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{APIToken: "master-token"}
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	// First create a key
	createBody := jsonBody(map[string]any{
		"name": "test-key", "actor": "test-actor", "role": "viewer",
		"scopes": []string{"tools:execute"}, "expiresInHrs": 24,
	})
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/agent-keys", createBody)
	createReq.Header.Set("Authorization", "Bearer master-token")
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	router.ServeHTTP(createW, createReq)
	require.Equal(t, http.StatusCreated, createW.Code)

	// Then list
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/agent-keys", nil)
	listReq.Header.Set("Authorization", "Bearer master-token")
	listW := httptest.NewRecorder()
	router.ServeHTTP(listW, listReq)

	require.Equal(t, http.StatusOK, listW.Code)
	var keys []any
	require.NoError(t, json.Unmarshal(listW.Body.Bytes(), &keys))
	assert.GreaterOrEqual(t, len(keys), 1)
}

func TestListAgentAPIKeys_NoAuth_ReturnsUnauthorized(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{}
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent-keys", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ------------------------------------------------------------------
// revokeAgentAPIKey
// ------------------------------------------------------------------

func TestRevokeAgentAPIKey_ValidKey_Revokes(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{APIToken: "master-token"}
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	// Create a key first
	createBody := jsonBody(map[string]any{
		"name": "revoke-me", "actor": "actor-revoke", "role": "viewer",
		"scopes": []string{"tools:execute"}, "expiresInHrs": 24,
	})
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/agent-keys", createBody)
	createReq.Header.Set("Authorization", "Bearer master-token")
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	router.ServeHTTP(createW, createReq)
	require.Equal(t, http.StatusCreated, createW.Code)

	var created map[string]any
	json.Unmarshal(createW.Body.Bytes(), &created)
	keyID := created["id"].(string)

	// Revoke it
	revokeReq := httptest.NewRequest(http.MethodPost, "/api/v1/agent-keys/"+keyID+"/revoke", nil)
	revokeReq.Header.Set("Authorization", "Bearer master-token")
	revokeW := httptest.NewRecorder()
	router.ServeHTTP(revokeW, revokeReq)

	require.Equal(t, http.StatusOK, revokeW.Code, revokeW.Body.String())
	var revoked map[string]any
	require.NoError(t, json.Unmarshal(revokeW.Body.Bytes(), &revoked))
	assert.Equal(t, keyID, revoked["id"])
}

func TestRevokeAgentAPIKey_NotFound_Returns404(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{APIToken: "master-token"}
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent-keys/nonexistent-key-id/revoke", nil)
	req.Header.Set("Authorization", "Bearer master-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRevokeAgentAPIKey_NoAuth_ReturnsUnauthorized(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{}
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent-keys/some-key/revoke", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ------------------------------------------------------------------
// helper
// ------------------------------------------------------------------

func jsonBody(v any) *bytes.Reader {
	b, _ := json.Marshal(v)
	return bytes.NewReader(b)
}
