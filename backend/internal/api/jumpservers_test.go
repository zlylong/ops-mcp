package api

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateProbeURLBlocksLiteralPrivateAndMetadataHosts(t *testing.T) {
	ctx := context.Background()
	blocked := []string{
		"http://127.0.0.1/",
		"http://localhost/",
		"http://169.254.169.254/latest/meta-data/",
		"http://10.0.0.1/",
		"http://192.168.1.1/",
		"http://[::1]/",
		"http://metadata.google.internal/",
	}
	for _, raw := range blocked {
		require.Error(t, validateProbeURL(ctx, raw), raw)
	}
}

func TestValidateProbeURLRejectsUnexpectedSchemeAndPort(t *testing.T) {
	ctx := context.Background()
	require.Error(t, validateProbeURL(ctx, "file:///etc/passwd"))
	require.Error(t, validateProbeURL(ctx, "http://example.com:22/"))
}

func TestValidateProbeURLAllowsPublicHTTPHost(t *testing.T) {
	require.NoError(t, validateProbeURL(context.Background(), "https://example.com/"))
}
