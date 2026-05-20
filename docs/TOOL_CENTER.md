# Tool Center User Guide (Beginner Edition)

> This guide assumes the reader has never used ops-mcp before. You do not need to know Go, React, Kubernetes, Prometheus, or MCP to follow it.

## 1. What is the Tool Center?

The Tool Center is the main operations toolbox in ops-mcp.

- Each row is an executable tool, such as checking disk usage, memory usage, Kubernetes pods, or Prometheus metrics.
- Every tool has a description, category, risk level, read-only flag, approval requirement, and input schema.
- When you execute a tool, the system checks policy first. It may:
  - run the tool immediately;
  - create an approval request;
  - deny the request.
- Every operation is recorded in execution history and audit logs.

The default mock mode is safe: it returns simulated data, does not connect to real production infrastructure, and does not mutate real machines or clusters.

## 2. How to open the Tool Center

1. Open your browser.
2. Open the frontend URL:

```text
http://localhost:5173
```

For a deployed server, use that server address, for example:

```text
http://192.168.20.166:5173
```

3. Use the left sidebar.
4. Click **Tool Center**.

## 3. What do the table columns mean?

- **Tool**: the unique tool name, for example `linux.disk_usage`.
- **Category**: the group the tool belongs to, such as `linux`, `kubernetes`, `prometheus`, or `custom`.
- **Risk**:
  - `low`: usually safe read-only query;
  - `medium`: may expose sensitive information or have wider impact;
  - `high`: high-risk operation;
  - `critical`: denied by default.
- **Type**:
  - read-only: expected to only inspect information;
  - write/change: may modify system state.
- **Description**: a short explanation of the tool.
- **Actions**:
  - Details: inspect metadata and input schema;
  - Edit: change tool metadata;
  - Execute: run the tool;
  - Delete: remove the tool.

## 4. Default tools

### Linux tools

- `linux.system_info`: host, kernel, distribution, architecture, uptime, and virtualization.
- `linux.load_average`: 1/5/15 minute load averages and CPU cores.
- `linux.memory_usage`: memory and swap usage.
- `linux.disk_usage`: disk usage for a path.
- `linux.process_list`: process list with CPU and memory usage.
- `linux.network_interfaces`: interface state, IP addresses, and traffic counters.
- `linux.service_status`: systemd service status.
- `linux.journal_tail`: recent journal logs; medium risk and requires approval.
- `linux.ping`: connectivity check.
- `linux.dns_lookup`: DNS lookup.

### Kubernetes tools

- `k8s.list_pods`
- `k8s.get_pod_logs`
- `k8s.list_events`
- `k8s.get_deployment_status`

### Prometheus tools

- `prometheus.query`
- `prometheus.service_error_rate`
- `prometheus.service_latency_p95`
- `prometheus.pod_cpu_usage`
- `prometheus.pod_memory_usage`

## 5. Search and filters

Use the search box to match tool name, description, or category. Examples:

- `linux`: show Linux tools.
- `disk`: find disk tools.
- `pod`: find Kubernetes pod tools.
- `dns`: find DNS tools.

You can also filter by category, risk, and read-only/change type.

## 6. Tool details and input schema

Click **Details** to open the side drawer. It shows the name, risk, read-only flag, description, category, approval requirement, and input schema.

The input schema tells you what parameters to provide. Example:

```json
{
  "path": "string?"
}
```

This means `path` is an optional string. Common markers:

- `string`: required text.
- `string?`: optional text.
- `number`: required number.
- `number?`: optional number.

## 7. Execute a tool

Example: `linux.disk_usage`.

1. Find `linux.disk_usage`.
2. Click **Execute**.
3. Fill in the execution form:
   - Actor: who runs the tool.
   - Role: `viewer`, `operator`, or `admin`.
   - Target: the target being operated on, such as `host=demo`.
   - Authorization: whether this request is already approved.
   - Parameters JSON: the tool input.
4. Use this parameter JSON:

```json
{
  "path": "/var"
}
```

5. Click **Execute**.
6. A successful run returns `completed` and result data.

## 8. Why does a tool enter approval?

A tool enters approval if it has medium/high risk or `requiresApproval=true`.

For example, `linux.journal_tail` is medium risk and requires approval. Its first execution usually returns:

```json
{
  "status": "pending_approval",
  "message": "pending approval"
}
```

Then open the Approval Center to approve or reject it.

## 9. Execution history and audit logs

- **Execution Center** shows each run and its status: `completed`, `pending_approval`, `denied`, `validation_failed`, or `error`.
- **Audit Center** shows who did what, when, against which target, and whether the system allowed it.

Sensitive parameter names such as `password`, `secret`, `token`, `api_key`, and `authorization` are masked before audit records are stored.

## 10. Create a custom tool

1. Click **Add Tool**.
2. Enter a unique tool name such as `custom.echo`.
3. Fill in category, description, risk, read-only flag, approval flag, and input schema.
4. Save the tool.

Example input schema:

```json
{
  "message": "string"
}
```

Current custom tools use a default mock handler. They still go through policy, approval, execution, and audit flow, but do not call a real external system.

## 11. Edit or delete tools

- Editing can change description, category, risk, read-only flag, approval flag, and input schema.
- Tool names cannot be changed. Delete and recreate the tool if you need a new name.
- Deleting removes the tool from the current registry. Historical execution and audit records may remain.
- Default mock tools may reappear after service restart because they are registered during startup.

## 12. Execution logic

When you click Execute, the backend:

1. Finds the tool definition.
2. Reads actor, role, target, authorization, and parameters.
3. Evaluates policy.
4. If denied, records a denied execution and audit event.
5. If allowed but approval is required, records a pending execution and creates an approval request.
6. If allowed directly, calls the tool handler, stores the result, records execution history, and writes an audit event.

## 13. Roles and risk

- `viewer`: can usually run read-only tools.
- `operator`: can run more operational tools.
- `admin`: broadest permissions.

Important rules:

- `critical` tools are denied by default.
- `requiresApproval=true` forces approval even for low-risk tools.
- Medium/high risk tools enter approval by default.

## 14. Recommended beginner path

1. Open Tool Center.
2. Search for `linux`.
3. Open details for `linux.system_info`.
4. Execute `linux.disk_usage` with:

```json
{
  "path": "/"
}
```

5. Open Execution Center to inspect the run.
6. Open Audit Center to inspect the audit record.
7. Execute `linux.journal_tail` to see the approval flow.
8. Open Approval Center and approve or reject it.

## 15. Common problems

### JSON error

Make sure you use valid JSON: double quotes, surrounding `{}`, and no trailing commas.

### Low-risk tool still requires approval

The tool may have `requiresApproval=true`.

### Request is denied

Common reasons: insufficient role, viewer trying to run a write tool, critical risk, or production change without approval.

### Deleted tool reappears

Default mock tools are registered at backend startup, so they may reappear after restart.

### Do Linux tools operate the real server?

Not in default mock mode. They return simulated data. Real server integration requires a real adapter implementation and additional safety controls.
