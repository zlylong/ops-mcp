package domain

import "time"

type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

type Environment string

const (
	EnvDevelopment Environment = "development"
	EnvStaging     Environment = "staging"
	EnvProduction  Environment = "production"
)

type Role string

const (
	RoleViewer   Role = "viewer"
	RoleOperator Role = "operator"
	RoleAdmin    Role = "admin"
)

type Tool struct {
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	Category         string            `json:"category"`
	ReadOnly         bool              `json:"readOnly"`
	Risk             RiskLevel         `json:"risk"`
	RequiresApproval bool              `json:"requiresApproval"`
	InputSchema      map[string]string `json:"inputSchema"`
}

type ExecuteRequest struct {
	Actor      string         `json:"actor"`
	Role       Role           `json:"role"`
	Target     string         `json:"target"`
	Approved   bool           `json:"approved"`
	Parameters map[string]any `json:"parameters"`
}

type ExecuteResult struct {
	ExecutionID string         `json:"executionId"`
	AuditID     string         `json:"auditId"`
	ApprovalID  string         `json:"approvalId,omitempty"`
	Status      string         `json:"status"`
	Message     string         `json:"message"`
	Data        map[string]any `json:"data,omitempty"`
}

type PolicyRequest struct {
	Tool        Tool        `json:"tool"`
	Actor       string      `json:"actor"`
	Role        Role        `json:"role"`
	Environment Environment `json:"environment"`
	Approved    bool        `json:"approved"`
}

type PolicyDecision struct {
	Allowed          bool   `json:"allowed"`
	RequiresApproval bool   `json:"requiresApproval"`
	Reason           string `json:"reason"`
}

type AuditRecord struct {
	ID          string         `json:"id"`
	ExecutionID string         `json:"executionId,omitempty"`
	TraceID     string         `json:"traceId,omitempty"`
	At          time.Time      `json:"at"`
	Actor       string         `json:"actor"`
	Role        Role           `json:"role"`
	Action      string         `json:"action"`
	Target      string         `json:"target"`
	Allowed     bool           `json:"allowed"`
	Reason      string         `json:"reason"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type Execution struct {
	ID         string         `json:"id"`
	Tool       string         `json:"tool"`
	Actor      string         `json:"actor"`
	Role       Role           `json:"role"`
	Target     string         `json:"target"`
	Status     string         `json:"status"`
	Reason     string         `json:"reason"`
	Parameters map[string]any `json:"parameters,omitempty"`
	Result     map[string]any `json:"result,omitempty"`
	AuditID    string         `json:"auditId"`
	CreatedAt  time.Time      `json:"createdAt"`
}

type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalRejected ApprovalStatus = "rejected"
)

type Approval struct {
	ID          string         `json:"id"`
	ExecutionID string         `json:"executionId"`
	Tool        string         `json:"tool"`
	Actor       string         `json:"actor"`
	Target      string         `json:"target"`
	Status      ApprovalStatus `json:"status"`
	Reason      string         `json:"reason"`
	CreatedAt   time.Time      `json:"createdAt"`
	DecidedAt   *time.Time     `json:"decidedAt,omitempty"`
}
