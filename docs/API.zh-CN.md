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

MVP 的审批接口目前只更新审批状态，尚不会自动重放被阻止的执行。

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

