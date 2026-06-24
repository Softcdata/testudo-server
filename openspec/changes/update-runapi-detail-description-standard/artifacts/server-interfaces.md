# Server 全量接口清单

> 生成依据：`internal/router/router.go` 与各模块 `router.go`。完整路径已展开 `/apis`、`/api` 与模块 API group 前缀。

- 接口总数：156
- 路径对账主键：`METHOD + 标准化路径`

## Kubernetes 资源

| 方法 | 完整路径 | Handler | WebSocket | 认证 | 直接操作资源（初步） |
|---|---|---|---|---|---|
| `GET` | `/apis/resources.testudo.softcdata.com/v1/:resource` | `k.getResources` | 否 | JWT（dev 环境跳过） | Kubernetes 原生资源列表 |

## 事件与历史

| 方法 | 完整路径 | Handler | WebSocket | 认证 | 直接操作资源（初步） |
|---|---|---|---|---|---|
| `GET` | `/apis/v1/:resource/:name/history` | `h.listResourceEvents` | 否 | JWT（dev 环境跳过） | Kubernetes Event |
| `GET` | `/apis/v1/events` | `h.listEvents` | 否 | JWT（dev 环境跳过） | Kubernetes Event |
| `GET` | `/apis/v1/watch/:resource/:name/events` | `h.watchResourceEvents` | 是 | JWT（dev 环境跳过） | Kubernetes Event |
| `GET` | `/apis/v1/watch/events` | `h.watchEvents` | 是 | JWT（dev 环境跳过） | Kubernetes Event |

## 删除检查

| 方法 | 完整路径 | Handler | WebSocket | 认证 | 直接操作资源（初步） |
|---|---|---|---|---|---|
| `POST` | `/api/v1/deletion/check` | `h.check` | 否 | JWT（dev 环境跳过） | 被删除目标资源依赖检查 |
| `POST` | `/apis/v1/deletion/check` | `h.check` | 否 | JWT（dev 环境跳过） | 被删除目标资源依赖检查 |

## 存储仓库

| 方法 | 完整路径 | Handler | WebSocket | 认证 | 直接操作资源（初步） |
|---|---|---|---|---|---|
| `GET` | `/apis/storage.testudo.softcdata.com/v1/storages` | `c.storages` | 否 | JWT（dev 环境跳过） | StorageRepository、Velero BackupStorageLocation |
| `POST` | `/apis/storage.testudo.softcdata.com/v1/storages` | `c.createStorage` | 否 | JWT（dev 环境跳过） | StorageRepository、Velero BackupStorageLocation |
| `DELETE` | `/apis/storage.testudo.softcdata.com/v1/storages/:name` | `c.deleteStorage` | 否 | JWT（dev 环境跳过） | StorageRepository、Velero BackupStorageLocation |
| `GET` | `/apis/storage.testudo.softcdata.com/v1/storages/:name` | `c.storage` | 否 | JWT（dev 环境跳过） | StorageRepository、Velero BackupStorageLocation |
| `PATCH` | `/apis/storage.testudo.softcdata.com/v1/storages/:name` | `c.patchStorage` | 否 | JWT（dev 环境跳过） | StorageRepository、Velero BackupStorageLocation |
| `PUT` | `/apis/storage.testudo.softcdata.com/v1/storages/:name` | `c.updateStorage` | 否 | JWT（dev 环境跳过） | StorageRepository、Velero BackupStorageLocation |
| `GET` | `/apis/storage.testudo.softcdata.com/v1/storages/:name/validate` | `c.validateStorage` | 否 | JWT（dev 环境跳过） | StorageRepository、Velero BackupStorageLocation |
| `POST` | `/apis/storage.testudo.softcdata.com/v1/storages/connectivity/validate` | `c.validateBSLConnectivity` | 否 | JWT（dev 环境跳过） | StorageRepository、Velero BackupStorageLocation |
| `GET` | `/apis/storage.testudo.softcdata.com/v1/storages/names` | `c.storageNames` | 否 | JWT（dev 环境跳过） | StorageRepository、Velero BackupStorageLocation |
| `POST` | `/apis/storage.testudo.softcdata.com/v1/storages/validate/connection` | `c.validateS3Connection` | 否 | JWT（dev 环境跳过） | StorageRepository、Velero BackupStorageLocation |
| `GET` | `/apis/storage.testudo.softcdata.com/v1/watch/storages` | `c.watchStorages` | 是 | JWT（dev 环境跳过） | StorageRepository、Velero BackupStorageLocation |

