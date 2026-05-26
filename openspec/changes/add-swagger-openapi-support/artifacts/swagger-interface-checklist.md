# Swagger/OpenAPI 逐接口勾选清单

> 2026-05-16 API 响应消息国际化备注：`disaster-server-openapi.yaml`
> 已补充共享语言请求头、WebSocket `lang` 参数、响应语言头、`message_key`
> 字段和中英文错误示例。鉴权错误 schema 已从旧 `code/msg` 更新为统一错误信封。

> 2026-05-15 CRD group 迁移备注：OpenAPI 路径已同步切换为
> `testudo.softcdata.com`，并通过本地 OpenAPI 校验。RunAPI 状态列保留的是
> 此前逐接口补充说明时的历史状态；Apipost MCP 已批量回写 live RunAPI 中
> `api` 类型目标的 URL 元数据，复查旧 group 后剩余项均为 `websocket2`
> 类型。当前 MCP `update_target` 对 `websocket2` 返回成功但不实际修改 URL，
> 这部分仍需通过 Apipost UI 或支持 websocket 目标的专用接口补迁移。

勾选规则：只有完成调用链取证、request schema、response schema、错误响应、扩展字段、Swagger UI 渲染检查后，才能将对应接口勾选为完成。

## Kubernetes 资源

- [x] `GET /apis/resources.testudo.softcdata.com/v1/:resource` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[无 operator 链路]
## 事件与历史

- [x] `GET /apis/v1/:resource/:name/history` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，读取 operator 写入的 Kubernetes Event]
- [x] `GET /apis/v1/events` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，读取 operator 写入的 Kubernetes Event]
- [x] `GET /apis/v1/watch/:resource/:name/events` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，实时监听 operator 写入的 Kubernetes Event]
- [x] `GET /apis/v1/watch/events` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，实时监听 operator 写入的 Kubernetes Event]
## 删除检查

- [x] `POST /api/v1/deletion/check` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，读取 operator 写入的依赖与 cleanup 标签]
- [x] `POST /apis/v1/deletion/check` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，读取 operator 写入的依赖与 cleanup 标签]
## 存储仓库

- [x] `GET /apis/storage.testudo.softcdata.com/v1/storages` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，读取 StorageRepository status，不触发 reconcile]
- [x] `POST /apis/storage.testudo.softcdata.com/v1/storages` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，创建 StorageRepository 后由 operator 校验 S3、创建 bucket、统计容量]
- [x] `DELETE /apis/storage.testudo.softcdata.com/v1/storages/:name` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，删除 StorageRepository 后由 operator handleDelete 上报事件并移除 finalizer]
- [x] `GET /apis/storage.testudo.softcdata.com/v1/storages/:name` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，读取 StorageRepository status，不触发 reconcile]
- [x] `PATCH /apis/storage.testudo.softcdata.com/v1/storages/:name` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，更新 StorageRepository spec 后由 operator 重新校验 S3 并回写状态]
- [x] `PUT /apis/storage.testudo.softcdata.com/v1/storages/:name` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，更新 StorageRepository spec 后由 operator 重新校验 S3 并回写状态]
- [x] `GET /apis/storage.testudo.softcdata.com/v1/storages/:name/validate` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，只读取 StorageRepository status.status]
- [x] `POST /apis/storage.testudo.softcdata.com/v1/storages/connectivity/validate` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，通过 Cluster ensure-storage 注解触发 operator 创建或更新 Velero BSL]
- [x] `GET /apis/storage.testudo.softcdata.com/v1/storages/names` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，读取 StorageRepository status，不触发 reconcile]
- [x] `POST /apis/storage.testudo.softcdata.com/v1/storages/validate/connection` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[无 operator 链路，server 直连 S3 校验请求参数]
- [x] `GET /apis/storage.testudo.softcdata.com/v1/watch/storages` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，监听 StorageRepository watch 事件，不触发 reconcile]
## 容灾任务

- [x] `GET /apis/disasterjobs.testudo.softcdata.com/v1/jobs` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，读取 DisasterJob status，不触发 reconcile]
- [x] `POST /apis/disasterjobs.testudo.softcdata.com/v1/jobs` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，创建 DisasterJob 后由 operator 创建或复用 Velero Backup/Restore]
- [x] `DELETE /apis/disasterjobs.testudo.softcdata.com/v1/jobs/:name` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，删除 DisasterJob 后由 finalizer 清理 Velero Backup/Restore]
- [x] `GET /apis/disasterjobs.testudo.softcdata.com/v1/jobs/:name` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，读取 DisasterJob status，不触发 reconcile]
- [x] `GET /apis/disasterjobs.testudo.softcdata.com/v1/watch/jobs` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，监听 DisasterJob watch 事件，不触发 reconcile]
- [x] `GET /apis/disasterjobs.testudo.softcdata.com/v1/watch/jobs/:name` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，按 fieldSelector 监听指定 DisasterJob watch 事件，不触发 reconcile]
## 容灾备份

