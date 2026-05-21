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
)

func TestMCPInitializeAndToolsList(t *testing.T) {
	r := NewRouter(config.Config{}, createTestRegistry(), &mockRecorder{}, slog.Default())

	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "darwin-ops-mcp")
	assert.Contains(t, w.Body.String(), "tools")

	body = []byte(`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`)
	req = httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "test.tool")
	assert.Contains(t, w.Body.String(), "inputSchema")
}

func TestMCPToolsCall(t *testing.T) {
	r := NewRouter(config.Config{}, createTestRegistry(), &mockRecorder{}, slog.Default())
	body := []byte(`{"jsonrpc":"2.0","id":"call-1","method":"tools/call","params":{"name":"test.tool","arguments":{"actor":"agent","role":"viewer","target":"host=demo"}}}`)
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "completed")
	assert.Contains(t, w.Body.String(), "structuredContent")
	assert.Contains(t, w.Body.String(), "isError")

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	result := resp["result"].(map[string]any)
	assert.Equal(t, false, result["isError"])
}

func TestBearerTokenProtectsMCPAndAPI(t *testing.T) {
	cfg := config.Config{APIToken: "test-token"}
	r := NewRouter(cfg, createTestRegistry(), &mockRecorder{}, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/tools", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	req = httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer wrong")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	req = httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "test.tool")
}
