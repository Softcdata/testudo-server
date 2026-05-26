# Change: 为容灾组添加操作执行状态的 WebSocket 事件流接口

## Why

前端触发容灾组操作（如 failover / reprotect / pause / resume 等）后，通过 POST `/groups/:name/actions` 创建 `DisasterOperation` 资源，但操作是异步执行的，前端无法感知操作何时完成、当前执行到哪个步骤、是否失败。需要提供与其他模块（DisasterDrill、DisasterJob 等）一致的 WebSocket 事件流接口，供前端实时订阅操作进度。

## What Changes

- 在容灾组 Handler 下新增两个 WebSocket 路由：
  - `GET /watch/groups/operations` — 监听该容灾组**所有**操作的状态变化（可按组名过滤）
  - `GET /watch/groups/operations/:operationName` — 监听**指定**操作的状态变化
- 新增 `DisasterOperationDTO` DTO，包含前端展示操作进度所需的核心字段：
  - 基础信息：名称、命名空间、操作类型、所属组名
  - 执行状态：`state`（Pending / Running / Completed / Failed）、`currentStep`、`message`
  - 步骤详情：`steps[]`（每步 name / state / startTime / completionTime / message）
  - 组级别状态：`groupStatus`（totalLevels / currentLevelIndex / levelStatuses[]）
  - 时间戳：`startTime`、`completionTime`
- 遵循 `StreamWatch` + `WatchEventDTO` 标准协议（与 DisasterDrill / DisasterJob Watch 实现保持一致）

## Impact

- 受影响的规范：`changes/add-group-operation-watch/specs/disaster-group/spec.md`（新增）
- 受影响的代码：
  - `internal/apis/disaster_group/v1/types.go`（新增 `DisasterOperationDTO` 及转换函数）
  - `internal/apis/disaster_group/v1/handler.go`（新增 `watchGroupOperations` / `watchGroupOperation` 方法）
  - `internal/apis/disaster_group/v1/router.go`（注册两条 Watch 路由）
