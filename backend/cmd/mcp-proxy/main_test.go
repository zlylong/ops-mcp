package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestReadWriteLineFrame(t *testing.T) {
	input := bufio.NewReader(strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1}\n"))
	fr, err := readFrame(input)
	require.NoError(t, err)
	require.Equal(t, frameLine, fr.mode)
	require.JSONEq(t, `{"jsonrpc":"2.0","id":1}`, string(fr.body))

	var out bytes.Buffer
	require.NoError(t, writeFrame(&out, fr.mode, []byte(` {"ok":true} `)))
	require.Equal(t, "{\"ok\":true}\n", out.String())
}

func TestReadWriteHeaderFrame(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":2}`
	input := bufio.NewReader(strings.NewReader("Content-Length: 24\r\n\r\n" + body))
	fr, err := readFrame(input)
	require.NoError(t, err)
	require.Equal(t, frameHeader, fr.mode)
	require.Equal(t, body, string(fr.body))

	var out bytes.Buffer
	require.NoError(t, writeFrame(&out, fr.mode, []byte(`{"result":{}}`)))
	require.Equal(t, "Content-Length: 13\r\n\r\n{\"result\":{}}", out.String())
}

func TestProxyForwardsAuthorizationAndBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		var req map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Equal(t, "tools/list", req["method"])
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`))
	}))
	defer server.Close()

	p := &proxy{
		cfg:    config{endpoint: server.URL, token: "test-token", timeout: time.Second},
		client: server.Client(),
		in:     bufio.NewReader(strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}` + "\n")),
		out:    &bytes.Buffer{},
		err:    &bytes.Buffer{},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := p.run(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "EOF")
	require.JSONEq(t, `{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`, strings.TrimSpace(p.out.(*bytes.Buffer).String()))
}

func TestParseConfigUsesEnvironment(t *testing.T) {
	env := map[string]string{
		"DARWIN_OPS_MCP_URL":           "http://example.test/mcp",
		"DARWIN_OPS_MCP_API_TOKEN":     "secret-token",
		"DARWIN_OPS_MCP_PROXY_TIMEOUT": "5s",
	}
	cfg := parseConfig(nil, func(key string) string { return env[key] })
	require.Equal(t, "http://example.test/mcp", cfg.endpoint)
	require.Equal(t, "secret-token", cfg.token)
	require.Equal(t, 5*time.Second, cfg.timeout)
}
