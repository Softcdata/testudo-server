## ADDED Requirements

### Requirement: 容灾组状态列表 WebSocket 事件流 (Group Status Watch List)

系统 SHALL 在容灾组路由组下提供 WebSocket 接口 `GET /watch/groups/status`，用于监听 `DisasterGroup` 资源状态变化。

该接口推送消息 SHALL 使用标准 `Envelope`，其中 `data` 字段 SHALL 为 `WatchEventDTO`，`WatchEventDTO.object` SHALL 为 `DisasterGroupDTO`。

#### Scenario: 建立连接后接收容灾组状态事件

- **WHEN** 客户端连接 `GET /watch/groups/status`
- **THEN** 服务端发送连接成功消息 `{"status":"connected"}`
- **AND** 服务端先执行一次 `List(DisasterGroups)` 获取 `resourceVersion`
- **AND** 服务端使用该 `resourceVersion` 发起 `Watch(DisasterGroups)`
- **AND** 当容灾组发生 `ADDED`、`MODIFIED`、`DELETED` 事件时，服务端推送 `WatchEventDTO`
- **AND** `WatchEventDTO.object` 包含 `name`、`status.totalInstances`、`status.readyInstances`、`status.summary`、`status.fsmState`、`status.availableOperations` 字段

#### Scenario: 空闲连接保持心跳

- **WHEN** 连接空闲达到默认心跳间隔
- **THEN** 服务端发送心跳消息
- **AND** 心跳消息通过 `Envelope.meta.type=heartbeat` 标识

### Requirement: 容灾组状态详情 WebSocket 事件流 (Group Status Watch Detail)

系统 SHALL 在容灾组路由组下提供 WebSocket 接口 `GET /watch/groups/status/:name`，用于监听指定容灾组的状态变化。

详情事件流 SHALL 使用字段选择器 `metadata.name=<name>` 进行精确过滤，`WatchEventDTO.object` SHALL 为对应容灾组的 `DisasterGroupDTO`。

#### Scenario: 订阅指定容灾组状态变化

- **WHEN** 客户端连接 `GET /watch/groups/status/app-group-001`
- **THEN** 服务端仅推送名称为 `app-group-001` 的容灾组事件
- **AND** 推送事件中的 `WatchEventDTO.object.name` 始终为 `app-group-001`

#### Scenario: 兼容现有操作事件流

- **WHEN** 客户端继续使用 `GET /watch/groups/operations` 与 `GET /watch/groups/operations/:operationName`
- **THEN** 现有接口的路由、过滤规则、事件对象结构保持不变
