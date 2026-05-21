# 架构（中文版）

darwin-ops-mcp 是一个 Docker-first 的全栈运维平台。

> English version: [ARCHITECTURE.md](ARCHITECTURE.md)

## 组件

- **Backend：** Go + Gin REST API。入口：`backend/cmd/server`。
- **Frontend：** React + TypeScript + Vite + Ant Design。
- **Database：** PostgreSQL 由 Docker Compose 提供。MVP 在 real mode 启用时会打开 PostgreSQL；mock mode 使用内存存储。
- **Cache：** Redis 是可选项，由 Compose 为未来后台任务/session 能力预留。

## 后端模块

- **Domain：** `backend/internal/domain` 包含 Tool、PolicyDecision、AuditRecord、Execution 和 Approval 等核心类型。
- **Tool Registry：** `backend/internal/app` 负责工具注册、输入校验、策略执行、审计写入和执行历史。
- **Policy Engine：** `backend/internal/policy` 实现基于角色、环境和风险等级的策略决策。
- **Audit System：** `backend/internal/audit` 存储审计记录并脱敏敏感参数。
- **Execution History：** `backend/internal/storage` 为 MVP 提供内存执行记录。
- **Approval Flow Skeleton：** 内存审批流支持 list/approve/reject API。
- **Adapters：** `backend/internal/adapters/kubernetes` 和 `backend/internal/adapters/prometheus` 提供 mock 实现。
- **REST Admin API：** `backend/internal/api` 通过 Gin 暴露 `/api/v1` 接口。

## 运行模式

`DARWIN_OPS_MCP_MODE=mock` 是默认模式。mock mode 返回确定性的 Kubernetes、Prometheus 和 Linux 数据，不会调用真实基础设施。

`DARWIN_OPS_MCP_MODE=local` 会从挂载的 `/host/proc`、`/host/etc` 和 `/host/usr/lib` 启用只读 Linux 主机采集。在该模式下 Kubernetes 和 Prometheus 仍保持 mock adapter。详见 `docs/LOCAL_LINUX_ADAPTER.zh-CN.md`。

## API 风格

REST endpoint 统一放在 `/api/v1` 下。详见 [API.zh-CN.md](API.zh-CN.md)。

## 未来 Helm 部署

Docker Compose 是第一阶段部署目标。等持久化、认证、审批重放和真实 adapter 边界稳定后，再补充 Helm 部署。
