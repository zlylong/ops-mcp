# darwin-ops-mcp 测试报告

生成时间：2026-05-25T06:32:12Z

## 摘要

本轮在 `192.168.20.166:/root/ops-mcp` 继续补充后端测试脚本，重点提升无外部依赖、稳定可重复的单元测试覆盖：

- `internal/storage`：补齐 `UserStore` 与 `JumpServerStore` 的 CRUD、not found 分支、凭据脱敏状态与状态更新时间测试。
- `internal/api`：补齐 JumpServer SSRF/URL 校验纯函数测试，包括 blocklist、端口限制、URL 归一化、请求默认值与 redirect 防护。
- `internal/app`：补齐 Agent API Key 生命周期、校验失败、撤销、鉴权、应用审批/驳回、store accessor 测试。

所有新增测试均为本地内存/纯函数测试，不依赖真实外部服务，不保存真实 API key、token、password、secret 或 credential。

## 执行环境

- 主机：`192.168.20.166`
- 仓库：`/root/ops-mcp`
- 后端目录：`/root/ops-mcp/backend`
- 前端目录：`/root/ops-mcp/frontend`
- 分支：`main`

## 本轮新增/修改测试文件

- `backend/internal/storage/memory_test.go`
  - 新增 `TestUserStore_CRUDAndPasswordHash`
  - 新增 `TestUserStore_NotFoundBranches`
  - 新增 `TestJumpServerStore_CRUDSanitizesCredentials`
  - 新增 `TestJumpServerStore_NotFoundBranchesAndCredentialFlag`
- `backend/internal/api/jumpservers_helpers_test.go`
  - 新增 JumpServer SSRF、端口、URL、请求校验与 redirect 防护测试。
- `backend/internal/app/registry_test.go`
  - 新增 `TestRegistry_AgentAPIKeyLifecycle`
  - 新增 `TestRegistry_AgentAPIKeyValidationAndExpiry`
  - 新增 `TestRegistry_ApplicationsAndRejectApplication`
  - 新增 `TestRegistry_AccessorStores`

## 测试命令与结果

### 后端 Go 全量测试

```bash
cd /root/ops-mcp/backend
go test ./...
```

结果：通过。

关键输出：

```text
ok  github.com/zlylong/darwin-ops-mcp/backend/internal/api      3.772s
ok  github.com/zlylong/darwin-ops-mcp/backend/internal/app      0.005s
ok  github.com/zlylong/darwin-ops-mcp/backend/internal/storage  0.006s
```

### 前端 TypeScript 校验

```bash
cd /root/ops-mcp/frontend
npm run lint
```

结果：通过。

### 前端生产构建

```bash
cd /root/ops-mcp/frontend
npm run build
```

结果：通过。

备注：Vite 仍提示单个 chunk 超过 500 kB，这是构建体积警告，不影响测试/构建成功。

## 覆盖率结果

按重点包单独采集覆盖率：

```bash
cd /root/ops-mcp/backend
for p in ./cmd/mcp-proxy ./internal/adapters/linux ./internal/adapters/remote ./internal/api ./internal/app ./internal/audit ./internal/config ./internal/domain ./internal/policy ./internal/storage; do
  cov=/tmp/cov-$(echo $p | tr "/." "__").out
  go test $p -coverprofile=$cov -covermode=count
  go tool cover -func=$cov | tail -1
done
```

覆盖率汇总：

| 包 | 覆盖率 |
| --- | ---: |
| `./cmd/mcp-proxy` | 74.1% |
| `./internal/adapters/linux` | 45.8% |
| `./internal/adapters/remote` | 39.1% |
| `./internal/api` | 59.8% |
| `./internal/app` | 82.5% |
| `./internal/audit` | 87.8% |
| `./internal/config` | 71.4% |
| `./internal/domain` | 95.0% |
| `./internal/policy` | 88.9% |
| `./internal/storage` | 93.7% |

## 覆盖率提升

本轮重点包提升：

| 包 | 本轮前 | 本轮后 | 增量 |
| --- | ---: | ---: | ---: |
| `internal/storage` | 29.7% | 93.7% | +64.0 pct |
| `internal/app` | 57.5% | 82.5% | +25.0 pct |
| `internal/api` | 49.9% | 59.8% | +9.9 pct |

## 仍可继续提升的区域

当前低覆盖区域主要集中在：

- `internal/adapters/remote`：39.1%
  - 可继续通过 mock SSH runner 或抽象 command executor 覆盖 option injection、超时、错误输出解析分支。
- `internal/adapters/linux`：45.8%
  - 可继续通过可注入命令执行器/临时 proc 文件覆盖 `DiskUsage`、`NetworkInterfaces`、`ServiceStatus`、`JournalTail`、`Ping`、`DNSLookup`。
- `internal/api`：59.8%
  - 剩余 0% 函数包括工具 CRUD、审批路由、dashboard、audit route、JumpServer HTTP CRUD handler 等；需要更多 HTTP handler 测试。
- `cmd/server`、`internal/adapters/kubernetes`、`internal/adapters/prometheus`、`internal/api/docs`
  - 当前无测试文件或多为启动/文档胶水代码，覆盖率可按收益优先级后置。

## 结论

本轮新增测试后，核心业务包覆盖率显著提升：

- `storage` 已提升到 93.7%，内存存储层的主要 CRUD 和异常分支已覆盖。
- `app` 已提升到 82.5%，Agent API Key、应用审批、store accessor 已有回归保障。
- `api` 已提升到 59.8%，JumpServer SSRF/URL 安全校验已补上纯函数级回归测试。

全量后端测试、前端 TypeScript 校验和前端生产构建均已通过。下一轮建议优先补 `internal/api` handler 级测试和 adapter 层可注入 executor 测试。
