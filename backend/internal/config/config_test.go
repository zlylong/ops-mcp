package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadBootstrapAdminPassword(t *testing.T) {
	t.Setenv("DARWIN_OPS_MCP_BOOTSTRAP_ADMIN_PASSWORD", "configured-password")
	cfg := Load()
	require.Equal(t, "configured-password", cfg.BootstrapAdminPassword)
}
