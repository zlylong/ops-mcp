# AGENTS.md

This repository is **ops-mcp**, an Ops MCP platform with a Go backend and React + TypeScript + Vite frontend.

These are permanent rules for Codex and other coding agents working in this repository.

## User context

- The user is a non-programmer.
- Do not leave TODOs that require the user to write code.
- If something is incomplete, either:
  - implement a safe mock that keeps the repository runnable, or
  - create a clear follow-up issue/task description for a future coding agent.
- Every task must leave the repository runnable.

## Security first

- Do not implement arbitrary shell execution.
- Do not implement `kubectl exec`.
- Do not implement resource deletion tools, including namespace deletion, PVC deletion, workload deletion, or cluster deletion.
- Do not hardcode credentials, tokens, kubeconfigs, API keys, database passwords, or other secrets.
- Do not allow production write operations without explicit approval.
- High-risk actions must be policy-checked, audited, and confirmed in the UI.

## Backend rules

- Use Go.
- Use clean architecture boundaries:
  - transport/API layer handles HTTP only;
  - application/service layer owns use cases;
  - domain layer owns core types and policies;
  - adapters/infrastructure layer integrates with external systems.
- All tool calls must go through the Tool Registry.
- All tool calls must go through the Policy Engine.
- All tool calls must write Audit Records.
- All adapters must support mock mode.
- Mock mode must run without real Kubernetes, Prometheus, PostgreSQL migrations, or Redis.
- REST APIs must remain versioned under `/api/v1` unless a migration plan is documented.

## Frontend rules

- Use React + TypeScript + Vite.
- Use Ant Design.
- All API calls must be typed.
- Show loading, error, and empty states for networked UI.
- High-risk actions must show confirmation UI before execution.
- Do not expose unsafe operations in the UI even if they are also blocked by the backend.

## Testing

- Add unit tests for backend policy, audit, and tool registry behavior.
- Add frontend basic tests where practical.
- Run tests before finishing a task.
- At minimum, run:

```bash
make test
```

- When touching build/runtime code, also run:

```bash
make lint
make build
```

## Documentation

Keep documentation current whenever behavior, setup, API, security assumptions, or operational workflows change:

- `README.md`
- `docs/API.md`
- `docs/SECURITY.md`
- relevant architecture or operations docs

## Delivery

- Every change must leave the project runnable with Docker Compose.
- Keep `docker-compose.yml`, Dockerfiles, and Makefile commands working.
- Common commands that should remain valid:

```bash
make dev
make test
make lint
make build
make docker-up
make docker-down
```
