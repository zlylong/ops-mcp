# Architecture

> 中文版：[ARCHITECTURE.zh-CN.md](ARCHITECTURE.zh-CN.md)

darwin-ops-mcp is a Docker-first full-stack operations platform.

## Components

- **Backend:** Go + Gin REST API. Entry point: `backend/cmd/server`.
- **Frontend:** React + TypeScript + Vite + Ant Design.
- **Database:** PostgreSQL is provided by Docker Compose. The MVP opens PostgreSQL when real mode is enabled, while mock mode uses in-memory stores.
- **Cache:** Redis is optional and provided by Compose for future background/session use.

## Backend modules

- **Domain:** `backend/internal/domain` contains core types such as Tool, PolicyDecision, AuditRecord, Execution, and Approval.
- **Tool Registry:** `backend/internal/app` owns tool registration, input validation, policy enforcement, audit writes, and execution history.
- **Policy Engine:** `backend/internal/policy` implements role/environment/risk policy decisions.
- **Audit System:** `backend/internal/audit` stores audit records and masks sensitive parameters.
- **Execution History:** `backend/internal/storage` provides in-memory execution records for the MVP.
- **Approval Flow Skeleton:** in-memory approvals support list/approve/reject APIs.
- **Adapters:** `backend/internal/adapters/kubernetes` and `backend/internal/adapters/prometheus` provide mock implementations. `backend/internal/adapters/linux` provides both deterministic mock data and an opt-in read-only local adapter.
- **REST Admin API:** `backend/internal/api` exposes `/api/v1` endpoints through Gin.
- **HTTP MCP Endpoint:** `POST /mcp` exposes the same tool registry to MCP-style JSON-RPC clients without bypassing policy, approval, execution history, or audit.

## Runtime modes

`DARWIN_OPS_MCP_MODE=mock` is the default. Mock mode returns deterministic Kubernetes, Prometheus, and Linux data and performs no real infrastructure calls.

`DARWIN_OPS_MCP_MODE=local` enables read-only Linux host collection from mounted `/host/proc`, `/host/etc`, and `/host/usr/lib`. Kubernetes and Prometheus remain mock adapters in this mode. See `docs/LOCAL_LINUX_ADAPTER.md`.

## API style

REST endpoints are versioned under `/api/v1`. See `docs/API.md`.

## Future Helm deployment

Docker Compose is the first deployment target. Helm should be added later after persistence, auth, approval replay, and real adapter boundaries are stable.
