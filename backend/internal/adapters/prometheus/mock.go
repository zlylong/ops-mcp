package prometheus

import "context"

type MockAdapter struct{}

func NewMockAdapter() *MockAdapter { return &MockAdapter{} }

func (a *MockAdapter) Query(ctx context.Context, params map[string]any) (map[string]any, error) {
	query := stringParam(params, "query", "up")
	time := stringParam(params, "time", "")
	result := []map[string]any{{"metric": map[string]string{"job": "mock"}, "value": []any{1234567890.0, "1"}}}
	if time != "" {
		result = []map[string]any{{"metric": map[string]string{"job": "mock"}, "value": []any{1234567890.0, "1"}}}
	}
	return map[string]any{"query": query, "time": time, "resultType": "vector", "result": result}, nil
}

func (a *MockAdapter) ServiceErrorRate(ctx context.Context, params map[string]any) (map[string]any, error) {
	service := stringParam(params, "service", "api")
	window := stringParam(params, "window", "5m")
	return map[string]any{"service": service, "errorRate": 0.012, "window": window}, nil
}

func (a *MockAdapter) ServiceLatencyP95(ctx context.Context, params map[string]any) (map[string]any, error) {
	service := stringParam(params, "service", "api")
	window := stringParam(params, "window", "5m")
	return map[string]any{"service": service, "latencyMsP95": 187.4, "window": window}, nil
}

func (a *MockAdapter) PodCPUUsage(ctx context.Context, params map[string]any) (map[string]any, error) {
	pod := stringParam(params, "pod", "api-7df8c9")
	namespace := stringParam(params, "namespace", "default")
	return map[string]any{"pod": pod, "namespace": namespace, "cpuCores": 0.18}, nil
}

func (a *MockAdapter) PodMemoryUsage(ctx context.Context, params map[string]any) (map[string]any, error) {
	pod := stringParam(params, "pod", "api-7df8c9")
	namespace := stringParam(params, "namespace", "default")
	return map[string]any{"pod": pod, "namespace": namespace, "memoryMiB": 246.5}, nil
}

func stringParam(params map[string]any, key, fallback string) string {
	if v, ok := params[key].(string); ok && v != "" {
		return v
	}
	return fallback
}

func intParam(params map[string]any, key string, fallback int) int {
	switch v := params[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	default:
		return fallback
	}
}