## 容灾任务

| 方法 | 完整路径 | Handler | WebSocket | 认证 | 直接操作资源（初步） |
|---|---|---|---|---|---|
| `GET` | `/apis/disasterjobs.testudo.softcdata.com/v1/jobs` | `c.configs` | 否 | JWT（dev 环境跳过） | DisasterJob |
| `POST` | `/apis/disasterjobs.testudo.softcdata.com/v1/jobs` | `c.createConfig` | 否 | JWT（dev 环境跳过） | DisasterJob |
| `DELETE` | `/apis/disasterjobs.testudo.softcdata.com/v1/jobs/:name` | `c.deleteConfig` | 否 | JWT（dev 环境跳过） | DisasterJob |
| `GET` | `/apis/disasterjobs.testudo.softcdata.com/v1/jobs/:name` | `c.config` | 否 | JWT（dev 环境跳过） | DisasterJob |
| `GET` | `/apis/disasterjobs.testudo.softcdata.com/v1/watch/jobs` | `c.watchJobs` | 是 | JWT（dev 环境跳过） | DisasterJob |
| `GET` | `/apis/disasterjobs.testudo.softcdata.com/v1/watch/jobs/:name` | `c.watchJob` | 是 | JWT（dev 环境跳过） | DisasterJob |

## 容灾备份

| 方法 | 完整路径 | Handler | WebSocket | 认证 | 直接操作资源（初步） |
|---|---|---|---|---|---|
| `GET` | `/apis/disasterbackups.testudo.softcdata.com/v1/backups` | `c.backups` | 否 | JWT（dev 环境跳过） | DisasterBackup |
| `POST` | `/apis/disasterbackups.testudo.softcdata.com/v1/backups` | `c.createBackup` | 否 | JWT（dev 环境跳过） | DisasterBackup |
| `DELETE` | `/apis/disasterbackups.testudo.softcdata.com/v1/backups/:name` | `c.deleteBackup` | 否 | JWT（dev 环境跳过） | DisasterBackup |
| `GET` | `/apis/disasterbackups.testudo.softcdata.com/v1/backups/:name` | `c.backup` | 否 | JWT（dev 环境跳过） | DisasterBackup |
| `PUT` | `/apis/disasterbackups.testudo.softcdata.com/v1/backups/:name` | `c.updateBackup` | 否 | JWT（dev 环境跳过） | DisasterBackup |
| `GET` | `/apis/disasterbackups.testudo.softcdata.com/v1/watch/backups` | `c.watchBackups` | 是 | JWT（dev 环境跳过） | DisasterBackup |
| `GET` | `/apis/disasterbackups.testudo.softcdata.com/v1/watch/backups/:name` | `c.watchBackup` | 是 | JWT（dev 环境跳过） | DisasterBackup |

## 容灾实例

| 方法 | 完整路径 | Handler | WebSocket | 认证 | 直接操作资源（初步） |
|---|---|---|---|---|---|
| `GET` | `/apis/disasterinstances.testudo.softcdata.com/v1/instances` | `h.listInstances` | 否 | JWT（dev 环境跳过） | DisasterInstance、DisasterOperation、DataSync、ResourceSync |
| `POST` | `/apis/disasterinstances.testudo.softcdata.com/v1/instances` | `h.createInstance` | 否 | JWT（dev 环境跳过） | DisasterInstance、DisasterOperation、DataSync、ResourceSync |
| `DELETE` | `/apis/disasterinstances.testudo.softcdata.com/v1/instances/:name` | `h.deleteInstance` | 否 | JWT（dev 环境跳过） | DisasterInstance、DisasterOperation、DataSync、ResourceSync |
| `GET` | `/apis/disasterinstances.testudo.softcdata.com/v1/instances/:name` | `h.getInstance` | 否 | JWT（dev 环境跳过） | DisasterInstance、DisasterOperation、DataSync、ResourceSync |
| `PUT` | `/apis/disasterinstances.testudo.softcdata.com/v1/instances/:name` | `h.updateInstance` | 否 | JWT（dev 环境跳过） | DisasterInstance、DisasterOperation、DataSync、ResourceSync |
| `POST` | `/apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/actions` | `h.executeAction` | 否 | JWT（dev 环境跳过） | DisasterInstance、DisasterOperation、DataSync、ResourceSync |
| `GET` | `/apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/groups` | `h.getInstanceGroups` | 否 | JWT（dev 环境跳过） | DisasterInstance、DisasterOperation、DataSync、ResourceSync |
| `GET` | `/apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/history` | `h.getHistory` | 否 | JWT（dev 环境跳过） | DisasterInstance、DisasterOperation、DataSync、ResourceSync |
| `GET` | `/apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/operations/:operationName` | `h.getOperationDetail` | 否 | JWT（dev 环境跳过） | DisasterInstance、DisasterOperation、DataSync、ResourceSync |
| `POST` | `/apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/restore-classes/validate` | `h.validateRestoreClasses` | 否 | JWT（dev 环境跳过） | DisasterInstance、DisasterOperation、DataSync、ResourceSync |
| `GET` | `/apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/sync-status` | `h.getSyncStatus` | 否 | JWT（dev 环境跳过） | DisasterInstance、DisasterOperation、DataSync、ResourceSync |
| `GET` | `/apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/validate-target` | `h.validateTarget` | 否 | JWT（dev 环境跳过） | DisasterInstance、DisasterOperation、DataSync、ResourceSync |
| `GET` | `/apis/disasterinstances.testudo.softcdata.com/v1/watch/instances` | `h.watchInstances` | 是 | JWT（dev 环境跳过） | DisasterInstance、DisasterOperation、DataSync、ResourceSync |
| `GET` | `/apis/disasterinstances.testudo.softcdata.com/v1/watch/instances/:name` | `h.watchInstance` | 是 | JWT（dev 环境跳过） | DisasterInstance、DisasterOperation、DataSync、ResourceSync |
| `GET` | `/apis/disasterinstances.testudo.softcdata.com/v1/watch/instances/operations/:operationName` | `h.watchOperation` | 是 | JWT（dev 环境跳过） | DisasterInstance、DisasterOperation、DataSync、ResourceSync |

