# API

Base URL: `/api/v1`

## Health

`GET /healthz`

Returns backend status and mode.

## Overview

`GET /api/v1/overview`

Returns counts for clusters, namespaces, workloads, alerts, current mode, and environment.

## Inventory

- `GET /api/v1/clusters`
- `GET /api/v1/namespaces`
- `GET /api/v1/workloads`

All return deterministic mock data when `OPS_MCP_MODE=mock`.

## Tools

`GET /api/v1/tools` returns supported safe tools.

`POST /api/v1/tools/execute`

Request:

```json
{
  "tool": "restart_rollout",
  "actor": "local-user",
  "target": "deployment/api",
  "approved": true,
  "parameters": {"namespace": "default"}
}
```

Successful response:

```json
{
  "auditId": "aud-...",
  "status": "ok",
  "message": "mock mode: no real cluster mutation was performed"
}
```

Blocked unsafe tools return `403`. Production writes without approval return `409`. Unknown tools return `404`.

## Audit

`GET /api/v1/audit` returns in-memory audit events for the current backend process.