- [x] `GET /apis/disasterbackups.testudo.softcdata.com/v1/backups` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，读取 DisasterBackup status，不触发 reconcile]
- [x] `POST /apis/disasterbackups.testudo.softcdata.com/v1/backups` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，创建 DisasterBackup 后由 operator 扫描源集群命名空间资源]
- [x] `DELETE /apis/disasterbackups.testudo.softcdata.com/v1/backups/:name` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，当前无 finalizer 清理链路，仅删除 DisasterBackup CRD]
- [x] `GET /apis/disasterbackups.testudo.softcdata.com/v1/backups/:name` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，读取 DisasterBackup status，不触发 reconcile]
- [x] `PUT /apis/disasterbackups.testudo.softcdata.com/v1/backups/:name` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，更新 DisasterBackup 后由 operator 重新扫描源集群命名空间资源]
- [x] `GET /apis/disasterbackups.testudo.softcdata.com/v1/watch/backups` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，监听 DisasterBackup watch 事件，不触发 reconcile]
- [x] `GET /apis/disasterbackups.testudo.softcdata.com/v1/watch/backups/:name` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，按 fieldSelector 监听指定 DisasterBackup watch 事件，不触发 reconcile]
## 容灾实例

- [x] `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances` - RunAPI：[已存在，本地证据已更新]；OpenAPI：[已补详细说明，2026-05-21 修正 `query.namespace` 为实例保护的业务 namespace 过滤，2026-05-25 改为支持 namespace 片段模糊搜索]；Schema：[已确认]；错误：[已确认]；operator：[已取证，读取 DisasterInstance/DataSync/ResourceSync/DisasterOperation 状态，不触发 reconcile]
- [x] `POST /apis/disasterinstances.testudo.softcdata.com/v1/instances` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，创建 DisasterInstance 后由 operator 创建 DataSync/ResourceSync 并推进 fsmState]
- [x] `DELETE /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，handleDeletion 当前只移除 finalizer，DataSync/ResourceSync 由 ownerReference GC]
- [x] `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，读取实例 fsmState/availableOperations/sync names，不触发 reconcile]
- [x] `PUT /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，更新实例后 operator 重新同步依赖标签和 DataSync/ResourceSync schedule]
- [x] `POST /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/actions` - RunAPI：[已存在，本地证据已更新，live 已补 202 示例]；OpenAPI：[已补详细说明，2026-05-22 补充 FailingOver 下 cancel 兼容语义]；Schema：[已确认]；错误：[已确认，FailingOver 下 cancel 不再返回 409]；operator：[已取证，创建 DisasterOperation 后由 operation controller 异步执行动作；2026-05-18 已补充同步失败 Failed 重试兼容语义；2026-05-22 已补充 FailingOver cancel 兼容语义]
- [x] `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/groups` - RunAPI：[已补缺失]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，只读 DisasterGroup.spec.levels，不触发 reconcile]
- [x] `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/history` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，读取 DisasterOperation status，不读取 DataSync/ResourceSync history]
- [x] `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/operations/:operationName` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，读取 DisasterOperation status 并校验归属]
- [x] `POST /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/restore-classes/validate` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，预检目标集群 StorageClass/IngressClass，后续 restorePolicy 由 operator 消费]
- [x] `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/sync-status` - RunAPI：[已存在，已补 lastSyncStatus]；OpenAPI：[已补详细说明]；Schema：[已确认，SubResourceStatusDTO.lastSyncStatus]；错误：[已确认]；operator：[已取证，读取 DataSync/ResourceSync/BackupRestoreStatistics 状态]
- [x] `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/sync-history` - RunAPI：[已新增，Target ID `1c8428770dc01001`]；OpenAPI：[已新增详细说明]；Schema：[已确认，SyncHistoryItemDTO/SyncHistoryCollectionEnvelope]；错误：[已确认，非法参数 400]；operator：[已取证，读取 DataSync/ResourceSync history 与同步类 DisasterOperation]
- [x] `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/validate-target` - RunAPI：[已补缺失，本地证据已更新，live 已补 200 示例并更新 operation query 说明]；OpenAPI：[已补详细说明，2026-05-22 补充 FailingOver 下 cancel 兼容语义]；Schema：[已确认]；错误：[已确认，FailingOver 下 cancel 返回 valid=true]；operator：[已取证，读取 availableOperations 与组归属，不创建 Operation；2026-05-18 已补充同步失败 Failed 重试兼容语义；2026-05-22 已补充 FailingOver cancel 兼容语义]
- [x] `GET /apis/disasterinstances.testudo.softcdata.com/v1/watch/instances` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，聚合监听 DisasterInstance/DataSync/ResourceSync watch 事件]
- [x] `GET /apis/disasterinstances.testudo.softcdata.com/v1/watch/instances/:name` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，按实例和固定同步子资源名称监听 watch 事件]
- [x] `GET /apis/disasterinstances.testudo.softcdata.com/v1/watch/instances/operations/:operationName` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，监听 DisasterOperation watch 事件并转换 OperationDetailDTO]
## 容灾演练

- [x] `GET /apis/disasterdrills.testudo.softcdata.com/v1/drills` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，只读 DisasterDrill status，不触发 reconcile]
- [x] `POST /apis/disasterdrills.testudo.softcdata.com/v1/drills` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，创建 DisasterDrill 后由 operator 前置校验并等待 confirm]
- [x] `DELETE /apis/disasterdrills.testudo.softcdata.com/v1/drills/:name` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，handleDeletion 只移除 finalizer，Operation 由 ownerReference GC]
- [x] `GET /apis/disasterdrills.testudo.softcdata.com/v1/drills/:name` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，读取 Drill status/validationResults/steps/groupProgress，不触发 reconcile]
- [x] `POST /apis/disasterdrills.testudo.softcdata.com/v1/drills/:name/cleanup` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，写 spec.cleanup 后由 operator 创建 drill-cleanup Operation]
- [x] `POST /apis/disasterdrills.testudo.softcdata.com/v1/drills/:name/confirm` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，写 spec.confirmed 后由 operator 创建 drill Operation]
- [x] `POST /apis/disasterdrills.testudo.softcdata.com/v1/drills/:name/restart` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，写 restart annotation 后由 operator 重置为 Pending]
- [x] `GET /apis/disasterdrills.testudo.softcdata.com/v1/drills/actions/protected-namespaces` - RunAPI：[已存在]；OpenAPI：[已补详细说明，2026-05-21 修正 namespace 为空时默认配置管理命名空间且不再全 namespace 回退]；Schema：[已确认]；错误：[已确认]；operator：[已取证，只读 DisasterInstance.spec.namespaces 和 DisasterGroup.spec.levels，不触发 reconcile]
- [x] `GET /apis/disasterdrills.testudo.softcdata.com/v1/watch/drills` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，监听全 namespace DisasterDrill watch 事件]
- [x] `GET /apis/disasterdrills.testudo.softcdata.com/v1/watch/drills/:name` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证，按名称监听 DisasterDrill watch 事件，findNamespace 失败不阻断连接]
## 容灾策略

- [x] `GET /apis/policies.testudo.softcdata.com/v1/policies` - RunAPI：[已存在]；OpenAPI：[已补]；Schema：[已确认]；错误：[已确认]；operator：[已取证]
- [x] `POST /apis/policies.testudo.softcdata.com/v1/policies` - RunAPI：[已存在]；OpenAPI：[已补]；Schema：[已确认]；错误：[已确认]；operator：[已取证]
- [x] `DELETE /apis/policies.testudo.softcdata.com/v1/policies/:name` - RunAPI：[已存在]；OpenAPI：[已补]；Schema：[已确认]；错误：[已确认]；operator：[已取证]
- [x] `GET /apis/policies.testudo.softcdata.com/v1/policies/:name` - RunAPI：[已存在]；OpenAPI：[已补]；Schema：[已确认]；错误：[已确认]；operator：[已取证]
- [x] `PUT /apis/policies.testudo.softcdata.com/v1/policies/:name` - RunAPI：[已存在]；OpenAPI：[已补]；Schema：[已确认]；错误：[已确认]；operator：[已取证]
- [x] `GET /apis/policies.testudo.softcdata.com/v1/policies/names` - RunAPI：[已存在]；OpenAPI：[已补]；Schema：[已确认]；错误：[已确认]；operator：[已取证]
## 容灾组

- [x] `GET /apis/disastergroups.testudo.softcdata.com/v1/groups` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证]
- [x] `POST /apis/disastergroups.testudo.softcdata.com/v1/groups` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证]
- [x] `DELETE /apis/disastergroups.testudo.softcdata.com/v1/groups/:name` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证]
- [x] `GET /apis/disastergroups.testudo.softcdata.com/v1/groups/:name` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证]
- [x] `PATCH /apis/disastergroups.testudo.softcdata.com/v1/groups/:name` - RunAPI：[已补缺失]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证]
- [x] `PUT /apis/disastergroups.testudo.softcdata.com/v1/groups/:name` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证]
- [x] `POST /apis/disastergroups.testudo.softcdata.com/v1/groups/:name/actions` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证]
- [x] `GET /apis/disastergroups.testudo.softcdata.com/v1/groups/:name/history` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证]
- [x] `GET /apis/disastergroups.testudo.softcdata.com/v1/groups/:name/instances` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证]
- [x] `GET /apis/disastergroups.testudo.softcdata.com/v1/groups/:name/operations/:operationName` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证]
- [x] `GET /apis/disastergroups.testudo.softcdata.com/v1/groups/instance-picker` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证]
- [x] `GET /apis/disastergroups.testudo.softcdata.com/v1/watch/groups/operations` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证]
- [x] `GET /apis/disastergroups.testudo.softcdata.com/v1/watch/groups/operations/:operationName` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证]
- [x] `GET /apis/disastergroups.testudo.softcdata.com/v1/watch/groups/status` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证]
- [x] `GET /apis/disastergroups.testudo.softcdata.com/v1/watch/groups/status/:name` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证]
## 容灾配置

- [x] `GET /apis/disasterconfigs.testudo.softcdata.com/v1/configs` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证]
- [x] `POST /apis/disasterconfigs.testudo.softcdata.com/v1/configs` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证]
- [x] `DELETE /apis/disasterconfigs.testudo.softcdata.com/v1/configs/:name` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证]
- [x] `GET /apis/disasterconfigs.testudo.softcdata.com/v1/configs/:name` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证]
- [x] `PUT /apis/disasterconfigs.testudo.softcdata.com/v1/configs/:name` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证]
- [x] `GET /apis/disasterconfigs.testudo.softcdata.com/v1/configs/names` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证]
- [x] `GET /apis/disasterconfigs.testudo.softcdata.com/v1/watch/configs` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证]
- [x] `GET /apis/disasterconfigs.testudo.softcdata.com/v1/watch/configs/:name` - RunAPI：[已存在]；OpenAPI：[已补详细说明]；Schema：[已确认]；错误：[已确认]；operator：[已取证]
## 应用备份

- [ ] `GET /apis/appbackups.testudo.softcdata.com/v1/appbackups` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `POST /apis/appbackups.testudo.softcdata.com/v1/appbackups` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `DELETE /apis/appbackups.testudo.softcdata.com/v1/appbackups/:name` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /apis/appbackups.testudo.softcdata.com/v1/appbackups/:name` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `PUT /apis/appbackups.testudo.softcdata.com/v1/appbackups/:name` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `POST /apis/appbackups.testudo.softcdata.com/v1/appbackups/:name/actions/:type` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /apis/appbackups.testudo.softcdata.com/v1/appbackups/:name/backups/:backupName/download` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /apis/appbackups.testudo.softcdata.com/v1/appbackups/:name/history` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /apis/appbackups.testudo.softcdata.com/v1/appbackups/clusters` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /apis/appbackups.testudo.softcdata.com/v1/velero/backups/:backupName/includes` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /apis/appbackups.testudo.softcdata.com/v1/watch/appbackups` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /apis/appbackups.testudo.softcdata.com/v1/watch/appbackups/:name` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
## 应用恢复

