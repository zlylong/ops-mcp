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

## Agent API Keys

`GET /api/v1/agent-keys`

Lists issued Agent API Key metadata. Responses include fields such as `id`, `name`, `actor`, `role`, `scopes`, `keyPrefix`, `status`, `createdAt`, `expiresAt`, `lastUsedAt`, and `revokedAt`; they never include plaintext secrets or hashes.

`POST /api/v1/agent-keys`

Issues a new Agent API Key. When `DARWIN_OPS_MCP_API_TOKEN` is enabled, this endpoint must be called with the master token; normal Agent keys cannot create or list other keys.

Request example:

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

Response example:

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

`secret` is returned only once in the create response and must be stored immediately by the caller. The backend stores only a SHA-256 hash and a short prefix. The current implementation uses in-process memory like execution/audit records, so issued keys are lost after backend restart; migrate this to a database table when persistent storage is enabled.

`POST /api/v1/agent-keys/:id/revoke` or `DELETE /api/v1/agent-keys/:id`

Revokes a key. After revocation, the same bearer token returns `401 Unauthorized`.

Agent keys are used like the original bearer token:

```http
Authorization: Bearer domcp_...
```

If a request body omits `actor`, the backend falls back to the key-bound `actor` as the execution identity. Agents should still pass a stable `actor` and `X-Trace-ID` explicitly for auditability.

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



## User Management

> Requires `DARWIN_OPS_MCP_API_TOKEN` or a valid User Token. Described here for master token usage.

### Login

`POST /api/v1/users/login`

Authenticate with username and password. Returns a Bearer token (`user:<userID>`) and user info.

Example request:

```json
{
  "username": "admin",
  "password": "admin1234"
}
```

Example response:

```json
{
  "token": "user:usr-1779513806634305005",
  "user": {
    "id": "usr-1779513806634305005",
    "username": "admin",
    "nickname": "Administrator",
    "role": "admin",
    "status": "active",
    "createdAt": "2026-05-23T05:23:26Z",
    "updatedAt": "2026-05-23T05:23:29Z"
  },
  "expiresIn": 604800
}
```

Error: `401 invalid username or password`

### Get Current User

`GET /api/v1/users/me`

Returns the authenticated user's profile. With master token, returns the first admin user.

### Update Profile

`PUT /api/v1/users/me`

Updates the authenticated user's nickname and email.

### Change Password

`PUT /api/v1/users/me/password`

Changes the authenticated user's password (requires old password verification).

Errors: `403 old password is incorrect`; `400 new password must be at least 8 characters`

### List Users (Admin)

`GET /api/v1/users`

Lists all users. Admin role required.

### Create User (Admin)

`POST /api/v1/users`

Creates a new user account.

Roles: `admin`, `operator`, `viewer`. Default: `viewer`.

### Get/Update/Delete User (Admin)

`GET /api/v1/users/:id` | `PUT /api/v1/users/:id` | `DELETE /api/v1/users/:id`

### Reset User Password (Admin)

`PUT /api/v1/users/:id/password`

Forcibly resets a user's password.
