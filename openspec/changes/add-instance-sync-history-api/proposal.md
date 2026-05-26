# Change: 增强实例同步状态与新增同步历史接口

## Why

当前 `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/sync-status` 已返回 DataSync 与 ResourceSync 的当前状态、最近同步时间和统计次数，但没有明确返回“上次同步结果状态”。前端在同步 Tab 中只能看到当前子资源状态，无法稳定展示最近一次同步是成功还是失败。

现有 `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/history` 已能展示 `syncdata`、`syncresource`、`synconce` 等同步相关操作，但它是通用操作历史，缺少只面向同步 Tab 的筛选、分页和同步字段口径。需要新增一个同步历史接口，让前端可以直接读取数据同步和资源同步历史。

## What Changes

### 1. 增强同步状态接口

增强现有接口：

- `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/sync-status`

在 `data.dataSync` 与 `data.resourceSync` 的 `SubResourceStatusDTO` 中新增：

- `lastSyncStatus`

字段结构：

```go
type LastSyncStatusDTO struct {
    Status               string       `json:"status"` // Completed, Failed, Unknown
    StartTime            *metav1.Time `json:"startTime,omitempty"`
    CompletionTime       *metav1.Time `json:"completionTime,omitempty"`
    Duration             string       `json:"duration,omitempty"`
    BackupName           string       `json:"backupName,omitempty"`
    RestoreName          string       `json:"restoreName,omitempty"`
    BackupResourceCount  int          `json:"backupResourceCount,omitempty"`
    RestoreResourceCount int          `json:"restoreResourceCount,omitempty"`
}
```

取值规则：

- DataSync 的 `lastSyncStatus` 来自 `DataSync.status.history` 中最新一条记录。
- ResourceSync 的 `lastSyncStatus` 来自 `ResourceSync.status.history` 中最新一条记录。
- 最新记录按 `completionTime` 最大值选择；`completionTime` 为空时按 `startTime` 最大值选择；两个时间都为空时按 `history` 数组中的最后一条记录选择。
- 当子资源存在但 `status.history` 为空时，省略 `lastSyncStatus`。
- 现有 `status` 字段继续表示 DataSync/ResourceSync 子资源当前状态，不改为上次同步结果。

### 2. 新增实例同步历史接口

新增接口：

- `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/sync-history`

查询参数：

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `namespace` | string | 空 | 实例所在 namespace；为空时复用 `findNamespace` 自动解析 |
| `syncType` | string | `all` | `all`、`dataSync`、`resourceSync`、`syncOnce` |
| `status` | string | `all` | `all`、`Pending`、`Running`、`Completed`、`Failed`、`Unknown` |
| `source` | string | `syncRecord` | `syncRecord` 读取 DataSync/ResourceSync `status.history`；`operation` 读取现有 `DisasterOperation` 同步操作；`all` 返回两类来源 |
| `page` | int | 1 | 页码 |
| `limit` | int | 10 | 分页大小，`-1` 表示全部 |

返回标准 collection envelope，`data.items[]` 为 `SyncHistoryItemDTO`：

```go
type SyncHistoryItemDTO struct {
    ID                   string           `json:"id"`
    SyncType             string           `json:"syncType"` // dataSync, resourceSync, syncOnce
    Source               string           `json:"source"`   // syncRecord, operation
    Status               HistoryStatusDTO `json:"status"`
    StartTime            *metav1.Time     `json:"startTime,omitempty"`
    CompletionTime       *metav1.Time     `json:"completionTime,omitempty"`
    Duration             string           `json:"duration,omitempty"`
    BackupName           string           `json:"backupName,omitempty"`
    RestoreName          string           `json:"restoreName,omitempty"`
    BackupResourceCount  int              `json:"backupResourceCount,omitempty"`
    RestoreResourceCount int              `json:"restoreResourceCount,omitempty"`
    SubResourceName      string           `json:"subResourceName,omitempty"`
    OperationName        string           `json:"operationName,omitempty"`
    OperationUID         string           `json:"operationUID,omitempty"`
    OperationType        string           `json:"operationType,omitempty"`
    HasOperationDetail   bool             `json:"hasOperationDetail"`
}
```

排序规则：

1. `completionTime` 倒序
2. `startTime` 倒序
3. `operation.creationTimestamp` 倒序
4. `id` 倒序

统计规则：

- `meta.summary.totalCount`：过滤后、分页前记录总数
- `meta.summary.dataSyncCount`：过滤后、分页前 `syncType=dataSync` 记录数
- `meta.summary.resourceSyncCount`：过滤后、分页前 `syncType=resourceSync` 记录数
- `meta.summary.completedCount`：过滤后、分页前 `status.state=Completed` 记录数
- `meta.summary.failedCount`：过滤后、分页前 `status.state=Failed` 记录数

### 3. 与现有操作历史的关系

- `GET /instances/:name/history` 继续作为通用操作历史接口，保留全部操作类型。
- `GET /instances/:name/sync-history` 只服务同步 Tab，默认读取 DataSync/ResourceSync 的真实同步记录。
- 当 `source=operation` 时，接口复用现有 `DisasterOperation` 历史来源，只返回 `operationType` 为 `syncdata`、`syncresource`、`synconce` 的记录。
- 当返回 `source=operation` 记录时，必须返回 `operationName`、`operationUID`、`operationType`、`hasOperationDetail=true`，前端可以继续调用现有 `GET /instances/:name/operations/:operationName` 查看详情。

## Non-Goals

- 不替换现有 `GET /instances/:name/history`。
- 不新增全局同步历史接口。
- 不在查询接口中触发同步动作。
- 不要求 operator 新增新的历史 CRD。
- 不通过中文 `message` 文本推断状态。

## Impact

- Affected specs:
  - `disaster_instance`
- Affected API:
  - `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/sync-status`
  - `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/sync-history`
- Affected code:
  - `internal/apis/disaster_instance/v1/router.go`
  - `internal/apis/disaster_instance/v1/handler.go`
  - `internal/apis/disaster_instance/v1/types.go`
  - `internal/apis/disaster_instance/v1/handler_test.go`
  - `openspec/specs/disaster-server-openapi.yaml`
- Documentation:
  - Swagger/OpenAPI 必须更新
  - RunAPI/Apipost 必须新增同步历史接口，并更新同步状态接口

## Acceptance

- `sync-status` 中的 DataSync 与 ResourceSync 均能返回独立的 `lastSyncStatus`。
- 当前状态 `status` 与上次同步结果 `lastSyncStatus.status` 不混用。
- `sync-history` 默认返回 DataSync/ResourceSync `status.history` 投影记录。
- `sync-history?source=operation` 只返回同步相关 `DisasterOperation`。
- `sync-history` 支持按 `syncType`、`status`、`source` 过滤，并在分页前计算 summary。
- Swagger/OpenAPI 与 RunAPI 均完成同步。
