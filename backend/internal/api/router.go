package api

import (
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/swaggo/gin-swagger"
	swaggerFiles "github.com/swaggo/files"
	"github.com/zlylong/ops-mcp/backend/internal/app"
	"github.com/zlylong/ops-mcp/backend/internal/audit"
	"github.com/zlylong/ops-mcp/backend/internal/config"
)

type Server struct {
	cfg      config.Config
	registry *app.Registry
	auditor  audit.Recorder
	logger   *slog.Logger
}

func NewRouter(cfg config.Config, registry *app.Registry, auditor audit.Recorder, logger *slog.Logger) http.Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), cors())
	s := &Server{cfg: cfg, registry: registry, auditor: auditor, logger: logger}
	r.GET("/healthz", s.health)
	
	// Swagger UI - use absolute path for swagger files
	swaggerPath := filepath.Join("/root/ops-mcp/backend", "swagger")
	swaggerHandler := ginSwagger.WrapHandler(swaggerFiles.NewHandler(), ginSwagger.URL("/swagger/swagger.json"))
	r.GET("/swagger/*any", swaggerHandler)
	
	v1 := r.Group("/api/v1")
	v1.GET("/dashboard/summary", s.dashboardSummary)
	v1.GET("/tools", s.tools)
	v1.GET("/tools/:name", s.toolDetail)
	v1.POST("/tools/:name/execute", s.executeTool)
	v1.GET("/executions", s.executions)
	v1.GET("/executions/:id", s.executionDetail)
	v1.GET("/audit", s.auditRecords)
	v1.GET("/approvals", s.approvals)
	v1.POST("/approvals/:id/approve", s.approve)
	v1.POST("/approvals/:id/reject", s.reject)
	return r
}

func (s *Server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "mode": s.cfg.Mode, "environment": s.cfg.Environment, "tools": len(s.registry.List()), "executions": len(s.registry.Executions()), "auditRecords": len(s.auditor.List()), "approvals": len(s.registry.Approvals())})
}
func (s *Server) dashboardSummary(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"mode": s.cfg.Mode, "environment": s.cfg.Environment, "tools": len(s.registry.List()), "executions": len(s.registry.Executions()), "auditRecords": len(s.auditor.List()), "approvals": len(s.registry.Approvals())})
}
func (s *Server) tools(c *gin.Context) { c.JSON(http.StatusOK, s.registry.List()) }
func (s *Server) toolDetail(c *gin.Context) {
	tool, ok := s.registry.Get(c.Param("name"))
	if !ok {
		c.JSON(404, gin.H{"error": "tool not found"})
		return
	}
	c.JSON(200, tool)
}
func (s *Server) executeTool(c *gin.Context) {
	var req executeHTTP
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid JSON body"})
		return
	}
	result, status, err := s.registry.Execute(c.Request.Context(), c.Param("name"), domainRequest(req))
	if err != nil {
		c.JSON(status, gin.H{"error": result.Message, "executionId": result.ExecutionID, "auditId": result.AuditID, "approvalId": result.ApprovalID})
		return
	}
	c.JSON(status, result)
}
func (s *Server) executions(c *gin.Context) { c.JSON(200, s.registry.Executions()) }
func (s *Server) executionDetail(c *gin.Context) {
	exe, ok := s.registry.Execution(c.Param("id"))
	if !ok {
		c.JSON(404, gin.H{"error": "execution not found"})
		return
	}
	c.JSON(200, exe)
}
func (s *Server) auditRecords(c *gin.Context) { c.JSON(200, s.auditor.List()) }
func (s *Server) approvals(c *gin.Context)    { c.JSON(200, s.registry.Approvals()) }
func (s *Server) approve(c *gin.Context) {
	approval, err := s.registry.Approve(c.Param("id"))
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, approval)
}
func (s *Server) reject(c *gin.Context) {
	approval, err := s.registry.Reject(c.Param("id"))
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, approval)
}

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
