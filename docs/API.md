# API

> 中文版：[API.zh-CN.md](API.zh-CN.md)

Base URL: `/api/v1`

## Health

`GET /healthz`

Returns backend status, mode, and environment.

## Dashboard summary

`GET /api/v1/dashboard/summary`

Response example:

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

Lists registered tools.

`GET /api/v1/tools/:name`

Returns one tool detail, including category, read-only flag, risk, approval requirement, and input schema.

Implemented tools:

Linux tools:

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

Kubernetes tools:

- `k8s.list_pods`
- `k8s.get_pod_logs`
- `k8s.list_events`
- `k8s.get_deployment_status`

Prometheus tools:

- `prometheus.query`
- `prometheus.service_error_rate`
- `prometheus.service_latency_p95`
- `prometheus.pod_cpu_usage`
- `prometheus.pod_memory_usage`

## Execute tool

`POST /api/v1/tools/:name/execute`

Request:

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

Successful response:

```json
{
  "executionId": "exe-...",
  "auditId": "aud-...",
  "status": "succeeded",
  "message": "tool executed",
  "data": {}
}
```

Error responses:

- `400` invalid JSON or input validation failure
- `403` policy denied
- `404` unknown tool
- `409` approval required
- `500` adapter execution failed

## Execution History

`GET /api/v1/executions`

Lists executions newest first.

`GET /api/v1/executions/:id`

Returns one execution record.

## Audit

`GET /api/v1/audit`

Returns in-memory audit records newest first. Sensitive parameter keys such as password, token, secret, api key, authorization, and credential are masked.

## Approval Flow Skeleton

`GET /api/v1/approvals`

Lists approval requests.

`POST /api/v1/approvals/:id/approve`

Marks an approval as approved.

`POST /api/v1/approvals/:id/reject`

Marks an approval as rejected.

The MVP approval endpoints update approval state only. They do not automatically replay blocked executions yet.

## Default Linux Tool Catalog

The backend registers a common Linux operations tool set in the Tools Center. In `mock` mode these tools return deterministic demo data. In `local` mode they switch to read-only host collection while still using the same policy, approval, execution, and audit pipeline as every other MCP tool. See [Read-only Local Linux Adapter](LOCAL_LINUX_ADAPTER.md).

- `linux.system_info`: host, kernel, distribution, architecture, uptime, and virtualization summary.
- `linux.load_average`: 1/5/15 minute load averages and CPU core count.
- `linux.memory_usage`: memory and swap capacity, usage, and availability.
- `linux.disk_usage`: filesystem capacity and usage for a requested `path`.
- `linux.process_list`: top process list with CPU and memory percentages.
- `linux.network_interfaces`: network interface state, addresses, and traffic counters.
- `linux.service_status`: systemd unit active state and restart information for a requested `service`.
- `linux.journal_tail`: recent journal lines for a requested `unit`; marked medium risk and requires approval.
- `linux.ping`: connectivity check for a requested `host` and optional `count`.
- `linux.dns_lookup`: DNS resolution result for a requested `host`.

