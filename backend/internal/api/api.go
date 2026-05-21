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
// @BasePath /api/v1
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
// @Router /tools [post]
func (s *Server) createTool(c *gin.Context) {
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
// @Router /tools/{name} [put]
func (s *Server) updateTool(c *gin.Context) {
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
// @Router /tools/{name} [delete]
func (s *Server) deleteTool(c *gin.Context) {
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

// executeTool executes a tool with the provided parameters
//
// @Summary Execute Tool
// @Description Executes a tool with the provided parameters. Returns 202 if approval
//     is required; returns 200 on immediate execution.
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
// @Router /tools/{name}/execute [post]
func (s *Server) executeTool(c *gin.Context) {
	var req executeHTTP
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return
	}
	result, status, err := s.registry.Execute(c.Request.Context(), c.Param("name"), domainRequest(req))
	if err != nil {
		c.JSON(status, gin.H{"error": result.Message, "executionId": result.ExecutionID, "auditId": result.AuditID, "approvalId": result.ApprovalID})
		return
	}
	c.JSON(status, result)
}