- [ ] `GET /apis/apprestores.testudo.softcdata.com/v1/apprestores` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `POST /apis/apprestores.testudo.softcdata.com/v1/apprestores` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `DELETE /apis/apprestores.testudo.softcdata.com/v1/apprestores/:name` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /apis/apprestores.testudo.softcdata.com/v1/apprestores/:name` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `PUT /apis/apprestores.testudo.softcdata.com/v1/apprestores/:name` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `POST /apis/apprestores.testudo.softcdata.com/v1/apprestores/:name/actions/:type` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `POST /apis/apprestores.testudo.softcdata.com/v1/apprestores/preflight/validate` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /apis/apprestores.testudo.softcdata.com/v1/watch/apprestores` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /apis/apprestores.testudo.softcdata.com/v1/watch/apprestores/:name` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
## 根接口

- [ ] `GET /healthz` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `POST /login` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /readyz` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `POST /refresh_token` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
## 用户管理

- [ ] `GET /api/v1/users` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `POST /api/v1/users` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `DELETE /api/v1/users/:username` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `PATCH /api/v1/users/:username/password` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `PATCH /api/v1/users/:username/status` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /apis/v1/users` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `POST /apis/v1/users` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `DELETE /apis/v1/users/:username` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `PATCH /apis/v1/users/:username/password` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `PATCH /apis/v1/users/:username/status` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
## 系统设置

- [ ] `GET /api/v1/system-settings` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `POST /api/v1/system-settings` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `DELETE /api/v1/system-settings/:config_key` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `PUT /api/v1/system-settings/:config_key` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /api/v1/system-settings/assets/:config_key` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `POST /api/v1/system-settings/assets/:config_key` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /api/v1/system-settings/public` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /apis/v1/system-settings` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `POST /apis/v1/system-settings` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `DELETE /apis/v1/system-settings/:config_key` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `PUT /apis/v1/system-settings/:config_key` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /apis/v1/system-settings/assets/:config_key` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `POST /apis/v1/system-settings/assets/:config_key` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /apis/v1/system-settings/public` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
## 统计

