package api

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/ops-mcp/backend/internal/audit"
	"github.com/example/ops-mcp/backend/internal/config"
	"github.com/example/ops-mcp/backend/internal/ops"
)

func TestHealthRoute(t *testing.T) {
	t.Parallel()
	auditor := audit.NewLogger(slog.Default())
	handler := NewRouter(config.Config{Mode: "mock", Environment: "development"}, ops.NewService("mock", auditor), auditor, slog.Default())
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want 200", rec.Code)
	}
}
