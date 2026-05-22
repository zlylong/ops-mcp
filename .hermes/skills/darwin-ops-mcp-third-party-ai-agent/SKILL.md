---
name: darwin-ops-mcp-third-party-ai-agent
description: "Use when a third-party AI Agent needs to learn how to request tool access and call tools through darwin-ops-mcp. Covers REST API, MCP tools/call, application approval, execution approval, policy rules, retries, and audit fields."
version: 1.0.0
author: Hermes Agent
license: MIT
metadata:
  hermes:
    tags: [darwin-ops-mcp, third-party-ai-agent, tool-application, tool-execution, mcp, rest-api, approval]
    related_skills: []
---

# darwin-ops-mcp Third-Party AI Agent Skill

## Overview

This skill is written for **third-party AI Agents** that connect to `darwin-ops-mcp` as an operations control plane.

The agent should not execute operating-system, Kubernetes, Prometheus, SSH, or custom tools directly. It must go through darwin-ops-mcp so that every action is:

1. validated against the tool input schema;
2. checked by the policy engine;
3. routed through the approval workflow when required;
4. recorded in execution history and audit logs.

Two integration surfaces are available:

- **REST API:** `/api/v1/...`
- **HTTP MCP JSON-RPC:** `/mcp` with `tools/list` and `tools/call`

`/healthz` is public. `/mcp` and `/api/v1/*` are protected when `DARWIN_OPS_MCP_API_TOKEN` is configured.

## When to Use

Load this skill when:

- You are an external AI Agent learning how to call darwin-ops-mcp tools.
- You need to request access to a new or high-risk tool.
- You need to understand when a tool call is allowed, denied, or sent to approval.
- You received `202 pending_approval`, `403 policy denied`, or a MCP `structuredContent.isError=true` result.
- You need to integrate through REST, OpenAPI, or MCP.

Do **not** use this skill for backend implementation details unless you are only trying to understand the external API contract.

## Base URLs and Authentication

Default local service URL:

```text
http://192.168.20.166:8080
```

REST base path:

```text
/api/v1
```

MCP HTTP endpoint:

```text
/mcp
```

When `DARWIN_OPS_MCP_API_TOKEN` is configured, pass:

```http
Authorization: Bearer <token>
```

### Agent API Key Issuance

For production-like integrations, prefer per-agent API keys over sharing the master `DARWIN_OPS_MCP_API_TOKEN`. A master-token caller can issue and manage keys through:

```text
GET    /api/v1/agent-keys
POST   /api/v1/agent-keys
POST   /api/v1/agent-keys/:id/revoke
DELETE /api/v1/agent-keys/:id
```

Create request:

```json
{
  "name": "external agent name",
  "actor": "external-agent-01",
  "role": "viewer",
  "reason": "read-only inspection automation",
  "scopes": ["tools:execute", "applications:create"],
  "expiresInHrs": 168
}
```

The create response includes `secret` exactly once. Store it immediately and never commit it to docs, Git, prompts, or logs. Later list responses expose only metadata such as `keyPrefix`, `status`, timestamps, role, actor, and scopes. Current key records are in-memory, so issued keys are invalidated by backend restart until a persistent store is added.

Use the issued key as a normal bearer token:

```http
Authorization: Bearer domcp_...
```

If the request body omits `actor`, the backend falls back to the key-bound actor. Still pass explicit `actor` / `X-Actor` and `X-Trace-ID` whenever possible so audit logs are clear. A normal agent key can call protected API/MCP endpoints but cannot create, list, or revoke keys; key management requires the master token.

All external agents should also set identity and trace fields:

```http
X-Actor: external-agent-name
X-Trace-ID: agent-task-or-run-id
```

`X-Trace-ID` is echoed back as a response header and recorded for request tracing. If omitted, the backend generates one.

## Important Concepts

### Tool Application vs Tool Execution

There are **two different workflows**:

