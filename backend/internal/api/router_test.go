package api

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zlylong/ops-mcp/backend/internal/adapters/kubernetes"
	"github.com/zlylong/ops-mcp/backend/internal/adapters/prometheus"
	"github.com/zlylong/ops-mcp/backend/internal/app"
	"github.com/zlylong/ops-mcp/backend/internal/audit"
	"github.com/zlylong/ops-mcp/backend/internal/config"
	"github.com/zlylong/ops-mcp/backend/internal/domain"
	"github.com/zlylong/ops-mcp/backend/internal/policy"
	"github.com/zlylong/ops-mcp/backend/internal/storage"
)

func TestHealthRoute(t *testing.T) {
	t.Parallel()
	auditor := audit.NewStore(slog.Default())
	registry := app.NewRegistry(policy.NewEngine(), auditor, storage.NewExecutionStore(), storage.NewApprovalStore(), domain.EnvDevelopment)
	if err := app.RegisterMockTools(registry, kubernetes.NewMockAdapter(), prometheus.NewMockAdapter()); err != nil {
		t.Fatal(err)
	}
	handler := NewRouter(config.Config{Mode: "mock", Environment: domain.EnvDevelopment}, registry, auditor, slog.Default())
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want 200", rec.Code)
	}
}
