# Change: 容灾组聚合状态（Server 侧计算）

## Why

`DisasterInstance` 拥有状态机（`fsmState`：Protected / FailingOver / Active / Paused / Failed 等），但将多个实例纳入 `DisasterGroup` 管理后，**容灾组没有一个语义化的整体状态**。当前 `DisasterGroupStatusDTO` 只有 `totalInstances` / `readyInstances` 数字统计，UI 无法直接知道"当前组整体处于什么状态"以及"现在可以对整组执行哪些操作"。

Server 侧已在 `listGroups` 和 `getGroup` 中预加载并遍历所有组内实例，计算聚合状态的边际成本几乎为零，且无需修改 Operator 和 CRD。

## What Changes

- **ADDED** `DisasterGroupStatusDTO.FsmState`：由 Server 根据组内所有实例状态投票推导的聚合状态字符串
- **ADDED** `DisasterGroupStatusDTO.AvailableOperations`：当前聚合状态下可对整组执行的操作列表（类比 Instance 的 `availableOperations`）
- **ADDED** `computeGroupFsmState()` 纯函数：优先级投票算法，在 `handler.go` 中实现，`listGroups` 和 `getGroup` 均调用

## Impact

- Affected specs: `disaster-group`（新增能力）
- Affected code:
  - `internal/apis/disaster_group/v1/types.go` — `DisasterGroupStatusDTO` 新增两个字段
  - `internal/apis/disaster_group/v1/handler.go` — 新增 `computeGroupFsmState()`，在 `getGroup` 和 `listGroups` 中调用
