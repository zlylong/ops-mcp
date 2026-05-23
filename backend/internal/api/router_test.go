package api

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/zlylong/darwin-ops-mcp/backend/internal/app"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/config"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/domain"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/policy"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/storage"
)

type mockRecorder struct{}

func (m *mockRecorder) Record(record domain.AuditRecord) domain.AuditRecord {
	record.ID = "aud-mock-123"
	return record
}
func (m *mockRecorder) List() []domain.AuditRecord { return nil }

func createTestRegistry() *app.Registry {
	registry := app.NewRegistry(policy.NewEngine(), &mockRecorder{}, storage.NewExecutionStore(), storage.NewApprovalStore(), storage.NewUserStore(), domain.EnvDevelopment)
	// 使用 ReadOnly 工具，Low 风险
	registry.Register(domain.Tool{Name: "test.tool", ReadOnly: true, Risk: domain.RiskLow}, func(ctx context.Context, params map[string]any) (map[string]any, error) {
		return map[string]any{"result": "ok"}, nil
	})
	return registry
}

func TestNewRouter_CORS(t *testing.T) {
	cfg := config.Config{}
	registry := createTestRegistry()
	auditor := &mockRecorder{}
	logger := slog.Default()
	r := NewRouter(cfg, registry, auditor, logger)
	assert.NotNil(t, r)
}

