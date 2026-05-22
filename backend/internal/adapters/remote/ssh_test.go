package remote

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSSHAdapterCommandRequiresHostAndCommand(t *testing.T) {
	adapter := NewSSHAdapter()
	_, err := adapter.Command(context.Background(), map[string]any{"command": "uptime"})
	require.ErrorContains(t, err, "host is required")

	_, err = adapter.Command(context.Background(), map[string]any{"host": "127.0.0.1"})
	require.ErrorContains(t, err, "command is required")
}

func TestSSHAdapterRejectsInvalidHost(t *testing.T) {
	adapter := NewSSHAdapter()
	_, err := adapter.Command(context.Background(), map[string]any{"host": "bad host", "command": "uptime"})
	require.ErrorContains(t, err, "invalid characters")
}
