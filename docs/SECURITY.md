# Security

## Non-goals and prohibited capabilities

The platform intentionally does not implement:

- Arbitrary shell execution
- `kubectl exec`
- Delete namespace
- Delete PVC

Requests for these tools are blocked and audited.

## Production write protection

Any tool marked as write-capable requires `approved: true` when `OPS_MCP_ENV=production`. Without approval the API returns `409 approval_required` and writes an audit event.

## Auditability

Every tool execution attempt records:

- Audit ID
- Timestamp
- Actor
- Action/tool
- Target
- Approval flag
- Allowed/blocked result
- Reason

The mock implementation stores audit events in memory and emits structured logs to stdout. A production implementation should persist audit events to PostgreSQL with immutable retention.

## Secrets

Do not commit real kubeconfigs, Prometheus credentials, database passwords, API keys, or tokens. Use environment variables or secret managers in future deployments.
