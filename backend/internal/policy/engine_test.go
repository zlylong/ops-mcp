package policy

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zlylong/ops-mcp/backend/internal/domain"
)

func TestEngine_Evaluate_CriticalTool(t *testing.T) {
	e := NewEngine()
	req := domain.PolicyRequest{
		Tool: domain.Tool{
			Name:             "k8s.delete_cluster",
			Description:      "Delete a Kubernetes cluster",
			Category:         "kubernetes",
			ReadOnly:         false,
			Risk:             domain.RiskCritical,
			RequiresApproval: true,
			InputSchema:      map[string]string{"name": "string"},
		},
		Actor:       "admin",
		Role:        domain.RoleAdmin,
		Environment: domain.EnvProduction,
		Approved:    false,
	}

	decision := e.Evaluate(req)
	assert.False(t, decision.Allowed)
	assert.Contains(t, decision.Reason, "critical tools")
}

func TestEngine_Evaluate_ProductionWriteWithoutApproval(t *testing.T) {
	e := NewEngine()
	req := domain.PolicyRequest{
		Tool: domain.Tool{
			Name:             "k8s.scale_deployment",
			Description:      "Scale a Kubernetes deployment",
			Category:         "kubernetes",
			ReadOnly:         false,
			Risk:             domain.RiskMedium,
			RequiresApproval: true,
			InputSchema:      map[string]string{"name": "string", "replicas": "int"},
		},
		Actor:       "operator",
		Role:        domain.RoleOperator,
		Environment: domain.EnvProduction,
		Approved:    false,
	}

	decision := e.Evaluate(req)
	assert.False(t, decision.Allowed)
	assert.True(t, decision.RequiresApproval)
	assert.Contains(t, decision.Reason, "require approval")
}

func TestEngine_Evaluate_ViewerReadOnlyAllowed(t *testing.T) {
	e := NewEngine()
	req := domain.PolicyRequest{
		Tool: domain.Tool{
			Name:             "k8s.list_pods",
			Description:      "List all pods in a namespace",
			Category:         "kubernetes",
			ReadOnly:         true,
			Risk:             domain.RiskLow,
			RequiresApproval: false,
			InputSchema:      map[string]string{"namespace": "string"},
		},
		Actor:       "viewer",
		Role:        domain.RoleViewer,
		Environment: domain.EnvDevelopment,
		Approved:    false,
	}

	decision := e.Evaluate(req)
	assert.True(t, decision.Allowed)
	assert.Contains(t, decision.Reason, "read-only tools")
}

func TestEngine_Evaluate_ViewerWriteDenied(t *testing.T) {
	e := NewEngine()
	req := domain.PolicyRequest{
		Tool: domain.Tool{
			Name:             "k8s.delete_pod",
			Description:      "Delete a pod",
			Category:         "kubernetes",
			ReadOnly:         false,
			Risk:             domain.RiskMedium,
			RequiresApproval: true,
			InputSchema:      map[string]string{"namespace": "string", "name": "string"},
		},
		Actor:       "viewer",
		Role:        domain.RoleViewer,
		Environment: domain.EnvDevelopment,
		Approved:    false,
	}

	decision := e.Evaluate(req)
	assert.False(t, decision.Allowed)
	assert.Contains(t, decision.Reason, "cannot execute write tools")
}

func TestEngine_Evaluate_OperatorMediumRiskDev(t *testing.T) {
	e := NewEngine()
	req := domain.PolicyRequest{
		Tool: domain.Tool{
			Name:             "k8s.scale_deployment",
			Description:      "Scale a Kubernetes deployment",
			Category:         "kubernetes",
			ReadOnly:         false,
			Risk:             domain.RiskMedium,
			RequiresApproval: true,
			InputSchema:      map[string]string{"name": "string", "replicas": "int"},
		},
		Actor:       "operator",
		Role:        domain.RoleOperator,
		Environment: domain.EnvDevelopment,
		Approved:    false,
	}

	decision := e.Evaluate(req)
	assert.True(t, decision.Allowed)
	assert.Contains(t, decision.Reason, "medium-risk tools in dev")
}

func TestEngine_Evaluate_OperatorMediumRiskProductionAllowed(t *testing.T) {
	e := NewEngine()
	req := domain.PolicyRequest{
		Tool: domain.Tool{
			Name:             "k8s.scale_deployment",
			Description:      "Scale a Kubernetes deployment",
			Category:         "kubernetes",
			ReadOnly:         false,
			Risk:             domain.RiskMedium,
			RequiresApproval: true,
			InputSchema:      map[string]string{"name": "string", "replicas": "int"},
		},
		Actor:       "operator",
		Role:        domain.RoleOperator,
		Environment: domain.EnvProduction,
		Approved:    true,
	}

	decision := e.Evaluate(req)
	assert.True(t, decision.Allowed)
	assert.Contains(t, decision.Reason, "medium-risk tools in production when approved")
}

func TestEngine_Evaluate_AdminAllowed(t *testing.T) {
	e := NewEngine()
	req := domain.PolicyRequest{
		Tool: domain.Tool{
			Name:             "k8s.delete_cluster",
			Description:      "Delete a Kubernetes cluster",
			Category:         "kubernetes",
			ReadOnly:         false,
			Risk:             domain.RiskCritical,
			RequiresApproval: true,
			InputSchema:      map[string]string{"name": "string"},
		},
		Actor:       "admin",
		Role:        domain.RoleAdmin,
		Environment: domain.EnvProduction,
		Approved:    false,
	}

	decision := e.Evaluate(req)
	assert.False(t, decision.Allowed)
	assert.Contains(t, decision.Reason, "critical tools")
}

func TestEngine_Evaluate_UnknownRole(t *testing.T) {
	e := NewEngine()
	req := domain.PolicyRequest{
		Tool: domain.Tool{
			Name:             "k8s.list_pods",
			Description:      "List all pods",
			Category:         "kubernetes",
			ReadOnly:         true,
			Risk:             domain.RiskLow,
			RequiresApproval: false,
			InputSchema:      map[string]string{"namespace": "string"},
		},
		Actor:       "unknown",
		Role:        "unknown_role",
		Environment: domain.EnvDevelopment,
		Approved:    false,
	}

	decision := e.Evaluate(req)
	assert.False(t, decision.Allowed)
	assert.Contains(t, decision.Reason, "unknown role")
}

func TestEngine_Evaluate_RiskHighDevDenied(t *testing.T) {
	e := NewEngine()
	req := domain.PolicyRequest{
		Tool: domain.Tool{
			Name:             "k8s.update_node_config",
			Description:      "Update node configuration",
			Category:         "kubernetes",
			ReadOnly:         false,
			Risk:             domain.RiskHigh,
			RequiresApproval: true,
			InputSchema:      map[string]string{"node": "string", "config": "string"},
		},
		Actor:       "operator",
		Role:        domain.RoleOperator,
		Environment: domain.EnvDevelopment,
		Approved:    false,
	}

	decision := e.Evaluate(req)
	assert.False(t, decision.Allowed)
	assert.Contains(t, decision.Reason, "not allowed for this environment or risk")
}
