# darwin-ops-mcp

基于 Go 的 Darwin Ops MCP 运维平台，配套 React + TypeScript + Vite 前端。默认本地体验是**安全 mock mode**：不会连接真实 Kubernetes 集群或 Prometheus 服务，并会自动填充示例工具、执行记录和审计日志，方便非程序员立即体验产品。

> English version: [README.md](README.md)

## 包含内容

- 使用 Gin 的 Go 后端
- 使用 Ant Design、TanStack Query、React Router、ECharts 和 Monaco Editor 的 React + TypeScript + Vite 前端
- Docker Compose 栈，包含三个应用服务：`backend`、`frontend`、`postgres`
- mock mode 种子数据
- Tool Registry、Policy Engine、Audit System、Execution History 和 Approval Flow 骨架
- Kubernetes mock adapter
- Prometheus mock adapter
- `docs/` 目录中的文档，包含面向新手的工具中心使用文档

## 1. 如何启动项目

最简单的方式是 Docker。安装 Docker Desktop 或 Docker Engine + Compose plugin 后，在仓库根目录运行：

```bash
make docker-up
```

这会从 GitHub Container Registry 拉取已经由 CI/CD 编译好的后端镜像，在本机仅构建前端，并启动：

- `backend`：端口 `8080`
- `frontend`：端口 `5173`
- `postgres`：端口 `5432`

后端二进制由 GitHub Actions（`.github/workflows/backend-image.yml`）编译，并发布到滚动 GitHub Release 标签 `backend-main`，文件名为 `darwin-ops-mcp-linux-amd64` 和 `darwin-ops-mcp-linux-arm64`。`Dockerfile.backend` 在 Docker 构建阶段下载该二进制，因此部署机器可以重建后端容器，但不需要本地 Go 工具链，也不需要执行高开销的 `go build`。同一条流水线也会发布 `ghcr.io/zlylong/ops-mcp-backend:main` 镜像。

如需强制从 GitHub 发布的二进制重新构建本地后端 Docker 镜像：

```bash
make docker-up-local-backend
```

若需要部署指定后端镜像，可在启动 Compose 前设置 `BACKEND_IMAGE`：

```bash
BACKEND_IMAGE=ghcr.io/zlylong/ops-mcp-backend:v1.0.0 docker compose up -d
```

检查后端健康状态：

```bash
curl http://localhost:8080/healthz
```

正常情况下会看到 JSON，其中包含 `"mode":"mock"`，以及 tools、executions、audit records 的数量。

## 2. 如何打开前端

在浏览器中打开：

```text
http://localhost:5173
```

前端会把 `/api` 请求代理到后端容器，因此无需额外配置。

顶部栏的语言选择器可以在英文和中文之间切换 Web UI。选择结果会保存在浏览器中。

## 3. 如何登录或使用 mock 用户

当前 MVP 没有密码登录。执行工具时使用内置 mock 身份：

- Actor：`mock.user`
- Role：`viewer`
- Target：`cluster=demo namespace=default`

默认 Docker 栈会设置：

```text
DARWIN_OPS_MCP_MODE=mock
DARWIN_OPS_MCP_SEED_MOCK=true
```

这意味着应用启动后会自动显示示例执行记录和审计日志。

## 4. 如何执行示例工具

从前端操作：

1. 打开 `http://localhost:5173`。
2. 进入 **Tool Center / 工具中心**。
3. 点击 `k8s.list_pods` 的 **Execute / 执行**。
4. 使用以下 JSON 输入：

```json
{
  "namespace": "default"
}
```

5. 确认并提交执行。
6. 结果中应显示 mock Kubernetes pods。

也可以用 curl 测试相同流程：

```bash
curl -s http://localhost:8080/api/v1/tools/k8s.list_pods/execute \
  -H 'Content-Type: application/json' \
  -d '{
    "actor": "mock.user",
    "role": "viewer",
    "target": "cluster=demo namespace=default",
    "parameters": {"namespace": "default"}
  }'
```

## 5. 如何查看审计日志

从前端操作：

1. 打开 `http://localhost:5173`。
2. 进入 **Audit Center / 审计中心**。
3. 可以看到预置的 mock 审计事件以及新产生的工具执行记录。
4. 点击某一行可查看详情。

使用 curl：

```bash
curl http://localhost:8080/api/v1/audit
```

`password`、`secret`、`token`、`api_key`、`authorization` 等敏感参数名会在审计记录保存前被脱敏。

## 6. 如何停止项目

停止容器但保留 PostgreSQL volume：

```bash
make docker-down
```

停止容器并重置本地 PostgreSQL volume：

```bash
make reset-db
```

执行 `make reset-db` 后，再运行 `make docker-up` 即可启动全新的环境。

## 常用命令

```bash
make setup        # 为本地开发安装 Go 和前端依赖
make dev          # 不使用 Docker，启动后端和前端开发服务器
make test         # 运行后端测试和前端类型检查
make docker-up    # 从 CI 构建的 GitHub 二进制重建后端，构建前端，并启动 Docker 服务
make docker-up-local-backend # 仅强制后端 Docker 镜像从 GitHub 二进制重建
make docker-down  # 停止 Docker 容器，保留数据库 volume
make reset-db     # 停止 Docker 容器并删除数据库 volume
```

