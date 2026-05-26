# Proposal: 统一 Watch 事件流与历史列表的数据结构

## Why
当前 `/watch/events` 接口返回的数据结构 (`TaskEvent`) 与 `/events` 历史接口不一致：
1. **任务名称 (TaskName)**: Watch 接口直接使用 Kubernetes Object Name，未解析结构化 JSON 消息中的 `task` 字段，导致显示不友好。
2. **动作 (Reason)**: Watch 接口未返回 Kubernetes Event 的 `Reason` 字段（如 `ExecutionStarted`, `ScalingReplicas`），导致前端无法展示当前具体动作。

## What Changes
1. 修改 `TaskEvent` 结构体，增加 `reason` 字段。
2. 修正 `ConvertToTaskEventDTO` 逻辑：
   - 优先从 `Message` JSON 载荷解析 `TaskName`（`task` 字段）。
   - 将 Event `Reason` 填充到 `TaskEvent.Reason`。
3. 同步更新 `list.go` 中的聚合逻辑，确保历史列表也能返回最新的 `Reason`。

## Impact
- **API**: `TaskEvent` DTO 新增 `reason` 字段。
- **Client**: 前端可通过 `reason` 字段展示当前动作，通过 `taskName` 展示友好名称。
