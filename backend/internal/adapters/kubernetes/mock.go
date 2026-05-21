package kubernetes

import "context"

type MockAdapter struct{}

func NewMockAdapter() *MockAdapter { return &MockAdapter{} }

func (a *MockAdapter) ListPods(ctx context.Context, params map[string]any) (map[string]any, error) {
	limit := intParam(params, "limit", 0)
	namespace := stringParam(params, "namespace", "default")
	pods := []map[string]any{
		{"name": "api-7df8c9", "namespace": namespace, "status": "Running", "restarts": 0},
		{"name": "worker-55f9d", "namespace": "default", "status": "Running", "restarts": 1},
	}
	if limit > 0 && limit < len(pods) {
		pods = pods[:limit]
	}
	return map[string]any{"pods": pods, "limit": limit}, nil
}

func (a *MockAdapter) GetPodLogs(ctx context.Context, params map[string]any) (map[string]any, error) {
	pod := stringParam(params, "pod", "api-7df8c9")
	lines := intParam(params, "lines", 100)
	return map[string]any{"pod": pod, "lines": []string{"mock log line 1", "mock log line 2", "request completed status=200"}, "requestedLines": lines}, nil
}

func (a *MockAdapter) ListEvents(ctx context.Context, params map[string]any) (map[string]any, error) {
	namespace := stringParam(params, "namespace", "default")
	limit := intParam(params, "limit", 50)
	events := []map[string]any{
		{"type": "Normal", "reason": "Scheduled", "message": "Successfully assigned pod"},
		{"type": "Warning", "reason": "BackOff", "message": "Mock warning for demo"},
	}
	if limit > 0 && limit < len(events) {
		events = events[:limit]
	}
	return map[string]any{"events": events, "namespace": namespace, "limit": limit}, nil
}

func (a *MockAdapter) GetDeploymentStatus(ctx context.Context, params map[string]any) (map[string]any, error) {
	deployment := stringParam(params, "deployment", "api")
	namespace := stringParam(params, "namespace", "default")
	replicas := intParam(params, "replicas", 3)
	return map[string]any{"deployment": deployment, "namespace": namespace, "readyReplicas": replicas, "desiredReplicas": replicas, "available": true}, nil
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
