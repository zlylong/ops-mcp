# Security

> 中文版：[SECURITY.zh-CN.md](SECURITY.zh-CN.md)

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


## API authorization boundaries

The HTTP and MCP execution APIs never trust caller-supplied identity or approval fields:

- `actor`, `role`, and `approved` in request bodies are ignored for execution authorization.
- Effective actor and role are derived from the authenticated user or Agent API Key.
- Direct execution approval cannot be self-asserted by clients; approval must flow through the server-side approval endpoints.
- Tool create/update/delete, task approval decisions, and tool-application review decisions require admin privileges.
- `X-Actor` is not a security boundary and must not be used for audit attribution.

## Cross-origin and supply-chain hardening

- CORS preflight intentionally does not allow `Authorization` or `X-Actor` headers.
- GitHub Actions workflows should use least-privilege permissions and pin third-party actions to immutable commit SHAs.

## Audit masking

Audit records mask sensitive input fields. Keys containing these markers are replaced with `***MASKED***`:

- `password`
- `secret`
- `token`
- `api_key`
- `apikey`
- `authorization`
- `credential`

## Runtime modes

`DARWIN_OPS_MCP_MODE=mock` is the default. Mock adapters return deterministic Kubernetes, Prometheus, and Linux data and do not contact external infrastructure.

`DARWIN_OPS_MCP_MODE=local` enables read-only Linux host collection. It does not expose arbitrary shell execution and does not mutate host state. It reads fixed host metadata from mounted read-only paths and uses fixed command shapes for service status, journal tail, ping, and DNS. `linux.journal_tail` remains approval-required because logs may expose sensitive context. See `docs/LOCAL_LINUX_ADAPTER.md`.

## PostgreSQL

The MVP includes PostgreSQL connection support and Docker Compose PostgreSQL. In mock mode, execution history, approvals, and audit records are stored in memory. A production implementation should persist these records to PostgreSQL with immutable audit retention.

## Secrets

Do not commit real kubeconfigs, Prometheus credentials, database passwords, API keys, or tokens. Use environment variables or secret managers in future deployments.
