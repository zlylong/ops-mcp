package policy

import "github.com/zlylong/ops-mcp/backend/internal/domain"

type Engine struct{}

func NewEngine() *Engine { return &Engine{} }

func (e *Engine) Evaluate(req domain.PolicyRequest) domain.PolicyDecision {
	if req.Tool.Risk == domain.RiskCritical {
		return domain.PolicyDecision{Allowed: false, Reason: "critical tools are denied by default"}
	}
	switch req.Role {
	case domain.RoleViewer:
		if req.Tool.ReadOnly {
			return domain.PolicyDecision{Allowed: true, Reason: "viewer may execute read-only tools"}
		}
		return domain.PolicyDecision{Allowed: false, Reason: "viewer cannot execute write tools"}
	case domain.RoleOperator:
		if req.Tool.ReadOnly {
			return domain.PolicyDecision{Allowed: true, Reason: "operator may execute read-only tools"}
		}
		if req.Tool.Risk == domain.RiskMedium && (req.Environment == domain.EnvDevelopment || req.Environment == domain.EnvStaging) {
			return domain.PolicyDecision{Allowed: true, Reason: "operator may execute medium-risk tools in dev/staging"}
		}
		if req.Tool.Risk == domain.RiskMedium && req.Environment == domain.EnvProduction && req.Approved {
			return domain.PolicyDecision{Allowed: true, Reason: "operator may execute medium-risk tools in production when approved"}
		}
		if req.Environment == domain.EnvProduction {
			return domain.PolicyDecision{Allowed: false, Reason: "production write operations require approval", RequiresApproval: true}
		}
		return domain.PolicyDecision{Allowed: false, Reason: "operator is not allowed for this environment or risk"}
	case domain.RoleAdmin:
		return domain.PolicyDecision{Allowed: true, Reason: "admin allowed by policy"}
	default:
		return domain.PolicyDecision{Allowed: false, Reason: "unknown role"}
	}
}
