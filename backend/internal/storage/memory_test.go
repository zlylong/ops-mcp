package storage

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/domain"
)

func TestExecutionStore_Add(t *testing.T) {
	store := NewExecutionStore()

	// 测试添加新执行
	e := domain.Execution{
		Tool:      "k8s.list_pods",
		AuditID:   "aud-123",
		Actor:     "test-user",
		Role:      domain.RoleViewer,
		Target:    "local-dev",
		Status:    "succeeded",
		Reason:    "test execution",
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

func TestExecutionStore_Update(t *testing.T) {
	store := NewExecutionStore()
	added := store.Add(domain.Execution{Tool: "test", Actor: "user", Role: domain.RoleViewer, Target: "t", Status: "completed", Reason: "test"})

	// Update to error status
	err := store.Update(added.ID, func(e *domain.Execution) {
		e.Status = "error"
		e.Reason = "handler failed"
	})
	assert.NoError(t, err)

	updated, ok := store.Get(added.ID)
	assert.True(t, ok)
	assert.Equal(t, "error", updated.Status)
	assert.Equal(t, "handler failed", updated.Reason)

	// Verify only one record exists
	list := store.List()
	assert.Len(t, list, 1)
	assert.Equal(t, "error", list[0].Status)

	// Update non-existent ID
	err = store.Update("nonexistent", func(ex *domain.Execution) { ex.Status = "updated" })
	assert.Error(t, err)
}

func TestApprovalStore_Decide_ReturnsCopy(t *testing.T) {
	store := NewApprovalStore()
	added := store.Add(domain.Approval{ExecutionID: "exe-1", Tool: "test", Actor: "user", Target: "t", Status: domain.ApprovalPending, Reason: "test"})

	approved, err := store.Decide(added.ID, domain.ApprovalApproved)
	assert.NoError(t, err)
	assert.Equal(t, domain.ApprovalApproved, approved.Status)
	assert.NotNil(t, approved.DecidedAt)

	// Mutate the returned value - stored data must not be affected
	// because Decide returns a copy, not the internal pointer.
	approved.Status = domain.ApprovalRejected
	approved.DecidedAt = nil

	list := store.List()
	assert.Len(t, list, 1)
	assert.Equal(t, domain.ApprovalApproved, list[0].Status, "stored item must not be affected by mutating returned value")
	assert.NotNil(t, list[0].DecidedAt, "stored DecidedAt must not be cleared by mutating returned value")
}

func TestExecutionStore_Update_Concurrent(t *testing.T) {
	store := NewExecutionStore()
	e := store.Add(domain.Execution{Tool: "test", Actor: "user", Role: domain.RoleViewer, Target: "t", Status: "started", Reason: "test"})

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = store.Update(e.ID, func(ex *domain.Execution) {
				ex.Status = "updated"
			})
		}()
	}
	wg.Wait()

	// Should have exactly one record (no duplication) and final status set
	list := store.List()
	assert.Len(t, list, 1)
	assert.Equal(t, "updated", list[0].Status)
}

func TestExecutionStore_Add_Concurrent(t *testing.T) {
	t.Parallel()
	store := NewExecutionStore()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			e := domain.Execution{
				Tool:      "tool",
				AuditID:   "aud",
				Actor:     "user",
				Role:      domain.RoleViewer,
				Target:    "t",
				Status:    "succeeded",
				Reason:    "test",
				CreatedAt: time.Now().UTC(),
			}
			result := store.Add(e)
			assert.NotEmpty(t, result.ID)
		}(i)
	}
	wg.Wait()
	list := store.List()
	assert.Len(t, list, 50)
}

func TestExecutionStore_Update_ConcurrentStress(t *testing.T) {
	t.Parallel()
	store := NewExecutionStore()
	e := store.Add(domain.Execution{Tool: "test", Actor: "user", Role: domain.RoleViewer, Target: "t", Status: "started", Reason: "test"})

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_ = store.Update(e.ID, func(ex *domain.Execution) {
				ex.Status = "done"
			})
		}(i)
	}
	wg.Wait()

	list := store.List()
	assert.Len(t, list, 1)
	assert.Equal(t, "done", list[0].Status)
}

func TestApprovalStore_Add_Concurrent(t *testing.T) {
	t.Parallel()
	store := NewApprovalStore()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			a := domain.Approval{
				ExecutionID: "exe-1",
				Tool:        "test",
				Actor:       "user",
				Target:      "t",
				Status:      domain.ApprovalPending,
				Reason:      "test",
				CreatedAt:   time.Now().UTC(),
			}
			result := store.Add(a)
			assert.NotEmpty(t, result.ID)
		}(i)
	}
	wg.Wait()
	list := store.List()
	assert.Len(t, list, 50)
}

func TestApprovalStore_Decide_Concurrent(t *testing.T) {
	t.Parallel()
	store := NewApprovalStore()
	added := store.Add(domain.Approval{ExecutionID: "exe-1", Tool: "test", Actor: "user", Target: "t", Status: domain.ApprovalPending, Reason: "test"})

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			approved, err := store.Decide(added.ID, domain.ApprovalApproved)
			assert.NoError(t, err)
			assert.Equal(t, domain.ApprovalApproved, approved.Status)
		}(i)
	}
	wg.Wait()

	list := store.List()
	assert.Len(t, list, 1)
}

