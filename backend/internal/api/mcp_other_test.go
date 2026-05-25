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

func mcpTestRegistry(t *testing.T) *app.Registry {
	t.Helper()
	return app.NewRegistry(
		policy.NewEngine(),
		&mcpMockAudit{},
		storage.NewExecutionStore(),
		storage.NewApprovalStore(),
		storage.NewUserStore(),
		storage.NewJumpServerStore(),
		domain.EnvDevelopment,
	)
}

type mcpMockAudit struct{}

func (m *mcpMockAudit) Record(record domain.AuditRecord) domain.AuditRecord { return record }
func (m *mcpMockAudit) List() []domain.AuditRecord                          { return nil }

func TestExecutions_Success(t *testing.T) {
	r := mcpTestRegistry(t)
	cfg := config.Config{APIToken: "test-agent-token"}
	handler := NewRouter(cfg, r, &mcpMockAudit{}, slog.Default())
	req := httptest.NewRequest("GET", "/api/v1/executions", nil)
	req.Header.Set("Authorization", "Bearer test-agent-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.NotEqual(t, http.StatusUnauthorized, rr.Code)
}

func TestExecutions_Unauthorized(t *testing.T) {
	r := mcpTestRegistry(t)
	cfg := config.Config{APIToken: "test-agent-token"}
	handler := NewRouter(cfg, r, &mcpMockAudit{}, slog.Default())
	req := httptest.NewRequest("GET", "/api/v1/executions", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.NotEqual(t, http.StatusOK, rr.Code)
}

func TestApplications_List(t *testing.T) {
	r := mcpTestRegistry(t)
	cfg := config.Config{APIToken: "test-agent-token"}
	handler := NewRouter(cfg, r, &mcpMockAudit{}, slog.Default())
	req := httptest.NewRequest("GET", "/api/v1/applications", nil)
	req.Header.Set("Authorization", "Bearer test-agent-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.NotEqual(t, http.StatusUnauthorized, rr.Code)
}

func TestApproveApplication_NotFound(t *testing.T) {
	r := mcpTestRegistry(t)
	cfg := config.Config{APIToken: "test-agent-token"}
	handler := NewRouter(cfg, r, &mcpMockAudit{}, slog.Default())
	req := httptest.NewRequest("POST", "/api/v1/applications/nonexistent/approve", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer user:admin")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	// admin but app not found
	require.NotEqual(t, http.StatusOK, rr.Code)
}

func TestRevokeAgentAPIKey_NotFound(t *testing.T) {
	r := mcpTestRegistry(t)
	cfg := config.Config{APIToken: "test-agent-token"}
	handler := NewRouter(cfg, r, &mcpMockAudit{}, slog.Default())
	req := httptest.NewRequest("DELETE", "/api/v1/agent/keys/nonexistent", nil)
	req.Header.Set("Authorization", "Bearer user:admin")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.NotEqual(t, http.StatusOK, rr.Code)
}
