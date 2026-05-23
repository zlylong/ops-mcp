package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/app"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/domain"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/policy"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/storage"
)


// createTestRegistryForApp is a local alias so we don't conflict with router_test's createTestRegistry.
func createTestRegistryForApp() *app.Registry {
	return app.NewRegistry(
		policy.NewEngine(),
		&mockRecorder{},
		storage.NewExecutionStore(),
		storage.NewApprovalStore(),
		storage.NewUserStore(),
		domain.EnvDevelopment,
	)
}

// TestSubmitApplication_LowRisk verifies that low-risk applications are auto-approved.
func TestSubmitApplication_LowRisk(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reg := createTestRegistryForApp()
	srv := &Server{registry: reg, auditor: &mockRecorder{}, logger: slog.Default()}

	body := map[string]any{
		"tool":   "server_exec",
		"risk":   "low",
		"role":   "viewer",
		"reason": "need to read server logs",
	}
	b, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/applications", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	srv.submitApplication(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp domain.ToolApplication
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "server_exec", resp.Tool)
	assert.Equal(t, domain.RiskLow, resp.Risk)
	assert.Equal(t, domain.RoleViewer, resp.Role)
	assert.Equal(t, domain.ApplicationApproved, resp.Status)
	assert.Equal(t, "auto-approved (low/medium risk)", resp.Decision)
	assert.Equal(t, 24, resp.DurationHrs)
}

// TestSubmitApplication_HighRisk verifies that high-risk applications are set to pending.
func TestSubmitApplication_HighRisk(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reg := createTestRegistryForApp()
	srv := &Server{registry: reg, auditor: &mockRecorder{}, logger: slog.Default()}

	body := map[string]any{
		"tool":   "server_exec",
		"risk":   "high",
		"role":   "operator",
		"reason": "need to restart API gateway",
	}
	b, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/applications", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	srv.submitApplication(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp domain.ToolApplication
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, domain.RiskHigh, resp.Risk)
	assert.Equal(t, domain.ApplicationPending, resp.Status)
	assert.Equal(t, "pending review (high/critical risk)", resp.Decision)
}

// TestSubmitApplication_CriticalRisk verifies that critical-risk applications set status to pending.
func TestSubmitApplication_CriticalRisk(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reg := createTestRegistryForApp()
	srv := &Server{registry: reg, auditor: &mockRecorder{}, logger: slog.Default()}

	body := map[string]any{
		"tool":        "db_exec",
		"risk":        "critical",
		"role":        "admin",
		"reason":      "production database migration required",
		"durationHrs": 2,
	}
	b, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/applications", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	srv.submitApplication(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp domain.ToolApplication
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, domain.RiskCritical, resp.Risk)
	assert.Equal(t, domain.ApplicationPending, resp.Status)
	assert.Equal(t, 2, resp.DurationHrs)
}

// TestSubmitApplication_InvalidRisk verifies that invalid risk values return 400.
func TestSubmitApplication_InvalidRisk(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reg := createTestRegistryForApp()
	srv := &Server{registry: reg, auditor: &mockRecorder{}, logger: slog.Default()}

	body := map[string]any{
		"tool":   "server_exec",
		"risk":   "dangerous",
		"role":   "viewer",
		"reason": "test",
	}
	b, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/applications", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	srv.submitApplication(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp["error"], "invalid risk level")
}

// TestSubmitApplication_MissingFields verifies that missing required fields return 400.
func TestSubmitApplication_MissingFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name string
		body map[string]any
	}{
		{"missing_tool", map[string]any{"risk": "low", "role": "viewer", "reason": "test"}},
		{"missing_role", map[string]any{"tool": "server_exec", "risk": "low", "reason": "test"}},
		{"missing_reason", map[string]any{"tool": "server_exec", "risk": "low", "role": "viewer"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reg := createTestRegistryForApp()
			srv := &Server{registry: reg, auditor: &mockRecorder{}, logger: slog.Default()}
			b, err := json.Marshal(tc.body)
			require.NoError(t, err)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/applications", bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			srv.submitApplication(c)
			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

// TestSubmitApplication_InvalidJSON verifies that non-JSON bodies return 400.
func TestSubmitApplication_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reg := createTestRegistryForApp()
	srv := &Server{registry: reg, auditor: &mockRecorder{}, logger: slog.Default()}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/applications", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	srv.submitApplication(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestSubmitApplication_ActorHeader verifies that X-Actor header is respected.
func TestSubmitApplication_ActorHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reg := createTestRegistryForApp()
	srv := &Server{registry: reg, auditor: &mockRecorder{}, logger: slog.Default()}

	body := map[string]any{
		"tool":   "server_exec",
		"risk":   "low",
		"role":   "viewer",
		"reason": "test",
	}
	b, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/applications", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Actor", "agent-ai-01")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	srv.submitApplication(c)

	var resp domain.ToolApplication
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "agent-ai-01", resp.Actor)
}

// TestSubmitApplication_AnonymousActor verifies fallback to "anonymous" when X-Actor is absent.
func TestSubmitApplication_AnonymousActor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reg := createTestRegistryForApp()
	srv := &Server{registry: reg, auditor: &mockRecorder{}, logger: slog.Default()}

	body := map[string]any{
		"tool":   "server_exec",
		"risk":   "low",
		"role":   "viewer",
		"reason": "test",
	}
	b, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/applications", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	srv.submitApplication(c)

	var resp domain.ToolApplication
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "anonymous", resp.Actor)
}

// TestListApplications_OK verifies GET /applications returns all stored applications.
func TestListApplications_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reg := createTestRegistryForApp()
	reg.SubmitApplication(domain.ToolApplicationRequest{
		Tool:   "tool_a",
		Risk:   domain.RiskHigh,
		Role:   domain.RoleOperator,
		Reason: "test",
	}, "tester")
	reg.SubmitApplication(domain.ToolApplicationRequest{
		Tool:   "tool_b",
		Risk:   domain.RiskLow,
		Role:   domain.RoleViewer,
		Reason: "test2",
	}, "tester2")
	srv := &Server{registry: reg, auditor: &mockRecorder{}, logger: slog.Default()}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/applications", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	srv.listApplications(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp []domain.ToolApplication
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Len(t, resp, 2)
}

// TestListApplications_Empty verifies GET /applications returns empty array when no applications.
func TestListApplications_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reg := createTestRegistryForApp()
	srv := &Server{registry: reg, auditor: &mockRecorder{}, logger: slog.Default()}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/applications", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	srv.listApplications(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp []domain.ToolApplication
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Len(t, resp, 0)
}
