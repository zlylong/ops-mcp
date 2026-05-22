# darwin-ops-mcp 项目 Agent

本项目已配置一个专用 Hermes 项目 Agent，用于长期处理 `darwin-ops-mcp` / `ops-mcp` 的后端、MCP/API、审批工作流、文档与 AI-readable Skill 维护。

## 本机启动方式

在已安装 Hermes 的机器上使用：

```bash
darwinopsmcp-agent
```

该启动器等价于：

```bash
hermes -p darwinopsmcp   -s darwin-ops-mcp-project-agent,darwin-ops-mcp-third-party-ai-agent
```

## Agent 固定上下文

- Hermes profile：`darwinopsmcp`
- 专用启动器：`darwinopsmcp-agent`
- 模型锁定：OpenAI Codex `gpt-5.5`，不使用本地/custom/fallback 模型
- 开发机：`192.168.20.166`
- 工作目录：`/root/ops-mcp`
- Git remote：`git@github.com:zlylong/darwin-ops-mcp.git`
- 默认加载 Skill：
  - `.hermes/skills/darwin-ops-mcp-project-agent/SKILL.md`
  - `.hermes/skills/darwin-ops-mcp-third-party-ai-agent/SKILL.md`

## Agent 职责

- 优先在 166 开发机真实工作区内执行代码、文档、测试与 Git 操作。
- 修改 API/MCP/审批语义时同步更新中文文档和 AI-readable Skill。
- 面向第三方 AI Agent 的接入场景，保持 REST/MCP endpoint、字段、状态码、审批语义与源码一致。
- 变更完成后执行必要验证，并自动 `git add`、`git commit`、`git push`。
- 不输出、不提交任何 token、密码、SSH 私钥或其他凭据。

## 验证命令

```bash
# 确认项目上下文
hostname && pwd && git remote -v && git status --short

# 后端测试
go test ./backend/internal/... ./backend/cmd/...

# Skill frontmatter 校验
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
