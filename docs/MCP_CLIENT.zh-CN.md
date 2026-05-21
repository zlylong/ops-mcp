# MCP 客户端接入

> English: [MCP_CLIENT.md](MCP_CLIENT.md)

Darwin Ops MCP 现在提供 REST/OpenAPI（`http://<host>:8080/swagger/doc.json`）和基于 HTTP 的 MCP 风格 JSON-RPC（`http://<host>:8080/mcp`）。`/mcp` 会把现有 Tool Registry 映射成 MCP 工具，同时复用策略判断、审批、执行历史和审计日志。

## 安全设置

如果要暴露给外部 Agent，先设置 API Token：

```bash
export DARWIN_OPS_MCP_API_TOKEN='replace-with-a-long-random-token'
docker compose up -d backend
```

设置 `DARWIN_OPS_MCP_API_TOKEN` 后，`/mcp` 和 `/api/v1/*` 都需要 `Authorization: Bearer <token>`。`/healthz` 保持免认证，用于健康检查。

## MCP HTTP 入口

入口：`POST http://<host>:8080/mcp`

支持方法：`initialize`、`notifications/initialized`、`ping`、`tools/list`、`tools/call`。

### 列出工具

```bash
curl -s http://192.168.20.166:8080/mcp \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <token>' \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'
```

返回内容包含工具名、描述和 JSON input schema。每个工具都支持通用审计/策略字段：`actor`、`role`、`target`、`approved`。

### 调用工具

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

`tools/call` 会返回 MCP text content，并在 `structuredContent` 中包含 `httpStatus`、`result` 和可选 `error`。如果工具需要审批，可通过 REST（`POST /api/v1/approvals/{id}/approve`）或 Web UI 审批，然后按需再次调用并设置 `approved: true`。

## Hermes Agent 配置示例

```yaml
mcp_servers:
  darwin_ops:
    url: "http://192.168.20.166:8080/mcp"
    headers:
      Authorization: "Bearer <token>"
    timeout: 120
    connect_timeout: 30
```

## OpenAPI 兼容方式

如果外部 Agent 不支持 MCP，但支持 OpenAPI，可以导入 `http://192.168.20.166:8080/swagger/doc.json`，然后调用 `POST /api/v1/tools/{name}/execute`。

## 当前限制

- 默认部署是安全 mock mode。设置 `DARWIN_OPS_MCP_MODE=local` 后可启用只读 Linux 主机采集，读取挂载的 `/host/proc` 和 `/host/etc`；Kubernetes 和 Prometheus 仍为 mock adapter。
- 仅支持本地 stdio MCP 的客户端还需要后续补 stdio proxy。
- 审批后不会自动重放执行；审批完成后需要带 `approved: true` 重新调用工具。
