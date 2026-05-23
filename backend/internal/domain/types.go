package domain

import (
	"errors"
	"time"
)

// ParamSchema describes a single input parameter for a tool.
type ParamSchema struct {
	Type        string `json:"type"`              // "string" | "number" | "boolean"
	Required    bool   `json:"required"`          // true = mandatory, false = optional
	Description string `json:"description"`       // human-readable description
	Default     any    `json:"default,omitempty"` // optional default value
}

// Validate checks whether params[key] satisfies this schema.
// It returns an error if a required key is missing, or if the value type does not match.
func (p ParamSchema) Validate(key string, params map[string]any) error {
	val, ok := params[key]
	if !ok {
		if p.Required {
			return errors.New("missing required parameter: " + key)
		}
		return nil
	}
	if val == nil {
		return errors.New("parameter " + key + " must not be null")
	}
	switch p.Type {
	case "string":
		if _, ok := val.(string); !ok {
			return errors.New("parameter " + key + " must be a string")
		}
	case "number":
		switch v := val.(type) {
		case float64, int, int32, int64:
			_ = v // use variable to avoid unused warning
		default:
			return errors.New("parameter " + key + " must be a number")
		}
	case "boolean":
		if _, ok := val.(bool); !ok {
			return errors.New("parameter " + key + " must be a boolean")
		}
	}
	return nil
}

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
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Category         string                 `json:"category"`
	ReadOnly         bool                   `json:"readOnly"`
	Risk             RiskLevel              `json:"risk"`
	RequiresApproval bool                   `json:"requiresApproval"`
	InputSchema      map[string]ParamSchema `json:"inputSchema"`
}

// ValidateParams checks that all required parameters are present and their types match.
func (t Tool) ValidateParams(params map[string]any) error {
	for name, schema := range t.InputSchema {
		if err := schema.Validate(name, params); err != nil {
			return err
		}
	}
	return nil
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

// ApplicationStatus represents the review status of a tool application.
type ApplicationStatus string

const (
	ApplicationPending  ApplicationStatus = "pending"
	ApplicationApproved ApplicationStatus = "approved"
	ApplicationRejected ApplicationStatus = "rejected"
)

// ToolApplicationRequest is submitted by an actor to request access to a tool
// at a specific role and environment level.
type ToolApplicationRequest struct {
	Tool        string         `json:"tool"`
	Risk        RiskLevel      `json:"risk"`
	Role        Role           `json:"role"`
	Reason      string         `json:"reason"`
	DurationHrs int            `json:"durationHrs"` // hours; 0 = default 24h
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// ToolApplication is the server-side record of a tool access application.
type ToolApplication struct {
	ID          string            `json:"id"`
	Tool        string            `json:"tool"`
	Risk        RiskLevel         `json:"risk"`
	Role        Role              `json:"role"`
	Actor       string            `json:"actor"`
	Reason      string            `json:"reason"`
	Status      ApplicationStatus `json:"status"`
	Decision    string            `json:"decision,omitempty"`
	DurationHrs int               `json:"durationHrs"`
	Parameters  map[string]any    `json:"parameters,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
	DecidedAt   *time.Time        `json:"decidedAt,omitempty"`
}

// AgentAPIKeyCreateRequest describes a one-time API key issuance request for an AI agent.
type AgentAPIKeyCreateRequest struct {
	Name         string   `json:"name"`
	Actor        string   `json:"actor"`
	Role         Role     `json:"role"`
	Reason       string   `json:"reason"`
	Scopes       []string `json:"scopes,omitempty"`
	ExpiresInHrs int      `json:"expiresInHrs"` // hours; 0 = non-expiring
}

// AgentAPIKey is the server-side metadata for an agent API key. It never contains
// the plaintext secret; the secret is returned only once by AgentAPIKeyCreateResponse.
type AgentAPIKey struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Actor      string     `json:"actor"`
	Role       Role       `json:"role"`
	Reason     string     `json:"reason,omitempty"`
	Scopes     []string   `json:"scopes,omitempty"`
	KeyPrefix  string     `json:"keyPrefix"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"createdAt"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	RevokedAt  *time.Time `json:"revokedAt,omitempty"`
}

// AgentAPIKeyCreateResponse includes the plaintext secret exactly once at issuance time.
type AgentAPIKeyCreateResponse struct {
	AgentAPIKey
	Secret string `json:"secret"`
}
// ── User management ────────────────────────────────────────────────────────────
// User represents a server-side user account.
type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Nickname  string    `json:"nickname"`
	Email     string    `json:"email,omitempty"`
	Role      Role      `json:"role"`
	Status    string    `json:"status"` // "active" | "inactive"
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
// UserCreateRequest is used to register a new user account (admin only).
type UserCreateRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Password string `json:"password" binding:"required,min=8,max=128"`
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
	Role     Role   `json:"role"`
}

// UserUpdateRequest is used to update an existing user (admin or self).
type UserUpdateRequest struct {
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
	Status   string `json:"status,omitempty"` // "active" | "inactive"; admin only
	Role     Role   `json:"role,omitempty"`   // admin only
}

// ChangePasswordRequest is used to change a user's own password.
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=8,max=128"`
}

// ChangePasswordByAdminRequest is used by an admin to reset a user's password.
type ChangePasswordByAdminRequest struct {
	NewPassword string `json:"newPassword" binding:"required,min=8,max=128"`
}
