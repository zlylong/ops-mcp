// @title Ops MCP API
// @version 1.0
// @description Ops MCP Backend API with Tool Registry, Policy Engine, and Approval Flow.

// @contact.name API Support
// @contact.url http://localhost:8080
// @contact.email support@ops-mcp.local

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

package api

import (
	"github.com/gin-gonic/gin"
)

// @Summary Health Check
// @Description Returns the health status of the API
// @Tags system
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /healthz [get]
func healthz(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok", "mode": "mock"})
}

// @Summary Get Dashboard Summary
// @Description Returns dashboard statistics including alerts, approvals, and executions count
// @Tags dashboard
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /dashboard/summary [get]
func dashboardSummary(c *gin.Context) {
	c.JSON(200, gin.H{"data": gin.H{}})
}

// @Summary List All Tools
// @Description Returns a list of all available tools
// @Tags tools
// @Produce json
// @Success 200 {array} map[string]interface{}
// @Router /tools [get]
func toolsList(c *gin.Context) {
	c.JSON(200, gin.H{"data": []interface{}{}})
}

// @Summary Get Tool Details
// @Description Returns detailed information about a specific tool
// @Tags tools
// @Param name path string true "Tool name"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /tools/{name} [get]
func toolDetail(c *gin.Context) {
	c.JSON(200, gin.H{"data": gin.H{}})
}

// @Summary Execute Tool
// @Description Executes a tool with the provided parameters
// @Tags tools
// @Param name path string true "Tool name"
// @Accept json
// @Produce json
// @Param request body object true "Execution request"
// @Success 200 {object} map[string]interface{} "Immediate execution result"
// @Success 202 {object} map[string]interface{} "Execution accepted, requires approval"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 403 {object} map[string]interface{} "Permission denied"
// @Failure 404 {object} map[string]interface{} "Tool not found"
// @Router /tools/{name}/execute [post]
func executeTool(c *gin.Context) {
	c.JSON(200, gin.H{"data": gin.H{}})
}

// @Summary List Approvals
// @Description Returns a list of pending approvals
// @Tags approvals
// @Produce json
// @Success 200 {array} map[string]interface{}
// @Router /approvals [get]
func approvalsList(c *gin.Context) {
	c.JSON(200, gin.H{"data": []interface{}{}})
}

// @Summary Approve Execution
// @Description Approves a pending execution request
// @Tags approvals
// @Param id path string true "Approval ID"
// @Accept json
// @Produce json
// @Param request body object true "Action request"
// @Success 200 {object} map[string]interface{}
// @Router /approvals/{id}/approve [post]
func approveApproval(c *gin.Context) {
	c.JSON(200, gin.H{"data": gin.H{}})
}

// @Summary Reject Execution
// @Description Rejects a pending execution request
// @Tags approvals
// @Param id path string true "Approval ID"
// @Accept json
// @Produce json
// @Param request body object true "Action request"
// @Success 200 {object} map[string]interface{}
// @Router /approvals/{id}/reject [post]
func rejectApproval(c *gin.Context) {
	c.JSON(200, gin.H{"data": gin.H{}})
}

// @Summary List Audit Records
// @Description Returns a list of audit records with filtering options
// @Tags audit
// @Param actor query string false "Filter by actor"
// @Param tool query string false "Filter by tool"
// @Param env query string false "Filter by environment"
// @Param risk query string false "Filter by risk level"
// @Param status query string false "Filter by status"
// @Produce json
// @Success 200 {array} map[string]interface{}
// @Router /audit [get]
func auditRecords(c *gin.Context) {
	c.JSON(200, gin.H{"data": []interface{}{}})
}