## 容灾演练

| 方法 | 完整路径 | Handler | WebSocket | 认证 | 直接操作资源（初步） |
|---|---|---|---|---|---|
| `GET` | `/apis/disasterdrills.testudo.softcdata.com/v1/drills` | `h.listDrills` | 否 | JWT（dev 环境跳过） | DisasterDrill |
| `POST` | `/apis/disasterdrills.testudo.softcdata.com/v1/drills` | `h.createDrill` | 否 | JWT（dev 环境跳过） | DisasterDrill |
| `DELETE` | `/apis/disasterdrills.testudo.softcdata.com/v1/drills/:name` | `h.deleteDrill` | 否 | JWT（dev 环境跳过） | DisasterDrill |
| `GET` | `/apis/disasterdrills.testudo.softcdata.com/v1/drills/:name` | `h.getDrill` | 否 | JWT（dev 环境跳过） | DisasterDrill |
| `POST` | `/apis/disasterdrills.testudo.softcdata.com/v1/drills/:name/cleanup` | `h.cleanupDrill` | 否 | JWT（dev 环境跳过） | DisasterDrill |
| `POST` | `/apis/disasterdrills.testudo.softcdata.com/v1/drills/:name/confirm` | `h.confirmDrill` | 否 | JWT（dev 环境跳过） | DisasterDrill |
| `POST` | `/apis/disasterdrills.testudo.softcdata.com/v1/drills/:name/restart` | `h.restartDrill` | 否 | JWT（dev 环境跳过） | DisasterDrill |
| `GET` | `/apis/disasterdrills.testudo.softcdata.com/v1/drills/actions/protected-namespaces` | `h.getProtectedNamespaces` | 否 | JWT（dev 环境跳过） | DisasterDrill |
| `GET` | `/apis/disasterdrills.testudo.softcdata.com/v1/watch/drills` | `h.watchDrills` | 是 | JWT（dev 环境跳过） | DisasterDrill |
| `GET` | `/apis/disasterdrills.testudo.softcdata.com/v1/watch/drills/:name` | `h.watchDrill` | 是 | JWT（dev 环境跳过） | DisasterDrill |

## 容灾策略

