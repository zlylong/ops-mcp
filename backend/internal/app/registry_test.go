package app

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/zlylong/darwin-ops-mcp/backend/internal/adapters/kubernetes"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/adapters/linux"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/adapters/prometheus"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/audit"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/domain"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/policy"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/storage"
)

type mockRecorder struct{}

func (m *mockRecorder) Record(record domain.AuditRecord) domain.AuditRecord {
	record.ID = "aud-mock-123"
	return record
}
func (m *mockRecorder) List() []domain.AuditRecord { return nil }

func createTestRegistry() *Registry {
	return NewRegistry(policy.NewEngine(), audit.NewStore(nil), storage.NewExecutionStore(), storage.NewApprovalStore(), storage.NewUserStore(), storage.NewJumpServerStore(), domain.EnvDevelopment)
}

func TestRegistry_Register(t *testing.T) {
	engine := policy.NewEngine()
	recorder := &mockRecorder{}
	execStore := storage.NewExecutionStore()
	approvStore := storage.NewApprovalStore()
	registry := NewRegistry(engine, recorder, execStore, approvStore, storage.NewUserStore(), storage.NewJumpServerStore(), domain.EnvDevelopment)

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
	registry := NewRegistry(engine, recorder, execStore, approvStore, storage.NewUserStore(), storage.NewJumpServerStore(), domain.EnvDevelopment)

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
	registry := NewRegistry(engine, recorder, execStore, approvStore, storage.NewUserStore(), storage.NewJumpServerStore(), domain.EnvDevelopment)

	registry.Register(domain.Tool{Name: "test.tool", Risk: domain.RiskLow}, func(ctx context.Context, params map[string]any) (map[string]any, error) {
		return map[string]any{"result": "ok"}, nil
	})

	tool, ok := registry.Get("test.tool")
	assert.True(t, ok)
	assert.Equal(t, "test.tool", tool.Name)

	_, ok = registry.Get("nonexistent")
	assert.False(t, ok)
}

func TestRegistry_Execute_Completed(t *testing.T) {
	registry := createTestRegistry()
	registry.Register(domain.Tool{Name: "test.tool", ReadOnly: true, Risk: domain.RiskLow, InputSchema: map[string]domain.ParamSchema{"key": {Type: "string", Required: false}}}, func(ctx context.Context, params map[string]any) (map[string]any, error) {
		return map[string]any{"key": "value"}, nil
	})

	result, code, err := registry.Execute(context.Background(), "test.tool", domain.ExecuteRequest{Actor: "test-user", Role: domain.RoleViewer, Target: "local-dev", Parameters: map[string]any{"key": "value"}})
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	assert.NotEmpty(t, result.ExecutionID)
}

func TestRegistry_Execute_Denied(t *testing.T) {
	registry := createTestRegistry()
	registry.Register(domain.Tool{Name: "test.tool", ReadOnly: true, Risk: domain.RiskLow, InputSchema: map[string]domain.ParamSchema{"key": {Type: "string", Required: false}}}, func(ctx context.Context, params map[string]any) (map[string]any, error) {
		return map[string]any{"key": "value"}, nil
	})

	_, code, err := registry.Execute(context.Background(), "nonexistent", domain.ExecuteRequest{Actor: "test-user", Role: domain.RoleViewer, Target: "local-dev", Parameters: map[string]any{"key": "value"}})
	assert.Error(t, err)
	assert.Equal(t, http.StatusNotFound, code)
}

func TestRegistry_Execute_PendingApproval(t *testing.T) {
	registry := createTestRegistry()
	registry.Register(domain.Tool{Name: "test.tool", ReadOnly: false, Risk: domain.RiskMedium, InputSchema: map[string]domain.ParamSchema{"key": {Type: "string", Required: false}}}, func(ctx context.Context, params map[string]any) (map[string]any, error) {
		return map[string]any{"key": "value"}, nil
	})

	result, code, err := registry.Execute(context.Background(), "test.tool", domain.ExecuteRequest{Actor: "test-user", Role: domain.RoleOperator, Target: "local-dev", Parameters: map[string]any{"key": "value"}})
	assert.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, code)
	assert.NotEmpty(t, result.ExecutionID)
	assert.NotEmpty(t, result.ApprovalID)
}

