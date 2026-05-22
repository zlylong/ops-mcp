package app

import (
	"github.com/zlylong/darwin-ops-mcp/backend/internal/adapters/remote"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/domain"
)

// RegisterRemoteTools registers approval-gated tools that can reach third-party
// servers through configured credentials, while preserving registry/policy/audit flow.
func RegisterRemoteTools(r *Registry, ssh *remote.SSHAdapter) error {
	return r.Register(domain.Tool{
		Name:             "remote.ssh_command",
		Description:      "Execute an approved shell command on a third-party server over SSH",
		Category:         "remote",
		ReadOnly:         false,
		Risk:             domain.RiskHigh,
		RequiresApproval: true,
		InputSchema: map[string]domain.ParamSchema{
			"host":           {Type: "string", Required: true, Description: "Remote host or IP address"},
			"command":        {Type: "string", Required: true, Description: "Shell command to execute after task approval"},
			"user":           {Type: "string", Required: false, Description: "SSH username", Default: "root"},
			"port":           {Type: "number", Required: false, Description: "SSH port", Default: 22},
			"timeoutSeconds": {Type: "number", Required: false, Description: "Command timeout, max 120 seconds", Default: 30},
		},
	}, ssh.Command)
}