| 方法 | 完整路径 | Handler | WebSocket | 认证 | 直接操作资源（初步） |
|---|---|---|---|---|---|
| `GET` | `/apis/policies.testudo.softcdata.com/v1/policies` | `h.policies` | 否 | JWT（dev 环境跳过） | DisasterPolicy、AppBackup 引用关系 |
| `POST` | `/apis/policies.testudo.softcdata.com/v1/policies` | `h.createPolicy` | 否 | JWT（dev 环境跳过） | DisasterPolicy、AppBackup 引用关系 |
| `DELETE` | `/apis/policies.testudo.softcdata.com/v1/policies/:name` | `h.deletePolicy` | 否 | JWT（dev 环境跳过） | DisasterPolicy、AppBackup 引用关系 |
| `GET` | `/apis/policies.testudo.softcdata.com/v1/policies/:name` | `h.policy` | 否 | JWT（dev 环境跳过） | DisasterPolicy、AppBackup 引用关系 |
| `PUT` | `/apis/policies.testudo.softcdata.com/v1/policies/:name` | `h.updatePolicy` | 否 | JWT（dev 环境跳过） | DisasterPolicy、AppBackup 引用关系 |
| `GET` | `/apis/policies.testudo.softcdata.com/v1/policies/names` | `h.policyNames` | 否 | JWT（dev 环境跳过） | DisasterPolicy、AppBackup 引用关系 |

## 容灾组

| 方法 | 完整路径 | Handler | WebSocket | 认证 | 直接操作资源（初步） |
|---|---|---|---|---|---|
| `GET` | `/apis/disastergroups.testudo.softcdata.com/v1/groups` | `h.listGroups` | 否 | JWT（dev 环境跳过） | DisasterGroup、DisasterOperation、DisasterInstance |
| `POST` | `/apis/disastergroups.testudo.softcdata.com/v1/groups` | `h.createGroup` | 否 | JWT（dev 环境跳过） | DisasterGroup、DisasterOperation、DisasterInstance |
| `DELETE` | `/apis/disastergroups.testudo.softcdata.com/v1/groups/:name` | `h.deleteGroup` | 否 | JWT（dev 环境跳过） | DisasterGroup、DisasterOperation、DisasterInstance |
| `GET` | `/apis/disastergroups.testudo.softcdata.com/v1/groups/:name` | `h.getGroup` | 否 | JWT（dev 环境跳过） | DisasterGroup、DisasterOperation、DisasterInstance |
| `PATCH` | `/apis/disastergroups.testudo.softcdata.com/v1/groups/:name` | `h.updateGroup` | 否 | JWT（dev 环境跳过） | DisasterGroup、DisasterOperation、DisasterInstance |
| `PUT` | `/apis/disastergroups.testudo.softcdata.com/v1/groups/:name` | `h.updateGroup` | 否 | JWT（dev 环境跳过） | DisasterGroup、DisasterOperation、DisasterInstance |
| `POST` | `/apis/disastergroups.testudo.softcdata.com/v1/groups/:name/actions` | `h.executeAction` | 否 | JWT（dev 环境跳过） | DisasterGroup、DisasterOperation、DisasterInstance |
| `GET` | `/apis/disastergroups.testudo.softcdata.com/v1/groups/:name/history` | `h.getHistory` | 否 | JWT（dev 环境跳过） | DisasterGroup、DisasterOperation、DisasterInstance |
| `GET` | `/apis/disastergroups.testudo.softcdata.com/v1/groups/:name/instances` | `h.listGroupInstances` | 否 | JWT（dev 环境跳过） | DisasterGroup、DisasterOperation、DisasterInstance |
| `GET` | `/apis/disastergroups.testudo.softcdata.com/v1/groups/:name/operations/:operationName` | `h.getOperationDetail` | 否 | JWT（dev 环境跳过） | DisasterGroup、DisasterOperation、DisasterInstance |
| `GET` | `/apis/disastergroups.testudo.softcdata.com/v1/groups/instance-picker` | `h.instancePicker` | 否 | JWT（dev 环境跳过） | DisasterGroup、DisasterOperation、DisasterInstance |
| `GET` | `/apis/disastergroups.testudo.softcdata.com/v1/watch/groups/operations` | `h.watchGroupOperations` | 是 | JWT（dev 环境跳过） | DisasterGroup、DisasterOperation、DisasterInstance |
| `GET` | `/apis/disastergroups.testudo.softcdata.com/v1/watch/groups/operations/:operationName` | `h.watchGroupOperation` | 是 | JWT（dev 环境跳过） | DisasterGroup、DisasterOperation、DisasterInstance |
| `GET` | `/apis/disastergroups.testudo.softcdata.com/v1/watch/groups/status` | `h.watchGroupStatuses` | 是 | JWT（dev 环境跳过） | DisasterGroup、DisasterOperation、DisasterInstance |
| `GET` | `/apis/disastergroups.testudo.softcdata.com/v1/watch/groups/status/:name` | `h.watchGroupStatus` | 是 | JWT（dev 环境跳过） | DisasterGroup、DisasterOperation、DisasterInstance |

## 容灾配置

