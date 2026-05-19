package api

import "github.com/zlylong/ops-mcp/backend/internal/domain"

type executeHTTP struct {
	Actor      string         `json:"actor"`
	Role       string         `json:"role"`
	Target     string         `json:"target"`
	Approved   bool           `json:"approved"`
	Parameters map[string]any `json:"parameters"`
}

func domainRequest(req struct {
	Actor      string         `json:"actor"`
	Role       string         `json:"role"`
	Target     string         `json:"target"`
	Approved   bool           `json:"approved"`
	Parameters map[string]any `json:"parameters"`
}) domain.ExecuteRequest {
	return domain.ExecuteRequest{Actor: req.Actor, Role: domain.Role(req.Role), Target: req.Target, Approved: req.Approved, Parameters: req.Parameters}
}
