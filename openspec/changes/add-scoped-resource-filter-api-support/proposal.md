# Change: 补齐 scoped 资源过滤 API 契约并增加前置校验

## Why

当前 server 侧存在两个契约缺口：

1. `AppBackup` API 只暴露了 `includedNamespaceScopedResources` 与 `excludedNamespaceScopedResources`，缺少 `includedClusterScopedResources` 与 `excludedClusterScopedResources`。
2. `DisasterInstance.spec.restorePolicy.resourceSelection` 需要跟随 operator 新增 scoped 四字段，当前缺少明确的 server 侧优先级策略。

这会导致前端与脚本无法完整表达 scoped 资源过滤意图，并且在恢复侧字段并存时存在语义不确定性。

## What Changes

### 1. AppBackup API 补齐 scoped 四字段

在以下结构补齐字段并实现完整透传：

- `CreateAppBackupRequest`
- `UpdateAppBackupRequest`
- `AppBackupSpecDTO`
- `ConvertSpecToDTO` / `ToCRD` / `MergeToCRD`

新增字段：

- `includedClusterScopedResources`
- `excludedClusterScopedResources`

### 2. DisasterInstance API 对齐 scoped 四字段

实例创建、更新、查询接口在 `restorePolicy.resourceSelection` 中支持并回显以下字段：

- `includedNamespaceScopedResources`
- `excludedNamespaceScopedResources`
- `includedClusterScopedResources`
- `excludedClusterScopedResources`

并采用恢复侧优先级语义：

- 当 `includeClusterResources=true` 时，server/operator 在恢复侧忽略 scoped 四字段。

### 3. 提交期校验策略

1. AppBackup 入口保持 fail-fast：
   - include/exclude 交集冲突返回 400
   - `exclude=["*"]` 且 include 非空返回 400
2. DisasterInstance 入口采用优先级策略：
   - 不因 old/scoped 并存直接拒绝
   - 当 `includeClusterResources=true` 时忽略 scoped 四字段冲突校验
   - 当进入 scoped 路径时，仍执行 scoped 组合合法性校验

### 4. 更新 API 文档与 Apipost

同步更新接口文档与 Apipost 示例，确保前端与测试平台字段一致。

## Non-Goals

- 不改动 operator 的执行逻辑。
- 不变更现有路由路径。
- 不改动 `modifierRulesText` 解析规则。

## Dependencies

- 依赖 `disaster-operator` 变更 `add-scoped-resource-selection-filters`。

## Impact

- Affected specs:
  - `app-backup`
  - `api-standards`
- Affected code:
  - `internal/apis/app_backup/v1/types.go`
  - `internal/apis/app_backup/v1/handler.go`
  - `internal/apis/disaster_instance/v1/types.go`
  - `internal/apis/disaster_instance/v1/handler.go`
  - `internal/apis/app_backup/v1/*_test.go`
  - `internal/apis/disaster_instance/v1/*_test.go`
