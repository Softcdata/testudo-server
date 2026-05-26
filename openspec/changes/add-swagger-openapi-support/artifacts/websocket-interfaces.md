# Swagger/OpenAPI WebSocket 接口清单

- 接口数量：24

| 方法 | 路径 | 模块 | Handler |
|---|---|---|---|
| `GET` | `/apis/v1/watch/:resource/:name/events` | 事件与历史 | `h.watchResourceEvents` |
| `GET` | `/apis/v1/watch/events` | 事件与历史 | `h.watchEvents` |
| `GET` | `/apis/storage.testudo.softcdata.com/v1/watch/storages` | 存储仓库 | `c.watchStorages` |
| `GET` | `/apis/disasterjobs.testudo.softcdata.com/v1/watch/jobs` | 容灾任务 | `c.watchJobs` |
| `GET` | `/apis/disasterjobs.testudo.softcdata.com/v1/watch/jobs/:name` | 容灾任务 | `c.watchJob` |
| `GET` | `/apis/disasterbackups.testudo.softcdata.com/v1/watch/backups` | 容灾备份 | `c.watchBackups` |
| `GET` | `/apis/disasterbackups.testudo.softcdata.com/v1/watch/backups/:name` | 容灾备份 | `c.watchBackup` |
| `GET` | `/apis/disasterinstances.testudo.softcdata.com/v1/watch/instances` | 容灾实例 | `h.watchInstances` |
| `GET` | `/apis/disasterinstances.testudo.softcdata.com/v1/watch/instances/:name` | 容灾实例 | `h.watchInstance` |
| `GET` | `/apis/disasterinstances.testudo.softcdata.com/v1/watch/instances/operations/:operationName` | 容灾实例 | `h.watchOperation` |
| `GET` | `/apis/disasterdrills.testudo.softcdata.com/v1/watch/drills` | 容灾演练 | `h.watchDrills` |
| `GET` | `/apis/disasterdrills.testudo.softcdata.com/v1/watch/drills/:name` | 容灾演练 | `h.watchDrill` |
| `GET` | `/apis/disastergroups.testudo.softcdata.com/v1/watch/groups/operations` | 容灾组 | `h.watchGroupOperations` |
| `GET` | `/apis/disastergroups.testudo.softcdata.com/v1/watch/groups/operations/:operationName` | 容灾组 | `h.watchGroupOperation` |
| `GET` | `/apis/disastergroups.testudo.softcdata.com/v1/watch/groups/status` | 容灾组 | `h.watchGroupStatuses` |
| `GET` | `/apis/disastergroups.testudo.softcdata.com/v1/watch/groups/status/:name` | 容灾组 | `h.watchGroupStatus` |
| `GET` | `/apis/disasterconfigs.testudo.softcdata.com/v1/watch/configs` | 容灾配置 | `c.watchConfigs` |
| `GET` | `/apis/disasterconfigs.testudo.softcdata.com/v1/watch/configs/:name` | 容灾配置 | `c.watchConfig` |
| `GET` | `/apis/appbackups.testudo.softcdata.com/v1/watch/appbackups` | 应用备份 | `c.watchAppBackups` |
| `GET` | `/apis/appbackups.testudo.softcdata.com/v1/watch/appbackups/:name` | 应用备份 | `c.watchAppBackup` |
| `GET` | `/apis/apprestores.testudo.softcdata.com/v1/watch/apprestores` | 应用恢复 | `c.watchAppRestores` |
| `GET` | `/apis/apprestores.testudo.softcdata.com/v1/watch/apprestores/:name` | 应用恢复 | `c.watchAppRestore` |
| `GET` | `/apis/cluster.testudo.softcdata.com/v1/watch/clusters` | 集群管理 | `c.watchClusters` |
| `GET` | `/apis/cluster.testudo.softcdata.com/v1/watch/clusters/:name` | 集群管理 | `c.watchCluster` |
