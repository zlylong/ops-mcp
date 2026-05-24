// Darwin Ops MCP API
//
// @title Darwin Ops MCP API
// @version 1.0
// @description Darwin Ops MCP Backend API with Tool Registry, Policy Engine, and Approval Flow.
// @description Returns immediate execution result or 202 + approval ID for high-risk tools.
// @contact.name API Support
// @contact.url http://localhost:8080
// @contact.email support@darwin-ops-mcp.local
// @license.name MIT
// @license.url https://opensource.org/licenses/MIT
// @host localhost:8080
// @BasePath /
package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/app"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/domain"
)

// createTool creates a custom tool definition in the runtime registry
//
// @Summary Create Tool
// @Description Creates a custom tool definition in the runtime registry
// @Tags tools
// @Accept json
// @Produce json
// @Param request body map[string]any true "Tool definition"
// @Success 201 {object} object
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/v1/tools [post]
func (s *Server) createTool(c *gin.Context) {
	if !s.requireAdminRole(c) {
		return
	}
	var tool domain.Tool
	if err := c.ShouldBindJSON(&tool); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return
	}
	created, err := s.registry.CreateTool(tool)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, app.ErrAlreadyExists) {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, created)
}

// updateTool updates an existing tool definition
//
// @Summary Update Tool
// @Description Updates an existing tool definition in the runtime registry
// @Tags tools
// @Accept json
// @Produce json
// @Param name path string true "Tool name"
// @Param request body map[string]any true "Tool definition"
// @Success 200 {object} object
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/tools/{name} [put]
func (s *Server) updateTool(c *gin.Context) {
	if !s.requireAdminRole(c) {
		return
	}
	var tool domain.Tool
	if err := c.ShouldBindJSON(&tool); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return
	}
	updated, err := s.registry.UpdateTool(c.Param("name"), tool)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, app.ErrToolNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updated)
}

// deleteTool deletes an existing tool definition from the runtime registry
//
// @Summary Delete Tool
// @Description Deletes an existing tool definition from the runtime registry
// @Tags tools
// @Param name path string true "Tool name"
// @Success 204
// @Failure 404 {object} map[string]string
// @Router /api/v1/tools/{name} [delete]
func (s *Server) deleteTool(c *gin.Context) {
	if !s.requireAdminRole(c) {
		return
	}
	if err := s.registry.DeleteTool(c.Param("name")); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, app.ErrToolNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// executeTool executes a tool with the provided parameters.
//
// @Summary Execute Tool
// @Description Executes a tool with the provided parameters. Returns 202 if approval
// is required; returns 200 on immediate execution.
//
// @Tags tools
// @Accept json
// @Produce json
// @Param name path string true "Tool name"
// @Param request body map[string]any true "Execution request"
// @Success 200 {object} object
// @Success 202 {object} object
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/tools/{name}/execute [post]
func (s *Server) executeTool(c *gin.Context) {
	var req executeHTTP
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return
	}
	execReq := domainRequest(req)

	// SECURITY: resolve authoritative role and actor from the authenticated identity.
	// The Actor, Role, and Approved fields from the HTTP request body are ALWAYS
	// ignored to prevent privilege escalation and audit-log spoofing. Only the
	// server-side authenticated identity determines the effective execution
	// identity.
	if agentKey, ok := authenticatedAgent(c); ok {
		execReq.Role = agentKey.Role
		execReq.Actor = agentKey.Actor
	} else if uid, ok := c.Get(authUserID); ok {
		if user, found := s.registry.Users().Get(uid.(string)); found {
			execReq.Role = user.Role
			execReq.Actor = user.Username
		}
	} else {
		execReq.Actor = "master"
	}

	// SECURITY: approved is always false; the only valid approval path is
	// registry.Approve() which re-executes the tool internally.
	execReq.Approved = false

	result, status, err := s.registry.Execute(c.Request.Context(), c.Param("name"), execReq)
	if err != nil {
		c.JSON(status, gin.H{"error": result.Message, "executionId": result.ExecutionID, "auditId": result.AuditID, "approvalId": result.ApprovalID})
		return
	}
	c.JSON(status, result)
}

// submitApplication submits a tool access application for review or auto-approval
//
// @Summary Submit Tool Application
// @Description Applies for access to a tool with a specific risk level, role, and reason.
//
//	Returns 201 with the application record. High-risk (high/critical) applications
//	are set to "pending" and require admin review; low/medium are auto-approved.
//
// @Tags applications
// @Accept json
// @Produce json
// @Param request body map[string]any true "Tool application request"
// @Success 201 {object} map[string]any
// @Failure 400 {object} map[string]string
// @Router /api/v1/applications [post]
func (s *Server) submitApplication(c *gin.Context) {
	var req domain.ToolApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body: " + err.Error()})
		return
	}
	if req.Tool == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tool name is required"})
		return
	}
	if req.Role == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role is required"})
		return
	}
	if req.Reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reason is required"})
		return
	}
	switch req.Risk {
	case domain.RiskLow, domain.RiskMedium, domain.RiskHigh, domain.RiskCritical:
		// valid
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid risk level: must be low, medium, high, or critical"})
		return
	}
	actor := authenticatedActor(c)
	if actor == "" {
		if uid, ok := c.Get(authUserID); ok {
			if user, found := s.registry.Users().Get(uid.(string)); found {
				actor = user.Username
			}
		}
	}
	if actor == "" {
		actor = "master"
	}
	app := s.registry.SubmitApplication(req, actor)
	c.JSON(http.StatusCreated, app)
}

// listApplications returns all tool access applications
//
// @Summary List Applications
// @Description Returns all tool access applications in creation order (newest last).
// @Tags applications
// @Produce json
// @Success 200 {array} map[string]any
// @Router /api/v1/applications [get]
func (s *Server) listApplications(c *gin.Context) {
	if !s.requireAdminRole(c) {
		return
	}
	c.JSON(http.StatusOK, s.registry.Applications())
}

// approveApplication approves a pending tool access application
//
// @Summary Approve Tool Application
// @Description Approves a pending tool access application and records the decision timestamp.
// @Tags applications
// @Produce json
// @Param id path string true "Application ID"
// @Success 200 {object} domain.ToolApplication
// @Failure 404 {object} map[string]string
// @Router /api/v1/applications/{id}/approve [post]
func (s *Server) approveApplication(c *gin.Context) {
	if !s.requireAdminRole(c) {
		return
	}
	application, err := s.registry.ApproveApplication(c.Param("id"))
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, app.ErrApplicationNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, application)
}

// rejectApplication rejects a pending tool access application
//
// @Summary Reject Tool Application
// @Description Rejects a pending tool access application and records the decision timestamp.
// @Tags applications
// @Produce json
// @Param id path string true "Application ID"
// @Success 200 {object} domain.ToolApplication
// @Failure 404 {object} map[string]string
// @Router /api/v1/applications/{id}/reject [post]
func (s *Server) rejectApplication(c *gin.Context) {
	if !s.requireAdminRole(c) {
		return
	}
	application, err := s.registry.RejectApplication(c.Param("id"))
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, app.ErrApplicationNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, application)
}