func TestExecutionStore_MixedReadWrite_Concurrent(t *testing.T) {
	t.Parallel()
	store := NewExecutionStore()
	added := store.Add(domain.Execution{Tool: "test", Actor: "user", Role: domain.RoleViewer, Target: "t", Status: "started", Reason: "test"})

	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			if idx%3 == 0 {
				// Write
				_ = store.Update(added.ID, func(ex *domain.Execution) { ex.Status = "updated" })
			} else {
				// Read
				_, ok := store.Get(added.ID)
				assert.True(t, ok)
			}
		}(i)
	}
	wg.Wait()
	list := store.List()
	assert.Len(t, list, 1)
}

func TestUserStore_CRUDAndPasswordHash(t *testing.T) {
	store := NewUserStore()
	hash := []byte("[REDACTED]")
	created := store.Add(domain.User{Username: "alice", Nickname: "Alice", Role: domain.RoleAdmin, Status: "active"}, "ignored-plaintext", hash)

	assert.NotEmpty(t, created.ID)
	assert.False(t, created.CreatedAt.IsZero())
	assert.False(t, created.UpdatedAt.IsZero())

	listed := store.List()
	assert.Len(t, listed, 1)
	assert.Equal(t, "alice", listed[0].Username)

	byID, ok := store.Get(created.ID)
	assert.True(t, ok)
	assert.Equal(t, created.ID, byID.ID)

	byName, storedHash, ok := store.GetByUsername("alice")
	assert.True(t, ok)
	assert.Equal(t, created.ID, byName.ID)
	assert.Equal(t, string(hash), storedHash)

	err := store.Update(created.ID, func(u *domain.User) {
		u.Nickname = "Alice Updated"
		u.Email = "alice@example.test"
	})
	assert.NoError(t, err)
	updated, ok := store.Get(created.ID)
	assert.True(t, ok)
	assert.Equal(t, "Alice Updated", updated.Nickname)
	assert.Equal(t, "alice@example.test", updated.Email)
	assert.True(t, !updated.UpdatedAt.Before(created.UpdatedAt))

	err = store.SetPassword(created.ID, []byte("[REDACTED]"))
	assert.NoError(t, err)
	_, storedHash, ok = store.GetByUsername("alice")
	assert.True(t, ok)
	assert.Equal(t, "[REDACTED]", storedHash)

	assert.NoError(t, store.Delete(created.ID))
	_, ok = store.Get(created.ID)
	assert.False(t, ok)
	assert.Empty(t, store.List())
}

func TestUserStore_NotFoundBranches(t *testing.T) {
	store := NewUserStore()
	_, ok := store.Get("missing")
	assert.False(t, ok)
	_, _, ok = store.GetByUsername("missing")
	assert.False(t, ok)
	assert.Error(t, store.Update("missing", func(u *domain.User) {}))
	assert.Error(t, store.SetPassword("missing", []byte("hash")))
	assert.Error(t, store.Delete("missing"))
}

func TestJumpServerStore_CRUDSanitizesCredentials(t *testing.T) {
	store := NewJumpServerStore()
	created := store.Add(domain.JumpServerInstance{
		Name:     "primary",
		BaseURL:  "https://jump.example.test",
		AuthType: domain.JumpServerAuthToken,
	}, "[REDACTED]", "", "")

	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "active", created.Status)
	assert.True(t, created.HasCredential)
	assert.False(t, created.CreatedAt.IsZero())
	assert.False(t, created.UpdatedAt.IsZero())

	listed := store.List()
	assert.Len(t, listed, 1)
	assert.Equal(t, created.ID, listed[0].ID)
	assert.True(t, listed[0].HasCredential)

	got, ok := store.Get(created.ID)
	assert.True(t, ok)
	assert.Equal(t, created.ID, got.ID)
	assert.True(t, got.HasCredential)

	updated, err := store.Update(created.ID, func(j *domain.JumpServerInstance) {
		j.Name = "primary-updated"
		j.Status = "inactive"
	}, "", "[REDACTED]", "[REDACTED]")
	assert.NoError(t, err)
	assert.Equal(t, "primary-updated", updated.Name)
	assert.Equal(t, "inactive", updated.Status)
	assert.True(t, updated.HasCredential)

	checkedAt := time.Now().UTC()
	checked, err := store.MarkChecked(created.ID, "unreachable", checkedAt)
	assert.NoError(t, err)
	assert.Equal(t, "unreachable", checked.Status)
	assert.NotNil(t, checked.LastCheckedAt)
	assert.Equal(t, checkedAt, *checked.LastCheckedAt)

	assert.NoError(t, store.Delete(created.ID))
	_, ok = store.Get(created.ID)
	assert.False(t, ok)
}

func TestJumpServerStore_NotFoundBranchesAndCredentialFlag(t *testing.T) {
	store := NewJumpServerStore()
	created := store.Add(domain.JumpServerInstance{ID: "fixed", Name: "no-creds", BaseURL: "https://jump.example.test", Status: "inactive"}, "", "", "")
	assert.Equal(t, "fixed", created.ID)
	assert.False(t, created.HasCredential)
	assert.Equal(t, "inactive", created.Status)

	_, ok := store.Get("missing")
	assert.False(t, ok)
	_, err := store.Update("missing", func(j *domain.JumpServerInstance) {}, "", "", "")
	assert.Error(t, err)
	_, err = store.MarkChecked("missing", "active", time.Now().UTC())
	assert.Error(t, err)
	assert.Error(t, store.Delete("missing"))
}
