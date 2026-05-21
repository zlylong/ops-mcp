# 只读本地 Linux Adapter

> English version: [LOCAL_LINUX_ADAPTER.md](LOCAL_LINUX_ADAPTER.md)

Darwin Ops MCP 默认提供安全 mock 工具目录。如果是在可信单机部署场景，可以通过 `DARWIN_OPS_MCP_MODE=local` 显式启用只读 Linux 主机采集。

local mode 是一个窄边界的可观测性 adapter，不是远程 Shell。它仍然复用 mock mode 的同一套 Tool Registry、策略校验、审批、执行历史、审计脱敏、REST API 和 MCP endpoint。

## 什么时候使用 local mode

当你希望外部 AI Agent 或 Web UI 检查运行 Darwin Ops MCP 的 Docker 主机时，可以使用 local mode：

- 主机身份、内核、发行版、uptime 和虚拟化标记
- Load average 和 CPU core 数量
- 内存与 swap 使用量
- 指定路径的文件系统使用量
- 按常驻内存排序的进程列表
- 网络接口和收发字节计数
- 可选的服务状态、journal tail、ping 和 DNS 检查

不要把 local mode 当成 fleet agent、SSH bastion 或 Kubernetes adapter。它只检查 backend 容器可见的主机/容器 namespace，以及显式只读挂载进来的主机文件系统。

## 使用 Docker Compose 启用 local mode

`docker-compose.yml` 已包含所需只读挂载：

```yaml
services:
  backend:
    environment:
      DARWIN_OPS_MCP_MODE: "${DARWIN_OPS_MCP_MODE:-mock}"
    volumes:
      - /proc:/host/proc:ro
      - /etc:/host/etc:ro
      - /usr/lib:/host/usr/lib:ro
```

在部署主机上创建或更新 `.env`：

```bash
DARWIN_OPS_MCP_MODE=local
# 暴露到可信网络之外前强烈建议设置：
DARWIN_OPS_MCP_API_TOKEN=<choose-a-strong-token>
```

然后重建并重启 backend：

```bash
docker compose build --no-cache backend
docker compose up -d backend
```

## 安全边界

local mode 刻意保持只读：

- 不暴露任意 Shell 执行。
- 不运行用户传入的命令。
- 不修改文件、服务、容器、Kubernetes 资源或 Prometheus 状态。
- 只使用固定形态的只读命令：`systemctl show`、`journalctl`、`ping` 和 DNS 解析。
- `linux.journal_tail` 仍是 medium risk 且需要审批，因为日志可能包含运维上下文或敏感信息。
- `/mcp` 和 `/api/v1/*` 可通过 `DARWIN_OPS_MCP_API_TOKEN` 保护；`/healthz` 保持免认证，供 Docker/监控健康检查使用。

请把返回的主机数据视为运维元数据。只要服务不只暴露在可信 LAN/VPN 内，就应启用 API token。

## 数据来源

local adapter 会优先使用挂载的主机路径：

- `/host/proc`：内核、uptime、load、内存、进程和网络计数器
- `/host/etc/hostname`：主机名
- `/host/etc/os-release` 与 `/host/usr/lib/os-release`：发行版信息；很多系统上 `/etc/os-release` 是指向 `/usr/lib/os-release` 的 symlink
- `statfs(2)`：从容器 mount namespace 读取指定路径文件系统使用量
- Go 的 `net.Interfaces` 和 resolver：网络接口与 DNS 解析

容器注意事项：即使 bind mount 了 `/proc`，`/proc/sys/kernel/hostname` 也可能返回容器 UTS hostname。因此 adapter 会优先使用 `/host/etc/hostname` 作为主机身份。

## MCP 示例

列出工具：

```bash
curl -s http://192.168.20.166:8080/mcp \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <token>' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

获取主机系统信息：

```bash
curl -s http://192.168.20.166:8080/mcp \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <token>' \
  -d '{
    "jsonrpc": "2.0",
    "id": "sysinfo-1",
    "method": "tools/call",
    "params": {
      "name": "linux.system_info",
      "arguments": {
        "actor": "external-agent",
        "role": "viewer",
        "target": "host=192.168.20.166"
      }
    }
  }'
```

获取内存使用量：

```bash
curl -s http://192.168.20.166:8080/mcp \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <token>' \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"linux.memory_usage","arguments":{"actor":"external-agent","role":"viewer","target":"host=192.168.20.166"}}}'
```

## REST 示例

同样的工具也可以通过 REST/OpenAPI 调用：

```bash
curl -s http://192.168.20.166:8080/api/v1/tools/linux.system_info/execute \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <token>' \
  -d '{
    "actor": "operator",
    "role": "viewer",
    "target": "host=192.168.20.166",
    "parameters": {}
  }'
```

## 验证清单

启用 local mode 后，先验证：

```bash
curl -fsS http://localhost:8080/healthz
curl -fsS http://localhost:8080/swagger/index.html | grep -i 'Swagger UI'
curl -fsS http://localhost:8080/api/v1/tools | python3 -c 'import json,sys; print(len(json.load(sys.stdin)))'
```

预期 health 关键字段：

```json
{
  "mode": "local",
  "status": "ok",
  "tools": 19
}
```

然后调用至少一个 local Linux 工具，并确认返回的 `data.source` 是 `local`。

## 故障排查

### `mode` 仍是 `mock`

检查 `.env` 是否位于 `docker-compose.yml` 同目录，并包含 `DARWIN_OPS_MCP_MODE=local`，然后重新创建 backend 容器：

```bash
docker compose up -d --force-recreate backend
```

### Backend 因 PostgreSQL 认证错误退出

local adapter mode 应使用内存 store。如果它尝试连接 PostgreSQL，说明运行中的二进制可能还不是包含 local-mode 修复的版本，请从最新 GitHub Release 二进制重建：

```bash
docker compose build --no-cache backend
docker compose up -d backend
```

### 发行版显示 `unknown`

确认 `/etc` 和 `/usr/lib` 都已只读挂载。Debian 等很多发行版的 `/etc/os-release` 是指向 `/usr/lib/os-release` 的 symlink。

### Hostname 看起来像容器 ID

确认 `/etc:/host/etc:ro` 已挂载。adapter 优先使用 `/host/etc/hostname`；如果缺失，容器 namespace hostname 可能泄漏到结果中。

### `journal_tail` 返回需要审批

这是预期行为。journal/log 内容可能暴露敏感运维上下文，因此 `linux.journal_tail` 虽然只读，但仍是 medium risk 并需要审批。
