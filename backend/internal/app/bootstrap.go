package app

import (
	"strings"
	"time"

	"github.com/zlylong/darwin-ops-mcp/backend/internal/adapters/kubernetes"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/adapters/linux"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/adapters/prometheus"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/domain"
)

func RegisterMockTools(r *Registry, k8s *kubernetes.MockAdapter, prom *prometheus.MockAdapter, linuxTools *linux.MockAdapter) error {
	registrations := []struct {
		tool    domain.Tool
		handler Handler
	}{
		{domain.Tool{Name: "k8s.list_pods", Description: "List Kubernetes pods", Category: "kubernetes", ReadOnly: true, Risk: domain.RiskLow, InputSchema: map[string]string{"namespace": "string?"}}, k8s.ListPods},
		{domain.Tool{Name: "k8s.get_pod_logs", Description: "Fetch pod logs", Category: "kubernetes", ReadOnly: true, Risk: domain.RiskMedium, InputSchema: map[string]string{"namespace": "string?", "pod": "string"}}, k8s.GetPodLogs},
		{domain.Tool{Name: "k8s.list_events", Description: "List Kubernetes events", Category: "kubernetes", ReadOnly: true, Risk: domain.RiskLow, InputSchema: map[string]string{"namespace": "string?"}}, k8s.ListEvents},
		{domain.Tool{Name: "k8s.get_deployment_status", Description: "Get deployment rollout status", Category: "kubernetes", ReadOnly: true, Risk: domain.RiskLow, InputSchema: map[string]string{"namespace": "string?", "deployment": "string"}}, k8s.GetDeploymentStatus},
		{domain.Tool{Name: "prometheus.query", Description: "Run a read-only Prometheus query", Category: "prometheus", ReadOnly: true, Risk: domain.RiskMedium, InputSchema: map[string]string{"query": "string"}}, prom.Query},
		{domain.Tool{Name: "prometheus.service_error_rate", Description: "Get service error rate", Category: "prometheus", ReadOnly: true, Risk: domain.RiskLow, InputSchema: map[string]string{"service": "string"}}, prom.ServiceErrorRate},
		{domain.Tool{Name: "prometheus.service_latency_p95", Description: "Get service p95 latency", Category: "prometheus", ReadOnly: true, Risk: domain.RiskLow, InputSchema: map[string]string{"service": "string"}}, prom.ServiceLatencyP95},
		{domain.Tool{Name: "prometheus.pod_cpu_usage", Description: "Get pod CPU usage", Category: "prometheus", ReadOnly: true, Risk: domain.RiskLow, InputSchema: map[string]string{"pod": "string"}}, prom.PodCPUUsage},
		{domain.Tool{Name: "prometheus.pod_memory_usage", Description: "Get pod memory usage", Category: "prometheus", ReadOnly: true, Risk: domain.RiskLow, InputSchema: map[string]string{"pod": "string"}}, prom.PodMemoryUsage},
		{domain.Tool{Name: "linux.system_info", Description: "Show Linux host, kernel, distro, uptime and virtualization info", Category: "linux", ReadOnly: true, Risk: domain.RiskLow, InputSchema: map[string]string{}}, linuxTools.SystemInfo},
		{domain.Tool{Name: "linux.load_average", Description: "Show Linux load average and CPU core count", Category: "linux", ReadOnly: true, Risk: domain.RiskLow, InputSchema: map[string]string{}}, linuxTools.LoadAverage},
		{domain.Tool{Name: "linux.memory_usage", Description: "Show Linux memory and swap usage", Category: "linux", ReadOnly: true, Risk: domain.RiskLow, InputSchema: map[string]string{}}, linuxTools.MemoryUsage},
		{domain.Tool{Name: "linux.disk_usage", Description: "Show filesystem disk usage for a path", Category: "linux", ReadOnly: true, Risk: domain.RiskLow, InputSchema: map[string]string{"path": "string?"}}, linuxTools.DiskUsage},
		{domain.Tool{Name: "linux.process_list", Description: "List top Linux processes by resource usage", Category: "linux", ReadOnly: true, Risk: domain.RiskLow, InputSchema: map[string]string{"limit": "number?"}}, linuxTools.ProcessList},
		{domain.Tool{Name: "linux.network_interfaces", Description: "Show Linux network interface addresses and traffic counters", Category: "linux", ReadOnly: true, Risk: domain.RiskLow, InputSchema: map[string]string{}}, linuxTools.NetworkInterfaces},
		{domain.Tool{Name: "linux.service_status", Description: "Check systemd service status", Category: "linux", ReadOnly: true, Risk: domain.RiskLow, InputSchema: map[string]string{"service": "string"}}, linuxTools.ServiceStatus},
		{domain.Tool{Name: "linux.journal_tail", Description: "Tail recent journal logs for a systemd unit", Category: "linux", ReadOnly: true, Risk: domain.RiskMedium, RequiresApproval: true, InputSchema: map[string]string{"unit": "string", "lines": "number?"}}, linuxTools.JournalTail},
		{domain.Tool{Name: "linux.ping", Description: "Run a ping connectivity check", Category: "linux", ReadOnly: true, Risk: domain.RiskLow, InputSchema: map[string]string{"host": "string", "count": "number?"}}, linuxTools.Ping},
		{domain.Tool{Name: "linux.dns_lookup", Description: "Resolve DNS records for a hostname", Category: "linux", ReadOnly: true, Risk: domain.RiskLow, InputSchema: map[string]string{"host": "string"}}, linuxTools.DNSLookup},
	}
	for _, reg := range registrations {
		if err := r.Register(reg.tool, reg.handler); err != nil {
			return err
		}
	}
	return nil
}

