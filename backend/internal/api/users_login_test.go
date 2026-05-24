package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/zlylong/darwin-ops-mcp/backend/internal/config"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/domain"
)

func TestLoginReturnsUsableUserBearerToken(t *testing.T) {
	registry := createTestRegistry()
	hash, err := bcrypt.GenerateFromPassword([]byte("correct horse battery staple"), bcrypt.DefaultCost)
	require.NoError(t, err)
	user := registry.Users().Add(domain.User{Username: "alice", Nickname: "Alice", Role: domain.RoleAdmin, Status: "active"}, "", hash)

	r := NewRouter(config.Config{APIToken: "master-token"}, registry, &mockRecorder{}, slog.Default())
	body, _ := json.Marshal(map[string]string{"username": "alice", "password": "correct horse battery staple"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var payload struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
	require.Equal(t, "user:"+user.ID, payload.Token)

	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+payload.Token)
	meW := httptest.NewRecorder()
	r.ServeHTTP(meW, meReq)
	require.Equal(t, http.StatusOK, meW.Code, meW.Body.String())
}
