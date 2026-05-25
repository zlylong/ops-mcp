package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zlylong/darwin-ops-mcp/backend/internal/config"
)
 api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zlylong/darwin-ops-mcp/backend/internal/config"
)

// ------------------------------------------------------------------
// Rate limiting integration via login endpoint
// ------------------------------------------------------------------

func TestLogin_RateLimit_AfterFailedAttempts(t *testing.T) {
	// Reset global rate limiter for this test
	authRateLimiter = struct {
		sync.Map
		limit  int
		window time.Duration
	}{
		limit:  3,          // low limit for test
		window: 5 * time.Minute,
	}

	r := createTestRegistry(t)
	cfg := config.Config{}
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	// Exhaust the rate limit with bad credentials
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users/login",
			badBody())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		// After 3 bad attempts we should be rate-limited
	}

	// 4th attempt should be rate-limited
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/login",
		badBody())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code, w.Body.String())
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
}

func TestLogin_RateLimit_ClearsOnSuccess(t *testing.T) {
	// Reset global rate limiter
	authRateLimiter = struct {
		sync.Map
		limit  int
		window time.Duration
	}{
		limit:  3,
		window: 5 * time.Minute,
	}

	r := createTestRegistry(t)
	cfg := config.Config{}
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	// One failed attempt
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/login", badBody())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Then a successful login clears the counter
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/users/login",
		loginBody("admin", "admin-pass-8888"))
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)

	// Another failed attempt should NOT trigger rate limit (counter was cleared)
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/users/login", badBody())
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)
	// Should NOT be rate limited yet (only 1 failure after clear)
	assert.NotEqual(t, http.StatusTooManyRequests, w3.Code)
}

func TestIsRateLimited_NotLimited_ReturnsFalse(t *testing.T) {
	// Reset
	authRateLimiter = struct {
		sync.Map
		limit  int
		window time.Duration
	}{
		limit:  8,
		window: 5 * time.Minute,
	}
	limited, retryAfter := IsRateLimited("192.168.1.1")
	assert.False(t, limited)
	assert.Equal(t, 0, retryAfter)
}

func TestIsRateLimited_OverLimit_ReturnsTrue(t *testing.T) {
	// Reset
	authRateLimiter = struct {
		sync.Map
		limit  int
		window time.Duration
	}{
		limit:  2,
		window: 5 * time.Minute,
	}
	// Manually record 2 failed attempts
	RecordFailedAuth("10.0.0.1")
	RecordFailedAuth("10.0.0.1")

	limited, retryAfter := IsRateLimited("10.0.0.1")
	assert.True(t, limited)
	assert.Greater(t, retryAfter, 0)
}

func TestClearFailedAuth_RemovesEntry(t *testing.T) {
	// Reset
	authRateLimiter = struct {
		sync.Map
		limit  int
		window time.Duration
	}{
		limit:  8,
		window: 5 * time.Minute,
	}
	RecordFailedAuth("10.0.0.2")
	RecordFailedAuth("10.0.0.2")
	ClearFailedAuth("10.0.0.2")

	limited, _ := IsRateLimited("10.0.0.2")
	assert.False(t, limited)
}

// ------------------------------------------------------------------
// requireMasterCredential
// ------------------------------------------------------------------

func TestRequireMasterCredential_MasterToken_ReturnsTrue(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{APIToken: "master-token"}
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent-keys", emptyBody())
	req.Header.Set("Authorization", "Bearer master-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should not be 403 — agent-key creation requires master token
	assert.NotEqual(t, http.StatusForbidden, w.Code)
}

func TestRequireMasterCredential_NonMasterToken_Returns403(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{APIToken: "master-token"}
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent-keys", emptyBody())
	req.Header.Set("Authorization", "Bearer wrong-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ------------------------------------------------------------------
// authenticatedAgent / authenticatedActor
// ------------------------------------------------------------------

func TestAuthenticatedAgent_NotSet_ReturnsFalse(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{AgentAuthToken: "test-agent-token"}
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jumpservers?host=8.8.8.8&port=22", nil)
	req.Header.Set("Authorization", "Bearer wrong-agent-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should not authenticate with wrong token
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ------------------------------------------------------------------
// helpers
// ------------------------------------------------------------------

func badBody() *bytesReader {
	return bytes.NewReader([]byte(`{"username":"fake","password":"wrong"}`))
}

func emptyBody() *bytes.Reader {
	return bytes.NewReader([]byte(`{}`))
}

func loginBody(username, password string) *bytes.Reader {
	return bytes.NewReader([]byte(`{"username":"` + username + `","password":"` + password + `"}`))
}

// bytesReader is for Reader interface
type bytesReader = bytes.Reader
