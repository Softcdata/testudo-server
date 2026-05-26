# API Standards Delta

## ADDED Requirements

### Requirement: WebSocket 事件流标准 (WebSocket Event Stream Standards)
所有基于 WebSocket 的事件流接口必须 (MUST) 遵循统一的响应格式和数据传输对象 (DTO) 规范。

#### Scenario: 标准化消息结构
- **WHEN** 服务端通过 WebSocket 推送事件
- **THEN** 消息体必须 (MUST) 是一个标准的 `Envelope` JSON 对象
- **AND** `code` 字段应为 0 (成功) 或错误码
- **AND** `data` 字段应包含 `WatchEventDTO`

#### Scenario: WatchEventDTO 结构
- **WHEN** 返回 `WatchEventDTO`
- **THEN** 它必须 (MUST) 包含 `type` 字段 (ADDED, MODIFIED, DELETED, ERROR)
- **AND** 它必须 (MUST) 包含 `object` 字段，该字段是资源的 DTO 表示，而不是原始 Kubernetes 对象

#### Scenario: 心跳消息
- **WHEN** 发送心跳保活消息
- **THEN** 应发送一个特殊的 `Envelope`
- **AND** `meta` 字段中包含 `type: "heartbeat"`
