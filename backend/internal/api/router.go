package api

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/app"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/audit"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/config"
	_ "github.com/zlylong/darwin-ops-mcp/docs"
)

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
	r.Use(gin.Recovery(), traceIDMiddleware(), requestLogger(logger), cors())
	s := &Server{cfg: cfg, registry: registry, auditor: auditor, logger: logger}
	r.GET("/healthz", s.health)

	protected := r.Group("")
	protected.Use(authRequired(cfg.APIToken, registry))
	protected.GET("/mcp", s.mcp)
	protected.POST("/mcp", s.mcp)

	publicV1 := r.Group("/api/v1")
	publicV1.POST("/users/login", s.login)

	swaggerHandler := ginSwagger.WrapHandler(swaggerFiles.NewHandler())
	r.GET("/swagger", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, "/swagger/index.html") })
	r.GET("/swagger/*any", func(c *gin.Context) {
		if c.Param("any") == "/" || c.Param("any") == "" {
			c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
			return
		}
		swaggerHandler(c)
	})

	v1 := protected.Group("/api/v1")
	v1.GET("/users/me", s.getMe)
	v1.PUT("/users/me", s.updateMe)
	v1.PUT("/users/me/password", s.changeMyPassword)
	v1.GET("/users", s.listUsers)
	v1.POST("/users", s.createUser)
	v1.GET("/users/:id", s.getUser)
	v1.PUT("/users/:id", s.updateUser)
	v1.DELETE("/users/:id", s.deleteUser)
	v1.PUT("/users/:id/password", s.changeUserPassword)
	v1.GET("/dashboard/summary", s.dashboardSummary)
	v1.GET("/jumpservers", s.listJumpServers)
	v1.POST("/jumpservers", s.createJumpServer)
	v1.GET("/jumpservers/:id", s.getJumpServer)
	v1.PUT("/jumpservers/:id", s.updateJumpServer)
	v1.DELETE("/jumpservers/:id", s.deleteJumpServer)
	v1.POST("/jumpservers/:id/test", s.testJumpServer)
	v1.GET("/agent-keys", s.listAgentAPIKeys)
	v1.POST("/agent-keys", s.createAgentAPIKey)
	v1.DELETE("/agent-keys/:id", s.revokeAgentAPIKey)
	v1.POST("/agent-keys/:id/revoke", s.revokeAgentAPIKey)
	v1.GET("/tools", s.tools)
	v1.POST("/tools", s.createTool)
	v1.GET("/tools/:name", s.toolDetail)
	v1.PUT("/tools/:name", s.updateTool)
	v1.DELETE("/tools/:name", s.deleteTool)
	v1.POST("/tools/:name/execute", s.executeTool)
	v1.GET("/executions", s.executions)
	v1.GET("/executions/:id", s.executionDetail)
	v1.GET("/audit", s.auditRecords)
	v1.GET("/applications", s.listApplications)
	v1.POST("/applications", s.submitApplication)
	v1.POST("/applications/:id/approve", s.approveApplication)
	v1.POST("/applications/:id/reject", s.rejectApplication)
	v1.GET("/approvals", s.approvals)
	v1.POST("/approvals/:id/approve", s.approve)
	v1.POST("/approvals/:id/reject", s.reject)
	return r
}

func traceIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader(TraceIDHeader)
		if traceID == "" {
			traceID = generateTraceID()
		}
		c.Set(TraceIDHeader, traceID)
		c.Header(TraceIDHeader, traceID)
		c.Next()
	}
}

func requestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		if q := c.Request.URL.RawQuery; q != "" {
			path += "?" + q
		}
		traceID, _ := c.Get(TraceIDHeader)
		c.Next()
		logger.Info("http request", "method", c.Request.Method, "path", path,
			"status", c.Writer.Status(), "latency_ms", time.Since(start).Milliseconds(),
			"client_ip", c.ClientIP(), "trace_id", traceID)
	}
}

func generateTraceID() string {
	return "tr-" + strconv.FormatInt(time.Now().UTC().UnixNano(), 36)
}

func (s *Server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "mode": s.cfg.Mode, "environment": s.cfg.Environment,
		"tools": len(s.registry.List()), "executions": len(s.registry.Executions()),
		"auditRecords": len(s.auditor.List()), "approvals": len(s.registry.Approvals())})
}

func (s *Server) dashboardSummary(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"mode": s.cfg.Mode, "environment": s.cfg.Environment,
		"tools": len(s.registry.List()), "executions": len(s.registry.Executions()),
		"auditRecords": len(s.auditor.List()), "approvals": len(s.registry.Approvals())})
}

func (s *Server) tools(c *gin.Context) { c.JSON(http.StatusOK, s.registry.List()) }
func (s *Server) toolDetail(c *gin.Context) {
	tool, ok := s.registry.Get(c.Param("name"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "tool not found"})
		return
	}
	c.JSON(http.StatusOK, tool)
}
func (s *Server) executions(c *gin.Context) { c.JSON(http.StatusOK, s.registry.Executions()) }
func (s *Server) executionDetail(c *gin.Context) {
	e, ok := s.registry.Execution(c.Param("id"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "execution not found"})
		return
	}
	c.JSON(http.StatusOK, e)
}
func (s *Server) auditRecords(c *gin.Context) { c.JSON(http.StatusOK, s.auditor.List()) }
func (s *Server) approvals(c *gin.Context) {
	if !s.requireAdminRole(c) {
		return
	}
	c.JSON(http.StatusOK, s.registry.Approvals())
}
func (s *Server) approve(c *gin.Context) {
	if !s.requireAdminRole(c) {
		return
	}
	approval, err := s.registry.Approve(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, approval)
}
func (s *Server) reject(c *gin.Context) {
	if !s.requireAdminRole(c) {
		return
	}
	approval, err := s.registry.Reject(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, approval)
}

// cors returns a CORS middleware.
//
// SECURITY NOTE:
//   - Authorization and X-Actor are intentionally omitted from Access-Control-Allow-Headers.
//     Bearer tokens must only be sent same-origin behind a reverse proxy.
//   - The API uses Authorization header (not cookies), so cross-origin pages
//     cannot supply a valid token regardless of this header configuration.
func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,X-Trace-ID")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
