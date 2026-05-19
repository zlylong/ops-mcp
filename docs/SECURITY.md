# Security

## Security-first backend flow

Every tool execution must pass through:

1. Tool Registry lookup
2. Input validation
3. Policy Engine evaluation
4. Adapter execution only if allowed
5. Audit Record creation
6. Execution History persistence

## Non-goals and prohibited capabilities

The platform intentionally does not implement:

- Arbitrary shell execution
- `kubectl exec`
- Namespace deletion
- PVC deletion
- Workload/resource deletion tools
- Hardcoded credentials

Requests for unknown, unsafe, or future critical tools must be denied by default.

## Policy rules

- `viewer` can execute read-only tools.
- `operator` can execute medium-risk write tools only in `development` or `staging`.
- Production write operations require explicit approval.
- Critical tools are denied by default, even for admin, until a future reviewed policy explicitly allows them.

## Audit masking

Audit records mask sensitive input fields. Keys containing these markers are replaced with `***MASKED***`:

- `password`
- `secret`
- `token`
- `api_key`
- `apikey`
- `authorization`
- `credential`

## Mock mode

`OPS_MCP_MODE=mock` is the default. Mock adapters return deterministic Kubernetes and Prometheus data and do not contact external infrastructure.

## PostgreSQL

The MVP includes PostgreSQL connection support and Docker Compose PostgreSQL. In mock mode, execution history, approvals, and audit records are stored in memory. A production implementation should persist these records to PostgreSQL with immutable audit retention.

## Secrets

Do not commit real kubeconfigs, Prometheus credentials, database passwords, API keys, or tokens. Use environment variables or secret managers in future deployments.
