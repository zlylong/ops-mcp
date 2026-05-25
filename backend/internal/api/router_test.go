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

func createTestRegistry(t ...*testing.T) *app.Registry {
	if len(t) > 0 && t[0] != nil {
		t[0].Helper()
	}
	return app.NewRegistry(
		policy.NewEngine(),
		&mockAudit{},
		storage.NewExecutionStore(),
		storage.NewApprovalStore(),
		storage.NewUserStore(),
		storage.NewJumpServerStore(),
		domain.EnvDevelopment,
	)
}

type mockAudit struct{}

func (m *mockAudit) Record(record domain.AuditRecord) domain.AuditRecord { return record }
func (m *mockAudit) List() []domain.AuditRecord                          { return nil }

func TestNewRouter_CORS(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{}
	handler := NewRouter(cfg, r, &mockAudit{}, tLogger())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://evil.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestNewRouter_CSRF(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{APIToken: "test-agent-token"}
	handler := NewRouter(cfg, r, &mockAudit{}, tLogger())
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusNotFound, rr.Code)
	require.NotEmpty(t, rr.Header().Get(TraceIDHeader))
}

func TestHealthEndpoint(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{APIToken: "test-agent-token"}
	handler := NewRouter(cfg, r, &mockAudit{}, tLogger())
	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestExecutions_Unauthenticated(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{APIToken: "test-agent-token"}
	handler := NewRouter(cfg, r, &mockAudit{}, tLogger())
	req := httptest.NewRequest("GET", "/api/v1/executions", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.NotEqual(t, http.StatusOK, rr.Code)
}

func TestApplications_Unauthenticated(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{APIToken: "test-agent-token"}
	handler := NewRouter(cfg, r, &mockAudit{}, tLogger())
	req := httptest.NewRequest("GET", "/api/v1/applications", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.NotEqual(t, http.StatusOK, rr.Code)
}

func tLogger() *slog.Logger {
	return slog.Default()
}
