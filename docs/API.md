# API

Base URL: `/api/v1`

## Health

`GET /healthz`

Returns backend status, mode, and environment.

## Dashboard summary

`GET /api/v1/dashboard/summary`

Response example:

```json
{
  "mode": "mock",
  "environment": "development",
  "tools": 9,
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

- `k8s.list_pods`
- `k8s.get_pod_logs`
- `k8s.list_events`
- `k8s.get_deployment_status`
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
