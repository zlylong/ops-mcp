package prometheus

import "context"

type MockAdapter struct{}

func NewMockAdapter() *MockAdapter { return &MockAdapter{} }

func (a *MockAdapter) Query(ctx context.Context, params map[string]any) (map[string]any, error) {
	query := "up"
	if v, ok := params["query"].(string); ok && v != "" {
		query = v
	}
	return map[string]any{"query": query, "resultType": "vector", "result": []map[string]any{{"metric": map[string]string{"job": "mock"}, "value": []any{1234567890, "1"}}}}, nil
}
func (a *MockAdapter) ServiceErrorRate(ctx context.Context, params map[string]any) (map[string]any, error) {
	return map[string]any{"service": service(params), "errorRate": 0.012, "window": "5m"}, nil
}
func (a *MockAdapter) ServiceLatencyP95(ctx context.Context, params map[string]any) (map[string]any, error) {
	return map[string]any{"service": service(params), "latencyMsP95": 187.4, "window": "5m"}, nil
}
func (a *MockAdapter) PodCPUUsage(ctx context.Context, params map[string]any) (map[string]any, error) {
	return map[string]any{"pod": pod(params), "cpuCores": 0.18}, nil
}
func (a *MockAdapter) PodMemoryUsage(ctx context.Context, params map[string]any) (map[string]any, error) {
	return map[string]any{"pod": pod(params), "memoryMiB": 246.5}, nil
}
func service(params map[string]any) string {
	if v, ok := params["service"].(string); ok && v != "" {
		return v
	}
	return "api"
}
func pod(params map[string]any) string {
	if v, ok := params["pod"].(string); ok && v != "" {
		return v
	}
	return "api-7df8c9"
}
