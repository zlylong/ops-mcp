package app

import (
	"github.com/zlylong/ops-mcp/backend/internal/adapters/kubernetes"
	"github.com/zlylong/ops-mcp/backend/internal/adapters/prometheus"
	"github.com/zlylong/ops-mcp/backend/internal/domain"
)

func RegisterMockTools(r *Registry, k8s *kubernetes.MockAdapter, prom *prometheus.MockAdapter) error {
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
	}
	for _, reg := range registrations {
		if err := r.Register(reg.tool, reg.handler); err != nil {
			return err
		}
	}
	return nil
}
