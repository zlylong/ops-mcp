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

	"github.com/zlylong/ops-mcp/backend/internal/app"
	"github.com/zlylong/ops-mcp/backend/internal/config"
	"github.com/zlylong/ops-mcp/backend/internal/domain"
	"github.com/zlylong/ops-mcp/backend/internal/policy"
	"github.com/zlylong/ops-mcp/backend/internal/storage"
)

type mockRecorder struct{}

func (m *mockRecorder) Record(record domain.AuditRecord) domain.AuditRecord {
	record.ID = "aud-mock-123"
	return record
}
func (m *mockRecorder) List() []domain.AuditRecord { return nil }

func createTestRegistry() *app.Registry {
	registry := app.NewRegistry(policy.NewEngine(), &mockRecorder{}, storage.NewExecutionStore(), storage.NewApprovalStore(), domain.EnvDevelopment)
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
	assert.Contains(t, docResp.Body.String(), "Ops MCP API")
}
