package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/zlylong/darwin-ops-mcp/backend/internal/config"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/domain"
)

func resetAuthRateLimiter() {
	authRateLimiter.Range(func(key, _ any) bool {
		authRateLimiter.Delete(key)
		return true
	})
}

func TestIsRateLimited_NoRecord(t *testing.T) {
	resetAuthRateLimiter()
	limited, retryAfter := IsRateLimited("203.0.113.1")
	assert.False(t, limited)
	assert.Equal(t, 0, retryAfter)
}

func TestIsRateLimited_UnderThreshold(t *testing.T) {
	resetAuthRateLimiter()
	for i := 0; i < 3; i++ {
		RecordFailedAuth("198.51.100.1")
	}
	limited, _ := IsRateLimited("198.51.100.1")
	assert.False(t, limited)
}

func TestIsRateLimited_AtThreshold(t *testing.T) {
	resetAuthRateLimiter()
	for i := 0; i < 8; i++ {
		RecordFailedAuth("198.51.100.2")
	}
	limited, retryAfter := IsRateLimited("198.51.100.2")
	assert.True(t, limited)
	assert.Greater(t, retryAfter, 0)
}

func TestIsRateLimited_ExcessAttempts(t *testing.T) {
	resetAuthRateLimiter()
	for i := 0; i < 12; i++ {
		RecordFailedAuth("198.51.100.3")
	}
	limited, retryAfter := IsRateLimited("198.51.100.3")
	assert.True(t, limited)
	assert.Greater(t, retryAfter, 0)
}

func TestIsRateLimited_StaleAttemptsExpire(t *testing.T) {
	resetAuthRateLimiter()
	entry, _ := authRateLimiter.LoadOrStore("203.0.113.99", &rateLimitEntry{})
	e := entry.(*rateLimitEntry)
	e.mu.Lock()
	for i := 0; i < 8; i++ {
		e.attempts = append(e.attempts, time.Now().Add(-10*time.Minute))
	}
	e.mu.Unlock()
	limited, _ := IsRateLimited("203.0.113.99")
	assert.False(t, limited)
}

func TestClearFailedAuth_RemovesRecord(t *testing.T) {
	resetAuthRateLimiter()
	for i := 0; i < 5; i++ {
		RecordFailedAuth("203.0.113.50")
	}
	ClearFailedAuth("203.0.113.50")
	limited, _ := IsRateLimited("203.0.113.50")
	assert.False(t, limited)
}

func TestRecordFailedAuth_AppendsNotReplaces(t *testing.T) {
	resetAuthRateLimiter()
	for i := 0; i < 3; i++ {
		RecordFailedAuth("203.0.113.60")
	}
	for i := 0; i < 5; i++ {
		RecordFailedAuth("203.0.113.60")
	}
	limited, _ := IsRateLimited("203.0.113.60")
	assert.True(t, limited)
}

func TestLogin_TooManyRequests(t *testing.T) {
	resetAuthRateLimiter()
	for i := 0; i < 8; i++ {
		RecordFailedAuth("203.0.113.10")
	}
	cfg := config.Config{}
	r := NewRouter(cfg, createTestRegistry(t), &mockRecorder{}, slog.Default())
	body := `{"username":"alice","test-password-123": "test-password-123"}`
	_ = body
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/login", nil)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.10:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
}

func TestLogin_ClearFailedAuthAfterSuccess(t *testing.T) {
	resetAuthRateLimiter()
	RecordFailedAuth("203.0.113.20")

	reg := createTestRegistry(t)
	plaintext := "test-password-123"
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	require.NoError(t, err)
	reg.Users().Add(domain.User{Username: "bob", Nickname: "Bob", Role: domain.RoleViewer, Status: "active"}, plaintext, hash)

	cfg := config.Config{}
	r := NewRouter(cfg, reg, &mockRecorder{}, slog.Default())

	loginBody, _ := json.Marshal(map[string]string{"username": "bob", "password": plaintext})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/users/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginReq.RemoteAddr = "203.0.113.20:1234"
	loginW := httptest.NewRecorder()
	r.ServeHTTP(loginW, loginReq)
	require.Equal(t, http.StatusOK, loginW.Code, loginW.Body.String())

	limited, _ := IsRateLimited("203.0.113.20")
	assert.False(t, limited)
}
