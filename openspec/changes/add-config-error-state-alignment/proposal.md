# Change: 对齐 ConfigError 语义并修正服务端组聚合

## Why

`disaster-operator` 将输出 `DisasterInstance.status.fsmState=ConfigError`。当前 `disaster-server` 存在三处语义缺口：

1. `determineCurrentState` 仅识别 `Failed`。
2. `computeGroupFsmState` 未把 `ConfigError` 纳入 `Degraded` 触发条件。
3. `deriveGroupMemberStatus` 在配置异常分支把成员状态重写为 `Failed`，导致前端读不到 `ConfigError`。

这些缺口会让“实例已进入配置错误态”在 API 层被吞掉，最终影响用户判断。

## What Changes

### 1) 实例 DTO 状态映射

文件：`internal/apis/disaster_instance/v1/types.go`

- `determineCurrentState` 增加 `ConfigError -> Error` 映射。
- 保持 `status.fsmState` 原值透传，前端可直接看到 `ConfigError`。

### 2) 容灾组聚合状态

文件：`internal/apis/disaster_group/v1/handler.go`

- `computeGroupFsmState` 在错误优先级分支新增 `ConfigError`。
- 保持现有优先级顺序：`FailingOver`、`FailingBack`、错误态、全量一致态、初始化态、混合态。

### 3) 组成员派生状态

文件：`internal/apis/disaster_group/v1/handler.go`

- `deriveGroupMemberStatus` 在配置异常分支输出 `ConfigError`。
- `NotFound` 语义继续优先保留，不被覆盖。

### 4) 状态筛选

文件：`internal/apis/disaster_group/v1/handler.go`

- `matchStatus` 的 `error` 筛选新增 `ConfigError` 命中。

### 5) 测试补齐

文件：
- `internal/apis/disaster_group/v1/handler_test.go`
- `internal/apis/disaster_instance/v1/handler_test.go`

新增 `ConfigError` 相关映射、聚合、筛选、成员派生测试。

## Non-Goals

- 不调整前端按钮交互规则。
- 不新增 API 路由。
- 不改动响应信封结构。

## Impact

- Affected specs:
  - `disaster-group`
  - `disaster-instance`
- Affected code:
  - `internal/apis/disaster_instance/v1/types.go`
  - `internal/apis/disaster_instance/v1/handler_test.go`
  - `internal/apis/disaster_group/v1/handler.go`
  - `internal/apis/disaster_group/v1/handler_test.go`
