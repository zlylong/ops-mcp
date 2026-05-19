# ops-mcp

Go-based Ops MCP platform with a React + TypeScript + Vite frontend. The backend MVP is **read-only by default**, runs in **mock mode** without real Kubernetes or Prometheus, and exposes a REST Admin API.

## What is included

- Go backend using Gin
- React + TypeScript + Vite frontend using Ant Design
- PostgreSQL connection support and Docker Compose PostgreSQL service
- Optional Redis service in Docker Compose
- JSON config file support via `--config` or `OPS_MCP_CONFIG`
- Tool Registry, Policy Engine, Audit System, Execution History, and Approval Flow skeleton
- Kubernetes mock adapter
- Prometheus mock adapter
- Documentation in `docs/`

## Quick start

Prerequisites: Go 1.25+, Node.js 20+, npm, Docker Compose.

```bash
make dev
```

Then open:

- Frontend: http://localhost:5173
- Backend health: http://localhost:8080/healthz

## Docker local development

```bash
make docker-up
make docker-down
```

Docker Compose starts backend, frontend, PostgreSQL, and Redis. The backend still uses `OPS_MCP_MODE=mock`, so it is safe without a real cluster.

## Common commands

```bash
make dev          # run backend and frontend dev servers
make test         # run backend tests and frontend type checks
make lint         # gofmt/go vet and frontend type checks
make build        # build backend binary and frontend assets
make docker-up    # compose up --build
make docker-down  # compose down -v
```

## Configuration

Default config is safe mock mode. Use a JSON config file:

```bash
OPS_MCP_CONFIG=config.example.json go run ./backend/cmd/server
# or
go run ./backend/cmd/server --config config.example.json
```

Backend environment variables override config file values:

- `OPS_MCP_ADDR` default `:8080`
- `OPS_MCP_MODE` default `mock`
- `OPS_MCP_ENV` default `development`; production write tools require approval
- `OPS_MCP_CONFIG` optional JSON config file path
- `DATABASE_URL` PostgreSQL connection string
- `REDIS_URL` optional Redis connection string

Frontend environment variables:

- `VITE_API_BASE` optional explicit API base URL. Empty means same-origin/proxy.

## Implemented tools

- `k8s.list_pods`
- `k8s.get_pod_logs`
- `k8s.list_events`
- `k8s.get_deployment_status`
- `prometheus.query`
- `prometheus.service_error_rate`
- `prometheus.service_latency_p95`
- `prometheus.pod_cpu_usage`
- `prometheus.pod_memory_usage`

All current tools are read-only mock tools. Every execution goes through input validation, Policy Engine, Audit System, and Execution History.

## Safety guarantees in this MVP

The backend does **not** implement arbitrary shell execution, `kubectl exec`, delete namespace, delete PVC, or any resource deletion tool. Tool execution is audited, critical tools are denied by default, and production write operations require approval. Mock mode never mutates real infrastructure.

See `docs/SECURITY.md` for details.
