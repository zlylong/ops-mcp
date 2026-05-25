package api

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/domain"
)

func TestJumpServerSSRFHelpers_BlocklistAndPorts(t *testing.T) {
	blocked := []string{"", "127.0.0.1", "10.0.0.1", "172.16.1.1", "192.168.1.1", "169.254.169.254", "::1", "fc00::1", "fe80::1"}
	for _, raw := range blocked {
		assert.True(t, isBlockedProbeIP(net.ParseIP(raw)), raw)
	}
	assert.False(t, isBlockedProbeIP(net.ParseIP("93.184.216.34")))
	assert.True(t, isSSRFHost(" localhost "))
	assert.True(t, isSSRFHost("metadata.google.internal"))
	assert.False(t, isSSRFHost("example.com"))
	assert.True(t, isAllowedProbePort(80))
	assert.True(t, isAllowedProbePort(443))
	assert.False(t, isAllowedProbePort(22))
	assert.Equal(t, 443, portFromString("443"))
	assert.Equal(t, 0, portFromString("22x"))
}

func TestValidateProbeURL_SchemesPortsAndHosts(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, validateProbeURL(ctx, "https://93.184.216.34/health"))
	require.NoError(t, validateProbeURL(ctx, "http://93.184.216.34:80/health"))
	assert.ErrorContains(t, validateProbeURL(ctx, "://bad"), "invalid probe URL")
	assert.ErrorContains(t, validateProbeURL(ctx, "ftp://93.184.216.34/"), "scheme")
	assert.ErrorContains(t, validateProbeURL(ctx, "https://93.184.216.34:8443/"), "port")
	assert.ErrorContains(t, validateProbeURL(ctx, "https://127.0.0.1/"), "blocklist")
	assert.ErrorContains(t, validateProbeURL(ctx, "https://localhost/"), "blocklist")
}

func TestNormalizeJumpServerURL_StripsQueryAndRejectsUnsafe(t *testing.T) {
	normalized, err := normalizeJumpServerURL(" https://93.184.216.34/root/?q=abc#frag ")
	require.NoError(t, err)
	assert.Equal(t, "https://93.184.216.34/root", normalized)

	_, err = normalizeJumpServerURL("")
	assert.ErrorContains(t, err, "baseUrl is required")
	_, err = normalizeJumpServerURL("http://127.0.0.1")
	assert.ErrorContains(t, err, "baseUrl")
	_, err = normalizeJumpServerURL("ssh://93.184.216.34")
	assert.ErrorContains(t, err, "scheme")
}

func TestValidateJumpServerRequest_DefaultsAndValidation(t *testing.T) {
	req, err := validateJumpServerRequest(domain.JumpServerInstanceRequest{
		Name:        " Primary ",
		BaseURL:     "https://93.184.216.34/console/",
		Description: " desc ",
	}, true)
	require.NoError(t, err)
	assert.Equal(t, "Primary", req.Name)
	assert.Equal(t, "https://93.184.216.34/console", req.BaseURL)
	assert.Equal(t, domain.JumpServerAuthToken, req.AuthType)
	assert.Equal(t, "active", req.Status)
	assert.Equal(t, "desc", req.Description)

	_, err = validateJumpServerRequest(domain.JumpServerInstanceRequest{BaseURL: "https://93.184.216.34"}, true)
	assert.ErrorContains(t, err, "name is required")
	_, err = validateJumpServerRequest(domain.JumpServerInstanceRequest{Name: "x", BaseURL: "https://93.184.216.34", AuthType: "bad"}, true)
	assert.ErrorContains(t, err, "authType")
	_, err = validateJumpServerRequest(domain.JumpServerInstanceRequest{Name: "x", BaseURL: "https://93.184.216.34", Status: "bad"}, true)
	assert.ErrorContains(t, err, "status")
}

func TestSafeProbeHTTPClient_BlocksRedirectToPrivateAddress(t *testing.T) {
	client := safeProbeHTTPClient()
	req := httptest.NewRequest(http.MethodGet, "https://93.184.216.34/", nil)
	redirect := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/metadata", nil)
	err := client.CheckRedirect(redirect, []*http.Request{req})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "blocklist")
}