func TestRegistry_Approvals(t *testing.T) {
	registry := createTestRegistry()
	now := time.Now()
	registry.AddApproval(domain.Approval{ID: "app-1", ExecutionID: "exec-123", Tool: "test.tool", Actor: "test-user", Target: "local-dev", Status: domain.ApprovalPending, Reason: "pending", CreatedAt: now})

	approvals := registry.Approvals()
	assert.Len(t, approvals, 1)
	assert.Equal(t, "app-1", approvals[0].ID)
}

func TestRegistry_Approve(t *testing.T) {
	registry := createTestRegistry()
	now := time.Now()
	registry.AddApproval(domain.Approval{ID: "app-1", ExecutionID: "exec-123", Tool: "test.tool", Actor: "test-user", Target: "local-dev", Status: domain.ApprovalPending, Reason: "pending", CreatedAt: now})

	approved, err := registry.Approve("app-1")
	assert.NoError(t, err)
	assert.Equal(t, domain.ApprovalApproved, approved.Status)
}

func TestRegistry_Reject(t *testing.T) {
	registry := createTestRegistry()
	now := time.Now()
	registry.AddApproval(domain.Approval{ID: "app-1", ExecutionID: "exec-123", Tool: "test.tool", Actor: "test-user", Target: "local-dev", Status: domain.ApprovalPending, Reason: "pending", CreatedAt: now})

	rejected, err := registry.Reject("app-1")
	assert.NoError(t, err)
	assert.Equal(t, domain.ApprovalRejected, rejected.Status)
}

func TestRegistry_ToolCRUD(t *testing.T) {
	registry := createTestRegistry()

	created, err := registry.CreateTool(domain.Tool{Name: "custom.echo", Description: "Echo params", Category: "custom", ReadOnly: true, Risk: domain.RiskLow, InputSchema: map[string]domain.ParamSchema{"message": {Type: "string", Required: false}}})
	assert.NoError(t, err)
	assert.Equal(t, "custom.echo", created.Name)

	updated, err := registry.UpdateTool("custom.echo", domain.Tool{Name: "custom.echo", Description: "Updated", Category: "custom", ReadOnly: true, Risk: domain.RiskMedium, InputSchema: map[string]domain.ParamSchema{"message": {Type: "string", Required: false}, "count": {Type: "number", Required: false}}})
	assert.NoError(t, err)
	assert.Equal(t, domain.RiskMedium, updated.Risk)
	assert.Equal(t, "Updated", updated.Description)

	result, code, err := registry.Execute(context.Background(), "custom.echo", domain.ExecuteRequest{Actor: "admin", Role: domain.RoleAdmin, Target: "local", Approved: true, Parameters: map[string]any{"message": "hello"}})
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, "custom.echo", result.Data["tool"])

	assert.NoError(t, registry.DeleteTool("custom.echo"))
	_, ok := registry.Get("custom.echo")
	assert.False(t, ok)
}

func TestRegistry_ToolCRUDValidation(t *testing.T) {
	registry := createTestRegistry()
	_, err := registry.CreateTool(domain.Tool{Name: "bad/tool", Risk: domain.RiskLow})
	assert.Error(t, err)
	_, err = registry.CreateTool(domain.Tool{Name: "bad.risk", Risk: domain.RiskLevel("extreme")})
	assert.Error(t, err)
	_, err = registry.UpdateTool("missing.tool", domain.Tool{Name: "missing.tool", Risk: domain.RiskLow})
	assert.Error(t, err)
	assert.Error(t, registry.DeleteTool("missing.tool"))
}