## 不使用 Docker 的本地开发

前置条件：Go 1.25+、Node.js 20+、npm。

```bash
make setup
make dev
```

然后打开：

- 前端：http://localhost:5173
- 后端健康检查：http://localhost:8080/healthz

## 配置

默认配置是安全 mock mode。可以使用 JSON 配置文件：

```bash
DARWIN_OPS_MCP_CONFIG=config.example.json go run ./backend/cmd/server
# 或
go run ./backend/cmd/server --config config.example.json
```

后端环境变量会覆盖配置文件值：

旧的 `OPS_MCP_*` 和 `MCP_*` 前缀仍会被兼容读取，但新部署建议使用 `DARWIN_OPS_MCP_*`。


- `DARWIN_OPS_MCP_ADDR` 默认 `:8080`
- `DARWIN_OPS_MCP_MODE` 默认 `mock`；设为 `local` 可从挂载的 `/host/proc`、`/host/etc` 和 `/host/usr/lib` 启用只读 Linux 主机采集
- `DARWIN_OPS_MCP_ENV` 默认 `development`；生产环境写工具需要审批
- `DARWIN_OPS_MCP_SEED_MOCK` 默认 `true`；设为 `false` 可不生成示例执行/审计日志
- `DARWIN_OPS_MCP_API_TOKEN` 可选 bearer token，用于保护 `/mcp` 和 `/api/v1/*`
- `DARWIN_OPS_MCP_CONFIG` 可选 JSON 配置文件路径
- `DATABASE_URL` PostgreSQL 连接字符串

前端环境变量：

- `VITE_API_BASE` 可选显式 API base URL。为空时表示同源/proxy。
- `VITE_MOCK_API=true` 启用浏览器侧 mock API client，用于无后端 UI 演示。

## 用户文档

- [工具中心使用文档](docs/TOOL_CENTER.zh-CN.md)：面向新手，详细说明工具搜索、查看、执行、新增、编辑、删除、审批和审计逻辑。
- [API 文档](docs/API.zh-CN.md)：HTTP API 与默认工具目录。
- [MCP 客户端接入](docs/MCP_CLIENT.zh-CN.md)：通过 MCP HTTP 或 OpenAPI 接入外部 AI Agent。
- [测试指南](docs/TESTING.zh-CN.md)：后端测试策略与命令。
- [安全指南](docs/SECURITY.zh-CN.md)：安全保证、脱敏和策略边界。
- [架构文档](docs/ARCHITECTURE.zh-CN.md)：后端/前端结构和请求流。

## 前端 MVP 页面

前端包含左侧菜单布局、顶部环境选择器、顶部集群选择器和用户区域。已实现页面：

- Dashboard / 仪表盘：告警、审批、执行统计，最近执行记录和风险分布图
- Tool Center / 工具中心：搜索、分类/风险/只读筛选、schema 查看器、Monaco JSON 执行弹窗
- Tool Detail / 工具详情
- Execution Center / 执行中心 与 Execution Detail / 执行详情：输入/输出 JSON、策略决策和审计 ID
- Audit Center / 审计中心：用户/工具/环境/风险/状态筛选和详情抽屉
- Approval Center / 审批中心：待审批请求及 approve/reject 操作
- Kubernetes Overview / Kubernetes 概览：namespace 选择器、pod/event 表格、deployment 卡片和日志查看器
- Prometheus Query / Prometheus 查询：快捷查询、PromQL 编辑器、图表结果和原始 JSON 查看器
- Settings / 设置

## 已实现工具

默认 mock mode 包含 Kubernetes、Prometheus 和常用 Linux 工具。

Linux 工具：

- `linux.system_info`
- `linux.load_average`
- `linux.memory_usage`
- `linux.disk_usage`
- `linux.process_list`
- `linux.network_interfaces`
- `linux.service_status`
- `linux.journal_tail`
- `linux.ping`
- `linux.dns_lookup`

Kubernetes 工具：

- `k8s.list_pods`
- `k8s.get_pod_logs`
- `k8s.list_events`
- `k8s.get_deployment_status`

Prometheus 工具：

- `prometheus.query`
- `prometheus.service_error_rate`
- `prometheus.service_latency_p95`
- `prometheus.pod_cpu_usage`
- `prometheus.pod_memory_usage`

当前默认工具都是 mock 工具。大多数为只读工具；`linux.journal_tail` 为中风险并需要审批。每次执行都会经过输入校验、Policy Engine、必要时的 Approval Flow、Audit System 和 Execution History。

## MVP 安全保证

后端**不会**实现任意 Shell 执行、`kubectl exec`、删除 namespace、删除 PVC 或任何资源删除工具。工具执行会被审计，critical 工具默认拒绝，生产环境写操作需要审批。mock mode 永远不会修改真实基础设施。

详情见 [docs/SECURITY.zh-CN.md](docs/SECURITY.zh-CN.md)。
