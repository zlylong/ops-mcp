# MCP 审批架构

> English: [APPROVAL_ARCHITECTURE.md](APPROVAL_ARCHITECTURE.md)

Darwin Ops MCP 的目标是成为运维控制面，而不是单纯的 REST 演示。生产目标架构是：

```text
External AI Agent / MCP Client
        |
        v
MCP Agent Gateway
        |
        +-- Tool Registry 工具注册表
        +-- Permission / Policy 权限策略
        +-- Tool Approval Center 工具审批中心
        +-- Task Approval Center 执行审批中心
        +-- Audit Log 审计日志
        |
        v
Third-party APIs / Servers / Shell / Internal Systems
```

## 当前能力映射

- **外部 AI Agent / MCP Client**：`/mcp` 暴露 JSON-RPC 方法 `initialize`、`tools/list`、`tools/call`、`ping`。它复用 REST 相同的工具注册表、策略、审批、执行历史和审计链路。
- **工具注册表**：后端启动时注册内置 Kubernetes、Prometheus、Linux 和远程 SSH 工具。审批通过后也可以运行时创建自定义工具。
- **权限策略**：Policy Engine 在 handler 执行前检查角色、风险等级、是否只读、环境和审批状态。
- **工具审批中心**：`/api/v1/applications` 记录缺失工具或高风险工具访问申请。高风险和关键风险请求会保持待审批，直到管理员处理。
- **执行审批中心**：中/高风险工具或 `requiresApproval=true` 的工具执行请求会创建任务审批。现在批准待审批任务后，后端会执行原始请求，并写入执行与审计状态。
- **审计日志**：参数校验失败、策略拒绝、完成执行、审批后执行都会写入审计记录，包含 action、actor、role、target 和脱敏参数。
- **第三方服务器 / Shell**：`remote.ssh_command` 可以通过运行时 SSH 客户端和挂载凭据，在人工批准后到第三方服务器执行命令。

## 审批语义

### 工具审批

当 AI Agent 或操作员需要一个尚不存在的工具，或者申请访问高风险工具时，使用工具审批。请求体可以包含 `parameters.toolDefinition` 对象。管理员批准后，如果工具尚不存在，后端会把该工具注册进运行时工具注册表。

### 执行审批

实际命令执行或写入操作使用执行审批。任务审批会保存原始执行参数。操作员批准后，后端会运行保存的 handler 调用，将执行状态从 `pending_approval` 更新为 `completed` 或 `error`，并记录审计事件。

`remote.ssh_command` 被明确标记为高风险且 `requiresApproval=true`；普通 MCP 调用只会返回 `pending_approval`，不会立即执行。

## 运维说明

- `remote.ssh_command` 要求后端镜像内有 `ssh` 二进制，并且运行时容器能读取 SSH key / known_hosts。
- Docker 镜像会安装 `openssh-client`，docker-compose 会以只读方式把 `/root/.ssh` 挂载给后端服务。
- API token、SSH key、连接串不要写进文档或日志。事故总结中统一使用 `[REDACTED]`。
- 仅支持 stdio MCP 的客户端可使用 [MCP_CLIENT.zh-CN.md](MCP_CLIENT.zh-CN.md) 中描述的 `darwin-ops-mcp-proxy` 桥接器。