func TestRegistry_Execute_RequiresApprovalFlag(t *testing.T) {
	registry := createTestRegistry()
	err := registry.Register(domain.Tool{Name: "approval.flag", ReadOnly: true, Risk: domain.RiskLow, RequiresApproval: true}, func(ctx context.Context, params map[string]any) (map[string]any, error) {
		return map[string]any{"result": "ok"}, nil
	})
	assert.NoError(t, err)

	result, code, err := registry.Execute(context.Background(), "approval.flag", domain.ExecuteRequest{Actor: "viewer", Role: domain.RoleViewer, Target: "local", Parameters: map[string]any{"message": "hello"}})
	assert.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, code)
	assert.Equal(t, "pending_approval", result.Status)
	assert.Equal(t, "pending approval", result.Message)
	assert.NotEmpty(t, result.ExecutionID)
	assert.NotEmpty(t, result.ApprovalID)

	approvals := registry.Approvals()
	assert.Len(t, approvals, 1)
	assert.Equal(t, result.ExecutionID, approvals[0].ExecutionID)
	assert.Equal(t, "approval.flag", approvals[0].Tool)
}

func TestRegistry_Execute_HighRiskRequiresApproval(t *testing.T) {
	registry := createTestRegistry()
	err := registry.Register(domain.Tool{Name: "approval.high", ReadOnly: true, Risk: domain.RiskHigh}, func(ctx context.Context, params map[string]any) (map[string]any, error) {
		return map[string]any{"result": "ok"}, nil
	})
	assert.NoError(t, err)

	result, code, err := registry.Execute(context.Background(), "approval.high", domain.ExecuteRequest{Actor: "viewer", Role: domain.RoleViewer, Target: "local", Parameters: map[string]any{}})
	assert.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, code)
	assert.Equal(t, "pending_approval", result.Status)
	assert.NotEmpty(t, result.ApprovalID)
}

func TestRegisterMockTools_IncludesCommonLinuxTools(t *testing.T) {
	registry := createTestRegistry()
	err := RegisterMockTools(registry, kubernetes.NewMockAdapter(), prometheus.NewMockAdapter(), linux.NewMockAdapter())
	assert.NoError(t, err)

	linuxToolNames := []string{
		"linux.system_info",
		"linux.load_average",
		"linux.memory_usage",
		"linux.disk_usage",
		"linux.process_list",
		"linux.network_interfaces",
		"linux.service_status",
		"linux.journal_tail",
		"linux.ping",
		"linux.dns_lookup",
	}
	for _, name := range linuxToolNames {
		tool, ok := registry.Get(name)
		assert.True(t, ok, name)
		assert.Equal(t, "linux", tool.Category)
		assert.True(t, tool.ReadOnly)
	}

	result, code, err := registry.Execute(context.Background(), "linux.disk_usage", domain.ExecuteRequest{Actor: "viewer", Role: domain.RoleViewer, Target: "host=demo", Parameters: map[string]any{"path": "/var"}})
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, "/var", result.Data["path"])

	pending, code, err := registry.Execute(context.Background(), "linux.journal_tail", domain.ExecuteRequest{Actor: "viewer", Role: domain.RoleViewer, Target: "host=demo", Parameters: map[string]any{"unit": "darwin-ops-mcp-backend"}})
	assert.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, code)
	assert.Equal(t, "pending_approval", pending.Status)
	assert.NotEmpty(t, pending.ApprovalID)
}

func TestExecute_HandlerError(t *testing.T) {
	registry := createTestRegistry()
	tool := domain.Tool{Name: "error.tool", Description: "errors", Category: "test", ReadOnly: true, Risk: domain.RiskLow}
	wantErr := errors.New("handler failed")
	handler := func(ctx context.Context, params map[string]any) (map[string]any, error) {
		return nil, wantErr
	}
	_ = registry.Register(tool, handler)

	result, status, err := registry.Execute(context.Background(), "error.tool", domain.ExecuteRequest{Actor: "tester", Role: domain.RoleViewer, Target: "test", Parameters: map[string]any{}})
	assert.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
	assert.Equal(t, 500, status)
	assert.Equal(t, "error", result.Status)
	assert.NotEmpty(t, result.ExecutionID)

	executions := registry.Executions()
	var count int
	for _, e := range executions {
		if e.ID == result.ExecutionID {
			count++
		}
	}
	assert.Equal(t, 1, count, "exactly one execution record for the failed run")
}

