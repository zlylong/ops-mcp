package app

import (
	"context"
	"log/slog"
	"testing"

	"github.com/zlylong/ops-mcp/backend/internal/adapters/kubernetes"
	"github.com/zlylong/ops-mcp/backend/internal/adapters/prometheus"
	"github.com/zlylong/ops-mcp/backend/internal/audit"
	"github.com/zlylong/ops-mcp/backend/internal/domain"
	"github.com/zlylong/ops-mcp/backend/internal/policy"
	"github.com/zlylong/ops-mcp/backend/internal/storage"
)

func newTestRegistry(t *testing.T) *Registry {
	t.Helper()
	r := NewRegistry(policy.NewEngine(), audit.NewStore(slog.Default()), storage.NewExecutionStore(), storage.NewApprovalStore(), domain.EnvDevelopment)
	if err := RegisterMockTools(r, kubernetes.NewMockAdapter(), prometheus.NewMockAdapter()); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestRegistryListAndGet(t *testing.T) {
	t.Parallel()
	r := newTestRegistry(t)
	if got := len(r.List()); got != 9 {
		t.Fatalf("tools=%d want 9", got)
	}
	tool, ok := r.Get("k8s.list_pods")
	if !ok {
		t.Fatal("expected k8s.list_pods")
	}
	if !tool.ReadOnly {
		t.Fatal("k8s.list_pods must be read-only")
	}
}

func TestRegistryExecuteValidatesInputAndAudits(t *testing.T) {
	t.Parallel()
	r := newTestRegistry(t)
	result, status, err := r.Execute(context.Background(), "k8s.get_pod_logs", domain.ExecuteRequest{Actor: "tester", Role: domain.RoleViewer, Parameters: map[string]any{"namespace": "default"}})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if status != 400 {
		t.Fatalf("status=%d want 400", status)
	}
	if result.AuditID == "" || result.ExecutionID == "" {
		t.Fatalf("missing ids: %#v", result)
	}
	if len(r.Executions()) != 1 {
		t.Fatalf("executions=%d want 1", len(r.Executions()))
	}
}

func TestRegistryExecuteSuccess(t *testing.T) {
	t.Parallel()
	r := newTestRegistry(t)
	result, status, err := r.Execute(context.Background(), "prometheus.service_error_rate", domain.ExecuteRequest{Actor: "tester", Role: domain.RoleViewer, Parameters: map[string]any{"service": "api"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != 200 {
		t.Fatalf("status=%d want 200", status)
	}
	if result.Data["errorRate"] == nil {
		t.Fatalf("missing errorRate result: %#v", result.Data)
	}
}
