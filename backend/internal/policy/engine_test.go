package policy

import (
	"testing"

	"github.com/zlylong/ops-mcp/backend/internal/domain"
)

func TestEngineEvaluate(t *testing.T) {
	t.Parallel()
	readOnlyTool := domain.Tool{Name: "k8s.list_pods", ReadOnly: true, Risk: domain.RiskLow}
	writeTool := domain.Tool{Name: "mock.write", ReadOnly: false, Risk: domain.RiskMedium}
	criticalTool := domain.Tool{Name: "critical", ReadOnly: false, Risk: domain.RiskCritical}
	tests := []struct {
		name         string
		req          domain.PolicyRequest
		wantAllowed  bool
		wantApproval bool
	}{
		{name: "viewer read-only", req: domain.PolicyRequest{Tool: readOnlyTool, Role: domain.RoleViewer, Environment: domain.EnvProduction}, wantAllowed: true},
		{name: "viewer write denied", req: domain.PolicyRequest{Tool: writeTool, Role: domain.RoleViewer, Environment: domain.EnvDevelopment}, wantAllowed: false},
		{name: "operator medium dev", req: domain.PolicyRequest{Tool: writeTool, Role: domain.RoleOperator, Environment: domain.EnvDevelopment}, wantAllowed: true},
		{name: "production write requires approval", req: domain.PolicyRequest{Tool: writeTool, Role: domain.RoleOperator, Environment: domain.EnvProduction}, wantAllowed: false, wantApproval: true},
		{name: "critical denied", req: domain.PolicyRequest{Tool: criticalTool, Role: domain.RoleAdmin, Environment: domain.EnvDevelopment, Approved: true}, wantAllowed: false},
	}
	engine := NewEngine()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.Evaluate(tt.req)
			if got.Allowed != tt.wantAllowed {
				t.Fatalf("Allowed=%v want %v: %s", got.Allowed, tt.wantAllowed, got.Reason)
			}
			if got.RequiresApproval != tt.wantApproval {
				t.Fatalf("RequiresApproval=%v want %v", got.RequiresApproval, tt.wantApproval)
			}
		})
	}
}
