package ops

import (
	"log/slog"
	"testing"

	"github.com/example/ops-mcp/backend/internal/audit"
)

func TestServiceExecuteBlocksForbiddenTools(t *testing.T) {
	t.Parallel()
	svc := NewService("mock", audit.NewLogger(slog.Default()))
	result, status, err := svc.Execute(ToolRequest{Tool: "delete_namespace", Actor: "tester", Target: "prod"}, "development")
	if err == nil {
		t.Fatal("expected forbidden tool error")
	}
	if status != 403 {
		t.Fatalf("status=%d want 403", status)
	}
	if result.Status != "blocked" {
		t.Fatalf("status=%q want blocked", result.Status)
	}
}

func TestServiceExecuteRequiresProductionApproval(t *testing.T) {
	t.Parallel()
	svc := NewService("mock", audit.NewLogger(slog.Default()))
	result, status, err := svc.Execute(ToolRequest{Tool: "scale_workload", Actor: "tester", Target: "deployment/api"}, "production")
	if err == nil {
		t.Fatal("expected approval error")
	}
	if status != 409 {
		t.Fatalf("status=%d want 409", status)
	}
	if result.Status != "approval_required" {
		t.Fatalf("status=%q want approval_required", result.Status)
	}
}

func TestServiceExecuteApprovedMockWrite(t *testing.T) {
	t.Parallel()
	recorder := audit.NewLogger(slog.Default())
	svc := NewService("mock", recorder)
	result, status, err := svc.Execute(ToolRequest{Tool: "restart_rollout", Actor: "tester", Target: "deployment/api", Approved: true}, "production")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != 200 {
		t.Fatalf("status=%d want 200", status)
	}
	if result.AuditID == "" {
		t.Fatal("expected audit id")
	}
	if len(recorder.List()) != 1 {
		t.Fatalf("audit events=%d want 1", len(recorder.List()))
	}
}
