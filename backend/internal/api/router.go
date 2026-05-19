package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/example/ops-mcp/backend/internal/audit"
	"github.com/example/ops-mcp/backend/internal/config"
	"github.com/example/ops-mcp/backend/internal/ops"
)

type Router struct {
	cfg     config.Config
	svc     *ops.Service
	auditor audit.Recorder
	logger  *slog.Logger
}

func NewRouter(cfg config.Config, svc *ops.Service, auditor audit.Recorder, logger *slog.Logger) http.Handler {
	r := &Router{cfg: cfg, svc: svc, auditor: auditor, logger: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", r.health)
	mux.HandleFunc("GET /api/v1/overview", r.overview)
	mux.HandleFunc("GET /api/v1/clusters", r.clusters)
	mux.HandleFunc("GET /api/v1/namespaces", r.namespaces)
	mux.HandleFunc("GET /api/v1/workloads", r.workloads)
	mux.HandleFunc("GET /api/v1/tools", r.tools)
	mux.HandleFunc("POST /api/v1/tools/execute", r.execute)
	mux.HandleFunc("GET /api/v1/audit", r.audit)
	return cors(mux)
}

func (r *Router) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, map[string]string{"status": "ok", "mode": r.cfg.Mode})
}
func (r *Router) overview(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, r.svc.Overview(r.cfg.Environment))
}
func (r *Router) clusters(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, r.svc.Clusters())
}
func (r *Router) namespaces(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, r.svc.Namespaces())
}
func (r *Router) workloads(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, r.svc.Workloads())
}
func (r *Router) tools(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, r.svc.Tools()) }
func (r *Router) audit(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, r.auditor.List()) }

func (r *Router) execute(w http.ResponseWriter, req *http.Request) {
	var body ops.ToolRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON body"})
		return
	}
	result, status, err := r.svc.Execute(body, r.cfg.Environment)
	if err != nil {
		writeJSON(w, status, map[string]any{"error": result.Message, "auditId": result.AuditID})
		return
	}
	writeJSON(w, status, result)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
