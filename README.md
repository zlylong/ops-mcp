# ops-mcp

Go-based Ops MCP platform with a React + TypeScript + Vite frontend. The first implementation is intentionally safe-by-default and runs in **mock mode** without real Kubernetes or Prometheus.

## What is included

- Go REST backend under `backend/`
- React + TypeScript + Vite frontend under `frontend/`
- Ant Design UI
- PostgreSQL and optional Redis in Docker Compose
- Mock data mode enabled by default
- Audited tool execution API with explicit safety blocks
- Documentation in `docs/`

## Quick start

Prerequisites: Go 1.24+, Node.js 20+, npm, Docker Compose.

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

Backend environment variables:

- `OPS_MCP_ADDR` default `:8080`
- `OPS_MCP_MODE` default `mock`
- `OPS_MCP_ENV` default `development`; production write tools require approval
- `DATABASE_URL` PostgreSQL connection string
- `REDIS_URL` optional Redis connection string

Frontend environment variables:

- `VITE_API_BASE` optional explicit API base URL. Empty means same-origin/proxy.

## Safety guarantees in this scaffold

The backend does **not** implement arbitrary shell execution, `kubectl exec`, delete namespace, or delete PVC. Tool execution is audited, and production write operations require approval. Mock mode never mutates real infrastructure.

See `docs/SECURITY.md` for details.
