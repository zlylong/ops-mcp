package app

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/zlylong/ops-mcp/backend/internal/audit"
	"github.com/zlylong/ops-mcp/backend/internal/domain"
	"github.com/zlylong/ops-mcp/backend/internal/policy"
	"github.com/zlylong/ops-mcp/backend/internal/storage"
)

type Handler func(context.Context, map[string]any) (map[string]any, error)

type registeredTool struct {
	tool    domain.Tool
	handler Handler
}

type Registry struct {
	tools       map[string]registeredTool
	policy      *policy.Engine
	auditor     audit.Recorder
	executions  *storage.ExecutionStore
	approvals   *storage.ApprovalStore
	environment domain.Environment
}

func NewRegistry(policyEngine *policy.Engine, auditor audit.Recorder, executions *storage.ExecutionStore, approvals *storage.ApprovalStore, env domain.Environment) *Registry {
	return &Registry{tools: make(map[string]registeredTool), policy: policyEngine, auditor: auditor, executions: executions, approvals: approvals, environment: env}
}

func (r *Registry) Register(tool domain.Tool, handler Handler) error {
	if strings.TrimSpace(tool.Name) == "" {
		return errors.New("tool name is required")
	}
	if handler == nil {
		return errors.New("tool handler is required")
	}
	if _, exists := r.tools[tool.Name]; exists {
		return fmt.Errorf("tool already registered: %s", tool.Name)
	}
	r.tools[tool.Name] = registeredTool{tool: tool, handler: handler}
	return nil
}

func (r *Registry) List() []domain.Tool {
	out := make([]domain.Tool, 0, len(r.tools))
	for _, item := range r.tools {
		out = append(out, item.tool)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
func (r *Registry) Get(name string) (domain.Tool, bool) {
	item, ok := r.tools[name]
	return item.tool, ok
}

func (r *Registry) Execute(ctx context.Context, name string, req domain.ExecuteRequest) (domain.ExecuteResult, int, error) {
	item, ok := r.tools[name]
	if !ok {
		return domain.ExecuteResult{Status: "not_found", Message: "tool not found"}, 404, errors.New("tool not found")
	}
	if req.Actor == "" {
		req.Actor = "anonymous"
	}
	if req.Role == "" {
		req.Role = domain.RoleViewer
	}
	if req.Parameters == nil {
		req.Parameters = map[string]any{}
	}
	if err := validateInput(item.tool, req.Parameters); err != nil {
		exe := domain.Execution{ID: newID("exe"), Tool: name, Actor: req.Actor, Role: req.Role, Target: req.Target, Status: "validation_failed", Reason: err.Error(), Parameters: req.Parameters}
		aud := r.auditor.Record(domain.AuditRecord{ExecutionID: exe.ID, Actor: req.Actor, Role: req.Role, Action: name, Target: req.Target, Allowed: false, Reason: err.Error(), Parameters: req.Parameters})
		exe.AuditID = aud.ID
		exe = r.executions.Add(exe)
		return domain.ExecuteResult{ExecutionID: exe.ID, AuditID: aud.ID, Status: "validation_failed", Message: err.Error()}, 400, err
	}
	decision := r.policy.Evaluate(domain.PolicyRequest{Tool: item.tool, Actor: req.Actor, Role: req.Role, Environment: r.environment, Approved: req.Approved})
	if !decision.Allowed {
		exe := domain.Execution{ID: newID("exe"), Tool: name, Actor: req.Actor, Role: req.Role, Target: req.Target, Status: "blocked", Reason: decision.Reason, Parameters: req.Parameters}
		aud := r.auditor.Record(domain.AuditRecord{ExecutionID: exe.ID, Actor: req.Actor, Role: req.Role, Action: name, Target: req.Target, Allowed: false, Reason: decision.Reason, Parameters: req.Parameters})
		exe.AuditID = aud.ID
		exe = r.executions.Add(exe)
		result := domain.ExecuteResult{ExecutionID: exe.ID, AuditID: aud.ID, Status: "blocked", Message: decision.Reason}
		status := 403
		if decision.RequiresApproval {
			approval := r.approvals.Add(domain.Approval{ExecutionID: exe.ID, Tool: name, Actor: req.Actor, Target: req.Target, Status: domain.ApprovalPending, Reason: decision.Reason})
			result.ApprovalID = approval.ID
			result.Status = "approval_required"
			status = 409
		}
		return result, status, errors.New(decision.Reason)
	}
	data, err := item.handler(ctx, req.Parameters)
	status := "succeeded"
	reason := "executed"
	if err != nil {
		status = "failed"
		reason = err.Error()
	}
	exe := domain.Execution{ID: newID("exe"), Tool: name, Actor: req.Actor, Role: req.Role, Target: req.Target, Status: status, Reason: reason, Parameters: req.Parameters, Result: data}
	aud := r.auditor.Record(domain.AuditRecord{ExecutionID: exe.ID, Actor: req.Actor, Role: req.Role, Action: name, Target: req.Target, Allowed: err == nil, Reason: reason, Parameters: req.Parameters})
	exe.AuditID = aud.ID
	exe = r.executions.Add(exe)
	if err != nil {
		return domain.ExecuteResult{ExecutionID: exe.ID, AuditID: aud.ID, Status: "failed", Message: err.Error()}, 500, err
	}
	return domain.ExecuteResult{ExecutionID: exe.ID, AuditID: aud.ID, Status: "succeeded", Message: "tool executed", Data: data}, 200, nil
}

func validateInput(tool domain.Tool, params map[string]any) error {
	for name, typ := range tool.InputSchema {
		if strings.HasSuffix(typ, "?") {
			continue
		}
		if _, ok := params[name]; !ok {
			return fmt.Errorf("missing required parameter: %s", name)
		}
	}
	for name, value := range params {
		typ, ok := tool.InputSchema[name]
		if !ok {
			continue
		}
		typ = strings.TrimSuffix(typ, "?")
		switch typ {
		case "string":
			if _, ok := value.(string); !ok {
				return fmt.Errorf("parameter %s must be string", name)
			}
		case "number":
			switch value.(type) {
			case float64, float32, int, int64, int32:
			default:
				return fmt.Errorf("parameter %s must be number", name)
			}
		}
	}
	return nil
}

func newID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, timeNowUnixNano())
}

var timeNowUnixNano = func() int64 { return timeNow().UnixNano() }
var timeNow = func() time.Time { return time.Now().UTC() }

func (r *Registry) Executions() []domain.Execution               { return r.executions.List() }
func (r *Registry) Execution(id string) (domain.Execution, bool) { return r.executions.Get(id) }
func (r *Registry) Approvals() []domain.Approval                 { return r.approvals.List() }
func (r *Registry) Approve(id string) (domain.Approval, error) {
	return r.approvals.Decide(id, domain.ApprovalApproved)
}
func (r *Registry) Reject(id string) (domain.Approval, error) {
	return r.approvals.Decide(id, domain.ApprovalRejected)
}
