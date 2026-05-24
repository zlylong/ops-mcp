---
name: darwin-ops-mcp-project-agent
description: "Use when acting as the dedicated project agent for darwin-ops-mcp / ops-mcp. Enforces the 192.168.20.166 development host, /root/ops-mcp workspace, source-grounded changes, tests, docs, commits, and pushes."
version: 1.0.0
author: Hermes Agent
license: MIT
metadata:
  hermes:
    tags: [darwin-ops-mcp, ops-mcp, project-agent, mcp, api, approval-workflow]
---

# darwin-ops-mcp Project Agent

This skill defines the operating procedure for the dedicated Hermes profile `darwinopsmcp`.
Use it whenever the user asks to work on `darwin-ops-mcp`, `ops-mcp`, the approval-gated MCP/API control plane, or related docs/skills.

## Non-negotiable project context

- Dedicated Hermes profile: `darwinopsmcp`.
- Dedicated launcher: `darwinopsmcp-agent`.
- Model lock: use OpenAI Codex `gpt-5.5` only; do not silently switch to local/custom/fallback models for this project agent.
- Real development host: `192.168.20.166`.
- Real project workspace: `/root/ops-mcp`.
- Real GitHub repository: `git@github.com:zlylong/darwin-ops-mcp.git` / `https://github.com/zlylong/darwin-ops-mcp`.
- Do not treat local `/tmp/darwin-ops-mcp` or local `/root/ops-mcp` as the source of truth.
- Do not expose tokens, passwords, SSH private keys, API tokens, or credential material. If content must be mentioned, replace with `[REDACTED]`.

## Start-of-task checklist

1. Confirm the active terminal context is the project host/workspace:
   ```bash
   hostname && pwd && git remote -v && git status --short
   ```
   Expected host is `mcpdev`; expected cwd is `/root/ops-mcp`.
2. If the terminal is not already on the project host, use SSH explicitly:
   ```bash
   ssh root@192.168.20.166 'cd /root/ops-mcp && <command>'
   ```
3. Before editing, inspect the relevant source/docs instead of relying on memory.
4. Prefer small, verified slices. The user prefers uninterrupted `[longlongago] 继续` style progress and fewer check-ins.

## Source-grounding rules

- For API behavior, read the backend implementation before documenting or changing behavior:
  - `backend/internal/api/router.go`
  - `backend/internal/api/api.go`
  - `backend/internal/api/types.go`
  - `backend/internal/api/mcp.go`
  - `backend/internal/app/registry.go`
  - `backend/internal/policy/engine.go`
  - `backend/internal/domain/types.go`
- For docs, keep Chinese docs and AI-readable skills aligned:
  - `docs/API.zh-CN.md`
  - `docs/MCP_CLIENT.zh-CN.md`
  - `docs/APPROVAL_ARCHITECTURE.zh-CN.md`
  - `docs/AI_AGENT_SKILL.zh-CN.md`
  - `.hermes/skills/*/SKILL.md`

## Third-party AI Agent integration work

When the task involves third-party AI Agent access to tools, load/use the repo skill:

- `.hermes/skills/darwin-ops-mcp-third-party-ai-agent/SKILL.md`

Updated key semantics (as of 2026-05-23):

- New user management APIs: `POST /api/v1/users/login`, `GET /api/v1/users/me`, `PUT /api/v1/users/me`, `PUT /api/v1/users/me/password`, `GET /api/v1/users`, `POST /api/v1/users`, `GET/PUT/DELETE /api/v1/users/:id`, `PUT /api/v1/users/:id/password`.
- User roles: `admin`, `operator`, `viewer`.
- User token format: `Bearer user:<userID>` (not JWT, a simple token).
- Default admin seeded on first start: username `admin`; production password must come from `DARWIN_OPS_MCP_BOOTSTRAP_ADMIN_PASSWORD` (local mock/dev may use `admin1234` for demos).
- Passwords stored as bcrypt hashes (never plaintext).
- Master token auth falls back to first admin user for user-specific endpoints.
- Frontend pages: `/login` (login form), `/profile` (view/edit self, change password), `/users` (admin user CRUD).

## Original key semantics to preserve unless source code changes:

- Protected REST APIs require the configured API token; never print the token.
- Trace header: `X-Trace-ID`.
- REST tool execution request fields: `actor`, `role`, `target`, `approved`, `parameters`.
- MCP endpoint: `GET /mcp`, `POST /mcp`.
- MCP protocol version: `2024-11-05`.
- MCP methods: `initialize`, `notifications/initialized`, `ping`, `tools/list`, `tools/call`.
- Tool execution statuses include `completed`, `pending_approval`, `denied`, `error`.
- Approval of a pending execution currently auto-replays the original pending task via backend logic.
- MCP `tools/call` may return HTTP 200 while the embedded `structuredContent.httpStatus` indicates pending/error; agents must inspect structured content.

## Change workflow

1. Inspect current state:
   ```bash
   git status --short
   git branch --show-current
   git log --oneline -3
   ```
2. Make focused edits.
3. If code or behavior changes, update relevant docs and AI-readable skill files in the same slice.
4. Run formatting/tests appropriate to the change. Common backend verification:
   ```bash
   go test ./backend/internal/... ./backend/cmd/...
   ```
5. For frontmatter in skills, verify YAML parseability and required fields (`name`, `description`).
6. Commit and push automatically after successful verification:
   ```bash
   git add <changed-files>
   git commit -m "<type>: <concise message>"
   git push origin main
   ```
7. Final response should include files changed, verification result, commit hash, and push status.

## Documentation standards

- Write docs in Chinese unless the user asks otherwise.
- Keep AI-readable docs procedural and explicit: endpoints, fields, status handling, retry behavior, audit fields, and safety rules.
- Do not invent behavior. If docs disagree with code, prefer code and update docs.
- Avoid vague statements like “may support”; cite the exact endpoint/status/field from implementation.

## Safety and deployment boundaries

- Never run destructive commands unless the user explicitly requested that scope.
- Never force-push unless explicitly requested and justified.
- Never commit generated secrets, local DBs, logs, node_modules, build caches, or private keys.
- Do not change production-like service state just to inspect code. Prefer read-only commands unless the task requires deployment/verification.

## Useful commands

```bash
# Project identity
hostname && pwd && git remote -v && git status --short

# Backend tests
go test ./backend/internal/... ./backend/cmd/...

# Skill frontmatter validation
python3 - <<'PY'
from pathlib import Path
import yaml
for p in Path('.hermes/skills').glob('*/SKILL.md'):
    text = p.read_text(encoding='utf-8')
    assert text.startswith('---\n'), p
    _, fm, body = text.split('---', 2)
    data = yaml.safe_load(fm)
    assert data.get('name') and data.get('description'), p
    assert body.strip(), p
print('OK')
PY
```


## JumpServer Multi-Instance Integration

- The project supports registering multiple JumpServer servers via `/api/v1/jumpservers` and the frontend `/jumpservers` page.
- Admin-only management actions: list, create, read, update, delete, and connectivity test (`POST /api/v1/jumpservers/{id}/test`).
- Supported auth modes align with JumpServer v2 REST API concepts: `token`, `private_token`, `access_key`, and `session`.
- Secret fields (`credential`, `accessKeySecret`, session material) are write-only. API responses must only expose `hasCredential`; never log or document real credentials.
- Current storage is in-memory. Treat it as a development slice; productionization requires database persistence and encrypted credential storage.
- For frontend work, update both `frontend/src/App.tsx` and the real Vite entry `frontend/src/main.tsx`, then deploy to `192.168.20.166` for verification.
