# Architecture

ops-mcp is a Docker-first full-stack operations platform.

## Components

- **Backend:** Go REST API. Entry point: `backend/cmd/server`. Internal packages own config, API routing, audit recording, and ops services.
- **Frontend:** React + TypeScript + Vite + Ant Design. It calls REST endpoints and presents a safe operations dashboard.
- **Database:** PostgreSQL is provided by Docker Compose for future persistence. The initial backend runs from deterministic mock data so local development does not require migrations.
- **Cache:** Redis is optional and provided by Compose for future background/session use.

## Mock mode

`OPS_MCP_MODE=mock` is the default. Mock mode returns deterministic clusters, namespaces, workloads, tools, and audit events. It performs no real Kubernetes or Prometheus calls.

## API style

REST endpoints are versioned under `/api/v1`. See `docs/API.md`.

## Future Helm deployment

Docker Compose is the first deployment target. Helm should be added later after the REST contracts, persistence model, auth model, and approval workflow are stable.