func TestHealthz(t *testing.T) {
	cfg := config.Config{}
	registry := createTestRegistry()
	auditor := &mockRecorder{}
	logger := slog.Default()
	r := NewRouter(cfg, registry, auditor, logger)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestDashboardSummary(t *testing.T) {
	cfg := config.Config{}
	registry := createTestRegistry()
	auditor := &mockRecorder{}
	logger := slog.Default()
	r := NewRouter(cfg, registry, auditor, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestToolsList(t *testing.T) {
	cfg := config.Config{}
	registry := createTestRegistry()
	auditor := &mockRecorder{}
	logger := slog.Default()
	r := NewRouter(cfg, registry, auditor, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tools", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestToolDetail(t *testing.T) {
	cfg := config.Config{}
	registry := createTestRegistry()
	auditor := &mockRecorder{}
	logger := slog.Default()
	r := NewRouter(cfg, registry, auditor, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tools/test.tool", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestToolDetail_NotFound(t *testing.T) {
	cfg := config.Config{}
	registry := createTestRegistry()
	auditor := &mockRecorder{}
	logger := slog.Default()
	r := NewRouter(cfg, registry, auditor, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tools/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestExecuteTool(t *testing.T) {
	cfg := config.Config{}
	registry := createTestRegistry()
	auditor := &mockRecorder{}
	logger := slog.Default()
	r := NewRouter(cfg, registry, auditor, logger)

	reqBody := map[string]any{
		"actor":      "test-user",
		"role":       "viewer",
		"target":     "local-dev",
		"parameters": map[string]any{},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/test.tool/execute", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 200 因为 tool 无 InputSchema，不校验参数
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestExecuteTool_ValidationFailure(t *testing.T) {
	cfg := config.Config{}
	registry := createTestRegistry()
	auditor := &mockRecorder{}
	logger := slog.Default()
	r := NewRouter(cfg, registry, auditor, logger)

	// 使用不存在的 tool，触发 404 Not Found
	reqBody := map[string]any{
		"actor":      "test-user",
		"role":       "viewer",
		"target":     "local-dev",
		"parameters": map[string]any{},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/nonexistent/execute", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 404 因为 tool 不存在
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestApprovalsList(t *testing.T) {
	cfg := config.Config{}
	registry := createTestRegistry()
	auditor := &mockRecorder{}
	logger := slog.Default()
	r := NewRouter(cfg, registry, auditor, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/approvals", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestApproveApproval(t *testing.T) {
	cfg := config.Config{}
	registry := createTestRegistry()
	auditor := &mockRecorder{}
	logger := slog.Default()
	r := NewRouter(cfg, registry, auditor, logger)

	now := time.Now()
	approval := domain.Approval{
		ID:          "app-1",
		ExecutionID: "exec-123",
		Tool:        "test.tool",
		Actor:       "test-user",
		Target:      "local-dev",
		Status:      domain.ApprovalPending,
		Reason:      "pending",
		CreatedAt:   now,
	}
	registry.AddApproval(approval)

	body := map[string]string{"action": "approve"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/approvals/app-1/approve", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRejectApproval(t *testing.T) {
	cfg := config.Config{}
	registry := createTestRegistry()
	auditor := &mockRecorder{}
	logger := slog.Default()
	r := NewRouter(cfg, registry, auditor, logger)

	now := time.Now()
	approval := domain.Approval{
		ID:          "app-2",
		ExecutionID: "exec-123",
		Tool:        "test.tool",
		Actor:       "test-user",
		Target:      "local-dev",
		Status:      domain.ApprovalPending,
		Reason:      "pending",
		CreatedAt:   now,
	}
	registry.AddApproval(approval)

	body := map[string]string{"action": "reject"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/approvals/app-2/reject", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditRecords(t *testing.T) {
	cfg := config.Config{}
	registry := createTestRegistry()
	auditor := &mockRecorder{}
	logger := slog.Default()
	r := NewRouter(cfg, registry, auditor, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSwaggerUIRoutes(t *testing.T) {
	cfg := config.Config{}
	registry := createTestRegistry()
	auditor := &mockRecorder{}
	logger := slog.Default()
	r := NewRouter(cfg, registry, auditor, logger)

	redirectReq := httptest.NewRequest(http.MethodGet, "/swagger", nil)
	redirectResp := httptest.NewRecorder()
	r.ServeHTTP(redirectResp, redirectReq)
	assert.Equal(t, http.StatusMovedPermanently, redirectResp.Code)
	assert.Equal(t, "/swagger/index.html", redirectResp.Header().Get("Location"))

	indexReq := httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	indexResp := httptest.NewRecorder()
	r.ServeHTTP(indexResp, indexReq)
	assert.Equal(t, http.StatusOK, indexResp.Code)
	assert.Contains(t, indexResp.Body.String(), "Swagger UI")

	docReq := httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil)
	docResp := httptest.NewRecorder()
	r.ServeHTTP(docResp, docReq)
	assert.Equal(t, http.StatusOK, docResp.Code)
	assert.Contains(t, docResp.Body.String(), "Darwin Ops MCP API")
}

func TestToolCRUDRoutes(t *testing.T) {
	cfg := config.Config{}
	registry := createTestRegistry()
	auditor := &mockRecorder{}
	logger := slog.Default()
	r := NewRouter(cfg, registry, auditor, logger)

	createBody := map[string]any{"name": "custom.echo", "description": "Echo params", "category": "custom", "readOnly": true, "risk": "low", "requiresApproval": false, "inputSchema": map[string]domain.ParamSchema{"message": {Type: "string", Required: false}}}
	body, _ := json.Marshal(createBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "custom.echo")

	updateBody := map[string]any{"name": "custom.echo", "description": "Updated", "category": "custom", "readOnly": true, "risk": "medium", "requiresApproval": true, "inputSchema": map[string]domain.ParamSchema{"message": {Type: "string", Required: false}}}
	body, _ = json.Marshal(updateBody)
	req = httptest.NewRequest(http.MethodPut, "/api/v1/tools/custom.echo", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Updated")

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/tools/custom.echo", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/tools/custom.echo", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestToolCRUDRoutesValidation(t *testing.T) {
	cfg := config.Config{}
	registry := createTestRegistry()
	auditor := &mockRecorder{}
	logger := slog.Default()
	r := NewRouter(cfg, registry, auditor, logger)

	body, _ := json.Marshal(map[string]any{"name": "bad/tool", "risk": "low"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	body, _ = json.Marshal(map[string]any{"name": "missing.tool", "risk": "low"})
	req = httptest.NewRequest(http.MethodPut, "/api/v1/tools/missing.tool", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/tools/missing.tool", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestExecuteTool_RequiresApprovalFlagCreatesApproval(t *testing.T) {
	cfg := config.Config{}
	registry := createTestRegistry()
	auditor := &mockRecorder{}
	logger := slog.Default()
	r := NewRouter(cfg, registry, auditor, logger)

	createBody := map[string]any{"name": "approval.flag", "description": "Approval flag", "category": "custom", "readOnly": true, "risk": "low", "requiresApproval": true, "inputSchema": map[string]domain.ParamSchema{"message": {Type: "string", Required: false}}}
	body, _ := json.Marshal(createBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tools", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	executeBody := map[string]any{"actor": "viewer", "role": "viewer", "target": "local", "parameters": map[string]any{"message": "hello"}}
	body, _ = json.Marshal(executeBody)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/tools/approval.flag/execute", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "pending_approval")

	req = httptest.NewRequest(http.MethodGet, "/api/v1/approvals", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "approval.flag")
}

func TestApplicationDecisionRoutes(t *testing.T) {
	cfg := config.Config{}
	registry := createTestRegistry()
	auditor := &mockRecorder{}
	logger := slog.Default()
	r := NewRouter(cfg, registry, auditor, logger)

	body, _ := json.Marshal(map[string]any{
		"tool":        "test.tool",
		"risk":        "high",
		"role":        "operator",
		"reason":      "temporary access",
		"durationHrs": 8,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/applications", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var application domain.ToolApplication
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &application))
	assert.Equal(t, domain.ApplicationPending, application.Status)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/applications/"+application.ID+"/approve", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), string(domain.ApplicationApproved))

	body, _ = json.Marshal(map[string]any{
		"tool":        "test.tool",
		"risk":        "critical",
		"role":        "admin",
		"reason":      "break glass",
		"durationHrs": 1,
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/applications", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &application))

	req = httptest.NewRequest(http.MethodPost, "/api/v1/applications/"+application.ID+"/reject", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), string(domain.ApplicationRejected))

	req = httptest.NewRequest(http.MethodPost, "/api/v1/applications/missing/reject", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