1. **Tool application** (`/api/v1/applications`) — request access to a tool or request that a new tool be added. This is about *permission/capability acquisition*.
2. **Tool execution** (`/api/v1/tools/:name/execute` or MCP `tools/call`) — actually run a registered tool. This is about *performing one operation*.

Do not confuse them:

- If a tool does not exist or the agent needs access to a high-risk capability, submit a **tool application**.
- If a tool already exists and the agent wants to use it now, submit a **tool execution**.

### Application Approval vs Execution Approval

- **Application approval** approves a tool access request. It may register a runtime custom tool if `parameters.toolDefinition` is included.
- **Execution approval** approves a specific blocked tool execution. In current backend behavior, approving the execution automatically replays the saved pending execution and updates its status.

## REST Flow A — Discover Available Tools

### Request

```bash
curl -s http://192.168.20.166:8080/api/v1/tools \
  -H 'Authorization: Bearer <token>' \
  -H 'X-Actor: external-agent-01' \
  -H 'X-Trace-ID: task-123'
```

### Response Shape

```json
[
  {
    "name": "linux.disk_usage",
    "description": "Show disk usage for a path",
    "category": "linux",
    "readOnly": true,
    "risk": "low",
    "requiresApproval": false,
    "inputSchema": {
      "path": {
        "type": "string",
        "required": false,
        "description": "Filesystem path",
        "default": "/"
      }
    }
  }
]
```

### Agent Behavior

- Cache tool metadata for a short period, e.g. 5 minutes.
- Always respect `inputSchema` before calling a tool.
- Treat `name` as the canonical tool identifier. Examples: `linux.system_info`, `linux.disk_usage`, `remote.ssh_command`.
- Do not invent tool names. If missing, submit a tool application.

## REST Flow B — Apply for Tool Access

Use this when the agent needs a missing tool, a high-risk tool, or a tool access duration.

### Endpoint

```text
POST /api/v1/applications
```

### Request Body

```json
{
  "tool": "remote.ssh_command",
  "risk": "high",
  "role": "operator",
  "reason": "Need to restart a failed service on a production host after user approval",
  "durationHrs": 2,
  "parameters": {
    "targetHost": "192.168.20.166",
    "purpose": "incident-remediation"
  }
}
```

### Headers

```http
Authorization: Bearer <token>
X-Actor: external-agent-01
X-Trace-ID: task-123
Content-Type: application/json
```

### Required Fields

- `tool`: tool name being requested.
- `risk`: one of `low`, `medium`, `high`, `critical`.
- `role`: one of `viewer`, `operator`, `admin`.
- `reason`: clear human-readable justification.

### Optional Fields

- `durationHrs`: access duration in hours. `0` or omitted defaults to `24`.
- `parameters`: extra context. If it includes `toolDefinition`, application approval can register a runtime custom tool.

### Auto-Approval Rules for Applications

The application endpoint uses risk-based application status:

- `low` and `medium` → `approved` automatically.
- `high` and `critical` → `pending` for admin review.

### Response: Auto-Approved Application

```json
{
  "id": "app-m1t3k9xyz",
  "tool": "linux.disk_usage",
  "risk": "low",
  "role": "viewer",
  "actor": "external-agent-01",
  "reason": "Need to inspect disk usage",
  "status": "approved",
  "decision": "auto-approved (low/medium risk)",
  "durationHrs": 24,
  "parameters": {},
  "createdAt": "2026-05-22T08:00:00Z"
}
```

### Response: Pending Application

```json
{
  "id": "app-m1t3k9xyz",
  "tool": "remote.ssh_command",
  "risk": "high",
  "role": "operator",
  "actor": "external-agent-01",
  "reason": "Need to restart a failed service",
  "status": "pending",
  "decision": "pending review (high/critical risk)",
  "durationHrs": 2,
  "createdAt": "2026-05-22T08:00:00Z"
}
```

### Poll Applications

```bash
curl -s http://192.168.20.166:8080/api/v1/applications \
  -H 'Authorization: Bearer <token>'
```

Admin endpoints:

```text
POST /api/v1/applications/:id/approve
POST /api/v1/applications/:id/reject
```

