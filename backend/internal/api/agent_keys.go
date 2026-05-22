package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/app"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/domain"
)

// createAgentAPIKey issues a new API key for an agent. The plaintext secret is returned only once.
//
// @Summary Create Agent API Key
// @Description Issues a bearer token for an AI agent. Requires the master API token when authentication is enabled.
// @Tags agent-keys
// @Accept json
// @Produce json
// @Param request body map[string]any true "Agent API key issuance request"
// @Success 201 {object} object
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /api/v1/agent-keys [post]
func (s *Server) createAgentAPIKey(c *gin.Context) {
	if !requireMasterCredential(c) {
		return
	}
	var req domain.AgentAPIKeyCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return
	}
	created, err := s.registry.CreateAgentAPIKey(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, created)
}

// listAgentAPIKeys lists issued agent API key metadata without secrets.
//
// @Summary List Agent API Keys
// @Description Lists API key metadata. Plaintext secrets and hashes are never returned.
// @Tags agent-keys
// @Produce json
// @Success 200 {array} object
// @Failure 403 {object} map[string]string
// @Router /api/v1/agent-keys [get]
func (s *Server) listAgentAPIKeys(c *gin.Context) {
	if !requireMasterCredential(c) {
		return
	}
	c.JSON(http.StatusOK, s.registry.AgentAPIKeys())
}

// revokeAgentAPIKey revokes an issued agent API key.
//
// @Summary Revoke Agent API Key
// @Description Revokes an agent API key by ID. Revoked keys can no longer authenticate.
// @Tags agent-keys
// @Produce json
// @Param id path string true "Agent API key ID"
// @Success 200 {object} object
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/agent-keys/{id}/revoke [post]
func (s *Server) revokeAgentAPIKey(c *gin.Context) {
	if !requireMasterCredential(c) {
		return
	}
	key, err := s.registry.RevokeAgentAPIKey(c.Param("id"))
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, app.ErrAgentAPIKeyNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, key)
}