- [ ] `GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/autobackups/execution-summary` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/backups` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/backups/success-rate` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/instances` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/operations` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/operations/by-time` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/restores` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/storage` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/tasks/progress` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
## 集群管理

- [ ] `GET /apis/cluster.testudo.softcdata.com/v1/clusters` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `POST /apis/cluster.testudo.softcdata.com/v1/clusters` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `DELETE /apis/cluster.testudo.softcdata.com/v1/clusters/:name` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /apis/cluster.testudo.softcdata.com/v1/clusters/:name` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `PATCH /apis/cluster.testudo.softcdata.com/v1/clusters/:name` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `POST /apis/cluster.testudo.softcdata.com/v1/clusters/:name/actions/refresh-namespaces` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /apis/cluster.testudo.softcdata.com/v1/clusters/:name/protected-namespaces` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /apis/cluster.testudo.softcdata.com/v1/clusters/:name/restore-classes` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /apis/cluster.testudo.softcdata.com/v1/clusters/:name/validate` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `POST /apis/cluster.testudo.softcdata.com/v1/clusters/kubeconfig/validate` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /apis/cluster.testudo.softcdata.com/v1/clusters/names` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /apis/cluster.testudo.softcdata.com/v1/watch/clusters` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]
- [ ] `GET /apis/cluster.testudo.softcdata.com/v1/watch/clusters/:name` - RunAPI：[已存在]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]

