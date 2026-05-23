# API（中文版）

Base URL：`/api/v1`

> English version: [API.md](API.md)

## Health

`GET /healthz`

返回后端状态、模式和环境。

## Dashboard summary

`GET /api/v1/dashboard/summary`

响应示例：

```json
{
  "mode": "local",
  "environment": "development",
  "tools": 19,
  "executions": 0,
  "auditRecords": 0,
  "approvals": 0
}
```

## Agent API 密钥

`GET /api/v1/agent-keys`

列出已颁发的 Agent API Key 元数据。响应只包含 `id`、`name`、`actor`、`role`、`scopes`、`keyPrefix`、`status`、`createdAt`、`expiresAt`、`lastUsedAt`、`revokedAt` 等字段，**不会返回明文 secret 或 hash**。

`POST /api/v1/agent-keys`

颁发新的 Agent API Key。启用 `DARWIN_OPS_MCP_API_TOKEN` 时，该接口必须使用 master token 调用；普通 Agent key 不能继续创建或查看其他 key。

请求示例：

```json
{
  "name": "opsagent topic 436",
  "actor": "opsagent-topic-436",
  "role": "viewer",
  "reason": "read-only inspection automation",
  "scopes": ["tools:execute", "applications:create"],
  "expiresInHrs": 168
}
```

响应示例：

```json
{
  "id": "key-...",
  "name": "opsagent topic 436",
  "actor": "opsagent-topic-436",
  "role": "viewer",
  "keyPrefix": "domcp_...",
  "status": "active",
  "createdAt": "2026-05-22T10:00:00Z",
  "expiresAt": "2026-05-29T10:00:00Z",
  "secret": "domcp_..."
}
```

`secret` 只在创建响应中出现一次；调用方必须立即保存。服务端仅保存 SHA-256 hash 和短前缀。当前实现与 execution/audit 一样使用进程内存存储，后端重启后已颁发 key 会失效；后续接入持久化存储时应迁移为数据库表。

`POST /api/v1/agent-keys/:id/revoke` 或 `DELETE /api/v1/agent-keys/:id`

吊销 key。吊销后再次使用该 bearer token 会返回 `401 Unauthorized`。

Agent key 使用方式与原始 bearer token 相同：

```http
Authorization: Bearer domcp_...
```

如果请求体没有显式 `actor`，后端会使用 key 绑定的 `actor` 作为默认执行身份；仍建议 Agent 明确传递稳定 `actor` 与 `X-Trace-ID`，便于审计。

## Tool Registry

`GET /api/v1/tools`

列出已注册工具。

`GET /api/v1/tools/:name`

返回单个工具详情，包括分类、是否只读、风险等级、是否需要审批以及输入 schema。

已实现工具：

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

## Execute tool

`POST /api/v1/tools/:name/execute`

请求：

```json
{
  "actor": "local-user",
  "role": "viewer",
  "target": "default/api",
  "approved": false,
  "parameters": {
    "namespace": "default"
  }
}
```

成功响应：

```json
{
  "executionId": "exe-...",
  "auditId": "aud-...",
  "status": "succeeded",
  "message": "tool executed",
  "data": {}
}
```

错误响应：

- `400` JSON 无效或输入校验失败
- `403` 策略拒绝
- `404` 未知工具
- `409` 需要审批
- `500` adapter 执行失败

## Execution History

`GET /api/v1/executions`

按时间倒序列出执行记录。

`GET /api/v1/executions/:id`

返回单条执行记录。

## Audit

`GET /api/v1/audit`

按时间倒序返回内存中的审计记录。包含 password、token、secret、api key、authorization 和 credential 等敏感标记的参数 key 会被脱敏。

## Approval Flow Skeleton

`GET /api/v1/approvals`

列出审批请求。

`POST /api/v1/approvals/:id/approve`

将审批标记为 approved。

`POST /api/v1/approvals/:id/reject`

将审批标记为 rejected。

执行审批接口会更新审批状态；批准后，后端会自动重放对应的 pending execution，并在执行记录与审计记录中写入最终结果。

## 默认 Linux 工具目录

后端会在工具中心注册一组常用 Linux 运维工具。在 `mock` 模式下这些工具返回确定性演示数据；在 `local` 模式下会切换为只读真实主机采集，同时仍与其他 MCP 工具一样统一经过策略、审批、执行和审计链路。详见[只读本地 Linux Adapter](LOCAL_LINUX_ADAPTER.zh-CN.md)。

- `linux.system_info`：查看主机名、内核、发行版、架构、运行时长和虚拟化信息。
- `linux.load_average`：查看 1/5/15 分钟负载均值与 CPU 核心数。
- `linux.memory_usage`：查看内存与 swap 容量、使用量和可用量。
- `linux.disk_usage`：按指定 `path` 查看文件系统容量与使用率。
- `linux.process_list`：查看按资源使用展示的进程列表。
- `linux.network_interfaces`：查看网卡状态、地址与流量计数。
- `linux.service_status`：按指定 `service` 查看 systemd 服务状态与重启信息。
- `linux.journal_tail`：按指定 `unit` 查看最近 journal 日志；该工具为中风险并需要审批。
- `linux.ping`：按指定 `host` 和可选 `count` 执行连通性检查。
- `linux.dns_lookup`：按指定 `host` 查看 DNS 解析结果。



## 用户管理

> 需启用 `DARWIN_OPS_MCP_API_TOKEN` 或通过已登录的 User Token 调用。以下说明基于 master token 场景。

### 登录

`POST /api/v1/users/login`

用户名密码登录，返回 JWT token（格式：`user:<userID>`）和用户信息。

请求示例：

```json
{
  "username": "admin",
  "password": "admin1234"
}
```

响应示例：

```json
{
  "token": "user:usr-1779513806634305005",
  "user": {
    "id": "usr-1779513806634305005",
    "username": "admin",
    "nickname": "超级管理员",
    "role": "admin",
    "status": "active",
    "createdAt": "2026-05-23T05:23:26Z",
    "updatedAt": "2026-05-23T05:23:29Z"
  },
  "expiresIn": 604800
}
```

错误：`401 invalid username or password`

### 获取当前用户信息

`GET /api/v1/users/me`

返回当前登录用户信息。Master token 场景下返回第一个 admin 用户。

### 更新个人信息

`PUT /api/v1/users/me`

更新当前用户的昵称和邮箱。

请求示例：

```json
{
  "nickname": "新昵称",
  "email": "user@example.com"
}
```

### 修改密码

`PUT /api/v1/users/me/password`

修改当前用户密码（需验证旧密码）。

请求示例：

```json
{
  "oldPassword": "oldpass",
  "newPassword": "newpass123"
}
```

错误：`403 old password is incorrect`；`400 new password must be at least 8 characters`

### 用户列表（Admin）

`GET /api/v1/users`

列出所有用户（Admin 专属）。

### 创建用户（Admin）

`POST /api/v1/users`

创建新用户账号。

请求示例：

```json
{
  "username": "alice",
  "password": "alice1234",
  "nickname": "Alice",
  "role": "viewer"
}
```

角色可选：`admin`、`operator`、`viewer`。默认 `viewer`。

### 获取/更新/删除用户（Admin）

`GET /api/v1/users/:id` | `PUT /api/v1/users/:id` | `DELETE /api/v1/users/:id`

### 重置用户密码（Admin）

`PUT /api/v1/users/:id/password`

强制重置指定用户密码。

请求示例：

```json
{
  "newPassword": "newpass123"
}
```