| 方法 | 完整路径 | Handler | WebSocket | 认证 | 直接操作资源（初步） |
|---|---|---|---|---|---|
| `GET` | `/apis/disasterconfigs.testudo.softcdata.com/v1/configs` | `c.configs` | 否 | JWT（dev 环境跳过） | DisasterConfig |
| `POST` | `/apis/disasterconfigs.testudo.softcdata.com/v1/configs` | `c.createConfig` | 否 | JWT（dev 环境跳过） | DisasterConfig |
| `DELETE` | `/apis/disasterconfigs.testudo.softcdata.com/v1/configs/:name` | `c.deleteConfig` | 否 | JWT（dev 环境跳过） | DisasterConfig |
| `GET` | `/apis/disasterconfigs.testudo.softcdata.com/v1/configs/:name` | `c.config` | 否 | JWT（dev 环境跳过） | DisasterConfig |
| `PUT` | `/apis/disasterconfigs.testudo.softcdata.com/v1/configs/:name` | `c.updateConfig` | 否 | JWT（dev 环境跳过） | DisasterConfig |
| `GET` | `/apis/disasterconfigs.testudo.softcdata.com/v1/configs/names` | `c.configNames` | 否 | JWT（dev 环境跳过） | DisasterConfig |
| `GET` | `/apis/disasterconfigs.testudo.softcdata.com/v1/watch/configs` | `c.watchConfigs` | 是 | JWT（dev 环境跳过） | DisasterConfig |
| `GET` | `/apis/disasterconfigs.testudo.softcdata.com/v1/watch/configs/:name` | `c.watchConfig` | 是 | JWT（dev 环境跳过） | DisasterConfig |

## 应用备份

| 方法 | 完整路径 | Handler | WebSocket | 认证 | 直接操作资源（初步） |
|---|---|---|---|---|---|
| `GET` | `/apis/appbackups.testudo.softcdata.com/v1/appbackups` | `c.appBackups` | 否 | JWT（dev 环境跳过） | AppBackup、Velero Backup/Schedule |
| `POST` | `/apis/appbackups.testudo.softcdata.com/v1/appbackups` | `c.createAppBackup` | 否 | JWT（dev 环境跳过） | AppBackup、Velero Backup/Schedule |
| `DELETE` | `/apis/appbackups.testudo.softcdata.com/v1/appbackups/:name` | `c.deleteAppBackup` | 否 | JWT（dev 环境跳过） | AppBackup、Velero Backup/Schedule |
| `GET` | `/apis/appbackups.testudo.softcdata.com/v1/appbackups/:name` | `c.appBackup` | 否 | JWT（dev 环境跳过） | AppBackup、Velero Backup/Schedule |
| `PUT` | `/apis/appbackups.testudo.softcdata.com/v1/appbackups/:name` | `c.updateAppBackup` | 否 | JWT（dev 环境跳过） | AppBackup、Velero Backup/Schedule |
| `POST` | `/apis/appbackups.testudo.softcdata.com/v1/appbackups/:name/actions/:type` | `c.executeAction` | 否 | JWT（dev 环境跳过） | AppBackup、Velero Backup/Schedule |
| `GET` | `/apis/appbackups.testudo.softcdata.com/v1/appbackups/:name/backups/:backupName/download` | `c.downloadBackup` | 否 | JWT（dev 环境跳过） | AppBackup、StorageRepository、下载票据 |
| `GET` | `/apis/appbackups.testudo.softcdata.com/v1/appbackups/:name/backups/:backupName/download/stream` | `c.downloadBackupStream` | 否 | 短期 `downloadToken` | AppBackup、StorageRepository、对象存储对象流 |
| `GET` | `/apis/appbackups.testudo.softcdata.com/v1/appbackups/:name/history` | `c.getBackupHistory` | 否 | JWT（dev 环境跳过） | AppBackup、Velero Backup/Schedule |
| `GET` | `/apis/appbackups.testudo.softcdata.com/v1/appbackups/clusters` | `c.getAppBackupClusters` | 否 | JWT（dev 环境跳过） | AppBackup、Velero Backup/Schedule |
| `GET` | `/apis/appbackups.testudo.softcdata.com/v1/velero/backups/:backupName/includes` | `c.getVeleroBackupIncludes` | 否 | JWT（dev 环境跳过） | AppBackup、Velero Backup/Schedule |
| `GET` | `/apis/appbackups.testudo.softcdata.com/v1/watch/appbackups` | `c.watchAppBackups` | 是 | JWT（dev 环境跳过） | AppBackup、Velero Backup/Schedule |
| `GET` | `/apis/appbackups.testudo.softcdata.com/v1/watch/appbackups/:name` | `c.watchAppBackup` | 是 | JWT（dev 环境跳过） | AppBackup、Velero Backup/Schedule |

