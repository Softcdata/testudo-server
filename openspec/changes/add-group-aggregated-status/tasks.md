## 1. 类型定义

- [x] 1.1 在 `internal/apis/disaster_group/v1/types.go` 的 `DisasterGroupStatusDTO` 中新增 `FsmState string` 和 `AvailableOperations []string` 两个字段

## 2. 实现聚合逻辑

- [x] 2.1 在 `internal/apis/disaster_group/v1/handler.go` 中新增 `computeGroupFsmState(instances []InstanceSummaryDTO) (string, []string)` 纯函数，实现优先级投票算法
- [x] 2.2 在 `getGroup` 中，调用 `collectInstanceSummaries` 之后调用 `computeGroupFsmState`，将结果写入 `dto.Status.FsmState` 和 `dto.Status.AvailableOperations`
- [x] 2.3 在 `listGroups` 中，调用 `collectInstanceSummariesWithCache` 之后同样调用 `computeGroupFsmState`

## 3. 测试

- [x] 3.1 为 `computeGroupFsmState` 编写单元测试，覆盖全部 9 种输出状态（表格驱动）
