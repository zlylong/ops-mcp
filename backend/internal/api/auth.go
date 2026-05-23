package api

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/app"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/domain"
)

const (
	authIsMasterKey = "auth.isMaster"
	authAgentKey    = "auth.agentKey"
)

func authRequired(masterToken string, registry *app.Registry) gin.HandlerFunc {
	masterToken = strings.TrimSpace(masterToken)
	return func(c *gin.Context) {
		if masterToken == "" {
			c.Set(authIsMasterKey, true)
			c.Next()
			return
		}
		auth := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(auth, prefix) {
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