As a third-party AI Agent, do not approve your own high/critical application unless explicitly operating under an admin-controlled automation context.

## REST Flow C — Execute a Tool

### Endpoint

```text
POST /api/v1/tools/:name/execute
```

Example for `linux.disk_usage`:

```bash
curl -s http://192.168.20.166:8080/api/v1/tools/linux.disk_usage/execute \
  -H 'Authorization: Bearer <token>' \
  -H 'X-Actor: external-agent-01' \
  -H 'X-Trace-ID: task-123' \
  -H 'Content-Type: application/json' \
  -d '{
    "actor": "external-agent-01",
    "role": "viewer",
    "target": "host=192.168.20.166",
    "approved": false,
    "parameters": {
      "path": "/"
    }
  }'
```

### Execute Request Fields

```json
{
  "actor": "external-agent-01",
  "role": "viewer",
  "target": "host=192.168.20.166",
  "approved": false,
  "parameters": {}
}
```

- `actor`: who is requesting the execution. Use stable agent identity.
- `role`: `viewer`, `operator`, or `admin`.
- `target`: human-readable target for audit, e.g. `host=192.168.20.166`, `namespace=default`, `service=api`.
- `approved`: set `true` only after a valid approval path. Do not set it blindly.
- `parameters`: tool-specific parameters matching `inputSchema`.

### Response: Completed

HTTP `200`:

```json
{
  "executionId": "exe-abc123",
  "auditId": "aud-xyz",
  "status": "completed",
  "message": "completed",
  "data": {
    "path": "/",
    "usedPercent": 42.1
  }
}
```

Agent action:

- Use `data` as the result.
- Store `executionId`, `auditId`, and `X-Trace-ID` for traceability.

### Response: Validation Failed

HTTP `400`:

```json
{
  "error": "missing required parameter: path",
  "executionId": "exe-abc123",
  "auditId": "aud-xyz"
}
```

Agent action:

- Fix request parameters according to `inputSchema`.
- Do not retry unchanged.

### Response: Policy Denied

HTTP `403`:

```json
{
  "error": "policy denied",
  "executionId": "exe-abc123",
  "auditId": "aud-xyz"
}
```

Agent action:

- Do not retry the same request.
- Check role, tool `readOnly`, risk, environment, and approval status.
- If the tool capability is needed, submit a tool application or ask a human/admin.

### Response: Pending Execution Approval

HTTP `202`:

```json
{
  "executionId": "exe-pending-456",
  "approvalId": "apr-pending-789",
  "status": "pending_approval",
  "message": "pending approval"
}
```

Agent action:

1. Stop re-submitting the same tool call.
2. Report that approval is required.
3. Poll `GET /api/v1/approvals` or wait for a UI/admin decision.
4. When approved, the backend automatically executes the saved pending request.
5. Check `GET /api/v1/executions/:executionId` for final status.

### Response: Handler Error

HTTP `500`:

```json
{
  "error": "adapter execution failed",
  "executionId": "exe-abc123",
  "auditId": "aud-xyz"
}
```

Agent action:

- Treat as tool/runtime failure.
- Retry only if the operation is idempotent and safe.
- Use exponential backoff for transient infrastructure errors.

## Execution Approval Workflow

### List Pending Approvals

```bash
curl -s http://192.168.20.166:8080/api/v1/approvals \
  -H 'Authorization: Bearer <token>'
```

Response:

```json
[
  {
    "id": "apr-pending-789",
    "executionId": "exe-pending-456",
    "tool": "remote.ssh_command",
    "actor": "external-agent-01",
    "target": "host=192.168.20.166 service=nginx",
    "status": "pending",
    "reason": "pending approval for high",
    "createdAt": "2026-05-22T08:00:00Z"
  }
]
```

### Approve / Reject

Admin-controlled endpoints:

```text
POST /api/v1/approvals/:id/approve
POST /api/v1/approvals/:id/reject
```

Current backend behavior:

