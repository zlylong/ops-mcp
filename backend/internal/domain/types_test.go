package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRiskLevel_Values(t *testing.T) {
	assert.Equal(t, "low", string(RiskLow))
	assert.Equal(t, "medium", string(RiskMedium))
	assert.Equal(t, "high", string(RiskHigh))
	assert.Equal(t, "critical", string(RiskCritical))
}

func TestEnvironment_Values(t *testing.T) {
	assert.Equal(t, "development", string(EnvDevelopment))
	assert.Equal(t, "staging", string(EnvStaging))
	assert.Equal(t, "production", string(EnvProduction))
}

func TestRole_Values(t *testing.T) {
	assert.Equal(t, "viewer", string(RoleViewer))
	assert.Equal(t, "operator", string(RoleOperator))
	assert.Equal(t, "admin", string(RoleAdmin))
}

func TestTool_Struct(t *testing.T) {
	tool := Tool{
		Name:             "k8s.list_pods",
		Description:      "List Kubernetes pods",
		Category:         "kubernetes",
		ReadOnly:         true,
		Risk:             RiskLow,
		RequiresApproval: false,
		InputSchema:      map[string]string{"namespace": "string"},
	}
	assert.Equal(t, "k8s.list_pods", tool.Name)
	assert.True(t, tool.ReadOnly)
	assert.Equal(t, RiskLow, tool.Risk)
}

func TestExecuteRequest_Struct(t *testing.T) {
	req := ExecuteRequest{
		Actor:      "test-user",
		Role:       RoleViewer,
		Target:     "local-dev",
		Approved:   true,
		Parameters: map[string]any{"namespace": "default"},
	}
	assert.Equal(t, "test-user", req.Actor)
	assert.Equal(t, RoleViewer, req.Role)
	assert.True(t, req.Approved)
}

func TestPolicyDecision_Struct(t *testing.T) {
	decision := PolicyDecision{
		Allowed:          true,
		RequiresApproval: false,
		Reason:           "allowed by policy",
	}
	assert.True(t, decision.Allowed)
	assert.Equal(t, "allowed by policy", decision.Reason)
}

func TestAuditRecord_Struct(t *testing.T) {
	record := AuditRecord{
		ID:          "aud-123",
		ExecutionID: "exe-123",
		TraceID:     "trace-abc",
		Actor:       "test-user",
		Role:        RoleViewer,
		Action:      "k8s.list_pods",
		Target:      "local-dev",
		Allowed:     true,
		Reason:      "allowed",
		Parameters:  map[string]any{"namespace": "default"},
	}
	assert.Equal(t, "aud-123", record.ID)
	assert.Equal(t, "trace-abc", record.TraceID)
	assert.True(t, record.Allowed)
}

func TestExecution_Struct(t *testing.T) {
	exec := Execution{
		ID:         "exe-123",
		Tool:       "k8s.list_pods",
		Actor:      "test-user",
		Role:       RoleViewer,
		Target:     "local-dev",
		Status:     "succeeded",
		Reason:     "executed",
		Parameters: map[string]any{"namespace": "default"},
		AuditID:    "aud-123",
	}
	assert.Equal(t, "exe-123", exec.ID)
	assert.Equal(t, "succeeded", exec.Status)
}

func TestApproval_Struct(t *testing.T) {
	approval := Approval{
		ID:          "app-123",
		ExecutionID: "exe-123",
		Tool:        "k8s.list_pods",
		Actor:       "test-user",
		Target:      "local-dev",
		Status:      ApprovalPending,
		Reason:      "pending review",
	}
	assert.Equal(t, "app-123", approval.ID)
	assert.Equal(t, ApprovalPending, approval.Status)
}

func TestApproval_Decision(t *testing.T) {
	approval := Approval{
		ID:          "app-123",
		ExecutionID: "exe-123",
		Tool:        "k8s.list_pods",
		Actor:       "test-user",
		Target:      "local-dev",
		Status:      ApprovalPending,
		Reason:      "pending",
	}

	approval.Status = ApprovalApproved
	assert.Equal(t, ApprovalApproved, approval.Status)

	approval.Status = ApprovalRejected
	assert.Equal(t, ApprovalRejected, approval.Status)
}
