package api

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/app"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/audit"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/config"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/domain"
	_ "github.com/zlylong/darwin-ops-mcp/docs"
)

// TraceIDHeader is the HTTP header used to propagate trace IDs.
// Clients may pass an existing trace ID; if absent, one is generated.
const TraceIDHeader = "X-Trace-ID"

type Server struct {
	cfg      config.Config
	registry *app.Registry
	auditor  audit.Recorder
	logger   *slog.Logger
}

func NewRouter(cfg config.Config, registry *app.Registry, auditor audit.Recorder, logger *slog.Logger) http.Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), requestLogger(logger), cors())
	s := &Server{cfg: cfg, registry: registry, auditor: auditor, logger: logger}
	r.GET("/healthz", s.health)

	// Swagger UI. The generated docs package is registered via blank import above.
	swaggerHandler := ginSwagger.WrapHandler(swaggerFiles.NewHandler())
	r.GET("/swagger", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})
	r.GET("/swagger/*any", func(c *gin.Context) {
		if c.Param("any") == "/" || c.Param("any") == "" {
			c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
			return
		}
		swaggerHandler(c)
	})
	v1 := r.Group("/api/v1")
	v1.GET("/dashboard/summary", s.dashboardSummary)
	v1.GET("/tools", s.tools)
	v1.POST("/tools", s.createTool)
	v1.GET("/tools/:name", s.toolDetail)
	v1.PUT("/tools/:name", s.updateTool)
	v1.DELETE("/tools/:name", s.deleteTool)
	v1.POST("/tools/:name/execute", s.executeTool)
	v1.GET("/executions", s.executions)
	v1.GET("/executions/:id", s.executionDetail)
	v1.GET("/audit", s.auditRecords)
	v1.GET("/approvals", s.approvals)
	v1.POST("/approvals/:id/approve", s.approve)
	v1.POST("/approvals/:id/reject", s.reject)
	return r
}

// requestLogger is a middleware that logs HTTP requests with structured fields
// for method, path, status, duration, client IP, and trace ID.
// It also propagates or generates a trace ID for the request lifetime.
func requestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		if query != "" {
			path = path + "?" + query
		}
		traceID := c.GetHeader(TraceIDHeader)
		if traceID == "" {
			traceID = generateTraceID()
		}
		c.Set(TraceIDHeader, traceID)
		c.Header(TraceIDHeader, traceID)

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		logger.Info("http request",
			"method", c.Request.Method,
			"path", path,
			"status", status,
			"latency_ms", latency.Milliseconds(),
			"client_ip", c.ClientIP(),
			"trace_id", traceID,
		)
	}
}

// generateTraceID creates a short random trace ID.
// In production this should be a full UUID; short format suffices for mock mode.
func generateTraceID() string {
	return "tr-" + strconv.FormatInt(time.Now().UTC().UnixNano(), 36)
}

func (s *Server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "mode": s.cfg.Mode, "environment": s.cfg.Environment, "tools": len(s.registry.List()), "executions": len(s.registry.Executions()), "auditRecords": len(s.auditor.List()), "approvals": len(s.registry.Approvals())})
}
func (s *Server) dashboardSummary(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"mode": s.cfg.Mode, "environment": s.cfg.Environment, "tools": len(s.registry.List()), "executions": len(s.registry.Executions()), "auditRecords": len(s.auditor.List()), "approvals": len(s.registry.Approvals())})
}
func (s *Server) tools(c *gin.Context)    { c.JSON(http.StatusOK, s.registry.List()) }
func (s *Server) createTool(c *gin.Context) {
	var tool domain.Tool
	if err := c.ShouldBindJSON(&tool); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return
	}
	created, err := s.registry.CreateTool(tool)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, created)
}
func (s *Server) toolDetail(c *gin.Context) {
	tool, ok := s.registry.Get(c.Param("name"))
	if !ok {
		c.JSON(404, gin.H{"error": "tool not found"})
		return
	}
	c.JSON(200, tool)
}
func (s *Server) updateTool(c *gin.Context) {
	var tool domain.Tool
	if err := c.ShouldBindJSON(&tool); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return
	}
	updated, err := s.registry.UpdateTool(c.Param("name"), tool)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updated)
}
func (s *Server) deleteTool(c *gin.Context) {
	if err := s.registry.DeleteTool(c.Param("name")); err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
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
func (s *Server) executions(c *gin.Context)   { c.JSON(200, s.registry.Executions()) }
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
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
