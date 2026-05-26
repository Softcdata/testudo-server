# Proposal: Expose Velero Backup Name in Status

## Background
用户反馈 `AppBackup` 的 API 响应中，`status` 字段虽然包含了 `backupStatus`（来自 Velero 的状态详情），但缺失了该状态对应的 Velero Backup 资源名称。这使得前端在展示详情时，难以将状态与具体的底层备份资源关联起来。

## Goal
1. 在 `AppBackup` 的 API 响应 `status` 结构中，增加 `veleroBackupName` 字段。
2. 在 `AppRestore` 的 API 响应中，增加 `targetNamespaces` 字段，以便前端直观展示恢复的目标命名空间。
3. 在 `AppRestore` 资源上，从关联的 `AppBackup` 中同步备份类型标签（如：手动/自动），以便区分恢复来源。
4. 在 `TaskEvent`（历史事件列表和 WS 推送）中增加 `StartTime` 和 `EndTime` 字段，以便前端计算和展示。
5. 在 `AppRestore` 创建/更新接口增加：`ExistingResourcePolicy` (资源冲突策略), `Timeout` (操作超时时间), `Description` (描述，存入 Annotation)。

## Implementation Plan

### 1. Server Side (AppBackup)
- 修改 `AppBackupStatusDTO` 结构体，增加 `veleroBackupName` 字段。
- 修改 `ConvertToAppBackupDTO` 转换函数：
    - 逻辑：尝试从 `AppBackup.Status.History` 中获取最近的一条记录（通常是列表的最后以项），将其 `Name` 赋值给 `veleroBackupName`。
    - 如果 `History` 为空，该字段留空。

### 2. Server Side (AppRestore)
- 修改 `AppRestoreDTO` 结构体，增加 `TargetNamespaces []string` 字段。
- 修改 `ConvertToAppRestoreDTO` 转换函数：
    - 逻辑：
        1. 优先收集 `Spec.NamespaceMapping` 中的所有 **Values**（目标命名空间）。
        2. 如果 Mapping 为空，且 `Spec.IncludedNamespaces` 不为空，则直接使用 `IncludedNamespaces`（原样恢复）。
        3. 仅作为 UI 展示辅助字段。

## API Changes
### AppBackup
```json
// GET /apis/app_backup/v1/appbackups/{name}
{
    "status": {
        "veleroBackupName": "app-backup-sample-20260115120000", // NEW FIELD
        "backupStatus": { ... }
    }
}
```

### AppRestore
```json
// GET /apis/app_restore/v1/apprestores/{name}
{
    "targetNamespaces": ["target-ns-1", "target-ns-2"], // NEW FIELD
    "spec": { ... },
    "status": { ... }
}
```