- `approve` marks the approval `approved` and automatically replays the saved pending execution.
- `reject` marks the approval `rejected` and updates the execution status to `rejected`.

After approval, check execution detail:

```bash
curl -s http://192.168.20.166:8080/api/v1/executions/exe-pending-456 \
  -H 'Authorization: Bearer <token>'
```

## Policy Rules for Tool Calls

The backend policy engine considers:

- tool `risk`: `low`, `medium`, `high`, `critical`;
- tool `readOnly`;
- tool `requiresApproval`;
- request `role`: `viewer`, `operator`, `admin`;
- backend environment: `development`, `staging`, `production`;
- request `approved` boolean.

### Policy Engine Rules

For policy evaluation:

- `critical` risk is denied by default.
- `viewer` can execute tools with `readOnly=true`.
- `viewer` cannot execute write tools.
- `operator` can execute tools with `readOnly=true`.
- `operator` can execute `medium` risk write tools in development/staging.
- `operator` can execute `medium` risk write tools in production only when `approved=true`.
- production write operations without approval return a policy decision with `requiresApproval=true`.
- `admin` is allowed by policy, except critical risk is still denied first by the current engine.
- unknown roles are denied.

Important: `readOnly` is a primary gate. Do not assume `risk=low` alone means a tool is safe; also inspect `readOnly` and `requiresApproval`.

### Execution Approval Overlay

After the policy decision allows the request, the registry applies an additional approval gate:

```text
requiresExecutionApproval(tool) == true when:
- tool.requiresApproval == true, OR
- tool.risk is medium, high, or critical
```

If this gate is true and the request has `approved=false`, the response is `202 pending_approval`.

This means some calls may pass policy but still require execution approval due to risk or explicit tool configuration.

## MCP Flow

MCP endpoint:

```text
POST /mcp
```

### Initialize

```json
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}
```

### List Tools

```bash
curl -s http://192.168.20.166:8080/mcp \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'
```

The returned MCP tool schema includes:

- the tool-specific input parameters;
- common fields: `actor`, `role`, `target`, `approved`.

### Call Tool

```bash
curl -s http://192.168.20.166:8080/mcp \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{
    "jsonrpc": "2.0",
    "id": "call-1",
    "method": "tools/call",
    "params": {
      "name": "linux.disk_usage",
      "arguments": {
        "actor": "external-agent-01",
        "role": "viewer",
        "target": "host=192.168.20.166",
        "path": "/"
      }
    }
  }'
```

MCP `tools/call` returns HTTP 200 at the JSON-RPC transport layer. Inspect the MCP result:

```json
{
  "jsonrpc": "2.0",
  "id": "call-1",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{...}"
      }
    ],
    "structuredContent": {
      "httpStatus": 202,
      "result": {
        "executionId": "exe-...",
        "approvalId": "apr-...",
        "status": "pending_approval",
        "message": "pending approval"
      }
    },
    "isError": false
  }
}
```

For MCP, treat `structuredContent.httpStatus` as the effective REST status. If `isError=true` or `httpStatus >= 400`, handle it like the REST error table.

### MCP Argument Flattening

`tools/call` accepts two styles:

Nested parameters:

```json
{
  "actor": "external-agent-01",
  "role": "viewer",
  "target": "host=192.168.20.166",
  "parameters": { "path": "/" }
}
```

Flat parameters:

```json
{
  "actor": "external-agent-01",
  "role": "viewer",
  "target": "host=192.168.20.166",
  "path": "/"
}
```

The server treats all keys except `actor`, `role`, `target`, and `approved` as tool parameters when no nested `parameters` object is present.

## Stdio MCP Clients

Clients that only support stdio MCP should use the project proxy:

```bash
go build -o bin/darwin-ops-mcp-proxy ./backend/cmd/mcp-proxy
```

Example Claude Desktop-style configuration:

```json
{
  "mcpServers": {
    "darwin_ops": {
      "command": "/absolute/path/to/darwin-ops-mcp-proxy",
      "args": ["-url", "http://192.168.20.166:8080/mcp"],
      "env": {
        "DARWIN_OPS_MCP_API_TOKEN": "<token>"
      }
    }
  }
}
```

