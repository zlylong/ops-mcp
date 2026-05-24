package api

import (
	"crypto/subtle"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/app"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/domain"
)

const (
	authIsMasterKey = "auth.isMaster"
	authAgentKey    = "auth.agentKey"
)

// rateLimitEntry tracks failed auth attempt timestamps for an IP.
type rateLimitEntry struct {
	mu       sync.Mutex
	attempts []time.Time
}

// authRateLimiter implements per-IP sliding-window brute-force protection for
// all authentication endpoints (master token, agent API key, user token).
var authRateLimiter = struct {
	sync.Map
	limit  int
	window time.Duration
}{
	limit:  8,
	window: 5 * time.Minute,
}

func init() {
	// Background cleanup of inactive IP entries.
	go func() {
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			authRateLimiter.Range(func(key, value any) bool {
				entry := value.(*rateLimitEntry)
				entry.mu.Lock()
				cutoff := now.Add(-authRateLimiter.window)
				i := 0
				for ; i < len(entry.attempts) && entry.attempts[i].Before(cutoff); i++ {
				}
				if i > 0 {
					entry.attempts = entry.attempts[i:]
				}
				if len(entry.attempts) == 0 {
					authRateLimiter.Delete(key)
				}
				entry.mu.Unlock()
				return true
			})
		}
	}()
}

// RecordFailedAuth records a failed authentication attempt from the given IP.
// Call this on every failed auth (invalid token, unknown key, etc.) AFTER the
// rate-limit check passes so that the first few attempts are never delayed.
func RecordFailedAuth(ip string) {
	entry, _ := authRateLimiter.LoadOrStore(ip, &rateLimitEntry{})
	e := entry.(*rateLimitEntry)
	e.mu.Lock()
	e.attempts = append(e.attempts, time.Now())
	// Trim to window size
	cutoff := time.Now().Add(-authRateLimiter.window)
	i := 0
	for ; i < len(e.attempts) && e.attempts[i].Before(cutoff); i++ {
	}
	if i > 0 {
		e.attempts = e.attempts[i:]
	}
	e.mu.Unlock()
}

// IsRateLimited returns true if the IP has exceeded the failed-auth threshold
// within the sliding window. Returns the number of seconds to retry after.
func IsRateLimited(ip string) (bool, int) {
	entry, ok := authRateLimiter.Load(ip)
	if !ok {
		return false, 0
	}
	e := entry.(*rateLimitEntry)
	e.mu.Lock()
	defer e.mu.Unlock()
	cutoff := time.Now().Add(-authRateLimiter.window)
	i := 0
	for ; i < len(e.attempts) && e.attempts[i].Before(cutoff); i++ {
	}
	if len(e.attempts)-i >= authRateLimiter.limit {
		oldest := e.attempts[i]
		retryAfter := int(time.Until(oldest.Add(authRateLimiter.window)).Seconds()) + 1
		if retryAfter < 1 {
			retryAfter = 1
		}
		return true, retryAfter
	}
	return false, 0
}

func authRequired(masterToken string, registry *app.Registry) gin.HandlerFunc {
	masterToken = strings.TrimSpace(masterToken)
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if limited, retryAfter := IsRateLimited(ip); limited {
			c.Header("Retry-After", string(rune('0'+retryAfter/10))+string(rune('0'+retryAfter%10)))
			c.AbortWithStatusJSON(http.StatusTooManyRequests,
				gin.H{"error": "too many requests, please retry later", "retry_after_seconds": retryAfter})
			return
		}

		if masterToken == "" {
			c.Set(authIsMasterKey, true)
			c.Next()
			return
		}
		auth := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(auth, prefix) {
			RecordFailedAuth(ip)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		provided := strings.TrimSpace(strings.TrimPrefix(auth, prefix))
		if subtle.ConstantTimeCompare([]byte(provided), []byte(masterToken)) == 1 {
			c.Set(authIsMasterKey, true)
			c.Next()
			return
		}
		// Try agent API key auth
		if registry != nil {
			if key, ok := registry.AuthenticateAgentAPIKey(provided); ok {
				c.Set(authIsMasterKey, false)
				c.Set(authAgentKey, key)
				c.Next()
				return
			}
			// Try user token auth (format: "user:<userID>")
			if strings.HasPrefix(provided, "user:") {
				userID := strings.TrimPrefix(provided, "user:")
				user, found := registry.Users().Get(userID)
				if found && user.Status == "active" {
					c.Set(authIsMasterKey, false)
					c.Set(authUserID, userID)
					c.Next()
					return
				}
			}
		}
		RecordFailedAuth(ip)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid bearer token"})
	}
}

func requireMasterCredential(c *gin.Context) bool {
	if v, ok := c.Get(authIsMasterKey); ok {
		if isMaster, ok := v.(bool); ok && isMaster {
			return true
		}
	}
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "master api token required"})
	return false
}

func authenticatedAgent(c *gin.Context) (domain.AgentAPIKey, bool) {
	v, ok := c.Get(authAgentKey)
	if !ok {
		return domain.AgentAPIKey{}, false
	}
	key, ok := v.(domain.AgentAPIKey)
	return key, ok
}

func authenticatedActor(c *gin.Context) string {
	if key, ok := authenticatedAgent(c); ok {
		return key.Actor
	}
	return ""
}