## 平台许可

- [x] `GET /apis/v1/platform-license/status` - RunAPI：[已存在，已补当前指纹示例并回读验证]；OpenAPI：[已补详细说明]；Schema：[PlatformLicenseStatus 新增 fingerprint/fingerprintVersion/fingerprintRequest]；错误：[已区分 License 内容错误与部署环境错误]；operator：[复用共享 verifier 与 KubernetesStore.Fingerprint]
- [x] `POST /apis/v1/platform-license/install` - RunAPI：[已存在，已补安装响应当前指纹示例并回读验证]；OpenAPI：[已补详细说明]；Schema：[PlatformLicenseInstallRequest/PlatformLicenseStatus 已确认，安装响应同步当前指纹字段]；错误：[已说明写入后 liveEvaluation 不返回旧 NoLicense，内容错误优先于指纹环境错误]；operator：[写入 License Secret 后由 operator 刷新 status CM]

## 全局响应约定

- [x] `server DTO 时间字段本地时区序列化` - RunAPI：[已新增 Apipost Markdown 说明文档，Target ID `1c8551dde7001000`]；OpenAPI：[已补 LocalRFC3339Time 共享说明和 info.description]；Schema：[所有响应 DTO 时间字段按 RFC3339 offset 字符串表达，容器 TZ=Asia/Shanghai 时为 +08:00]；错误：[不涉及]；operator：[不改 CRD/YAML UTC 存储]
- [x] `管理面 namespace 可配置` - OpenAPI：[已将固定 `disaster-system` 资源范围描述改为配置的管理命名空间（默认 `disaster-system`）]；Schema：[不改变请求/响应结构]；行为：[server/operator/chart 统一使用安装 namespace 作为管理面 namespace]
