package ops

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/example/ops-mcp/backend/internal/audit"
)

type Service struct {
	mode    string
	auditor audit.Recorder
}

func NewService(mode string, auditor audit.Recorder) *Service {
	if mode == "" {
		mode = "mock"
	}
	return &Service{mode: mode, auditor: auditor}
}

func (s *Service) Overview(environment string) Overview {
	return Overview{Mode: s.mode, Clusters: len(s.Clusters()), Namespaces: len(s.Namespaces()), Workloads: len(s.Workloads()), Alerts: 2, Environment: environment}
}

func (s *Service) Clusters() []Cluster {
	return []Cluster{{Name: "local-dev", Status: "Healthy", Version: "v1.30.0-mock", Nodes: 3}, {Name: "edge-prod", Status: "Mocked", Version: "v1.29.4-mock", Nodes: 5}}
}

func (s *Service) Namespaces() []Namespace {
	return []Namespace{{Name: "default", Phase: "Active", Age: "18d", Cluster: "local-dev"}, {Name: "observability", Phase: "Active", Age: "14d", Cluster: "local-dev"}, {Name: "payments", Phase: "Active", Age: "4d", Cluster: "edge-prod"}}
}

func (s *Service) Workloads() []Workload {
	return []Workload{{Name: "api", Namespace: "default", Kind: "Deployment", Ready: "3/3", Image: "ops-mcp/api:mock"}, {Name: "prometheus", Namespace: "observability", Kind: "StatefulSet", Ready: "1/1", Image: "prom/prometheus:mock"}, {Name: "worker", Namespace: "payments", Kind: "Deployment", Ready: "2/3", Image: "ops-mcp/worker:mock"}}
}

func (s *Service) Tools() []Tool {
	return []Tool{
		{Name: "get_cluster_health", Description: "Read-only health summary", Write: false, RequiresApproval: false},
		{Name: "restart_rollout", Description: "Restart a Kubernetes deployment rollout", Write: true, RequiresApproval: true},
		{Name: "scale_workload", Description: "Scale a deployment/statefulset replica count", Write: true, RequiresApproval: true},
	}
}

var forbiddenTools = map[string]string{
	"shell":            "arbitrary shell execution is prohibited",
	"kubectl_exec":     "kubectl exec is prohibited",
	"delete_namespace": "namespace deletion is prohibited",
	"delete_pvc":       "PVC deletion is prohibited",
}

func (s *Service) Execute(req ToolRequest, environment string) (ToolResult, int, error) {
	actor := strings.TrimSpace(req.Actor)
	if actor == "" {
		actor = "anonymous"
	}
	auditID := fmt.Sprintf("aud-%d", time.Now().UnixNano())
	event := audit.Event{ID: auditID, At: time.Now().UTC(), Actor: actor, Action: req.Tool, Target: req.Target, Approved: req.Approved}

	if reason, blocked := forbiddenTools[req.Tool]; blocked {
		event.Allowed = false
		event.Reason = reason
		s.auditor.Record(event)
		return ToolResult{AuditID: auditID, Status: "blocked", Message: reason}, 403, errors.New(reason)
	}
	allowed := false
	for _, tool := range s.Tools() {
		if tool.Name == req.Tool {
			allowed = true
			if tool.Write && environment == "production" && !req.Approved {
				reason := "production write operations require approval"
				event.Allowed = false
				event.Reason = reason
				s.auditor.Record(event)
				return ToolResult{AuditID: auditID, Status: "approval_required", Message: reason}, 409, errors.New(reason)
			}
		}
	}
	if !allowed {
		reason := "unknown or unsupported tool"
		event.Allowed = false
		event.Reason = reason
		s.auditor.Record(event)
		return ToolResult{AuditID: auditID, Status: "blocked", Message: reason}, 404, errors.New(reason)
	}
	event.Allowed = true
	event.Reason = "mock execution recorded"
	s.auditor.Record(event)
	return ToolResult{AuditID: auditID, Status: "ok", Message: "mock mode: no real cluster mutation was performed"}, 200, nil
}
