## ADDED Requirements

### Requirement: 容灾组操作列表 WebSocket 事件流 (Group Operation Watch List)

系统 SHALL 在容灾组路由组下提供 WebSocket 接口 `GET /watch/groups/operations`，用于监听 `DisasterOperation` 资源的实时变化，供前端感知组操作执行进度。

该接口 SHALL 支持按容灾组名（`groupName` Query 参数）过滤，仅返回属于该组的操作事件。

推送的每条消息 SHALL 遵循标准 `Envelope` + `WatchEventDTO` 格式：
```json
{
  "code": 0,
  "data": {
    "type": "MODIFIED",
    "object": { /* DisasterOperationDTO */ }
  }
}
```

其中 `DisasterOperationDTO` SHALL 包含：

| 字段 | 来源 | 说明 |
|------|------|------|
| `name` | `metadata.name` | 操作名称 |
| `namespace` | `metadata.namespace` | 命名空间 |
| `groupName` | `spec.groupName` | 所属容灾组 |
| `operationType` | `spec.operationType` | 操作类型 |
| `state` | `status.state` | 全局状态：Pending / Running / Completed / Failed |
| `currentStep` | `status.currentStep` | 当前执行步骤名称 |
| `message` | `status.message` | 附加消息或错误信息 |
| `steps` | `status.steps` | 各步骤详情数组 |
| `groupStatus` | `status.groupStatus` | 组级别分层执行状态 |
| `startTime` | `status.startTime` | 操作开始时间 |
| `completionTime` | `status.completionTime` | 操作完成时间 |
| `creationTimestamp` | `metadata.creationTimestamp` | 创建时间 |

#### Scenario: 连接成功并接收操作状态推送

- **WHEN** 前端建立 WebSocket 连接到 `/watch/groups/operations?groupName=my-group`
- **THEN** 服务端发送连接成功消息 `{"status": "connected"}`
- **AND** 每当 `DisasterOperation`（label `testudo.softcdata.com/group=my-group`）发生变化时，推送 `WatchEventDTO`
- **AND** `WatchEventDTO.type` 为 `ADDED` / `MODIFIED` / `DELETED` 之一
- **AND** `WatchEventDTO.object` 为对应的 `DisasterOperationDTO`

#### Scenario: 不传 groupName 时监听所有操作

- **WHEN** 前端连接到 `/watch/groups/operations`（不带 `groupName`）
- **THEN** 服务端监听 `disaster-system` 命名空间下所有 `DisasterOperation` 的变化
- **AND** 所有操作的变更均被推送

#### Scenario: 心跳保活

- **WHEN** 连接空闲超过 30 秒未有事件
- **THEN** 服务端发送心跳消息：`{"data": null, "meta": {"type": "heartbeat"}}`
- **AND** 前端收到心跳后无需响应

#### Scenario: 连接超时关闭

- **WHEN** 连接持续时间超过 30 分钟
- **THEN** 服务端主动关闭连接，发送 `{"meta": {"type": "timeout"}}`

---

### Requirement: 容灾组单个操作 WebSocket 事件流 (Group Single Operation Watch)

系统 SHALL 在容灾组路由组下提供 WebSocket 接口 `GET /watch/groups/operations/:operationName`，用于监听**指定名称**的 `DisasterOperation` 资源状态变化。

该接口专用于前端在触发操作后，通过操作名称精确订阅该操作的执行过程（Pending → Running → Completed/Failed），通过 FieldSelector `metadata.name=<operationName>` 过滤。

#### Scenario: 订阅指定操作进度直至完成

- **WHEN** 前端在执行 `POST /groups/:name/actions` 后，获取到返回的 `operationName`
- **AND** 前端建立 WebSocket 连接到 `/watch/groups/operations/<operationName>`
- **THEN** 每当该 `DisasterOperation` 的 `status` 发生变化，推送 `MODIFIED` 事件
- **AND** `WatchEventDTO.object.state` 依次变化：`Pending` → `Running` → `Completed` 或 `Failed`
- **AND** `WatchEventDTO.object.currentStep` 实时反映当前步骤
- **AND** `WatchEventDTO.object.steps` 数组包含每个步骤的执行详情
- **AND** 当 `state` 为 `Completed` 或 `Failed` 时，前端可据此关闭连接

#### Scenario: 操作不存在时的行为

- **WHEN** 前端订阅一个不存在的 operationName
- **THEN** 连接建立成功，但 watcher 不推送任何 `WatchEventDTO`
- **AND** 心跳正常发送
