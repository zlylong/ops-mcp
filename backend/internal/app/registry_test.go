package app

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/zlylong/ops-mcp/backend/internal/domain"
	"github.com/zlylong/ops-mcp/backend/internal/policy"
	"github.com/zlylong/ops-mcp/backend/internal/storage"
)

type mockRecorder struct{}

func (m *mockRecorder) Record(record domain.AuditRecord) domain.AuditRecord {
	record.ID = "aud-mock-123"
	return record
}
func (m *mockRecorder) List() []domain.AuditRecord { return nil }

func TestRegistry_Register(t *testing.T) {
	engine := policy.NewEngine()
	recorder := &mockRecorder{}
	execStore := storage.NewExecutionStore()
	approvStore := storage.NewApprovalStore()
	registry := NewRegistry(engine, recorder, execStore, approvStore, domain.EnvDevelopment)

	err := registry.Register(domain.Tool{Name: "test.tool", Risk: domain.RiskLow}, func(ctx context.Context, params map[string]any) (map[string]any, error) {
		return map[string]any{"result": "ok"}, nil
	})
	assert.NoError(t, err)

	err = registry.Register(domain.Tool{Name: "test.tool", Risk: domain.RiskLow}, func(ctx context.Context, params map[string]any) (map[string]any, error) {
		return map[string]any{"result": "ok"}, nil
	})
	assert.Error(t, err)

	err = registry.Register(domain.Tool{Name: "", Risk: domain.RiskLow}, func(ctx context.Context, params map[string]any) (map[string]any, error) {
		return map[string]any{"result": "ok"}, nil
	})
	assert.Error(t, err)
}

func TestRegistry_List(t *testing.T) {
	engine := policy.NewEngine()
	recorder := &mockRecorder{}
	execStore := storage.NewExecutionStore()
	approvStore := storage.NewApprovalStore()
	registry := NewRegistry(engine, recorder, execStore, approvStore, domain.EnvDevelopment)

	registry.Register(domain.Tool{Name: "b.tool", Risk: domain.RiskLow}, func(ctx context.Context, params map[string]any) (map[string]any, error) {
		return map[string]any{"result": "ok"}, nil
	})
	registry.Register(domain.Tool{Name: "a.tool", Risk: domain.RiskLow}, func(ctx context.Context, params map[string]any) (map[string]any, error) {
		return map[string]any{"result": "ok"}, nil
	})
	registry.Register(domain.Tool{Name: "c.tool", Risk: domain.RiskLow}, func(ctx context.Context, params map[string]any) (map[string]any, error) {
		return map[string]any{"result": "ok"}, nil
	})

	tools := registry.List()
	assert.Len(t, tools, 3)
	assert.Equal(t, "a.tool", tools[0].Name)
	assert.Equal(t, "b.tool", tools[1].Name)
	assert.Equal(t, "c.tool", tools[2].Name)
}

func TestRegistry_Get(t *testing.T) {
	engine := policy.NewEngine()
	recorder := &mockRecorder{}
	execStore := storage.NewExecutionStore()
	approvStore := storage.NewApprovalStore()
	registry := NewRegistry(engine, recorder, execStore, approvStore, domain.EnvDevelopment)

	registry.Register(domain.Tool{Name: "test.tool", Risk: domain.RiskLow}, func(ctx context.Context, params map[string]any) (map[string]any, error) {
		return map[string]any{"result": "ok"}, nil
	})

	tool, ok := registry.Get("test.tool")
	assert.True(t, ok)
	assert.Equal(t, "test.tool", tool.Name)

	tool, ok = registry.Get("nonexistent")
	assert.False(t, ok)
	assert.Empty(t, tool.Name)
}