## 应用恢复

| 方法 | 完整路径 | Handler | WebSocket | 认证 | 直接操作资源（初步） |
|---|---|---|---|---|---|
| `GET` | `/apis/apprestores.testudo.softcdata.com/v1/apprestores` | `c.appRestores` | 否 | JWT（dev 环境跳过） | AppRestore、Velero Restore |
| `POST` | `/apis/apprestores.testudo.softcdata.com/v1/apprestores` | `c.createAppRestore` | 否 | JWT（dev 环境跳过） | AppRestore、Velero Restore |
| `DELETE` | `/apis/apprestores.testudo.softcdata.com/v1/apprestores/:name` | `c.deleteAppRestore` | 否 | JWT（dev 环境跳过） | AppRestore、Velero Restore |
| `GET` | `/apis/apprestores.testudo.softcdata.com/v1/apprestores/:name` | `c.appRestore` | 否 | JWT（dev 环境跳过） | AppRestore、Velero Restore |
| `PUT` | `/apis/apprestores.testudo.softcdata.com/v1/apprestores/:name` | `c.updateAppRestore` | 否 | JWT（dev 环境跳过） | AppRestore、Velero Restore |
| `POST` | `/apis/apprestores.testudo.softcdata.com/v1/apprestores/:name/actions/:type` | `c.executeAction` | 否 | JWT（dev 环境跳过） | AppRestore、Velero Restore |
| `POST` | `/apis/apprestores.testudo.softcdata.com/v1/apprestores/preflight/validate` | `c.validateRestorePreflight` | 否 | JWT（dev 环境跳过） | AppRestore、Velero Restore |
| `GET` | `/apis/apprestores.testudo.softcdata.com/v1/watch/apprestores` | `c.watchAppRestores` | 是 | JWT（dev 环境跳过） | AppRestore、Velero Restore |
| `GET` | `/apis/apprestores.testudo.softcdata.com/v1/watch/apprestores/:name` | `c.watchAppRestore` | 是 | JWT（dev 环境跳过） | AppRestore、Velero Restore |

## 根接口

| 方法 | 完整路径 | Handler | WebSocket | 认证 | 直接操作资源（初步） |
|---|---|---|---|---|---|
| `GET` | `/healthz` | `func(ctx context.Context, c *app.RequestContext` | 否 | 公开 | 服务健康与认证 token |
| `POST` | `/login` | `jwtMiddleware.LoginHandler` | 否 | 公开 | 服务健康与认证 token |
| `GET` | `/readyz` | `func(ctx context.Context, c *app.RequestContext` | 否 | 公开 | 服务健康与认证 token |
| `POST` | `/refresh_token` | `middleware.RefreshTokenHandler` | 否 | 公开 | 服务健康与认证 token |

## 用户管理

| 方法 | 完整路径 | Handler | WebSocket | 认证 | 直接操作资源（初步） |
|---|---|---|---|---|---|
| `GET` | `/api/v1/users` | `h.listUsers` | 否 | JWT（dev 环境跳过） | disaster-server 用户 Secret |
| `POST` | `/api/v1/users` | `h.createUser` | 否 | JWT（dev 环境跳过） | disaster-server 用户 Secret |
| `DELETE` | `/api/v1/users/:username` | `h.deleteUser` | 否 | JWT（dev 环境跳过） | disaster-server 用户 Secret |
| `PATCH` | `/api/v1/users/:username/password` | `h.patchUserPassword` | 否 | JWT（dev 环境跳过） | disaster-server 用户 Secret |
| `PATCH` | `/api/v1/users/:username/status` | `h.patchUserStatus` | 否 | JWT（dev 环境跳过） | disaster-server 用户 Secret |
| `GET` | `/apis/v1/users` | `h.listUsers` | 否 | JWT（dev 环境跳过） | disaster-server 用户 Secret |
| `POST` | `/apis/v1/users` | `h.createUser` | 否 | JWT（dev 环境跳过） | disaster-server 用户 Secret |
| `DELETE` | `/apis/v1/users/:username` | `h.deleteUser` | 否 | JWT（dev 环境跳过） | disaster-server 用户 Secret |
| `PATCH` | `/apis/v1/users/:username/password` | `h.patchUserPassword` | 否 | JWT（dev 环境跳过） | disaster-server 用户 Secret |
| `PATCH` | `/apis/v1/users/:username/status` | `h.patchUserStatus` | 否 | JWT（dev 环境跳过） | disaster-server 用户 Secret |

