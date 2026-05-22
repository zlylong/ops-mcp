# MCP Client Integration

> 中文版：[MCP_CLIENT.zh-CN.md](MCP_CLIENT.zh-CN.md)

Darwin Ops MCP exposes REST/OpenAPI (`http://<host>:8080/swagger/doc.json`) and MCP-style JSON-RPC over HTTP (`http://<host>:8080/mcp`). The MCP endpoint maps the existing Tool Registry to MCP tools while preserving policy checks, approvals, execution history, and audit logs.

## Security

Set an API token before exposing the service outside a trusted network:

```bash
export DARWIN_OPS_MCP_API_TOKEN='replace-with-a-strong-random-token'
docker compose up -d backend
```

When `DARWIN_OPS_MCP_API_TOKEN` is set, `/mcp` and `/api/v1/*` require this header:

```http
Authorization: Bearer <token>
```

`/healthz` remains unauthenticated for health checks.

## MCP HTTP endpoint

Endpoint: `POST http://<host>:8080/mcp`

Supported methods: `initialize`, `notifications/initialized`, `ping`, `tools/list`, `tools/call`.

### List tools

```bash
curl -s http://192.168.20.166:8080/mcp \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <token>' \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'
```

Omit the `Authorization` header only for trusted demo deployments that do not set `DARWIN_OPS_MCP_API_TOKEN`.

The response includes tool names, descriptions, and JSON input schemas. Tool arguments include common audit/policy fields: `actor`, `role`, `target`, and `approved`.

### Call a tool

```bash
curl -s http://192.168.20.166:8080/mcp \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <token>' \
  -d '{
    "jsonrpc": "2.0",
    "id": "call-1",
    "method": "tools/call",
    "params": {
      "name": "linux.disk_usage",
      "arguments": {
        "actor": "external-agent",
        "role": "viewer",
        "target": "host=192.168.20.166",
        "path": "/"
      }
    }
  }'
```

`tools/call` returns MCP text content plus `structuredContent` with `httpStatus`, `result`, and optional `error`. If a tool requires approval, approve it through REST (`POST /api/v1/approvals/{id}/approve`) or the UI, then call the tool again with `approved: true` if appropriate.

## HTTP MCP clients

Clients that support remote HTTP/streamable HTTP MCP can connect directly.

### Hermes Agent

```yaml
mcp_servers:
  darwin_ops:
    url: "http://192.168.20.166:8080/mcp"
    headers:
      Authorization: "Bearer <token>"
    timeout: 120
    connect_timeout: 30
```

### mcporter smoke test

```bash
npx -y mcporter list \
  --http-url http://192.168.20.166:8080/mcp \
  --name darwin_ops \
  --allow-http \
  --schema

npx -y mcporter call \
  --http-url http://192.168.20.166:8080/mcp \
  --name darwin_ops \
  --allow-http \
  linux.system_info \
  --args '{"actor":"external-agent","role":"viewer","target":"host=192.168.20.166"}' \
  --output json
```

## Stdio-only MCP clients

Some clients only launch local stdio MCP servers and cannot connect to a remote HTTP MCP URL directly. Use the included proxy binary in those clients.

Build locally from the repository:

```bash
go build -o bin/darwin-ops-mcp-proxy ./backend/cmd/mcp-proxy
```

Or download the release asset that matches your platform from the rolling `backend-main` release:

- `darwin-ops-mcp-proxy-linux-amd64`
- `darwin-ops-mcp-proxy-linux-arm64`

The proxy accepts either newline-delimited JSON-RPC or `Content-Length` framed MCP stdio messages and forwards each request to the HTTP MCP endpoint.

### Claude Desktop / stdio config pattern

```json
{
  "mcpServers": {
    "darwin_ops": {
      "command": "/absolute/path/to/darwin-ops-mcp-proxy",
      "args": [
        "-url",
        "http://192.168.20.166:8080/mcp"
      ],
      "env": {
        "DARWIN_OPS_MCP_API_TOKEN": "<token>"
      }
    }
  }
}
```

If no token is configured on the server, omit the `env` block. You can also pass `-token <token>`, but environment variables are easier to keep out of shell history.

## OpenAPI fallback

Agents that do not support MCP but do support OpenAPI can import `http://192.168.20.166:8080/swagger/doc.json` and call `POST /api/v1/tools/{name}/execute`.

For Linux host collection details, see [Read-only Local Linux Adapter](LOCAL_LINUX_ADAPTER.md).

## Current limitations

- Default deployment is safe mock mode. Set `DARWIN_OPS_MCP_MODE=local` to enable read-only Linux host collection from mounted `/host/proc`, `/host/etc`, and `/host/usr/lib`; Kubernetes and Prometheus remain mock adapters.
- The stdio proxy bridges request/response MCP tool calls; it does not add OAuth, SSE session management, or approval replay by itself.
- Approval replay is not automatic; after approval, re-run the tool with `approved: true`.
