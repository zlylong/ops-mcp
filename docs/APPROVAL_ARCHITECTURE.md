# MCP Approval Architecture

> 中文: [APPROVAL_ARCHITECTURE.zh-CN.md](APPROVAL_ARCHITECTURE.zh-CN.md)

Darwin Ops MCP is intended to be an operations control plane, not just a REST demo. The production target is:

```text
External AI Agent / MCP Client
        |
        v
MCP Agent Gateway
        |
        +-- Tool Registry
        +-- Permission / Policy
        +-- Tool Approval Center
        +-- Task Approval Center
        +-- Audit Log
        |
        v
Third-party APIs / Servers / Shell / Internal Systems
```

## Current capability map

- **External AI Agent / MCP Client**: `/mcp` exposes JSON-RPC methods `initialize`, `tools/list`, `tools/call`, and `ping`. It reuses the same registry, policy, approval, execution history, and audit path as REST.
- **Tool Registry**: built-in Kubernetes, Prometheus, Linux, and remote SSH tools are registered at backend startup. Runtime custom tools can also be created after approval.
- **Permission / Policy**: the policy engine evaluates role, risk, read-only status, environment, and approval state before a handler can run.
- **Tool Approval Center**: `/api/v1/applications` records requests for tools that are missing or need access. High and critical requests are pending until an operator approves them.
- **Task Approval Center**: execution requests for medium/high-risk tools or tools with `requiresApproval=true` create task approvals. Approving a pending task now executes the original request and writes execution/audit state.
- **Audit Log**: validation failures, denied requests, completed executions, and approved task executions write audit records with action, actor, role, target, and redacted parameters.
- **Third-party servers / shell**: `remote.ssh_command` can execute an approved command on a third-party server through the runtime SSH client and mounted credentials.

## Approval semantics

### Tool approval

Use tool approval when an AI agent or operator needs a tool that is not available yet, or asks for access to a risky tool. The request body may include a `parameters.toolDefinition` object. After an admin approves the application, the backend registers that tool in the runtime registry when it is not already present.

### Task approval

Use task approval for actual command or write execution. A task approval stores the original execution parameters. When an operator approves the task, the backend runs the saved handler call, updates the execution from `pending_approval` to `completed` or `error`, and records an audit event.

`remote.ssh_command` is deliberately high-risk and `requiresApproval=true`; normal MCP calls return `pending_approval` instead of running immediately.

## Operational notes

- `remote.ssh_command` requires an `ssh` binary in the backend image and readable SSH keys/known hosts in the runtime container.
- The Docker image installs `openssh-client`, and docker-compose mounts `/root/.ssh` read-only for the backend service.
- Keep API tokens, SSH keys, and connection strings out of documentation and logs. Use `[REDACTED]` when summarizing incidents.
- For clients that only support stdio MCP, use the `darwin-ops-mcp-proxy` bridge described in [MCP_CLIENT.md](MCP_CLIENT.md).