func TestRegistry_ApproveExecutesPendingApproval(t *testing.T) {
	registry := createTestRegistry()
	err := registry.Register(domain.Tool{Name: "approval.exec", ReadOnly: true, Risk: domain.RiskMedium, RequiresApproval: true, InputSchema: map[string]domain.ParamSchema{"message": {Type: "string", Required: true}}}, func(ctx context.Context, params map[string]any) (map[string]any, error) {
		return map[string]any{"echo": params["message"]}, nil
	})
	assert.NoError(t, err)

	result, code, err := registry.Execute(context.Background(), "approval.exec", domain.ExecuteRequest{Actor: "agent", Role: domain.RoleViewer, Target: "host=demo", Parameters: map[string]any{"message": "hello"}})
	assert.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, code)

	approval, err := registry.Approve(result.ApprovalID)
	assert.NoError(t, err)
	assert.Equal(t, domain.ApprovalApproved, approval.Status)

	execution, ok := registry.Execution(result.ExecutionID)
	assert.True(t, ok)
	assert.Equal(t, "completed", execution.Status)
	assert.Equal(t, "approved by task approval", execution.Reason)
	assert.Equal(t, "hello", execution.Result["echo"])
	assert.NotEmpty(t, execution.AuditID)
}

func TestRegistry_ExecuteValidationFailure(t *testing.T) {
	registry := createTestRegistry()
	err := registry.Register(domain.Tool{Name: "validation.tool", ReadOnly: true, Risk: domain.RiskLow, InputSchema: map[string]domain.ParamSchema{"host": {Type: "string", Required: true}}}, func(ctx context.Context, params map[string]any) (map[string]any, error) {
		return map[string]any{"ok": true}, nil
	})
	assert.NoError(t, err)

	result, code, err := registry.Execute(context.Background(), "validation.tool", domain.ExecuteRequest{Actor: "agent", Role: domain.RoleViewer, Target: "host=demo", Parameters: map[string]any{}})
	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Equal(t, "validation_failed", result.Status)
	assert.Contains(t, result.Message, "missing required parameter")
}

func TestRegistry_ApproveApplicationCreatesRequestedTool(t *testing.T) {
	registry := createTestRegistry()
	application := registry.SubmitApplication(domain.ToolApplicationRequest{
		Tool:   "custom.requested",
		Risk:   domain.RiskHigh,
		Role:   domain.RoleOperator,
		Reason: "need new approved tool",
		Parameters: map[string]any{"toolDefinition": map[string]any{
			"name": "custom.requested", "description": "Requested custom tool", "category": "custom", "readOnly": true, "risk": "low", "requiresApproval": false,
			"inputSchema": map[string]any{"message": map[string]any{"type": "string", "required": false}},
		}},
	}, "agent")
	assert.Equal(t, domain.ApplicationPending, application.Status)

	approved, err := registry.ApproveApplication(application.ID)
	assert.NoError(t, err)
	assert.Equal(t, domain.ApplicationApproved, approved.Status)
	tool, ok := registry.Get("custom.requested")
	assert.True(t, ok)
	assert.Equal(t, "Requested custom tool", tool.Description)
}

