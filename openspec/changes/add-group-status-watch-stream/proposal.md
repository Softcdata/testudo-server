# Change: 新增容灾组状态 WebSocket 事件流

## Why

当前容灾组模块仅提供 `/watch/groups/operations` 与 `/watch/groups/operations/:operationName` 两条操作事件流接口。  
服务端缺少容灾组状态事件流接口，导致调用方无法以统一语义订阅 `DisasterGroup` 状态变化。

## What Changes

- 新增容灾组状态事件流路由：
  - `GET /watch/groups/status`
  - `GET /watch/groups/status/:name`
- 新增状态事件流处理逻辑，监听 `DisasterGroup` 资源变化并输出 `DisasterGroupDTO`。
- 保持 WebSocket 包络格式不变：`Envelope + WatchEventDTO`。
- 保持现有操作事件流接口行为不变：
  - `GET /watch/groups/operations`
  - `GET /watch/groups/operations/:operationName`

## Out Of Scope

- 前端订阅切换
- 现有操作事件流语义调整
- 容灾组控制器聚合策略调整

## Impact

- 受影响规范：`disaster-group`
- 受影响代码：
  - `internal/apis/disaster_group/v1/router.go`
  - `internal/apis/disaster_group/v1/handler.go`
  - `internal/apis/disaster_group/v1/types.go`
  - `internal/apis/disaster_group/v1/handler_test.go`
  - `internal/apis/disaster_group/v1/types_test.go`