func TestRegistry_Execute_Completed(t *testing.T) {
	engine := policy.NewEngine()
	recorder := &mockRecorder{}
	execStore := storage.NewExecutionStore()
	approvStore := storage.NewApprovalStore()
	registry := NewRegistry(engine, recorder, execStore, approvStore, domain.EnvDevelopment)

	// 使用 ReadOnly 工具，Viewer 可以执行
	registry.Register(domain.Tool{Name: "test.tool", ReadOnly: true, Risk: domain.RiskLow, InputSchema: map[string]string{"key": "string"}}, func(ctx context.Context, params map[string]any) (map[string]any, error) {
		return map[string]any{"key": "value"}, nil
	})

	req := domain.ExecuteRequest{
		Actor:      "test-user",
		Role:       domain.RoleViewer,
		Target:     "local-dev",
		Parameters: map[string]any{"key": "value"},
	}

	result, code, err := registry.Execute(context.Background(), "test.tool", req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	assert.NotEmpty(t, result.ExecutionID)
}

func TestRegistry_Execute_Denied(t *testing.T) {
	engine := policy.NewEngine()
	recorder := &mockRecorder{}
	execStore := storage.NewExecutionStore()
	approvStore := storage.NewApprovalStore()
	registry := NewRegistry(engine, recorder, execStore, approvStore, domain.EnvDevelopment)

	registry.Register(domain.Tool{Name: "test.tool", ReadOnly: true, Risk: domain.RiskLow, InputSchema: map[string]string{"key": "string"}}, func(ctx context.Context, params map[string]any) (map[string]any, error) {
		return map[string]any{"key": "value"}, nil
	})

	req := domain.ExecuteRequest{
		Actor:      "test-user",
		Role:       domain.RoleViewer,
		Target:     "local-dev",
		Parameters: map[string]any{"key": "value"},
	}

	result, code, err := registry.Execute(context.Background(), "nonexistent", req)
	assert.Error(t, err)
	assert.Equal(t, http.StatusNotFound, code)
	assert.Empty(t, result.ExecutionID)
}

func TestRegistry_Execute_PendingApproval(t *testing.T) {
	engine := policy.NewEngine()
	recorder := &mockRecorder{}
	execStore := storage.NewExecutionStore()
	approvStore := storage.NewApprovalStore()
	registry := NewRegistry(engine, recorder, execStore, approvStore, domain.EnvDevelopment)

	registry.Register(domain.Tool{Name: "test.tool", ReadOnly: false, Risk: domain.RiskMedium, InputSchema: map[string]string{"key": "string"}}, func(ctx context.Context, params map[string]any) (map[string]any, error) {
		return map[string]any{"key": "value"}, nil
	})

	req := domain.ExecuteRequest{
		Actor:      "test-user",
		Role:       domain.RoleOperator,
		Target:     "local-dev",
		Parameters: map[string]any{"key": "value"},
	}

	result, code, err := registry.Execute(context.Background(), "test.tool", req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, code)
	assert.NotEmpty(t, result.ExecutionID)
	assert.NotEmpty(t, result.ApprovalID)
}

func TestRegistry_Approvals(t *testing.T) {
	engine := policy.NewEngine()
	recorder := &mockRecorder{}
	execStore := storage.NewExecutionStore()
	approvStore := storage.NewApprovalStore()
	registry := NewRegistry(engine, recorder, execStore, approvStore, domain.EnvDevelopment)

	now := time.Now()
	approval := domain.Approval{
		ID:          "app-1",
		ExecutionID: "exec-123",
		Tool:        "test.tool",
		Actor:       "test-user",
		Target:      "local-dev",
		Status:      domain.ApprovalPending,
		Reason:      "pending",
		CreatedAt:   now,
	}
	registry.AddApproval(approval)

	approvals := registry.Approvals()
	assert.Len(t, approvals, 1)
	assert.Equal(t, "app-1", approvals[0].ID)
}

func TestRegistry_Approve(t *testing.T) {
	engine := policy.NewEngine()
	recorder := &mockRecorder{}
	execStore := storage.NewExecutionStore()
	approvStore := storage.NewApprovalStore()
	registry := NewRegistry(engine, recorder, execStore, approvStore, domain.EnvDevelopment)

	now := time.Now()
	approval := domain.Approval{
		ID:          "app-1",
		ExecutionID: "exec-123",
		Tool:        "test.tool",
		Actor:       "test-user",
		Target:      "local-dev",
		Status:      domain.ApprovalPending,
		Reason:      "pending",
		CreatedAt:   now,
	}
	registry.AddApproval(approval)

	approved, err := registry.Approve("app-1")
	assert.NoError(t, err)
	assert.Equal(t, domain.ApprovalApproved, approved.Status)
}

func TestRegistry_Reject(t *testing.T) {
	engine := policy.NewEngine()
	recorder := &mockRecorder{}
	execStore := storage.NewExecutionStore()
	approvStore := storage.NewApprovalStore()
	registry := NewRegistry(engine, recorder, execStore, approvStore, domain.EnvDevelopment)

	now := time.Now()
	approval := domain.Approval{
		ID:          "app-1",
		ExecutionID: "exec-123",
		Tool:        "test.tool",
		Actor:       "test-user",
		Target:      "local-dev",
		Status:      domain.ApprovalPending,
		Reason:      "pending",
		CreatedAt:   now,
	}
	registry.AddApproval(approval)

	rejected, err := registry.Reject("app-1")
	assert.NoError(t, err)
	assert.Equal(t, domain.ApprovalRejected, rejected.Status)
}
