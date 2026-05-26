## ADDED Requirements

### Requirement: 实例同步状态必须返回上次同步结果

系统 MUST 在 `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/sync-status` 的 `data.dataSync` 与 `data.resourceSync` 中返回上次同步结果字段 `lastSyncStatus`。

`lastSyncStatus` MUST 来自对应子资源的 `status.history` 最新记录。最新记录的选择规则 MUST 固定为：先选择 `completionTime` 最大的记录；没有 `completionTime` 时选择 `startTime` 最大的记录；两个时间都为空时选择 `history` 数组最后一条记录。

`lastSyncStatus.status` MUST 使用 `SyncHistoryRecord.status`。当对应子资源存在且 `status.history` 为空时，响应 MUST 省略该子资源的 `lastSyncStatus` 字段。

现有 `SubResourceStatusDTO.status` MUST 继续表示 DataSync/ResourceSync 子资源当前状态，不得改为上次同步结果。

#### Scenario: DataSync 返回上次同步结果
- **GIVEN** `DisasterInstance.status.dataSyncName` 指向的 DataSync 存在
- **AND** 该 DataSync `status.history` 至少包含一条记录
- **WHEN** 客户端请求 `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/demo/sync-status`
- **THEN** 响应 `data.dataSync.lastSyncStatus.status` 必须等于最新 history 记录的 `status`
- **AND** 响应 `data.dataSync.status` 必须继续等于 DataSync 当前 `status.state`

#### Scenario: ResourceSync 无历史时省略上次同步结果
- **GIVEN** `DisasterInstance.status.resourceSyncName` 指向的 ResourceSync 存在
- **AND** 该 ResourceSync `status.history` 为空
- **WHEN** 客户端请求 `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/demo/sync-status`
- **THEN** 响应 `data.resourceSync` 必须存在
- **AND** 响应 `data.resourceSync.lastSyncStatus` 必须省略

### Requirement: 系统必须提供实例同步历史接口

系统 MUST 提供 `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/sync-history`，用于查询指定 DisasterInstance 的数据同步和资源同步历史。

该接口 MUST 支持以下查询参数：

| 参数 | 取值 |
|------|------|
| `namespace` | 实例 namespace；为空时自动解析 |
| `syncType` | `all`、`dataSync`、`resourceSync`、`syncOnce` |
| `status` | `all`、`Pending`、`Running`、`Completed`、`Failed`、`Unknown` |
| `source` | `syncRecord`、`operation`、`all` |
| `page` | 页码 |
| `limit` | 分页大小，`-1` 表示全部 |

当 `source=syncRecord` 时，系统 MUST 从 DataSync 与 ResourceSync 的 `status.history` 构建历史记录。当 `source=operation` 时，系统 MUST 从 `DisasterOperation` 构建历史记录，并且只包含 `operationType` 为 `syncdata`、`syncresource`、`synconce` 的记录。当 `source=all` 时，系统 MUST 返回两类来源的并集，并通过 `source` 字段区分。

返回结果 MUST 使用标准 collection envelope。`meta.summary` MUST 基于过滤后、分页前集合计算，包含 `totalCount`、`dataSyncCount`、`resourceSyncCount`、`completedCount`、`failedCount`。

#### Scenario: 默认返回同步记录历史
- **GIVEN** 指定实例关联的 DataSync 有 1 条 history
- **AND** 指定实例关联的 ResourceSync 有 1 条 history
- **WHEN** 客户端请求 `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/demo/sync-history`
- **THEN** 响应 HTTP 状态码必须为 `200`
- **AND** 响应 `data.items` 必须包含 2 条记录
- **AND** 每条记录的 `source` 必须为 `syncRecord`
- **AND** 响应 `meta.summary.totalCount` 必须为 `2`

#### Scenario: 查询同步操作历史
- **GIVEN** 指定实例存在 3 条 DisasterOperation
- **AND** 其中 2 条 `operationType` 分别为 `syncdata` 与 `syncresource`
- **AND** 另外 1 条 `operationType` 为 `failover`
- **WHEN** 客户端请求 `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/demo/sync-history?source=operation`
- **THEN** 响应 `data.items` 必须只包含 2 条记录
- **AND** 每条记录的 `source` 必须为 `operation`
- **AND** 每条记录必须返回 `operationName`
- **AND** 每条记录必须返回 `operationUID`
- **AND** 每条记录的 `hasOperationDetail` 必须为 `true`

#### Scenario: 无同步历史时返回空集合
- **GIVEN** 指定实例存在
- **AND** 该实例关联的 DataSync 与 ResourceSync 不存在 history 记录
- **WHEN** 客户端请求 `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/demo/sync-history`
- **THEN** 响应 HTTP 状态码必须为 `200`
- **AND** 响应 `data.items` 必须为空数组
- **AND** 响应 `meta.summary.totalCount` 必须为 `0`

#### Scenario: 非法查询参数返回请求错误
- **WHEN** 客户端请求 `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/demo/sync-history?source=bad`
- **THEN** 服务端必须返回 `400`
- **AND** 响应业务码必须为 `1000`