func TestRegistry_AgentAPIKeyLifecycle(t *testing.T) {
	registry := createTestRegistry()

	created, err := registry.CreateAgentAPIKey(domain.AgentAPIKeyCreateRequest{
		Name:         "ci-agent",
		Actor:        "automation",
		Role:         domain.RoleOperator,
		Reason:       "coverage regression",
		Scopes:       []string{"tools:execute"},
		ExpiresInHrs: 1,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, created.ID)
	assert.NotEmpty(t, created.Secret)
	assert.Equal(t, "ci-agent", created.Name)
	assert.Equal(t, domain.RoleOperator, created.Role)
	assert.NotNil(t, created.ExpiresAt)
	assert.Equal(t, keyPrefix(created.Secret), created.KeyPrefix)
	assert.Equal(t, hashAgentAPISecret(created.Secret), hashAgentAPISecret(created.Secret))

	listed := registry.AgentAPIKeys()
	assert.Len(t, listed, 1)
	assert.Equal(t, created.ID, listed[0].ID)
	assert.Empty(t, listed[0].RevokedAt)

	authenticated, ok := registry.AuthenticateAgentAPIKey(created.Secret)
	assert.True(t, ok)
	assert.Equal(t, created.ID, authenticated.ID)
	assert.NotNil(t, authenticated.LastUsedAt)

	_, ok = registry.AuthenticateAgentAPIKey("[REDACTED]")
	assert.False(t, ok)
	_, ok = registry.AuthenticateAgentAPIKey("   ")
	assert.False(t, ok)

	revoked, err := registry.RevokeAgentAPIKey(" " + created.ID + " ")
	assert.NoError(t, err)
	assert.Equal(t, "revoked", revoked.Status)
	assert.NotNil(t, revoked.RevokedAt)
	_, ok = registry.AuthenticateAgentAPIKey(created.Secret)
	assert.False(t, ok)
}

func TestRegistry_AgentAPIKeyValidationAndExpiry(t *testing.T) {
	registry := createTestRegistry()

	_, err := registry.CreateAgentAPIKey(domain.AgentAPIKeyCreateRequest{Name: "", Actor: "agent", Role: domain.RoleViewer})
	assert.ErrorContains(t, err, "name is required")
	_, err = registry.CreateAgentAPIKey(domain.AgentAPIKeyCreateRequest{Name: "key", Actor: "", Role: domain.RoleViewer})
	assert.ErrorContains(t, err, "actor is required")
	_, err = registry.CreateAgentAPIKey(domain.AgentAPIKeyCreateRequest{Name: "key", Actor: "agent", Role: domain.Role("owner")})
	assert.ErrorContains(t, err, "role")

	expired, err := registry.CreateAgentAPIKey(domain.AgentAPIKeyCreateRequest{Name: "expired", Actor: "agent", Role: domain.RoleViewer, ExpiresInHrs: -1})
	assert.NoError(t, err)
	_, ok := registry.AuthenticateAgentAPIKey(expired.Secret)
	assert.True(t, ok, "negative ExpiresInHrs means non-expiring by current contract")

	assert.ErrorContains(t, func() error { _, err := registry.RevokeAgentAPIKey(""); return err }(), "key id is required")
	_, err = registry.RevokeAgentAPIKey("missing-key")
	assert.ErrorIs(t, err, ErrAgentAPIKeyNotFound)
}

func TestRegistry_ApplicationsAndRejectApplication(t *testing.T) {
	registry := createTestRegistry()

	low := registry.SubmitApplication(domain.ToolApplicationRequest{Tool: "safe.tool", Risk: domain.RiskLow, Role: domain.RoleViewer, Reason: "read-only access", DurationHrs: 0}, "alice")
	assert.Equal(t, domain.ApplicationApproved, low.Status)
	assert.Equal(t, 24, low.DurationHrs)

	high := registry.SubmitApplication(domain.ToolApplicationRequest{Tool: "danger.tool", Risk: domain.RiskCritical, Role: domain.RoleOperator, Reason: "break-glass", DurationHrs: 2}, "bob")
	assert.Equal(t, domain.ApplicationPending, high.Status)

	apps := registry.Applications()
	assert.Len(t, apps, 2)
	apps[0].Status = domain.ApplicationRejected
	fresh := registry.Applications()
	assert.Equal(t, domain.ApplicationApproved, fresh[0].Status, "Applications returns a copy")

	rejected, err := registry.RejectApplication(high.ID)
	assert.NoError(t, err)
	assert.Equal(t, domain.ApplicationRejected, rejected.Status)
	assert.Equal(t, "rejected by admin", rejected.Decision)
	assert.NotNil(t, rejected.DecidedAt)

	_, err = registry.RejectApplication("")
	assert.ErrorContains(t, err, "application id is required")
	_, err = registry.RejectApplication("missing-app")
	assert.ErrorIs(t, err, ErrApplicationNotFound)
}

func TestRegistry_AccessorStores(t *testing.T) {
	registry := createTestRegistry()
	assert.NotNil(t, registry.Users())
	assert.NotNil(t, registry.JumpServers())
	created := registry.JumpServers().Add(domain.JumpServerInstance{Name: "jump", BaseURL: "https://jump.example.test", AuthType: domain.JumpServerAuthToken}, "", "", "")
	got, ok := registry.JumpServers().Get(created.ID)
	assert.True(t, ok)
	assert.Equal(t, "jump", got.Name)
}
