package kubernetes

import "context"

type MockAdapter struct{}

func NewMockAdapter() *MockAdapter { return &MockAdapter{} }

func (a *MockAdapter) ListPods(ctx context.Context, params map[string]any) (map[string]any, error) {
	return map[string]any{"pods": []map[string]any{{"name": "api-7df8c9", "namespace": param(params, "namespace", "default"), "status": "Running", "restarts": 0}, {"name": "worker-55f9d", "namespace": "default", "status": "Running", "restarts": 1}}}, nil
}
func (a *MockAdapter) GetPodLogs(ctx context.Context, params map[string]any) (map[string]any, error) {
	return map[string]any{"pod": param(params, "pod", "api-7df8c9"), "lines": []string{"mock log line 1", "mock log line 2", "request completed status=200"}}, nil
}
func (a *MockAdapter) ListEvents(ctx context.Context, params map[string]any) (map[string]any, error) {
	return map[string]any{"events": []map[string]any{{"type": "Normal", "reason": "Scheduled", "message": "Successfully assigned pod"}, {"type": "Warning", "reason": "BackOff", "message": "Mock warning for demo"}}}, nil
}
func (a *MockAdapter) GetDeploymentStatus(ctx context.Context, params map[string]any) (map[string]any, error) {
	return map[string]any{"deployment": param(params, "deployment", "api"), "namespace": param(params, "namespace", "default"), "readyReplicas": 3, "desiredReplicas": 3, "available": true}, nil
}
func param(params map[string]any, key, fallback string) string {
	if v, ok := params[key].(string); ok && v != "" {
		return v
	}
	return fallback
}