## 系统设置

| 方法 | 完整路径 | Handler | WebSocket | 认证 | 直接操作资源（初步） |
|---|---|---|---|---|---|
| `GET` | `/api/v1/system-settings` | `h.listSettings` | 否 | JWT（dev 环境跳过） | SystemSettings ConfigMap/Secret 及资产 |
| `POST` | `/api/v1/system-settings` | `h.createSetting` | 否 | JWT（dev 环境跳过） | SystemSettings ConfigMap/Secret 及资产 |
| `DELETE` | `/api/v1/system-settings/:config_key` | `h.deleteSetting` | 否 | JWT（dev 环境跳过） | SystemSettings ConfigMap/Secret 及资产 |
| `PUT` | `/api/v1/system-settings/:config_key` | `h.updateSetting` | 否 | JWT（dev 环境跳过） | SystemSettings ConfigMap/Secret 及资产 |
| `GET` | `/api/v1/system-settings/assets/:config_key` | `h.getAsset` | 否 | JWT（dev 环境跳过） | SystemSettings ConfigMap/Secret 及资产 |
| `POST` | `/api/v1/system-settings/assets/:config_key` | `h.uploadAsset` | 否 | JWT（dev 环境跳过） | SystemSettings ConfigMap/Secret 及资产 |
| `GET` | `/api/v1/system-settings/public` | `h.listPublicSettings` | 否 | 公开 | SystemSettings ConfigMap/Secret 及资产 |
| `GET` | `/apis/v1/system-settings` | `h.listSettings` | 否 | JWT（dev 环境跳过） | SystemSettings ConfigMap/Secret 及资产 |
| `POST` | `/apis/v1/system-settings` | `h.createSetting` | 否 | JWT（dev 环境跳过） | SystemSettings ConfigMap/Secret 及资产 |
| `DELETE` | `/apis/v1/system-settings/:config_key` | `h.deleteSetting` | 否 | JWT（dev 环境跳过） | SystemSettings ConfigMap/Secret 及资产 |
| `PUT` | `/apis/v1/system-settings/:config_key` | `h.updateSetting` | 否 | JWT（dev 环境跳过） | SystemSettings ConfigMap/Secret 及资产 |
| `GET` | `/apis/v1/system-settings/assets/:config_key` | `h.getAsset` | 否 | JWT（dev 环境跳过） | SystemSettings ConfigMap/Secret 及资产 |
| `POST` | `/apis/v1/system-settings/assets/:config_key` | `h.uploadAsset` | 否 | JWT（dev 环境跳过） | SystemSettings ConfigMap/Secret 及资产 |
| `GET` | `/apis/v1/system-settings/public` | `h.listPublicSettings` | 否 | JWT（dev 环境跳过） | SystemSettings ConfigMap/Secret 及资产 |

## 平台许可

| 方法 | 完整路径 | Handler | WebSocket | 认证 | 直接操作资源（初步） |
|---|---|---|---|---|---|
| `GET` | `/apis/v1/platform-license/status` | `h.status` | 否 | JWT（dev 环境跳过） | License 状态 ConfigMap、License Secret、Cluster |
| `POST` | `/apis/v1/platform-license/install` | `h.install` | 否 | JWT（dev 环境跳过） | License Secret |

## 统计

