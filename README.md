# darwin-ops-mcp

> 中文版：[README.zh-CN.md](README.zh-CN.md)

Go-based Darwin Ops MCP platform with a React + TypeScript + Vite frontend. The default local experience is **safe mock mode**: it does not connect to a real Kubernetes cluster or Prometheus server, and it seeds sample tools, executions, and audit logs so a non-programmer can try the product immediately.

## What is included

- Go backend using Gin
- React + TypeScript + Vite frontend using Ant Design, TanStack Query, React Router, ECharts, and Monaco Editor
- Docker Compose stack with exactly three app services: `backend`, `frontend`, and `postgres`
- Seed data for mock mode
- Tool Registry, Policy Engine, Audit System, Execution History, and Approval Flow skeleton
- Kubernetes mock adapter
- Prometheus mock adapter
- Documentation in `docs/`, including a beginner-friendly Tool Center guide

## 1. How to start the project

The easiest path is Docker. Install Docker Desktop or Docker Engine with the Compose plugin, then run this from the repository root:

```bash
make docker-up
```

This pulls the latest backend image when available, rebuilds the backend container from the GitHub-published binary without compiling Go locally, builds the frontend locally, and starts:

- `backend` on port `8080`
- `frontend` on port `5173`
- `postgres` on port `5432`

The backend binary is compiled by GitHub Actions (`.github/workflows/backend-image.yml`) and published to the rolling GitHub Release tag `backend-main` as `darwin-ops-mcp-linux-amd64` and `darwin-ops-mcp-linux-arm64`. `Dockerfile.backend` downloads that binary during Docker build, so deployment hosts can rebuild the backend container without a local Go toolchain or expensive `go build`. The same workflow also publishes `ghcr.io/zlylong/ops-mcp-backend:main`.

To force a local backend Docker rebuild from the GitHub-published binary:

```bash
make docker-up-local-backend
```

To deploy a different backend image, set `BACKEND_IMAGE` before starting Compose:

```bash
BACKEND_IMAGE=ghcr.io/zlylong/ops-mcp-backend:v1.0.0 docker compose up -d
```

Check that the backend is healthy:

```bash
curl http://localhost:8080/healthz
```

You should see JSON with `"mode":"mock"` plus counts for tools, executions, and audit records.

## 2. How to open the frontend

Open this URL in your browser:

```text
http://localhost:5173
```

The frontend proxies `/api` requests to the backend container, so no extra configuration is needed.

The language selector in the top bar can switch the web UI between English and Chinese. The selected language is saved in the browser.

## 3. How to login or use mock user

There is no password login in this MVP. Use the built-in mock identity when executing tools:

- Actor: `mock.user`
- Role: `viewer`
- Target: `cluster=demo namespace=default`

The default Docker stack sets:

```text
DARWIN_OPS_MCP_MODE=mock
DARWIN_OPS_MCP_SEED_MOCK=true
```

That means the app starts with sample executions and audit logs already visible.

## 4. How to execute a sample tool

From the frontend:

1. Open `http://localhost:5173`.
2. Go to **Tool Center**.
3. Click **Execute** for `k8s.list_pods`.
4. Use this JSON input:

```json
{
  "namespace": "default"
}
```

5. Confirm/submit the execution.
6. The result should show mock Kubernetes pods.

You can also test the same flow with curl:

```bash
curl -s http://localhost:8080/api/v1/tools/k8s.list_pods/execute \
  -H 'Content-Type: application/json' \
  -d '{
    "actor": "mock.user",
    "role": "viewer",
    "target": "cluster=demo namespace=default",
    "parameters": {"namespace": "default"}
  }'
```

## 5. How to view audit logs

From the frontend:

1. Open `http://localhost:5173`.
2. Go to **Audit Center**.
3. You will see seeded mock audit events and any new tool executions.
4. Click a row to inspect details.

With curl:

```bash
curl http://localhost:8080/api/v1/audit
```

Sensitive parameter names such as `password`, `secret`, `token`, `api_key`, and `authorization` are masked before audit records are stored.

## 6. How to stop the project

Stop containers but keep the PostgreSQL volume:

```bash
make docker-down
```

Stop containers and reset the local PostgreSQL volume:

```bash
make reset-db
```

After `make reset-db`, run `make docker-up` again to start a fresh stack.

## Common commands

```bash
make setup        # install Go and frontend dependencies for local development
make dev          # run backend and frontend dev servers without Docker
make test         # run backend tests and frontend type checks
make docker-up    # rebuild backend from CI-built GitHub binary, build frontend, and start Docker services
make docker-up-local-backend # force only the backend Docker image to rebuild from the GitHub binary
make docker-down  # stop Docker containers, keep database volume
make reset-db     # stop Docker containers and delete the database volume
```

## Local development without Docker

Prerequisites: Go 1.25+, Node.js 20+, npm.

```bash
make setup
make dev
```

Then open:

- Frontend: http://localhost:5173
- Backend health: http://localhost:8080/healthz

## Configuration

Default config is safe mock mode. Use a JSON config file:

```bash
DARWIN_OPS_MCP_CONFIG=config.example.json go run ./backend/cmd/server
# or
go run ./backend/cmd/server --config config.example.json
```

Backend environment variables override config file values:

The legacy `OPS_MCP_*` and `MCP_*` prefixes are still accepted for backward compatibility, but new deployments should use `DARWIN_OPS_MCP_*`.


- `DARWIN_OPS_MCP_ADDR` default `:8080`
- `DARWIN_OPS_MCP_MODE` default `mock`
- `DARWIN_OPS_MCP_ENV` default `development`; production write tools require approval
- `DARWIN_OPS_MCP_SEED_MOCK` default `true`; set to `false` to start without sample executions/audit logs
- `DARWIN_OPS_MCP_API_TOKEN` optional bearer token protecting `/mcp` and `/api/v1/*`
- `DARWIN_OPS_MCP_CONFIG` optional JSON config file path
- `DATABASE_URL` PostgreSQL connection string

Frontend environment variables:

- `VITE_API_BASE` optional explicit API base URL. Empty means same-origin/proxy.
- `VITE_MOCK_API=true` enables the browser-side mock API client for UI-only demos without backend.

## User guides

- [Tool Center User Guide](docs/TOOL_CENTER.md): beginner-friendly guide for searching, viewing, executing, creating, editing, deleting, approving, and auditing tools.
- [API Documentation](docs/API.md): HTTP API and default tool catalog.
- [MCP Client Integration](docs/MCP_CLIENT.md): connect external AI agents through MCP HTTP or OpenAPI.
- [Testing Guide](docs/TESTING.md): backend test strategy and commands.
- [Security Guide](docs/SECURITY.md): safety guarantees, masking, and policy boundaries.
- [Architecture](docs/ARCHITECTURE.md): backend/frontend structure and request flow.

## Frontend MVP pages

The frontend includes a left sidebar layout, top environment selector, top cluster selector, and user area. Implemented pages:

- Dashboard with alert/approval/execution statistics, recent executions, and risk distribution chart
- Tool Center with search, category/risk/read-only filters, schema viewer, and Monaco JSON execution modal
- Tool Detail
- Execution Center and Execution Detail with input/output JSON, policy decision, and audit ID
- Audit Center with user/tool/environment/risk/status filters and detail drawer
- Approval Center with pending approvals and approve/reject actions
- Kubernetes Overview with namespace selector, pod/event tables, deployment cards, and logs viewer
- Prometheus Query with quick queries, PromQL editor, chart result, and raw JSON viewer
- Settings

## Implemented tools

Default mock mode includes Kubernetes, Prometheus, and common Linux tools.

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

All default tools are mock tools. Most are read-only; `linux.journal_tail` is medium risk and requires approval. Every execution goes through input validation, Policy Engine, Approval Flow when required, Audit System, and Execution History.

## Safety guarantees in this MVP

The backend does **not** implement arbitrary shell execution, `kubectl exec`, delete namespace, delete PVC, or any resource deletion tool. Tool execution is audited, critical tools are denied by default, and production write operations require approval. Mock mode never mutates real infrastructure.

See `docs/SECURITY.md` for details.
