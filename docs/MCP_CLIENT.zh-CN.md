# MCP 客户端接入

> English: [MCP_CLIENT.md](MCP_CLIENT.md)

Darwin Ops MCP 现在提供 REST/OpenAPI（`http://<host>:8080/swagger/doc.json`）和基于 HTTP 的 MCP 风格 JSON-RPC（`http://<host>:8080/mcp`）。`/mcp` 会把现有 Tool Registry 映射成 MCP 工具，同时复用策略判断、审批、执行历史和审计日志。

## 安全设置

如果要暴露给外部 Agent，先设置 API Token：

```bash
export DARWIN_OPS_MCP_API_TOKEN='replace-with-a-strong-random-token'
docker compose up -d backend
```

设置 `DARWIN_OPS_MCP_API_TOKEN` 后，`/mcp` 和 `/api/v1/*` 都需要以下请求头：

```http
Authorization: Bearer <token>
```

`/healthz` 保持免认证，用于健康检查。

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

只有在可信演示环境且未设置 `DARWIN_OPS_MCP_API_TOKEN` 时，才省略 `Authorization` 请求头。

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

## HTTP MCP 客户端

支持 remote HTTP / streamable HTTP MCP 的客户端可以直接连接。

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

### mcporter 烟测

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

## 仅支持 stdio 的 MCP 客户端

部分客户端只会启动本地 stdio MCP server，不能直接填写远程 HTTP MCP URL。此时使用项目内置 proxy 二进制。

从仓库本地构建：

```bash
go build -o bin/darwin-ops-mcp-proxy ./backend/cmd/mcp-proxy
```

也可以从滚动发布 `backend-main` 下载匹配平台的 release asset：

- `darwin-ops-mcp-proxy-linux-amd64`
- `darwin-ops-mcp-proxy-linux-arm64`

proxy 支持 newline-delimited JSON-RPC 和 `Content-Length` framed MCP stdio 消息，会把每个请求转发到 HTTP MCP 入口。

### Claude Desktop / stdio 配置模式

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

如果服务端没有配置 token，可以省略 `env` 块。也可以用 `-token <token>` 传入，但环境变量更不容易进入 shell 历史。

## OpenAPI 兼容方式

如果外部 Agent 不支持 MCP，但支持 OpenAPI，可以导入 `http://192.168.20.166:8080/swagger/doc.json`，然后调用 `POST /api/v1/tools/{name}/execute`。

Linux 主机采集细节见[只读本地 Linux Adapter](LOCAL_LINUX_ADAPTER.zh-CN.md)。

## 当前限制

- 默认部署是安全 mock mode。设置 `DARWIN_OPS_MCP_MODE=local` 后可启用只读 Linux 主机采集，读取挂载的 `/host/proc`、`/host/etc` 和 `/host/usr/lib`；Kubernetes 和 Prometheus 仍为 mock adapter。
- stdio proxy 负责桥接 MCP 工具请求/响应；它本身不新增 OAuth、SSE 会话管理或审批自动重放。
- 审批后不会自动重放执行；审批完成后需要带 `approved: true` 重新调用工具。
