# disaster-group Specification

## Purpose
TBD - created by archiving change add-instance-picker-api. Update Purpose after archive.
## Requirements
### Requirement: 容灾实例选择器接口 (Instance Picker API)

系统 SHALL 在容灾组路由组下提供一个专用的轻量级实例选择器接口，供前端构建"选择容灾实例"UI 组件时使用。

该接口 SHALL 返回所有 `DisasterInstance` 资源的简要信息，仅包含：

| 字段 | 来源 | 说明 |
|------|------|------|
| `name` | `metadata.name` | 实例名称 |
| `namespace` | `metadata.namespace` | 实例所在命名空间 |
| `labels` | `metadata.labels` | 实例标签（map） |
| `fsmState` | `status.fsmState` | 当前状态（如 `Protected`、`Paused`、`Failed` 等） |

接口 SHALL 不返回 Config 详情、Storage 详情、DataSyncStatus、ResourceSyncStatus 等重量级字段。

#### Scenario: 无过滤条件时返回所有实例简要列表

- **WHEN** 客户端 `GET /apis/disastergroups.testudo.softcdata.com/v1/groups/instance-picker` 且不带任何查询参数
- **THEN** 返回 HTTP 200
- **AND** 响应信封 `code` 为 0
- **AND** `data.type` 为 `"collection"`
- **AND** `data.data` 为 `InstancePickerItemDTO` 数组，每项仅包含 `name`、`namespace`、`labels`、`fsmState` 四个字段

#### Scenario: 按关键词搜索实例

- **WHEN** 客户端请求携带 `keyword=nginx`
- **THEN** 服务端在内存中对每个实例执行 Contains 匹配（不区分大小写），匹配范围包括：
  - `metadata.name`
  - `metadata.namespace`
  - `metadata.labels` 的所有 **值**（value）
- **AND** 仅返回至少匹配其中一个字段的实例
- **AND** 未匹配实例不出现在结果中

#### Scenario: 按状态过滤实例

- **WHEN** 客户端请求携带 `status=Protected`
- **THEN** 服务端仅返回 `fsmState == "Protected"` 的实例
- **AND** 其他状态的实例不出现在结果中

#### Scenario: 同时使用关键词与状态过滤

- **WHEN** 客户端请求携带 `keyword=app&status=Protected`
- **THEN** 服务端先按关键词 Contains 过滤，再按状态精确过滤（AND 语义）
- **AND** 仅返回同时满足两个条件的实例

#### Scenario: 标准分页

- **WHEN** 客户端请求携带 `page=1&limit=10`
- **THEN** 响应 `meta.pagination` 包含 `limit`、`total`、`page` 字段
- **AND** `data.data` 数组长度不超过 `limit` 值

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

