# AGENTS.md

This repository is ops-mcp. Keep the repository runnable after every task.

## Required workflow

1. Implement backend, frontend, tests, docs, and scripts directly; do not ask the user to manually fix code.
2. Run `make test` after changes.
3. Run `make lint` and `make build` when touching build or runtime code.
4. Keep mock mode working without Kubernetes, Prometheus, PostgreSQL migrations, or Redis.
5. Update README/docs whenever behavior, setup, API, or security assumptions change.

## Safety constraints

- Never add arbitrary shell execution.
- Never add delete namespace.
- Never add delete PVC.
- Never add `kubectl exec`.
- Production write operations must require approval.
- Every tool execution must produce an audit event.