## Retry and Polling Rules

- `200 completed`: do not retry; consume result.
- `202 pending_approval`: do not immediately re-submit; poll approvals/execution or wait for admin.
- `400 validation_failed`: fix parameters; do not retry unchanged.
- `403 policy denied`: do not retry unchanged; request access or change role/tool.
- `404 tool not found`: refresh tool list; if still absent, submit application.
- `500 handler error`: retry only if idempotent and safe.

Recommended polling:

```text
interval: 15-30 seconds
max wait: agent/user configured, commonly 10 minutes
```

Prefer polling `GET /api/v1/executions/:executionId` after an approval because approval may automatically replay the pending execution.

## Audit and Compliance

Every meaningful event produces records:

- execution history: `GET /api/v1/executions`
- single execution: `GET /api/v1/executions/:id`
- audit log: `GET /api/v1/audit`

Agents should preserve:

- `executionId`
- `approvalId` if present
- `auditId` if present
- `X-Trace-ID`
- actor identity
- target description

Sensitive parameter keys are masked in audit logs when they contain password/token/secret/api key/authorization/credential style names.

## Common Pitfalls

1. **Confusing application approval with execution approval.** Applications request access; executions run tools.
2. **Blindly setting `approved=true`.** Only use it when a valid approval process has happened.
3. **Ignoring `202 pending_approval`.** This is not a failure; it means wait for approval and then inspect execution status.
4. **Treating MCP transport HTTP 200 as success.** Always inspect `structuredContent.httpStatus` and `isError`.
5. **Using a missing tool name.** Refresh `GET /api/v1/tools`; submit `/applications` if the tool is absent.
6. **Omitting actor/target.** The call may work, but audit records become useless.
7. **Retrying denied requests.** A policy denial will continue to deny until role/tool/risk/approval changes.
8. **Assuming docs examples are live credentials.** Never put tokens, SSH keys, or production secrets in request examples or chat logs.

## Minimal Agent Algorithm

```python
def call_tool(tool_name, args, actor, role="viewer", target=""):
    tools = get_json("/api/v1/tools")
    if tool_name not in [t["name"] for t in tools]:
        submit_application(tool_name, risk="medium", role=role, reason="Tool required by current task")
        return {"status": "application_submitted"}

    resp = post_json(f"/api/v1/tools/{tool_name}/execute", {
        "actor": actor,
        "role": role,
        "target": target,
        "approved": False,
        "parameters": args,
    })

    if resp.http_status == 200:
        return resp.json["data"]
    if resp.http_status == 202:
        return wait_for_execution_after_approval(resp.json["executionId"], resp.json["approvalId"])
    if resp.http_status == 400:
        raise ValueError("Fix parameters: " + resp.json.get("error", "validation failed"))
    if resp.http_status == 403:
        raise PermissionError("Policy denied: " + resp.json.get("error", "denied"))
    if resp.http_status == 404:
        submit_application(tool_name, risk="medium", role=role, reason="Tool not found")
        return {"status": "application_submitted"}
    raise RuntimeError(resp.json.get("error", "tool execution failed"))
```

## Verification Checklist for Integrators

- [ ] `GET /healthz` works without token.
- [ ] `GET /api/v1/tools` works with `Authorization: Bearer <token>` when token is configured.
- [ ] `POST /api/v1/applications` creates `approved` for low/medium and `pending` for high/critical.
- [ ] A safe read-only tool call returns `200 completed`.
- [ ] A medium/high/requiresApproval tool call returns `202 pending_approval` when `approved=false`.
- [ ] Admin approval changes the approval state and updates the pending execution.
- [ ] `GET /api/v1/executions/:id` shows final status after approval.
- [ ] MCP `tools/list` returns tool schemas.
- [ ] MCP `tools/call` returns `structuredContent.httpStatus` and the agent handles it.
- [ ] Audit log contains useful `actor`, `target`, `executionId`, and trace ID.
