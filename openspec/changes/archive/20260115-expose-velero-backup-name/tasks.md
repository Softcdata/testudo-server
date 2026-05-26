# 任务: 在状态中暴露 Velero 备份名称

## 1. 服务端实现 (AppBackup)
- [x] 1.1 修改 `internal/apis/app_backup/v1/types.go` 中的 `AppBackupStatusDTO`
    - 增加 `VeleroBackupName string json:"veleroBackupName,omitempty"`
- [x] 1.2 更新 `internal/apis/app_backup/v1/handler.go` 中的 `ConvertToStatusDTO`
    - 逻辑：如果可用，从 `obj.Status.History` 的最后一个元素提取名称。

## 2. Operator 实现 (disaster-operator)
- [x] 2.1 修改 `pkg/apis/disaster/v1/apprestore_types.go` 中的 `AppRestoreStatus`
    - 增加 `TargetNamespaces []string json:"targetNamespaces,omitempty"`
- [x] 2.2 更新 `internal/controller/apprestore/apprestore_state.go` 中的 `Reconcile` 逻辑
    - 逻辑：在 `PendingHandler` 中，获取 `BackupSource` (Velero Backup) 并从 `Backup.Spec.IncludedNamespaces` 填充 `Status.TargetNamespaces`。
- [x] 2.3 更新 `PendingHandler` 逻辑以传播备份类型标签
    - 逻辑：读取 `AppBackup` (源)，获取标签 `testudo.softcdata.com/backup-type` (或类似)，并将其作为 `testudo.softcdata.com/backup-source-type` 应用于 `AppRestore` 标签。

## 4. 服务端实现 (TaskEvent)
- [x] 4.1 修改 `internal/apis/event/v1/list.go` (和 `types.go`) 中的 `TaskEvent` 结构体
    - 增加 `StartTime *time.Time json:"startTime,omitempty"`
    - 增加 `EndTime *time.Time json:"endTime,omitempty"`
- [x] 4.2 更新 `internal/apis/event/v1/list.go` 中的 `aggregateEvents`
    - 逻辑：
        - `StartTime`: 使用 `ExecutionStarted` 事件的时间戳（或最早的事件）。
        - `EndTime`: 使用 `ExecutionFinished` 事件的时间戳（如果存在）。
- [x] 4.3 更新 `internal/apis/event/v1/types.go` 中的 `ConvertToTaskEventDTO` (用于 WS)
    - 逻辑：如果可用，填充 `StartTime` 和 `EndTime`（对于单个事件更新，StartTime 是事件时间，如果已完成，EndTime 可能为空或相同）。

## 5. 服务端实现 (AppRestore 增强)
- [x] 5.1 修改 `internal/apis/app_restore/v1/types.go` 中的 `AppRestore` DTOs 和请求
    - 在 `Create/Update` 请求和 `SpecDTO` 中增加 `ExistingResourcePolicy`, `Timeout`。
    - 在 `Create/Update` 请求和 `AppRestoreDTO` 中增加 `Description`。
    - 定义 `AppRestoreDescriptionAnnotation`。
- [x] 5.2 更新 `internal/apis/app_restore/v1/handler.go` 中的 `AppRestoreHandler`
    - 逻辑：处理 `Description` 字段以写入/更新 Annotation。

## 2. 验证

- [x] 2.1 验证 API 响应包含新字段。
