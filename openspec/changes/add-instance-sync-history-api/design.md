# Design: 实例同步状态与同步历史接口

## 背景

实例详情页的同步 Tab 同时需要两个视角：

- 当前视角：DataSync/ResourceSync 子资源当前是否 Ready、InProgress、Failed。
- 历史视角：最近一次以及过去多次同步任务的执行结果。

当前 `sync-status` 只覆盖当前视角和统计次数，缺少上次同步结果。当前 `history` 接口覆盖通用操作视角，能看到 `syncdata`、`syncresource`、`synconce`，但它不适合作为同步 Tab 的主列表。

## Goals

- 在 `sync-status` 中稳定展示上次同步结果。
- 新增同步历史接口，直接服务同步 Tab。
- 复用 operator 已有 `DataSync.status.history` 与 `ResourceSync.status.history`。
- 复用现有 `DisasterOperation` 历史作为同步操作来源。
- 保持查询接口只读。

## Non-Goals

- 不修改 operator CRD。
- 不引入新的持久化表。
- 不改变 `GET /instances/:name/history` 的返回结构。
- 不在 server 中根据中文消息推断成功失败。

## Decisions

### D1. `sync-status` 新增 `lastSyncStatus`

`SubResourceStatusDTO.status` 继续表示当前子资源状态。上次同步结果单独放入 `lastSyncStatus`，避免前端把当前状态和上次任务结果混淆。

### D2. 同步历史默认读取子资源历史

`sync-history` 的默认 `source=syncRecord`。该模式读取：

- DataSync：`DataSync.status.history`
- ResourceSync：`ResourceSync.status.history`

这能覆盖策略触发和手动触发后最终落在子资源上的真实执行结果。

### D3. 操作历史作为可选来源

当客户端传 `source=operation` 时，server 读取 `DisasterOperation`，并只返回：

- `syncdata`
- `syncresource`
- `synconce`

该模式用于复用现有操作详情链路。`synconce` 投影为 `syncType=syncOnce`。

### D4. 不做弱关联去重

`SyncHistoryRecord` 当前没有 `operationName`、`operationUID` 字段。server 不通过时间窗口和名称猜测把 sync record 与 operation 强行合并。

当 `source=all` 时，接口返回两类来源的并集，并通过 `source` 字段区分。前端需要避免重复展示时，可以固定使用默认 `source=syncRecord`，在需要操作详情时切换到 `source=operation`。

### D5. 排序与分页固定

server 必须先收集记录，再执行过滤，再排序，再分页。排序优先级固定为：

1. `completionTime` 倒序
2. `startTime` 倒序
3. `operation.creationTimestamp` 倒序
4. `id` 倒序

`meta.summary` 必须基于过滤后、分页前集合计算。

## DTO

### LastSyncStatusDTO

```go
type LastSyncStatusDTO struct {
    Status               string       `json:"status"`
    StartTime            *metav1.Time `json:"startTime,omitempty"`
    CompletionTime       *metav1.Time `json:"completionTime,omitempty"`
    Duration             string       `json:"duration,omitempty"`
    BackupName           string       `json:"backupName,omitempty"`
    RestoreName          string       `json:"restoreName,omitempty"`
    BackupResourceCount  int          `json:"backupResourceCount,omitempty"`
    RestoreResourceCount int          `json:"restoreResourceCount,omitempty"`
}
```

### SyncHistoryItemDTO

```go
type SyncHistoryItemDTO struct {
    ID                   string           `json:"id"`
    SyncType             string           `json:"syncType"`
    Source               string           `json:"source"`
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

## Route Contract

### 同步状态

- Route: `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/sync-status`
- 行为：读取实例、DataSync、ResourceSync、BackupRestoreStatistics。
- 增量：`data.dataSync.lastSyncStatus` 与 `data.resourceSync.lastSyncStatus`。

### 同步历史

- Route: `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/sync-history`
- 行为：读取实例、DataSync、ResourceSync、DisasterOperation。
- 返回：标准 collection envelope。
- 无匹配记录：返回 `200`，`data.items=[]`，summary 全部为 0。

## 错误语义

- 自动解析 namespace 失败：返回 `404 code=3004`。
- 实例 Get 失败：返回 `404 code=3004`。
- DataSync、ResourceSync 读取失败：对应来源记录缺失，不让整个接口失败。
- DisasterOperation List 失败：当 `source=operation` 和 `source=all` 时返回 `500 code=5000`。
- 非法 `syncType`、`status`、`source`：返回 `400 code=1000`。

## 文档同步要求

- Swagger/OpenAPI 必须新增 `LastSyncStatusDTO`、`SyncHistoryItemDTO`、`SyncHistoryCollectionEnvelope`。
- Swagger/OpenAPI 必须更新 `SubResourceStatusDTO`，补充 `lastSyncStatus`。
- RunAPI 必须更新现有同步状态接口，并新增同步历史接口。
- RunAPI 更新必须保留原说明到 `## 原有说明`。
