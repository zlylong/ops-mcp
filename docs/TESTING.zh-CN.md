# 测试指南

本文档介绍 ops-mcp 后端的测试策略和实现细节。

## 概述

后端使用 Go 内置测试框架配合 `testify` 断言，提供清晰易读的测试代码。

## 测试组织

```
backend/internal/
├── domain/      # 领域实体测试 (types_test.go)
├── policy/      # 策略引擎测试 (engine_test.go)
├── storage/     # 内存存储测试 (store_test.go)
├── app/         # 应用服务测试 (registry_test.go)
├── api/         # HTTP 处理器测试 (router_test.go)
├── audit/       # 审计日志测试 (audit_test.go)
└── adapters/    # 集成 Mock（无单元测试）
```

## 运行测试

```bash
# 运行所有后端测试
go test -v ./internal/...

# 运行特定包的测试
go test -v ./internal/api
go test -v ./internal/policy

# 运行并显示覆盖率
go test -cover ./internal/...

# 运行匹配模式的测试
go test -v -run TestEngine ./internal/policy
```

## 测试分类

### 领域测试 (domain/types_test.go)

测试领域实体和枚举：
- TestRiskLevel_Values - 验证风险等级常量
- TestEnvironment_Values - 验证环境常量
- TestRole_Values - 验证角色常量
- TestTool_Struct - Tool 实体结构
- TestExecuteRequest_Struct - ExecuteRequest 结构
- TestPolicyDecision_Struct - PolicyDecision 结构
- TestAuditRecord_Struct - AuditRecord 结构
- TestExecution_Struct - Execution 结构
- TestApproval_Struct - Approval 结构
- TestApproval_Decision - Approval 决策逻辑

### 策略测试 (policy/engine_test.go)

测试策略引擎决策矩阵：
- TestEngine_Evaluate_CriticalTool - 关键工具始终被拒绝
- TestEngine_Evaluate_ProductionWriteWithoutApproval - 生产环境写操作需要审批
- TestEngine_Evaluate_ViewerReadOnlyAllowed - Viewer 可以运行只读工具
- TestEngine_Evaluate_ViewerWriteDenied - Viewer 不能运行写工具
- TestEngine_Evaluate_OperatorMediumRiskDev - Operator 可在开发环境运行中等风险工具
- TestEngine_Evaluate_OperatorMediumRiskProductionAllowed - Operator 在生产环境获批后可运行中等风险工具
- TestEngine_Evaluate_AdminAllowed - Admin 拥有完全访问权限
- TestEngine_Evaluate_UnknownRole - 未知角色被拒绝
- TestEngine_Evaluate_RiskHighDevDenied - 高风险开发操作被拒绝

### 存储测试 (storage/store_test.go)

测试内存存储：
- TestExecutionStore_Add - 添加和列出执行记录
- TestExecutionStore_Get - 根据 ID 获取执行记录
- TestApprovalStore_Add - 添加和列出审批记录
- TestApprovalStore_Decide - 批准/拒绝工作流

### 应用测试 (app/registry_test.go)

测试注册表服务：
- TestRegistry_Register - 工具注册和重复检查
- TestRegistry_List - 按名称排序列出工具
- TestRegistry_Get - 根据名称获取工具
- TestRegistry_Execute_Completed - 成功执行
- TestRegistry_Execute_Denied - 工具不存在
- TestRegistry_Execute_PendingApproval - 触发审批工作流
- TestRegistry_Approvals - 列出审批
- TestRegistry_Approve - 批准工作流
- TestRegistry_Reject - 拒绝工作流

### API 测试 (api/router_test.go)

测试 HTTP 处理器：
- TestNewRouter_CORS - 初始化带 CORS 的路由器
- TestHealthz - 健康检查端点
- TestDashboardSummary - 仪表盘摘要端点
- TestToolsList - 列出所有工具
- TestToolDetail - 获取工具详情
- TestToolDetail_NotFound - 未找到工具返回 404
- TestExecuteTool - 成功执行工具
- TestExecuteTool_ValidationFailure - 处理验证失败
- TestApprovalsList - 列出待审批
- TestApproveApproval - 批准操作
- TestRejectApproval - 拒绝操作
- TestAuditRecords - 列出审计记录

### 审计测试 (audit/audit_test.go)

测试审计日志：
- TestStore_Record - 记录审计事件
- TestStore_Record_AutoID - 自动生成审计 ID
- TestStore_Record_AutoTimestamp - 自动生成时间戳
- TestStore_List - 列出审计记录
- TestMask_SensitiveKeys - 敏感数据遮蔽
- TestMask_CaseInsensitive - 不区分大小写遮蔽
- TestMask_NilInput - 处理 nil 输入
- TestMask_EmptyInput - 处理空输入
- TestIsSensitiveKey - 敏感键检测

## 策略逻辑测试

策略引擎测试覆盖以下决策矩阵：

| 角色 | 风险 | 环境 | 已批准 | 结果 |
|------|------|------|--------|------|
| Viewer | Low | 任意 | - | 允许（只读） |
| Viewer | Medium/Critical | 任意 | - | 拒绝 |
| Operator | Low | 任意 | - | 允许 |
| Operator | Medium | Dev/Staging | - | 允许 |
| Operator | Medium | Production | 是 | 允许 |
| Operator | Medium | Production | 否 | 拒绝（需要审批） |
| Operator | High | 任意 | - | 拒绝 |
| Admin | 任意 | 任意 | - | 允许 |
| Unknown | 任意 | 任意 | - | 拒绝 |

## Mock 依赖

使用 mockRecorder 测试依赖审计记录的服务：

```go
type mockRecorder struct{}

func (m *mockRecorder) Record(record domain.AuditRecord) domain.AuditRecord {
    record.ID = "aud-mock-123"
    return record
}
func (m *mockRecorder) List() []domain.AuditRecord { return nil }
```

## 覆盖率要求

目标覆盖率：**80%+** 所有后端包。

```bash
go test -cover ./internal/...
```

## 持续集成

测试自动运行于：
- PR 创建
- 推送到 main 分支

查看 .github/workflows/test.yml 了解 CI 配置。
