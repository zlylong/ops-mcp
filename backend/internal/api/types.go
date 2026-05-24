package api

import "github.com/zlylong/darwin-ops-mcp/backend/internal/domain"

// executeHTTP is the HTTP request body for tool execution via the REST API.
//
// SECURITY NOTES:
// - Role and Approved fields are server-side decisions derived from the
//   authenticated identity (agent key or user session). They are accepted
//   in the request body only for API compatibility but are ALWAYS over-ridden
//   server-side in executeTool() before policy evaluation.
//   Callers MUST NOT rely on these fields having any effect.
type executeHTTP struct {
	Actor      string         `json:"actor"`
	Role       string         `json:"role"`     // OVER-RIDDEN server-side; caller role is ignored
	Target     string         `json:"target"`
	Approved   bool           `json:"approved"` // OVER-RIDDEN server-side; caller approval flag is ignored
	Parameters map[string]any `json:"parameters"`
}

// domainRequest converts an executeHTTP request to an internal ExecuteRequest.
// After conversion, callers MUST override Role and Approved server-side.
func domainRequest(req struct {
	Actor      string         `json:"actor"`
	Role       string         `json:"role"`
	Target     string         `json:"target"`
	Approved   bool           `json:"approved"`
	Parameters map[string]any `json:"parameters"`
}) domain.ExecuteRequest {
	return domain.ExecuteRequest{
		Actor:      req.Actor,
		Role:       domain.Role(req.Role), // OVER-RIDDEN server-side; caller role is ignored
		Target:     req.Target,
		Approved:   req.Approved, // OVER-RIDDEN server-side; caller approval flag is ignored
		Parameters: req.Parameters,
	}
}
