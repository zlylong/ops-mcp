package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/zlylong/darwin-ops-mcp/backend/internal/audit"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/domain"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/policy"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/storage"
)

// Sentinel errors for registry operations.
var (
	ErrToolNotFound        = errors.New("tool not found")
	ErrAlreadyExists       = errors.New("tool already exists")
	ErrApplicationNotFound = errors.New("application not found")
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
	appStore    []domain.ToolApplication
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
	if err := t.tool.ValidateParams(req.Parameters); err != nil {
		result.Message = err.Error()
		exe := r.executions.Add(domain.Execution{
			Tool:       name,
			Actor:      req.Actor,
			Role:       req.Role,
			Target:     req.Target,
			Status:     "validation_failed",
			Reason:     result.Message,
			Parameters: req.Parameters,
		})
		result.ExecutionID = exe.ID
		record := r.auditor.Record(domain.AuditRecord{
			ExecutionID: exe.ID,
			Actor:       req.Actor,
			Role:        req.Role,
			Action:      "tool." + name,
			Target:      req.Target,
			Allowed:     false,
			Reason:      result.Message,
			Parameters:  req.Parameters,
		})
		result.AuditID = record.ID
		result.Status = "validation_failed"
		return result, 400, err
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
	record = r.auditor.Record(record)
	result.ExecutionID = exe.ID
	result.AuditID = record.ID
	_ = r.executions.Update(exe.ID, func(e *domain.Execution) { e.AuditID = record.ID })
	result.Data = output
	return result, 200, nil
}

func (r *Registry) Executions() []domain.Execution               { return r.executions.List() }
func (r *Registry) Execution(id string) (domain.Execution, bool) { return r.executions.Get(id) }
func (r *Registry) Approvals() []domain.Approval                 { return r.approvals.List() }
func (r *Registry) Approve(id string) (domain.Approval, error) {
	approval, err := r.approvals.Decide(id, domain.ApprovalApproved)
	if err != nil {
		return approval, err
	}
	_ = r.executeApprovedApproval(context.Background(), approval)
	return approval, nil
}
func (r *Registry) Reject(id string) (domain.Approval, error) {
	approval, err := r.approvals.Decide(id, domain.ApprovalRejected)
	if err != nil {
		return approval, err
	}
	_ = r.executions.Update(approval.ExecutionID, func(e *domain.Execution) {
		e.Status = "rejected"
		e.Reason = "rejected by task approval"
	})
	return approval, nil
}

func (r *Registry) executeApprovedApproval(ctx context.Context, approval domain.Approval) error {
	exe, ok := r.executions.Get(approval.ExecutionID)
	if !ok || exe.Status != "pending_approval" {
		return nil
	}
	registered, ok := r.tools[approval.Tool]
	if !ok {
		_ = r.executions.Update(exe.ID, func(e *domain.Execution) {
			e.Status = "error"
			e.Reason = "approved tool no longer exists"
		})
		return ErrToolNotFound
	}
	output, err := registered.handler(ctx, exe.Parameters)
	if err != nil {
		_ = r.executions.Update(exe.ID, func(e *domain.Execution) {
			e.Status = "error"
			e.Reason = err.Error()
			e.Result = output
		})
		record := r.auditor.Record(domain.AuditRecord{
			ExecutionID: exe.ID,
			Actor:       exe.Actor,
			Role:        exe.Role,
			Action:      "tool." + approval.Tool,
			Target:      exe.Target,
			Allowed:     true,
			Reason:      "approved by task approval; handler error",
			Parameters:  exe.Parameters,
		})
		_ = r.executions.Update(exe.ID, func(e *domain.Execution) { e.AuditID = record.ID })
		return err
	}
	record := r.auditor.Record(domain.AuditRecord{
		ExecutionID: exe.ID,
		Actor:       exe.Actor,
		Role:        exe.Role,
		Action:      "tool." + approval.Tool,
		Target:      exe.Target,
		Allowed:     true,
		Reason:      "approved by task approval",
		Parameters:  exe.Parameters,
	})
	_ = r.executions.Update(exe.ID, func(e *domain.Execution) {
		e.Status = "completed"
		e.Reason = "approved by task approval"
		e.Result = output
		e.AuditID = record.ID
	})
	return nil
}

// SubmitApplication records a tool access application and returns it.
// High-risk (high/critical) tools are auto-set to pending; others default to approved.
func (r *Registry) SubmitApplication(req domain.ToolApplicationRequest, actor string) domain.ToolApplication {
	duration := req.DurationHrs
	if duration <= 0 {
		duration = 24
	}
	status := domain.ApplicationApproved
	decision := "auto-approved (low/medium risk)"
	if req.Risk == domain.RiskHigh || req.Risk == domain.RiskCritical {
		status = domain.ApplicationPending
		decision = "pending review (high/critical risk)"
	}
	app := domain.ToolApplication{
		ID:          "app-" + strconv.FormatInt(time.Now().UTC().UnixNano(), 36),
		Tool:        req.Tool,
		Risk:        req.Risk,
		Role:        req.Role,
		Actor:       actor,
		Reason:      req.Reason,
		Status:      status,
		Decision:    decision,
		DurationHrs: duration,
		Parameters:  req.Parameters,
		CreatedAt:   time.Now().UTC(),
	}
	r.appStore = append(r.appStore, app)
	return app
}

// Applications returns all tool access applications in creation order (newest last).
func (r *Registry) Applications() []domain.ToolApplication {
	out := make([]domain.ToolApplication, len(r.appStore))
	copy(out, r.appStore)
	return out
}

func (r *Registry) decideApplication(id string, status domain.ApplicationStatus, decision string) (domain.ToolApplication, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.ToolApplication{}, errors.New("application id is required")
	}
	now := time.Now().UTC()
	for i := range r.appStore {
		if r.appStore[i].ID == id {
			r.appStore[i].Status = status
			r.appStore[i].Decision = decision
			r.appStore[i].DecidedAt = &now
			return r.appStore[i], nil
		}
	}
	return domain.ToolApplication{}, fmt.Errorf("%w: %s", ErrApplicationNotFound, id)
}

// ApproveApplication approves a tool application. If the application contains
// parameters.toolDefinition, the approved tool is registered into the runtime registry.
func (r *Registry) ApproveApplication(id string) (domain.ToolApplication, error) {
	application, err := r.decideApplication(id, domain.ApplicationApproved, "approved by admin")
	if err != nil {
		return application, err
	}
	if raw, ok := application.Parameters["toolDefinition"]; ok && raw != nil {
		buf, err := json.Marshal(raw)
		if err != nil {
			return application, err
		}
		var tool domain.Tool
		if err := json.Unmarshal(buf, &tool); err != nil {
			return application, err
		}
		if strings.TrimSpace(tool.Name) == "" {
			tool.Name = application.Tool
		}
		if _, exists := r.Get(tool.Name); !exists {
			if _, err := r.CreateTool(tool); err != nil {
				return application, err
			}
		}
	}
	return application, nil
}

// RejectApplication rejects a tool access application.
func (r *Registry) RejectApplication(id string) (domain.ToolApplication, error) {
	return r.decideApplication(id, domain.ApplicationRejected, "rejected by admin")
}