| 方法 | 完整路径 | Handler | WebSocket | 认证 | 直接操作资源（初步） |
|---|---|---|---|---|---|
| `GET` | `/apis/backuprestorestatistics.testudo.softcdata.com/v1/autobackups/execution-summary` | `h.GetAutoBackupExecutionSummary` | 否 | JWT（dev 环境跳过） | BackupRestoreStatistics、AppBackup、AppRestore、DisasterOperation、DisasterInstance、DisasterPolicy |
| `GET` | `/apis/backuprestorestatistics.testudo.softcdata.com/v1/backups` | `h.GetBackupStatistics` | 否 | JWT（dev 环境跳过） | BackupRestoreStatistics、AppBackup、AppRestore、DisasterOperation、DisasterInstance、DisasterPolicy |
| `GET` | `/apis/backuprestorestatistics.testudo.softcdata.com/v1/backups/success-rate` | `h.GetBackupSuccessRate` | 否 | JWT（dev 环境跳过） | BackupRestoreStatistics、AppBackup、AppRestore、DisasterOperation、DisasterInstance、DisasterPolicy |
| `GET` | `/apis/backuprestorestatistics.testudo.softcdata.com/v1/instances` | `h.GetInstanceStatistics` | 否 | JWT（dev 环境跳过） | BackupRestoreStatistics、AppBackup、AppRestore、DisasterOperation、DisasterInstance、DisasterPolicy |
| `GET` | `/apis/backuprestorestatistics.testudo.softcdata.com/v1/operations` | `h.GetOperationStatistics` | 否 | JWT（dev 环境跳过） | BackupRestoreStatistics、AppBackup、AppRestore、DisasterOperation、DisasterInstance、DisasterPolicy |
| `GET` | `/apis/backuprestorestatistics.testudo.softcdata.com/v1/operations/by-time` | `h.GetOperationStatisticsByTime` | 否 | JWT（dev 环境跳过） | BackupRestoreStatistics、AppBackup、AppRestore、DisasterOperation、DisasterInstance、DisasterPolicy |
| `GET` | `/apis/backuprestorestatistics.testudo.softcdata.com/v1/restores` | `h.GetRestoreStatistics` | 否 | JWT（dev 环境跳过） | BackupRestoreStatistics、AppBackup、AppRestore、DisasterOperation、DisasterInstance、DisasterPolicy |
| `GET` | `/apis/backuprestorestatistics.testudo.softcdata.com/v1/storage` | `h.GetStorageStatistics` | 否 | JWT（dev 环境跳过） | BackupRestoreStatistics、AppBackup、AppRestore、DisasterOperation、DisasterInstance、DisasterPolicy |
| `GET` | `/apis/backuprestorestatistics.testudo.softcdata.com/v1/tasks/progress` | `h.GetTaskProgressTrend` | 否 | JWT（dev 环境跳过） | BackupRestoreStatistics、AppBackup、AppRestore、DisasterOperation、DisasterInstance、DisasterPolicy |

## 集群管理

| 方法 | 完整路径 | Handler | WebSocket | 认证 | 直接操作资源（初步） |
|---|---|---|---|---|---|
| `GET` | `/apis/cluster.testudo.softcdata.com/v1/clusters` | `c.clusters` | 否 | JWT（dev 环境跳过） | Cluster |
| `POST` | `/apis/cluster.testudo.softcdata.com/v1/clusters` | `c.createCluster` | 否 | JWT（dev 环境跳过） | Cluster |
| `DELETE` | `/apis/cluster.testudo.softcdata.com/v1/clusters/:name` | `c.deleteCluster` | 否 | JWT（dev 环境跳过） | Cluster |
| `GET` | `/apis/cluster.testudo.softcdata.com/v1/clusters/:name` | `c.cluster` | 否 | JWT（dev 环境跳过） | Cluster |
| `PATCH` | `/apis/cluster.testudo.softcdata.com/v1/clusters/:name` | `c.patchCluster` | 否 | JWT（dev 环境跳过） | Cluster |
| `POST` | `/apis/cluster.testudo.softcdata.com/v1/clusters/:name/actions/refresh-namespaces` | `c.refreshNamespaces` | 否 | JWT（dev 环境跳过） | Cluster |
| `GET` | `/apis/cluster.testudo.softcdata.com/v1/clusters/:name/protected-namespaces` | `c.protectedNamespaces` | 否 | JWT（dev 环境跳过） | Cluster |
| `GET` | `/apis/cluster.testudo.softcdata.com/v1/clusters/:name/restore-classes` | `c.listRestoreClasses` | 否 | JWT（dev 环境跳过） | Cluster |
| `GET` | `/apis/cluster.testudo.softcdata.com/v1/clusters/:name/validate` | `c.validateCluster` | 否 | JWT（dev 环境跳过） | Cluster |
| `POST` | `/apis/cluster.testudo.softcdata.com/v1/clusters/kubeconfig/validate` | `c.validateKubeConfig` | 否 | JWT（dev 环境跳过） | Cluster |
| `GET` | `/apis/cluster.testudo.softcdata.com/v1/clusters/names` | `c.clusterNames` | 否 | JWT（dev 环境跳过） | Cluster |
| `GET` | `/apis/cluster.testudo.softcdata.com/v1/watch/clusters` | `c.watchClusters` | 是 | JWT（dev 环境跳过） | Cluster |
| `GET` | `/apis/cluster.testudo.softcdata.com/v1/watch/clusters/:name` | `c.watchCluster` | 是 | JWT（dev 环境跳过） | Cluster |
