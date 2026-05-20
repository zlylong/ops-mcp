package storage

import (
	"testing"
	"time"

	"github.com/zlylong/ops-mcp/backend/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestExecutionStore_Add(t *testing.T) {
	store := NewExecutionStore()

	// 测试添加新执行
	e := domain.Execution{
		Tool:     "k8s.list_pods",
		AuditID:  "aud-123",
		Actor:    "test-user",
		Role:     domain.RoleViewer,
		Target:   "local-dev",
		Status:   "succeeded",
		Reason:   "test execution",
		CreatedAt: time.Now().UTC(),
	}
	result := store.Add(e)

	assert.NotEmpty(t, result.ID)
	assert.Equal(t, "k8s.list_pods", result.Tool)
	assert.Equal(t, "aud-123", result.AuditID)
	assert.Equal(t, "test-user", result.Actor)
}

func TestExecutionStore_List(t *testing.T) {
	store := NewExecutionStore()

	e1 := domain.Execution{Tool: "tool1", AuditID: "aud-1", Actor: "user1", Role: domain.RoleViewer, Target: "t1", Status: "succeeded", Reason: "test"}
	e2 := domain.Execution{Tool: "tool2", AuditID: "aud-2", Actor: "user2", Role: domain.RoleOperator, Target: "t2", Status: "failed", Reason: "test"}
	store.Add(e1)
	store.Add(e2)

	list := store.List()
	assert.Len(t, list, 2)
	assert.Equal(t, "tool2", list[0].Tool)
	assert.Equal(t, "tool1", list[1].Tool)
}

func TestExecutionStore_Get(t *testing.T) {
	store := NewExecutionStore()

	e := domain.Execution{Tool: "test-tool", AuditID: "aud-test", Actor: "test", Role: domain.RoleViewer, Target: "test", Status: "succeeded", Reason: "test"}
	added := store.Add(e)

	found, ok := store.Get(added.ID)
	assert.True(t, ok)
	assert.Equal(t, added.ID, found.ID)
	assert.Equal(t, "test-tool", found.Tool)

	_, ok = store.Get("nonexistent")
	assert.False(t, ok)
}

func TestApprovalStore_Add(t *testing.T) {
	store := NewApprovalStore()

	a := domain.Approval{
		ExecutionID: "exe-123",
		Tool:        "k8s.list_pods",
		Actor:       "test-user",
		Target:      "local-dev",
		Status:      domain.ApprovalPending,
		Reason:      "test approval",
		CreatedAt:   time.Now().UTC(),
	}
	result := store.Add(a)

	assert.NotEmpty(t, result.ID)
	assert.Equal(t, "exe-123", result.ExecutionID)
	assert.Equal(t, domain.ApprovalPending, result.Status)
}

func TestApprovalStore_Decide(t *testing.T) {
	store := NewApprovalStore()

	a := domain.Approval{
		ExecutionID: "exe-123",
		Tool:        "k8s.list_pods",
		Actor:       "test-user",
		Target:      "local-dev",
		Status:      domain.ApprovalPending,
		Reason:      "test approval",
		CreatedAt:   time.Now().UTC(),
	}
	added := store.Add(a)

	// 测试批准
	approved, err := store.Decide(added.ID, domain.ApprovalApproved)
	assert.NoError(t, err)
	assert.Equal(t, domain.ApprovalApproved, approved.Status)
	assert.NotNil(t, approved.DecidedAt)

	// 测试拒绝
	rejected, err := store.Decide(added.ID, domain.ApprovalRejected)
	assert.NoError(t, err)
	assert.Equal(t, domain.ApprovalRejected, rejected.Status)

	// 测试未找到
	_, err = store.Decide("nonexistent", domain.ApprovalApproved)
	assert.Error(t, err)
}
