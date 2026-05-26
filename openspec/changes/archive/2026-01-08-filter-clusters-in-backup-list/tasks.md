# 任务：实现 AppBackup 关联集群列表 API

## 1. 实现 (Implementation)
- [x] 1.1 **AppBackup 集群列表 API**: 在 `internal/apis/app_backup/v1` 中实现 `GET /app-backups/clusters`，并在路由中注册。
  - 需要在 `handler.go` 中新增 `getAppBackupClusters` 方法。
  - 逻辑：List 所有 AppBackup -> 收集 Unique `spec.cluster` -> 返回。
- [x] 1.2 **OpenSpec 更新**: 为 `app-backup` 模块添加新的 API 规范。
