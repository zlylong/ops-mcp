# MCP Client Integration

> 中文版：[MCP_CLIENT.zh-CN.md](MCP_CLIENT.zh-CN.md)

Darwin Ops MCP exposes REST/OpenAPI (`http://<host>:8080/swagger/doc.json`) and MCP-style JSON-RPC over HTTP (`http://<host>:8080/mcp`). The MCP endpoint maps the existing Tool Registry to MCP methods while preserving policy checks, approvals, execution history, and audit logs.

## Security

Set an API token before exposing the service outside a trusted network:

```bash
export DARWIN_OPS_MCP_API_TOKEN='replace-with-a-long-random-token'
docker compose up -d backend
```

When `DARWIN_OPS_MCP_API_TOKEN` is set, `/mcp` and `/api/v1/*` require `Authorization: Bearer <token>`. `/healthz` remains unauthenticated for health checks.

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

## Hermes Agent configuration

```yaml
mcp_servers:
  darwin_ops:
    url: "http://192.168.20.166:8080/mcp"
    headers:
      Authorization: "Bearer <token>"
    timeout: 120
    connect_timeout: 30
```

## OpenAPI fallback

Agents that do not support MCP but do support OpenAPI can import `http://192.168.20.166:8080/swagger/doc.json` and call `POST /api/v1/tools/{name}/execute`.

## Current limitations

- Default deployment is safe mock mode. Set `DARWIN_OPS_MCP_MODE=local` to enable read-only Linux host collection from mounted `/host/proc` and `/host/etc`; Kubernetes and Prometheus remain mock adapters.
- Stdio proxy packaging for clients that only support local stdio MCP is not included yet.
- Approval replay is not automatic; after approval, re-run the tool with `approved: true`.
