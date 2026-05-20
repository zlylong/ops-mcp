package app

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/zlylong/darwin-ops-mcp/backend/internal/audit"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/domain"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/policy"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/storage"
)

// Sentinel errors for registry operations.
var (
	ErrToolNotFound  = errors.New("tool not found")
	ErrAlreadyExists = errors.New("tool already exists")
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
	if err := validateTool(tool); err != nil {
		return err
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

func (r *Registry) CreateTool(tool domain.Tool) (domain.Tool, error) {
	if err := r.Register(tool, customToolHandler(tool.Name)); err != nil {
		return domain.Tool{}, err
	}
	return tool, nil
}

func (r *Registry) UpdateTool(name string, tool domain.Tool) (domain.Tool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Tool{}, errors.New("tool name is required")
	}
	if strings.TrimSpace(tool.Name) == "" {
		tool.Name = name
	}
	if tool.Name != name {
		return domain.Tool{}, errors.New("tool name cannot be changed")
	}
	existing, exists := r.tools[name]
	if !exists {
		return domain.Tool{}, fmt.Errorf("%w: %s", ErrToolNotFound, name)
	}
	if err := validateTool(tool); err != nil {
		return domain.Tool{}, err
	}
	r.tools[name] = registeredTool{tool: tool, handler: existing.handler}
	return tool, nil
}

func (r *Registry) DeleteTool(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("tool name is required")
	}
	if _, exists := r.tools[name]; !exists {
		return fmt.Errorf("%w: %s", ErrToolNotFound, name)
	}
	delete(r.tools, name)
	return nil
}

func validateTool(tool domain.Tool) error {
	if strings.TrimSpace(tool.Name) == "" {
		return errors.New("tool name is required")
	}
	if strings.ContainsAny(tool.Name, " /\\") {
		return errors.New("tool name cannot contain spaces or slashes")
	}
	switch tool.Risk {
	case "", domain.RiskLow, domain.RiskMedium, domain.RiskHigh, domain.RiskCritical:
		return nil
	default:
		return fmt.Errorf("invalid risk level: %s", tool.Risk)
	}
}

func requiresExecutionApproval(tool domain.Tool) bool {
	if tool.RequiresApproval {
		return true
	}
	switch tool.Risk {
	case domain.RiskMedium, domain.RiskHigh, domain.RiskCritical:
		return true
	default:
		return false
	}
}

func approvalReason(tool domain.Tool) string {
	if tool.RequiresApproval {
		return "pending approval by tool configuration"
	}
	return "pending approval for " + string(tool.Risk)
}

func customToolHandler(name string) Handler {
	return func(ctx context.Context, params map[string]any) (map[string]any, error) {
		return map[string]any{"tool": name, "parameters": params, "message": "custom tool executed"}, nil
	}
}

func (r *Registry) AddApproval(approval domain.Approval) domain.Approval {
	return r.approvals.Add(approval)
}

func (r *Registry) List() []domain.Tool {
	out := make([]domain.Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t.tool)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (r *Registry) Get(name string) (domain.Tool, bool) {
	t, ok := r.tools[name]
	return t.tool, ok
}

func (r *Registry) Execute(ctx context.Context, name string, req domain.ExecuteRequest) (domain.ExecuteResult, int, error) {
	result := domain.ExecuteResult{
		ExecutionID: "",
		AuditID:     "",
		ApprovalID:  "",
	}
	t, ok := r.tools[name]
	if !ok {
		result.Message = "tool not found"
		return result, 404, errors.New(result.Message)
	}
	policyReq := domain.PolicyRequest{
		Tool:        t.tool,
		Actor:       req.Actor,
		Role:        req.Role,
		Environment: r.environment,
		Approved:    req.Approved,
	}
	decision := r.policy.Evaluate(policyReq)
	if !decision.Allowed {
		result.Message = "policy denied"
		exe := r.executions.Add(domain.Execution{
			Tool:       name,
			Actor:      req.Actor,
			Role:       req.Role,
			Target:     req.Target,
			Status:     "denied",
			Reason:     result.Message,
			Parameters: req.Parameters,
		})
		result.ExecutionID = exe.ID
		record := domain.AuditRecord{
			ExecutionID: exe.ID,
			Actor:       req.Actor,
			Role:        req.Role,
			Action:      "tool." + name,
			Target:      req.Target,
			Allowed:     false,
			Reason:      result.Message,
		}
		record = r.auditor.Record(record)
		result.AuditID = record.ID
		return result, 403, errors.New(result.Message)
	}
	if requiresExecutionApproval(t.tool) && !req.Approved {
		exe := r.executions.Add(domain.Execution{
			Tool:       name,
			Actor:      req.Actor,
			Role:       req.Role,
			Target:     req.Target,
			Status:     "pending_approval",
			Reason:     "pending approval",
			Parameters: req.Parameters,
		})
		approval := r.approvals.Add(domain.Approval{
			ExecutionID: exe.ID,
			Tool:        name,
			Actor:       req.Actor,
			Target:      req.Target,
			Status:      domain.ApprovalPending,
			Reason:      approvalReason(t.tool),
		})
		result.ExecutionID = exe.ID
		result.ApprovalID = approval.ID
		result.Status = "pending_approval"
		result.Message = "pending approval"
		return result, 202, nil
	}
	exe := r.executions.Add(domain.Execution{
		Tool:       name,
		Actor:      req.Actor,
		Role:       req.Role,
		Target:     req.Target,
		Status:     "completed",
		Parameters: req.Parameters,
	})
	result.ExecutionID = exe.ID
	result.Status = "completed"
	result.Message = "completed"
	output, err := t.handler(ctx, req.Parameters)
	if err != nil {
		exe.Status = "error"
		exe.Reason = err.Error()
		result.Message = err.Error()
		result.Status = "error"
		r.executions.Update(exe.ID, func(e *domain.Execution) { e.Status = "error"; e.Reason = err.Error() })
		record := domain.AuditRecord{
			ExecutionID: exe.ID,
			Actor:       req.Actor,
			Role:        req.Role,
			Action:      "tool." + name,
			Target:      req.Target,
			Allowed:     true,
			Reason:      "error in handler",
		}
		r.auditor.Record(record)
		return result, 500, err
	}
	exe.Result = output
		r.executions.Update(exe.ID, func(e *domain.Execution) { e.Result = output })
	record := domain.AuditRecord{
		ExecutionID: exe.ID,
		Actor:       req.Actor,
		Role:        req.Role,
		Action:      "tool." + name,
		Target:      req.Target,
		Allowed:     true,
		Reason:      "approved",
	}
	r.auditor.Record(record)
	result.ExecutionID = exe.ID
	result.AuditID = record.ID
	result.Data = output
	return result, 200, nil
}

func (r *Registry) Executions() []domain.Execution               { return r.executions.List() }
func (r *Registry) Execution(id string) (domain.Execution, bool) { return r.executions.Get(id) }
func (r *Registry) Approvals() []domain.Approval                 { return r.approvals.List() }
func (r *Registry) Approve(id string) (domain.Approval, error) {
	return r.approvals.Decide(id, domain.ApprovalApproved)
}
func (r *Registry) Reject(id string) (domain.Approval, error) {
	return r.approvals.Decide(id, domain.ApprovalRejected)
}