// seedID is the fixed ID used for seeded executions and audit records.
// Using a fixed ID makes SeedMockData idempotent: re-calling it will
// find the existing records and skip insertion rather than duplicating.
const seedIDPrefix = "seed-"

func (r *Registry) SeedMockData() {
	// Idempotency: skip if any seeded execution already exists.
	for _, exe := range r.executions.List() {
		if strings.HasPrefix(exe.ID, seedIDPrefix) {
			return
		}
	}

	now := time.Now().UTC()
	seeded := []domain.Execution{
		{
			ID:        "seed-exe-list-pods",
			Tool:      "k8s.list_pods",
			Actor:     "mock.user",
			Role:      domain.RoleViewer,
			Target:    "cluster=demo namespace=default",
			Status:    "succeeded",
			Reason:    "seeded mock execution",
			AuditID:   "seed-aud-list-pods",
			CreatedAt: now.Add(-35 * time.Minute),
			Parameters: map[string]any{
				"namespace": "default",
			},
			Result: map[string]any{
				"pods": []map[string]any{
					{"name": "api-7df6c9d5b6-vlz8p", "namespace": "default", "status": "Running", "restarts": 0},
					{"name": "worker-778fd9c889-c8nwp", "namespace": "default", "status": "Running", "restarts": 1},
				},
			},
		},
		{
			ID:        "seed-exe-error-rate",
			Tool:      "prometheus.service_error_rate",
			Actor:     "mock.operator",
			Role:      domain.RoleOperator,
			Target:    "service=api",
			Status:    "succeeded",
			Reason:    "seeded mock execution",
			AuditID:   "seed-aud-error-rate",
			CreatedAt: now.Add(-20 * time.Minute),
			Parameters: map[string]any{
				"service": "api",
			},
			Result: map[string]any{
				"service":   "api",
				"errorRate": 0.012,
				"unit":      "ratio",
			},
		},
		{
			ID:        "seed-exe-validation",
			Tool:      "k8s.get_pod_logs",
			Actor:     "mock.user",
			Role:      domain.RoleViewer,
			Target:    "cluster=demo namespace=default",
			Status:    "validation_failed",
			Reason:    "missing required parameter: pod",
			AuditID:   "seed-aud-validation",
			CreatedAt: now.Add(-5 * time.Minute),
			Parameters: map[string]any{
				"namespace": "default",
			},
		},
	}
	for _, exe := range seeded {
		r.executions.Add(exe)
	}
	for _, record := range []domain.AuditRecord{
		{ID: "seed-aud-list-pods", ExecutionID: "seed-exe-list-pods", At: now.Add(-35 * time.Minute), Actor: "mock.user", Role: domain.RoleViewer, Action: "k8s.list_pods", Target: "cluster=demo namespace=default", Allowed: true, Reason: "seeded mock execution", Parameters: map[string]any{"namespace": "default"}},
		{ID: "seed-aud-error-rate", ExecutionID: "seed-exe-error-rate", At: now.Add(-20 * time.Minute), Actor: "mock.operator", Role: domain.RoleOperator, Action: "prometheus.service_error_rate", Target: "service=api", Allowed: true, Reason: "seeded mock execution", Parameters: map[string]any{"service": "api"}},
		{ID: "seed-aud-validation", ExecutionID: "seed-exe-validation", At: now.Add(-5 * time.Minute), Actor: "mock.user", Role: domain.RoleViewer, Action: "k8s.get_pod_logs", Target: "cluster=demo namespace=default", Allowed: false, Reason: "missing required parameter: pod", Parameters: map[string]any{"namespace": "default"}},
	} {
		r.auditor.Record(record)
	}
}
