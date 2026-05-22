# AI Agent Skills

This directory contains AI-readable skills for `darwin-ops-mcp`.


## Dedicated Project Agent Skill

- `darwin-ops-mcp-project-agent/SKILL.md`

Use this skill when a Hermes Agent is acting as the dedicated project agent for this repository. It fixes the development host/workspace, source-grounding rules, verification commands, documentation expectations, and commit/push workflow.

## Primary Skill for Third-Party AI Agents

- `darwin-ops-mcp-third-party-ai-agent/SKILL.md`

Use this skill when an external AI Agent needs to learn:

- how to request tool access through `/api/v1/applications`;
- how to list and call tools through REST `/api/v1/tools/:name/execute`;
- how to call tools through MCP `/mcp` `tools/list` and `tools/call`;
- how to handle `200`, `202 pending_approval`, `400`, `403`, `404`, `500`;
- how application approval differs from execution approval;
- how policy rules, `readOnly`, `risk`, `requiresApproval`, `role`, and `approved` interact;
- what audit fields to preserve.

## Recommended Loading

Third-party AI systems should load the full SKILL.md before attempting tool calls:

```text
.hermes/skills/darwin-ops-mcp-project-agent/SKILL.md
.hermes/skills/darwin-ops-mcp-third-party-ai-agent/SKILL.md
```
