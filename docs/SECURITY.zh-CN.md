# 安全（中文版）

> English version: [SECURITY.md](SECURITY.md)

## 安全优先的后端流程

每次工具执行都必须经过：

1. Tool Registry 查询
2. 输入校验
3. Policy Engine 评估
4. 仅在允许时执行 adapter
5. 创建 Audit Record
6. 持久化 Execution History

## 非目标与禁止能力

平台有意不实现：

- 任意 Shell 执行
- `kubectl exec`
- Namespace 删除
- PVC 删除
- Workload/resource 删除工具
- 硬编码凭据

未知、不安全或未来新增的 critical 工具请求必须默认拒绝。

## 策略规则

- `viewer` 可以执行只读工具。
- `operator` 只能在 `development` 或 `staging` 中执行中风险写工具。
- 生产环境写操作需要显式审批。
- critical 工具默认拒绝，即使是 admin 也一样；除非未来经过评审的策略明确允许。


## API 授权边界

HTTP 与 MCP 执行 API 不信任调用方提交的身份或审批字段：

- 请求体中的 `actor`、`role`、`approved` 不参与执行授权判断。
- 实际执行人和角色必须从已认证用户或 Agent API Key 派生。
- 客户端不能自声明“已审批”；审批必须走服务端审批接口。
- 工具创建/更新/删除、任务审批决策、工具申请审核决策都需要 admin 权限。
- `X-Actor` 不是安全边界，不应用于审计归属。

## 跨域与供应链加固

- CORS 预检刻意不允许 `Authorization` 或 `X-Actor` 请求头。
- GitHub Actions workflow 应使用最小权限，并将第三方 action 固定到不可变 commit SHA。
- 前端依赖必须固定到具体 semver 范围，发布前应通过 `npm audit --audit-level=moderate`。
- 后端容器镜像从仓库源码多阶段构建，不在镜像构建时下载可变的远程二进制。

## 登录与 bootstrap 凭据

- 密码登录路由公开，但成功登录只返回服务端可识别的 `user:<id>` Bearer token，失败尝试会进入限流。
- 生产环境启动必须同时设置 `DARWIN_OPS_MCP_API_TOKEN` 与 `DARWIN_OPS_MCP_BOOTSTRAP_ADMIN_PASSWORD`。
- 演示密码 `admin1234` 仅保留给非生产 mock/dev 首次启动体验。

## JumpServer SSRF 防护

JumpServer Base URL 与连通性探测限制为 80/443 端口的 `http`/`https`。Host 会同时按字面值与 DNS 解析结果校验；解析到 loopback、private、link-local、CGNAT、metadata、unspecified、IPv6 ULA/link-local 等地址会被拒绝。探测过程中的 redirect 也会在跟随前重新校验。

## 审计脱敏

审计记录会脱敏敏感输入字段。包含以下标记的 key 会被替换为 `***MASKED***`：

- `password`
- `secret`
- `token`
- `api_key`
- `apikey`
- `authorization`
- `credential`

## 运行模式

`DARWIN_OPS_MCP_MODE=mock` 是默认模式。mock adapter 返回确定性的 Kubernetes、Prometheus 和 Linux 数据，不会访问外部基础设施。

`DARWIN_OPS_MCP_MODE=local` 会启用只读 Linux 主机采集。它不暴露任意 Shell 执行，也不会修改主机状态。它从只读挂载路径读取固定主机元数据，并只使用固定形态的 service status、journal tail、ping 和 DNS 命令。`linux.journal_tail` 仍然需要审批，因为日志可能暴露敏感上下文。详见 `docs/LOCAL_LINUX_ADAPTER.zh-CN.md`。

## PostgreSQL

MVP 包含 PostgreSQL 连接支持和 Docker Compose PostgreSQL。在 mock mode 中，执行历史、审批和审计记录存储在内存中。生产实现应将这些记录持久化到 PostgreSQL，并提供不可篡改的审计保留机制。

## Secrets

不要提交真实 kubeconfig、Prometheus 凭据、数据库密码、API key 或 token。未来部署应使用环境变量或 secret manager。
