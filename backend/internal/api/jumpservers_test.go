package api

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zlylong/darwin-ops-mcp/backend/internal/app"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/config"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/domain"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/policy"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/storage"
)

func jumpTestRegistry(t *testing.T) *app.Registry {
	t.Helper()
	return app.NewRegistry(
		policy.NewEngine(),
		&jumpMockAudit{},
		storage.NewExecutionStore(),
		storage.NewApprovalStore(),
		storage.NewUserStore(),
		storage.NewJumpServerStore(),
		domain.EnvDevelopment,
	)
}

type jumpMockAudit struct{}

func (m *jumpMockAudit) Record(record domain.AuditRecord) domain.AuditRecord { return record }
func (m *jumpMockAudit) List() []domain.AuditRecord                          { return nil }

func TestJumpServersSSRF_PrivateIPv4(t *testing.T) {
	r := jumpTestRegistry(t)
	cfg := config.Config{APIToken: "test-agent-token"}
	handler := NewRouter(cfg, r, &jumpMockAudit{}, slog.Default())
	req := httptest.NewRequest("GET", "/api/v1/jumpservers?host=10.0.0.1&port=22", nil)
	req.Header.Set("Authorization", "Bearer test-agent-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestJumpServersSSRF_LoopbackIPv6(t *testing.T) {
	r := jumpTestRegistry(t)
	cfg := config.Config{APIToken: "test-agent-token"}
	handler := NewRouter(cfg, r, &jumpMockAudit{}, slog.Default())
	req := httptest.NewRequest("GET", "/api/v1/jumpservers?host=::1&port=22", nil)
	req.Header.Set("Authorization", "Bearer test-agent-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestJumpServersSSRF_LoopbackIPv4(t *testing.T) {
	r := jumpTestRegistry(t)
	cfg := config.Config{APIToken: "test-agent-token"}
	handler := NewRouter(cfg, r, &jumpMockAudit{}, slog.Default())
	req := httptest.NewRequest("GET", "/api/v1/jumpservers?host=127.0.0.1&port=22", nil)
	req.Header.Set("Authorization", "Bearer test-agent-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestListJumpServers_Unauthenticated(t *testing.T) {
	r := jumpTestRegistry(t)
	cfg := config.Config{APIToken: "test-agent-token"}
	handler := NewRouter(cfg, r, &jumpMockAudit{}, slog.Default())
	req := httptest.NewRequest("GET", "/api/v1/jumpservers", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.NotEqual(t, http.StatusOK, rr.Code)
}

func TestListJumpServers_WithAgentToken(t *testing.T) {
	r := jumpTestRegistry(t)
	cfg := config.Config{APIToken: "test-agent-token"}
	handler := NewRouter(cfg, r, &jumpMockAudit{}, slog.Default())
	req := httptest.NewRequest("GET", "/api/v1/jumpservers", nil)
	req.Header.Set("Authorization", "Bearer test-agent-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.NotEqual(t, http.StatusUnauthorized, rr.Code)
}
