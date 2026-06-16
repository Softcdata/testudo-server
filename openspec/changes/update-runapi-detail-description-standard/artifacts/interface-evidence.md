# 单接口取证记录

> 2026-05-16 API 响应消息国际化备注：服务端新增 `X-Language`
> 与 `Accept-Language` 语言协商，响应头返回 `Content-Language` 和
> `Vary: X-Language, Accept-Language`。统一错误信封新增稳定字段
> `message_key`，JWT、刷新令牌、Recovery、WebSocket 与首批业务模块错误
> 已迁移。Apipost 项目 `5650333c5c52000` 已新增共享 Markdown 文档
> `API 响应消息国际化协议`，Target ID `1c8686a538401000`；`POST /login`
> 与 `POST /refresh_token` 已更新统一信封说明、新增 `X-Language` 请求头和当前错误响应示例。
> 后续逐接口详情补写时继续保留原有说明。

> 2026-05-15 CRD group 迁移备注：server 路由 namespace 已从
> `disaster.wuxs.vip` 断代切换为 `testudo.softcdata.com`，本地 evidence
> 中的路径已同步替换。Apipost MCP 已批量回写 live RunAPI 中 `api` 类型目标的
> URL 元数据；复查旧 group 后剩余项均为 `websocket2` 类型。当前 MCP
> `update_target` 对 `websocket2` 返回成功但不实际修改 URL，这部分仍需通过
> Apipost UI 或支持 websocket 目标的专用接口补迁移。本文件下方“已回读验证”
> 记录保留的是此前接口详情补充时的历史状态。

## GET /apis/resources.testudo.softcdata.com/v1/:resource

- RunAPI Target ID：`25559f2338c0c8`
- RunAPI 状态：已存在，已更新详细说明，已回读验证。
- server 路由：`internal/apis/kubernetes_resources/router.go`
- server handler：`internal/apis/kubernetes_resources/handler.go`，`KubernetesResourcesHandler.getResources`
- 请求链路：`path.resource` -> `k.resources[resource]` -> `Resources.List(query.namespace)`
- 当前资源映射：`internal/apis/kubernetes_resources/resources/resources.go` 只注册 `namespaces`
- 具体资源实现：`internal/apis/kubernetes_resources/resources/namespaces/namespace.go`，读取 `corev1.NamespaceList`
- operator 链路：无；不创建、不更新 disaster-operator CRD，不触发 controller reconcile
- 下层资源：Kubernetes `Namespace`
- 已写入内容：五段详细说明、`namespace` query 参数说明、`400` Kubernetes List 失败响应示例
- 主要错误点：Kubernetes List 失败返回 `400 {"message": err}`；JWT 失败由中间件返回；非 `namespaces` resource 当前会触发 nil handler 并由 recovery 处理为内部错误

## GET /apis/v1/:resource/:name/history

- RunAPI Target ID：`1be82b9e38801001`
- RunAPI 状态：已存在，已更新详细说明，已回读验证；URL 已从样例路径改为 `{{baseurl}}/apis/v1/:resource/:name/history`
- server 路由：`internal/apis/event/v1/router.go`
- server handler：`internal/apis/event/v1/list.go`，`EventHandler.listResourceEvents`
- 请求链路：`path.resource` -> `resolveEventResourceKind` -> `path.name` 组成 `fieldSelector: involvedObject.name=<name>` -> `CoreV1().Events(namespace).List`
- 过滤链路：固定 `LabelSelector: testudo.softcdata.com/task-event=true`；再按 `involvedObject.kind` 过滤；再按 `startTime`/`endTime` 过滤；最后内存分页
- 返回转换：`aggregateEvents` 将 Kubernetes Event JSON message 聚合为 `TaskEvent`，再由 `transport.BuildCollectionResponse` 和 `transport.WriteSuccess` 包成统一 envelope
- operator 链路：接口本身不触发 reconcile；读取 disaster-operator 通过 `pkg/helper/event_reporter.go` 写入的结构化 Kubernetes Event
- 下层资源：Kubernetes `Event`
- 已写入内容：五段详细说明、当前支持的 `resource` 别名、query 参数、统一 envelope 返回字段、当前响应示例、`400` 不支持资源示例，原说明保留到 `## 原有说明`
- 主要错误点：不支持资源返回 `400 code=1000`；Events List 失败返回 `500 code=5000`；JWT 失败由中间件返回；operator 写入的 `errorCode` 当前 server 聚合 DTO 未映射为返回字段

## GET /apis/v1/events

- RunAPI Target ID：`1be82b9e13c01001`
- RunAPI 状态：已存在，已更新详细说明，已回读验证；URL 已从带查询串样例改为 `{{baseurl}}/apis/v1/events`
- server 路由：`internal/apis/event/v1/router.go`
- server handler：`internal/apis/event/v1/list.go`，`EventHandler.listEvents`
- 请求链路：`query.namespace` -> `CoreV1().Events(namespace).List`，固定 `LabelSelector: testudo.softcdata.com/task-event=true`
- 过滤链路：`aggregateEvents` 后按 `taskType` 包含匹配、`status` 精确匹配、`ownerUID` 精确匹配、`ownerName` 包含匹配、`traceId` 精确匹配、`startTime`/`endTime` 时间范围、`keyword` 包含匹配 `taskName/cluster/namespace/traceId`
- 返回转换：`aggregateEvents` -> 固定按 `TaskEvent.time` 倒序 -> `transport.Paginate` -> `transport.BuildCollectionResponse` -> `transport.WriteSuccess`
- operator 链路：接口本身不触发 reconcile；读取 disaster-operator 通过 `pkg/helper/event_reporter.go` 写入的结构化 Kubernetes Event
- 下层资源：Kubernetes `Event`
- 已写入内容：五段详细说明、查询参数修正、统一 envelope 返回字段、当前响应示例，原说明保留到 `## 原有说明`
- 主要错误点：Events List 失败返回 `500 code=5000`；JWT 失败由中间件返回；RunAPI 原有 `errorCode` 查询参数与当前 server 代码不一致，已删除；operator 写入的 `errorCode` 当前 server 聚合 DTO 未映射为返回字段

## GET /apis/v1/watch/:resource/:name/events

- RunAPI Target ID：`1c7aff33ad001001`
- RunAPI 状态：缺失，已新增到 `容灾云平台 / V1 / 历史事件`，已回读验证
- server 路由：`internal/apis/event/v1/router.go`
- server handler：`internal/apis/event/v1/handler.go`，`EventHandler.watchResourceEvents`
- 请求链路：`path.resource` -> `resolveEventResourceKind`；`path.name` -> `fieldSelector: involvedObject.name=<name>`；`query.origin` -> `buildTaskEventLabelSelector`
- 初始化链路：先 `CoreV1().Events(namespace).List` 获取最新 `resourceVersion`，避免推送历史全量事件
- Watch 链路：`CoreV1().Events(namespace).Watch` -> `StreamWatch` -> `ConvertToTaskEventDTO` -> WebSocket envelope
- operator 链路：接口本身不触发 reconcile；实时监听 disaster-operator 通过 `pkg/helper/event_reporter.go` 写入的结构化 Kubernetes Event
- 下层资源：Kubernetes `Event`
- 已写入内容：五段详细说明、query/header 参数、连接成功示例、事件推送示例、心跳示例、`400` origin 非法示例
- 主要错误点：resource/name 缺失或 resource 不支持返回 `400 code=1000`；origin 非法返回 `400 code=1000`；初始化 List 失败返回 `500 code=5000`；WebSocket 升级失败返回 HTTP `500 {"message": ...}`；watcher 创建失败通过 WebSocket 返回 `code=5000`

## GET /apis/v1/watch/events

- RunAPI Target ID：`2bc6c1dd38c073`
- RunAPI 状态：已存在，已更新详细说明，已回读验证；该目标类型为 `websocket2`，MCP 常规 update 返回成功但不落库，已用底层 Apipost OpenAPI 保留 `target_type=websocket2` 更新
- server 路由：`internal/apis/event/v1/router.go`
- server handler：`internal/apis/event/v1/handler.go`，`EventHandler.watchEvents`
- 请求链路：`query.namespace` 默认 `disaster-system`；`query.origin` -> `buildTaskEventLabelSelector`
- 初始化链路：先 `CoreV1().Events(namespace).List` 获取最新 `resourceVersion`，避免推送历史全量事件
- Watch 链路：`CoreV1().Events(namespace).Watch` -> `StreamWatch` -> `ConvertToTaskEventDTO` -> WebSocket envelope
- operator 链路：接口本身不触发 reconcile；实时监听 disaster-operator 通过 `pkg/helper/event_reporter.go` 写入的结构化 Kubernetes Event
- 下层资源：Kubernetes `Event`
- 已写入内容：五段详细说明、query/header 参数、WebSocket 推送帧字段说明；`websocket2` 详情无 response 容器，响应示例未能作为 response example 持久化
- 主要错误点：origin 非法返回 `400 code=1000`；初始化 List 失败返回 `500 code=5000`；WebSocket 升级失败返回 HTTP `500 {"message": ...}`；watcher 创建失败通过 WebSocket 返回 `code=5000`

## POST /api/v1/deletion/check

- RunAPI Target ID：`1c2f51b2b9801001`
- RunAPI 状态：已存在，已更新详细说明，已回读验证
- server 路由：`internal/apis/deletion_check/v1/router.go`
- server handler：`internal/apis/deletion_check/v1/handler.go`，`DeletionCheckHandler.check`
- 请求链路：`BindJSON` -> `normalizeKind` -> `getObjectByKind` -> `buildCleanupPlan` -> `queryUpstreamByLabelKey` -> `queryDownstreamByLabels` -> `canDeleteFromUpstreamCount`
- operator 链路：接口本身不触发 reconcile；读取 disaster-operator 已写入的 `dependency-token`、`dependency-to-*`、cleanup labels 和 ownerReferences。operator 对应基础能力在 `pkg/metadata/dependency_labels.go`、`pkg/metadata/dependency_query.go`
- 下层资源：disaster CRDs；AppBackup/AppRestore 场景会读取远端 Velero `Schedule`、`Backup`、`Restore` 和 resource modifier `ConfigMap`；AppBackup 还会读取 `BackupRestoreStatistics`
- 已写入内容：五段详细说明、完整 request/response 字段说明、cleanup_plan 字段、错误说明、当前成功示例，原说明保留到 `## 原有说明`
- 主要错误点：JSON 绑定失败/资源类型不支持/name 为空返回 `400 code=1000`；目标不存在返回 `404 code=3004`；目标读取失败或 token 推导失败返回 `500 code=5000`；cleanup plan 的远端失败通常作为 `resolved=false` 与 `unresolved_reason` 返回，不使整个接口失败

## POST /apis/v1/deletion/check

- RunAPI Target ID：`1c7b02d04a401001`
- RunAPI 状态：缺失，已新增到 `容灾云平台 / V1 / 统一删除保护`，已补充请求头并回读验证
- server 路由：`internal/apis/deletion_check/v1/router.go`
- server handler：`internal/apis/deletion_check/v1/handler.go`，`DeletionCheckHandler.check`
- 请求链路：与 `/api/v1/deletion/check` 完全一致，`/apis` 兼容路径注册到同一个 handler；`BindJSON` -> `normalizeKind` -> `getObjectByKind` -> `buildCleanupPlan` -> `queryUpstreamByLabelKey` -> `queryDownstreamByLabels` -> `canDeleteFromUpstreamCount`
- operator 链路：接口本身不触发 reconcile；读取 disaster-operator 已写入的 `dependency-token`、`dependency-to-*`、cleanup labels 和 ownerReferences。operator 对应基础能力在 `pkg/metadata/dependency_labels.go`、`pkg/metadata/dependency_query.go`
- 下层资源：disaster CRDs；AppBackup/AppRestore 场景会读取远端 Velero `Schedule`、`Backup`、`Restore` 和 resource modifier `ConfigMap`；AppBackup 还会读取 `BackupRestoreStatistics`
- 已写入内容：五段详细说明、完整 request/response 字段说明、cleanup_plan 字段、错误说明、当前成功示例、`Authorization` 和 `Content-Type` header
- 主要错误点：JSON 绑定失败/资源类型不支持/name 为空返回 `400 code=1000`；目标不存在返回 `404 code=3004`；目标读取失败或 token 推导失败返回 `500 code=5000`；cleanup plan 的远端失败通常作为 `resolved=false` 与 `unresolved_reason` 返回，不使整个接口失败

## GET /apis/storage.testudo.softcdata.com/v1/storages

- RunAPI Target ID：`25559f2238c0b6`
- RunAPI 状态：已存在，已更新详细说明，URL 已规范为不带样例查询串的路径，已补充 `keyword` query 参数和当前 200/400 响应示例，已回读验证
- server 路由：`internal/apis/disaster_storage/v1/router.go`
- server handler：`internal/apis/disaster_storage/v1/handler.go`，`StorageHandler.storages`
- 请求链路：`transport.ParseOptions` -> `transport.BuildLabelSelector` -> `StorageRepositoryLister.StorageRepositories(disaster-system).List` -> 额外 query 按 label value 模糊过滤 -> 按 `name` 或 `creationTimestamp` 内存排序 -> `transport.Paginate` -> `ConvertToDisasterStorageDTO` -> `transport.BuildCollectionResponse` -> `transport.WriteSuccess`
- operator 链路：接口本身不触发 reconcile；读取 `StorageRepositoryReconciler` 已写入的 `status`、`lastCheckTime`、`reason`、`message`、`usedSpaceBytes`、`totalBackupCount`。operator 在 reconcile 中执行 `ValidateS3Configuration`，读取 CA `Secret`，访问 S3 bucket，必要时创建 bucket，并统计对象使用量和备份数量
- 下层资源：`StorageRepository` CRD、同命名空间 CA `Secret`、S3 bucket/object；AppBackup/AppRestore 调用链会基于该仓库在目标集群创建或更新 Velero `Secret` 和 `BackupStorageLocation`
- 已写入内容：五段详细说明、查询参数说明、统一 collection envelope 返回字段、当前成功示例、列表读取失败示例
- 主要错误点：Lister 列表读取失败返回 `400 code=1000`；JWT 失败由中间件返回；`keyword` 当前解析但未参与过滤；不支持的 `sort` 字段不会报错，只会等同不排序

## POST /apis/storage.testudo.softcdata.com/v1/storages

- RunAPI Target ID：`3ee6850638c06f`
- RunAPI 状态：已存在，已更新详细说明，已保留原说明，已补齐 raw JSON body 示例/schema/header 和当前 201/400/409 响应示例，已回读验证
- server 路由：`internal/apis/disaster_storage/v1/router.go`
- server handler：`internal/apis/disaster_storage/v1/handler.go`，`StorageHandler.createStorage`
- 请求链路：`BindJSON(CreateDisasterStorageRequest)` -> `validateAddressingStyle` -> `validateCAWriteRequest` -> endpoint scheme 校验 -> 可选检查 `caSecretRef` Secret 存在 -> 可选 `upsertManagedCASecret` 创建或更新 `storage-ca-<name>` -> 注入 trace/user annotation -> `StorageRepositories(disaster-system).Create` -> `ConvertToDisasterStorageDTO` -> `transport.WriteSuccess(201)`
- operator 链路：创建 `StorageRepository` 后触发 `StorageRepositoryReconciler`；operator 同步 `dependency-token` label、添加 `storage-finalizer`、上报创建事件、执行 `ValidateS3Configuration`，读取 CA Secret，访问 S3 bucket，必要时创建 bucket，成功写 `Available`，失败写 `Unavailable`，并统计 `usedSpaceBytes` 和 `totalBackupCount`
- 下层资源：`StorageRepository` CRD、可选托管 CA `Secret`、S3 bucket/object；后续 AppBackup/AppRestore 会基于该仓库在目标集群创建或更新 Velero `Secret` 和 `BackupStorageLocation`
- 已写入内容：五段详细说明、完整 request 字段说明、敏感字段不返回说明、异步 operator 状态说明、错误说明、当前响应示例，原说明保留到 `## 原有说明`
- 主要错误点：JSON/binding 失败、`addressingStyle` 非法、CA 参数互斥、CA Secret 不存在、endpoint scheme 非法返回 `400 code=1000`；同名资源返回 `409 code=3009`；托管 CA Secret 创建/更新失败或 CRD 创建失败返回 `500 code=5000`

## DELETE /apis/storage.testudo.softcdata.com/v1/storages/:name

- RunAPI Target ID：`25559f2338c0c2`
- RunAPI 状态：已存在，已更新详细说明，URL 已从样例路径改为 `{{baseurl}}/apis/storage.testudo.softcdata.com/v1/storages/:name`，已补充 path 参数和当前 200/404/500 响应示例，已回读验证
- server 路由：`internal/apis/disaster_storage/v1/router.go`
- server handler：`internal/apis/disaster_storage/v1/handler.go`，`StorageHandler.deleteStorage`
- 请求链路：`path.name` -> 尽力 `Get` 目标 `StorageRepository` -> 尽力写入 trace/user annotation -> 若引用 server 托管 CA Secret 则尽力删除 `storage-ca-<name>` -> `StorageRepositories(disaster-system).Delete` -> `transport.WriteSuccess(200, {"name": name})`
- operator 链路：删除 CRD 后触发 `StorageRepositoryReconciler.handleDelete`；当前旧的 DisasterPolicy 引用拦截逻辑已注释旁路，operator 上报删除开始/完成事件后移除 `testudo.softcdata.com/storage-finalizer`
- 下层资源：直接删除 `StorageRepository` CRD；可能删除 server 托管 CA `Secret`；不会直接删除 S3 bucket/object、Velero `BackupStorageLocation` 或 Velero credential `Secret`
- 已写入内容：五段详细说明、path/header 参数说明、当前成功示例、NotFound 示例、删除失败示例
- 主要错误点：目标不存在返回 `404 code=3004`；Kubernetes 删除失败返回 `500 code=5000`；删除前 annotation 更新失败和托管 CA Secret 删除失败当前被忽略

## GET /apis/storage.testudo.softcdata.com/v1/storages/:name

- RunAPI Target ID：`25559f2238c0b8`
- RunAPI 状态：已存在，已更新详细说明，URL 已从样例路径改为 `{{baseurl}}/apis/storage.testudo.softcdata.com/v1/storages/:name`，已补充 path 参数和当前 200/404/500 响应示例，已回读验证
- 对账备注：清单原候选 `1be2230920001001` 回读后确认为 `GET /apis/storage.testudo.softcdata.com/v1/storages/names`，不属于详情接口
- server 路由：`internal/apis/disaster_storage/v1/router.go`
- server handler：`internal/apis/disaster_storage/v1/handler.go`，`StorageHandler.storage`
- 请求链路：`path.name` -> `DisasterClient.DisasterV1().StorageRepositories(disaster-system).Get` -> `ConvertToDisasterStorageDTO` -> `transport.WriteSuccess(200)`
- operator 链路：接口本身不触发 reconcile；读取 `StorageRepositoryReconciler` 已写入的 status、S3 校验结果、容量和备份数量统计
- 下层资源：`StorageRepository` CRD；状态来源间接涉及 CA `Secret`、S3 bucket/object、后续 Velero `Secret` 和 `BackupStorageLocation`
- 已写入内容：五段详细说明、path/header 参数说明、详情 DTO 字段、当前成功示例、NotFound 示例、读取失败示例
- 主要错误点：目标不存在返回 `404 code=3004`；Kubernetes 读取失败返回 `500 code=5000`；JWT 失败由中间件返回

## PATCH /apis/storage.testudo.softcdata.com/v1/storages/:name

- RunAPI Target ID：`1bd50e8b7d801000`
- RunAPI 状态：已存在，已更新详细说明，URL 已从样例路径改为 `{{baseurl}}/apis/storage.testudo.softcdata.com/v1/storages/:name`，已补齐 raw JSON body/schema/path/header 和当前 200/400/404/500 响应示例，已回读验证
- server 路由：`internal/apis/disaster_storage/v1/router.go`
- server handler：`internal/apis/disaster_storage/v1/handler.go`，`StorageHandler.patchStorage`
- 请求链路：`path.name` -> `BindJSON(PatchStorageRepositoryRequest)` -> `validateAddressingStyle` -> `validateCAWriteRequest` -> `Get` 现有 `StorageRepository` -> 按非 nil 字段局部更新 `accessKey/secretKey/bucket/region/addressingStyle/CA` -> 可选创建/更新/删除托管 CA Secret -> 实际更新时注入 trace/user annotation -> `StorageRepositories(disaster-system).Update` -> `ConvertToDisasterStorageDTO` -> `transport.WriteSuccess(200)`
- operator 链路：CRD spec 更新后触发 `StorageRepositoryReconciler`；operator 上报“编辑存储”事件，重新执行 S3 校验，更新可用状态、原因、消息、最近检查时间、容量和备份数量统计
- 下层资源：`StorageRepository` CRD、可选托管 CA `Secret`、S3 bucket/object；后续 Velero `Secret` 和 `BackupStorageLocation` 使用更新后的凭据和配置
- 已写入内容：五段详细说明、局部更新字段范围说明、CA 参数互斥说明、敏感字段不返回说明、当前响应示例，原说明保留到 `## 原有说明`
- 主要错误点：JSON/binding 失败、`addressingStyle` 非法、CA 参数互斥、CA Secret 不存在返回 `400 code=1000`；目标不存在返回 `404 code=3004`；CRD 或托管 CA Secret 更新失败返回 `500 code=5000`

## PUT /apis/storage.testudo.softcdata.com/v1/storages/:name

- RunAPI Target ID：`3ee6850678c071`
- RunAPI 状态：已存在，已更新详细说明，URL 已从样例路径改为 `{{baseurl}}/apis/storage.testudo.softcdata.com/v1/storages/:name`，已补齐 raw JSON body/schema/path/header 和当前 200/400/404/409/500 响应示例，已回读验证
- server 路由：`internal/apis/disaster_storage/v1/router.go`
- server handler：`internal/apis/disaster_storage/v1/handler.go`，`StorageHandler.updateStorage`
- 请求链路：`BindJSON(UpdateDisasterStorageRequest)` -> `validateAddressingStyle` -> `validateCAWriteRequest` -> 可选 endpoint scheme 校验 -> `RetryOnConflict` -> 按 `body.name` `Get` 现有 `StorageRepository` -> 可选检查 CA Secret -> `MergeToCRD` 非空字段覆盖 -> 可选创建/更新/删除托管 CA Secret -> 补 `testudo.softcdata.com/storage=true` label -> 注入 trace/user annotation -> `Update` -> `ConvertToDisasterStorageDTO` -> `transport.WriteSuccess(200)`
- 关键差异：path `:name` 当前只用于路由匹配，handler 实际使用 `body.name` 定位资源；文档已要求两者保持一致
- operator 链路：CRD spec 更新后触发 `StorageRepositoryReconciler`；operator 上报“编辑存储”事件，重新执行 S3 校验，更新可用状态、原因、消息、最近检查时间、容量和备份数量统计
- 下层资源：`StorageRepository` CRD、可选托管 CA `Secret`、S3 bucket/object；后续 Velero `Secret` 和 `BackupStorageLocation` 使用更新后的配置
- 已写入内容：五段详细说明、body/path 差异说明、冲突重试说明、CA 参数互斥说明、敏感字段不返回说明，原说明保留到 `## 原有说明`
- 主要错误点：JSON/binding 失败、`addressingStyle` 非法、CA 参数互斥、endpoint scheme 非法、CA Secret 不存在返回 `400 code=1000`；`body.name` 目标不存在返回 `404 code=3004`；多次冲突返回 `409 code=3009`；CRD 或托管 CA Secret 操作失败返回 `500 code=5000`

## GET /apis/storage.testudo.softcdata.com/v1/storages/:name/validate

- RunAPI Target ID：`1bdf92f4e4801000`
- RunAPI 状态：已存在，已更新详细说明，URL 已从样例路径改为 `{{baseurl}}/apis/storage.testudo.softcdata.com/v1/storages/:name/validate`，已补充 path/header 参数和当前 true/false/404/500 响应示例，已回读验证；auth 已从历史 `noauth` 修正为继承项目鉴权
- server 路由：`internal/apis/disaster_storage/v1/router.go`
- server handler：`internal/apis/disaster_storage/v1/handler.go`，`StorageHandler.validateStorage`
- 请求链路：`path.name` -> `DisasterClient.DisasterV1().StorageRepositories(disaster-system).Get` -> 判断 `item.Status.Status == StorageRepositoryStatusAvailable` -> `transport.WriteSuccess(200, bool)`
- operator 链路：接口本身不触发 reconcile，也不主动连接 S3；仅读取 `StorageRepositoryReconciler` 最近一次校验后写入的 `status.status`
- 下层资源：直接读取 `StorageRepository` CRD；不直接访问 CA Secret、S3 bucket/object 或 Velero BSL
- 已写入内容：五段详细说明、只读状态判定说明、path/header 参数说明、当前可用/不可用示例、NotFound 示例、读取失败示例
- 主要错误点：目标不存在返回 `404 code=3004`；Kubernetes 读取失败返回 `500 code=5000`；JWT 失败由中间件返回

## POST /apis/storage.testudo.softcdata.com/v1/storages/connectivity/validate

- RunAPI Target ID：`1bfa8b208a801001`
- RunAPI 状态：已存在，已更新详细说明，已补齐 raw JSON body/schema/header 和当前 200/400/500 响应示例，auth 已从历史 `noauth` 修正为继承项目鉴权，已回读验证
- server 路由：`internal/apis/disaster_storage/v1/router.go`
- server handler：`internal/apis/disaster_storage/v1/handler.go`，`StorageHandler.validateBSLConnectivity`
- 请求链路：`BindJSON(ValidateConnectivityRequest)` -> `GetKubeClient(cluster_name)` -> `BSLVerifier.VerifyBSL(targetClient, mgmtClient, storage_name, cluster_name)` -> `transport.WriteSuccess(200, bool, meta)`
- verifier 链路：按 `<storage_name>-<cluster_name>` 读取目标集群 `velero` 命名空间 BSL；若 phase 为 `Available` 返回 true；若 phase 非空但不是 Available 返回 false；若 BSL 不存在，则 patch 管理集群 `Cluster` annotation `testudo.softcdata.com/ensure-storage=<storage_name>`，随后每 500ms 轮询一次，最多 20 次
- operator 链路：`ClusterReconciler` 监听 `ensure-storage` 注解，读取 `StorageRepository`，调用 `DefaultBSL.ApplyStorageRepository` 在目标集群创建或更新 Velero credential `Secret` 和 `BackupStorageLocation`，然后移除注解
- 下层资源：管理集群 `Cluster` CR、`StorageRepository` CR；目标集群 Velero `BackupStorageLocation`、Velero credential `Secret` 和 S3 bucket
- 已写入内容：五段详细说明、当前实现与旧设计稿差异、body/header 参数、`data` 布尔值与 `meta.message/error` 返回说明，原说明保留到 `## 原有说明`
- 主要错误点：JSON/binding 失败或目标集群 client 获取失败返回 `400 code=1000`；读取 BSL 非 NotFound 错误、读取 Cluster 失败、patch ensure-storage 失败返回 `500 code=5000`；BSL phase 不可用或等待超时返回 `200 code=0 data=false`

## GET /apis/storage.testudo.softcdata.com/v1/storages/names

- RunAPI Target ID：`1be2230920001001`
- RunAPI 状态：已存在，已更新详细说明，已补齐 `Authorization` header 和当前 200/400 响应示例，auth 已从历史 `noauth` 修正为继承项目鉴权，已回读验证
- server 路由：`internal/apis/disaster_storage/v1/router.go`
- server handler：`internal/apis/disaster_storage/v1/handler.go`，`StorageHandler.storageNames`
- 请求链路：`transport.ParseOptions` -> `transport.BuildLabelSelector` 固定返回空 selector -> `StorageRepositoryLister.StorageRepositories(disaster-system).List` -> 额外 query 按 `metadata.labels` value 包含匹配 -> 组装 `[]DisasterStorageNameDTO` -> `transport.WriteSuccess(200, dtos, nil)`
- operator 链路：接口本身不触发 reconcile；仅读取 `StorageRepositoryReconciler` 已写入的 `status.status`
- 下层资源：直接读取 `StorageRepository` CRD informer 缓存；不访问 S3、CA Secret、Velero `Secret` 或 Velero `BackupStorageLocation`
- 已写入内容：五段详细说明、轻量名称列表语义、动态 label 过滤说明、`limit/page/sort/order/keyword` 当前不生效说明、返回字段来源、当前成功和列表读取失败示例，原说明保留到 `## 原有说明`
- 主要错误点：Lister 列表读取失败返回 `400 code=1000`；JWT 失败由中间件返回；operator 的 S3/CA/bucket 校验失败不导致该接口失败，只表现为 `data[].status=Unavailable`

## POST /apis/storage.testudo.softcdata.com/v1/storages/validate/connection

- RunAPI Target ID：`6c722f5f8c74b`
- RunAPI 状态：已存在，已更新详细说明，已修正请求体为合法 JSON，已补齐 raw JSON body/schema/header 和当前 200/400 响应示例，已回读验证
- server 路由：`internal/apis/disaster_storage/v1/router.go`
- server handler：`internal/apis/disaster_storage/v1/handler.go`，`StorageHandler.validateS3Connection`
- 请求链路：`BindJSON(ValidateS3ConnectionRequest)` -> `validateAddressingStyle` -> `validateCAWriteRequest` -> `resolveRequestCABundle` -> `buildValidationHTTPClient` -> AWS `config.LoadDefaultConfig` -> S3 client -> 有 `bucket` 时执行 `HeadBucket`，无 `bucket` 时执行 `ListBuckets` -> `transport.WriteSuccess(200, bool, meta)`
- operator 链路：无；该接口不创建、不更新 `StorageRepository`，不触发 disaster-operator reconcile
- 下层资源：请求指定的 S3/MinIO endpoint 和 bucket；可选读取 `disaster-system/<caSecretRef.name>` Secret 的 `ca.crt`
- 已写入内容：五段详细说明、S3 直连校验语义、body/header 参数说明、`data=false + meta.error` 失败语义、当前成功/失败示例，原说明保留到 `## 原有说明`
- 主要错误点：JSON/binding 失败、`addressingStyle` 非法、CA 参数互斥、`caSecretRef.name` 为空返回 `400 code=1000`；CA Secret 读取失败、PEM 解析失败、AWS config 加载失败、`HeadBucket/ListBuckets` 失败返回 `200 code=0 data=false meta.error`

## GET /apis/storage.testudo.softcdata.com/v1/watch/storages

- RunAPI Target ID：`263ef8a0b8c07c`
- RunAPI 状态：已存在，已更新详细说明，已补齐 `Authorization`、`Sec-WebSocket-Protocol` header 和 `token` query，目标类型 `websocket2` 已保留，已回读验证
- server 路由：`internal/apis/disaster_storage/v1/router.go`
- server handler：`internal/apis/disaster_storage/v1/handler.go`，`StorageHandler.watchStorages`
- 请求链路：`WebSocketTokenAdapter` -> JWT 中间件 -> `upgrader.Upgrade` -> `StorageRepositories(disaster-system).Watch(ctx, metav1.ListOptions{})` -> `StreamWatch` 连接成功帧、心跳帧、watch event 帧、关闭/超时帧
- 返回转换：watch event object 为 `StorageRepository` 时调用 `ConvertToDisasterStorageDTO`，推送到 `data.object`；事件类型写入 `data.type`
- operator 链路：接口本身不触发 reconcile；operator 更新 `StorageRepository.status` 时通过 Kubernetes watch 表现为 `MODIFIED` 事件
- 下层资源：直接监听 `StorageRepository` CRD；不直接访问 S3、CA Secret、Velero `Secret` 或 Velero `BackupStorageLocation`
- 已写入内容：五段详细说明、WebSocket token 三种传入方式、无初始 List 快照说明、连接/事件/心跳/关闭/超时帧示例、完整 DTO 字段说明；`websocket2` 详情无 response 容器，响应示例写在详细说明内
- 主要错误点：握手前 JWT 失败返回普通 HTTP JSON；WebSocket 升级失败返回 HTTP `500 {"message": ...}`；watcher 创建失败通过 WebSocket 返回 `code=5000`；watcher 关闭和 30 分钟超时返回 `code=0` meta 帧

## GET /apis/disasterjobs.testudo.softcdata.com/v1/jobs

- RunAPI Target ID：`25559f21f8c0a5`
- RunAPI 状态：已存在，已更新详细说明，已补齐 query/header 参数和当前 200/400 响应示例，已回读验证
- server 路由：`internal/apis/disaster_jobs/v1/router.go`
- server handler：`internal/apis/disaster_jobs/v1/handler.go`，`JobsHandler.configs`
- 请求链路：`transport.ParseOptions` -> `transport.BuildLabelSelector` 固定返回空 selector -> `DisasterJobLister.DisasterJobs(disaster-system).List` -> 额外 query 按 `metadata.labels` value 包含匹配 -> 按 `name` 或 `creationTimestamp` 内存排序 -> `transport.Paginate` -> `ConvertToDisasterJobDTO` -> `transport.BuildCollectionResponse` -> `transport.WriteSuccess`
- operator 链路：接口本身不触发 reconcile；读取 `DisasterJobReconciler` 已写入的 `status.phase`、`reason`、`startTime`、`conditions`
- 下层资源链路：`DisasterJobReconciler` 读取 `DisasterBackup` 和 `DisasterConfig`；按 `syncType` 决定源/目标集群方向；在源集群 `velero` 命名空间创建或复用 `yaoshi-backup-<jobName>` Velero `Backup`，备份完成并同步到目标集群后在目标集群创建或复用 `yaoshi-restore-<jobName>` Velero `Restore`
- 已写入内容：五段详细说明、列表分页排序与 label 过滤说明、`keyword` 当前不生效说明、完整 collection envelope 字段、DisasterJob DTO 字段、operator phase 来源和下层 Velero 链路，当前成功和列表读取失败示例
- 主要错误点：Lister 列表读取失败返回 `400 code=1000`；JWT 失败由中间件返回；operator 执行备份/恢复失败不导致列表接口失败，只通过 `data.items[].status.phase=Failed` 和 `status.reason` 返回

## POST /apis/disasterjobs.testudo.softcdata.com/v1/jobs

- RunAPI Target ID：`25559f21f8c0aa`
- RunAPI 状态：已存在，已更新详细说明，已修正请求体为当前扁平 DTO，已补齐 raw JSON body/schema/header 和当前 201/400/409/500 响应示例，已回读验证
- server 路由：`internal/apis/disaster_jobs/v1/router.go`
- server handler：`internal/apis/disaster_jobs/v1/handler.go`，`JobsHandler.createConfig`
- 请求链路：`BindJSON(CreateDisasterJobRequest)` -> 组装 `DisasterJob{metadata.name=req.Name, namespace=disaster-system, spec=req.ToCRD()}` -> `transport.SetTraceAnnotation` -> `DisasterJobs(disaster-system).Create` -> `ConvertToDisasterJobDTO` -> `transport.WriteSuccess(201)`
- operator 链路：创建 `DisasterJob` 后触发 `DisasterJobReconciler`；operator 维护 dependency labels，空 `syncType` 明确回写为 `forward`，空 `status.phase` 明确回写为 `Pending`
- 下层资源链路：operator 读取 `DisasterBackup` 和 `DisasterConfig`；按 `syncType` 决定源/目标集群方向；在源集群创建或复用 `yaoshi-backup-<jobName>` Velero `Backup`，备份完成并同步到目标集群后创建或复用 `yaoshi-restore-<jobName>` Velero `Restore`
- 已写入内容：五段详细说明、旧 Kubernetes 原生对象请求体与当前 server DTO 的差异、`syncType` 空值和非 `forward` 值的实际 operator 语义、创建响应 status 为空的异步语义、当前响应示例
- 主要错误点：JSON/binding 失败返回 `400 code=1000`；同名任务返回 `409 code=3009`；非法名称、CRD/RBAC/API server 创建失败返回 `500 code=5000`；`DisasterBackup`/`DisasterConfig`/集群/Velero 失败不在创建接口同步返回，而是由 operator 写入 `status.phase=Failed` 和 `status.reason`

## DELETE /apis/disasterjobs.testudo.softcdata.com/v1/jobs/:name

- RunAPI Target ID：`25559f2238c0ad`
- RunAPI 状态：已存在，已更新详细说明，URL 已从样例路径改为 `{{baseurl}}/apis/disasterjobs.testudo.softcdata.com/v1/jobs/:name`，已补齐 path/header 参数和当前 200/404/500 响应示例，已回读验证
- server 路由：`internal/apis/disaster_jobs/v1/router.go`
- server handler：`internal/apis/disaster_jobs/v1/handler.go`，`JobsHandler.deleteConfig`
- 请求链路：`path.name` -> `DisasterJobs(disaster-system).Delete` -> `transport.WriteSuccess(200, {"name": name})`
- operator 链路：删除带 finalizer 的 `DisasterJob` 后触发 `DisasterJobReconciler` 删除分支；operator 根据关联的 `DisasterBackup` 和 `DisasterConfig` 计算源/目标集群
- 下层资源链路：operator 在源/目标集群 `velero` 命名空间创建 `DeleteBackupRequest`，`spec.backupName=yaoshi-backup-<jobName>`，并尝试删除目标集群 `yaoshi-restore-<jobName>` Velero `Restore`；随后移除 `disasterjob.disaster.io/finalizer`
- 已写入内容：五段详细说明、同步删除 CRD 与异步 finalizer 清理区别、path/header 参数、删除成功不代表下层 Velero 资源已清理完成说明、当前响应示例
- 主要错误点：path name 为空返回 `400 code=1000`；目标不存在返回 `404 code=3004`；Kubernetes 删除失败返回 `500 code=5000`；operator finalizer 阶段的下层清理问题不作为本接口同步错误返回

## GET /apis/disasterjobs.testudo.softcdata.com/v1/jobs/:name

- RunAPI Target ID：`25559f21f8c0a7`
- RunAPI 状态：已存在，已更新详细说明，已补齐 path/header 参数和当前 200/404/500 响应示例，已回读验证
- server 路由：`internal/apis/disaster_jobs/v1/router.go`
- server handler：`internal/apis/disaster_jobs/v1/handler.go`，`JobsHandler.config`
- 请求链路：`path.name` -> `DisasterJobs(disaster-system).Get` -> `ConvertToDisasterJobDTO` -> `transport.WriteSuccess(200)`
- operator 链路：接口本身不触发 reconcile；读取 `DisasterJobReconciler` 已写入的 `status.phase`、`reason`、`startTime`、`conditions`
- 下层资源链路：详情接口不直接访问下层资源；返回状态间接反映 operator 基于 `DisasterBackup`、`DisasterConfig`、源/目标集群 Velero `Backup` 与 `Restore` 的执行结果
- 已写入内容：五段详细说明、path/header 参数、完整 DisasterJob DTO 字段、operator phase 来源、当前详情成功/不存在/读取失败示例
- 主要错误点：目标不存在返回 `404 code=3004`；Kubernetes 读取失败返回 `500 code=5000`；JWT 失败由中间件返回；下层备份/恢复失败不作为详情接口同步错误返回，而是体现在 `status.phase=Failed` 和 `status.reason`

## GET /apis/disasterjobs.testudo.softcdata.com/v1/watch/jobs

- RunAPI Target ID：`25559f2238c0b0`
- RunAPI 状态：已存在，已更新详细说明，已补齐 `Authorization`、`Sec-WebSocket-Protocol` header 和 `token` query，已回读验证
- RunAPI 类型备注：该目标历史 `target_type` 为 `api`；尝试原地切换为 `websocket2` 时 Apipost OpenAPI 返回 `target_id` 冲突，本次保留既有 target 并按 WebSocket 行为写入详细说明
- server 路由：`internal/apis/disaster_jobs/v1/router.go`
- server handler：`internal/apis/disaster_jobs/v1/handler.go`，`JobsHandler.watchJobs`
- 请求链路：`WebSocketTokenAdapter` -> JWT 中间件 -> `upgrader.Upgrade` -> `DisasterJobs(disaster-system).Watch(ctx, metav1.ListOptions{})` -> `StreamWatch` 连接成功帧、心跳帧、watch event 帧、关闭/超时帧
- 返回转换：watch event object 为 `DisasterJob` 时调用 `ConvertToDisasterJobDTO`，推送到 `data.object`；事件类型写入 `data.type`
- operator 链路：接口本身不触发 reconcile；operator 更新 `DisasterJob.status` 时通过 Kubernetes watch 表现为 `MODIFIED` 事件
- 下层资源：直接监听 `DisasterJob` CRD；不直接访问 Velero、源集群或目标集群
- 已写入内容：五段详细说明、WebSocket token 三种传入方式、无初始 List 快照说明、连接/事件/心跳/关闭/超时帧示例、完整 DTO 字段说明；响应帧写在详细说明内
- 主要错误点：握手前 JWT 失败返回普通 HTTP JSON；WebSocket 升级失败返回 HTTP `500 {"message": ...}`；watcher 创建失败通过 WebSocket 返回 `code=5000`；watcher 关闭和 30 分钟超时返回 `code=0` meta 帧

## GET /apis/disasterjobs.testudo.softcdata.com/v1/watch/jobs/:name

- RunAPI Target ID：`25559f2238c0b2`
- RunAPI 状态：已存在，已更新详细说明，已补齐 `Authorization`、`Sec-WebSocket-Protocol` header、`token` query 和 `name` path 参数，已回读验证
- RunAPI 类型备注：该目标历史 `target_type` 为 `api`；本次保留既有 target 并按 WebSocket 行为写入详细说明
- server 路由：`internal/apis/disaster_jobs/v1/router.go`
- server handler：`internal/apis/disaster_jobs/v1/handler.go`，`JobsHandler.watchJob`
- 请求链路：`path.name` 参数检查 -> `WebSocketTokenAdapter` -> JWT 中间件 -> `upgrader.Upgrade` -> `DisasterJobs(disaster-system).Watch(ctx, metav1.ListOptions{FieldSelector: "metadata.name=<name>"})` -> `StreamWatch` 连接成功帧、心跳帧、watch event 帧、关闭/超时帧
- 返回转换：watch event object 为 `DisasterJob` 时调用 `ConvertToDisasterJobDTO`，推送到 `data.object`；事件类型写入 `data.type`
- operator 链路：接口本身不触发 reconcile；operator 更新该任务 `status` 时通过 Kubernetes watch 表现为 `MODIFIED` 事件
- 下层资源：直接监听指定名称的 `DisasterJob` CRD；不直接访问 Velero、源集群或目标集群
- 已写入内容：五段详细说明、指定任务 fieldSelector 语义、WebSocket token 三种传入方式、目标不存在不会同步返回 404 的 watch 语义、连接/事件/心跳/关闭/超时帧示例、完整 DTO 字段说明
- 主要错误点：path name 为空时握手前返回 HTTP `400 {"message":"name is required"}`；握手前 JWT 失败返回普通 HTTP JSON；WebSocket 升级失败返回 HTTP `500 {"message": ...}`；watcher 创建失败通过 WebSocket 返回 `code=5000`；watcher 关闭和 30 分钟超时返回 `code=0` meta 帧

## GET /apis/disasterbackups.testudo.softcdata.com/v1/backups

- RunAPI Target ID：`1c7b0e63d9401001`
- RunAPI 状态：缺失，已新增到 `容灾云平台 / V1 / 容灾备份`，已补齐 `Authorization` header、分页/排序/关键字/动态 label query、当前 200/400/401/500 响应示例，已回读验证
- server 路由：`internal/apis/disaster_backup/v1/router.go`
- server handler：`internal/apis/disaster_backup/v1/handler.go`，`BackupHandler.backups`
- 请求链路：`transport.ParseOptions` -> `transport.BuildLabelSelector` 固定返回空 selector -> `DisasterBackupLister.DisasterBackups(disaster-system).List` -> 额外 query 按 `metadata.labels` value 包含匹配 -> 按 `name` 或 `creationTimestamp` 内存排序 -> `transport.Paginate` -> `ConvertToDisasterBackupDTO` -> `transport.BuildCollectionResponse` -> `transport.WriteSuccess`
- operator 链路：接口本身不触发 reconcile；读取 `DisasterBackupReconciler` 已写入的 `status.phase`、`resources`、`workload`、`conditions`、`updateTime`
- 下层资源链路：`DisasterBackupReconciler` 根据 `spec.disasterConfig` 读取 `DisasterConfig`，再读取 `DisasterConfig.spec.sourceCluster` 指向的 `Cluster`，用 kubeconfig 或 token/endpoint 连接源集群，扫描 `spec.namespace` 下命名空间级 API 资源并写入 `status.resources`；后续 `DisasterJobReconciler` 创建 Velero `Backup`/`Restore` 时会使用该备份的 namespace、labelSelector、newNamespace、newStorageName
- 已写入内容：五段详细说明、列表分页排序与动态 label 过滤说明、`keyword` 当前不生效说明、完整 collection envelope 字段、DisasterBackup DTO 字段、operator phase 和资源扫描来源、当前成功和错误响应示例
- 主要错误点：Lister 列表读取失败返回 `400 code=1000`；JWT 失败由中间件返回；operator 源集群扫描失败不导致列表接口失败，只通过 `data.items[].status.phase=Failed` 和状态字段返回

## POST /apis/disasterbackups.testudo.softcdata.com/v1/backups

- RunAPI Target ID：`1c7b0ed711801001`
- RunAPI 状态：缺失，已新增到 `容灾云平台 / V1 / 容灾备份`，已补齐 raw JSON body/schema、`Authorization`、`Content-Type` header 和当前 201/400/409/500 响应示例，已回读验证
- server 路由：`internal/apis/disaster_backup/v1/router.go`
- server handler：`internal/apis/disaster_backup/v1/handler.go`，`BackupHandler.createBackup`
- 请求链路：`BindJSON(CreateDisasterBackupRequest)` -> 组装 `DisasterBackup{metadata.name=req.Name, namespace=disaster-system, spec=req.ToCRD()}` -> `transport.SetTraceAnnotation` -> `DisasterBackups(disaster-system).Create` -> `ConvertToDisasterBackupDTO` -> `transport.WriteSuccess(201)`
- operator 链路：创建 `DisasterBackup` 后触发 `DisasterBackupReconciler`；operator 同步 dependency token 和 `spec.disasterConfig` 依赖标签，读取 `DisasterConfig` 与源 `Cluster`，扫描源命名空间资源，成功后写 `status.phase=Ready`，失败写 `status.phase=Failed`
- 下层资源链路：创建接口不直接访问源集群或 Velero；后续 `DisasterJobReconciler` 读取该备份后在源集群 `velero` 命名空间创建 `yaoshi-backup-<jobName>` Velero `Backup`，并使用 `spec.namespace`、`spec.labelSelector`、`newNamespace`、`newStorageName`
- 已写入内容：五段详细说明、完整 request 字段说明、`namespace` 业务必传但 server 未强制说明、创建响应 status 异步语义、当前响应示例；默认成功响应码已通过 Apipost OpenAPI 修正为 `201`
- 主要错误点：JSON/binding 失败返回 `400 code=1000`；同名备份返回 `409 code=3009`；非法 Kubernetes 名称、CRD/RBAC/API server 创建失败返回 `500 code=5000`；`DisasterConfig` 或源集群问题不在创建接口同步返回，而是由 operator 写入 `status.phase=Failed`

## DELETE /apis/disasterbackups.testudo.softcdata.com/v1/backups/:name

- RunAPI Target ID：`1c7b0f565a801001`
- RunAPI 状态：缺失，已新增到 `容灾云平台 / V1 / 容灾备份`，已补齐 `Authorization` header、`name` path 参数和当前 200/404/500 响应示例，鉴权类型已设为继承项目鉴权，已回读验证
- server 路由：`internal/apis/disaster_backup/v1/router.go`
- server handler：`internal/apis/disaster_backup/v1/handler.go`，`BackupHandler.deleteBackup`
- 请求链路：`path.name` -> 尽力 `Get` 目标 `DisasterBackup` -> 尽力写入 trace annotation 并 `Update` -> `DisasterBackups(disaster-system).Delete` -> `transport.WriteSuccess(200, {"name": name})`
- operator 链路：当前 `DisasterBackupReconciler` 未设置专门 finalizer；资源不存在时直接返回，不执行下层清理
- 下层资源链路：接口只删除 `DisasterBackup` CRD，不直接清理源/目标集群 Velero `Backup`、`Restore`、S3 对象或历史 `DisasterJob`
- 已写入内容：五段详细说明、path/header 参数说明、删除前 annotation best-effort 说明、删除成功不代表下层备份产物清理说明、当前响应示例
- 主要错误点：目标不存在返回 `404 code=3004`；Kubernetes 删除失败返回 `500 code=5000`；删除前读取或 annotation 更新失败当前被忽略

## GET /apis/disasterbackups.testudo.softcdata.com/v1/backups/:name

- RunAPI Target ID：`1c7b0f91e8c01001`
- RunAPI 状态：缺失，已新增到 `容灾云平台 / V1 / 容灾备份`，已补齐 `Authorization` header、`name` path 参数和当前 200/400/404/500 响应示例，鉴权类型已设为继承项目鉴权，已回读验证
- server 路由：`internal/apis/disaster_backup/v1/router.go`
- server handler：`internal/apis/disaster_backup/v1/handler.go`，`BackupHandler.backup`
- 请求链路：`path.name` 参数检查 -> `DisasterBackups(disaster-system).Get` -> `ConvertToDisasterBackupDTO` -> `transport.WriteSuccess(200)`
- operator 链路：接口本身不触发 reconcile；读取 `DisasterBackupReconciler` 已写入的状态和资源扫描结果
- 下层资源链路：详情接口不直接访问源/目标集群或 Velero；返回的 `status.resources` 间接来源于 operator 使用源集群 discovery/dynamic client 对 `spec.namespace` 的扫描
- 已写入内容：五段详细说明、path/header 参数说明、完整 DisasterBackup DTO 字段、operator phase 和资源扫描来源、当前详情成功/不存在/读取失败示例
- 主要错误点：path name 为空返回 `400 code=1000`；目标不存在返回 `404 code=3004`；Kubernetes 读取失败返回 `500 code=5000`；operator 依赖或扫描失败不作为详情接口同步错误返回，而是体现在 `status.phase=Failed`

## PUT /apis/disasterbackups.testudo.softcdata.com/v1/backups/:name

- RunAPI Target ID：`1c7b0fcf32001001`
- RunAPI 状态：缺失，已新增到 `容灾云平台 / V1 / 容灾备份`，已补齐 raw JSON body/schema、`Authorization`、`Content-Type` header、`name` path 参数和当前 200/400/404/409/500 响应示例，鉴权类型已设为继承项目鉴权，已回读验证
- server 路由：`internal/apis/disaster_backup/v1/router.go`
- server handler：`internal/apis/disaster_backup/v1/handler.go`，`BackupHandler.updateBackup`
- 请求链路：`BindJSON(UpdateDisasterBackupRequest)` -> `RetryOnConflict` -> 按 `body.name` `Get` 现有 `DisasterBackup` -> `MergeToCRD` 非空字段覆盖 -> 注入 trace annotation -> `Update` -> `ConvertToDisasterBackupDTO` -> `transport.WriteSuccess(200)`
- 关键差异：path `:name` 当前只用于路由匹配，handler 实际使用 `body.name` 定位资源；文档已要求两者保持一致
- operator 链路：CRD spec 更新后触发 `DisasterBackupReconciler`；operator 重新同步 dependency labels，读取 `DisasterConfig` 和源 `Cluster`，重新扫描 `spec.namespace` 资源并更新状态
- 下层资源链路：更新接口不直接访问 Velero；后续 `DisasterJobReconciler` 使用更新后的 `spec.namespace`、`spec.labelSelector`、`newNamespace`、`newStorageName` 创建 Velero `Backup`/`Restore`
- 已写入内容：五段详细说明、body/path 差异说明、局部非空合并语义、无法用空字符串清空字段说明、冲突重试说明、当前响应示例
- 主要错误点：JSON/binding 失败返回 `400 code=1000`；`body.name` 目标不存在返回 `404 code=3004`；多次冲突返回 `409 code=3009`；CRD 读取或更新失败返回 `500 code=5000`；`DisasterConfig` 或源集群问题由 operator 后续写入状态

## GET /apis/disasterbackups.testudo.softcdata.com/v1/watch/backups

- RunAPI Target ID：`6345b0ecacc6000`
- RunAPI 状态：缺失，已新增到 `容灾云平台 / V1 / 容灾备份`，目标类型为 `websocket2`，已补齐 `Authorization`、`Sec-WebSocket-Protocol` header 和 `token` query，已回读验证
- server 路由：`internal/apis/disaster_backup/v1/router.go`
- server handler：`internal/apis/disaster_backup/v1/handler.go`，`BackupHandler.watchBackups`
- 请求链路：`WebSocketTokenAdapter` -> JWT 中间件 -> `upgrader.Upgrade` -> `DisasterBackups(disaster-system).Watch(ctx, metav1.ListOptions{})` -> `StreamWatch` 连接成功帧、心跳帧、watch event 帧、关闭/超时帧
- 返回转换：watch event object 为 `DisasterBackup` 时调用 `ConvertToDisasterBackupDTO`，推送到 `data.object`；事件类型写入 `data.type`
- operator 链路：接口本身不触发 reconcile；operator 更新 `DisasterBackup.status` 或 dependency labels 时通过 Kubernetes watch 表现为 `MODIFIED` 事件
- 下层资源链路：直接监听 `DisasterBackup` CRD；不直接访问源/目标集群、S3 或 Velero，状态字段间接反映 operator 读取 `DisasterConfig`、源 `Cluster` 和扫描源命名空间资源的结果
- 已写入内容：五段详细说明、WebSocket token 三种传入方式、无初始 List 快照说明、连接/事件/心跳/关闭/超时帧示例、完整 DTO 字段说明；`websocket2` 详情无 response 容器，响应帧写在详细说明内
- 主要错误点：握手前 JWT 失败返回普通 HTTP JSON；WebSocket 升级失败返回 HTTP `500 {"message": ...}`；watcher 创建失败通过 WebSocket 返回 `code=5000`；watcher 关闭和 30 分钟超时返回 `code=0` meta 帧

## GET /apis/disasterbackups.testudo.softcdata.com/v1/watch/backups/:name

- RunAPI Target ID：`6345b54530c6000`
- RunAPI 状态：缺失，已新增到 `容灾云平台 / V1 / 容灾备份`，目标类型为 `websocket2`，已补齐 `Authorization`、`Sec-WebSocket-Protocol` header 和 `token` query，已回读验证
- RunAPI 结构备注：`websocket2` 详情未显示 restful 容器，path `name` 已完整写入详细说明
- server 路由：`internal/apis/disaster_backup/v1/router.go`
- server handler：`internal/apis/disaster_backup/v1/handler.go`，`BackupHandler.watchBackup`
- 请求链路：`path.name` 参数检查 -> `WebSocketTokenAdapter` -> JWT 中间件 -> `upgrader.Upgrade` -> `DisasterBackups(disaster-system).Watch(ctx, metav1.ListOptions{FieldSelector: "metadata.name=<name>"})` -> `StreamWatch` 连接成功帧、心跳帧、watch event 帧、关闭/超时帧
- 返回转换：watch event object 为 `DisasterBackup` 时调用 `ConvertToDisasterBackupDTO`，推送到 `data.object`；事件类型写入 `data.type`
- operator 链路：接口本身不触发 reconcile；operator 更新该备份状态或 dependency labels 时通过 Kubernetes watch 表现为 `MODIFIED` 事件
- 下层资源链路：直接监听指定名称的 `DisasterBackup` CRD；不直接访问源/目标集群、S3 或 Velero，状态字段间接反映 operator 读取 `DisasterConfig`、源 `Cluster` 和扫描源命名空间资源的结果
- 已写入内容：五段详细说明、指定备份 fieldSelector 语义、WebSocket token 三种传入方式、目标不存在不会同步返回 404 的 watch 语义、连接/事件/心跳/关闭/超时帧示例、完整 DTO 字段说明
- 主要错误点：path name 为空时握手前返回 HTTP `400 code=1000`；握手前 JWT 失败返回普通 HTTP JSON；WebSocket 升级失败返回 HTTP `500 {"message": ...}`；watcher 创建失败通过 WebSocket 返回 `code=5000`；watcher 关闭和 30 分钟超时返回 `code=0` meta 帧

## GET /apis/disasterinstances.testudo.softcdata.com/v1/instances

- RunAPI Target ID：`1c01f46e4f001000`
- RunAPI 状态：已存在，已更新详细说明，URL 已从带样例 query 的路径规范为 `{{baseurl}}/apis/disasterinstances.testudo.softcdata.com/v1/instances`，已补齐 `Authorization` header、分页/排序/关键字/namespace/动态 label query、`meta.summary.protectedCount`、当前 200/500 响应示例，鉴权类型已设为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证；2026-05-21 已修正 `query.namespace` 为受保护业务 namespace 过滤，不再表示 CR 存储 namespace；2026-05-25 已改为对 `spec.namespaces` 做包含匹配，支持传 namespace 片段模糊搜索
- server 路由：`internal/apis/disaster_instance/v1/router.go`
- server handler：`internal/apis/disaster_instance/v1/handler.go`，`InstanceHandler.listInstances`
- 请求链路：`transport.ParseOptions` -> 解析并移除 `namespace` 过滤作为受保护业务 namespace 片段 -> 默认 `sort=creationTimestamp&order=desc` -> `DisasterInstances(common.DisasterSystemNamespace).List` 从平台管理命名空间读取实例 CR -> 当 `namespace` 非空且非 `*` 时按 `DisasterInstance.spec.namespaces` 包含匹配业务 namespace -> 动态 label 模糊过滤 -> `keyword` 对 `name/CR namespace/label values` 包含匹配 -> 按 `creationTimestamp/name/namespace` 稳定排序 -> `summarizeDisasterInstanceList` 统计过滤后分页前 `fsmState=Protected` 数量 -> `transport.Paginate` -> 按当前页逐项读取 `DisasterConfig` -> `ConvertToDisasterInstanceDTO` -> `enrichListSyncStatus` 读取 `DataSync`/`ResourceSync` -> `enrichListAutoCancel` 读取最近 failover `DisasterOperation` -> `BuildCollectionResponse` -> 写入 `meta.summary.protectedCount` -> `WriteSuccess`
- operator 链路：接口本身不触发 reconcile；读取 `DisasterInstanceReconciler` 已写入的 `fsmState`、主备集群、可执行操作、DataSync/ResourceSync 名称和最近同步时间
- 下层资源链路：`DisasterInstanceReconciler` 根据 `DisasterConfig` 创建或更新 `DataSync` 与 `ResourceSync`，子资源继续驱动 AppBackup/AppRestore/Velero；列表接口不直接访问源/目标集群或 Velero
- 已写入内容：五段详细说明、namespace 作为受保护业务 namespace 片段的模糊过滤语义、动态 label 过滤语义、keyword 当前实际匹配范围、`meta.summary.protectedCount` 统计语义、完整 collection envelope、DisasterInstance DTO、spec/status/sync summary/autoCancel 字段、operator 状态机来源、当前响应示例
- 取证备注：当前代码中 `currentState` 来自 operator `CurrentStateFromFSM`，`ConfigError` 映射为 `Failed`；`protectedCount` 严格统计 `status.fsmState=Protected`，不会把 `currentState=Running` 中的 Pending/Initializing 计入保护中实例；RunAPI 原说明中 `ConfigError -> Error` 已按要求保留在 `## 原有说明`，但未覆盖当前实现说明
- 主要错误点：`DisasterInstances(common.DisasterSystemNamespace).List` 失败返回 `500 code=5000`；JWT 失败由中间件返回；单个实例的 `DisasterConfig`、`DataSync`、`ResourceSync` 或最近 failover `DisasterOperation` 读取失败不会让整个列表失败，只表现为嵌套字段为空或缺失

## POST /apis/disasterinstances.testudo.softcdata.com/v1/instances

- RunAPI Target ID：`1c01f46f55801000`
- RunAPI 状态：已存在，已更新详细说明，已补齐 `Authorization` header、raw JSON body/schema、当前 201/400/403/409/500 响应示例，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证；2026-05-22 已追加 `ModifierRuleRejected：replace 缺失最终路径仍拒绝` 400 响应示例；2026-06-15 已在 live RunAPI 前置追加 `bulkModifierActions` 受保护路径扫描修复说明并回读验证
- server 路由：`internal/apis/disaster_instance/v1/router.go`
- server handler：`internal/apis/disaster_instance/v1/handler.go`，`InstanceHandler.createInstance`
- 请求链路：`rejectUnsupportedSyncPolicyField(ctx.Request.Body())` -> `BindJSON(CreateDisasterInstanceRequest)` -> `req.ToCRD()` -> `DisasterConfigs().Get(req.Config)` -> `validateProtectedNamespaces(config.Spec.SourceCluster, spec.Namespaces, "", "")` -> `prepareRestorePolicyForPersist(c, &spec, nil)` -> 默认 `namespace=disaster-system` 与 `podRestoreMethod=replica` -> `SetTraceAnnotation` -> `DisasterInstances(ns).Create` -> `ConvertToDisasterInstanceDTO(rc, config, nil)` -> `WriteSuccess(201)`
- restore policy 解析链路：`ResolveRestorePolicy` 会解析 `resourceSelection`、`execution`、`storageClassMapping`、`ingressClassMapping`、`bulkModifierActions`、`bulkModifierActionsText`、`modifierRules`、`modifierRulesText`、`useUnifiedDirectionResolver`；文本入口与结构化入口同时存在时必须语义一致；资源 include/exclude、modifier rule 结构和 live 路径校验失败都会在创建前返回 `400`；live path 校验按 JSON Patch 语义处理，`add` 允许最终一级 map key 不存在但父路径必须存在，`replace/remove` 要求完整路径存在；bulk 快照生成阶段会跳过 `/status/**`、`/metadata/finalizers/**`、`/metadata/ownerReferences/**`，因此镜像值同时出现在 workload spec 和 Pod status 时不会为 status 路径生成快照规则，也不要求用户用 `resourceSelection.excludedResources=["pods"]` 规避
- live 文档同步：Apipost Target ID `1c01f46f55801000` 已回读完整详情后保留原说明，并在描述前置追加 `2026-06-15 bulkModifierActions 受保护路径扫描修复`，避免覆盖既有 request/response 示例
- operator 链路：创建 `DisasterInstance` 后触发 `DisasterInstanceReconciler`；operator 添加 finalizer、同步依赖 label、把空 `status.fsmState` 初始化为 `Pending`，随后创建/更新 `DataSync` `dr-ds-<instance>` 和 `ResourceSync` `dr-rs-<instance>`，写入主备集群并进入 `Initializing`
- 下层资源链路：本接口不直接创建 Velero 资源；后续由 `DataSync`/`ResourceSync` controller 基于实例范围和恢复策略驱动 AppBackup/AppRestore、Velero `Backup`、Velero `Restore` 与目标集群 standby 资源
- 已写入内容：五段详细说明、完整创建入参字段说明、`syncPolicy` 禁用说明、实例级双策略 override 继承语义、受保护命名空间冲突 meta、restorePolicy 嵌套字段和 modifier 规则约束、同步创建响应状态为空/`currentState=Unknown` 的当前实现说明、operator 异步状态推进说明，旧说明完整保留到 `## 原有说明`
- 取证备注：创建接口同步返回的是 API Server 刚写入的对象，operator 通常尚未回写 `status.fsmState`，因此当前实现的同步成功示例按 `status={}` 与 `currentState=Unknown` 写入；原 RunAPI 示例中 `Pending` 类描述保留为历史说明
- 主要错误点：顶层 `syncPolicy`、JSON/binding、`Config <name> not found`、`ResourceSelectionInvalid`、`ModifierRulesTextInvalid`、`BulkModifierActionsTextInvalid`、`ModifierRulesInputConflict`、`BulkModifierActionsInputConflict`、`ModifierRuleRejected`（例如 `add` 父路径不存在、`replace/remove` 路径不存在、bulk action 过滤受保护路径后没有可执行命中）或 Kubernetes Invalid/BadRequest 返回 `400 code=1000`；受保护命名空间冲突或同名实例返回 `409 code=3009`；RBAC 禁止创建返回 `403 code=2003`；配置读取非 NotFound、保护索引构建或 Kubernetes 创建内部错误返回 `500 code=5000`；operator 后续 `ConfigError`、`PolicyNotReady`、`DataSyncFailed`、`ResourceSyncFailed`、`InitializationFailed` 不作为本接口同步错误返回，只通过状态接口、详情、列表或 watch 体现

## DELETE /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name

- RunAPI Target ID：`1c01f472a2c01000`
- RunAPI 状态：已存在，已更新详细说明，URL 已从样例实例名规范为 `{{baseurl}}/apis/disasterinstances.testudo.softcdata.com/v1/instances/:name`，已补齐 `Authorization` header、`name` path 参数、`namespace` query 参数和当前 200/500 响应示例，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_instance/v1/router.go`
- server handler：`internal/apis/disaster_instance/v1/handler.go`，`InstanceHandler.deleteInstance`
- 请求链路：`path.name` -> 可选 `query.namespace` -> namespace 为空时调用 `findNamespace` -> `DisasterInstances(namespace).Delete` -> `WriteSuccess(200, {"name": name})`
- 幂等语义：`findNamespace` 失败时当前实现直接返回成功；Delete 返回 NotFound 时也返回成功，因此该接口不会用 `404` 表达目标不存在
- operator 链路：删除带 finalizer 的 `DisasterInstance` 后触发 `DisasterInstanceReconciler.handleDeletion`；operator 记录结构化删除任务，重新获取最新对象，移除 finalizer `testudo.softcdata.com/disasterinstance-finalizer`
- 下层资源链路：`DataSync` `dr-ds-<instance>` 与 `ResourceSync` `dr-rs-<instance>` 依赖 ownerReference 由 Kubernetes 垃圾回收级联删除；本接口不直接清理 AppBackup、AppRestore、Velero `Backup`、Velero `Restore`、S3 对象或目标集群业务资源
- 已写入内容：五段详细说明、path/query/header 参数、幂等删除和 namespace 自动解析语义、当前 operator 删除保护停用事实、finalizer 异步完成说明、返回 `200` 不代表下层资源彻底清理完成说明
- 取证备注：operator 中旧逻辑曾阻止 `Protected/Active/FailingOver/FailingBack` 状态或被 `DisasterGroup` 引用的实例删除，但当前代码已整段注释停用；文档按当前实现写明不会阻塞 finalizer 移除
- 主要错误点：JWT 失败由中间件返回 `401 code=2001`；Kubernetes Delete 返回非 NotFound 错误时返回 `500 code=5000`；目标不存在或 namespace 自动解析失败当前均返回 `200 code=0`

## GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name

- RunAPI Target ID：`1c01f470ba401000`
- RunAPI 状态：已存在，已更新详细说明，已补齐 `Authorization` header、`name` path 参数、`namespace` query 参数和当前 200/404/500 响应示例，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_instance/v1/router.go`
- server handler：`internal/apis/disaster_instance/v1/handler.go`，`InstanceHandler.getInstance`
- 请求链路：`path.name` -> 可选 `query.namespace` -> namespace 为空时调用 `findNamespace` -> `DisasterInstances(namespace).Get` -> 尽力读取 `DisasterConfig` -> 尽力读取 `StorageRepository` -> `ConvertToDisasterInstanceDTO` -> 尽力读取最近 failover `DisasterOperation` 并生成 `autoCancel` -> `WriteSuccess(200)`
- operator 链路：接口本身不触发 reconcile；读取 `DisasterInstanceReconciler` 已写入的 `fsmState`、主备集群、可执行操作、DataSync/ResourceSync 名称、同步时间、conditions 与错误原因
- 下层资源链路：详情接口不直接访问源/目标集群或 Velero；返回状态间接反映 operator 通过 `DataSync`、`ResourceSync` 以及下层 AppBackup/AppRestore/Velero 链路完成的结果
- 已写入内容：五段详细说明、path/query/header 参数、完整 `DisasterInstanceDTO` 字段、`config/storage/autoCancel` 聚合来源、`effectiveDataSyncPolicy/effectiveResourceSyncPolicy` 及来源字段、`CurrentStateFromFSM` 当前映射、operator 状态字段和可能原因
- 取证备注：RunAPI 原说明将 `status.fsmState=ConfigError` 映射为 `currentState=Error`；当前 operator 类型代码实际为 `ConfigError/Failed -> Failed`，新说明和新增响应示例已按当前实现写入，原说明按规则保留在 `## 原有说明`
- 主要错误点：namespace 自动解析或指定 namespace 下目标不存在返回 `404 code=3004`；namespace 自动解析 List 失败或实例 Get 非 NotFound 错误返回 `500 code=5000`；JWT 失败由中间件返回；关联 `DisasterConfig`、`StorageRepository`、最近 failover `DisasterOperation` 读取失败不会导致详情失败，只会使对应字段为空或省略

## PUT /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name

- RunAPI Target ID：`1c01f4718c001000`
- RunAPI 状态：已存在，已更新详细说明，已补齐 `Authorization`、`Content-Type` header、`name` path 参数、`namespace` query 参数、raw JSON body/schema 和当前 200/400/403/404/409/500 响应示例，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证；2026-05-22 已追加 `ModifierRuleRejected：replace 缺失最终路径仍拒绝` 400 响应示例；2026-06-15 已在 live RunAPI 前置追加 `bulkModifierActions` 受保护路径扫描修复说明并回读验证
- server 路由：`internal/apis/disaster_instance/v1/router.go`
- server handler：`internal/apis/disaster_instance/v1/handler.go`，`InstanceHandler.updateInstance`
- 请求链路：`path.name` -> `rejectUnsupportedSyncPolicyField` -> `BindJSON(UpdateDisasterInstanceRequest)` -> `ResolveRestorePolicy` -> 可选 `query.namespace` 或 `findNamespace` -> `RetryOnConflict` 中读取现有实例 -> 按出现字段更新 spec/annotation -> 重新做受保护 namespace 冲突校验 -> 按出现的 `restorePolicy` 子字段合并 -> 必要时 `prepareRestorePolicyForPersist` -> `Update` -> 尽力读取 `DisasterConfig` -> `ConvertToDisasterInstanceDTO` -> `WriteSuccess(200)`
- 更新语义：`dataSyncPolicy/resourceSyncPolicy` 未传或 `null` 保持原值，空字符串清空 override；`namespaces` 未传或 `null` 保持原值，空数组清空；`labelSelector` 未传或 `null` 保持原值，传对象整体替换；`description` 空字符串清空；`restorePolicy` 未传或 `null` 不更新，传对象时只合并对象中出现的子字段
- operator 链路：更新 CRD 后触发 `DisasterInstanceReconciler`；operator 继续同步 dependency labels，按实例 override 或配置继承值更新 `DataSync`/`ResourceSync` schedule，后续 DataSync/ResourceSync/Drill 使用更新后的恢复策略构建 AppRestore
- 下层资源链路：更新接口不直接访问 Velero 或业务集群；只有在 bulk modifier 需要重建快照或 modifier rule live 校验时，server 会读取源集群资源来验证和生成 `modifierRuleSnapshot`；live path 校验按 JSON Patch 语义处理，`add` 允许最终一级 map key 不存在但父路径必须存在，`replace/remove` 要求完整路径存在；bulk 快照生成阶段会跳过 `/status/**`、`/metadata/finalizers/**`、`/metadata/ownerReferences/**`，因此镜像值同时出现在 workload spec 和 Pod status 时不会为 status 路径生成快照规则，也不要求用户用 `resourceSelection.excludedResources=["pods"]` 规避
- live 文档同步：Apipost Target ID `1c01f4718c001000` 已回读完整详情后保留原说明，并在描述前置追加 `2026-06-15 bulkModifierActions 受保护路径扫描修复`，避免覆盖既有 request/response 示例
- 已写入内容：五段详细说明、全部可更新字段的未传/null/空值语义、`restorePolicy` 子字段合并和清空语义、modifier 文本入口与结构化入口冲突规则、namespace 冲突 meta、响应 DTO 和当前错误分类
- 取证备注：该接口使用 `RetryOnConflict`，但最终返回的 Kubernetes resourceVersion 冲突未单独映射为 `409`；除受保护 namespace 冲突外，重试耗尽后会进入通用 `500 code=5000`
- 主要错误点：顶层 `syncPolicy`、JSON/binding、现有实例 `spec.config` 无法通过 lister 找到、恢复策略校验失败（例如 `add` 父路径不存在、`replace/remove` 路径不存在、bulk action 过滤受保护路径后没有可执行命中）或 Kubernetes Invalid/BadRequest 返回 `400 code=1000`；目标不存在返回 `404 code=3004`；受保护命名空间冲突返回 `409 code=3009`；RBAC 禁止更新返回 `403 code=2003`；重试耗尽、读取或更新 CRD 非分类错误返回 `500 code=5000`；operator 后续失败只体现在实例状态字段中

## POST /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/actions

- RunAPI Target ID：`1c01fb364ac01000`
- RunAPI 状态：已存在，已更新详细说明，URL 已从样例实例名规范为 `{{baseurl}}/apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/actions`，已补齐 `Authorization`、`Content-Type` header、`name` path 参数、`namespace` query 参数、raw JSON body/schema 和当前 202/400/404/409/500 响应示例，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证
- 2026-05-18 RunAPI 追加响应示例：`同步失败数据同步重试受理成功`，覆盖 `Failed` 可恢复同步失败下 `sync-data` 返回 202 的兼容语义
- 2026-05-22 代码与 OpenAPI 更新：`FailingOver` 状态下 server 会把 `cancel` 补入有效操作集，避免 operator 进行中清空 `availableOperations` 时取消请求被 409 拦截；RunAPI live 已补充 `FailingOver取消受理成功` 202 响应示例
- server 路由：`internal/apis/disaster_instance/v1/router.go`
- server handler：`internal/apis/disaster_instance/v1/handler_action.go`，`InstanceHandler.executeAction`
- 请求链路：`path.name` -> `BindJSON(ExecuteActionRequest)` -> 可选 `query.namespace` 或 `findNamespace` -> `DisasterInstances(namespace).Get` -> switch 校验 operation -> `validateInstanceOperationAllowed` 校验有效 `availableOperations`（含可恢复同步失败的 Failed 重试兼容项和 FailingOver 下 cancel 补入项）-> 组装 `DisasterOperation` -> 提取 `config.force/skipFinalSync/skipScaleDownSource/timeoutMinutes/skipPodReadyCheck/waitUntilReady` -> `DisasterOperations(namespace).Create` -> `WriteSuccess(202, {"operationID": createdOp.Name, "status": "Processing"})`
- 当前请求值备注：server switch 接收 `sync-data` 与 `sync-resource`，并映射为 CRD enum `syncdata` 与 `syncresource`；直接传 `syncdata` 或 `syncresource` 会返回 `400 Unknown operation type`；`reset` 虽可能出现在实例 `availableOperations`，但当前 handler 不接收
- operator 链路：`DisasterOperationReconciler` 为操作补 ownerReference 与 dependency labels，初始化 `status.state=Running`，再分发到 `handleFailover/handleReprotect/handleUndo/handleCancel/handlePause/handleResume/handleSync`
- 下层资源链路：`failover/reprotect/undo/cancel` 通过步骤编排修改实例状态、DataSync/ResourceSync 调度和源/目标集群 workload 副本；`pause/resume` 修改 DataSync/ResourceSync `spec.paused` 并更新实例状态；`synconce/sync-data/sync-resource` 修改 DataSync/ResourceSync `spec.trigger.manual` 并等待状态 Ready
- 已写入内容：五段详细说明、全部入参字段和大小写兼容字段、operation 请求值与 CRD enum 映射、`skipPodReadyCheck/waitUntilReady` 优先级、`skipScaleDownSource` annotation/spec 写入、202 受理响应、operator 后续状态和失败原因
- 取证备注：RunAPI 原响应示例中有“组内实例禁止单操”示例；当前 `executeAction` handler 未执行 `findContainingGroups` 或组内禁用校验，新说明按当前 server 代码写入并保留旧说明到 `## 原有说明`
- 主要错误点：JSON/binding 或未知 operation 返回 `400 code=1000`；namespace 自动解析或实例读取失败返回 `404 code=3004`；operation 不在有效 `availableOperations` 返回 `409 code=3009` 并带 `ValidateTargetDTO` meta；`Failed` 且 reason 为 `DataSyncFailed/ResourceSyncFailed/InitializationFailed/BackupFailed/BuildRestoreSpecFailed/RestoreFailed/DependencyFailed/StorageUnavailable` 时，`sync-data/sync-resource` 重试按兼容规则放行；`FailingOver` 时 `cancel` 按兼容规则放行；namespace 自动解析非 NotFound 错误和创建 `DisasterOperation` 失败返回 `500 code=5000`；operator 异步执行失败只体现在 `DisasterOperation.status` 与实例状态中

## GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/groups

- RunAPI Target ID：`1c7b14c8c7401001`
- RunAPI 状态：缺失，已新增到 `容灾云平台 / V1 / 容灾实例`，已补齐 `Authorization` header、`name` path 参数、`namespace` query 参数和当前 200/404/500 响应示例，鉴权类型已设为继承项目鉴权，已回读验证
- server 路由：`internal/apis/disaster_instance/v1/router.go`
- server handler：`internal/apis/disaster_instance/v1/handler.go`，`InstanceHandler.getInstanceGroups`
- 请求链路：`path.name` -> 可选 `query.namespace` 或 `findNamespace` -> `DisasterInstances(namespace).Get` 确认实例存在 -> `findContainingGroups` 列出同 namespace `DisasterGroup` -> 遍历 `spec.levels` 查找实例名 -> 去重排序 -> `WriteSuccess(200, InstanceGroupsDTO)`
- operator 链路：接口本身不触发 reconcile；`DisasterGroupReconciler` 同样基于 `DisasterGroup.spec.levels` 监听实例和组变化并聚合组 status，但本接口只读取 spec 归属关系，不读取组 status
- 下层资源链路：不访问 DataSync、ResourceSync、源/目标集群、Velero 或 S3
- 已写入内容：五段详细说明、path/query/header 参数、`InstanceGroupsDTO` 字段、`DisasterGroup.spec.levels` 二维数组归属语义、无归属时 `groups=[]` 与 `inGroup=false`、当前错误分类
- 主要错误点：namespace 自动解析或指定 namespace 下实例不存在返回 `404 code=3004`；namespace 自动解析非 NotFound 错误、实例 Get 非 NotFound 错误、列出 `DisasterGroup` 失败返回 `500 code=5000`；JWT 失败由中间件返回

## GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/history

- RunAPI Target ID：`1c01f7948bc01000`
- RunAPI 状态：已存在，已更新详细说明，已补齐 `Authorization` header、`name` path 参数、`namespace` query 参数和当前 200/404/500 响应示例，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_instance/v1/router.go`
- server handler：`internal/apis/disaster_instance/v1/handler.go`，`InstanceHandler.getHistory`
- 请求链路：`path.name` -> 可选 `query.namespace`；namespace 为空时 `findNamespace` -> `DisasterOperations(namespace).List(LabelSelector: testudo.softcdata.com/instance=<name>)` -> `ConvertToHistoryDTO` -> 按 `time` 倒序排序 -> `WriteSuccess(200, []HistoryDTO)`
- 当前实现范围：只读取 `DisasterOperation`，不读取 `DataSync`/`ResourceSync` status history，不读取 `BackupRestoreStatistics`，不做分页
- operator 链路：`DisasterOperationReconciler` 在动作执行过程中维护 `status.state/reason/message/currentStep/steps/autoCancel/roleStatus`；历史接口读取这些已写入状态并转换为展示 DTO
- 下层资源链路：接口本身不访问源/目标集群、Velero、DataSync、ResourceSync 或 S3；下层失败仅通过 `DisasterOperation.status` 反映
- 已写入内容：五段详细说明、path/query/header 参数、显式 namespace 不校验实例存在的当前行为、完整 `HistoryDTO` 字段、autoCancel 字段、兼容字段 `result/reason/operator/note` 来源、无分页说明、当前错误分类
- 取证备注：RunAPI 原示例包含 `ResourceSync/DataSync` 类历史和 `total/page/limit` 字段；当前 handler 只返回操作历史数组，旧说明和旧示例保留为历史内容，新说明按当前 server 代码写入
- 主要错误点：未传 namespace 且 `findNamespace` 失败返回 `404 code=3004`；列出 `DisasterOperation` 失败返回 `500 code=5000`；JWT 失败由中间件返回；显式传入 namespace 时即使实例不存在也可能返回 `200 code=0 data=[]`

## GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/operations/:operationName

- RunAPI Target ID：`1c5ead7bab401001`
- RunAPI 状态：已存在，已更新详细说明，URL 已规范为 `{{baseurl}}/apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/operations/:operationName`，已补齐 `Authorization` header、`name` 与 `operationName` path 参数、`namespace` query 参数和当前 200/404/500 响应示例，鉴权类型已设为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_instance/v1/router.go`
- server handler：`internal/apis/disaster_instance/v1/handler_operation.go`，`InstanceHandler.getOperationDetail`
- 请求链路：`path.name` -> `path.operationName` -> 可选 `query.namespace`；namespace 为空时 `findNamespace` -> `DisasterOperations(namespace).Get(operationName)` -> 校验 `op.Spec.InstanceName == path.name` -> `ConvertToOperationDetailDTO` -> `WriteSuccess(200, OperationDetailDTO)`
- operator 链路：接口本身不触发 reconcile；读取 `DisasterOperationReconciler` 在动作执行中维护的 `status.state/reason/currentStep/message/steps/autoCancel/roleStatus/groupStatus/startTime/completionTime`
- 下层资源链路：不直接读取 DataSync、ResourceSync、源/目标集群、Velero 或 S3；下层执行结果通过 `DisasterOperation.status` 投影到详情 DTO
- 已写入内容：五段详细说明、path/query/header 参数、operation 归属校验语义、完整 `OperationDetailDTO` 字段、实例 operation 与组 operation 的成功/拒绝边界、operator 异步执行失败只体现在状态字段中的说明、当前错误分类
- 取证备注：RunAPI 原说明已完整保留；当前说明明确 `groupStatus` 是组操作字段，本接口成功读取实例 operation 时通常为空，operation 存在但 `spec.instanceName` 不匹配时按不存在返回 `404`
- 主要错误点：未传 namespace 且 `findNamespace` 失败返回 `404 code=3004`；operation 不存在返回 `404 code=3004`；operation 不属于该实例返回 `404 code=3004`；读取 operation 的非 NotFound 错误返回 `500 code=5000`；JWT 失败由中间件返回 `401 code=2001`

## POST /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/restore-classes/validate

- RunAPI Target ID：`1c49f6d741801001`
- RunAPI 状态：已存在，已更新详细说明，已补齐 `Authorization`、`Content-Type` header、`name` path 参数、`namespace` query 参数、raw JSON body/schema 和当前 200/400/404 响应示例，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_instance/v1/router.go`
- server handler：`internal/apis/disaster_instance/v1/handler.go`，`InstanceHandler.validateRestoreClasses`
- 请求链路：`path.name` -> 可选 `query.namespace` 或 `findNamespace` -> 可选 `BindJSON(ValidateRestoreClassesRequest)` -> `DisasterInstances(namespace).Get` -> `resolveRestoreClassValidationTarget` 按 body、实例 `status.secondaryCluster`、`DisasterConfig.spec.targetCluster` 解析目标集群 -> `getClusterClient` -> `applyStorageClassCheck` 与 `applyIngressClassCheck` -> `WriteSuccess(200, ValidateRestoreClassesDTO)`
- 校验语义：`storageClassMapping` 与 `ingressClassMapping` 至少传一个；`unmatchedPolicy` 只接受 `Keep/Fail/空`；`unmatchedPolicy=Fail` 时 mappings 必须非空；mapping 中 source/target 不能为空；当前重复冲突判断只按 `sourceClass` 判断，不把 `namespaces` 纳入区分
- 返回语义：目标 class 缺失但 `strictTargetValidation=false` 时 `valid=true`；缺失且严格校验为 true 时仍返回 HTTP `200 code=0`，并写入 `data.valid=false`、`data.code`、`data.message` 和 `missingTargets`
- operator 链路：接口本身不触发 reconcile；同一 class mapping 在 operator `restore.ApplyInstanceRestorePolicy` 链路中生成恢复 modifier，StorageClass 映射生成 PVC/PV `/spec/storageClassName` patch，IngressClass 映射生成 Ingress `/spec/ingressClassName` patch
- 下层资源链路：本接口只读目标集群 `StorageClassList` 与 `IngressClassList`；不创建 DataSync、ResourceSync、AppRestore、Velero Restore，也不访问 S3
- 已写入内容：五段详细说明、目标集群解析优先级、全部 body 字段与默认值、`namespaces` 在预检与 operator 中的不同作用、严格/非严格缺失返回语义、完整 `ValidateRestoreClassesDTO` 字段、当前错误分类
- 取证备注：RunAPI 原说明已经包含基础语义和示例，新说明按当前 server/operator 调用链补齐了重复 source、targetCluster 回退、目标集群 client 错误、List class 错误和 operator modifier 规则来源，并把旧说明完整保留
- 主要错误点：JSON 解析失败、未传映射、非法 `unmatchedPolicy`、`unmatchedPolicy=Fail` 但 mappings 为空、source/target 为空或重复 source 冲突返回 `400 code=1000`；无法解析目标集群、目标集群 client 获取失败或列出目标 class 失败返回 `400 code=1000`；实例不存在返回 `404 code=3004`；namespace 自动解析或读取实例非 NotFound 错误返回 `500 code=5000`；JWT 失败由中间件返回

## GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/sync-status

- RunAPI Target ID：`1c01f74a5e401000`
- RunAPI 状态：已存在，已更新详细说明，URL 已从样例实例名规范为 `{{baseurl}}/apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/sync-status`，已补齐 `Authorization` header、`name` path 参数、`namespace` query 参数和当前 200/404 响应示例；本次追加新版 `### 响应参数说明 (Data)`，补齐 `dataSync.lastSyncStatus.*` 与 `resourceSync.lastSyncStatus.*` 全量字段，并追加当前成功示例，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_instance/v1/router.go`
- server handler：`internal/apis/disaster_instance/v1/handler.go`，`InstanceHandler.getSyncStatus`
- 请求链路：`path.name` -> 可选 `query.namespace` 或 `findNamespace` -> `DisasterInstances(namespace).Get` -> 按实例 `status.dataSyncName` 读取 `DataSync` -> 按实例 `status.resourceSyncName` 读取 `ResourceSync` -> 对每个子资源按 UID label `disaster.io/scope-uid=<uid>` 读取 `BackupRestoreStatistics` -> 从 DataSync/ResourceSync `status.history` 选择最新记录并投影为 `lastSyncStatus` -> `WriteSuccess(200, SyncStatusDTO)`
- 当前容错语义：`findNamespace` 或实例 Get 的任何错误都按 `404 code=3004` 返回；DataSync/ResourceSync 读取失败不会返回错误，只省略对应对象；BackupRestoreStatistics 读取失败不会返回错误，只使统计保持 `0`
- operator 链路：`DisasterInstanceReconciler` 创建并维护 DataSync/ResourceSync 名称和实例 LastSyncTime；`DataSyncReconciler` 与 `ResourceSyncReconciler` 驱动 AppBackup/AppRestore/Velero 并写入子资源 status/history/conditions；两个 reconciler 同步写入 `BackupRestoreStatistics`
- 下层资源链路：本接口不直接访问源/目标业务集群、Velero 或 S3；这些下层结果只通过 DataSync、ResourceSync、BackupRestoreStatistics 反映
- 已写入内容：五段详细说明、path/query/header 参数、完整 `SyncStatusDTO` 与 `SubResourceStatusDTO` 字段、`data.dataSync.lastSyncStatus`、`data.resourceSync.lastSyncStatus`、reason/message 只在 Failed 状态回显的当前逻辑、history 匹配规则、统计 CR 聚合规则、子资源读取失败省略字段的当前行为、错误分类
- 取证备注：RunAPI 旧示例包含 `code=200/msg=success`、`lastBackupTime/lastSyncTime` 等历史字段；当前 server 统一 envelope 为 `code=0/message=OK/trace_id`，时间字段为 `lastTime`，旧说明和旧示例已按规则保留
- 主要错误点：JWT 失败由中间件返回 `401 code=2001`；namespace 自动解析失败返回 `404 code=3004`；实例读取失败返回 `404 code=3004`；DataSync、ResourceSync、BackupRestoreStatistics 或 operator 异步同步失败不会让本接口 HTTP 失败

## GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/sync-history

- RunAPI Target ID：`1c8428770dc01001`
- RunAPI 状态：缺失，已新增到 `容灾云平台 / V2 / 容灾实例配置`，URL 为 `{{baseurl}}/apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/sync-history`，已补齐 `Authorization` header、`namespace/syncType/status/source/page/limit` query 参数和当前 200/400 响应示例，已回读验证
- server 路由：`internal/apis/disaster_instance/v1/router.go`
- server handler：`internal/apis/disaster_instance/v1/handler.go`，`InstanceHandler.getSyncHistory`
- 请求链路：`path.name` -> 可选 `query.namespace` 或 `findNamespace` -> `DisasterInstances(namespace).Get` -> 校验并归一化 `syncType/status/source` -> 当 `source=syncRecord/all` 时读取 DataSync 与 ResourceSync `status.history` -> 当 `source=operation/all` 时按实例 label 读取 `DisasterOperation` 并过滤 `syncdata/syncresource/synconce` -> 过滤 -> 固定排序 -> 分页 -> 写入 `meta.summary`
- 统计语义：`meta.summary.totalCount/dataSyncCount/resourceSyncCount/completedCount/failedCount` 均基于过滤后、分页前集合计算；`source=operation` 的 `synconce` 投影为 `syncType=syncOnce`
- operator 链路：接口只读 DataSync、ResourceSync 和 DisasterOperation 的 status；同步执行仍由 DataSyncReconciler、ResourceSyncReconciler 和 DisasterOperationReconciler 异步维护
- 下层资源链路：不创建同步动作，不访问业务集群、Velero、S3、AppBackup 或 AppRestore；无同步历史时返回 `200 data.items=[]`
- 已写入内容：五段详细说明、全部 query 参数、`SyncHistoryItemDTO` 字段、排序和分页规则、summary 口径、source 语义、当前错误分类
- 主要错误点：非法 `source/syncType/status` 返回 `400 code=1000`；JWT 失败由中间件返回 `401 code=2001`；namespace 自动解析或实例读取失败返回 `404 code=3004`；`source=operation/all` 时 DisasterOperation List 失败返回 `500 code=5000`

## GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/validate-target

- RunAPI Target ID：`1c7b171878001001`
- RunAPI 状态：缺失，已新增到 `容灾云平台 / V1 / 容灾实例`，已补齐 `Authorization` header、`name` path 参数、`namespace/operation` query 参数和当前 200/404/500 响应示例，鉴权类型已设为继承项目鉴权，已回读验证
- 2026-05-18 RunAPI 追加响应示例：`同步失败允许数据同步重试`，覆盖 `Failed` 可恢复同步失败下 `validate-target` 返回 `valid=true` 的兼容语义
- 2026-05-22 代码与 OpenAPI 更新：`FailingOver` 状态下 `operation=cancel` 返回 `valid=true`，并在 `availableOperations` 中补入 `cancel`；RunAPI live 已补充 `FailingOver取消允许执行` 200 响应示例，并更新 `operation` query 参数说明
- server 路由：`internal/apis/disaster_instance/v1/router.go`
- server handler：`internal/apis/disaster_instance/v1/handler.go`，`InstanceHandler.validateTarget`
- 请求链路：`path.name` -> `query.operation` trim/lower -> 可选 `query.namespace` 或 `findNamespace` -> `DisasterInstances(namespace).Get` -> `findContainingGroups` 列出同 namespace `DisasterGroup` 并遍历 `spec.levels` -> `validateInstanceOperationAllowed` 比较 operation 与实例有效 `availableOperations` -> `WriteSuccess(200, ValidateTargetDTO)`
- 当前校验语义：`operation` 为空时直接 `valid=true`；`sync-data` 归一化为 `syncdata`，`sync-resource` 归一化为 `syncresource`；当 `Failed` 实例 reason 属于可恢复同步失败时，server 会把同步重试补入有效操作集；当实例为 `FailingOver` 时，server 会把 `cancel` 补入有效操作集；operation 不在有效 `availableOperations` 时返回 `200 code=0 data.valid=false/reason=OperationNotAllowed`，不会返回 HTTP `409`
- operator 链路：`DisasterInstanceReconciler` 根据状态机写入 `status.fsmState` 与 `status.availableOperations`；常见写入包括 Protected 下 `failover/pause/synconce/syncdata/syncresource`，Paused 下 `resume`，Active 下 `reprotect/undo`，同步失败类 Failed 下 `reset` 加对应的 `syncdata/syncresource`，其它 Failed 下 `reset`
- 组链路：`DisasterGroupReconciler` 维护组状态，但本接口只读 `DisasterGroup.spec.levels` 判断归属；当前 server 不因 `inGroup=true` 自动禁止实例单操作
- 下层资源链路：不创建 `DisasterOperation`，不访问 DataSync、ResourceSync、Velero、S3 或业务集群；真正执行动作由 `POST /instances/:name/actions` 负责
- 已写入内容：五段详细说明、path/query/header 参数、operation 可选和归一化语义、完整 `ValidateTargetDTO` 字段、组归属返回、当前 `valid=false` 仍为 200 的行为、错误分类
- 主要错误点：namespace 自动解析或实例 Get 返回 NotFound 时为 `404 code=3004`；namespace 自动解析非 NotFound、实例 Get 非 NotFound 或列出 `DisasterGroup` 失败返回 `500 code=5000`；JWT 失败由中间件返回

## GET /apis/disasterinstances.testudo.softcdata.com/v1/watch/instances

- RunAPI Target ID：`3b8699c1b8c14b`
- RunAPI 状态：已存在，目标类型保持 `websocket2`，已更新五段详细说明，已补齐 `Authorization`、`Sec-WebSocket-Protocol` header 和 `token` query，已回读验证
- RunAPI 结构备注：`websocket2` 详情不保留 response example 容器；连接成功帧、心跳帧、事件帧、错误帧、关闭帧和超时帧已写入详细说明
- server 路由：`internal/apis/disaster_instance/v1/router.go`
- server handler：`internal/apis/disaster_instance/v1/handler.go`，`InstanceHandler.watchInstances`
- 请求链路：`WebSocketTokenAdapter` -> JWT 中间件 -> `upgrader.Upgrade` -> 创建 `DisasterInstances("").Watch`、`DataSyncs("").Watch`、`ResourceSyncs("").Watch` -> `NewMultiWatcher` 聚合事件 -> `StreamWatch` 连接成功帧、30 秒心跳帧、watch event 帧、关闭/超时帧
- 转换链路：`DisasterInstance` 事件直接转换为 `DisasterInstanceDTO`；`DataSync` 事件要求名称前缀 `dr-ds-` 并反查同 namespace 父实例；`ResourceSync` 事件要求名称前缀 `dr-rs-` 并反查同 namespace 父实例；转换后调用 `enrichListSyncStatus` 补充同步摘要
- operator 链路：接口本身不触发 reconcile；`DisasterInstanceReconciler`、`DataSyncReconciler`、`ResourceSyncReconciler` 的 status/spec 更新通过 Kubernetes watch 表现为事件流
- 下层资源链路：不直接访问业务集群、Velero 或 S3；下层执行结果由 operator 写入实例和同步子资源状态后被 watch 推送
- 已写入内容：五段详细说明、三种 WebSocket 鉴权方式、全 namespace 三 watcher 聚合语义、DataSync/ResourceSync 事件反查父实例规则、`data.object=null` 场景、默认 30 秒心跳和 30 分钟超时、WebSocket 内外错误分类
- 主要错误点：握手前 JWT 失败返回 HTTP `401 code=2001`；升级失败返回普通 HTTP `500 {"message": ...}`；任一 watcher 创建失败通过 WebSocket 返回 `code=5000`；watcher 关闭或连接超时返回 `code=0` meta 帧；父实例反查失败只使 `data.object=null`

## GET /apis/disasterinstances.testudo.softcdata.com/v1/watch/instances/:name

- RunAPI Target ID：`3b86fa34f8c221`
- RunAPI 状态：已存在，目标类型保持 `websocket2`，URL 已从样例实例名规范为 `{{baseurl}}/apis/disasterinstances.testudo.softcdata.com/v1/watch/instances/:name`，已补齐 `Authorization`、`Sec-WebSocket-Protocol` header 和 `token` query，已回读验证
- RunAPI 结构备注：`websocket2` 详情未显示 restful 容器，path `name` 已完整写入详细说明
- server 路由：`internal/apis/disaster_instance/v1/router.go`
- server handler：`internal/apis/disaster_instance/v1/handler.go`，`InstanceHandler.watchInstance`
- 请求链路：`WebSocketTokenAdapter` -> JWT 中间件 -> `upgrader.Upgrade` -> `findNamespace(name)` 尝试定位 namespace，失败时忽略并使用空 namespace -> 创建 `DisasterInstances(ns).Watch(FieldSelector: metadata.name=<name>)`、`DataSyncs(ns).Watch(FieldSelector: metadata.name=dr-ds-<name>)`、`ResourceSyncs(ns).Watch(FieldSelector: metadata.name=dr-rs-<name>)` -> `NewMultiWatcher` -> `StreamWatch`
- 转换链路：`DisasterInstance` 事件直接转换为 `DisasterInstanceDTO` 并补同步摘要；DataSync/ResourceSync 事件使用事件对象 namespace 和 path `name` 重新读取父实例，成功后转换并补同步摘要，失败时返回 `null`
- operator 链路：接口本身不触发 reconcile；实例与同步子资源的 operator 更新通过 watch 事件触发推送
- 下层资源链路：不直接访问业务集群、Velero 或 S3；只监听管理集群 CRD 事件
- 已写入内容：五段详细说明、三种 WebSocket 鉴权方式、path `name` 与固定子资源名称规则、`findNamespace` 失败退化为 all-namespaces watch 的当前行为、帧结构、默认 30 秒心跳和 30 分钟超时、WebSocket 内外错误分类
- 主要错误点：握手前 JWT 失败返回 HTTP `401 code=2001`；升级失败返回普通 HTTP `500 {"message": ...}`；任一 watcher 创建失败通过 WebSocket 返回 `code=5000`；watcher 关闭或连接超时返回 `code=0` meta 帧；`findNamespace` 失败和父实例反查失败不会直接返回错误

## GET /apis/disasterinstances.testudo.softcdata.com/v1/watch/instances/operations/:operationName

- RunAPI Target ID：`1c5eadf4f1801001`
- RunAPI 状态：已存在，历史目标类型保持 `api`，已更新详细说明，已补齐 `Authorization`、`Sec-WebSocket-Protocol` header、`operationName` path 参数、`namespace/token` query 参数和当前帧示例，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_instance/v1/router.go`
- server handler：`internal/apis/disaster_instance/v1/handler_operation.go`，`InstanceHandler.watchOperation`
- 请求链路：`WebSocketTokenAdapter` -> JWT 中间件 -> 可选 `query.namespace`；namespace 为空时 `findOperationNamespace(operationName)`，先查 `disaster-system` 同名 operation，再 list 全 namespace 按名称匹配 -> `upgrader.Upgrade` -> `DisasterOperations(namespace).Watch(FieldSelector: metadata.name=<operationName>)` -> `StreamWatch`
- 转换链路：watch event object 为 `DisasterOperation` 时调用 `ConvertToOperationDetailDTO`；非 operation 对象返回 `null`
- 当前范围：handler 只有 `operationName`，不校验实例名；因此该 URL 下也可能推送 `ownerKind=DisasterGroup` 的 operation，归属以 `ownerKind/ownerName` 为准
- operator 链路：`DisasterOperationReconciler` 在动作执行过程中维护 `status.state/reason/currentStep/message/steps/autoCancel/roleStatus/groupStatus/startTime/completionTime`；本接口只监听并转换这些状态
- 下层资源链路：不直接访问 DataSync、ResourceSync、业务集群、Velero 或 S3；下层结果通过 operation status 体现
- 已写入内容：五段详细说明、三种 WebSocket 鉴权方式、operation namespace 自动查找顺序、显式 namespace 不做同步存在性校验的语义、完整 `OperationDetailDTO` 字段、默认 30 秒心跳和 30 分钟超时、WebSocket 内外错误分类
- 取证备注：RunAPI 原说明已保留；当前 RunAPI target_type 仍是 `api`，但说明和响应示例均按 WebSocket 行为写入
- 主要错误点：握手前 JWT 失败返回 HTTP `401 code=2001`；未传 namespace 且自动查找失败返回 `404 code=3004`；升级失败返回普通 HTTP `500 {"message": ...}`；watcher 创建失败通过 WebSocket 返回 `code=5000`；显式 namespace 下 operation 不存在时可能连接成功但无业务事件

## GET /apis/disasterdrills.testudo.softcdata.com/v1/drills

- RunAPI Target ID：`1c0aec042dc01001`
- RunAPI 状态：已存在，已更新详细说明，已补齐 `Authorization` header、namespace/keyword/type/state/instanceName/groupName/limit/page/sort/order query 参数和当前 200/500 响应示例，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_drill/v1/router.go`
- server handler：`internal/apis/disaster_drill/v1/handler.go`，`DrillHandler.listDrills`
- 请求链路：`transport.ParseOptions` -> 取 `qParams.Filters["namespace"]` -> `DisasterDrills(ns).List` -> 内存中过滤 `keyword/type/instanceName/groupName/state` -> 按 `creationTimestamp` 倒序排序 -> `transport.Paginate` -> `ConvertToDisasterDrillDTO` -> `BuildCollectionResponse(resourceType=disasterDrill)` -> `WriteSuccess(200)`
- 过滤语义：`keyword` 只匹配演练名称；`type=Instance` 要求 `spec.instanceName` 非空，`type=Group` 要求 `spec.groupName` 非空；`state=NotStarted` 匹配 `Pending/Ready`，`state=InProgress` 匹配 `Executing`，其他值精确匹配 CRD `status.state`
- 当前排序语义：handler 固定按 `creationTimestamp` 倒序，未使用 query `sort/order`；`sort/order` 只进入 meta
- operator 链路：`DisasterDrillReconciler` 初始化 `Pending`、执行前置校验进入 `Ready`、确认后进入 `Executing`、维护 `operationName/steps/validationResults/groupProgress`，并处理 `Completed/Failed/CleaningUp/CleanedUp`；列表接口只读这些 status 结果
- 下层资源链路：列表接口不读取实例、组、operation、AppRestore、业务集群、Velero 或 S3；下层状态通过 Drill status 投影
- 已写入内容：五段详细说明、全部 query 参数和默认值、集合响应 envelope、完整 `DisasterDrillDTO` 字段、当前状态过滤映射、当前返回结构为 `data.items[].status.*` 的事实、operator 状态来源和当前错误分类
- 取证备注：RunAPI 旧示例包含历史平铺字段 `state/restoreMode/currentStep` 和数组型 `data`；当前 DTO 代码实际返回 `status` 对象且集合数据在 `data.items[]`，新说明和新增示例按当前实现写入，旧说明/旧示例保留
- 主要错误点：JWT 失败由中间件返回 `401 code=2001`；`DisasterDrills(ns).List` 失败返回 `500 code=5000`；过滤无匹配返回 `200 code=0 data.items=[]`

## POST /apis/disasterdrills.testudo.softcdata.com/v1/drills

- RunAPI Target ID：`1c0aec0a8f401001`
- RunAPI 状态：已存在，已更新详细说明，已补齐 `Authorization`、`Content-Type` header、raw JSON body/schema、当前 201/400/404/409/500 响应示例，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证；2026-05-22 已追加 `ModifierRuleRejected：replace 缺失最终路径仍拒绝` 400 响应示例
- server 路由：`internal/apis/disaster_drill/v1/router.go`
- server handler：`internal/apis/disaster_drill/v1/handler.go`，`DrillHandler.createDrill`
- 请求链路：`BindJSON(CreateDrillRequest)` -> 校验 `instanceName/groupName` 二选一且不能同时传 -> 校验 `name` 长度不超过 63 -> 校验 `veleroHooks.dataBackup` 禁止、`veleroHooks.dataRestore` 合法性，并保留顶层 `veleroHooks:{}` 为空对象覆盖语义 -> 默认 `namespace=disaster-system` -> 按实例或组在指定 namespace 查询，NotFound 时全 namespace 按名称回退查询并切换到目标对象真实 namespace -> `buildPreparedDrillRestorePolicy` -> 自动生成或使用请求演练名 -> 写入 `DisasterDrill` -> `WriteSuccess(201, DisasterDrillDTO)`
- restorePolicy 准备链路：请求体 `restorePolicy` 先执行 `RestorePolicyRequest.ToCRD`，解析 `modifierRulesText`、`bulkModifierActionsText`，校验 resourceSelection include/exclude 冲突；随后复用实例模块 `PrepareRestorePolicyForPersist` 归一化 bulk action、生成 `modifierRuleSnapshot/modifierRuleSnapshotHash`、校验 modifier rule 与可选 live 校验；live path 校验按 JSON Patch 语义处理，`add` 允许最终一级 map key 不存在但父路径必须存在，`replace/remove` 要求完整路径存在
- 组演练 restorePolicy 额外链路：读取 `DisasterGroup.spec.levels` 内所有实例，读取各实例 `spec.config` 指向的 `DisasterConfig.spec.sourceCluster`；成员实例必须能解析到唯一 sourceCluster，否则创建阶段返回 `400`
- operator 链路：`DisasterDrillReconciler` 为 Drill 加 finalizer、同步依赖标签，首次 reconcile 初始化 `Pending/startTime/message`；Pending 校验实例或组、目标集群和 Cluster Ready 后写入 `Ready/restoreMode=FullRestore`；确认后创建 `OperationType=drill` 的 `DisasterOperation`，并将 `drill.spec.veleroHooks` 原样复制到 `operation.spec.drillConfig.veleroHooks`；Operation 继续执行 `RestoreResource/RestoreData/ScaleUp`
- 下层资源链路：创建接口不直接访问业务集群、Velero、S3、Deployment、PVC 或 Namespace；后续 Operation 会读取 DataSync/ResourceSync 最近备份、创建 AppRestore，并在资源/数据恢复时应用实例 restorePolicy 与演练级 restorePolicy override
- 已写入内容：五段详细说明、全部 body/header 参数、二选一和默认 namespace 语义、目标实例/组 namespace 回退逻辑、演练级 restorePolicy 覆盖与继承语义、`veleroHooks:{}` 清空继承语义、bulk action 与 modifier rule 的取值/约束、完整 `DisasterDrillDTO` 字段、创建即时响应 `status.state` 可能为空的当前事实、server 同步错误与 operator 异步失败边界
- 取证备注：RunAPI 旧说明已包含基础示例和 restorePolicy 概要，但成功状态码和响应结构仍是历史平铺字段；当前 server 实际返回 HTTP `201`，`data.status` 为对象，且 Kubernetes Create 后立即返回时 status 可能尚未由 operator 初始化，因此新说明和默认成功示例按当前实现写入，旧说明和旧示例保留。显式 `veleroHooks:{}` 会写入 Drill CR 的非 nil 空对象，使后续 Operation 保留清空覆盖，避免继承实例级 dataRestore Hook
- 主要错误点：JSON 解析失败、`instanceName/groupName` 同传或都不传、`name` 超 63 字符、restorePolicy 文本 JSON 非法、结构化与文本输入冲突、resourceSelection 冲突、modifier rule/bulk action 非法或 live 校验失败（例如 `add` 父路径不存在、`replace/remove` 路径不存在）、组演练 restorePolicy 成员实例 sourceCluster 不唯一返回 `400 code=1000`；目标实例或组不存在返回 `404 code=3004`；同名 Drill 已存在返回 `409 code=3009`；目标对象 namespaced Get 非 NotFound 或创建 Drill 非 AlreadyExists 错误返回 `500 code=5000`；JWT 失败由中间件返回 `401 code=2001`

## DELETE /apis/disasterdrills.testudo.softcdata.com/v1/drills/:name

- RunAPI Target ID：`1c0aec15e5001001`
- RunAPI 状态：已存在，已更新详细说明，URL 已从样例名称规范为 `{{baseurl}}/apis/disasterdrills.testudo.softcdata.com/v1/drills/:name`，已补齐 `Authorization` header、`name` path 参数、`namespace` query 参数和当前 200/500 响应示例，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_drill/v1/router.go`
- server handler：`internal/apis/disaster_drill/v1/handler.go`，`DrillHandler.deleteDrill`
- 请求链路：`path.name` -> 可选 `query.namespace`；namespace 为空时 `findNamespace(name)` 通过全 namespace List + `metadata.name=<name>` 定位；定位失败直接 `WriteSuccess(200, {"name": name})` -> `DisasterDrills(namespace).Delete` -> Delete 成功返回 `{"name": name, "deleted": true}`，Delete NotFound 返回 `{"name": name}`
- operator 链路：删除请求使 `DisasterDrill` 进入 deletionTimestamp；`DisasterDrillReconciler.handleDeletion` 报告删除任务、移除 finalizer 并记录删除事件；确认/清理时创建的 `DisasterOperation` 使用 Drill ownerReference，Kubernetes GC 会级联删除这些 Operation
- 下层资源链路：本接口不直接清理业务集群 Namespace、Deployment、PVC/PV、AppRestore、Velero Restore 或 S3 备份；恢复出的演练资源需要通过 cleanup 流程触发，删除 Drill 不等同于执行演练资源清理
- 已写入内容：五段详细说明、path/query/header 参数、幂等删除返回结构、`deleted=true` 只在 Delete 成功时出现、自动 namespace 解析失败也返回成功的当前行为、finalizer 与 ownerReference 级联边界、当前错误分类
- 取证备注：RunAPI 原 URL 使用样例 `drills01`，已规范为 `:name`；原说明包含“删除即清理资源”的宽泛表述，新说明明确当前 handler 只删除管理集群 Drill CR，不同步触发目标集群清理
- 主要错误点：JWT 失败由中间件返回 `401 code=2001`；显式 namespace 或自动定位成功后 Delete 返回非 NotFound 错误时返回 `500 code=5000`；Drill 不存在、未传 namespace 且 `findNamespace` 失败、或 Delete 返回 NotFound 均返回 `200 code=0`

## GET /apis/disasterdrills.testudo.softcdata.com/v1/drills/:name

- RunAPI Target ID：`1c0aec0a6cc01001`
- RunAPI 状态：已存在，已更新详细说明，URL 已从样例名称规范为 `{{baseurl}}/apis/disasterdrills.testudo.softcdata.com/v1/drills/:name`，已补齐 `Authorization` header、`name` path 参数、`namespace` query 参数和当前 200/404/500 响应示例，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_drill/v1/router.go`
- server handler：`internal/apis/disaster_drill/v1/handler.go`，`DrillHandler.getDrill`
- 请求链路：`path.name` -> 可选 `query.namespace`；namespace 为空时 `findNamespace(name)` 通过全 namespace List + `metadata.name=<name>` 定位 -> `DisasterDrills(namespace).Get(name)` -> `ConvertToDisasterDrillDTO` -> `WriteSuccess(200, DisasterDrillDTO)`
- operator 链路：接口本身不触发 reconcile；`DisasterDrillReconciler` 维护 `status.state/reason/restoreMode/operationName/currentStep/message/startTime/readyTime/executionTime/completionTime/validationResults/steps/groupProgress`；`DisasterOperationReconciler` 执行 Drill Operation 后，Drill Reconciler 将 Operation 步骤与组进度同步到 Drill status
- 下层资源链路：详情接口不直接访问业务集群、Velero、S3、DataSync、ResourceSync、AppRestore 或 Cluster；下层状态通过 operator 写入 `DisasterDrill.status` 后再由本接口读取
- 已写入内容：五段详细说明、path/query/header 参数、单对象 envelope、完整 `DisasterDrillDTO` 字段、当前返回结构为 `data.status.*` 的事实、状态字段来源、restorePolicy 回显字段、operator 异步失败只体现在状态字段中的说明、当前错误分类
- 取证备注：RunAPI 旧 URL 使用真实样例名，且旧示例为 `data.state/restoreMode/currentStep` 平铺字段；当前 server DTO 实际返回 `status` 对象，新说明和默认成功示例按当前实现写入，旧说明和旧示例保留
- 主要错误点：JWT 失败由中间件返回 `401 code=2001`；未传 namespace 且 `findNamespace` 找不到或显式 namespace 下 Get NotFound 返回 `404 code=3004`；namespace 自动解析 List 失败或 Get 非 NotFound 错误返回 `500 code=5000`

## POST /apis/disasterdrills.testudo.softcdata.com/v1/drills/:name/cleanup

- RunAPI Target ID：`1c28bf2907c01000`
- RunAPI 状态：已存在，已更新详细说明，已补齐 `Authorization` header、`name` path 参数、`namespace` query 参数和当前 200/400/404/500 响应示例，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_drill/v1/router.go`
- server handler：`internal/apis/disaster_drill/v1/handler.go`，`DrillHandler.cleanupDrill`
- 请求链路：`path.name` -> 可选 `query.namespace` 或 `findNamespace` -> `RetryOnConflict` -> `DisasterDrills(namespace).Get` -> 校验 `status.state == Completed` -> 校验 `spec.cleanup == false` -> 写入 `spec.cleanup=true` -> 注入 trace annotation -> `DisasterDrills(namespace).Update` -> `ConvertToDisasterDrillDTO` -> `WriteSuccess(200)`
- operator 链路：HTTP handler 只写 cleanup 标记；`DisasterDrillReconciler` 在 Completed 且 `spec.cleanup=true` 时创建 `OperationType=drill-cleanup`，把 Drill 状态置为 `CleaningUp` 并写入清理 `operationName`；清理 Operation 完成后 Drill 状态变为 `CleanedUp`，失败则变为 `Failed`
- 下层资源链路：`DisasterOperationReconciler.handleDrillCleanup` 读取关联 `DisasterInstance` 和目标集群；无 `namespaceMapping` 时对目标集群原命名空间工作负载执行缩容；有 `namespaceMapping` 时删除映射后的目标命名空间；本接口自身不直接访问业务集群、Velero、S3、AppRestore、Deployment 或 Namespace
- 已写入内容：五段详细说明、path/query/header 参数、只允许 `Completed` 状态清理、重复 cleanup 拒绝、成功响应 `cleanup=true` 但 `status.state` 可能仍为 `Completed` 的异步边界、清理 Operation 的真实下层资源行为、当前错误分类
- 取证备注：RunAPI 旧示例把成功响应直接写成 `state=CleaningUp` 且使用历史平铺字段；当前 handler 更新 spec 后立即返回，operator 通常尚未同步状态，所以新说明和默认示例按 `data.status` 对象与异步状态变更写入，旧说明和旧示例保留
- 主要错误点：JWT 失败由中间件返回 `401 code=2001`；Drill 不存在返回 `404 code=3004`；状态不是 `Completed` 或 cleanup 已触发返回 `400 code=1000`；当前 handler 在 RetryOnConflict 返回错误且 result 为空时会按 `400 code=1000` 返回，可能包含部分 Get/Update 非 NotFound 错误；namespace 自动解析非 NotFound 错误返回 `500 code=5000`；operator 后续目标集群连接、缩容或删除 namespace 失败只体现在异步状态中

## POST /apis/disasterdrills.testudo.softcdata.com/v1/drills/:name/confirm

- RunAPI Target ID：`1c0aec15bc001001`
- RunAPI 状态：已存在，已更新详细说明，URL 已从样例名称规范为 `{{baseurl}}/apis/disasterdrills.testudo.softcdata.com/v1/drills/:name/confirm`，已补齐 `Authorization` header、`name` path 参数、`namespace` query 参数和当前 200/400/404/500 响应示例，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_drill/v1/router.go`
- server handler：`internal/apis/disaster_drill/v1/handler.go`，`DrillHandler.confirmDrill`
- 请求链路：`path.name` -> 可选 `query.namespace` 或 `findNamespace` -> `RetryOnConflict` -> `DisasterDrills(namespace).Get` -> 校验 `status.state == Ready` -> 如果 `spec.confirmed=true` 直接返回现有对象 -> 否则写入 `spec.confirmed=true` -> 注入 trace annotation -> `DisasterDrills(namespace).Update` -> `ConvertToDisasterDrillDTO` -> `WriteSuccess(200)`
- operator 链路：HTTP handler 只写 confirmed 标记；`DisasterDrillReconciler.handleReady` 在 Ready 且 confirmed=true 时创建 `OperationType=drill` 的 `DisasterOperation`，实例演练写入 `instanceName`，组演练写入 `groupName`，并把 Drill 状态置为 `Executing`、写入 `operationName/executionTime/message`
- 下层资源链路：`DisasterOperationReconciler` 执行 `RestoreResource/RestoreData/ScaleUp`；资源/数据恢复读取 ResourceSync/DataSync 最近备份并创建 AppRestore，ScaleUp 访问目标集群工作负载；Drill 级 restorePolicy 作为 `drillConfig.restorePolicy` 覆盖层传入 `ApplyInstanceRestorePolicy(applyTo=drill)`
- 已写入内容：五段详细说明、path/query/header 参数、只允许 `Ready` 状态确认、重复 confirmed 在 Ready 下幂等返回、成功响应 `confirmed=true` 但 `status.state` 可能仍为 `Ready` 的异步边界、Operation 创建和下层恢复/扩容链路、当前错误分类
- 取证备注：RunAPI 旧 URL 使用样例 `drills1`，旧说明写成确认后状态直接变为 `Executing`；当前 handler 只是写 spec，operator 异步更新状态，所以新说明和默认示例按即时响应仍可能为 `Ready` 写入，旧说明和旧示例保留
- 主要错误点：JWT 失败由中间件返回 `401 code=2001`；Drill 不存在返回 `404 code=3004`；状态不是 `Ready` 返回 `400 code=1000`；当前 handler 在 RetryOnConflict 返回错误且 result 为空时会按 `400 code=1000` 返回，可能包含部分 Get/Update 非 NotFound 错误；namespace 自动解析非 NotFound 错误返回 `500 code=5000`；operator 后续拓扑变化、备份不可用、AppRestore 或 ScaleUp 失败只体现在异步状态中

## POST /apis/disasterdrills.testudo.softcdata.com/v1/drills/:name/restart

- RunAPI Target ID：`1c0c8731f1401001`
- RunAPI 状态：已存在，已更新详细说明，URL 已从样例名称规范为 `{{baseurl}}/apis/disasterdrills.testudo.softcdata.com/v1/drills/:name/restart`，已补齐 `Authorization` header、`name` path 参数、`namespace` query 参数和当前 200/400/404/500 响应示例，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_drill/v1/router.go`
- server handler：`internal/apis/disaster_drill/v1/handler.go`，`DrillHandler.restartDrill`
- 请求链路：`path.name` -> 可选 `query.namespace` 或 `findNamespace` -> `RetryOnConflict` -> `DisasterDrills(namespace).Get` -> 校验 `status.state` 为 `Completed` 或 `Failed` -> 写入 annotation `testudo.softcdata.com/restart-timestamp=<RFC3339 now>` -> 注入 trace annotation -> `DisasterDrills(namespace).Update` -> `ConvertToDisasterDrillDTO` -> `WriteSuccess(200)`
- operator 链路：HTTP handler 只写 restart annotation；`DisasterDrillReconciler.shouldRestart` 在 Completed/Failed 状态比较 restart timestamp 与 `status.completionTime`，满足后调用 `resetDrill`；若 `spec.confirmed=true`，先更新为 false 并重新排队，再清空 status 并写入 `Pending/startTime/message=演练已重置，重新开始校验...`
- 下层资源链路：重跑接口不直接访问业务集群、Velero、S3、DataSync、ResourceSync、AppRestore 或工作负载；reset 后重新走 Pending 校验，Ready 后需要再次调用 confirm 才创建新的 Drill Operation
- 已写入内容：五段详细说明、path/query/header 参数、只允许 `Completed/Failed` 重跑、restart annotation 名称与时间语义、成功响应 `status.state` 可能仍为 `Completed/Failed` 的异步边界、`spec.confirmed` 后续被 reset 为 false、CleanedUp 且 cleanup=true 不会直接重跑的边界、当前错误分类
- 取证备注：RunAPI 原 URL 使用样例 `drills15`，旧说明写成响应即返回重置后的 `Pending`；当前 handler 只更新 annotation，operator 异步 reset，所以新说明和默认示例按当前即时响应写入，旧说明和旧示例保留
- 主要错误点：JWT 失败由中间件返回 `401 code=2001`；Drill 不存在返回 `404 code=3004`；状态不是 `Completed` 或 `Failed` 返回 `400 code=1000`；当前 handler 在 RetryOnConflict 返回错误且 result 为空时会按 `400 code=1000` 返回，可能包含部分 Get/Update 非 NotFound 错误；namespace 自动解析非 NotFound 错误返回 `500 code=5000`；operator 后续重置失败或重置后的前置校验失败只体现在异步状态中

## GET /apis/disasterdrills.testudo.softcdata.com/v1/drills/actions/protected-namespaces

- RunAPI Target ID：`1c380061ca801001`
- RunAPI 状态：已存在，已更新详细说明，已补齐 `Authorization` header、`instanceName/groupName/namespace` query 参数和当前 200/400/404/500 响应示例，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证；2026-05-21 已再次更新 live RunAPI，修正 namespace 空值默认配置管理命名空间且不再全 namespace 回退，并新增当前 500 示例
- server 路由：`internal/apis/disaster_drill/v1/router.go`
- server handler：`internal/apis/disaster_drill/v1/handler.go`，`DrillHandler.getProtectedNamespaces`
- 请求链路：trim `query.instanceName/groupName/namespace` -> namespace 为空时使用 `common.DisasterSystemNamespace` -> 校验 `instanceName` 与 `groupName` 必须且只能传一个 -> 实例模式读取 `DisasterInstances(namespace).Get(instanceName)` -> 返回 `uniqueSortedStrings(inst.Spec.Namespaces)`；组模式读取 `DisasterGroups(namespace).Get(groupName)` -> `flattenAndDedupeGroupInstances(group.Spec.Levels)` -> 在组 namespace 逐实例读取并聚合 namespaces -> 返回 `ProtectedNamespacesDTO`
- 资源查找语义：容灾实例和容灾组按平台管理命名空间查询；不传 namespace 时使用配置的管理命名空间（默认 `disaster-system`），传入时使用指定 namespace；该接口不再全 namespace 回退查找实例或组；非 NotFound 错误返回 500；组模式读取组内实例时使用 group namespace 查询
- operator 链路：接口本身不触发 reconcile；读取 `DisasterInstance.spec.namespaces` 和 `DisasterGroup.spec.levels`。DisasterGroup controller 维护组状态，但本接口只读 spec，不读取组 status
- 下层资源链路：不创建 Drill，不创建 Operation，不访问业务集群、Velero、S3、DataSync、ResourceSync、AppRestore 或 Cluster；下层资源只通过实例 spec.namespaces 间接体现
- 已写入内容：五段详细说明、二选一 query 参数约束、namespace 空值默认配置管理命名空间、实例/组两种返回结构、namespaces 去空去重排序、组内实例缺失不失败而返回 `missingInstances` 的当前行为、当前错误分类
- 取证备注：RunAPI 原说明已有基本参数约束，新说明已修正为平台管理命名空间查询语义，并记录不再全 namespace 回退查找实例或组；同时保留组内缺失实例容忍语义和返回字段来源
- 主要错误点：JWT 失败由中间件返回 `401 code=2001`；`instanceName/groupName` 都空或同传返回 `400 code=1000`；目标实例或目标组不存在返回 `404 code=3004`；CRD Get/List 非 NotFound 错误或组内实例读取非 NotFound 错误返回 `500 code=5000`；组内实例 NotFound 不会返回错误，会进入 `data.missingInstances`

## GET /apis/disasterdrills.testudo.softcdata.com/v1/watch/drills

- RunAPI Target ID：`477a33c78c1c8`
- RunAPI 状态：已存在，目标类型保持 `websocket2`，已更新五段详细说明，已补齐 `Authorization`、`Sec-WebSocket-Protocol` header 和 `token` query，已回读验证
- RunAPI 结构备注：`websocket2` 详情不保留 response example 容器；连接成功帧、心跳帧、事件帧、错误帧、关闭帧和超时帧已写入详细说明
- server 路由：`internal/apis/disaster_drill/v1/router.go`
- server handler：`internal/apis/disaster_drill/v1/handler.go`，`DrillHandler.watchDrills`
- 请求链路：`WebSocketTokenAdapter` -> JWT 中间件 -> `upgrader.Upgrade` -> `DisasterDrills("").Watch(ListOptions{})` -> `StreamWatch` 连接成功帧、30 秒心跳帧、watch event 帧、关闭/超时帧
- 转换链路：watch event object 为 `DisasterDrill` 时调用 `ConvertToDisasterDrillDTO`；非 Drill 对象返回 `null`；当前 handler 不读取或应用 namespace、instanceName、groupName、state 等过滤 query，固定监听所有 namespace 的 Drill 事件
- operator 链路：接口本身不触发 reconcile；`DisasterDrillReconciler` 和 `DisasterOperationReconciler` 写入 Drill spec/status 后，通过 Kubernetes watch 表现为 `ADDED/MODIFIED/DELETED/ERROR` 事件流
- 下层资源链路：不直接访问业务集群、Velero、S3、DataSync、ResourceSync、AppRestore 或 Cluster；下层执行结果通过 Drill status 变化被 watch 推送
- 已写入内容：五段详细说明、三种 WebSocket 鉴权方式、全 namespace watch 语义、当前无业务过滤的事实、WebSocket envelope 与 `DisasterDrillDTO` 字段、默认 30 秒心跳和 30 分钟超时、WebSocket 内外错误分类
- 主要错误点：握手前 JWT 失败返回 HTTP `401 code=2001`；升级失败返回普通 HTTP `500 {"message": ...}`；watcher 创建失败通过 WebSocket 返回 `code=5000`；watcher 关闭或连接超时返回 `code=0` meta 帧；operator 异步失败只体现在后续事件对象的 status 字段中

## GET /apis/disasterdrills.testudo.softcdata.com/v1/watch/drills/:name

- RunAPI Target ID：`477bda678c1cb`
- RunAPI 状态：已存在，目标类型保持 `websocket2`，已更新五段详细说明，已补齐 `Authorization`、`Sec-WebSocket-Protocol` header 和 `token` query，已回读验证
- RunAPI 结构备注：`websocket2` 详情未稳定显示 restful/path 参数容器，path `name` 已完整写入详细说明；连接成功帧、心跳帧、事件帧、错误帧、关闭帧和超时帧已写入详细说明
- server 路由：`internal/apis/disaster_drill/v1/router.go`
- server handler：`internal/apis/disaster_drill/v1/handler.go`，`DrillHandler.watchDrill`
- 请求链路：`WebSocketTokenAdapter` -> JWT 中间件 -> `path.name` -> `upgrader.Upgrade` -> watcher 创建时调用 `findNamespace(name)`，失败忽略并使用空 namespace -> `DisasterDrills(ns).Watch(FieldSelector: metadata.name=<name>)` -> `StreamWatch`
- 转换链路：watch event object 为 `DisasterDrill` 时调用 `ConvertToDisasterDrillDTO`；非 Drill 对象返回 `null`；当前 handler 不读取 `query.namespace`，即使客户端传入 namespace 也不会影响 watcher namespace
- operator 链路：接口本身不触发 reconcile；指定 Drill 的 spec/status 更新由 `DisasterDrillReconciler` 和 `DisasterOperationReconciler` 写入后，通过该 watch 推送 `ADDED/MODIFIED/DELETED/ERROR` 事件
- 下层资源链路：不直接访问业务集群、Velero、S3、DataSync、ResourceSync、AppRestore 或 Cluster；下层执行结果通过 Drill status 变化被 watch 推送
- 已写入内容：五段详细说明、三种 WebSocket 鉴权方式、path `name` 语义、`findNamespace` 失败退化为 all-namespaces watch 的当前行为、`query.namespace` 当前无效的事实、WebSocket envelope 与 `DisasterDrillDTO` 字段、默认 30 秒心跳和 30 分钟超时、WebSocket 内外错误分类
- 主要错误点：握手前 JWT 失败返回 HTTP `401 code=2001`；升级失败返回普通 HTTP `500 {"message": ...}`；watcher 创建失败通过 WebSocket 返回 `code=5000`；watcher 关闭或连接超时返回 `code=0` meta 帧；Drill 不存在或 namespace 自动定位失败不会返回 404，连接可能成功但无业务事件

## GET /apis/policies.testudo.softcdata.com/v1/policies

- RunAPI Target ID：`35ec1a7278c00b`
- RunAPI 状态：已存在，已更新详细说明，URL 已从带默认 query 的 `{{baseurl}}/apis/policies.testudo.softcdata.com/v1/policies?sort=creationTimestamp&order=desc` 规范为 `{{baseurl}}/apis/policies.testudo.softcdata.com/v1/policies`，已补齐 `Authorization` header、`page/limit/sort/order/keyword` query、正确的 `testudo.softcdata.com/disaster-policy-type/state/name` label query，并删除旧的 `testudo.softcdata.com/policy-type/state/name` 错误参数，已追加当前 400/401 响应示例，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_policy/v1/router.go`
- server handler：`internal/apis/disaster_policy/v1/handler.go`，`PolicyHandler.policies`
- 请求链路：`ParseOptions` 解析 `page/limit/sort/order/keyword` 和其他 query filters -> `BuildLabelSelector` 当前返回空 selector -> `DisasterPolicyLister.List(selector)` 读取 informer 缓存 -> 遍历 `qParams.Filters` 对 `item.Labels[k]` 执行 `MatchFuzzy` 包含匹配 -> `transport.Sort` 按 `name` 或 `creationTimestamp` 排序 -> `transport.Paginate` 分页 -> `ConvertToDisasterPolicyDTO` -> `BuildCollectionResponse` -> `WriteSuccess(200)`
- operator 链路：`DisasterPolicyReconciler` 为策略添加 finalizer，校验 `spec.schedule` cron 表达式和 AutoBackup `spec.ttl`，同步 `testudo.softcdata.com/disaster-policy-type/name/state` labels，成功时写 `status.phase=Active` 并清理状态错误；cron 非法写 `reason=InvalidSchedule`，AutoBackup TTL 小于等于 0 写 `reason=InvalidTTL`
- 下层资源链路：列表接口不直接访问业务集群、Velero、S3、DataSync、ResourceSync、AppBackup 或 DisasterInstance；AutoBackup 策略后续由 AppBackup controller 读取并写入 Velero Schedule/backup TTL/paused，SyncPolicy 后续由 DisasterInstance controller 通过 `dataSyncPolicy/resourceSyncPolicy` 对齐 DataSync/ResourceSync 调度
- 已写入内容：五段详细说明、无 path/body 的事实、通用分页排序参数、`keyword` 当前无效的事实、任意 label filter 模糊匹配语义、正确策略 label key、列表接口 `SyncPolicy` 只在 DTO 返回归一化而不参与过滤映射的边界、collection envelope 和 `DisasterPolicyDTO` 字段、operator status 来源、当前错误分类
- 取证备注：RunAPI 原 query 参数缺少 `disaster-policy-` 前缀，与 server/operator 实际 label key 不匹配；新文档删除了错误参数并补齐真实 label key。RunAPI 原说明已经写到 SyncPolicy 返回归一化，本次保留原说明并补充了过滤发生在 DTO 转换前的具体原因
- 主要错误点：JWT 失败由中间件返回 `401 code=2001`；lister/selector 异常返回非 envelope 的 HTTP `400 {"message": "..."}`；本 handler 正常路径不显式返回 500；operator 异步校验失败不改变列表接口 HTTP 状态，只体现在 `data.items[].status.reason/message` 或下游资源状态中

## POST /apis/policies.testudo.softcdata.com/v1/policies

- RunAPI Target ID：`3ee6850638c06a`
- RunAPI 状态：已存在，已更新详细说明，已补齐 `Authorization` header 和当前 201/400/401/409 响应示例，原说明已保留到 `## 原有说明`，已回读验证
- RunAPI 结构备注：历史请求体示例仍是常用 `SyncPolicy` 创建示例；`startTime` 与 `ttl` 的真实字段语义、类型、必填性和约束已写入详细说明
- server 路由：`internal/apis/disaster_policy/v1/router.go`
- server handler：`internal/apis/disaster_policy/v1/handler.go`，`PolicyHandler.createPolicy`
- 请求链路：`BindJSON(CreateDisasterPolicyRequest)` -> `ToCRD` -> `policyTypeFromCreateRequest` 转换 `type` -> `validatePolicyTTL` 校验 AutoBackup TTL -> 构造 `DisasterPolicy{Namespace: disaster-system, Spec: spec}` -> 注入 trace/user annotation -> `DisasterPolicies(disaster-system).Create` -> `ConvertToDisasterPolicyDTO` -> `WriteSuccess(201)`
- 类型转换链路：外部 `AutoBackup` 写入底层 `spec.type=AutoBackup`；外部 `SyncPolicy` 以及兼容值 `DataSync/ResourceSync` 创建时都写入底层 `spec.type=DataSync`；返回 DTO 再将底层 `DataSync` 归一显示为 `SyncPolicy`
- operator 链路：`DisasterPolicyReconciler` 监听新建策略，添加 finalizer，校验 `spec.schedule` 与 AutoBackup `spec.ttl`，同步策略 type/name/state labels；校验成功写 `status.phase=Active`，cron 非法写 `reason=InvalidSchedule`，TTL 非法写 `reason=InvalidTTL`
- 下层资源链路：创建接口不直接访问业务集群、Velero、S3、DataSync、ResourceSync 或 AppBackup；AutoBackup 后续由 AppBackup controller 应用到 Velero Schedule、backup TTL 和 paused 状态；SyncPolicy 后续由 DisasterInstance controller 通过 `dataSyncPolicy/resourceSyncPolicy` 对齐 DataSync/ResourceSync 调度
- 已写入内容：五段详细说明、固定创建命名空间、trace/user annotation、请求体字段 `name/type/schedule/startTime/ttl/description/state`、`ttl` 仅 AutoBackup 支持且必须大于 0、创建阶段不解析 cron 的异步边界、成功 HTTP 201 但 envelope `message=OK` 的当前事实、创建后 status/labels 可能为空的异步边界、当前错误分类
- 取证备注：RunAPI 原成功示例中 `message=Created` 与当前 `transport.WriteSuccess` 实现不一致；新说明和新增成功示例按当前实现写为 `message=OK`，旧示例保留
- 主要错误点：JWT 失败由中间件返回 `401 code=2001`；JSON 绑定或 `type/ttl` 校验失败返回 `400 code=1000`；同名策略已存在返回 `409 code=3009`；其他 Kubernetes Create 错误返回 `500 code=5000`；operator 后续 cron/TTL 校验或下游消费失败只体现在 status、事件或下游资源中

## DELETE /apis/policies.testudo.softcdata.com/v1/policies/:name

- RunAPI Target ID：`35ec1a7378c012`
- RunAPI 状态：已存在，已更新详细说明，URL 已从样例名称规范为 `{{baseurl}}/apis/policies.testudo.softcdata.com/v1/policies/:name`，已补齐 `Authorization` header 和当前 200/401/404 响应示例，已回读验证
- server 路由：`internal/apis/disaster_policy/v1/router.go`
- server handler：`internal/apis/disaster_policy/v1/handler.go`，`PolicyHandler.deletePolicy`
- 请求链路：`path.name` -> `DisasterPolicies(disaster-system).Get(name)` -> NotFound 返回 404 -> best-effort 写 trace/user annotation 并 Update，Update 错误被忽略 -> `DisasterPolicies(disaster-system).Delete(name)` -> NotFound 返回 404 -> `WriteSuccess(200, {"name": name})`
- operator 链路：`DisasterPolicyReconciler.handleDelete` 在 deletionTimestamp 存在时发射删除 Started/Finished 任务事件；当前代码临时禁用了旧的 AppBackup/DisasterJob 引用阻塞逻辑，然后移除 `LabelPolicyFinalizer`
- 下层资源链路：删除接口不直接删除 AppBackup、DataSync、ResourceSync、DisasterInstance、Velero Schedule、Velero Backup 或业务集群资源；仍引用该策略的下游资源由各自 controller 后续 reconcile 处理
- 已写入内容：五段详细说明、path `name`、固定 `disaster-system` 命名空间、非幂等删除语义、best-effort annotation 更新、finalizer 异步清理、当前引用阻塞已禁用的 operator 行为、当前错误分类
- 取证备注：RunAPI 原 URL 使用真实样例 `example-policy001`，本次改为 path 参数 `:name`；原 description 为空，因此没有 `## 原有说明`
- 主要错误点：JWT 失败由中间件返回 `401 code=2001`；删除前 Get 或 Delete 返回 NotFound 均是 `404 code=3004`，不存在不会按成功处理；Get/Delete 非 NotFound 错误返回 `500 code=5000`；删除前 annotation Update 失败被忽略

## GET /apis/policies.testudo.softcdata.com/v1/policies/:name

- RunAPI Target ID：`35ec1a72b8c00c`
- RunAPI 状态：已存在，已更新详细说明，URL 已从样例名称规范为 `{{baseurl}}/apis/policies.testudo.softcdata.com/v1/policies/:name`，已补齐 `Authorization` header 和当前 200/401/404 响应示例，原说明已保留到 `## 原有说明`，已回读验证
- Target 冲突核对：清单原候选 `1be2230900401001` 回读后确认是 `GET {{baseurl}}/apis/policies.testudo.softcdata.com/v1/policies/names?enabled=true&type=SyncPolicy`，不属于详情接口
- server 路由：`internal/apis/disaster_policy/v1/router.go`
- server handler：`internal/apis/disaster_policy/v1/handler.go`，`PolicyHandler.policy`
- 请求链路：`path.name` -> `DisasterPolicies(disaster-system).Get(name)` -> NotFound 返回 404 -> 其他 Get 错误返回 500 -> `ConvertToDisasterPolicyDTO` -> `WriteSuccess(200, DisasterPolicyDTO)`
- operator 链路：详情接口不触发 reconcile；`DisasterPolicyReconciler` 负责维护 finalizer、校验 `spec.schedule` 和 AutoBackup `spec.ttl`、同步策略 labels、成功写 `status.phase=Active`，失败写 `status.reason/status.message`
- 下层资源链路：详情接口不直接访问业务集群、Velero、S3、DataSync、ResourceSync、AppBackup 或 DisasterInstance；AutoBackup 与 SyncPolicy 的下游消费结果只通过 CRD spec/status/labels 间接体现
- 已写入内容：五段详细说明、path `name`、固定 `disaster-system` 命名空间、无 query/body 的事实、单对象 `DisasterPolicyDTO` 字段、SyncPolicy DTO 归一化、status 字段来源、当前错误分类
- 取证备注：RunAPI 旧成功示例仍有历史 `msg=success` 结构；当前 server 使用 `transport.WriteSuccess`，新说明和新增示例按 `message=OK`、`trace_id` 结构写入，旧说明和旧示例保留
- 主要错误点：JWT 失败由中间件返回 `401 code=2001`；策略不存在返回 `404 code=3004`；Kubernetes Get 非 NotFound 错误返回 `500 code=5000`；operator 异步校验或下游消费失败只体现在 status、事件或下游资源中

## PUT /apis/policies.testudo.softcdata.com/v1/policies/:name

- RunAPI Target ID：`3ee6850638c06c`（通用更新记录）和 `1c721610dc401001`（AutoBackup 专项说明记录）
- RunAPI 状态：两条记录均已存在，均已更新详细说明，URL 均已从样例名称规范为 `{{baseurl}}/apis/policies.testudo.softcdata.com/v1/policies/:name`，均已补齐 `Authorization` header，原说明均已保留到 `## 原有说明`，均已回读验证；`1c721610dc401001` 的鉴权类型已从历史 `noauth` 修正为继承项目鉴权
- server 路由：`internal/apis/disaster_policy/v1/router.go`
- server handler：`internal/apis/disaster_policy/v1/handler.go`，`PolicyHandler.updatePolicy`
- 请求链路：`BindJSON(UpdateDisasterPolicyRequest)` -> 校验 `body.name == path.name` -> `RetryOnConflict` -> `DisasterPolicies(disaster-system).Get(name)` -> 当前底层类型非 AutoBackup 时按 `testudo.softcdata.com/disaster-policy-uid=<uid>` 查询 AppBackup 引用并阻止更新 -> `MergeToCRD` 合并请求字段 -> 注入 trace/user annotation -> `DisasterPolicies(disaster-system).Update` -> `ConvertToDisasterPolicyDTO` -> `WriteSuccess(200)`
- 字段合并链路：`type` 为空则保留；`SyncPolicy` 更新时若当前底层为 DataSync/ResourceSync 则保持当前具体类型，否则默认 DataSync；`schedule/description` 仅非空时覆盖；`startTime` 非 nil 时覆盖；`clearTTL=true` 清空 TTL 且优先于 `ttl`；`state` 非 nil 时覆盖；最终调用 `validatePolicyTTL` 确保 TTL 只用于 AutoBackup 且大于 0
- operator 链路：`DisasterPolicyReconciler` 在更新后重新校验 cron 和 AutoBackup TTL，同步 type/name/state labels；成功写 `status.phase=Active` 并清理错误，失败写 `reason=InvalidSchedule` 或 `reason=InvalidTTL`
- 下层资源链路：HTTP handler 不直接访问业务集群、Velero、S3、DataSync 或 ResourceSync；AutoBackup 更新由 AppBackup controller 重新入队引用该策略的 AppBackup，并最终更新 Velero `Schedule.spec.schedule/spec.template.ttl/spec.paused`；同步类策略更新由 DisasterInstance 稳态 reconcile 对齐 DataSync/ResourceSync 调度
- 已写入内容：五段详细说明、同一路由重复 RunAPI 记录说明、path/body `name` 一致性、部分合并更新而非全量替换、空字符串不能清空字符串字段、`clearTTL` 清空语义、SyncPolicy 底层类型保持/默认规则、AutoBackup 引用中允许更新、非 AutoBackup 被 AppBackup 引用时冲突、即时响应和 operator/下游异步边界、当前错误分类
- 取证备注：`1c721610dc401001` 是同一路由的 AutoBackup 专项文档，不是独立 server endpoint；其历史 auth 为 `noauth`，通过 Apipost OpenAPI 全量回读后仅改 `request.auth.type=inherit` 并保留其他结构
- 主要错误点：JWT 失败由中间件返回 `401 code=2001`；JSON 绑定、path/body name 不一致、type/ttl 校验失败返回 `400 code=1000`；策略不存在返回 `404 code=3004`；RetryOnConflict 仍冲突或非 AutoBackup 被 AppBackup 引用返回 `409 code=3009`；Get/List/Update 其他错误返回 `500 code=5000`；operator 和下游收敛失败只体现在 status、事件或下游资源中

## GET /apis/policies.testudo.softcdata.com/v1/policies/names

- RunAPI Target ID：`1be2230900401001`
- RunAPI 状态：已存在，已更新详细说明，URL 已从带默认 query 的 `{{baseurl}}/apis/policies.testudo.softcdata.com/v1/policies/names?enabled=true&type=SyncPolicy` 规范为 `{{baseurl}}/apis/policies.testudo.softcdata.com/v1/policies/names`，已补齐 `Authorization` header，已更新 `enabled/type` query 说明并补充策略名称 label 过滤参数，已追加当前 200/400/401 响应示例，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_policy/v1/router.go`
- server handler：`internal/apis/disaster_policy/v1/handler.go`，`PolicyHandler.policyNames`
- 请求链路：`ParseOptions` -> `BuildLabelSelector` 当前返回空 selector -> `DisasterPolicyLister.List(selector)` -> 读取 `query.enabled` 后从 `qParams.Filters` 删除 -> 读取 `query.type` 后从 `qParams.Filters` 删除 -> 按 `spec.state` 处理 enabled=true/false -> 按 `matchesExternalPolicyTypeFilter` 处理 type -> 排除 `status.phase=Deleting` -> 对剩余 filters 按 label 做 `MatchFuzzy` -> 转换为 `DisasterPolicyNameDTO` -> `WriteSuccess(200, []DisasterPolicyNameDTO)`
- 类型过滤链路：`type=AutoBackup` 只匹配底层 AutoBackup；`type=SyncPolicy` 同时匹配底层 DataSync 和 ResourceSync；兼容 `type=DataSync`、`type=ResourceSync`；未知 type 会匹配不到任何策略
- operator 链路：接口不触发 reconcile；`DisasterPolicyReconciler` 负责同步 labels、维护 `status.phase`，本接口读取 spec/status 后返回名称、UID、外部类型、schedule 和 ttl
- 下层资源链路：接口不直接访问业务集群、Velero、S3、DataSync、ResourceSync、AppBackup 或 DisasterInstance；返回结果通常供 AppBackup、DisasterConfig、DisasterInstance 等后续配置接口引用
- 已写入内容：五段详细说明、下拉列表用途、无分页/排序/meta 的事实、`enabled` 只有 true/false 生效的语义、`type=SyncPolicy` 映射 DataSync/ResourceSync 的特殊行为、删除中策略排除、剩余 query 作为 label 模糊过滤、返回数组字段、当前错误分类
- 取证备注：该 Target ID 曾作为 `GET /policies/:name` 候选出现，回读确认其真实路径为 `GET /policies/names`；直连修正 auth 后再次通过 MCP 重写 description，避免中文字符损坏
- 主要错误点：JWT 失败由中间件返回 `401 code=2001`；lister/selector 异常返回非 envelope 的 HTTP `400 {"message": "..."}`；查询不到匹配策略返回 `200 data=[]`；本 handler 正常路径不显式返回 404/500；策略自身 operator 校验失败不会改变本接口 HTTP 状态

## GET /apis/disastergroups.testudo.softcdata.com/v1/groups

- RunAPI Target ID：`1c048e6aa6401000`
- RunAPI 状态：已存在，已更新详细说明，已补齐 `Authorization` header、`keyword/status/page/limit` query 参数、`meta.summary.instanceCount` 和当前 200/401 响应示例；本次追加 `meta.summary.abnormalCount`、`status=error` 命中 `NotFound`、成员 `reason/message` 说明与当前成功示例，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_group/v1/router.go`
- server handler：`internal/apis/disaster_group/v1/handler.go`，`GroupHandler.listGroups`
- 请求链路：`ParseOptions` -> 读取 `query.keyword/status` -> `DisasterGroups(disaster-system).List` -> `preloadInstances(disaster-system)` 与 `preloadConfigs()` -> 对每个组调用 `buildGroupDTO` 补齐实例摘要和组展示态 -> `keyword` 匹配组名/描述/实例名/实例命名空间 -> `status` 匹配组内实例状态 -> 按创建时间倒序排序 -> `summarizeDisasterGroupList` 统计过滤后分页前所有组的 `instances[]` 条目数量和异常容灾组个数 -> `Paginate` -> `BuildCollectionResponse(resourceType=disasterGroup)` -> 写入 `meta.summary.instanceCount/abnormalCount` -> `WriteSuccess(200)`
- operator 链路：`DisasterGroupReconciler` 监听 DisasterGroup 和组内 DisasterInstance，按 `spec.levels` 聚合 total/ready，缺失实例写 `reason=InstanceNotFound`，实例失败写 `reason=InstanceFailed`，并维护 conditions；server 再根据成员 `FsmState` 推导 `status.fsmState/availableOperations`
- 下层资源链路：列表接口不直接访问业务集群、Velero、S3、DataSync、ResourceSync、AppBackup 或 DisasterOperation；组级实际动作由 `POST /groups/:name/actions` 创建 `DisasterOperation` 后异步执行
- 已写入内容：五段详细说明、keyword/status/page/limit 入参、sort/order 当前不控制排序的事实、`meta.summary.instanceCount` 与 `meta.summary.abnormalCount` 统计语义、当前真实响应为 `data.items[]` 而非历史 `data.data[]`、DisasterGroupDTO 字段、instances 摘要来源、成员 `reason/message`、组聚合状态和可用操作规则、预加载失败降级行为、当前错误分类
- 取证备注：RunAPI 旧成功示例存在历史 collection 结构 `data.type/resourceType/data[]`；当前代码实际使用 `BuildCollectionResponse`，因此新说明和新增成功示例按 `data.items[] + meta` 写入，旧示例保留
- 主要错误点：JWT 失败由中间件返回 `401 code=2001`；列举 DisasterGroup 失败返回 `500 code=5000`；实例/配置预加载失败不会直接报错；查询不到匹配组返回 `200 data.items=[]`；operator 异步聚合错误通过 status/instances 字段体现

## POST /apis/disastergroups.testudo.softcdata.com/v1/groups

- RunAPI Target ID：`1c048e6950c01000`
- RunAPI 状态：已存在，已更新详细说明，已补齐 `Authorization`、`Content-Type` header 和当前 201/400/401/409 响应示例，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_group/v1/router.go`
- server handler：`internal/apis/disaster_group/v1/handler.go`，`GroupHandler.createGroup`
- 请求链路：`BindJSON(CreateDisasterGroupRequest)` -> description 写入 annotation `testudo.softcdata.com/description` -> 构造 `DisasterGroup{Namespace: disaster-system, Spec: req.ToCRD()}` -> `SetTraceAnnotation` -> `DisasterGroups(disaster-system).Create` -> AlreadyExists 返回 409 -> `ConvertToDisasterGroupDTO` -> `WriteSuccess(201)`
- operator 链路：`DisasterGroupReconciler` 新建后 `syncDependencyLabels` 写依赖 token/instance dependency labels；`ensureGroupCreateEvent` 发射创建 Started/Finished task；随后按 `spec.levels` 聚合 `status.totalInstances/readyInstances/reason/message/conditions`
- 下层资源链路：创建组本身不执行组动作，不创建 `DisasterOperation`，不访问业务集群、Velero、S3、DataSync、ResourceSync 或 AppBackup；组动作需后续 `POST /groups/:name/actions`
- 已写入内容：五段详细说明、body 字段 `name/description/levels/policy`、description annotation 来源、levels 层级语义、policy failPolicy/timeoutMin/parallelism/retryPolicy、即时响应 status 可能未聚合、当前错误分类
- 取证备注：请求结构体 tag 使用 `json:"name,required"` 而不是标准 binding tag，创建阶段未主动校验实例存在；缺失实例由 operator/status 或后续组操作暴露
- 主要错误点：JWT 失败返回 `401 code=2001`；JSON 绑定失败返回 `400 code=1000`；同名组返回 `409 code=3009`；Kubernetes Create 其他错误返回 `500 code=5000`；operator 后续聚合/依赖标签错误不改变创建 HTTP 结果

## DELETE /apis/disastergroups.testudo.softcdata.com/v1/groups/:name

- RunAPI Target ID：`1c04bcb222801001`
- RunAPI 状态：已存在，已更新详细说明，URL 已从样例名称规范为 `{{baseurl}}/apis/disastergroups.testudo.softcdata.com/v1/groups/:name`，已补齐 `Authorization` header 和当前 200/401/500 响应示例，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_group/v1/router.go`
- server handler：`internal/apis/disaster_group/v1/handler.go`，`GroupHandler.deleteGroup`
- 请求链路：`path.name` -> `DisasterGroups(disaster-system).Delete(name)` -> NotFound 返回成功 `WriteSuccess(200, {"name": name})` -> 非 NotFound 错误返回 `500 code=5000` -> 删除成功返回 `WriteSuccess(200, {"name": name})`
- operator 链路：`DisasterGroupReconciler` 对已删除对象的 reconcile 结果是 Get NotFound 后直接返回；当前没有组删除 finalizer，也没有专门的组删除 Started/Finished 事件发射逻辑
- 下层资源链路：删除组不删除组内 `DisasterInstance`、已创建的 `DisasterOperation`、`DisasterDrill`、历史事件、业务集群资源、Velero、S3、DataSync、ResourceSync 或 AppBackup；组动作仍只由 `POST /groups/:name/actions` 触发
- 已写入内容：五段详细说明、path `name`、无 query/body 的事实、幂等删除语义、NotFound 不返回 404 的当前行为、删除不等待 watch 事件、operator 无删除 finalizer、当前错误分类
- 取证备注：RunAPI 原 URL 使用真实样例 `core-app-group1`，本次改为 path 参数 `:name`；原说明中的“删除不存在的组会返回成功”与当前代码一致，已保留并补充实现细节
- 主要错误点：JWT 失败返回 `401 code=2001`；Kubernetes Delete 返回非 NotFound 错误时返回 `500 code=5000`；对象不存在返回 `200 code=0`；引用关系、下层操作和 operator 异步状态不影响本次 HTTP 删除响应

## GET /apis/disastergroups.testudo.softcdata.com/v1/groups/:name

- RunAPI Target ID：`1c048e6d34001000`
- RunAPI 状态：已存在，已更新详细说明，URL 已从样例名称规范为 `{{baseurl}}/apis/disastergroups.testudo.softcdata.com/v1/groups/:name`，已补齐 `Authorization` header 和当前 200/401/404/500 响应示例，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证
- Target 冲突核对：清单原候选 `1c20cb43ab001001` 回读后确认是 `GET {{baseurl}}/apis/disastergroups.testudo.softcdata.com/v1/groups/instance-picker?page=1&limit=-1`，不属于组详情接口，将在实例选择器清单项处理
- server 路由：`internal/apis/disaster_group/v1/router.go`
- server handler：`internal/apis/disaster_group/v1/handler.go`，`GroupHandler.getGroup`
- 请求链路：`path.name` -> `DisasterGroups(disaster-system).Get(name)` -> NotFound 返回 404 -> `buildGroupDTO` -> `collectInstanceSummaries` 按 `spec.levels` 逐个读取 `DisasterInstance(disaster-system)` -> 可选读取 `DisasterConfig` 补 `storageRepository` -> `computeGroupFsmState` 推导组展示态与可用操作 -> `WriteSuccess(200)`
- operator 链路：`DisasterGroupReconciler` 按 `spec.levels` 维护 `status.totalInstances/readyInstances/reason/message/conditions`；缺失成员写 `InstanceNotFound`，成员失败、ConfigError 或 status reason 非空写 `InstanceFailed`
- 下层资源链路：详情接口不直接访问业务集群、Velero、S3、DataSync、ResourceSync、AppBackup 或 DisasterOperation；下层执行结果只通过成员实例 status、组 status 或其他详情接口间接体现
- 已写入内容：五段详细说明、path `name`、无 query/body 的事实、`DisasterGroupDTO` 字段、policy 字段、operator 聚合 status 字段、server 推导 `fsmState/availableOperations` 优先级、成员 `instances[]` 字段、成员读取失败返回 `fsmState=NotFound` 的降级行为、配置读取失败只影响 `storageRepository` 的边界、当前错误分类
- 取证备注：详情接口只用 `DisasterConfig` 补 `storageRepository`，不会像实例选择器那样根据配置 NotReady/Error 派生 `ConfigError`；新写入的 200 示例已修正缺失成员只返回 `name/fsmState/level`
- 主要错误点：JWT 失败返回 `401 code=2001`；组不存在返回 `404 code=3004`；读取组本身发生非 NotFound 错误返回 `500 code=5000`；成员实例或配置读取失败不会改变 HTTP 状态，而是体现在 `instances[]` 或空 `storageRepository` 中

## PATCH /apis/disastergroups.testudo.softcdata.com/v1/groups/:name

- RunAPI Target ID：`1c7b252d8bc01001`
- RunAPI 状态：RunAPI 原缺失，已新增到容灾组模块，已写入五段详细说明、JSON body 示例、`Authorization` 与 `Content-Type` header、当前 200/400/401/500 响应示例，鉴权类型为继承项目鉴权，已回读验证
- server 路由：`internal/apis/disaster_group/v1/router.go`
- server handler：`internal/apis/disaster_group/v1/handler.go`，`GroupHandler.updateGroup`
- 请求链路：`path.name` -> `BindJSON(UpdateDisasterGroupRequest)` -> `RetryOnConflict` -> `DisasterGroups(disaster-system).Get(name)` -> 按字段存在性更新 description annotation、整体替换 `spec.levels`、整体替换 `spec.policy` -> 写 trace annotation -> `DisasterGroups(disaster-system).Update` -> `ConvertToDisasterGroupDTO` -> `WriteSuccess(200)`
- operator 链路：更新后 `DisasterGroupReconciler` 重新维护 dependency token 和 `spec.levels` 依赖标签，重新聚合 `status.totalInstances/readyInstances/reason/message/conditions`，成员缺失或失败由后续 status 暴露
- 下层资源链路：PATCH 只更新组配置，不创建 `DisasterOperation`，不执行组动作，不直接访问业务集群、Velero、S3、DataSync、ResourceSync 或 AppBackup；新的 policy 会被后续 `POST /groups/:name/actions` 读取
- 已写入内容：五段详细说明、缺失接口新增记录、PATCH 与 PUT 共用 handler 的当前事实、部分更新语义、`description/levels/policy` 字段存在性语义、policy 整体替换而非子字段 merge、成功响应不补齐 `summary/fsmState/availableOperations/instances` 的边界、当前错误分类
- 取证备注：当前 handler 对目标组不存在、Get 非 NotFound 错误、Update 冲突重试失败都统一返回 `500 code=5000`，不会返回 404/409；server 不校验 levels 成员是否存在，也不校验 failPolicy 枚举值
- 主要错误点：JSON 绑定失败返回 `400 code=1000`；JWT 失败返回 `401 code=2001`；读取或更新组失败返回 `500 code=5000`；operator 异步状态重算失败不改变本次 HTTP 成功响应

## PUT /apis/disastergroups.testudo.softcdata.com/v1/groups/:name

- RunAPI Target ID：`1c239548ddc01000`
- RunAPI 状态：已存在，已更新详细说明，URL 已从样例名称规范为 `{{baseurl}}/apis/disastergroups.testudo.softcdata.com/v1/groups/:name`，已补齐 `Authorization`、`Content-Type` header 和当前 200/400/401/500 响应示例，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，已回读验证
- RunAPI 结构备注：原 description 为空，因此没有 `## 原有说明`；旧成功示例保留，新成功示例按当前 `transport.WriteSuccess` 的 `code/message/trace_id` envelope 补齐
- server 路由：`internal/apis/disaster_group/v1/router.go`
- server handler：`internal/apis/disaster_group/v1/handler.go`，`GroupHandler.updateGroup`
- 请求链路：`path.name` -> `BindJSON(UpdateDisasterGroupRequest)` -> `RetryOnConflict` -> `DisasterGroups(disaster-system).Get(name)` -> 按字段存在性更新 description annotation、整体替换 `spec.levels`、整体替换 `spec.policy` -> 写 trace annotation -> `DisasterGroups(disaster-system).Update` -> `ConvertToDisasterGroupDTO` -> `WriteSuccess(200)`
- operator 链路：更新后 `DisasterGroupReconciler` 重新维护 dependency token 和 `spec.levels` 依赖标签，重新聚合 `status.totalInstances/readyInstances/reason/message/conditions`，成员缺失或失败由后续 status 暴露
- 下层资源链路：PUT 只更新组配置，不创建 `DisasterOperation`，不执行组动作，不直接访问业务集群、Velero、S3、DataSync、ResourceSync 或 AppBackup；新的 policy 会被后续 `POST /groups/:name/actions` 读取
- 已写入内容：五段详细说明、PUT 与 PATCH 共用 handler 的当前事实、当前 PUT 是部分更新而非严格全量替换、`description/levels/policy` 字段存在性语义、policy 整体替换而非子字段 merge、成功响应不补齐 `summary/fsmState/availableOperations/instances` 的边界、当前错误分类
- 取证备注：当前 handler 对目标组不存在、Get 非 NotFound 错误、Update 冲突重试失败都统一返回 `500 code=5000`，不会返回 404/409；server 不校验 levels 成员是否存在，也不校验 failPolicy 枚举值
- 主要错误点：JSON 绑定失败返回 `400 code=1000`；JWT 失败返回 `401 code=2001`；读取或更新组失败返回 `500 code=5000`；operator 异步状态重算失败不改变本次 HTTP 成功响应

## POST /apis/disastergroups.testudo.softcdata.com/v1/groups/:name/actions

- RunAPI Target ID：`1c048e6d67c01000`
- RunAPI 状态：已存在，已更新详细说明，URL 已从样例名称规范为 `{{baseurl}}/apis/disastergroups.testudo.softcdata.com/v1/groups/:name/actions`，已补齐合法 JSON body 示例、`Authorization`、`Content-Type` header 和当前 202/400/401/404/500 响应示例，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_group/v1/router.go`
- server handler：`internal/apis/disaster_group/v1/handler.go`，`GroupHandler.executeAction`
- 请求链路：`path.name` -> `BindJSON(instanceapi.ExecuteActionRequest)` -> `DisasterGroups(disaster-system).Get(name)` 校验组存在 -> 读取 `config.force/skipFinalSync/skipScaleDownSource/timeoutMinutes/skipPodReadyCheck/waitUntilReady` -> `sync-data` 映射 `syncdata`，`sync-resource` 映射 `syncresource` -> 生成父级操作名 `<request operation>-<group>-<unix seconds>` -> 再次读取组获取 `spec.policy.retryPolicy` -> 创建父级 `DisasterOperation` -> `WriteSuccess(202, operationName/message)`
- operator 链路：`DisasterOperationReconciler` 对 `spec.groupName` 非空的父操作进入 `handleGroupOperation`；按 `DisasterGroup.spec.levels` 从 `Level-0` 分层创建子 `DisasterOperation`；子操作继承 operationType、force、skipFinalSync、skipScaleDownSource、skipPodReadyCheck、waitUntilReady、retryPolicy、trace annotation 和 ownerReference；同层子操作并行等待，完成后进入下一层
- 下层资源链路：父级操作只编排子操作；真正访问业务集群、Velero、S3、DataSync、ResourceSync、AppBackup、Deployment/StatefulSet 等资源的是每个成员实例的子操作，具体取决于 `operationType`
- 已写入内容：五段详细说明、异步 202 受理语义、operation 可传值与 `sync-data/sync-resource` 映射、config 大小写兼容字段、skipPodReadyCheck 与 waitUntilReady 优先级、skipScaleDownSource annotation 兼容链路、父级/子级 `DisasterOperation` 字段与 labels、FailPolicy Continue/Stop 行为、当前错误分类
- 取证备注：`config.timeoutMinutes` 当前只写入父级组操作 `spec.timeoutMinutes`；operator 创建子操作时未透传该字段，父级分层编排也没有独立按该字段做 Level 超时控制。RunAPI 原请求体含 `//` 注释，本次改为合法 JSON 示例
- 主要错误点：JSON 绑定或 operation 缺失返回 `400 code=1000`；JWT 失败返回 `401 code=2001`；组读取失败统一返回 `404 code=3004` 且当前不区分非 NotFound 错误；父级操作创建失败返回 `500 code=5000`；成员实例或下层执行失败不改变本次 HTTP `202`，只体现在操作详情、历史或 WebSocket 中

## GET /apis/disastergroups.testudo.softcdata.com/v1/groups/:name/history

- RunAPI Target ID：`1c04e21a5c801001`
- RunAPI 状态：已存在，已更新详细说明，已补齐 `Authorization` header 和当前 200/401/500 响应示例，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_group/v1/router.go`
- server handler：`internal/apis/disaster_group/v1/handler.go`，`GroupHandler.getHistory`
- 请求链路：`path.name` -> `DisasterOperations(disaster-system).List(LabelSelector: testudo.softcdata.com/group=<name>)` -> 对每个父级组操作调用 `instanceapi.ConvertToHistoryDTO` -> 按 `HistoryDTO.Time` 倒序排序 -> `WriteSuccess(200, []HistoryDTO)`
- operator 链路：`DisasterOperationReconciler` 维护父级组操作 `status.state/reason/message/autoCancel*`，本接口只读取已落库状态，不触发 reconcile；组操作子 `DisasterOperation` 通常没有 `testudo.softcdata.com/group` label，因此不在本列表中
- 下层资源链路：历史接口不直接访问业务集群、Velero、S3、DataSync、ResourceSync 或 AppBackup；下层执行结果只通过父级 `DisasterOperation.status` 反映到历史 DTO
- 已写入内容：五段详细说明、path `name`、无 query/body 的事实、固定 label selector、无分页的当前行为、`HistoryDTO` 字段、autoCancel 字段、operator 固定 `operator=admin` 的转换事实、组不存在不返回 404、只返回父级组操作的边界、当前错误分类
- 取证备注：RunAPI 旧示例使用历史 `code/msg` envelope，新示例按当前 `message=OK/trace_id` envelope 补齐；旧增强说明已保留
- 主要错误点：JWT 失败返回 `401 code=2001`；列举 `DisasterOperation` 失败返回 `500 code=5000`；组不存在、组已删除或无历史均返回 `200 data=[]`；子操作和 label 缺失的操作不会出现在结果中

## GET /apis/disastergroups.testudo.softcdata.com/v1/groups/:name/instances

- RunAPI Target ID：`1c2100948fc01001`
- RunAPI 状态：已存在，已更新详细说明，URL 已从样例名称规范为 `{{baseurl}}/apis/disastergroups.testudo.softcdata.com/v1/groups/:name/instances`，已补齐 `Authorization` header、`keyword/status/page/limit` query 和当前 200/401/404/500 响应示例，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_group/v1/router.go`
- server handler：`internal/apis/disaster_group/v1/handler.go`，`GroupHandler.listGroupInstances`
- 请求链路：`path.name` -> `ParseOptions` -> `DisasterGroups("").List(FieldSelector: metadata.name=<name>)` -> 取第一条 group -> 展平并去重 `spec.levels` -> 对每个成员在 `group.Namespace` 读取 `DisasterInstance` -> 可选读取 `DisasterConfig` -> `deriveGroupMemberStatus` 派生 `status.state/reason/message` -> 按 `status` 和 `keyword` 过滤 -> `Paginate` -> `BuildCollectionResponse(resourceType=groupMemberInstance)` -> `WriteSuccess(200)`
- operator 链路：`DisasterGroupReconciler` 维护组 status，`DisasterInstance` 和 `DisasterConfig` controller 维护成员和配置状态；本接口只读取并派生展示 DTO，不触发 reconcile
- 下层资源链路：成员列表接口不直接访问业务集群、Velero、S3、DataSync、ResourceSync 或 AppBackup；下层异常通过成员实例 status 或配置 status 间接体现
- 已写入内容：五段详细说明、全命名空间查组且取第一条的当前行为、成员只在组命名空间读取且无全局 fallback、`keyword/status/page/limit/sort/order` 入参、`GroupMemberInstanceDTO` 字段、`ConfigError/NotFound/Failed` 派生规则、当前真实响应为 `data.items[] + meta`、sort/order 仅回显不排序、当前错误分类
- 取证备注：RunAPI 旧示例混用了 `data.data[]`、`data.items[]` 和 `namespace` 字段；新说明与新增示例按当前 `BuildCollectionResponse` 和 `GroupMemberInstanceDTO` 写入。新 200 示例已修正为未带 `status` 过滤时返回多状态成员
- 主要错误点：JWT 失败返回 `401 code=2001`；未找到组返回 `404 code=3004`；列举组失败返回 `500 code=5000`；成员读取失败不改变 HTTP 状态，而是返回成员 `status.state=NotFound`；配置 NotFound 返回成员 `ConfigError`，配置读取非 NotFound 错误当前不会返回 HTTP 错误

## GET /apis/disastergroups.testudo.softcdata.com/v1/groups/:name/operations/:operationName

- RunAPI Target ID：`1c5ead7bcb801001`
- RunAPI 状态：已存在，已更新详细说明，已补齐 `Authorization` header 和当前 200/401/404/500 响应示例，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_group/v1/router.go`
- server handler：`internal/apis/disaster_group/v1/handler_operation.go`，`GroupHandler.getOperationDetail`
- 请求链路：`path.name/path.operationName` -> `DisasterOperations(disaster-system).Get(operationName)` -> NotFound 返回 404 -> 校验 `op.Spec.GroupName == groupName`，不匹配返回 404 -> `instanceapi.ConvertToOperationDetailDTO` -> `WriteSuccess(200)`
- operator 链路：父级组操作由 `DisasterOperationReconciler.handleGroupOperation` 维护 `status.state/currentStep/message/startTime/completionTime` 并分层创建子操作；实例级子操作维护实际步骤。本接口只读取父操作自身已落库 status，不触发 reconcile
- 下层资源链路：详情接口不直接访问业务集群、Velero、S3、DataSync、ResourceSync 或 AppBackup；下层执行结果通过父操作 status、自动补偿字段或子操作状态间接体现
- 已写入内容：五段详细说明、只读操作且不读取 DisasterGroup 的事实、operation 归属组校验、`OperationDetailDTO` 字段、steps/autoCancel/roleStatus/groupStatus 字段、`groupStatus` 当前可选且 operator 不稳定写入的边界、当前错误分类
- 取证备注：RunAPI 原说明把 `groupStatus` 描述成组级执行状态，本次补充当前代码边界：CRD/DTO 支持该字段，但当前 `handleGroupOperation` 主要通过 `currentStep/message` 表示 Level 进度，没有稳定写入 `status.groupStatus`
- 主要错误点：JWT 失败返回 `401 code=2001`；操作不存在或归属组不匹配返回 `404 code=3004`；读取操作发生非 NotFound 错误返回 `500 code=5000`；本接口不查询子操作列表，子操作明细需通过其他操作查询能力获取

## GET /apis/disastergroups.testudo.softcdata.com/v1/groups/instance-picker

- RunAPI Target ID：`1c20cb43ab001001`
- RunAPI 状态：已存在，已更新详细说明，URL 已从带默认 query 的 `{{baseurl}}/apis/disastergroups.testudo.softcdata.com/v1/groups/instance-picker?page=1&limit=-1` 规范为 `{{baseurl}}/apis/disastergroups.testudo.softcdata.com/v1/groups/instance-picker`，已补齐 `Authorization` header、`keyword/status/page/limit/sort/order` query 和当前 200/401/500 响应示例，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_group/v1/router.go`
- server handler：`internal/apis/disaster_group/v1/handler.go`，`GroupHandler.instancePicker`
- 请求链路：`ParseOptions` -> `DisasterInstances("").List` 全命名空间列举实例 -> 可选读取 `DisasterConfig` -> `deriveGroupMemberStatus` 派生 `status.state/reason/message` -> 按 `status` 匹配派生状态或原始 `fsmState` -> 按 `keyword` 匹配 name/namespaces/description -> `Paginate` -> `BuildCollectionResponse(resourceType=instancePickerItem)` -> `WriteSuccess(200)`
- operator 链路：接口不触发 reconcile；实例状态来源于 `DisasterInstanceReconciler` 和 `DisasterOperationReconciler` 写入的 `status.fsmState/reason/message`，配置状态来源于 `DisasterConfigReconciler` 写入的 `status.status/reason/message`，server 只做展示态派生
- 下层资源链路：不直接访问业务集群、Velero、S3、DataSync、ResourceSync 或 AppBackup/AppRestore；下层异常只通过实例状态或配置状态间接反映
- 已写入内容：五段详细说明、无 path/body 的事实、query 参数、当前真实响应 `data.items[] + meta`、`InstancePickerItemDTO` 字段、`ConfigError/Failed/Unknown` 派生规则、`keyword` 不匹配 labels 的事实、`sort/order` 仅回显不排序、当前错误分类
- 取证备注：RunAPI 原说明和旧示例里有历史 `data.data[]`、`namespace/labels` 字段，当前代码实际返回 `data.items[]`、`namespaces`、`description`、`status`、`fsmState`；旧说明已完整保留，新示例按当前 `BuildCollectionResponse` 补齐
- 主要错误点：JWT 失败返回 `401 code=2001`；列举 `DisasterInstance` 失败返回 `500 code=5000`；配置读取失败不返回 HTTP 错误；查询不到匹配项返回 `200 data.items=[]`

## GET /apis/disastergroups.testudo.softcdata.com/v1/watch/groups/operations

- RunAPI Target ID：`1a44c9e9f8c0a8`
- RunAPI 状态：已存在，已更新详细说明，已补齐 `Authorization`、`Sec-WebSocket-Protocol` header 和 `token/groupName` query，目标类型 `websocket2` 已保留，已回读验证；该类型无 response 容器，连接帧、心跳帧、事件帧、关闭帧和错误帧示例已写入详细说明
- server 路由：`internal/apis/disaster_group/v1/router.go`
- server handler：`internal/apis/disaster_group/v1/handler.go`，`GroupHandler.watchGroupOperations`
- 请求链路：`WebSocketTokenAdapter` -> JWT 中间件 -> `query.groupName` -> 可选 `LabelSelector: testudo.softcdata.com/group=<groupName>` -> 先 `DisasterOperations(disaster-system).List` 获取最新 `resourceVersion` -> 从该版本 `Watch` -> `StreamWatch` 推送 connected/heartbeat/watch event/closed/timeout/error 帧
- 返回转换：watch event object 为 `DisasterOperation` 时调用 `ConvertToDisasterOperationDTO`，推送到 `data.object`；事件类型写入 `data.type`
- operator 链路：`DisasterOperationReconciler` 处理 `spec.groupName` 非空父级组操作时进入 `handleGroupOperation`，按 `DisasterGroup.spec.levels` 分层创建实例级子操作，并更新父级 `status.currentStep/message/state` 触发 `MODIFIED` 推送；`cancel/undo` 等新操作可能把同组运行中 failover 标记为 `Failed/Superseded`
- 下层资源链路：接口不直接访问业务集群、Velero、S3、DataSync、ResourceSync 或 AppBackup/AppRestore；下层执行由 operator 的具体操作流程触发，结果间接体现在 `DisasterOperation.status`
- 已写入内容：五段详细说明、WebSocket token 三种传入方式、`groupName` label selector 语义、不传 `groupName` 时监听全部操作的边界、List+Watch 跳过历史 `ADDED` 全量的行为、`DisasterOperationDTO` 字段、`groupStatus` 当前不保证稳定写入的边界、心跳/关闭/超时/错误帧
- 取证备注：RunAPI 原 description 为空；`websocket2` 的参数组件不能通过 MCP 组件接口稳定落库，已用 Apipost OpenAPI 直接更新 request header/query 并回读确认
- 主要错误点：握手前 JWT 失败返回普通 HTTP JSON，当前中间件 body 通常为 `{"code":401,"msg":"..."}`；WebSocket 升级失败返回 HTTP `500 {"message":"WebSocket 升级失败: ..."}`；初始 List 或 Watch `DisasterOperation` 失败通过 WebSocket 返回 `code=5000` 错误帧；`groupName` 不存在或无匹配操作不会报错，只是无事件推送

## GET /apis/disastergroups.testudo.softcdata.com/v1/watch/groups/operations/:operationName

- RunAPI Target ID：`1c5eadf517801001`、`1a44ffeb38c0ab`
- RunAPI 状态：两个历史目标均已更新；`1c5eadf517801001` 为 `api` 类型但按 WebSocket 语义写入详细说明，鉴权类型已从 `noauth` 修正为继承项目鉴权，并保留原说明到 `## 原有说明`；`1a44ffeb38c0ab` 为 `websocket2` 类型，URL 已从具体样例 `.../operations/app-group-1772076095894` 规范为 `.../operations/:operationName`；两个目标均已补齐 `Authorization`、`Sec-WebSocket-Protocol`、`token` 参数说明并回读验证
- server 路由：`internal/apis/disaster_group/v1/router.go`
- server handler：`internal/apis/disaster_group/v1/handler.go`，`GroupHandler.watchGroupOperation`
- 请求链路：`WebSocketTokenAdapter` -> JWT 中间件 -> `path.operationName` -> 先 `DisasterGroups(disaster-system).Get(operationName)` 动态探测 -> 若组存在则按 `LabelSelector: testudo.softcdata.com/group=<operationName>` 先 List 获取 `resourceVersion` 再 Watch -> 若组不存在则按 `FieldSelector: metadata.name=<operationName>` 直接 Watch 单个 `DisasterOperation` -> `StreamWatch` 推送 connected/heartbeat/watch event/closed/timeout/error 帧
- 返回转换：watch event object 为 `DisasterOperation` 时调用 `ConvertToDisasterOperationDTO`，推送到 `data.object`；事件类型写入 `data.type`
- operator 链路：`DisasterOperationReconciler` 维护 `status.state/reason/currentStep/message/steps/autoCancel*/groupStatus`；组操作由 `handleGroupOperation` 按 `DisasterGroup.spec.levels` 创建实例级子操作，持续更新父级 `status.message` 触发进度 `MODIFIED`；同组新 `cancel/undo` 等操作可能把运行中 failover 标记为 `Failed/Superseded`
- 下层资源链路：接口不直接访问业务集群、Velero、S3、DataSync、ResourceSync 或 AppBackup/AppRestore；下层执行由 operator 的具体操作流程触发，结果间接体现在 `DisasterOperation.status`
- 已写入内容：五段详细说明、`operationName` 可为组名或操作名的动态探测语义、组名模式与单操作模式的 Watch 差异、名称冲突时组名优先、WebSocket token 三种传入方式、`DisasterOperationDTO` 字段、`groupStatus` 当前不保证稳定写入的边界、心跳/关闭/超时/错误帧
- 取证备注：当前 handler 对 `DisasterGroup.Get` 的非 NotFound 错误没有区分处理，会继续进入单操作模式；组名模式使用 List+Watch 跳过历史全量，单操作模式不先 List，注释说明目的是立即下发当前状态
- 主要错误点：握手前 JWT 失败返回普通 HTTP JSON，当前中间件 body 通常为 `{"code":401,"msg":"..."}`；WebSocket 升级失败返回 HTTP `500 {"message":"WebSocket 升级失败: ..."}`；组名模式初始 List 或后续 Watch 失败通过 WebSocket 返回 `code=5000` 错误帧；`operationName` 对应组或操作不存在不会返回 404，只是进入单操作 watch 或无事件推送

## GET /apis/disastergroups.testudo.softcdata.com/v1/watch/groups/status

- RunAPI Target ID：`39a5775f8c2ec`
- RunAPI 状态：已存在，已更新详细说明，已补齐 `Authorization`、`Sec-WebSocket-Protocol` header 和 `token` query，目标类型 `websocket2` 已保留，已回读验证；该类型无 response 容器，连接帧、心跳帧、事件帧、关闭帧和错误帧示例已写入详细说明
- server 路由：`internal/apis/disaster_group/v1/router.go`
- server handler：`internal/apis/disaster_group/v1/handler.go`，`GroupHandler.watchGroupStatuses`
- 请求链路：`WebSocketTokenAdapter` -> JWT 中间件 -> 先 `DisasterGroups(disaster-system).List` 获取最新 `resourceVersion` -> 从该版本 `Watch` 全部 `DisasterGroup` -> `StreamWatch` 推送 connected/heartbeat/watch event/closed/timeout/error 帧
- 返回转换：watch event object 为 `DisasterGroup` 时调用 `buildGroupDTO`；该转换会读取成员 `DisasterInstance` 和可选 `DisasterConfig`，补齐 `instances[]`、`status.summary`、`status.fsmState`、`status.availableOperations` 后推送到 `data.object`
- operator 链路：`DisasterGroupReconciler` 监听 `DisasterGroup` 与 `DisasterInstance`，按 `spec.levels` 聚合 `status.totalInstances/readyInstances/reason/message/conditions`，并同步依赖标签；成员实例变化会触发对应组重新计算 status
- 下层资源链路：接口不直接访问业务集群、Velero、S3、DataSync、ResourceSync 或 AppBackup/AppRestore；下层异常通过成员实例状态、配置状态或组 status 间接体现
- 已写入内容：五段详细说明、WebSocket token 三种传入方式、无 path/body/query 过滤的事实、List+Watch 跳过历史 `ADDED` 全量的行为、完整 `DisasterGroupDTO` 字段、`status.fsmState` 与 `availableOperations` 计算规则、成员读取失败和配置读取失败边界、心跳/关闭/超时/错误帧
- 取证备注：`status.summary/status.fsmState/status.availableOperations` 不是 operator 原始 status 字段，而是 server 在 `buildGroupDTO/computeGroupFsmState` 中现场计算；watch 转换会额外读成员实例，读失败不会让 WebSocket 失败
- 主要错误点：握手前 JWT 失败返回普通 HTTP JSON，当前中间件 body 通常为 `{"code":401,"msg":"..."}`；WebSocket 升级失败返回 HTTP `500 {"message":"WebSocket 升级失败: ..."}`；初始 List 或 Watch `DisasterGroup` 失败通过 WebSocket 返回 `code=5000` 错误帧；当前没有容灾组不会报错，只是没有业务事件推送

## GET /apis/disastergroups.testudo.softcdata.com/v1/watch/groups/status/:name

- RunAPI Target ID：`39a699978c2f0`
- RunAPI 状态：已存在，已更新详细说明，URL 已从具体样例 `{{baseurl}}/apis/disastergroups.testudo.softcdata.com/v1/watch/groups/status/app-group-1774505807766` 规范为 `{{baseurl}}/apis/disastergroups.testudo.softcdata.com/v1/watch/groups/status/:name`，已补齐 `Authorization`、`Sec-WebSocket-Protocol` header 和 `token` query，目标类型 `websocket2` 已保留，已回读验证；该类型无 response 容器，连接帧、心跳帧、事件帧、关闭帧和错误帧示例已写入详细说明
- server 路由：`internal/apis/disaster_group/v1/router.go`
- server handler：`internal/apis/disaster_group/v1/handler.go`，`GroupHandler.watchGroupStatus`
- 请求链路：`WebSocketTokenAdapter` -> JWT 中间件 -> `path.name` -> `DisasterGroups(disaster-system).Watch(FieldSelector: metadata.name=<name>)` -> `StreamWatch` 推送 connected/heartbeat/watch event/closed/timeout/error 帧
- 返回转换：watch event object 为 `DisasterGroup` 时调用 `buildGroupDTO`；该转换会读取成员 `DisasterInstance` 和可选 `DisasterConfig`，补齐 `instances[]`、`status.summary`、`status.fsmState`、`status.availableOperations` 后推送到 `data.object`
- operator 链路：`DisasterGroupReconciler` 监听 `DisasterGroup` 与 `DisasterInstance`，按 `spec.levels` 聚合 `status.totalInstances/readyInstances/reason/message/conditions`，并同步依赖标签；目标组成员实例变化会触发该组重新计算 status
- 下层资源链路：接口不直接访问业务集群、Velero、S3、DataSync、ResourceSync 或 AppBackup/AppRestore；下层异常通过成员实例状态、配置状态或组 status 间接体现
- 已写入内容：五段详细说明、path `name`、WebSocket token 三种传入方式、无 body 的事实、直接 FieldSelector Watch 且不先 List 校验目标存在的行为、完整 `DisasterGroupDTO` 字段、`status.fsmState` 与 `availableOperations` 计算规则、成员读取失败和配置读取失败边界、心跳/关闭/超时/错误帧
- 取证备注：该接口与全部组状态流不同，不先 List 获取 `resourceVersion`；path `name` 不存在不会返回 404，连接仍可建立并等待同名对象事件
- 主要错误点：握手前 JWT 失败返回普通 HTTP JSON，当前中间件 body 通常为 `{"code":401,"msg":"..."}`；WebSocket 升级失败返回 HTTP `500 {"message":"WebSocket 升级失败: ..."}`；Watch `DisasterGroup` 失败通过 WebSocket 返回 `code=5000` 错误帧；目标组不存在不会报错，只是没有业务事件推送

## GET /apis/disasterconfigs.testudo.softcdata.com/v1/configs

- RunAPI Target ID：`25559f2138c090`
- RunAPI 状态：已存在，已更新详细说明，已补齐 `Authorization` header、`page/limit/sort/order/keyword/environment` query、`meta.summary.healthyCount/abnormalCount` 和当前 200/400/401 响应示例，鉴权类型为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证；本次追加 200 响应示例“成功（creation_timestamp 本地时区）”
- server 路由：`internal/apis/disaster_config/v1/router.go`
- server handler：`internal/apis/disaster_config/v1/handler.go`，`ConfigHandler.configs`
- 请求链路：`ParseOptions` -> `BuildLabelSelector` 空 selector -> `DisasterConfigLister.List` -> `keyword` 对 `metadata.name` 做不区分大小写的包含匹配 -> 动态 query 按 `metadata.labels` value 包含匹配 -> `Sort(name/creationTimestamp)` -> `summarizeDisasterConfigList` 统计过滤后分页前 `Ready` 与 `NotReady/Error` 数量 -> `Paginate` -> `toDisasterConfigDTO` -> `ConvertToDisasterConfigDTO` 将 `metadata.creationTimestamp` 按 server 进程本地时区格式化为 RFC3339 offset 字符串 -> `populatePolicyCrons` 可选读取 `DisasterPolicy` -> `BuildCollectionResponse(resourceType=disasterConfig)` -> 写入 `meta.summary.healthyCount/abnormalCount` -> `WriteSuccess(200)`
- operator 链路：`DisasterConfigReconciler` 添加 finalizer、同步依赖 labels，校验 source/target `Cluster`、`StorageRepository`，向源/目标集群应用 Velero credential `Secret` 和 `BackupStorageLocation`，写入 `status.status/reason/message`
- 下层资源链路：列表接口本身不直接访问业务集群、Velero 或 S3；operator reconcile 会通过 `DefaultBSL.ApplyStorageRepository` 更新远端 Velero `Secret` 和 `BackupStorageLocation`
- 已写入内容：五段详细说明、无 path/body 的事实、分页排序、动态 label 过滤、`keyword` 按配置名称搜索、`meta.summary.healthyCount/abnormalCount` 统计语义、`DisasterConfigDTO` 字段、`creation_timestamp` 本地时区 RFC3339 offset 格式、policy cron 补充规则、imageRewrite 字段、operator 状态原因、当前错误分类
- 取证备注：RunAPI 旧说明/示例包含旧字段 `dataSyncInterval/resourcesSyncInterval` 和旧 envelope `msg/success`，当前代码实际返回 `data.items[] + meta` 且 DTO 不返回 interval 字段；Kubernetes 原始 `metadata.creationTimestamp` 仍为 UTC，server DTO 从本次调整起按 `time.Local/TZ` 输出，例如 `2026-05-15T11:08:19+08:00`；新示例已按当前 `BuildCollectionResponse` 补齐
- 主要错误点：JWT 失败由中间件返回普通 HTTP JSON；Lister 列表失败返回 `400 code=1000`；无匹配返回 `200 data.items=[]`；关联策略读取失败只导致 cron 为空；operator 校验失败不影响列表 HTTP 状态，只体现在 `status.status/reason/message` 与 `meta.summary.abnormalCount`

## POST /apis/disasterconfigs.testudo.softcdata.com/v1/configs

- RunAPI Target ID：`3ee68505b8c065`
- RunAPI 状态：已存在，已更新详细说明，已补齐 `Authorization` header、当前 `imageRewrite` 请求体示例和 201/400/401/409/500 响应示例，鉴权类型为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证；本次追加 201 响应示例“创建成功（creation_timestamp 本地时区）”
- server 路由：`internal/apis/disaster_config/v1/router.go`
- server handler：`internal/apis/disaster_config/v1/handler.go`，`ConfigHandler.createConfig`
- 请求链路：`rejectUnsupportedSyncPolicyField` 拒绝顶层 `syncPolicy` -> `BindJSON(CreateDisasterConfigRequest)` -> `ToCRD` 写入 `spec.sourceCluster/targetCluster/storageRepository/dataSyncType/dataSyncPolicy/resourceSyncPolicy` -> `EffectiveImageRewrite` 在 `imageRewrite` 和旧 `imageSources` 间取有效配置 -> `validateAndBuildImageRewrite` 校验源/目标 `Cluster.spec.imageSources` 别名对 -> 写入 description 和 trace annotation -> `DisasterConfigs().Create` -> `toDisasterConfigDTO` -> `ConvertToDisasterConfigDTO` 将 `metadata.creationTimestamp` 按 server 进程本地时区格式化为 RFC3339 offset 字符串 -> `populatePolicyCrons` -> `WriteSuccess(201)`
- operator 链路：`DisasterConfigReconciler` 收到新 CR 后添加 finalizer、同步依赖 labels，首次 status 为空时置 `Pending`；随后校验源/目标 `Cluster` 是否存在且 `Ready`，校验 `StorageRepository`，把存储仓库应用到源/目标集群，最终写入 `Ready/NotReady/Error` 和 `reason/message`
- 下层资源链路：HTTP 创建接口本身不访问业务集群、Velero 或 S3；operator 后续通过 `GetKubeClientSetWithCluster` 连接源/目标集群，并由 `DefaultBSL.ApplyStorageRepository` 维护 Velero credential `Secret` 和 `BackupStorageLocation`
- 已写入内容：五段详细说明、无 path/query 的事实、`name/description/sourceCluster/targetCluster/storageRepository/dataSyncType/dataSyncPolicy/resourcesSyncPolicy/resourceSyncPolicy/imageRewrite/imageSources/syncPolicy` 入参、`resourcesSyncPolicy` 优先级、`imageSources` 旧字段转换规则、镜像源映射校验规则、201 DTO 字段、`creation_timestamp` 本地时区 RFC3339 offset 格式、operator 状态含义、当前错误分类
- 取证备注：RunAPI 旧说明中的 `dataSyncType` 历史值 `s3-replices/nfs-share` 与当前 CRD 注释不一致；新说明按当前 operator CRD 写为 `fsb/snapshot/none/external`，并注明当前 server 不做枚举校验。旧说明和旧示例已完整保留到 `## 原有说明`
- 主要错误点：顶层 `syncPolicy`、JSON 绑定失败、`imageRewrite` 非法或镜像源别名无法在源/目标集群中匹配均返回 `400 code=1000`；JWT 失败返回 `401`；同名对象返回 `409 code=3009`；创建 CR 非冲突错误返回 `500 code=5000`；策略不存在、集群未就绪、存储仓库不存在或 Velero 应用失败不改变本次创建成功状态，只体现在后续 status

## DELETE /apis/disasterconfigs.testudo.softcdata.com/v1/configs/:name

- RunAPI Target ID：`25559f21b8c098`
- RunAPI 状态：已存在，已更新详细说明，URL 已从具体样例名称规范为 `{{baseurl}}/apis/disasterconfigs.testudo.softcdata.com/v1/configs/:name`，已补齐 path `name` 参数和当前 200/401/404/500 响应示例，鉴权类型为继承项目鉴权，已回读验证；原 description 为空，因此未追加 `## 原有说明`
- server 路由：`internal/apis/disaster_config/v1/router.go`
- server handler：`internal/apis/disaster_config/v1/handler.go`，`ConfigHandler.deleteConfig`
- 请求链路：`path.name` -> best-effort `DisasterConfigs().Get` -> best-effort 写入 trace annotation 并 `Update` -> `DisasterConfigs().Delete(name)` -> `WriteSuccess(200, {name})`
- operator 链路：删除对象带 `testudo.softcdata.com/config-finalizer` 时，`DisasterConfigReconciler.handleDelete` 进入删除分支并移除 finalizer；当前旧的 `DisasterInstance.spec.config` 引用检查代码被注释，不会因已有实例引用而阻止删除
- 下层资源链路：HTTP 删除接口本身不访问业务集群、Velero 或 S3；当前 operator 删除分支只移除 finalizer，没有清理已应用到源/目标集群的 Velero credential `Secret` 或 `BackupStorageLocation`
- 已写入内容：五段详细说明、path `name`、无 query/body 的事实、删除前 trace annotation best-effort 行为、finalizer 异步删除语义、当前不做实例引用删除保护的边界、当前错误分类
- 取证备注：删除前的 `Get` 和 `Update` 错误被忽略，最终 HTTP 结果以 `Delete` 调用为准；返回 `200` 后对象可能仍因 finalizer 处于 Terminating 状态，需要通过列表、详情或 watch 观察最终消失
- 主要错误点：JWT 失败返回 `401`；删除目标不存在返回 `404 code=3004`；删除调用非 NotFound 错误返回 `500 code=5000`；operator 后续 finalizer 处理失败、已有实例引用或下层 Velero 资源残留不改变本次 HTTP 响应

## GET /apis/disasterconfigs.testudo.softcdata.com/v1/configs/:name

- RunAPI Target ID：`25559f2138c092`
- RunAPI 状态：已存在，已更新详细说明，URL 已从具体样例名称规范为 `{{baseurl}}/apis/disasterconfigs.testudo.softcdata.com/v1/configs/:name`，已补齐 path `name` 参数和当前 200/401/404/500 响应示例，鉴权类型为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证；原候选 `1c304687b0801001` 已确认实际为 `GET /apis/disasterconfigs.testudo.softcdata.com/v1/configs/names`；本次追加 200 响应示例“详情成功（creation_timestamp 本地时区）”
- server 路由：`internal/apis/disaster_config/v1/router.go`
- server handler：`internal/apis/disaster_config/v1/handler.go`，`ConfigHandler.config`
- 请求链路：`path.name` -> `DisasterConfigs().Get(name)` -> `toDisasterConfigDTO` -> `ConvertToDisasterConfigDTO` 将 `metadata.creationTimestamp` 按 server 进程本地时区格式化为 RFC3339 offset 字符串 -> `populatePolicyCrons` 可选读取 `DisasterPolicy(disaster-system)` -> `WriteSuccess(200)`
- operator 链路：详情接口只读取状态，不触发 reconcile；`DisasterConfigReconciler` 后续校验源/目标 `Cluster`、`StorageRepository`，应用 Velero credential `Secret` 和 `BackupStorageLocation`，并写入 `status.status/reason/message`
- 下层资源链路：详情接口本身不访问业务集群、Velero 或 S3；下层异常只通过 operator 写入的 `status` 字段反映到详情 DTO
- 已写入内容：五段详细说明、path `name`、无 query/body 的事实、`DisasterConfigDTO` 字段、`creation_timestamp` 本地时区 RFC3339 offset 格式、`imageRewrite` DTO 字段、`resourcesSyncPolicy/resourceSyncPolicy` 同源回显、policy cron 补充规则、operator 状态原因、当前错误分类
- 取证备注：RunAPI 旧说明中的策略 cron 默认值与当前代码不一致；当前代码不会伪造默认 cron，只在 `DisasterPolicy` 存在且未禁用时返回 `policy.spec.schedule`，读取失败或策略禁用时为空
- 主要错误点：JWT 失败返回 `401`；目标配置不存在返回 `404 code=3004`；查询 CR 发生非 NotFound 错误返回 `500 code=5000`；关联策略读取失败不影响详情 HTTP 状态

## PUT /apis/disasterconfigs.testudo.softcdata.com/v1/configs/:name

- RunAPI Target ID：`3ee68505f8c067`
- RunAPI 状态：已存在，已更新详细说明，已补齐 `Authorization` header、path `name` 参数、当前 `imageRewrite` 请求体示例和 200/400/401/404/409/500 响应示例，鉴权类型为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证；本次追加 200 响应示例“更新成功（creation_timestamp 本地时区）”
- server 路由：`internal/apis/disaster_config/v1/router.go`
- server handler：`internal/apis/disaster_config/v1/handler.go`，`ConfigHandler.updateConfig`
- 请求链路：`path.name` 空值校验 -> `rejectUnsupportedSyncPolicyField` -> `BindJSON(UpdateDisasterConfigRequest)` -> `retry.RetryOnConflict` -> `Get(existing)` -> 计算 imageRewrite 校验用的 effective source/target -> `validateAndBuildImageRewrite` -> `MergeToCRD` 合并非空字段和策略指针字段 -> 更新 description 和 trace annotation -> `DisasterConfigs().Update` -> `toDisasterConfigDTO` -> `ConvertToDisasterConfigDTO` 将 `metadata.creationTimestamp` 按 server 进程本地时区格式化为 RFC3339 offset 字符串 -> `populatePolicyCrons` -> `WriteSuccess(200)`
- operator 链路：更新 spec 后由 `DisasterConfigReconciler` 重新同步依赖 labels，校验源/目标 `Cluster`、`StorageRepository`，向源/目标集群应用 Velero credential `Secret` 和 `BackupStorageLocation`，并刷新 `status.status/reason/message`
- 下层资源链路：HTTP 更新接口本身不访问业务集群、Velero 或 S3；只有更新 `imageRewrite` 且包含映射时读取源/目标 `Cluster.spec.imageSources` 做别名校验；真正下层资源更新由 operator 处理
- 已写入内容：五段详细说明、PUT 当前是合并更新而非完整替换的事实、path `name`、无 query、`description/sourceCluster/targetCluster/storageRepository/dataSyncType/dataSyncPolicy/resourcesSyncPolicy/resourceSyncPolicy/imageRewrite/imageSources/syncPolicy` 入参、哪些字段可清空、哪些字段空字符串不生效、`imageRewrite` 替换和不能直接清除为 nil 的边界、200 DTO 字段、`creation_timestamp` 本地时区 RFC3339 offset 格式、当前错误分类
- 取证备注：`sourceCluster/targetCluster/storageRepository/dataSyncType` 只有非空字符串才覆盖；`dataSyncPolicy` 和资源同步策略字段传空字符串可清空；`resourcesSyncPolicy` 一旦出现即优先于 `resourceSyncPolicy`，即使为空也不回退；`imageRewrite` 不传则保持原值，传空对象会保存禁用空配置而不是删除字段
- 主要错误点：path name 空、顶层 `syncPolicy`、JSON 绑定失败、`imageRewrite` 非法或别名对不匹配返回 `400 code=1000`；JWT 失败返回 `401`；目标配置不存在返回 `404 code=3004`；冲突重试耗尽返回 `409 code=3009`；查询或更新 CR 的其他错误返回 `500 code=5000`

## GET /apis/disasterconfigs.testudo.softcdata.com/v1/configs/names

- RunAPI Target ID：`1c304687b0801001`
- RunAPI 状态：已存在，已更新详细说明，已补齐 `Authorization` header、`environment/keyword` query 说明和当前 200/400/401 响应示例，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_config/v1/router.go`
- server handler：`internal/apis/disaster_config/v1/handler.go`，`ConfigHandler.configNames`
- 请求链路：`ParseOptions` -> `BuildLabelSelector` 空 selector -> `DisasterConfigLister.List` -> 动态 query 按 `metadata.labels` value 包含匹配 -> 组装 `DisasterConfigNameDTO{name/id/sourceCluster/targetCluster/status}` -> `WriteSuccess(200)`
- operator 链路：名称列表只读取 `DisasterConfig.status.status`，不触发 reconcile；`DisasterConfigReconciler` 后续负责校验依赖并维护 `Pending/Ready/NotReady/Error` 状态
- 下层资源链路：名称列表接口不访问业务集群、Velero、S3、`DisasterPolicy`、`DisasterInstance` 或 `DisasterOperation`
- 已写入内容：五段详细说明、无 path/body 的事实、动态 label query 过滤、`keyword/page/limit/sort/order` 当前不生效、返回轻量数组而不是 collection envelope、`DisasterConfigNameDTO` 字段、当前错误分类
- 取证备注：该接口调用 `ParseOptions` 但不使用 `Paginate`、`Sort` 或 `BuildCollectionResponse`；因此没有 `meta`，`page/limit/sort/order` 只会被通用解析排除在动态 filters 外，不影响返回数量或顺序
- 主要错误点：JWT 失败返回 `401`；Lister 列表失败返回 `400 code=1000`；无匹配项返回 `200 data=[]`

## GET /apis/disasterconfigs.testudo.softcdata.com/v1/watch/configs

- RunAPI Target ID：`267c3f73f8c40a`、`3b831dfe78c08c`
- RunAPI 状态：两个历史 `websocket2` 目标均已更新，已补齐 `Authorization`、`Sec-WebSocket-Protocol` header 和 `token` query，连接帧、心跳帧、事件帧、关闭帧、超时帧和错误说明已写入详细说明，已回读验证；两个目标原 description 均为空，因此未追加 `## 原有说明`
- server 路由：`internal/apis/disaster_config/v1/router.go`
- server handler：`internal/apis/disaster_config/v1/handler.go`，`ConfigHandler.watchConfigs`
- 请求链路：`WebSocketTokenAdapter` -> JWT 中间件 -> `DisasterConfigs().Watch(ListOptions{})` -> `StreamWatch` 推送 connected/heartbeat/watch event/closed/timeout/error 帧
- 返回转换：watch event object 为 `DisasterConfig` 时调用 `toDisasterConfigDTO`，其中 `ConvertToDisasterConfigDTO` 会将 `metadata.creationTimestamp` 按 server 进程本地时区格式化为 RFC3339 offset 字符串，并通过 `populatePolicyCrons` 可选读取 `DisasterPolicy(disaster-system)` 补齐 cron；事件对象不是 `DisasterConfig` 时 `data.object=null`
- operator 链路：`DisasterConfigReconciler` 对创建、更新、删除、依赖 labels、finalizer 和 status 的写入会触发 `ADDED/MODIFIED/DELETED` 等事件；本接口只推送对象变化，不触发 reconcile
- 下层资源链路：WebSocket 接口本身不访问业务集群、Velero 或 S3；下层资源处理结果通过 `DisasterConfig.status` 或对象字段变化反映到事件对象
- 已写入内容：五段详细说明、WebSocket token 三种传入方式、无 path/body/业务 query 的事实、不先 List 且不推当前全量快照的当前行为、完整事件帧字段、`DisasterConfigDTO` 字段、`creation_timestamp` 本地时区 RFC3339 offset 格式、policy cron 补充规则、心跳/关闭/超时/错误帧
- 取证备注：该接口直接 Watch 空 `ListOptions`，没有先 List 获取 `resourceVersion`；因此客户端需要先调用列表接口获取当前快照，再建立 WebSocket 接收增量变化
- 主要错误点：握手前 JWT 失败返回普通 HTTP JSON；WebSocket 升级失败返回 HTTP `500 {\"message\":\"WebSocket 升级失败: ...\"}`；创建 watcher 失败通过 WebSocket 推送 `code=5000` 错误帧；连接 30 分钟默认超时后推送 `meta.type=timeout`

## GET /apis/disasterconfigs.testudo.softcdata.com/v1/watch/configs/:name

- RunAPI Target ID：`267c641278c417`
- RunAPI 状态：已存在，已更新详细说明，URL 已从具体样例名称规范为 `{{baseurl}}/apis/disasterconfigs.testudo.softcdata.com/v1/watch/configs/:name`，已补齐 `Authorization`、`Sec-WebSocket-Protocol` header 和 `token` query，连接帧、心跳帧、事件帧、关闭帧、超时帧和错误说明已写入详细说明，已回读验证；原 description 为空，因此未追加 `## 原有说明`；`websocket2` 目标回读未保留 restful 参数容器，path `name` 已在 URL 和详细说明中明确
- server 路由：`internal/apis/disaster_config/v1/router.go`
- server handler：`internal/apis/disaster_config/v1/handler.go`，`ConfigHandler.watchConfig`
- 请求链路：`WebSocketTokenAdapter` -> JWT 中间件 -> `path.name` 空值校验 -> `DisasterConfigs().Watch(FieldSelector: metadata.name=<name>)` -> `StreamWatch` 推送 connected/heartbeat/watch event/closed/timeout/error 帧
- 返回转换：watch event object 为 `DisasterConfig` 时调用 `toDisasterConfigDTO`，其中 `ConvertToDisasterConfigDTO` 会将 `metadata.creationTimestamp` 按 server 进程本地时区格式化为 RFC3339 offset 字符串，并通过 `populatePolicyCrons` 可选读取 `DisasterPolicy(disaster-system)` 补齐 cron；事件对象不是 `DisasterConfig` 时 `data.object=null`
- operator 链路：目标配置的创建、更新、删除、依赖 labels、finalizer 和 status 更新会触发 `ADDED/MODIFIED/DELETED` 等事件；本接口只推送对象变化，不触发 reconcile
- 下层资源链路：WebSocket 接口本身不访问业务集群、Velero 或 S3；下层资源处理结果通过目标 `DisasterConfig.status` 或对象字段变化反映到事件对象
- 已写入内容：五段详细说明、WebSocket token 三种传入方式、path `name`、无 body/业务 query 的事实、不先 Get/List 且不推当前对象快照的当前行为、目标不存在不返回 404 的边界、完整事件帧字段、`creation_timestamp` 本地时区 RFC3339 offset 格式、心跳/关闭/超时/错误帧
- 取证备注：该接口直接使用 `FieldSelector` Watch，path `name` 不存在时连接仍可建立并保持心跳，直到同名对象被创建、修改、删除或连接结束
- 主要错误点：path `name` 为空会在握手前返回 `400 code=1000`；握手前 JWT 失败返回普通 HTTP JSON；WebSocket 升级失败返回 HTTP `500 {\"message\":\"WebSocket 升级失败: ...\"}`；创建 watcher 失败通过 WebSocket 推送 `code=5000` 错误帧；目标不存在不返回错误

## GET /apis/appbackups.testudo.softcdata.com/v1/appbackups

- RunAPI Target ID：`2a25b664f8c042`
- RunAPI 状态：已存在，已更新详细说明，URL 已从带默认 query 的地址规范为 `{{baseurl}}/apis/appbackups.testudo.softcdata.com/v1/appbackups`，已补齐 `Authorization` header、`origin/page/limit/sort/order/keyword` 和当前有效 appbackup label query，已补齐 200/400/401 响应示例，鉴权类型为继承项目鉴权，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/app_backup/v1/router.go`
- server handler：`internal/apis/app_backup/v1/handler.go`，`AppBackupHandler.appBackups`
- 请求链路：`ParseOptions` -> `parseAppResourceOriginFilter` 归一化 `origin` -> `BuildLabelSelector` 空 selector -> `AppBackupLister.AppBackups(disaster-system).List` -> `matchAppResourceOriginFilter` 按来源过滤 -> 动态 query 按 `metadata.labels` value 包含匹配 -> `Sort(name/creationTimestamp)` -> `Paginate` -> `ConvertToAppBackupDTO` -> `BuildCollectionResponse(resourceType=appBackup)` -> `WriteSuccess(200)`
- operator 链路：`AppBackupReconciler` 添加 finalizer，校验目标集群和 `StorageRepository`，应用 Velero credential `Secret` 和 `BackupStorageLocation`；Ready 阶段按 `schedule/disasterPolicy` 创建或更新 Velero `Schedule`，或创建一次性 Velero `Backup`；监听 Velero `Backup` 变化后同步 `status.backupStatus/history/latestBackupStatus/totalBackups/reason/message` 和统计
- 下层资源链路：列表接口本身不访问业务集群、Velero 或对象存储；下层 Velero `Schedule/Backup/BackupStorageLocation/Secret` 状态只通过 `AppBackup.status` 和 labels 反映
- 已写入内容：五段详细说明、无 path/body 的事实、`origin` 来源过滤、动态 label 模糊过滤、`keyword` 当前不生效、分页排序、`AppBackupDTO` 字段、列表 DTO 不返回完整 history 的边界、operator 状态和 Velero 下层资源链路、当前错误分类
- 取证备注：`origin=user` 默认会包含历史未打 `testudo.softcdata.com/app-resource-origin` 标签的数据；`origin=instance` 匹配来源 label 为 `disaster-instance`，或 controller ownerReference kind 为 `DataSync/ResourceSync` 的对象；RunAPI 旧参数 `testudo.softcdata.com/app-backup-namespace` 已清理为当前 operator 实际写入的 `testudo.softcdata.com/app-backup-include-namespace`
- 主要错误点：`origin` 非 `user/instance/all` 返回 `400 code=1000`；Lister 列表失败返回 `400 code=1000`；JWT 失败返回 `401`；无匹配项返回 `200 data.items=[]`；operator 下层失败不影响列表 HTTP 状态，只体现在 `status.phase/reason/message/latestBackupStatus`

## POST /apis/appbackups.testudo.softcdata.com/v1/appbackups

- RunAPI Target ID：`3ee68503b8c051`、`3efbb43238c57f`
- RunAPI 状态：两个历史目标均已更新；`3ee68503b8c051` 为“使用策略配置”场景，`3efbb43238c57f` 为“手动”场景，但两者实际同路由同 handler；均已补齐五段详细说明、`Authorization`/`Content-Type` header、当前合法 JSON body 示例和 201/400/401/409/500 响应示例，鉴权类型为继承项目鉴权，原说明均已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/app_backup/v1/router.go`
- server handler：`internal/apis/app_backup/v1/handler.go`，`AppBackupHandler.createAppBackup`
- 请求链路：`BindJSON(CreateAppBackupRequest)` -> `validateCreateAppBackupResourceFilters` 校验旧版资源过滤和 scoped 资源过滤不能混用且 include/exclude 不冲突 -> `ValidateClusterReady` 校验 `Cluster` 存在且 Ready -> `ValidateStorageRepositoryAvailable` 校验 `StorageRepository(disaster-system)` 存在且 Available -> 写入 description、trace 和 user annotation -> `req.ToCRD` -> `AppBackups(disaster-system).Create` -> `ConvertToAppBackupDTO` -> `WriteSuccess(201)`
- operator 链路：`AppBackupReconciler` 添加 finalizer 并记录创建任务事件；Pending 阶段校验目标集群客户端、存储仓库和 BSL；Ready 阶段按 `schedule/disasterPolicy` 创建或更新 Velero `Schedule`，或创建一次性 Velero `Backup`；监听 Velero Backup 后同步 `status.backupStatus/history/latestBackupStatus/totalBackups/reason/message` 和统计
- 下层资源链路：server 创建接口本身不访问业务集群、Velero 或对象存储；operator 后续访问目标集群，维护 Velero credential `Secret`、`BackupStorageLocation`、`Schedule`、`Backup`，对象存储数据由 Velero 写入
- 已写入内容：五段详细说明、无 path/query 的事实、`name/cluster/schedule/disasterPolicy/paused/skipImmediately/description`、Velero template 字段、旧版资源过滤字段、scoped 资源过滤字段、`volumeSnapshotLocations` 当前未写入 CR 的事实、AutoBackup 策略覆盖 schedule/ttl/paused 的 operator 规则、201 DTO 字段、当前错误分类
- 取证备注：RunAPI 历史“手动”样例未传 `schedule`，但当前 `CreateAppBackupRequest.Schedule` 标注 `binding:\"required\"`；新说明和新示例已按当前代码补齐 `schedule`。使用 scoped 资源过滤字段时，server 会把 `includeClusterResources` 归零为 nil，避免与 scoped 字段混用
- 主要错误点：JSON 绑定失败或缺少 `name/cluster/schedule` 返回 `400 code=1000`；资源过滤混用、通配符冲突或 include/exclude 冲突返回 `400 code=1000`；集群不存在/未 Ready 或存储仓库不存在/非 Available 返回 `400 code=1000`；JWT 失败返回 `401`；同名 AppBackup 返回 `409 code=3009`；创建 CR 非冲突错误返回 `500 code=5000`；`disasterPolicy` 不存在等 operator 后续问题不影响本次 HTTP 创建结果

## DELETE /apis/appbackups.testudo.softcdata.com/v1/appbackups/:name

- RunAPI Target ID：`2a25b66578c049`
- RunAPI 状态：已存在，已更新详细说明，URL 已从具体样例名称规范为 `{{baseurl}}/apis/appbackups.testudo.softcdata.com/v1/appbackups/:name`，已补齐 `Authorization` header 和 200/401/404/500 响应示例，鉴权类型为继承项目鉴权，原说明为空因此未追加 `## 原有说明`，已回读验证
- server 路由：`internal/apis/app_backup/v1/router.go`
- server handler：`internal/apis/app_backup/v1/handler.go`，`AppBackupHandler.deleteAppBackup`
- 请求链路：读取 `path.name` -> 尽力 `Get(AppBackup)` -> 写入 trace annotation 和 `testudo.softcdata.com/user` annotation -> 尽力 `Update(AppBackup)` -> `AppBackups(disaster-system).Delete(name)` -> NotFound 映射 `404 code=3004`，其他删除错误映射 `500 code=5000` -> 成功返回 `WriteSuccess(200, {"name": name})`
- operator 链路：Kubernetes delete 设置 `deletionTimestamp` 后触发 `AppBackupReconciler`；`DeletingHandler` 在 finalizer `testudo.softcdata.com/finalizer` 存在时记录删除任务 Started/Finished 事件，获取 `spec.cluster` 目标集群 client，调用 `deleteExternalResources` 清理下层资源，最后移除 finalizer
- 下层资源链路：operator 先按 label `testudo.softcdata.com/app-backup-uid=<AppBackup UID>` 删除目标集群 `velero` 命名空间 Velero `Schedule`，再列出同 UID 的 Velero `Backup` 并为每个未处于删除中的 Backup 创建 `DeleteBackupRequest`；随后最多等待约 5 秒，超时后尽力移除 Backup finalizer 并强制删除 Backup CR
- 已写入内容：五段详细说明、path `name`、无 query/body 的事实、成功返回 `data.name` 但不代表下层资源清理完成的异步边界、trace/user annotation 的 best-effort 写入、operator finalizer 删除流程、Velero Schedule/Backup/DeleteBackupRequest 清理链路、当前错误分类
- 取证备注：删除前 `Get/Update` 只用于补 trace/user 审计信息，失败会被忽略并继续执行 Delete；目标集群不存在时 operator 会跳过外部资源删除并移除 finalizer；目标集群连接失败、外部资源删除失败或对象存储清理异常不会由本 HTTP 请求同步返回
- 主要错误点：JWT 失败返回 `401`；AppBackup 不存在或已删除返回 `404 code=3004`；管理集群 Delete 返回非 NotFound 错误返回 `500 code=5000`；operator 异步清理异常只体现在事件、任务记录或资源终止状态中

## GET /apis/appbackups.testudo.softcdata.com/v1/appbackups/:name

- RunAPI Target ID：`2a25b664f8c043`
- RunAPI 状态：已存在，已更新详细说明，URL 已从具体样例名称规范为 `{{baseurl}}/apis/appbackups.testudo.softcdata.com/v1/appbackups/:name`，已补齐 `Authorization` header 和 200/400/401/404/500 响应示例，鉴权类型为继承项目鉴权，原 AutoBackup 策略回显说明已保留到 `## 原有说明`，已回读验证；候选 `1be2239988801001` 实际为 `GET /apis/appbackups.testudo.softcdata.com/v1/appbackups/clusters`，未在本接口处理
- server 路由：`internal/apis/app_backup/v1/router.go`
- server handler：`internal/apis/app_backup/v1/handler.go`，`AppBackupHandler.appBackup`
- 请求链路：读取 `path.name` -> 空值校验 -> `AppBackups(disaster-system).Get(name)` -> NotFound 映射 `404 code=3004`，其他查询错误映射 `500 code=5000` -> `ConvertToAppBackupDTO` -> 手动填充 `dto.Status.History = item.Status.History` -> `WriteSuccess(200)`
- operator 链路：`AppBackupReconciler` 在 Pending 阶段添加 finalizer、校验目标集群和 `StorageRepository`、应用 BSL；Ready 阶段按 `schedule/disasterPolicy` 维护 Velero `Schedule` 或一次性 Velero `Backup`，并通过 `syncStatus` 将 Velero Backup 观测结果合并到 `status.history/latestBackupStatus/backupStatus/totalBackups/reason/message`
- 下层资源链路：详情接口本身不访问业务集群、Velero 或对象存储；下层 Velero `Secret/BackupStorageLocation/Schedule/Backup/DeleteBackupRequest` 的状态只通过 operator 回写到 AppBackup CR 后展示
- 已写入内容：五段详细说明、path `name`、无 query/body 的事实、`AppBackupDTO` 全量字段、详情接口会填充完整 `status.history` 的差异、`BackupRecord` 字段、`lastAction` 字段、`ConvertSpecToDTO` 当前未回填部分 Velero template 字段的边界、AutoBackup 策略回显等待 reconcile 的原说明
- 取证备注：列表接口不返回完整 `status.history`，详情接口会在 `ConvertToAppBackupDTO` 后手动填充；`status.history` 由 operator 从 Velero Backup 列表合并并按开始时间倒序排序，被取消的记录会保留，非取消且已从 Velero 集群消失的历史记录会被移除
- 主要错误点：path `name` 为空返回 `400 code=1000`；JWT 失败返回 `401`；目标 AppBackup 不存在返回 `404 code=3004`；管理集群 Get 返回非 NotFound 错误返回 `500 code=5000`；operator 下层失败不影响详情 HTTP 状态，只体现在 `status.phase/reason/message/history`

## PUT /apis/appbackups.testudo.softcdata.com/v1/appbackups/:name

- RunAPI Target ID：`3ee6850438c053`
- RunAPI 状态：已存在，已更新详细说明，URL 已从具体样例名称规范为 `{{baseurl}}/apis/appbackups.testudo.softcdata.com/v1/appbackups/:name`，已补齐 `Authorization`/`Content-Type` header、path `name` 参数、当前 JSON body 示例和 200/400/401/404/409/500 响应示例，鉴权类型为继承项目鉴权，原说明为空因此未追加 `## 原有说明`，已回读验证
- server 路由：`internal/apis/app_backup/v1/router.go`
- server handler：`internal/apis/app_backup/v1/handler.go`，`AppBackupHandler.updateAppBackup`
- 请求链路：`BindJSON(UpdateAppBackupRequest)` -> `validateUpdateAppBackupResourceFilters` -> `RetryOnConflict` -> 按 body `req.Name` 执行 `Get(AppBackup)` -> 按 `description` 指针写入或删除描述 annotation -> `req.MergeToCRD(&existing.Spec)` 合并更新 spec -> 写入 trace/user annotation -> `AppBackups(disaster-system).Update` -> `ConvertToAppBackupDTO` -> `WriteSuccess(200)`
- operator 链路：AppBackup spec 更新触发 `AppBackupReconciler`；Pending/Ready 阶段重新校验目标集群、`StorageRepository` 和 BSL；Ready 阶段按新 `schedule/disasterPolicy/template` 创建、更新或重建 Velero `Schedule`，或维护一次性 Velero `Backup`；AutoBackup 策略会在 reconcile 中再次覆盖有效 `schedule/ttl/paused`
- 下层资源链路：server 更新接口本身不访问业务集群、Velero 或对象存储，也不校验新 cluster/repo；operator 后续访问目标集群，维护 Velero `Secret`、`BackupStorageLocation`、`Schedule`、`Backup`，并回写 status
- 已写入内容：五段详细说明、path `name` 当前不被 handler 读取的边界、body `name` 是实际更新对象、合并更新规则、哪些字段空值不清除、`description` 空字符串可清除、资源过滤校验、scoped 字段会归零 `includeClusterResources`、响应不返回完整 history、当前错误分类
- 取证备注：当前 handler 不校验 path `name` 与 body `name` 是否一致；`cluster/schedule/disasterPolicy/storageLocation` 只有非空才覆盖，不能用空字符串清空；数组字段只有非空数组才覆盖，不能用空数组清空；`paused/skipImmediately/snapshotVolumes/defaultVolumesToFsBackup/includeClusterResources` 是指针字段，传 `false` 会生效
- 主要错误点：JSON 绑定失败或缺少 body `name` 返回 `400 code=1000`；资源过滤字段混用、通配符冲突或 include/exclude 冲突返回 `400 code=1000`；JWT 失败返回 `401`；body `name` 对应 AppBackup 不存在返回 `404 code=3004`；资源版本持续冲突返回 `409 code=3009`；Get/Update 其他错误返回 `500 code=5000`；operator 下层失败不影响本次 HTTP 更新结果

## POST /apis/appbackups.testudo.softcdata.com/v1/appbackups/:name/actions/:type

- RunAPI Target ID：`1bd135d4c5c01001`、`1bd135d316401001`、`3f0947a838c682`、`3c1cb82178c001`、`1bd135d64b401001`
- RunAPI 状态：五个历史场景目标均已更新，URL 均规范为 `{{baseurl}}/apis/appbackups.testudo.softcdata.com/v1/appbackups/:name/actions/:type`；已补齐 `Authorization`/`Content-Type` header、path `name/type` 参数、可选 `targetBackup` JSON body、异步动作成功响应、pause/resume 成功响应和 400/401/404/409/500 响应示例，鉴权类型为继承项目鉴权；`retry/cancel/delete` 三个目标的原说明已保留到 `## 原有说明`，`backup/pause` 原说明为空因此未追加，已回读验证；2026-05-18 追加响应示例 `成功（暂停并记录手动意图）`，说明 pause 会写入 `testudo.softcdata.com/app-backup-manual-paused=true`
- server 路由：`internal/apis/app_backup/v1/router.go`
- server handler：`internal/apis/app_backup/v1/handler.go`，`AppBackupHandler.executeAction`
- 请求链路：读取 path `name/type` -> `type` 转小写 -> path name 空值校验 -> body 非空时 `BindJSON(AppBackupActionRequest)` -> `backup/retry/cancel/delete` 分支通过 `RetryOnConflict` 写入 `spec.action.type/targetBackup/requestAt` 和 trace/user annotation -> `pause/resume` 分支通过 `RetryOnConflict` 写入 `spec.paused`、trace/user annotation 和 `testudo.softcdata.com/app-backup-manual-paused=true/false` -> 成功返回动作信息或 paused 状态
- operator 链路：`AppBackupReconciler` Ready 阶段比较 `spec.action.requestAt` 与 `status.lastAction.requestAt`，只处理新动作；`backup` 确保 BSL 后创建 Velero `Backup`；`retry` 删除目标或最新 Backup 后按相同名称创建；`cancel` 删除 New/InProgress/空 phase 的 Backup 并把历史标记为 `Canceled`；`delete` 创建 `DeleteBackupRequest` 并清理 history；执行后写入 `status.lastAction` 并重新入队
- 下层资源链路：动作接口本身不访问业务集群、Velero 或对象存储；operator 后续访问目标集群，维护 Velero `BackupStorageLocation`、`Backup`、`DeleteBackupRequest`、`Schedule`，并通过 `status.history/latestBackupStatus/reason/message/lastAction` 反映结果
- 已写入内容：五段详细说明、六种 `type` 值、`targetBackup` 条件必传规则、同步响应和 operator 异步执行边界、pause/resume 与异步动作返回结构差异、`delete` 未传目标不会同步报错的边界、当前错误分类
- 取证备注：`delete` 的 `targetBackup` 在 server 侧不校验，缺失仍返回 200，operator 后续记录 `DeleteActionFailed` 并将动作标记已处理；`cancel` 指定目标不存在时通常视为已完成；`retry` 指定目标不存在时仍会尝试用该名称创建新 Backup；`pause/resume` 直接改 `spec.paused` 并记录手动暂停意图，operator 后续把手动意图与 AutoBackup 策略 Disabled 状态合成后同步到 Velero Schedule
- 主要错误点：path `name` 为空、body 非空但 JSON 非法、`type` 不支持返回 `400 code=1000`；JWT 失败返回 `401`；AppBackup 不存在返回 `404 code=3004`；资源版本持续冲突返回 `409 code=3009`；Get/Update 其他错误返回 `500 code=5000`；目标备份不存在、BSL 不可用、Velero Backup 创建/删除失败等 operator 下层问题不作为本次 HTTP 错误同步返回

## GET /apis/appbackups.testudo.softcdata.com/v1/appbackups/:name/backups/:backupName/download

- RunAPI Target ID：`1bd135d66a001001`
- RunAPI 状态：已存在，已更新详细说明，URL 已从具体样例名称规范为 `{{baseurl}}/apis/appbackups.testudo.softcdata.com/v1/appbackups/:name/backups/:backupName/download`，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，已补齐 `Authorization` header、path `name/backupName` 参数、`type` query、resource/data/all 当前响应示例和 400/401/404/500 响应示例，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/app_backup/v1/router.go`
- server handler：`internal/apis/app_backup/v1/handler.go`，`AppBackupHandler.downloadBackup`
- 请求链路：读取 path `name/backupName` 和 query `type` -> 空值校验 -> `Get(AppBackup)` -> 校验 `backupName` 存在于 `status.history[]` -> 校验 `spec.template.storageLocation` 非空 -> `Get(StorageRepository)` -> 可选读取 `caSecretRef` Secret 的 `ca.crt` -> `type=data/all` 分支列出对象并逐个生成预签名 URL，默认/resource 分支直接为 `<cluster>/backups/<backupName>/<backupName>.tar.gz` 生成预签名 URL -> `WriteSuccess(200)`
- operator 链路：appbackup operator 负责创建或观测 Velero `Backup` 并维护 `status.history`；Velero 负责把资源清单、Kopia/Restic 数据写入 `StorageRepository` 对应对象存储；本接口依赖 history 和存储配置生成下载 URL
- 下层资源链路：server 读取管理集群 AppBackup/StorageRepository/CA Secret，并连接 S3 兼容对象存储；`type=data` 列出 `<cluster>/kopia/<namespace>/` 和 `<cluster>/restic/<namespace>/`；`type=all` 额外列出 `<cluster>/backups/<backupName>/`
- 已写入内容：五段详细说明、当前不会流式返回二进制而是返回 JSON 预签名 URL 的事实、`type` 取值、对象 key/prefix 规则、`BackupDownloadResponse` 字段、1 小时过期时间、CA Secret 和 addressingStyle 使用、当前错误分类
- 取证备注：旧 RunAPI 说明称 `data/all` 直接返回 `application/tar+gzip` 数据流；当前代码实际调用 `ListObjects` 后返回 `data.files[]`，每个文件有独立 `download_url/key/size`；`type` 传非 `data/all` 值不会报错，会按默认 resource 生成单个 URL
- 主要错误点：path 缺失或 storageLocation 缺失返回 `400 code=1000`；JWT 失败返回 `401`；AppBackup 不存在、history 中无该 backup、StorageRepository 不存在或 data/all 无对象返回 `404 code=3004`；查询 CR、加载 CA、列对象或生成预签名 URL 失败返回 `500 code=5000`

## GET /apis/appbackups.testudo.softcdata.com/v1/appbackups/:name/history

- RunAPI Target ID：`1bd4dbb7f4401001`
- RunAPI 状态：已存在，已更新详细说明，URL 已从具体样例名称规范为 `{{baseurl}}/apis/appbackups.testudo.softcdata.com/v1/appbackups/:name/history`，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，已补齐 `Authorization` header、path `name` 参数、`status` query、当前 200/400/401/404/500 响应示例，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/app_backup/v1/router.go`
- server handler：`internal/apis/app_backup/v1/handler.go`，`AppBackupHandler.getBackupHistory`
- 请求链路：读取 path `name` -> 空值校验 -> `AppBackups(disaster-system).Get(name)` -> 可选读取 query `status` -> 如果 `status` 非空则按 `BackupRecord.phase == status` 精确过滤 -> `WriteSuccess(200, []BackupRecord)`
- operator 链路：`AppBackupReconciler` Ready 阶段通过 `syncStatus` 合并目标集群 Velero Backup 列表到 `status.history`；按 Backup 名称去重，更新 phase、时间、错误告警、过期时间和完整 Velero status；按开始时间倒序排序，并同步 `totalBackups/latestBackupStatus`
- 下层资源链路：历史查询接口本身不访问业务集群、Velero 或对象存储；下层状态由 operator 回写到 AppBackup CR 后才会出现在本接口
- 已写入内容：五段详细说明、`status` query 当前按 Velero 原始 phase 精确过滤而不是 managedStatus 的事实、`BackupRecord` 字段、无历史和过滤无匹配返回 `200 data=[]`、operator history 合并/清理规则、当前错误分类
- 取证备注：被取消的历史记录会保留；非 Canceled 且对应 Velero Backup 已从目标集群消失的历史记录会被 operator 移除；`status` 大小写敏感，传 `completed` 不会匹配 `Completed`
- 主要错误点：path `name` 为空返回 `400 code=1000`；JWT 失败返回 `401`；AppBackup 不存在返回 `404 code=3004`；管理集群 Get 返回非 NotFound 错误返回 `500 code=5000`；无历史或过滤无匹配不算错误

## GET /apis/appbackups.testudo.softcdata.com/v1/appbackups/clusters

- RunAPI Target ID：`1be2239988801001`
- RunAPI 状态：已存在，已更新详细说明，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，已补齐 `Authorization` header 和当前 200/400/401 响应示例，原说明已保留到 `## 原有说明`，已回读验证；此前该 Target 曾被列为详情接口候选，已确认为 clusters 接口
- server 路由：`internal/apis/app_backup/v1/router.go`
- server handler：`internal/apis/app_backup/v1/handler.go`，`AppBackupHandler.getAppBackupClusters`
- 请求链路：`AppBackupLister.AppBackups(disaster-system).List(labels.NewSelector())` -> 遍历全部 AppBackup -> 跳过空 `spec.cluster` -> map 去重 -> 组装 `[]ClusterNameDTO{{name}}` -> `WriteSuccess(200)`
- operator 链路：该接口只读取 `AppBackup.spec.cluster`，不触发 reconcile；`spec.cluster` 由创建/更新接口写入，operator 后续用它连接目标集群
- 下层资源链路：不访问 `Cluster` CR、目标业务集群、Velero 或对象存储，不校验集群是否存在或 Ready
- 已写入内容：五段详细说明、无 path/query/body 的事实、返回对象数组而不是字符串数组、无分页/排序/过滤的事实、返回顺序无保证、当前错误分类
- 取证备注：旧 RunAPI 示例是 `data: ["cluster1","cluster2"]`，当前代码实际返回 `data: [{"name":"cluster1"}]`；Go map 遍历无稳定排序，调用方需要自行排序
- 主要错误点：Lister 列表失败返回 `400 code=1000`；JWT 失败返回 `401`；没有 AppBackup 或没有非空 cluster 返回 `200 data=[]`

## GET /apis/appbackups.testudo.softcdata.com/v1/velero/backups/:backupName/includes

- RunAPI Target ID：`1c30a700d3001001`
- RunAPI 状态：已存在，已更新详细说明，URL 已从具体备份名和 query 样例规范为 `{{baseurl}}/apis/appbackups.testudo.softcdata.com/v1/velero/backups/:backupName/includes`，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，已补齐 `Authorization` header、path `backupName` 参数、必填 query `cluster`、当前 200 响应示例，原说明已保留到 `## 原有说明`，已回读验证；兼容 query `clusterName` 只写入详细说明，不在 RunAPI 参数区标为必填
- server 路由：`internal/apis/app_backup/v1/router.go`
- server handler：`internal/apis/app_backup/v1/velero_backup_includes.go`，`AppBackupHandler.getVeleroBackupIncludes`
- 请求链路：读取 path `backupName` -> 空值校验 -> 读取 query `cluster`，为空时兼容读取 `clusterName` -> `getRemoteClient(cluster)` -> 读取目标集群 `velero` 命名空间 Velero `Backup` -> 命中 `cluster|namespace|backupName` 且 `resourceVersion` 未变的 5 分钟缓存则直接返回 -> `computeVeleroBackupIncludes` -> 优先 `loadBackupResourceListUncached` 读取实际资源清单 -> 成功时从资源清单反推 namespace/resource 并排序 -> 失败时记录告警并退回 `Backup.spec.includedNamespaces/includedResources` -> `WriteSuccess(200)`
- operator 链路：AppBackup operator 负责根据 `StorageRepository` 维护目标集群 Velero `BackupStorageLocation`，并通过 Velero `Schedule` 或一次性 `Backup` 生成下层 Velero Backup；该接口只读取这些下层结果，不触发新的备份动作
- 下层资源链路：直接读取目标集群 Velero `Backup` 和可选 `BackupStorageLocation`；读取管理集群 `StorageRepository(disaster-system)` 和可选 CA `Secret`；优先生成对象存储 `<prefix>/backups/<backupName>/<backupName>-resource-list.json.gz` 的 2 分钟预签名 URL 并用 5 秒 HTTP 超时下载，prefix 来自 BSL `spec.objectStorage.prefix` 或默认 `cluster`；直接读取失败时创建临时 Velero `DownloadRequest(kind=BackupResourceList)`，最多等待 10 秒并在结束后尽力删除
- 已写入内容：五段详细说明、path/query/header 入参、兼容 `clusterName` 行为、无 body 的事实、实际资源清单优先于 Backup Spec 的当前行为、缓存 key/TTL/resourceVersion 失效规则、StorageRepository/BSL/CA Secret/对象存储/DownloadRequest 链路、`includedNamespaces/includedResources` 排序与兜底规则、当前错误分类
- 取证备注：旧 RunAPI 说明只写了来自 `Backup.spec.includedNamespaces/includedResources`；当前代码会优先解析实际 `BackupResourceList`，从 map key 生成资源类型，从 `namespace/name` 形式的资源实例字符串中提取 namespace，集群级资源如 `nodes: ["node-1"]` 不贡献 namespace；资源清单读取失败通常不返回错误，而是退回 Spec 字段
- 主要错误点：path `backupName` 为空返回 `400 code=1000`；query `cluster` 和 `clusterName` 都为空返回 `400 code=1000`；目标集群 remote client 获取失败返回 `400 code=1000`；Velero Backup 不存在返回 `404 code=3004`；remote client getter 未初始化、读取 Backup 发生非 NotFound 错误或内部 handler 初始化异常返回 `500 code=5000`；对象存储、CA、DownloadRequest、下载和解析失败一般不作为 HTTP 错误同步返回

## GET /apis/appbackups.testudo.softcdata.com/v1/watch/appbackups

- RunAPI Target ID：`2a418fe378c0a9`
- RunAPI 状态：已存在，已更新详细说明，URL 保持为 `{{baseurl}}/apis/appbackups.testudo.softcdata.com/v1/watch/appbackups`，已补齐 WebSocket 鉴权 header/query 参数、租户 header/query 参数和全部消息帧示例；`websocket2` 目标无 response 容器，连接成功、心跳、事件、关闭、超时和错误帧均写入 description，原说明为空因此未追加 `## 原有说明`，已回读验证
- server 路由：`internal/apis/app_backup/v1/router.go`
- server handler：`internal/apis/app_backup/v1/handler.go`，`AppBackupHandler.watchAppBackups`
- 请求链路：WebSocket token adapter 从 query `token` 或 `Sec-WebSocket-Protocol` 归一化到 `Authorization` -> JWT/tenant 中间件 -> `watchAppBackups` 构造 `AppBackups(disaster-system).Watch(ctx, metav1.ListOptions{})` -> `watchutils.StreamWatch` 升级 WebSocket -> 创建 watcher -> 发送 connected 帧 -> 每 30 秒心跳 -> 读取 Kubernetes watch event -> `ConvertToAppBackupDTO` -> 推送 `WatchEventDTO{type, object}` -> watcher 关闭、客户端断开或 30 分钟超时后结束
- operator 链路：事件对象里的 `status` 由 `AppBackupReconciler` 回写；operator 负责校验目标集群和 `StorageRepository`、维护 Velero `Secret/BackupStorageLocation/Schedule/Backup/DeleteBackupRequest`、同步 `status.scheduleStatus/backupStatus/totalBackups/latestBackupStatus/reason/message/lastAction`
- 下层资源链路：本接口只 watch 管理集群 `AppBackup` CR，不直接访问目标业务集群、Velero API、对象存储或 `StorageRepository`；下层状态变化需要 operator 回写 AppBackup 后才会以 `MODIFIED` 事件体现
- 已写入内容：五段详细说明、WebSocket token 三种传入方式、tenant 默认行为、无 path/body/业务 query 的事实、不支持筛选/分页/排序、不先推当前全量快照的边界、连接成功/心跳/事件/关闭/超时/error 帧、`WatchEventDTO` 字段、`AppBackupDTO` 字段、watch DTO 不返回完整 history 的事实、当前错误分类
- 取证备注：`watchAppBackups` 使用空 `ListOptions{}`，不会按 `origin/cluster/name/label` 过滤；连接成功时只发送 `connected` 帧，不会先 `List` 当前 AppBackup；`ConvertStatusToDTO` 不填充 `history`，只从 history 最后一条派生 `veleroBackupName`
- 主要错误点：WebSocket 升级前 JWT 失败返回普通 HTTP JSON；WebSocket 升级失败返回 HTTP `500 {"message":"WebSocket 升级失败: ..."}`；创建 watcher 失败通过 WebSocket 推送 `code=5000` 错误帧；Kubernetes watch `ERROR` 事件会作为事件帧推送，非 AppBackup 错误对象在当前 converter 下 `object=null`；无 AppBackup 不算错误，只保持 connected/heartbeat

## GET /apis/appbackups.testudo.softcdata.com/v1/watch/appbackups/:name

- RunAPI Target ID：`2aab0af7b0c0a6`
- RunAPI 状态：已存在，已更新详细说明，URL 已从具体样例名称规范为 `{{baseurl}}/apis/appbackups.testudo.softcdata.com/v1/watch/appbackups/:name`，已补齐 WebSocket 鉴权 header/query 参数、租户 header/query 参数、path `name` 说明和全部消息帧示例；`websocket2` 目标无 response 容器，连接成功、心跳、事件、关闭、超时和错误帧均写入 description，原说明为空因此未追加 `## 原有说明`，已回读验证；ApiPost websocket2 回读不展示 restful 容器，path 参数已通过 URL 和 description 固化
- server 路由：`internal/apis/app_backup/v1/router.go`
- server handler：`internal/apis/app_backup/v1/handler.go`，`AppBackupHandler.watchAppBackup`
- 请求链路：WebSocket token adapter 从 query `token` 或 `Sec-WebSocket-Protocol` 归一化到 `Authorization` -> JWT/tenant 中间件 -> 读取 path `name` -> 空值返回 `400 code=1000` -> 构造 `AppBackups(disaster-system).Watch(ctx, ListOptions{FieldSelector: metadata.name=<name>})` -> `watchutils.StreamWatch` 升级 WebSocket -> 创建 watcher -> 发送 connected 帧 -> 每 30 秒心跳 -> 读取 Kubernetes watch event -> `ConvertToAppBackupDTO` -> 推送 `WatchEventDTO{type, object}` -> watcher 关闭、客户端断开或 30 分钟超时后结束
- operator 链路：事件对象里的 `status` 由 `AppBackupReconciler` 回写；operator 负责校验目标集群和 `StorageRepository`、维护 Velero `Secret/BackupStorageLocation/Schedule/Backup/DeleteBackupRequest`、同步 `status.scheduleStatus/backupStatus/totalBackups/latestBackupStatus/reason/message/lastAction`
- 下层资源链路：本接口只 watch 管理集群指定名称的 `AppBackup` CR，不先读取对象、不直接访问目标业务集群、Velero API、对象存储或 `StorageRepository`；下层状态变化需要 operator 回写该 AppBackup 后才会以 `MODIFIED` 事件体现
- 已写入内容：五段详细说明、WebSocket token 三种传入方式、tenant 默认行为、path `name`、无 body/业务 query 的事实、field selector 行为、不先 Get/List 且不推当前对象快照的边界、目标不存在不返回 404 的当前行为、连接成功/心跳/事件/关闭/超时/error 帧、`WatchEventDTO` 字段、`AppBackupDTO` 字段、watch DTO 不返回完整 history 的事实、当前错误分类
- 取证备注：`watchAppBackup` 不校验对象是否存在；`name` 当前不存在时连接仍可建立并保持心跳，后续同名对象创建时可能收到 `ADDED` 事件；`ConvertStatusToDTO` 不填充 `history`，只从 history 最后一条派生 `veleroBackupName`
- 主要错误点：path `name` 为空会在握手前返回 `400 code=1000`；WebSocket 升级前 JWT 失败返回普通 HTTP JSON；WebSocket 升级失败返回 HTTP `500 {"message":"WebSocket 升级失败: ..."}`；创建 watcher 失败通过 WebSocket 推送 `code=5000` 错误帧；Kubernetes watch `ERROR` 事件会作为事件帧推送，非 AppBackup 错误对象在当前 converter 下 `object=null`；目标不存在不算错误

## GET /apis/apprestores.testudo.softcdata.com/v1/apprestores

- RunAPI Target ID：`34a0ae4978c001`
- RunAPI 状态：已存在，已更新详细说明，URL 已从带默认 query 的地址规范为 `{{baseurl}}/apis/apprestores.testudo.softcdata.com/v1/apprestores`，已补齐 `Authorization` header、`origin/page/limit/sort/order/keyword/targetNamespaces` 和当前有效 apprestore label query，已补齐当前 200/400 响应示例，鉴权类型为继承项目鉴权，原来源过滤说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/app_restore/v1/router.go`
- server handler：`internal/apis/app_restore/v1/handler.go`，`AppRestoreHandler.appRestores`
- 请求链路：`ParseOptions` -> `parseAppResourceOriginFilter` 归一化 `origin` -> `BuildLabelSelector` 空 selector -> `AppRestoreLister.AppRestores(disaster-system).List` -> `matchAppResourceOriginFilter` 按来源过滤 -> `targetNamespaces` 特殊过滤 -> 其他动态 query 按 `metadata.labels` value 包含匹配 -> `Sort(name/creationTimestamp)` -> `Paginate` -> `ConvertToAppRestoreDTO` -> `BuildCollectionResponse(resourceType=appRestore)` -> `WriteSuccess(200)`
- operator 链路：`AppRestoreReconciler` 添加 finalizer，获取目标集群 client；跨集群恢复时根据 `sourceCluster/storageRepository` 在目标集群预加载 Velero `BackupStorageLocation`；读取目标集群 Velero `Backup` 作为恢复源；计算 `status.targetNamespaces`；同步源 AppBackup 的 `Manual/Schedule` 类型 label；创建资源修改 `ConfigMap` 和 Velero `Restore(res-<name>)`；观测 Restore phase 后回写 `status.status/restoreStatus/reason/message/lastAction` 并同步统计
- 下层资源链路：列表接口本身不访问目标业务集群、Velero 或对象存储；目标集群连接、BSL、Velero Backup/Restore、ConfigMap 和 PVR 状态只通过 operator 回写到 AppRestore CR 后展示
- 已写入内容：五段详细说明、无 path/body 的事实、`origin` 来源过滤、演练资源 `ddr-/drr-` 和 drill label 兜底、动态 label 模糊过滤、`targetNamespaces` 非 label 特殊过滤、`keyword` 当前不生效、分页排序、`AppRestoreDTO` 字段、operator 状态和 Velero 下层资源链路、当前错误分类
- 取证备注：`origin=user` 默认会排除 DataSync/ResourceSync ownerReference、`disaster-instance` 标签、`testudo.softcdata.com/drill` 标签以及名称前缀 `ddr-`/`drr-` 的恢复任务；`targetNamespaces` 优先使用 `status.targetNamespaces`，为空时从 `namespaceMapping` 的目标值或 `includedNamespaces` 推导；`testudo.softcdata.com/app-restore-updated` 是历史旧标签，当前 operator 会删除该标签，未作为推荐过滤项写入
- 主要错误点：`origin` 非 `user/instance/all` 返回 `400 code=1000`；Lister 列表失败返回 `400 code=1000`；JWT 失败返回 `401` 或中间件自定义普通 JSON；无匹配项返回 `200 data.items=[]`；operator 下层失败不影响列表 HTTP 状态，只体现在 `status.phase/reason/message/restoreStatus`

## POST /apis/apprestores.testudo.softcdata.com/v1/apprestores

- RunAPI Target ID：`3ee6850478c056`
- RunAPI 状态：已存在，已更新详细说明，URL 已规范为 `{{baseurl}}/apis/apprestores.testudo.softcdata.com/v1/apprestores`，鉴权类型为继承项目鉴权，已补齐 `Authorization` 和 `Content-Type` header、当前合法 JSON body 示例、`name/backupSource/cluster` 必填 schema、201/400/404 响应示例，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/app_restore/v1/router.go`
- server handler：`internal/apis/app_restore/v1/handler.go`，`AppRestoreHandler.createAppRestore`
- 请求链路：`BindJSON(CreateAppRestoreRequest)` -> `resolveStorageClassMapping(storageClassMapping, scMapping)` -> 构造 `AppRestore(disaster-system)` 并写入描述、trace、user annotation -> 读取源 `AppBackup(disaster-system, backupSource)` -> 校验源 AppBackup `spec.cluster` 和 `spec.template.storageLocation` -> body 未传 `backupName` 时使用 `AppBackup.status.history[0].name` -> 由恢复映射字段生成 `ResourceModifierRules` -> `ValidateClusterReady(cluster)` -> 获取目标集群 client -> `VerifyRestorePreflight(targetCli, runtimeClient, appBackup, cluster, 0)` -> 校验 `existingResourcePolicy` 只能为空、`none`、`update` -> 创建 `AppRestore` -> `ConvertToAppRestoreDTO` -> `WriteSuccess(201)`
- operator 链路：`AppRestoreReconciler` 添加 finalizer；跨集群恢复时根据 `sourceCluster/storageRepository` 在目标集群创建或刷新 Velero `BackupStorageLocation`；读取目标集群 Velero `Backup`；计算 `status.targetNamespaces`；同步源 AppBackup 的 `Manual/Schedule` 类型 label；创建资源修改规则 `ConfigMap`；创建目标集群 `velero` 命名空间下的 Velero `Restore(res-<AppRestore.name>)`；观测 Restore phase 后回写 `status.status/restoreStatus/reason/message/lastAction`；删除或取消时清理 Velero Restore、ResourceModifier ConfigMap 和部分 Pending 资源
- 下层资源链路：创建接口同步读取管理集群 `AppBackup` 和 `Cluster`，并通过目标集群 client 校验 Velero `BackupStorageLocation`；operator 后续访问目标集群 Velero `Backup/Restore/PodVolumeRestore`、资源修改 `ConfigMap`、存储仓库对应 BSL 和实际业务资源
- 已写入内容：五段详细说明、无 path/query 的事实、`name/backupSource/cluster` 必填、`backupName` 缺省推导、`existingResourcePolicy` 取值和校验时机、`timeout` 写入 `spec.template.itemOperationTimeout`、`storageClassMapping` 与 `scMapping` 兼容冲突规则、`restorePVs/cleanVolumes` 自动追加幂等 PVC `volumeName` 清理规则、StorageClass/IngressClass 映射、缩容/待机/无流量恢复规则、uploader 配置、创建成功 DTO 字段、operator 异步恢复边界、当前错误分类
- 取证备注：`cleanVolumes=true` 或 `restorePVs=true` 都会追加 PVC 清理规则；当前规则使用 JSON Patch `add /spec/volumeName` 且 value 为空字符串，避免字段不存在时 `remove` 失败导致 Velero Restore `PartiallyFailed`；update 会将 legacy `remove /spec/volumeName` 规范化为新的幂等规则。`scaleToZeroList` 和 `standbyList` 传单元素 `*` 时匹配全部，否则生成 workload 名称正则且当前代码不转义正则特殊字符；创建成功只代表 `AppRestore` CR 创建完成，不代表 Velero Restore 已完成；`existingResourcePolicy` 当前在 preflight 之后校验，非法值可能先触发目标集群前置校验
- 主要错误点：JSON 绑定失败、必填缺失、StorageClass 映射冲突、源 AppBackup 缺少必要字段、未传 `backupName` 且源 AppBackup 无 history、目标集群不存在或非 Ready、目标集群 client 获取失败、preflight 异常或未通过、`existingResourcePolicy` 非法均返回 `400/500` 对应业务错误；源 AppBackup 不存在返回 `404 code=3004`；同名 AppRestore 返回 `409 code=3009`；创建 CR 非冲突错误返回 `500 code=5000`；JWT 失败返回 `401` 或中间件自定义普通 JSON；Velero Restore 创建失败、资源修改 ConfigMap 创建失败、恢复执行失败、PVR 卡住和恢复超时等 operator 异步问题不作为创建接口同步错误返回

## DELETE /apis/apprestores.testudo.softcdata.com/v1/apprestores/:name

- RunAPI Target ID：`34a0ae4b38c008`
- RunAPI 状态：已存在，已更新详细说明，URL 已从具体样例名称规范为 `{{baseurl}}/apis/apprestores.testudo.softcdata.com/v1/apprestores/:name`，鉴权类型为继承项目鉴权，已补齐 `Authorization` header、path `name` 参数、当前 200/404/500 响应示例；原说明为空，因此未追加 `## 原有说明`，已回读验证
- server 路由：`internal/apis/app_restore/v1/router.go`
- server handler：`internal/apis/app_restore/v1/handler.go`，`AppRestoreHandler.deleteAppRestore`
- 请求链路：读取 path `name` -> 尽力 `Get(AppRestore)` -> 如果对象存在则写入 trace/user annotation 并尽力 `Update`，失败不阻塞 -> `AppRestores(disaster-system).Delete(name)` -> NotFound 返回 `404 code=3004` -> 其他删除错误返回 `500 code=5000` -> 成功返回 `200 data.name`
- operator 链路：`AppRestoreReconciler` 检测 `metadata.deletionTimestamp` 后进入 `DeletingHandler`；如果存在 finalizer `testudo.softcdata.com/finalizer`，记录删除任务事件，获取 `spec.cluster` 对应目标集群 client；目标集群不存在时跳过外部资源清理；目标集群可访问时调用 `deleteExternalResources`，删除关联 Velero Restore 和 ResourceModifier ConfigMap；清理成功后记录完成事件，并由主 reconcile 循环移除 finalizer
- 下层资源链路：server 同步删除管理集群 `AppRestore` CR；operator 异步删除目标集群 `velero` 命名空间中 label `apprestore.testudo.softcdata.com/uid=<AppRestore.UID>` 匹配的 Velero `Restore` 和 ResourceModifier `ConfigMap`；不会直接删除业务命名空间中已经恢复成功的 Deployment、Service、PVC 等应用资源
- 已写入内容：五段详细说明、无 query/body 的事实、path `name`、删除前 trace/user annotation 尽力补充且失败不阻塞、成功响应只返回名称、finalizer 异步清理边界、operator 删除 Velero Restore 和 ResourceModifier ConfigMap 的 label 匹配规则、当前错误分类
- 取证备注：`200` 只表示删除请求已提交；如果对象带 finalizer，短时间内仍可能在列表或详情中看到删除中对象；如果 operator 获取目标集群时返回 NotFound，会跳过下层资源清理并继续移除 finalizer；删除链路不负责回滚已经恢复成功的业务资源
- 主要错误点：AppRestore 不存在返回 `404 code=3004`；删除 CR 时 API Server 返回非 NotFound 错误则返回 `500 code=5000`；JWT 失败返回 `401` 或中间件自定义普通 JSON；删除前 Get/Update annotation 失败、operator 下层清理失败不会作为本次 HTTP 同步错误返回

## GET /apis/apprestores.testudo.softcdata.com/v1/apprestores/:name

- RunAPI Target ID：`34a0ae49b8c002`
- RunAPI 状态：已存在，已更新详细说明，URL 已从具体样例名称规范为 `{{baseurl}}/apis/apprestores.testudo.softcdata.com/v1/apprestores/:name`，鉴权类型为继承项目鉴权，已补齐 `Authorization` header、path `name` 参数、当前 200/404/500 响应示例；原说明为空，因此未追加 `## 原有说明`，已回读验证
- server 路由：`internal/apis/app_restore/v1/router.go`
- server handler：`internal/apis/app_restore/v1/handler.go`，`AppRestoreHandler.appRestore`
- 请求链路：读取 path `name` -> 空值返回 `400 code=1000` -> `AppRestores(disaster-system).Get(name)` -> NotFound 返回 `404 code=3004` -> 其他查询错误返回 `500 code=5000` -> `ConvertToAppRestoreDTO` -> `WriteSuccess(200)`
- operator 链路：`AppRestoreReconciler` 根据 `spec.cluster` 连接目标集群；跨集群恢复时根据 `spec.sourceCluster/storageRepository` 预加载 Velero BSL；读取目标集群 Velero Backup；创建 ResourceModifier ConfigMap 和 Velero Restore；观测 Restore phase、进度、告警、错误和 hook 状态后回写 `status.restoreStatus`，并维护 `status.status/reason/message/lastAction/targetNamespaces`
- 下层资源链路：详情接口本身只读取管理集群 `AppRestore` CR，不访问目标集群、Velero API、StorageRepository、BackupStorageLocation、PodVolumeRestore、对象存储或业务资源；下层状态必须先由 operator 回写到 AppRestore 后才会在本接口展示
- 已写入内容：五段详细说明、无 query/body 的事实、path `name`、`AppRestoreDTO` metadata/spec/status 字段、`timeout` 优先来自 `spec.template.itemOperationTimeout` 且兼容 `spec.timeout`、`targetNamespaces` 优先 status 并按 namespaceMapping 兜底推导、`backupSourceType` 来源 label、DTO 不直接返回 `resourceModifierRules/sourceCluster/storageRepository/action` 的事实、当前错误分类
- 取证备注：旧 RunAPI 详情示例存在未包裹 `code/message/data` 的历史样例，已更新为当前 `WriteSuccess` 包裹结构；`targetNamespaces` 在 operator 尚未写入时由 server DTO 从 `includedNamespaces` 和 `namespaceMapping` 推导，operator 未处理且未配置 includedNamespaces 时为空；operator 下层失败不影响详情 HTTP 状态，只体现在响应 status 或事件中
- 主要错误点：path `name` 为空返回 `400 code=1000`；AppRestore 不存在返回 `404 code=3004`；查询 CR 非 NotFound 错误返回 `500 code=5000`；JWT 失败返回 `401` 或中间件自定义普通 JSON；目标集群不可访问、Velero Backup 不存在、Velero Restore 失败、恢复超时和 PVR 异常等不作为详情接口同步错误返回

## PUT /apis/apprestores.testudo.softcdata.com/v1/apprestores/:name

- RunAPI Target ID：`3ee68504b8c058`
- RunAPI 状态：已存在，已更新详细说明，URL 保持为 `{{baseurl}}/apis/apprestores.testudo.softcdata.com/v1/apprestores/:name`，鉴权类型为继承项目鉴权，已补齐 `Authorization`/`Content-Type` header、path `name` 参数、当前合法 JSON body 示例、无 binding 必填 body 字段的 schema、200/400/404/409 响应示例，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/app_restore/v1/router.go`
- server handler：`internal/apis/app_restore/v1/handler.go`，`AppRestoreHandler.updateAppRestore`
- 请求链路：`BindJSON(UpdateAppRestoreRequest)` -> 归一化 path `name` 和兼容 body `name` -> 校验 URL/body 名称一致 -> `resolveStorageClassMapping(storageClassMapping, scMapping)` -> 校验 `existingResourcePolicy` 只能为空、`none`、`update` -> `RetryOnConflict` 内 Get 现有 AppRestore -> `MergeToCRD` 合并非空/非 nil 字段 -> 追加 StorageClass/IngressClass/ScaleToZero/Standby ResourceModifierRule -> `cleanVolumes=true` 或 `restorePVs=true` 时确保幂等 PVC `volumeName` 清理规则并替换 legacy remove -> 更新 trace/user annotation -> 按需更新或删除 description annotation -> Update AppRestore -> `ConvertToAppRestoreDTO` -> `WriteSuccess(200)`
- operator 链路：更新 AppRestore CR 后触发 reconcile；如果对象仍在 `Pending/Initiating/Restoring` 等阶段，operator 后续可能读取更新后的 `spec.cluster/spec.template/spec.resourceModifierRules`；如果 Velero Restore 已创建，很多 RestoreSpec 字段和 resource modifier 规则不会自动作用到已创建的 Velero Restore，通常需要配合动作接口 retry；已处于 `Succeeded/Failed/Cancelled` 等终态的对象不会因普通 spec 更新自动重置为 `Pending`
- 下层资源链路：更新接口本身只读写管理集群 `AppRestore` CR，不访问源 AppBackup、目标 Cluster、目标集群、Velero API、BSL 或对象存储；operator 后续才可能根据更新后的 CR 访问目标集群 Velero Backup/Restore、ResourceModifier ConfigMap 和业务资源
- 已写入内容：五段详细说明、无 query 的事实、path `name` 与 body `name` 兼容和冲突规则、所有可更新 body 字段、`description` 空字符串删除 annotation、数组/map 空值不会清空旧值、`cleanVolumes=false` 不删除已有清理规则、映射/缩容/待机规则追加而非替换、更新不做 Ready/preflight/源 AppBackup 校验、成功 DTO 字段、当前错误分类
- 取证备注：旧 RunAPI schema 把 body `name` 和 `cleanVolumes` 标成必填，与当前代码不符，已改为无 body 必填字段；`storageClassMapping/ingressClassMapping/scaleToZeroList/standbyList` 重复更新可能追加重复 rules，只有 PVC `volumeName` 清理规则会通过 `ensureCleanVolumeRule` 去重并规范化为 `add /spec/volumeName` 空值；Update 成功响应不直接返回 `spec.resourceModifierRules`
- 主要错误点：JSON 绑定失败、资源名为空、URL/body 名称不一致、StorageClass 兼容字段冲突、`existingResourcePolicy` 非法返回 `400 code=1000`；AppRestore 不存在返回 `404 code=3004`；资源版本冲突多次重试后仍失败返回 `409 code=3009`；Get/Update 非 NotFound/Conflict 错误返回 `500 code=5000`；JWT 失败返回 `401` 或中间件自定义普通 JSON；更新后的源/目标/Velero 下层问题不作为本接口同步错误返回

## POST /apis/apprestores.testudo.softcdata.com/v1/apprestores/:name/actions/:type

- RunAPI Target ID：`3c211cc3f8c006`、`3c211cc3f8c008`
- RunAPI 状态：两个历史目标均已更新；`3c211cc3f8c006` 为取消示例，`3c211cc3f8c008` 为重试示例，但两者实际同路由同 handler；URL 均已从具体样例动作规范为 `{{baseurl}}/apis/apprestores.testudo.softcdata.com/v1/apprestores/:name/actions/:type`，鉴权类型为继承项目鉴权，已补齐 `Authorization` header、path `name/type` 参数、当前 200/400/404/409 响应示例；原说明为空，因此未追加 `## 原有说明`，两个目标均已回读验证
- server 路由：`internal/apis/app_restore/v1/router.go`
- server handler：`internal/apis/app_restore/v1/handler.go`，`AppRestoreHandler.executeAction`
- 请求链路：读取 path `name/type` -> name 为空返回 `400 code=1000` -> `strings.ToLower(type)` 后只允许 `cancel` 或 `retry` -> `RetryOnConflict` 内 Get 现有 AppRestore -> 写入 `spec.action.type` 和 `spec.action.requestAt=now` -> 更新 trace/user annotation -> Update AppRestore -> 返回 `200 data.type/request_at`
- operator 链路：`processAction` 通过 `spec.action.requestAt` 与 `status.lastAction.requestAt` 判断是否为新动作；`cancel` 会强制删除目标集群 Velero Restore，必要时移除 Restore finalizer，并按 `velero.io/restore-name=res-<name>` 清理部分 Pending/失败中间资源，成功后进入 `Cancelled` 并清空 `status.restoreStatus`；`retry` 在有 restoreStatus phase 时删除旧 Velero Restore 并清空 `status.restoreStatus`，随后返回 `Pending` 重新排队并触发后续重新创建 Restore
- 下层资源链路：server 同步阶段只更新管理集群 AppRestore CR，不访问目标集群；operator 后续访问目标集群 Velero Restore、Restore finalizer、目标命名空间中的 Deployment/StatefulSet/Job/独立 Pending Pod/Pending PVC 等中间资源
- 已写入内容：五段详细说明、无 query/body 的事实、`type` 大小写不敏感且仅支持 `cancel/retry`、同步响应只代表动作请求写入、`request_at` 字段来源、operator 异步执行和执行 phase 边界、取消/重试下层资源链路、当前错误分类
- 取证备注：动作接口不校验当前 AppRestore phase、目标集群可达性或 Velero Restore 是否存在；`processAction` 当前在 `Restoring/Failed/Cancelled` 等处理器中执行，写入动作后不会由 server 立即执行；重复动作依赖新的 `requestAt` 触发；取消时清理 Pending 资源失败只记录日志，不一定使 cancel 动作失败
- 主要错误点：path `name` 为空或 `type` 不是 `cancel/retry` 返回 `400 code=1000`；AppRestore 不存在返回 `404 code=3004`；资源版本冲突多次重试后仍失败返回 `409 code=3009`；Get/Update 非 NotFound/Conflict 错误返回 `500 code=5000`；JWT 失败返回 `401` 或中间件自定义普通 JSON；当前 phase 不适合执行动作、目标集群不可访问、Velero Restore 删除失败、重试失败等 operator 异步问题不作为本接口同步错误返回

## POST /apis/apprestores.testudo.softcdata.com/v1/apprestores/preflight/validate

- RunAPI Target ID：`1c385ce4c7801001`
- RunAPI 状态：已存在，已更新详细说明，URL 保持为 `{{baseurl}}/apis/apprestores.testudo.softcdata.com/v1/apprestores/preflight/validate`，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，已补齐 `Authorization`/`Content-Type` header、`backupSource/targetCluster/waitSeconds` body schema、当前 `message/trace_id` envelope 响应示例、200 true/200 false/404/400 响应示例，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/app_restore/v1/router.go`
- server handler：`internal/apis/app_restore/v1/handler.go`，`AppRestoreHandler.validateRestorePreflight`
- 请求链路：`BindJSON(ValidateRestorePreflightRequest)` -> `Get(AppBackup(disaster-system, backupSource))` -> `getClusterClient(targetCluster)` -> `VerifyRestorePreflight(targetCli, runtimeClient, appBackup, targetCluster, waitSeconds)` -> verifier 出错返回 `500` 附带 meta -> verifier 返回 nil 返回 `500` -> `result.Valid=false` 返回 `200 data=false meta` -> `result.Valid=true` 返回 `200 data=true meta`
- verifier 链路：从 AppBackup 读取 `spec.cluster` 和 `spec.template.storageLocation` -> 计算 required BSL `<storageRepository>-<sourceCluster>` -> 读取目标集群 `velero` 命名空间 BSL -> BSL `Available` 返回 valid true -> BSL `Unavailable` 返回 valid false -> BSL 不存在时 patch 管理集群 `Cluster(targetCluster)` 注解 `testudo.softcdata.com/ensure-storage` 和 `testudo.softcdata.com/ensure-storage-source-cluster` -> 每秒重新检查 BSL，等待秒数按 `<=0` 默认 20、`>60` 截断 60 归一化 -> 超时返回 `data=false meta.state=NotFound` 或最后观测到的 BSL phase
- 下层资源链路：接口本身不创建 AppRestore，不启动 Velero Restore，不访问对象存储；它读取源 AppBackup、目标集群 Velero BSL，并在 BSL 缺失时 patch 管理集群 Cluster 注解，由下层控制器异步创建或刷新目标集群 BSL
- 已写入内容：五段详细说明、无 path/query 的事实、body 必填 `backupSource/targetCluster`、`waitSeconds` 数字类型和边界、required BSL 命名规则、ensure-storage 注解、`data=true/false` 语义、meta 字段来源、`data=false` 仍为 HTTP 200 的边界、当前错误分类
- 取证备注：旧 RunAPI 以 `msg` 为响应字段且鉴权为 `noauth`，已按当前 `transport.WriteSuccess/WriteError` 修正为 `message/trace_id` 和继承项目鉴权；AppBackup 缺少 `spec.cluster` 或 `spec.template.storageLocation` 不返回 400，而是 `200 data=false` 并在 meta.error 中说明；创建接口内部调用 preflight 时 `waitSeconds=0`，即默认等待 20 秒
- 主要错误点：JSON 绑定失败、缺少 `backupSource/targetCluster` 或 `waitSeconds` 类型错误返回 `400 code=1000`；目标集群 client 获取失败返回 `400 code=1000`；源 AppBackup 不存在返回 `404 code=3004`；查询源 AppBackup 非 NotFound 错误、目标 BSL 查询异常、patch ensure-storage 失败、runtime client 缺失或 verifier nil 返回 `500 code=5000`；JWT 失败返回 `401` 或中间件自定义普通 JSON；BSL 不存在、Unavailable、源 AppBackup 缺字段和等待超时作为校验未通过返回 `200 data=false`

## GET /apis/apprestores.testudo.softcdata.com/v1/watch/apprestores

- RunAPI Target ID：`34a10209f8c00c`
- RunAPI 状态：已存在，已更新详细说明，URL 保持为 `{{baseurl}}/apis/apprestores.testudo.softcdata.com/v1/watch/apprestores`，已补齐 WebSocket 鉴权 header/query 参数和全部消息帧示例；`websocket2` 目标无 response 容器，连接成功、心跳、事件、关闭、超时和错误帧均写入 description；原说明为空，因此未追加 `## 原有说明`，已回读验证
- server 路由：`internal/apis/app_restore/v1/router.go`
- server handler：`internal/apis/app_restore/v1/handler.go`，`AppRestoreHandler.watchAppRestores`
- 请求链路：`WebSocketTokenAdapter` 从 query `token` 或 `Sec-WebSocket-Protocol` 归一化到 `Authorization` -> JWT 中间件 -> `watchAppRestores` 构造 `AppRestores(disaster-system).Watch(ctx, ListOptions{})` -> `watchutils.StreamWatch` 升级 WebSocket -> 创建 watcher -> 发送 connected 帧 -> 每 30 秒 heartbeat -> 读取 Kubernetes watch event -> `ConvertToAppRestoreDTO` -> 推送 `WatchEventDTO{type, object}` -> watcher 关闭、客户端断开或 30 分钟超时后结束
- operator 链路：事件对象里的 `status/labels/annotations` 由 `AppRestoreReconciler` 回写；operator 负责目标集群 client、BSL、Velero Backup/Restore、ResourceModifier ConfigMap、取消/重试、PVR 状态和统计同步
- 下层资源链路：本接口只 watch 管理集群全部 AppRestore CR，不直接访问目标业务集群、Velero API、StorageRepository、BackupStorageLocation、对象存储或业务资源；下层状态变化必须先由 operator 回写 AppRestore 后才会以 `MODIFIED` 事件体现
- 已写入内容：五段详细说明、WebSocket token 三种传入方式、当前 `/apis` 路由未启用 TenantID 中间件的事实、无 path/body/业务 query 的事实、不支持筛选/分页/排序、不先 List 推当前全量快照的边界、连接成功/心跳/事件/关闭/超时/error 帧、`WatchEventDTO` 字段和 AppRestore DTO 字段、当前错误分类
- 取证备注：`watchAppRestores` 使用空 `ListOptions{}`，不会按 `origin/cluster/name/label` 过滤；连接成功时只发送 `connected` 帧；Kubernetes `ERROR` event 的对象通常不是 AppRestore，当前 converter 会返回 `object=null`
- 主要错误点：WebSocket 升级前 JWT 失败返回普通 HTTP JSON；WebSocket 升级失败返回 HTTP `500 {"message":"WebSocket 升级失败: ..."}`；创建 watcher 失败通过 WebSocket 推送 `code=5000` 错误帧；Kubernetes watch `ERROR` 事件作为事件帧推送；无 AppRestore 不算错误，只保持 connected/heartbeat

## GET /apis/apprestores.testudo.softcdata.com/v1/watch/apprestores/:name

- RunAPI Target ID：`34a12bb3b8c041`
- RunAPI 状态：已存在，已更新详细说明，URL 保持为 `{{baseurl}}/apis/apprestores.testudo.softcdata.com/v1/watch/apprestores/:name`，已补齐 WebSocket 鉴权 header/query 参数、path `name` 说明和全部消息帧示例；`websocket2` 目标无 response 容器，连接成功、心跳、事件、关闭、超时和错误帧均写入 description；原说明为空，因此未追加 `## 原有说明`，已回读验证；ApiPost websocket2 回读不展示 restful 容器，path 参数已通过 URL 和 description 固化
- server 路由：`internal/apis/app_restore/v1/router.go`
- server handler：`internal/apis/app_restore/v1/handler.go`，`AppRestoreHandler.watchAppRestore`
- 请求链路：`WebSocketTokenAdapter` 从 query `token` 或 `Sec-WebSocket-Protocol` 归一化到 `Authorization` -> JWT 中间件 -> 读取 path `name` -> 空值在握手前返回 HTTP `400 {"message":"name is required"}` -> 构造 `AppRestores(disaster-system).Watch(ctx, ListOptions{FieldSelector: metadata.name=<name>})` -> `watchutils.StreamWatch` 升级 WebSocket -> 创建 watcher -> 发送 connected 帧 -> 每 30 秒 heartbeat -> 读取 Kubernetes watch event -> `ConvertToAppRestoreDTO` -> 推送 `WatchEventDTO{type, object}` -> watcher 关闭、客户端断开或 30 分钟超时后结束
- operator 链路：事件对象里的 `status/labels/annotations` 由 `AppRestoreReconciler` 回写；operator 负责目标集群 client、BSL、Velero Backup/Restore、ResourceModifier ConfigMap、取消/重试、PVR 状态和统计同步
- 下层资源链路：本接口只 watch 管理集群指定名称的 AppRestore CR，不先读取对象、不直接访问目标业务集群、Velero API、StorageRepository、BackupStorageLocation、对象存储或业务资源；下层状态变化必须先由 operator 回写该 AppRestore 后才会以 `MODIFIED` 事件体现
- 已写入内容：五段详细说明、WebSocket token 三种传入方式、当前 `/apis` 路由未启用 TenantID 中间件的事实、path `name`、无 body/业务 query 的事实、field selector 行为、不先 Get/List 且不推当前对象快照的边界、目标不存在不返回 404、连接成功/心跳/事件/关闭/超时/error 帧、`WatchEventDTO` 字段和 AppRestore DTO 字段、当前错误分类
- 取证备注：`watchAppRestore` 不校验对象是否存在；`name` 当前不存在时连接仍可建立并保持心跳，后续同名对象创建时可能收到 `ADDED` 事件；path `name` 为空时返回的是普通 `{"message":"name is required"}`，不是 transport envelope
- 主要错误点：path `name` 为空会在握手前返回 HTTP 400；WebSocket 升级前 JWT 失败返回普通 HTTP JSON；WebSocket 升级失败返回 HTTP `500 {"message":"WebSocket 升级失败: ..."}`；创建 watcher 失败通过 WebSocket 推送 `code=5000` 错误帧；Kubernetes watch `ERROR` 事件作为事件帧推送，非 AppRestore 对象在当前 converter 下 `object=null`；目标不存在不算错误

## GET /healthz

- RunAPI Target ID：`25559f1e78c062`
- RunAPI 状态：已存在，已更新详细说明，URL 保持为 `{{baseurl}}/healthz`，鉴权类型已修正为 `noauth`，已清理无效 header/query/body 参数，响应示例已改为当前纯文本 `ok`，原说明为空，因此未追加 `## 原有说明`，已回读验证
- server 路由：`internal/router/router.go`
- server handler：匿名 handler `sh.GET("/healthz", func(ctx context.Context, c *app.RequestContext) { c.String(200, "ok") })`
- 请求链路：请求命中根路由 `/healthz` -> 不经过 `/apis` 或 `/api` 分组中间件 -> 不执行 JWT -> handler 直接 `c.String(200, "ok")`
- operator 链路：无；该接口不访问 disaster-operator，也不依赖任何 CR 状态
- 下层资源链路：不读取或写入 Kubernetes CR、Secret、ConfigMap、StorageRepository、Cluster、Velero 资源、对象存储或数据库
- 已写入内容：五段详细说明、无入参、公开接口无鉴权、纯文本响应而不是 JSON envelope、无业务错误分支
- 取证备注：该接口只能证明 HTTP 进程和路由仍可响应，不能证明 informer、Kubernetes API、operator、对象存储或业务集群可用
- 主要错误点：handler 无固定业务错误；进程不可达、端口未监听、网关不可达或路径被上游拦截时，错误由网络层、网关或部署平台返回

## POST /login

- RunAPI Target ID：`25559f1e78c064`
- RunAPI 状态：已存在，已更新详细说明，URL 保持为 `{{baseurl}}/login`，鉴权类型已修正为 `noauth`，已补齐 `Content-Type` header、`username/password` body schema、当前成功响应 `data.accessToken/refreshToken/expire/userid/username` 示例和 400/403 错误示例，原说明已保留到 `## 原有说明`，已回读验证；RunAPI 后置任务 `$.data.accessToken -> token` 已保留
- server 路由：`internal/router/router.go`
- server handler：`internal/middleware/jwt.go`，`jwtMiddleware.LoginHandler`
- 请求链路：`LoginHandler` -> `Authenticator` 使用 `BindAndValidate(LoginRequest)` 解析 `username/password` -> `authenticateLogin` -> `KubeStore.GetUserByUsername` 读取 Secret `disaster-server-users` key `users.json` -> bcrypt 校验密码 -> 禁用用户返回 `ErrUserDisabled` -> `PayloadFunc` 写 access token claims `id/username/exp` -> `GenerateRefreshToken` 写 refresh token claims `id/username/exp/type=refresh` -> `LoginResponse` 返回 `200 {code:0,data:{accessToken,refreshToken,expire,userid,username}}`
- operator 链路：无；该接口不访问 disaster-operator，也不依赖任何 CR reconcile
- 下层资源链路：读取管理集群 `disaster-system` 命名空间 Secret `disaster-server-users`；服务启动阶段 `EnsureInitialized` 会初始化默认 `admin/123456`；不访问 Velero、目标业务集群或对象存储
- 已写入内容：五段详细说明、公开接口无鉴权、`username/password` 必填、bcrypt 校验、用户禁用逻辑、access token 默认 24 小时和 refresh token 默认 7 天、token claims、成功响应不属于 `transport.Envelope` 的事实、JWT middleware `code/msg` 错误结构、当前错误分类
- 取证备注：旧 RunAPI 成功示例把 `accessToken` 放在顶层，已修正为当前代码的 `data.accessToken`；旧文档中的 `jwt.timeout` 已按当前配置字段修正为 `jwt.accessExpire/refreshExpire`
- 主要错误点：用户不存在、密码错误、Secret 不存在、用户文档解析失败、密码哈希缺失或用户存储未初始化统一走认证失败，通常返回 `400 {code:400,msg:<jwt.ErrFailedAuthentication>}`；用户状态 `disabled` 返回 `403 {code:403,msg:"user is disabled"}`；缺少登录字段或 JSON 绑定失败由 JWT middleware 返回 `401` 或其传入的状态；服务启动时用户 Secret 初始化失败会导致服务不可用

## GET /readyz

- RunAPI Target ID：`25559f1e78c063`
- RunAPI 状态：已存在，已更新详细说明，URL 保持为 `{{baseurl}}/readyz`，鉴权类型已修正为 `noauth`，已清理无效 header/query/body 参数，响应示例已改为当前纯文本 `ok`，原说明为空，因此未追加 `## 原有说明`，已回读验证
- server 路由：`internal/router/router.go`
- server handler：匿名 handler `sh.GET("/readyz", func(ctx context.Context, c *app.RequestContext) { c.String(200, "ok") })`
- 请求链路：请求命中根路由 `/readyz` -> 不经过 `/apis` 或 `/api` 分组中间件 -> 不执行 JWT -> handler 直接 `c.String(200, "ok")`
- operator 链路：无；该接口不访问 disaster-operator，也不依赖任何 CR 状态
- 下层资源链路：不读取或写入 Kubernetes CR、Secret、ConfigMap、StorageRepository、Cluster、Velero 资源、对象存储或数据库
- 已写入内容：五段详细说明、无入参、公开接口无鉴权、纯文本响应而不是 JSON envelope、当前就绪检查不校验 informer/Kubernetes/operator/对象存储的事实、无业务错误分支
- 取证备注：该接口当前只能证明 HTTP 进程和路由仍可响应，不能证明 informer 已同步、ClusterClient 已启动、Kubernetes API 可用或业务集群 Ready
- 主要错误点：handler 无固定业务错误；进程不可达、端口未监听、网关不可达或路径被上游拦截时，错误由网络层、网关或部署平台返回

## POST /refresh_token

- RunAPI Target ID：`1bf8e6af53c01001`
- RunAPI 状态：已存在，已更新详细说明，URL 保持为 `{{baseurl}}/refresh_token`，鉴权类型保持 `noauth`，已清理不需要的 `Authorization` header，补齐 `Content-Type` header、`refreshToken` body schema、当前成功响应 `data.accessToken/expire` 示例和 400/401 错误示例，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/router/router.go`
- server handler：`internal/middleware/jwt.go`，`RefreshTokenHandler`
- 请求链路：`BindAndValidate(RefreshRequest)` -> 空 `refreshToken` 返回 `400 code=1` -> `jwt.Parse(refreshToken)` 并校验 HS256 签名和有效性 -> claims 必须是 `jwt.MapClaims` -> 如果 claims 存在 `type` 且不等于 `refresh` 返回 `401` -> 从 claims 读取 `id/username` -> `GenerateAccessToken` 使用配置 `jwt.secret` 和 `jwt.accessExpire` 生成新 access token -> 返回 `200 {code:0,data:{accessToken,expire}}`
- operator 链路：无；该接口不访问 disaster-operator，也不依赖任何 CR reconcile
- 下层资源链路：不读取用户 Secret，不重新检查用户存在性或禁用状态，不访问 Kubernetes 业务资源、Velero、目标集群或对象存储；只使用本地 JWT 配置和传入 token 的 claims
- 已写入内容：五段详细说明、公开接口无鉴权、`refreshToken` 必填、不会轮换 refresh token、access token claims 和默认有效期、成功响应不属于 `transport.Envelope` 的事实、`code/msg` 错误结构、当前错误分类
- 取证备注：旧 RunAPI 保留了不需要的 Authorization header 且存在顶层 `accessToken` 成功示例，已修正为当前代码实际的 `data.accessToken`；当前代码允许无 `type` claim 的历史 token，但如果存在 `type` 则必须为 `refresh`
- 主要错误点：JSON 绑定失败返回 `400 {code:1,msg:"invalid request"}`；缺少 refreshToken 返回 `400 {code:1,msg:"refresh token required"}`；解析失败、签名错误、过期或无效返回 `401 {code:401,msg:"invalid refresh token"}`；claims 类型错误返回 `401 {code:401,msg:"invalid token claims"}`；传入 access token 返回 `401 {code:401,msg:"not a refresh token"}`；生成 access token 失败返回 `500 {code:1,msg:"failed to generate token"}`；签名有效但缺少 `id/username` 的异常 token 可能触发 panic 并由全局 recovery 处理

## GET /api/v1/users

- RunAPI Target ID：`1c4ce9c769001001`
- RunAPI 状态：已存在，已更新详细说明，URL 已从带查询串样例改为 `{{baseurl}}/api/v1/users`，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，已补齐 `Authorization` header 和 `page/limit/sort/order/keyword` query 参数，当前 collection 响应示例已回读验证，原说明已保留到 `## 原有说明`
- server 路由：`internal/apis/user/v1/router.go`；`internal/router/router.go` 同时在 `/api` 和 `/apis` 分组注册用户 handler，本条为 `/api` 兼容路径
- server handler：`internal/apis/user/v1/handler.go`，`UserHandler.listUsers`
- 请求链路：`/api` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> `transport.ParseOptions` -> `KubeStore.ListUsers` -> `EnsureInitialized` -> 读取 Secret `disaster-system/disaster-server-users` key `users.json` -> `normalizeUser` -> `toUserDTO` -> `keyword` 对 `username/email/role/status` 大小写不敏感包含过滤 -> `transport.Sort` -> `transport.Paginate` -> `transport.BuildCollectionResponse` -> `transport.WriteSuccess(200)`
- operator 链路：无；用户管理不使用 disaster-operator CRD/controller，不触发 reconcile
- 下层资源链路：读取并可能初始化管理集群 Kubernetes Secret `disaster-server-users`；不访问 Velero、StorageRepository、业务集群、对象存储或统计 CRD
- 已写入内容：五段详细说明、JWT header、分页/排序/关键字参数、`limit=-1` 全量返回、`sort` 支持字段、collection envelope、`UserDTO` 字段来源、Secret 初始化和默认 admin 回填行为、当前错误分类
- 取证备注：`meta.links.<username>` 会生成用户详情链接，但当前 server 未实现 `GET /api/v1/users/:username`；`keyword` 参与过滤但不会出现在 `meta.filters`；旧 RunAPI 第一条成功示例为 `data` 数组，已修正为当前 `data.items` collection 结构
- 主要错误点：JWT 失败由中间件返回 `401/403` 普通 JSON；Store 未初始化返回 `500 code=5000`；Secret 初始化/读取/更新失败或 `users.json` 解码失败返回 `500 code=5000`

## POST /api/v1/users

- RunAPI Target ID：`1c4cd93b5fc01001`
- RunAPI 状态：已存在，已更新详细说明，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，已补齐 `Authorization` 和 `Content-Type` header，已补齐当前 `username/email/password` JSON body schema，成功响应 HTTP 状态码已修正为 `201`，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/user/v1/router.go`；`internal/router/router.go` 同时在 `/api` 和 `/apis` 分组注册用户 handler，本条为 `/api` 兼容路径
- server handler：`internal/apis/user/v1/handler.go`，`UserHandler.createUser`
- 请求链路：`/api` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> `BindJSON(CreateUserRequest)` -> `normalizeCreateRequest` -> `validateUsername/validateEmail/validatePassword` -> `actorFromContext` -> `KubeStore.CreateUser` -> `EnsureInitialized` -> `retry.RetryOnConflict` 读取 Secret `disaster-server-users` -> 用户名唯一校验 -> 邮箱大小写不敏感唯一校验 -> `HashPassword` bcrypt 哈希 -> 写入 `users.json.users.<username>` -> 更新 Secret -> `toUserDTO` -> `transport.WriteSuccess(201)`
- operator 链路：无；用户管理不使用 disaster-operator CRD/controller，不触发 reconcile
- 下层资源链路：写入管理集群 Kubernetes Secret `disaster-system/disaster-server-users` 的 `users.json`；不访问 Velero、StorageRepository、业务集群、对象存储或统计 CRD
- 已写入内容：五段详细说明、JWT 和 JSON header、创建请求字段、用户名正则、邮箱唯一性、密码长度与 bcrypt 上限、当前固定 `role=admin/status=active`、不返回密码/hash/id/updatedAt 的事实、错误分类
- 取证备注：当前请求结构不包含 `role/status`，调用方传入也不会生效；创建接口成功只表示用户 Secret 已写入，不会触发 operator 或任何外部账号系统
- 主要错误点：JWT 失败由中间件返回 `401/403` 普通 JSON；JSON 绑定、用户名、邮箱、密码校验失败返回 `400 code=1000`；用户名或邮箱重复返回 `409 code=3009`；Store 未初始化、Secret 初始化/读取/更新、bcrypt 哈希或文档解码失败返回 `500 code=5000`

## DELETE /api/v1/users/:username

- RunAPI Target ID：`1c4ce9c787001001`
- RunAPI 状态：已存在，已更新详细说明，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，已补齐 `Authorization` header 和 path `username` 参数，成功响应已补充 `trace_id`，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/user/v1/router.go`；`internal/router/router.go` 同时在 `/api` 和 `/apis` 分组注册用户 handler，本条为 `/api` 兼容路径
- server handler：`internal/apis/user/v1/handler.go`，`UserHandler.deleteUser`
- 请求链路：`/api` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> 读取并 trim path `username` -> `validateUsername` -> `actorFromContext` -> `KubeStore.DeleteUser` -> `EnsureInitialized` -> 拒绝 `admin` -> `retry.RetryOnConflict` 读取 Secret `disaster-server-users` -> 检查目标用户存在 -> 从 `users.json.users` 删除目标 key -> 更新 `UpdatedAt/UpdatedBy` -> 更新 Secret -> `transport.WriteSuccess(200)`
- operator 链路：无；用户管理不使用 disaster-operator CRD/controller，不触发 reconcile
- 下层资源链路：更新管理集群 Kubernetes Secret `disaster-system/disaster-server-users` 的 `users.json`；不访问 Velero、StorageRepository、业务集群、对象存储或统计 CRD
- 已写入内容：五段详细说明、JWT header、path 用户名规则、内置 `admin` 禁删、删除响应字段、当前错误分类
- 取证备注：删除成功只删除 server 本地用户文档记录，不会注销已签发 token，不会删除任何 Kubernetes RBAC 用户或外部身份源账号
- 主要错误点：JWT 失败由中间件返回 `401/403` 普通 JSON；path 用户名非法或删除 `admin` 返回 `400 code=1000`；目标不存在返回 `404 code=3004`；Store 未初始化、Secret 初始化/读取/更新或文档解码失败返回 `500 code=5000`

## PATCH /api/v1/users/:username/password

- RunAPI Target ID：`1c4ce9c7a7401001`
- RunAPI 状态：已存在，已更新详细说明，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，已补齐 `Authorization` 和 `Content-Type` header，已补齐 path `username` 和当前 `password` JSON body schema，成功响应已补充 `trace_id`，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/user/v1/router.go`；`internal/router/router.go` 同时在 `/api` 和 `/apis` 分组注册用户 handler，本条为 `/api` 兼容路径
- server handler：`internal/apis/user/v1/handler.go`，`UserHandler.patchUserPassword`
- 请求链路：`/api` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> 读取并 trim path `username` -> `validateUsername` -> `BindJSON(PatchUserPasswordRequest)` -> trim `password` -> `validatePassword` -> `actorFromContext` -> `KubeStore.UpdateUserPassword` -> `EnsureInitialized` -> `HashPassword` bcrypt 哈希 -> `retry.RetryOnConflict` 读取 Secret `disaster-server-users` -> 检查用户存在 -> 更新 `passwordHash/updatedAt` 和文档 `UpdatedAt/UpdatedBy` -> 更新 Secret -> `toUserDTO` -> `transport.WriteSuccess(200)`
- operator 链路：无；用户管理不使用 disaster-operator CRD/controller，不触发 reconcile
- 下层资源链路：更新管理集群 Kubernetes Secret `disaster-system/disaster-server-users` 的 `users.json`；不访问 Velero、StorageRepository、业务集群、对象存储或统计 CRD
- 已写入内容：五段详细说明、JWT 和 JSON header、path 用户名规则、密码长度 6-72 与 bcrypt 上限、响应不返回密码/hash/updatedAt、不会主动吊销已签发 token 的事实、当前错误分类
- 取证备注：修改密码不会改变 `role/status/createdAt`；禁用用户仍可被修改密码；新密码仅影响后续登录校验
- 主要错误点：JWT 失败由中间件返回 `401/403` 普通 JSON；path 用户名非法、JSON 绑定失败或密码校验失败返回 `400 code=1000`；目标不存在返回 `404 code=3004`；Store 未初始化、Secret 初始化/读取/更新、bcrypt 哈希或文档解码失败返回 `500 code=5000`

## PATCH /api/v1/users/:username/status

- RunAPI Target ID：`1c4cd93b7b001001`
- RunAPI 状态：已存在，已更新详细说明，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，已补齐 `Authorization` 和 `Content-Type` header，已补齐 path `username` 和当前 `status` JSON body schema，成功响应已补充 `trace_id`，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/user/v1/router.go`；`internal/router/router.go` 同时在 `/api` 和 `/apis` 分组注册用户 handler，本条为 `/api` 兼容路径
- server handler：`internal/apis/user/v1/handler.go`，`UserHandler.patchUserStatus`
- 请求链路：`/api` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> 读取并 trim path `username` -> `validateUsername` -> `BindJSON(PatchUserStatusRequest)` -> trim/lower `status` -> `validateStatus(active|disabled)` -> `actorFromContext` -> `KubeStore.UpdateUserStatus` -> `EnsureInitialized` -> `retry.RetryOnConflict` 读取 Secret `disaster-server-users` -> 检查用户存在 -> 更新 `status/updatedAt` 和文档 `UpdatedAt/UpdatedBy` -> 更新 Secret -> `toUserDTO` -> `transport.WriteSuccess(200)`
- operator 链路：无；用户管理不使用 disaster-operator CRD/controller，不触发 reconcile
- 下层资源链路：更新管理集群 Kubernetes Secret `disaster-system/disaster-server-users` 的 `users.json`；登录链路 `authenticateLogin` 会读取该状态，`disabled` 用户后续登录返回 `ErrUserDisabled`；不访问 Velero、StorageRepository、业务集群、对象存储或统计 CRD
- 已写入内容：五段详细说明、JWT 和 JSON header、path 用户名规则、`active/disabled` 状态含义、登录链路影响、不会主动吊销已签发 token 的事实、当前错误分类
- 取证备注：JWT 中间件校验已有 access token 时不重新读取用户 Secret，因此禁用账号不能立即拦截已经签发且未过期的 access token；状态更新不改变 `role/email/createdAt`
- 主要错误点：JWT 失败由中间件返回 `401/403` 普通 JSON；path 用户名非法、JSON 绑定失败或 status 非 `active/disabled` 返回 `400 code=1000`；目标不存在返回 `404 code=3004`；Store 未初始化、Secret 初始化/读取/更新或文档解码失败返回 `500 code=5000`

## GET /apis/v1/users

- RunAPI Target ID：`1c7b437084c01001`
- RunAPI 状态：缺失，已新增到 `用户管理` 目录，已写入五段详细说明，已补齐 `Authorization` header 和 `page/limit/sort/order/keyword` query 参数，当前 collection 响应示例已回读验证
- server 路由：`internal/apis/user/v1/router.go`；`internal/router/router.go` 在 `/apis` 分组注册 `userv1.NewUserHandler(store, bashPath)`，本条为 `/apis` 主路径
- server handler：`internal/apis/user/v1/handler.go`，`UserHandler.listUsers`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> `transport.ParseOptions` -> `KubeStore.ListUsers` -> `EnsureInitialized` -> 读取 Secret `disaster-system/disaster-server-users` key `users.json` -> `normalizeUser` -> `toUserDTO` -> `keyword` 对 `username/email/role/status` 大小写不敏感包含过滤 -> `transport.Sort` -> `transport.Paginate` -> `transport.BuildCollectionResponse` -> `transport.WriteSuccess(200)`
- operator 链路：无；用户管理不使用 disaster-operator CRD/controller，不触发 reconcile
- 下层资源链路：读取并可能初始化管理集群 Kubernetes Secret `disaster-server-users`；不访问 Velero、StorageRepository、业务集群、对象存储或统计 CRD
- 已写入内容：五段详细说明、JWT header、分页/排序/关键字参数、`limit=-1` 全量返回、`sort` 支持字段、collection envelope、`UserDTO` 字段来源、Secret 初始化和默认 admin 回填行为、当前错误分类
- 取证备注：该接口与 `/api/v1/users` 共用同一 handler；`meta.links.<username>` 会生成 `/apis/v1/users/<username>`，但当前 server 未实现用户详情 GET
- 主要错误点：JWT 失败由中间件返回 `401/403` 普通 JSON；Store 未初始化返回 `500 code=5000`；Secret 初始化/读取/更新失败或 `users.json` 解码失败返回 `500 code=5000`

## POST /apis/v1/users

- RunAPI Target ID：`1c7b43df0d401000`
- RunAPI 状态：缺失，已新增到 `用户管理` 目录，已写入五段详细说明，已补齐 `Authorization` 和 `Content-Type` header，已补齐当前 `username/email/password` JSON body schema，成功响应 HTTP 状态码已设为 `201`，已回读验证
- server 路由：`internal/apis/user/v1/router.go`；`internal/router/router.go` 在 `/apis` 分组注册 `userv1.NewUserHandler(store, bashPath)`，本条为 `/apis` 主路径
- server handler：`internal/apis/user/v1/handler.go`，`UserHandler.createUser`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> `BindJSON(CreateUserRequest)` -> `normalizeCreateRequest` -> `validateUsername/validateEmail/validatePassword` -> `actorFromContext` -> `KubeStore.CreateUser` -> `EnsureInitialized` -> `retry.RetryOnConflict` 读取 Secret `disaster-server-users` -> 用户名唯一校验 -> 邮箱大小写不敏感唯一校验 -> `HashPassword` bcrypt 哈希 -> 写入 `users.json.users.<username>` -> 更新 Secret -> `toUserDTO` -> `transport.WriteSuccess(201)`
- operator 链路：无；用户管理不使用 disaster-operator CRD/controller，不触发 reconcile
- 下层资源链路：写入管理集群 Kubernetes Secret `disaster-system/disaster-server-users` 的 `users.json`；不访问 Velero、StorageRepository、业务集群、对象存储或统计 CRD
- 已写入内容：五段详细说明、JWT 和 JSON header、创建请求字段、用户名正则、邮箱唯一性、密码长度与 bcrypt 上限、当前固定 `role=admin/status=active`、不返回密码/hash/id/updatedAt 的事实、错误分类
- 取证备注：该接口与 `/api/v1/users` 共用同一 handler；当前请求结构不包含 `role/status`，调用方传入也不会生效
- 主要错误点：JWT 失败由中间件返回 `401/403` 普通 JSON；JSON 绑定、用户名、邮箱、密码校验失败返回 `400 code=1000`；用户名或邮箱重复返回 `409 code=3009`；Store 未初始化、Secret 初始化/读取/更新、bcrypt 哈希或文档解码失败返回 `500 code=5000`

## DELETE /apis/v1/users/:username

- RunAPI Target ID：`1c7b440fdec01001`
- RunAPI 状态：缺失，已新增到 `用户管理` 目录，已写入五段详细说明，已补齐 `Authorization` header 和 path `username` 参数，成功响应已回读验证
- server 路由：`internal/apis/user/v1/router.go`；`internal/router/router.go` 在 `/apis` 分组注册 `userv1.NewUserHandler(store, bashPath)`，本条为 `/apis` 主路径
- server handler：`internal/apis/user/v1/handler.go`，`UserHandler.deleteUser`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> 读取并 trim path `username` -> `validateUsername` -> `actorFromContext` -> `KubeStore.DeleteUser` -> `EnsureInitialized` -> 拒绝 `admin` -> `retry.RetryOnConflict` 读取 Secret `disaster-server-users` -> 检查目标用户存在 -> 从 `users.json.users` 删除目标 key -> 更新 `UpdatedAt/UpdatedBy` -> 更新 Secret -> `transport.WriteSuccess(200)`
- operator 链路：无；用户管理不使用 disaster-operator CRD/controller，不触发 reconcile
- 下层资源链路：更新管理集群 Kubernetes Secret `disaster-system/disaster-server-users` 的 `users.json`；不访问 Velero、StorageRepository、业务集群、对象存储或统计 CRD
- 已写入内容：五段详细说明、JWT header、path 用户名规则、内置 `admin` 禁删、删除响应字段、当前错误分类
- 取证备注：该接口与 `/api/v1/users/:username` 共用同一 handler；删除成功只删除 server 本地用户文档记录，不会注销已签发 token
- 主要错误点：JWT 失败由中间件返回 `401/403` 普通 JSON；path 用户名非法或删除 `admin` 返回 `400 code=1000`；目标不存在返回 `404 code=3004`；Store 未初始化、Secret 初始化/读取/更新或文档解码失败返回 `500 code=5000`

## PATCH /apis/v1/users/:username/password

- RunAPI Target ID：`1c7b4441e6801001`
- RunAPI 状态：缺失，已新增到 `用户管理` 目录，已写入五段详细说明，已补齐 `Authorization` 和 `Content-Type` header，已补齐 path `username` 和当前 `password` JSON body schema，成功响应已回读验证
- server 路由：`internal/apis/user/v1/router.go`；`internal/router/router.go` 在 `/apis` 分组注册 `userv1.NewUserHandler(store, bashPath)`，本条为 `/apis` 主路径
- server handler：`internal/apis/user/v1/handler.go`，`UserHandler.patchUserPassword`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> 读取并 trim path `username` -> `validateUsername` -> `BindJSON(PatchUserPasswordRequest)` -> trim `password` -> `validatePassword` -> `actorFromContext` -> `KubeStore.UpdateUserPassword` -> `EnsureInitialized` -> `HashPassword` bcrypt 哈希 -> `retry.RetryOnConflict` 读取 Secret `disaster-server-users` -> 检查用户存在 -> 更新 `passwordHash/updatedAt` 和文档 `UpdatedAt/UpdatedBy` -> 更新 Secret -> `toUserDTO` -> `transport.WriteSuccess(200)`
- operator 链路：无；用户管理不使用 disaster-operator CRD/controller，不触发 reconcile
- 下层资源链路：更新管理集群 Kubernetes Secret `disaster-system/disaster-server-users` 的 `users.json`；不访问 Velero、StorageRepository、业务集群、对象存储或统计 CRD
- 已写入内容：五段详细说明、JWT 和 JSON header、path 用户名规则、密码长度 6-72 与 bcrypt 上限、响应不返回密码/hash/updatedAt、不会主动吊销已签发 token 的事实、当前错误分类
- 取证备注：该接口与 `/api/v1/users/:username/password` 共用同一 handler；修改密码不会改变 `role/status/createdAt`，新密码仅影响后续登录校验
- 主要错误点：JWT 失败由中间件返回 `401/403` 普通 JSON；path 用户名非法、JSON 绑定失败或密码校验失败返回 `400 code=1000`；目标不存在返回 `404 code=3004`；Store 未初始化、Secret 初始化/读取/更新、bcrypt 哈希或文档解码失败返回 `500 code=5000`

## PATCH /apis/v1/users/:username/status

- RunAPI Target ID：`1c7b4474d2001001`
- RunAPI 状态：缺失，已新增到 `用户管理` 目录，已写入五段详细说明，已补齐 `Authorization` 和 `Content-Type` header，已补齐 path `username` 和当前 `status` JSON body schema，成功响应已回读验证
- server 路由：`internal/apis/user/v1/router.go`；`internal/router/router.go` 在 `/apis` 分组注册 `userv1.NewUserHandler(store, bashPath)`，本条为 `/apis` 主路径
- server handler：`internal/apis/user/v1/handler.go`，`UserHandler.patchUserStatus`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> 读取并 trim path `username` -> `validateUsername` -> `BindJSON(PatchUserStatusRequest)` -> trim/lower `status` -> `validateStatus(active|disabled)` -> `actorFromContext` -> `KubeStore.UpdateUserStatus` -> `EnsureInitialized` -> `retry.RetryOnConflict` 读取 Secret `disaster-server-users` -> 检查用户存在 -> 更新 `status/updatedAt` 和文档 `UpdatedAt/UpdatedBy` -> 更新 Secret -> `toUserDTO` -> `transport.WriteSuccess(200)`
- operator 链路：无；用户管理不使用 disaster-operator CRD/controller，不触发 reconcile
- 下层资源链路：更新管理集群 Kubernetes Secret `disaster-system/disaster-server-users` 的 `users.json`；登录链路 `authenticateLogin` 会读取该状态，`disabled` 用户后续登录返回 `ErrUserDisabled`；不访问 Velero、StorageRepository、业务集群、对象存储或统计 CRD
- 已写入内容：五段详细说明、JWT 和 JSON header、path 用户名规则、`active/disabled` 状态含义、登录链路影响、不会主动吊销已签发 token 的事实、当前错误分类
- 取证备注：该接口与 `/api/v1/users/:username/status` 共用同一 handler；JWT 中间件校验已有 access token 时不重新读取用户 Secret，因此禁用账号不能立即拦截已经签发且未过期的 access token
- 主要错误点：JWT 失败由中间件返回 `401/403` 普通 JSON；path 用户名非法、JSON 绑定失败或 status 非 `active/disabled` 返回 `400 code=1000`；目标不存在返回 `404 code=3004`；Store 未初始化、Secret 初始化/读取/更新或文档解码失败返回 `500 code=5000`

## GET /api/v1/system-settings

- RunAPI Target ID：`1c31b7bf28801000`
- RunAPI 状态：已存在，已更新详细说明，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，已补齐 `Authorization` header，已补齐 `keys/q/keyword/page/limit` query 参数，旧 `data` 数组成功示例已修正为当前 `data.items` collection 结构，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/system_settings/v1/router.go`；`internal/router/router.go` 在 `/api` 分组通过 `RegisterWithoutPublic` 注册管理路由，本条为 `/api` 管理路径
- server handler：`internal/apis/system_settings/v1/handler.go`，`SystemSettingsHandler.listSettings`
- 请求链路：`/api` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> `transport.ParseOptions` -> `getSettingsDocument` -> 读取 ConfigMap `disaster-system/disaster-platform-settings` key `settings` -> NotFound 返回空文档 -> `decodeSettingsDocument` -> `parseKeys(query.keys)` -> `q` 优先、否则使用 `keyword` -> `collectItems` 按 keys 过滤并对 `name/config_key` 包含匹配 -> 固定按 `config_key` 升序 -> `transport.Paginate` -> `transport.BuildCollectionResponse` -> `transport.WriteSuccess(200)`
- operator 链路：无；系统设置不使用 disaster-operator CRD/controller，不触发 reconcile
- 下层资源链路：只读管理集群 Kubernetes ConfigMap `disaster-platform-settings`；不访问 Velero、StorageRepository、业务集群、对象存储或统计 CRD；图片设置值会以 data URL 字符串直接随列表返回
- 已写入内容：五段详细说明、JWT header、`keys/q/keyword/page/limit` 参数、ConfigMap 文档结构、ConfigMap 缺失返回空列表、固定 `config_key` 升序、`sort/order` 不实际生效、collection envelope 字段、当前错误分类
- 取证备注：列表接口只要求已登录，不调用 `requireSystemAdmin`；`meta.links.<config_key>` 会生成配置详情链接，但当前 server 未实现 `GET /system-settings/:config_key` 详情 JSON 接口
- 主要错误点：JWT 失败由中间件返回 `401/403` 普通 JSON；ConfigMap store 未初始化、读取 ConfigMap 失败或 `data.settings` JSON 解码失败返回 `500 code=5000`

## POST /api/v1/system-settings

- RunAPI Target ID：`1c31b7bf87401001`
- RunAPI 状态：已存在，已更新详细说明，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，已补齐 `Authorization` 和 `Content-Type` header，已补齐当前 `name/config_key/value/remark` JSON body schema，成功响应 HTTP 状态码已修正为 `201`，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/system_settings/v1/router.go`；`internal/router/router.go` 在 `/api` 分组通过 `RegisterWithoutPublic` 注册管理路由，本条为 `/api` 管理路径
- server handler：`internal/apis/system_settings/v1/handler.go`，`SystemSettingsHandler.createSetting`
- 请求链路：`/api` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> `requireSystemAdmin` -> `BindJSON(CreateSystemSettingRequest)` -> `normalizeCreateRequest` -> `validateConfigKey/validateName/validateRemark/validateValue` -> `mutateSettingsDocument` -> `loadSettingsConfigMap` 读取或初始化空文档 -> 检查 `config_key` 不存在 -> 写入 `doc.Items[config_key]` -> 设置 `schemaVersion/updatedAt/updatedBy` -> JSON marshal 并校验总大小不超过 900 KiB -> 创建或更新 ConfigMap `disaster-platform-settings` -> `transport.WriteSuccess(201)`
- operator 链路：无；系统设置不使用 disaster-operator CRD/controller，不触发 reconcile
- 下层资源链路：创建或更新管理集群 Kubernetes ConfigMap `disaster-system/disaster-platform-settings` 的 `data.settings`；不访问 Velero、StorageRepository、业务集群、对象存储或统计 CRD
- 已写入内容：五段详细说明、admin 权限要求、JWT 和 JSON header、创建字段约束、ConfigMap 缺失时创建、文档级 `updatedAt/updatedBy` 写入、单值 350 KiB 与总 JSON 900 KiB 限制、当前错误分类
- 取证备注：`value` 和 `remark` 在 Go 结构上没有指针，字段缺失会绑定为空字符串；`value` 允许为空，`remark` 允许为空；只有 `name` 和 `config_key` 必须非空
- 主要错误点：JWT 失败由中间件返回 `401/403` 普通 JSON；非 admin 返回 `403 code=2003`；JSON 绑定或字段校验失败返回 `400 code=1000`；settings 总大小超限返回 `400 code=1000`；配置键已存在或 Kubernetes 冲突返回 `409 code=3009`；ConfigMap store 未初始化、读取/创建/更新 ConfigMap 或解码失败返回 `500 code=5000`

## DELETE /api/v1/system-settings/:config_key

- RunAPI Target ID：`1c31b7ca57001001`
- RunAPI 状态：已存在，已更新详细说明，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，已补齐 `Authorization` header 和 path `config_key` 参数，成功响应已补充 `trace_id`，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/system_settings/v1/router.go`；`internal/router/router.go` 在 `/api` 分组通过 `RegisterWithoutPublic` 注册管理路由，本条为 `/api` 管理路径
- server handler：`internal/apis/system_settings/v1/handler.go`，`SystemSettingsHandler.deleteSetting`
- 请求链路：`/api` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> `requireSystemAdmin` -> 读取并 trim path `config_key` -> `validateConfigKey` -> `mutateSettingsDocument` -> `loadSettingsConfigMap` 读取或初始化空文档 -> 检查配置项存在 -> `delete(doc.Items, configKey)` -> 设置 `schemaVersion/updatedAt/updatedBy` -> JSON marshal 并校验总大小不超过 900 KiB -> 更新 ConfigMap `disaster-platform-settings` -> `transport.WriteSuccess(200)`
- operator 链路：无；系统设置不使用 disaster-operator CRD/controller，不触发 reconcile
- 下层资源链路：更新管理集群 Kubernetes ConfigMap `disaster-system/disaster-platform-settings` 的 `data.settings`；不访问 Velero、StorageRepository、业务集群、对象存储或统计 CRD
- 已写入内容：五段详细说明、admin 权限要求、JWT header、path `config_key` 规则、删除响应字段、ConfigMap 缺失时返回 NotFound 的行为、当前错误分类
- 取证备注：删除成功不返回旧配置值；如果 ConfigMap 不存在，`loadSettingsConfigMap` 会返回空文档，因此最终是 `setting not found`
- 主要错误点：JWT 失败由中间件返回 `401/403` 普通 JSON；非 admin 返回 `403 code=2003`；path 配置键非法返回 `400 code=1000`；目标不存在返回 `404 code=3004`；settings 总大小超限返回 `400 code=1000`；Kubernetes 冲突返回 `409 code=3009`；ConfigMap store 未初始化、读取/更新 ConfigMap 或解码失败返回 `500 code=5000`

## PUT /api/v1/system-settings/:config_key

- RunAPI Target ID：`1c31b7bfa1c01001`
- RunAPI 状态：已存在，已更新详细说明，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，已补齐 `Authorization` 和 `Content-Type` header，已补齐 path `config_key` 和当前局部更新 JSON body schema，成功响应已补充 `trace_id`，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/system_settings/v1/router.go`；`internal/router/router.go` 在 `/api` 分组通过 `RegisterWithoutPublic` 注册管理路由，本条为 `/api` 管理路径
- server handler：`internal/apis/system_settings/v1/handler.go`，`SystemSettingsHandler.updateSetting`
- 请求链路：`/api` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> `requireSystemAdmin` -> 读取并 trim path `config_key` -> `validateConfigKey` -> `BindJSON(UpdateSystemSettingRequest)` -> 校验至少一个字段非 nil -> 对提供的 `name/remark/value` 分别校验 -> `mutateSettingsDocument` -> `loadSettingsConfigMap` 读取或初始化空文档 -> 检查配置项存在 -> 按非 nil 字段覆盖 -> 设置 `schemaVersion/updatedAt/updatedBy` -> JSON marshal 并校验总大小不超过 900 KiB -> 更新 ConfigMap `disaster-platform-settings` -> `transport.WriteSuccess(200)`
- operator 链路：无；系统设置不使用 disaster-operator CRD/controller，不触发 reconcile
- 下层资源链路：更新管理集群 Kubernetes ConfigMap `disaster-system/disaster-platform-settings` 的 `data.settings`；不访问 Velero、StorageRepository、业务集群、对象存储或统计 CRD
- 已写入内容：五段详细说明、admin 权限要求、JWT 和 JSON header、path `config_key` 规则、局部更新字段语义、至少提供一个字段、字段长度限制、当前错误分类
- 取证备注：`name` 和 `remark` 提供时会 trim；`value` 提供时不会 trim；如果旧 item 的 `config_key` 为空，会在响应和保存时补为 path `config_key`
- 主要错误点：JWT 失败由中间件返回 `401/403` 普通 JSON；非 admin 返回 `403 code=2003`；path 配置键非法、JSON 绑定失败、空更新或字段校验失败返回 `400 code=1000`；目标不存在返回 `404 code=3004`；settings 总大小超限返回 `400 code=1000`；Kubernetes 冲突返回 `409 code=3009`；ConfigMap store 未初始化、读取/更新 ConfigMap 或解码失败返回 `500 code=5000`

## POST /api/v1/system-settings/assets/:config_key

- RunAPI Target ID：`1c31b7ca73001000`
- RunAPI 状态：已存在，已更新详细说明，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，已补齐 `Authorization`、multipart `Content-Type` header、path `config_key` 和 form-data `file` 参数，成功响应已补充 `trace_id`，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/system_settings/v1/router.go`；`internal/router/router.go` 在 `/api` 分组通过 `RegisterWithoutPublic` 注册管理路由，本条为 `/api` 管理路径
- server handler：`internal/apis/system_settings/v1/handler.go`，`SystemSettingsHandler.uploadAsset`
- 请求链路：`/api` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> `requireSystemAdmin` -> 读取并 trim path `config_key` -> `validateConfigKey` -> `ctx.FormFile(\"file\")` -> 校验 header size 和实际读取大小不超过 256 KiB -> `resolveContentType` 从上传头或扩展名识别 `image/png/image/jpeg` -> `encodeDataURL` -> 校验 data URL 不超过 350 KiB -> `mutateSettingsDocument` -> 配置不存在时创建默认 item -> 更新 item.Value -> 设置 `schemaVersion/updatedAt/updatedBy` -> JSON marshal 并校验总大小不超过 900 KiB -> 创建或更新 ConfigMap -> `transport.WriteSuccess(200)`
- operator 链路：无；系统设置不使用 disaster-operator CRD/controller，不触发 reconcile
- 下层资源链路：创建或更新管理集群 Kubernetes ConfigMap `disaster-system/disaster-platform-settings` 的 `data.settings`；不访问 Velero、StorageRepository、业务集群、对象存储或统计 CRD
- 已写入内容：五段详细说明、admin 权限要求、JWT 和 multipart header、path `config_key` 规则、form-data `file`、支持图片类型、256 KiB 原始文件限制、350 KiB data URL 限制、不存在配置项自动创建、当前错误分类
- 取证备注：`image/jpg` 会规范化为 `image/jpeg`；当 Content-Type 为空或 `application/octet-stream` 时，server 会尝试通过文件扩展名推断；资产内容直接保存在 ConfigMap，不写对象存储
- 主要错误点：JWT 失败由中间件返回 `401/403` 普通 JSON；非 admin 返回 `403 code=2003`；path 配置键非法、缺少 file、文件过大、类型不支持或 data URL 超限返回 `400 code=1000`；settings 总大小超限返回 `400 code=1000`；Kubernetes 冲突返回 `409 code=3009`；ConfigMap store 未初始化、读取/创建/更新 ConfigMap 或解码失败返回 `500 code=5000`

## GET /api/v1/system-settings/assets/:config_key

- RunAPI Target ID：`1c31b7ca8b001000`
- RunAPI 状态：已存在，已更新详细说明，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，已补齐 `Authorization` header 和 path `config_key` 参数，已补充图片流成功响应示例，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/system_settings/v1/router.go`；`internal/router/router.go` 在 `/api` 分组通过 `RegisterWithoutPublic` 注册管理路由，本条为 `/api` 管理路径
- server handler：`internal/apis/system_settings/v1/handler.go`，`SystemSettingsHandler.getAsset`
- 请求链路：`/api` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> 读取并 trim path `config_key` -> `validateConfigKey` -> `getSettingsDocument` 读取 ConfigMap `disaster-platform-settings` -> 按 `config_key` 查找 item -> `decodeDataURL(item.Value)` -> `ctx.Data(200, contentType, decoded)`
- operator 链路：无；系统设置不使用 disaster-operator CRD/controller，不触发 reconcile
- 下层资源链路：只读管理集群 Kubernetes ConfigMap `disaster-platform-settings` 的 `data.settings`；不访问 Velero、StorageRepository、业务集群、对象存储或统计 CRD
- 已写入内容：五段详细说明、JWT header、path `config_key` 规则、成功响应为二进制流且不是 JSON envelope、data URL 解码规则、当前错误分类
- 取证备注：该接口不调用 `requireSystemAdmin`，已登录用户即可读取；成功 Content-Type 直接来自 data URL 头部，server 不重新限制为 png/jpeg
- 主要错误点：JWT 失败由中间件返回 `401/403` 普通 JSON；path 配置键非法返回 `400 code=1000`；目标配置不存在返回 `404 code=3004`；value 不是合法 data URL 或 Base64 解码失败返回 `400 code=1000`；ConfigMap store 未初始化、读取 ConfigMap 或解码 settings JSON 失败返回 `500 code=5000`

## GET /api/v1/system-settings/public

- RunAPI Target ID：`1c31b7bf6e801000`
- RunAPI 状态：已存在，已更新详细说明，鉴权类型保持 `noauth`，已补齐必填 `keys` 和分页 query 参数，旧 `data` 数组成功示例已修正为当前 `data.items` collection 结构，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/system_settings/v1/router.go`；`internal/router/router.go` 在 `/api` 公开分组通过 `RegisterPublicOnly` 注册，本条为 `/api` 公开路径
- server handler：`internal/apis/system_settings/v1/handler.go`，`SystemSettingsHandler.listPublicSettings`
- 请求链路：`/api` 公开分组 `WebSocketTokenAdapter` -> `TraceMiddleware`，不经过 JWT 中间件 -> `transport.ParseOptions` -> `parseKeys(query.keys)` -> keys 为空返回 400 -> `getSettingsDocument` 读取 ConfigMap `disaster-platform-settings` -> `collectItems` 只按 keys 过滤且 keyword 固定为空 -> 固定按 `config_key` 升序 -> `transport.Paginate` -> `transport.BuildCollectionResponse` -> `transport.WriteSuccess(200)`
- operator 链路：无；系统设置不使用 disaster-operator CRD/controller，不触发 reconcile
- 下层资源链路：只读管理集群 Kubernetes ConfigMap `disaster-platform-settings` 的 `data.settings`；不访问 Velero、StorageRepository、业务集群、对象存储或统计 CRD
- 已写入内容：五段详细说明、公开无鉴权事实、`keys` 必填、分页参数、只返回指定 keys、固定 `config_key` 升序、collection envelope 字段、当前错误分类
- 取证备注：该接口不支持 `q/keyword` 过滤；`meta.links.<config_key>` 会生成 `/api/v1/system-settings/public/<key>` 形式的 item link，但当前 server 未实现公开详情路径
- 主要错误点：缺少 keys 或 keys 解析后为空返回 `400 code=1000`；ConfigMap store 未初始化、读取 ConfigMap 或解码 settings JSON 失败返回 `500 code=5000`

## GET /apis/v1/system-settings

- RunAPI Target ID：`1c7b469bcbc01001`
- RunAPI 状态：缺失，已新增到 `系统设置` 目录，已写入五段详细说明，已补齐 `Authorization` header 和 `keys/q/keyword/page/limit` query 参数，当前 collection 响应示例已回读验证
- server 路由：`internal/apis/system_settings/v1/router.go`；`internal/router/router.go` 在 `/apis` 分组注册 `Register`，本条为 `/apis` 主路径
- server handler：`internal/apis/system_settings/v1/handler.go`，`SystemSettingsHandler.listSettings`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> `transport.ParseOptions` -> `getSettingsDocument` -> 读取 ConfigMap `disaster-system/disaster-platform-settings` key `settings` -> NotFound 返回空文档 -> `decodeSettingsDocument` -> `parseKeys(query.keys)` -> `q` 优先、否则使用 `keyword` -> `collectItems` 按 keys 过滤并对 `name/config_key` 包含匹配 -> 固定按 `config_key` 升序 -> `transport.Paginate` -> `transport.BuildCollectionResponse` -> `transport.WriteSuccess(200)`
- operator 链路：无；系统设置不使用 disaster-operator CRD/controller，不触发 reconcile
- 下层资源链路：只读管理集群 Kubernetes ConfigMap `disaster-platform-settings`；不访问 Velero、StorageRepository、业务集群、对象存储或统计 CRD
- 已写入内容：五段详细说明、JWT header、`keys/q/keyword/page/limit` 参数、ConfigMap 文档结构、ConfigMap 缺失返回空列表、固定 `config_key` 升序、`sort/order` 不实际生效、collection envelope 字段、当前错误分类
- 取证备注：该接口与 `/api/v1/system-settings` 共用同一 handler；列表接口只要求已登录，不调用 `requireSystemAdmin`
- 主要错误点：JWT 失败由中间件返回 `401/403` 普通 JSON；ConfigMap store 未初始化、读取 ConfigMap 失败或 `data.settings` JSON 解码失败返回 `500 code=5000`

## POST /apis/v1/system-settings

- RunAPI Target ID：`1c7b46cc9d801001`
- RunAPI 状态：缺失，已新增到 `系统设置` 目录，已写入五段详细说明，已补齐 `Authorization` 和 `Content-Type` header，已补齐当前 `name/config_key/value/remark` JSON body schema，成功响应 HTTP 状态码已设为 `201`，已回读验证
- server 路由：`internal/apis/system_settings/v1/router.go`；`internal/router/router.go` 在 `/apis` 分组注册 `Register`，本条为 `/apis` 主路径
- server handler：`internal/apis/system_settings/v1/handler.go`，`SystemSettingsHandler.createSetting`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> `requireSystemAdmin` -> `BindJSON(CreateSystemSettingRequest)` -> `normalizeCreateRequest` -> `validateConfigKey/validateName/validateRemark/validateValue` -> `mutateSettingsDocument` -> `loadSettingsConfigMap` 读取或初始化空文档 -> 检查 `config_key` 不存在 -> 写入 `doc.Items[config_key]` -> 设置 `schemaVersion/updatedAt/updatedBy` -> JSON marshal 并校验总大小不超过 900 KiB -> 创建或更新 ConfigMap `disaster-platform-settings` -> `transport.WriteSuccess(201)`
- operator 链路：无；系统设置不使用 disaster-operator CRD/controller，不触发 reconcile
- 下层资源链路：创建或更新管理集群 Kubernetes ConfigMap `disaster-system/disaster-platform-settings` 的 `data.settings`；不访问 Velero、StorageRepository、业务集群、对象存储或统计 CRD
- 已写入内容：五段详细说明、admin 权限要求、JWT 和 JSON header、创建字段约束、ConfigMap 缺失时创建、文档级 `updatedAt/updatedBy` 写入、单值 350 KiB 与总 JSON 900 KiB 限制、当前错误分类
- 取证备注：该接口与 `/api/v1/system-settings` 共用同一 handler；`value` 和 `remark` 字段缺失会绑定为空字符串，只有 `name` 和 `config_key` 必须非空
- 主要错误点：JWT 失败由中间件返回 `401/403` 普通 JSON；非 admin 返回 `403 code=2003`；JSON 绑定或字段校验失败返回 `400 code=1000`；settings 总大小超限返回 `400 code=1000`；配置键已存在或 Kubernetes 冲突返回 `409 code=3009`；ConfigMap store 未初始化、读取/创建/更新 ConfigMap 或解码失败返回 `500 code=5000`

## DELETE /apis/v1/system-settings/:config_key

- RunAPI Target ID：`1c7b470b1b001000`
- RunAPI 状态：缺失，已新增到 `系统设置` 目录，已写入五段详细说明，已补齐 `Authorization` header 和 path `config_key` 参数，成功响应已回读验证
- server 路由：`internal/apis/system_settings/v1/router.go`；`internal/router/router.go` 在 `/apis` 分组注册 `Register`，本条为 `/apis` 主路径
- server handler：`internal/apis/system_settings/v1/handler.go`，`SystemSettingsHandler.deleteSetting`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> `requireSystemAdmin` -> 读取并 trim path `config_key` -> `validateConfigKey` -> `mutateSettingsDocument` -> `loadSettingsConfigMap` 读取或初始化空文档 -> 检查配置项存在 -> 删除 `doc.Items[configKey]` -> 设置 `schemaVersion/updatedAt/updatedBy` -> JSON marshal 并校验总大小不超过 900 KiB -> 更新 ConfigMap `disaster-platform-settings` -> `transport.WriteSuccess(200)`
- operator 链路：无；系统设置不使用 disaster-operator CRD/controller，不触发 reconcile
- 下层资源链路：更新管理集群 Kubernetes ConfigMap `disaster-system/disaster-platform-settings` 的 `data.settings`；不访问 Velero、StorageRepository、业务集群、对象存储或统计 CRD
- 已写入内容：五段详细说明、admin 权限要求、JWT header、path `config_key` 规则、删除响应字段、ConfigMap 缺失时返回 NotFound 的行为、当前错误分类
- 取证备注：该接口与 `/api/v1/system-settings/:config_key` 共用同一 handler；删除成功不返回旧配置值
- 主要错误点：JWT 失败由中间件返回 `401/403` 普通 JSON；非 admin 返回 `403 code=2003`；path 配置键非法返回 `400 code=1000`；目标不存在返回 `404 code=3004`；settings 总大小超限返回 `400 code=1000`；Kubernetes 冲突返回 `409 code=3009`；ConfigMap store 未初始化、读取/更新 ConfigMap 或解码失败返回 `500 code=5000`

## PUT /apis/v1/system-settings/:config_key

- RunAPI Target ID：`1c7b474554801001`
- RunAPI 状态：缺失，已新增到 `系统设置` 目录，已写入五段详细说明，已补齐 `Authorization` 和 `Content-Type` header，已补齐 path `config_key` 和当前局部更新 JSON body schema，成功响应已回读验证
- server 路由：`internal/apis/system_settings/v1/router.go`；`internal/router/router.go` 在 `/apis` 分组注册 `Register`，本条为 `/apis` 主路径
- server handler：`internal/apis/system_settings/v1/handler.go`，`SystemSettingsHandler.updateSetting`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> `requireSystemAdmin` -> 读取并 trim path `config_key` -> `validateConfigKey` -> `BindJSON(UpdateSystemSettingRequest)` -> 校验至少一个字段非 nil -> 对提供的 `name/remark/value` 分别校验 -> `mutateSettingsDocument` -> `loadSettingsConfigMap` 读取或初始化空文档 -> 检查配置项存在 -> 按非 nil 字段覆盖 -> 设置 `schemaVersion/updatedAt/updatedBy` -> JSON marshal 并校验总大小不超过 900 KiB -> 更新 ConfigMap `disaster-platform-settings` -> `transport.WriteSuccess(200)`
- operator 链路：无；系统设置不使用 disaster-operator CRD/controller，不触发 reconcile
- 下层资源链路：更新管理集群 Kubernetes ConfigMap `disaster-system/disaster-platform-settings` 的 `data.settings`；不访问 Velero、StorageRepository、业务集群、对象存储或统计 CRD
- 已写入内容：五段详细说明、admin 权限要求、JWT 和 JSON header、path `config_key` 规则、局部更新字段语义、至少提供一个字段、字段长度限制、当前错误分类
- 取证备注：该接口与 `/api/v1/system-settings/:config_key` 共用同一 handler；`name` 和 `remark` 提供时会 trim，`value` 提供时不会 trim
- 主要错误点：JWT 失败由中间件返回 `401/403` 普通 JSON；非 admin 返回 `403 code=2003`；path 配置键非法、JSON 绑定失败、空更新或字段校验失败返回 `400 code=1000`；目标不存在返回 `404 code=3004`；settings 总大小超限返回 `400 code=1000`；Kubernetes 冲突返回 `409 code=3009`；ConfigMap store 未初始化、读取/更新 ConfigMap 或解码失败返回 `500 code=5000`

## POST /apis/v1/system-settings/assets/:config_key

- RunAPI Target ID：`1c7b4783d4401001`
- RunAPI 状态：缺失，已新增到 `系统设置` 目录，已写入五段详细说明，已补齐 `Authorization`、multipart `Content-Type` header、path `config_key` 和 form-data `file` 参数，成功响应已回读验证
- server 路由：`internal/apis/system_settings/v1/router.go`；`internal/router/router.go` 在 `/apis` 分组注册 `Register`，本条为 `/apis` 主路径
- server handler：`internal/apis/system_settings/v1/handler.go`，`SystemSettingsHandler.uploadAsset`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> `requireSystemAdmin` -> 读取并 trim path `config_key` -> `validateConfigKey` -> `ctx.FormFile(\"file\")` -> 校验 header size 和实际读取大小不超过 256 KiB -> `resolveContentType` -> `encodeDataURL` -> 校验 data URL 不超过 350 KiB -> `mutateSettingsDocument` -> 配置不存在时创建默认 item -> 更新 item.Value -> 设置 `schemaVersion/updatedAt/updatedBy` -> JSON marshal 并校验总大小不超过 900 KiB -> 创建或更新 ConfigMap -> `transport.WriteSuccess(200)`
- operator 链路：无；系统设置不使用 disaster-operator CRD/controller，不触发 reconcile
- 下层资源链路：创建或更新管理集群 Kubernetes ConfigMap `disaster-system/disaster-platform-settings` 的 `data.settings`；不访问 Velero、StorageRepository、业务集群、对象存储或统计 CRD
- 已写入内容：五段详细说明、admin 权限要求、JWT 和 multipart header、path `config_key` 规则、form-data `file`、支持图片类型、256 KiB 原始文件限制、350 KiB data URL 限制、不存在配置项自动创建、当前错误分类
- 取证备注：该接口与 `/api/v1/system-settings/assets/:config_key` 共用同一 handler；资产内容直接保存在 ConfigMap，不写对象存储
- 主要错误点：JWT 失败由中间件返回 `401/403` 普通 JSON；非 admin 返回 `403 code=2003`；path 配置键非法、缺少 file、文件过大、类型不支持或 data URL 超限返回 `400 code=1000`；settings 总大小超限返回 `400 code=1000`；Kubernetes 冲突返回 `409 code=3009`；ConfigMap store 未初始化、读取/创建/更新 ConfigMap 或解码失败返回 `500 code=5000`

## GET /apis/v1/system-settings/assets/:config_key

- RunAPI Target ID：`1c7b47c5ea801001`
- RunAPI 状态：缺失，已新增到 `系统设置` 目录，已写入五段详细说明，已补齐 `Authorization` header 和 path `config_key` 参数，已补充图片流成功响应示例，已回读验证
- server 路由：`internal/apis/system_settings/v1/router.go`；`internal/router/router.go` 在 `/apis` 分组注册 `Register`，本条为 `/apis` 主路径
- server handler：`internal/apis/system_settings/v1/handler.go`，`SystemSettingsHandler.getAsset`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> 读取并 trim path `config_key` -> `validateConfigKey` -> `getSettingsDocument` 读取 ConfigMap `disaster-platform-settings` -> 按 `config_key` 查找 item -> `decodeDataURL(item.Value)` -> `ctx.Data(200, contentType, decoded)`
- operator 链路：无；系统设置不使用 disaster-operator CRD/controller，不触发 reconcile
- 下层资源链路：只读管理集群 Kubernetes ConfigMap `disaster-platform-settings` 的 `data.settings`；不访问 Velero、StorageRepository、业务集群、对象存储或统计 CRD
- 已写入内容：五段详细说明、JWT header、path `config_key` 规则、成功响应为二进制流且不是 JSON envelope、data URL 解码规则、当前错误分类
- 取证备注：该接口与 `/api/v1/system-settings/assets/:config_key` 共用同一 handler；不调用 `requireSystemAdmin`，已登录用户即可读取；成功 Content-Type 直接来自 data URL 头部
- 主要错误点：JWT 失败由中间件返回 `401/403` 普通 JSON；path 配置键非法返回 `400 code=1000`；目标配置不存在返回 `404 code=3004`；value 不是合法 data URL 或 Base64 解码失败返回 `400 code=1000`；ConfigMap store 未初始化、读取 ConfigMap 或解码 settings JSON 失败返回 `500 code=5000`

## GET /apis/v1/system-settings/public

- RunAPI Target ID：`1c7b4869a4801001`
- RunAPI 状态：缺失，已新增到 `系统设置` 目录，已写入五段详细说明，已补齐 `Authorization` header 和 `keys/page/limit/sort/order` query 参数，鉴权类型已设为继承项目鉴权，当前 collection 响应示例已回读验证
- server 路由：`internal/apis/system_settings/v1/router.go`；`internal/router/router.go` 在 `/apis` 分组注册 `Register`，本条为 `/apis` 主路径
- server handler：`internal/apis/system_settings/v1/handler.go`，`SystemSettingsHandler.listPublicSettings`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> `transport.ParseOptions` -> `parseKeys(query.keys)` -> keys 为空返回 400 -> `getSettingsDocument` 读取 ConfigMap `disaster-platform-settings` -> `collectItems` 只按 keys 过滤且 keyword 固定为空 -> 固定按 `config_key` 升序 -> `transport.Paginate` -> `transport.BuildCollectionResponse` -> `transport.WriteSuccess(200)`
- operator 链路：无；系统设置不使用 disaster-operator CRD/controller，不触发 reconcile
- 下层资源链路：只读管理集群 Kubernetes ConfigMap `disaster-platform-settings` 的 `data.settings`；不访问 Velero、StorageRepository、业务集群、对象存储或统计 CRD
- 已写入内容：五段详细说明、`/apis` JWT 鉴权要求、public 语义不等于无鉴权、`keys` 必填、分页参数、`sort/order` 仅回显不影响实际排序、collection envelope 字段、当前错误分类
- 取证备注：该接口与 `/api/v1/system-settings/public` 共用同一 handler；handler 不调用 `requireSystemAdmin`，所以已登录用户即可读取；`meta.links.<config_key>` 会生成 `/apis/v1/system-settings/public/<key>` 形式的 item link，但当前 server 未实现该详情路径
- 主要错误点：JWT 失败由中间件返回 `401/403` 普通 JSON；缺少 keys 或 keys 解析后为空返回 `400 code=1000`；ConfigMap store 未初始化、读取 ConfigMap 或解码 settings JSON 失败返回 `500 code=5000`

## GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/autobackups/execution-summary

- RunAPI Target ID：`1c7215aa3e401001`
- RunAPI 状态：已存在，已更新详细说明，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，已补齐必填 `Authorization` header，URL 已去掉固定 `?period=7d`，保留 `period/namespace/range` query 参数和原有成功/非法 period 响应示例，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/statistics/v1/router.go`，`/apis/backuprestorestatistics.testudo.softcdata.com/v1/autobackups/execution-summary`
- server handler：`internal/apis/statistics/v1/handler.go`，`StatisticsHandler.GetAutoBackupExecutionSummary`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> 读取 `namespace/period/range` -> `period` 为空时回退读取 `range` -> 默认 `7d` -> `parseAutoBackupSummaryPeriod(7d|30d|90d)` -> 按 namespace 或全部 namespace 列出 `AppBackup` -> `isAutoBackupPolicyAppBackup` 读取同 namespace `DisasterPolicy` 并要求 `spec.type=AutoBackup` -> 遍历 `AppBackup.status.history` -> 以 `completionTimestamp` 优先、否则 `startTimestamp` 过滤时间窗口 -> 根据 `managedStatus` 或原始 `phase` 计入 success/failed -> 计算 total 与四舍五入百分比 -> `transport.WriteSuccess(200)`
- operator 链路：只读 operator 维护的 `AppBackup` 和 `DisasterPolicy` CR；`AppBackupReconciler` 创建/跟踪 Velero Backup 并把执行历史写入 `AppBackup.status.history`，本接口不触发 reconcile，不创建或更新 `BackupRestoreStatistics`
- 下层资源链路：读取管理集群/本地 informer cache 中的 `AppBackup` 与 `DisasterPolicy`；不访问 Velero API、StorageRepository、对象存储或业务集群
- 已写入内容：五段详细说明、JWT header、`period/range/namespace` 参数、自动备份筛选口径、history 时间字段优先级、成功/失败判定、`total=0` 百分比为 0、当前错误分类
- 取证备注：该接口统计自动备份策略执行情况，不等同于 `/backups/success-rate`；`DisasterPolicy` 读取失败会被 `isAutoBackupPolicyAppBackup` 当作非自动备份跳过，不直接返回错误
- 主要错误点：JWT 失败由中间件返回 `401/403`；非法 `period/range` 返回 `400 code=1000`；`AppBackup` lister 列表失败返回 `500 code=5000`

## GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/backups

- RunAPI Target ID：`1a7322a38c001`
- RunAPI 状态：已存在，已写入五段详细说明，已补齐必填 `Authorization` header 和 `namespace/origin` query 参数，已替换空响应示例为当前成功示例和非法 origin 示例，已回读验证
- server 路由：`internal/apis/statistics/v1/router.go`，`/apis/backuprestorestatistics.testudo.softcdata.com/v1/backups`
- server handler：`internal/apis/statistics/v1/handler.go`，`StatisticsHandler.GetBackupStatistics` -> `getStatistics(ownerKind=AppBackup)`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> 读取 `namespace/origin` -> `parseAppResourceOriginFilter(user|instance|all)`，默认 `user` -> 用 label selector `testudo.softcdata.com/owner-kind=AppBackup` 按 namespace 或全部 namespace 列出 `BackupRestoreStatistics` -> 非 `all` 时通过 `scopeRef` 读取对应 `AppBackup` 并推断来源 -> 聚合 `status.statistics.total/inProgress/completed/failed/canceled/unknown` -> `transport.WriteSuccess(200)`
- operator 链路：`AppBackupReconciler.syncStatistics` 根据 `AppBackup.status.history` 生成 `BackupRestoreStatistics.status.statistics` 并写入 `owner-kind=AppBackup` 标签；本接口只读统计 CR，不触发 reconcile
- 下层资源链路：读取管理集群/本地 informer cache 中的 `BackupRestoreStatistics` 与用于来源识别的 `AppBackup`；不直接访问 Velero Backup、StorageRepository、对象存储或业务集群
- 已写入内容：五段详细说明、JWT header、`namespace/origin` 参数、来源过滤默认值与可选值、统计字段来源、operator 写入统计快照链路、当前错误分类
- 取证备注：RunAPI 原说明为空，因此未追加 `## 原有说明`；`origin=user` 是默认行为，若调用方想看容灾实例链路生成的备份统计必须传 `origin=instance` 或 `origin=all`
- 主要错误点：JWT 失败由中间件返回 `401/403`；非法 `origin` 返回 `400 code=1000`；`BackupRestoreStatistics` lister 列表失败返回 `500 code=5000`

## GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/backups/success-rate

- RunAPI Target ID：`1c239ca7d4801000`
- RunAPI 状态：已存在，已更新详细说明，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，已补齐必填 `Authorization` header 和 `namespace/period/startTime/endTime` query 参数，URL 已去掉固定 `?period=all`，已修正历史 `code/msg` 响应示例为当前 `code/message/trace_id` envelope，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/statistics/v1/router.go`，`/apis/backuprestorestatistics.testudo.softcdata.com/v1/backups/success-rate`
- server handler：`internal/apis/statistics/v1/handler.go`，`StatisticsHandler.GetBackupSuccessRate`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> 读取 `namespace/period/startTime/endTime` -> `parseTimeRange` 计算时间窗口 -> 用 label selector `testudo.softcdata.com/owner-kind=AppBackup` 按 namespace 或全部 namespace 列出 `BackupRestoreStatistics` -> 按 `metadata.creationTimestamp` 过滤时间窗口 -> 聚合 `status.statistics.completed/failed` -> `successRate=completed/(completed+failed)*100`，分母为 0 时返回 100 -> `transport.WriteSuccess(200)`
- operator 链路：`AppBackupReconciler.syncStatistics` 根据 `AppBackup.status.history` 同步 `BackupRestoreStatistics.status.statistics`；本接口只读统计 CR，不触发 reconcile
- 下层资源链路：读取管理集群/本地 informer cache 中的 `BackupRestoreStatistics`；不直接读取 `AppBackup.status.history`、Velero Backup、StorageRepository、对象存储或业务集群
- 已写入内容：五段详细说明、JWT header、`namespace/period/startTime/endTime` 参数、RFC3339 校验、按统计 CR 创建时间过滤的口径、成功率分母只含 completed/failed、空分母返回 100、当前错误分类
- 取证备注：当前 `parseTimeRange` 对非 `today/week/month/all` 的 period 不返回错误，只会回显该值且不生成快捷时间窗口；只有 `startTime/endTime` 格式非法会返回 400
- 主要错误点：JWT 失败由中间件返回 `401/403`；非法 `startTime/endTime` 返回 `400 code=1000`；`BackupRestoreStatistics` lister 列表失败返回 `500 code=5000`

## GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/instances

- RunAPI Target ID：`1c239ac3a6001000`
- RunAPI 状态：已存在，已更新详细说明，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，已补齐必填 `Authorization` header 和 `namespace` query 参数，已修正历史 `code/msg` 响应示例为当前 `code/message/trace_id` envelope，并补充当前 DTO 实际返回的 `degraded` 字段，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/statistics/v1/router.go`，`/apis/backuprestorestatistics.testudo.softcdata.com/v1/instances`
- server handler：`internal/apis/statistics/v1/handler.go`，`StatisticsHandler.GetInstanceStatistics`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> 读取 `namespace` -> 按 namespace 或全部 namespace 列出 `DisasterInstance` -> 聚合 `status.fsmState` 到 `protected/paused/failingOver/active/failingBack/initializing/pending/failed` -> `transport.WriteSuccess(200)`
- operator 链路：只读 `DisasterInstance` CR；`DisasterInstance` controller 根据数据同步、资源同步、故障切换和回切链路维护 `status.fsmState`，本接口不触发 reconcile
- 下层资源链路：读取管理集群/本地 informer cache 中的 `DisasterInstance`；不访问 `BackupRestoreStatistics`、Velero、StorageRepository、对象存储或业务集群
- 已写入内容：五段详细说明、JWT header、`namespace` 参数、各 `fsmState` 到返回字段的映射、`degraded` 预留字段、当前错误分类
- 取证备注：当前 handler 未单独处理 `ConfigError`，也不会递增 `degraded`；`ConfigError` 和其他未识别状态会落入 `pending`
- 主要错误点：JWT 失败由中间件返回 `401/403`；`DisasterInstance` lister 列表失败返回 `500 code=5000`

## GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/operations

- RunAPI Target ID：`1c1fe2a799801001`
- RunAPI 状态：已存在，已更新详细说明，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，已补齐必填 `Authorization` header 和 `namespace` query 参数，已修正历史裸对象/`code/msg` 响应示例为当前 `code/message/trace_id` envelope，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/statistics/v1/router.go`，`/apis/backuprestorestatistics.testudo.softcdata.com/v1/operations`
- server handler：`internal/apis/statistics/v1/handler.go`，`StatisticsHandler.GetOperationStatistics` -> `getStatistics(ownerKind=DisasterOperation)`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> 读取 `namespace` -> 用 label selector `testudo.softcdata.com/owner-kind=DisasterOperation` 按 namespace 或全部 namespace 列出 `BackupRestoreStatistics` -> 聚合 `status.statistics.total/inProgress/completed/failed/canceled/unknown` -> `transport.WriteSuccess(200)`
- operator 链路：`DisasterOperationReconciler.syncStatistics` 为每个操作维护 `op-<operation.Name>-stats`，写入 `owner-kind=DisasterOperation`、`operation-type`、`scope-uid` 标签，并根据 `operation.status.state` 写入 `total/completed/failed/inProgress`
- 下层资源链路：读取管理集群/本地 informer cache 中的 `BackupRestoreStatistics`；不直接访问 `DisasterOperation`、Velero、AppBackup、AppRestore、StorageRepository、对象存储或业务集群
- 已写入内容：五段详细说明、JWT header、`namespace` 参数、统计 CR label selector、operator 写入统计快照链路、返回字段来源、当前错误分类
- 取证备注：当前 operator 同步逻辑对单个 `DisasterOperation` 的 `total` 通常写为 1，`canceled/unknown` 不主动写入，接口仍按通用 StatisticsDTO 返回这些字段
- 主要错误点：JWT 失败由中间件返回 `401/403`；`BackupRestoreStatistics` lister 列表失败返回 `500 code=5000`

## GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/operations/by-time

- RunAPI Target ID：`1c2398ad1d001000`
- RunAPI 状态：已存在，已更新详细说明，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，已补齐必填 `Authorization` header 和 `namespace/period/startTime/endTime` query 参数，URL 已去掉固定 `?period=month`，已修正历史 `code/msg` 响应示例为当前 `code/message/trace_id` envelope 并补充 `period/startTime/endTime` 响应字段，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/statistics/v1/router.go`，`/apis/backuprestorestatistics.testudo.softcdata.com/v1/operations/by-time`
- server handler：`internal/apis/statistics/v1/handler.go`，`StatisticsHandler.GetOperationStatisticsByTime`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> 读取 `namespace/period/startTime/endTime` -> `parseTimeRange` 计算时间窗口 -> 用 label selector `testudo.softcdata.com/owner-kind=DisasterOperation` 按 namespace 或全部 namespace 列出 `BackupRestoreStatistics` -> 按 `metadata.creationTimestamp` 过滤时间窗口 -> 聚合 `status.statistics.total/inProgress/completed/failed/canceled/unknown` -> 写入 `period` 和非零 `startTime/endTime` -> `transport.WriteSuccess(200)`
- operator 链路：`DisasterOperationReconciler.syncStatistics` 为每个操作维护 `op-<operation.Name>-stats` 统计 CR；本接口只读统计 CR，不触发 reconcile
- 下层资源链路：读取管理集群/本地 informer cache 中的 `BackupRestoreStatistics`；不直接访问 `DisasterOperation`、Velero、AppBackup、AppRestore、StorageRepository、对象存储或业务集群
- 已写入内容：五段详细说明、JWT header、`namespace/period/startTime/endTime` 参数、RFC3339 校验、按统计 CR 创建时间过滤的口径、返回字段来源、当前错误分类
- 取证备注：当前 `parseTimeRange` 对非 `today/week/month/all` 的 period 不返回错误，只会回显该值且不生成快捷时间窗口；只有 `startTime/endTime` 格式非法会返回 400
- 主要错误点：JWT 失败由中间件返回 `401/403`；非法 `startTime/endTime` 返回 `400 code=1000`；`BackupRestoreStatistics` lister 列表失败返回 `500 code=5000`

## GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/restores

- RunAPI Target ID：`1a7322a38c003`
- RunAPI 状态：已存在，已写入五段详细说明，已补齐必填 `Authorization` header 和 `namespace/origin` query 参数，已替换空响应示例为当前成功示例和非法 origin 示例，已回读验证
- server 路由：`internal/apis/statistics/v1/router.go`，`/apis/backuprestorestatistics.testudo.softcdata.com/v1/restores`
- server handler：`internal/apis/statistics/v1/handler.go`，`StatisticsHandler.GetRestoreStatistics` -> `getStatistics(ownerKind=AppRestore)`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> 读取 `namespace/origin` -> `parseAppResourceOriginFilter(user|instance|all)`，默认 `user` -> 用 label selector `testudo.softcdata.com/owner-kind=AppRestore` 按 namespace 或全部 namespace 列出 `BackupRestoreStatistics` -> 非 `all` 时通过 `scopeRef` 读取对应 `AppRestore` 并推断来源 -> 聚合 `status.statistics.total/inProgress/completed/failed/canceled/unknown` -> `transport.WriteSuccess(200)`
- operator 链路：`AppRestoreReconciler.syncStatistics` 根据 `AppRestore.status.status` 生成 `BackupRestoreStatistics.status.statistics` 并写入 `owner-kind=AppRestore` 标签；本接口只读统计 CR，不触发 reconcile
- 下层资源链路：读取管理集群/本地 informer cache 中的 `BackupRestoreStatistics` 与用于来源识别的 `AppRestore`；不直接访问 Velero Restore、StorageRepository、对象存储或业务集群
- 已写入内容：五段详细说明、JWT header、`namespace/origin` 参数、来源过滤默认值与可选值、统计字段来源、operator 写入统计快照链路、当前错误分类
- 取证备注：RunAPI 原说明为空，因此未追加 `## 原有说明`；演练标记、`ddr-`/`drr-`/`rec-ds-`/`rec-rs-` 前缀和 DataSync/ResourceSync OwnerReference 会被识别为 `instance` 来源
- 主要错误点：JWT 失败由中间件返回 `401/403`；非法 `origin` 返回 `400 code=1000`；`BackupRestoreStatistics` lister 列表失败返回 `500 code=5000`

## GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/storage

- RunAPI Target ID：`1c2620b6d7801001`
- RunAPI 状态：已存在，已更新详细说明，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，已补齐必填 `Authorization` header，URL 已去掉固定 `?storageName=s3-709`，保留并完善 `storageName` query 参数，已修正历史 `code=200/message=Success` 响应示例为当前 `code=0/message=OK/trace_id` envelope，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/statistics/v1/router.go`，`/apis/backuprestorestatistics.testudo.softcdata.com/v1/storage`
- server handler：`internal/apis/statistics/v1/handler.go`，`StatisticsHandler.GetStorageStatistics`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> 读取 `storageName` -> 列出 `StorageRepository`，若传入名称则精确匹配单个仓库 -> 聚合 `status.totalBackupCount/status.usedSpaceBytes/spec.quotaBytes` -> 计算 `availableSpaceBytes` 与 `usageRate` -> `transport.WriteSuccess(200)`
- operator 链路：`StorageRepositoryReconciler` 校验 S3 配置并扫描 bucket/prefix 后写入 `StorageRepository.status.usedSpaceBytes/totalBackupCount`；本接口只读状态，不触发 reconcile
- 下层资源链路：读取管理集群/本地 informer cache 中的 `StorageRepository`；接口本身不访问 S3/对象存储、Velero、`BackupRestoreStatistics` 或业务集群
- 已写入内容：五段详细说明、JWT header、`storageName` 参数、名称不存在返回全 0 的当前行为、容量字段来源、配额为 0 的计算规则、当前错误分类
- 取证备注：`availableSpaceBytes` 小于 0 时会被压到 0，但 `usageRate=usedSpaceBytes/quotaBytes` 可能大于 1；`quotaBytes=0` 时 `availableSpaceBytes=0` 且 `usageRate=0`
- 主要错误点：JWT 失败由中间件返回 `401/403`；`StorageRepository` lister 列表失败返回 `500 code=5000`

## GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/tasks/progress

- RunAPI Target ID：`1c71f52b8b401001`
- RunAPI 状态：已存在，已更新详细说明，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，已补齐必填 `Authorization` header，URL 已去掉固定 query，`type` query 已标记必填，保留原有成功/恢复/参数非法响应示例，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/statistics/v1/router.go`，`/apis/backuprestorestatistics.testudo.softcdata.com/v1/tasks/progress`
- server handler：`internal/apis/statistics/v1/handler.go`，`StatisticsHandler.GetTaskProgressTrend`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> `parseTaskProgressQuery(type/scope/range/timezone/namespace/cluster)` -> 以指定时区计算连续自然日窗口 -> `type=backup` 时读取 AppBackup 并遍历 `status.history`，按 `managedStatus` 或 `phase` 映射状态并按执行时间分桶 -> `type=restore` 时读取 AppRestore，按 `status.status` 映射状态并按 completion/start/creation 时间分桶 -> 按来源 scope 聚合 summary/buckets/sources/series -> `transport.WriteSuccess(200, meta.resourceType=backupRestoreTaskProgress)`
- operator 链路：只读 AppBackup/AppRestore controllers 写入的 CR 状态；本接口不读取 `BackupRestoreStatistics`，不触发 reconcile，不创建或更新下层资源
- 下层资源链路：读取管理集群/本地 informer cache 中的 `AppBackup` 或 `AppRestore`；不直接访问 Velero、StorageRepository、对象存储或业务集群
- 已写入内容：五段详细说明、JWT header、`type/scope/range/timezone/namespace/cluster` 参数、状态映射、日期分桶、来源拆分、series 文案、meta.filters、当前错误分类
- 取证备注：`sources[].total` 会统计全部状态，但 `sources[].completed/failed` 只统计成功和失败；进行中、取消、未知只进入 `summary` 和 `buckets`
- 主要错误点：JWT 失败由中间件返回 `401/403`；缺少或非法 `type`、非法 `scope/range/timezone` 返回 `400 code=1000`；AppBackup/AppRestore lister 列表失败返回 `500 code=5000`

## GET /apis/cluster.testudo.softcdata.com/v1/clusters

- RunAPI Target ID：`25559f1f78c067`
- RunAPI 状态：已存在，已更新详细说明，URL 已去掉固定 query 和尾部空格，鉴权类型保持继承项目鉴权，已补齐必填 `Authorization` header 并删除历史空 header，已将分页/排序/keyword/tag/label filter query 标为非必填，原说明已保留到 `## 原有说明`，已回读验证；2026-06-01 追加 `veleroInstall.username` 回显成功响应示例
- server 路由：`internal/apis/disaster_cluster/v1/router.go`，`/apis/cluster.testudo.softcdata.com/v1/clusters`
- server handler：`internal/apis/disaster_cluster/v1/handler.go`，`ClusterHandler.clusters`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> `transport.ParseOptions` -> `tag` 映射为 `testudo.softcdata.com/cluster-tag` -> `BuildLabelSelector` 返回空 selector -> `ClusterLister.List` 读取全部 Cluster 缓存 -> 对所有 filters 执行 label 包含匹配 -> `keyword` 对 `metadata.name` 和 cluster tag 包含匹配 -> `transport.Sort` 支持 `name/creationTimestamp` -> `transport.Paginate` -> `convertToDisasterClusterDTO` -> 若引用 server 管理的 Velero registry Secret 则解析 username -> `transport.BuildCollectionResponse` -> `transport.WriteSuccess(200)`
- operator 链路：只读 cluster-scoped `Cluster` CR；`ClusterReconciler` 根据 kubeconfig/token 连接目标集群，更新 `status.status/endpoint/k8sVersion/veleroVersion/nodeCount/namespaceStats/workloadNamespaceStats/tokenExpiration` 等状态；本接口不触发 reconcile
- 下层资源链路：接口本身不访问目标 Kubernetes 集群、Velero 或对象存储；状态和统计来自 operator 已写入的 `Cluster.status` 与 labels；当 Cluster 引用 server 管理的 Velero registry Secret 时，会只读管理面 `disaster-system/cluster-velero-regcred-<clusterName>` Secret 以解析 username
- 已写入内容：五段详细说明、JWT header、分页/排序/keyword/tag/label filter 参数、DTO 脱敏字段、imageSources、veleroInstall 脱敏回显、namespace/workload 统计口径、collection envelope、当前错误分类
- 取证备注：列表接口不返回 `spec.token`、`spec.kubeConfig`、registry password 或 dockerconfigjson；`veleroInstall.username` 仅在管理面 dockerconfigjson Secret 可解析时回显
- 主要错误点：JWT 失败由中间件返回 `401/403`；`ClusterLister.List` 失败时当前 handler 使用 `400 code=1000`

## POST /apis/cluster.testudo.softcdata.com/v1/clusters

- RunAPI Target ID：`3ee6850538c060`（添加集群 kubeconfig 添加）与 `69ee1a0f8c0b7`（添加集群 token 添加）
- RunAPI 状态：已存在两个同 URL/同 handler 的请求样例，已分别更新详细说明，鉴权类型保持继承项目鉴权，已补齐必填 `Authorization` 与 `Content-Type` header，已按 kubeconfig/token 两种样例修正 JSON body、schema required、字段中文说明和 `201/400/409` 响应示例，原说明已分别保留到 `## 原有说明`，已回读验证；2026-06-01 为 kubeconfig/token 两个样例追加 `veleroInstall.username` 回显成功响应示例
- server 路由：`internal/apis/disaster_cluster/v1/router.go`，`POST /apis/cluster.testudo.softcdata.com/v1/clusters`
- server handler：`internal/apis/disaster_cluster/v1/handler.go`，`ClusterHandler.createCluster`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> `BindJSON(CreateDisasterClusterRequest)` -> `validateVeleroInstallWriteRequest` -> `normalizeClusterImageSources` -> `resolveCreateClusterEffectiveEndpoint` -> `ClusterLister.List(labels.Everything())` -> `findClusterEndpointConflict` -> 组装 `Cluster` CR labels/annotations/spec -> 可选创建管理面 Velero registry dockerconfigjson Secret -> 写入 trace/user annotations -> `DisasterClient.DisasterV1().Clusters().Create` -> `convertToDisasterClusterDTO` -> 若刚创建了 server 管理的 Velero registry Secret 则解析 username -> `transport.WriteSuccess(201)`
- operator 链路：`ClusterReconciler` watch 新建 `Cluster` 后添加 finalizer、发射创建开始事件、优先使用 `spec.kubeConfig` 或使用 `spec.token/spec.endpoint` 构建目标集群客户端、检查 token 过期、检查或安装 Velero、同步 registry pull secret 到目标集群、检查 Kubernetes/Velero 版本和 CRD 兼容性、统计节点/命名空间/资源数量并更新 `Cluster.status` 与统计 labels
- 下层资源链路：server 同步写管理集群 `Cluster` CR 和可选 `disaster-system/cluster-velero-regcred-<clusterName>` Secret；operator 后续访问目标 Kubernetes 集群、Velero CRD、Velero Deployment/DaemonSet、`ServerStatusRequest`、Node、Namespace 与可保护资源；HTTP 创建请求不等待这些下层动作完成
- 已写入内容：五段详细说明、kubeconfig 与 token 两种 RunAPI 样例差异、JWT/JSON header、`name/description/tag/kubeConfig/token/endpoint/imageSources/veleroInstall` 全字段说明、`username` 可回显和 `password` write-only 约束、endpoint 去重规则、Secret 创建与回滚、DTO 脱敏返回、operator 异步状态链路、当前错误分类
- 取证备注：server 当前只强制绑定 `name`，但要让 operator 后续把集群推进到 `Ready`，必须提供合法 `kubeConfig`，或者同时提供合法 `token` 和 `endpoint`；若两类凭据都不提供，HTTP 创建可以成功，operator 会标记 `NotReady/InvalidSpec`
- 主要错误点：JWT 失败由中间件返回 `401/403`；JSON、kubeconfig、endpoint、imageSources、veleroInstall 或 Kubernetes 对象校验失败返回 `400 code=1000`；同名 Cluster 或归一化 endpoint 冲突返回 `409 code=3009`；Cluster 列表、registry Secret、Cluster CR 创建等下层失败返回 `500 code=5000`

## DELETE /apis/cluster.testudo.softcdata.com/v1/clusters/:name

- RunAPI Target ID：`25559f2038c06f`
- RunAPI 状态：已存在，已更新详细说明，URL 已由固定 `/clusters/ip172` 修正为 `/clusters/:name`，鉴权类型保持继承项目鉴权，已补齐必填 `Authorization` header、path `name` 参数、query `uninstallVelero` 参数和 `200/400/404/500` 响应示例，原说明为空因此未追加 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_cluster/v1/router.go`，`DELETE /apis/cluster.testudo.softcdata.com/v1/clusters/:name`
- server handler：`internal/apis/disaster_cluster/v1/handler.go`，`ClusterHandler.deleteCluster`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> 读取 path `name` -> 解析 query `uninstallVelero` -> 尽力读取现有 Cluster -> 写入 trace/user annotation -> 若显式传入 `uninstallVelero` 则持久化或删除 `testudo.softcdata.com/uninstall-velero` annotation -> `DisasterClient.DisasterV1().Clusters().Delete` -> 若引用 server 管理的 registry Secret，则尽力删除管理面 Secret -> `transport.WriteSuccess(200, {"name": name})`
- operator 链路：`ClusterReconciler` 看到 deletionTimestamp 后将状态置为 `Deleting` 并发射删除开始事件；若 annotation `testudo.softcdata.com/uninstall-velero=true`，执行目标集群 Velero uninstall；随后清理目标集群 Velero registry pull secret，发射删除完成事件并移除 `testudo.softcdata.com/cluster-finalizer`
- 下层资源链路：server 同步删除管理集群 `Cluster` CR 并尽力删除管理面 `disaster-system/cluster-velero-regcred-<clusterName>` Secret；operator 异步访问目标 Kubernetes 集群执行 Velero Helm uninstall 和目标 `velero` 命名空间 registry pull secret 清理；HTTP 响应不等待 finalizer 完成
- 已写入内容：五段详细说明、JWT header、path `name`、query `uninstallVelero` 可选值和默认行为、删除前 annotation 持久化要求、管理面 Secret 尽力清理、operator finalizer 删除链路、成功响应和当前错误分类
- 取证备注：当前 operator 中依赖检查逻辑被注释，删除不会因 AppBackup/AppRestore/DisasterConfig 引用在 controller 侧阻塞；如果 finalizer 阶段目标集群不可达或清理失败，CR 会停留在删除中并按 10 秒间隔重试相关失败项
- 主要错误点：JWT 失败由中间件返回 `401/403`；`uninstallVelero` 非法返回 `400 code=1000`；目标 Cluster 不存在返回 `404 code=3004`；显式传入 `uninstallVelero` 后删除前持久化 annotation 失败或 Kubernetes delete 失败返回 `500 code=5000`

## GET /apis/cluster.testudo.softcdata.com/v1/clusters/:name

- RunAPI Target ID：`25559f1fb8c069`
- RunAPI 状态：已存在，已更新详细说明，URL 已由固定 `/clusters/cluster-ip172` 修正为 `/clusters/:name`，鉴权类型保持继承项目鉴权，已补齐必填 `Authorization` header、path `name` 参数和 `200/404/500` 响应示例，原说明已保留到 `## 原有说明`，已回读验证；2026-06-01 追加 `veleroInstall.username` 回显成功响应示例
- server 路由：`internal/apis/disaster_cluster/v1/router.go`，`GET /apis/cluster.testudo.softcdata.com/v1/clusters/:name`
- server handler：`internal/apis/disaster_cluster/v1/handler.go`，`ClusterHandler.cluster`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> 读取 path `name` -> `DisasterClient.DisasterV1().Clusters().Get` -> `convertToDisasterClusterDTO` -> 若引用 server 管理的 Velero registry Secret 则解析 username -> `transport.WriteSuccess(200)`
- operator 链路：本接口不触发 reconcile；返回的 `status` 与统计 labels 来自 `ClusterReconciler` 已异步维护的 `Cluster.status` 和 labels，包括目标集群连通性、Velero 版本、节点数、namespace/resource/workload 统计、token 过期时间、异常 reason/message 等
- 下层资源链路：接口本身只读管理集群 `Cluster` CR，不访问目标 Kubernetes 集群、Velero、对象存储或业务资源；当 Cluster 引用 server 管理的 Velero registry Secret 时，会只读管理面 `disaster-system/cluster-velero-regcred-<clusterName>` Secret 以解析 username；下层状态来源于 operator 已完成的目标集群访问和统计结果
- 已写入内容：五段详细说明、JWT header、path `name`、DTO 全字段来源、敏感字段脱敏规则、`imageSources`、`veleroInstall` 脱敏回显、namespace/workload 统计口径、operator 状态字段来源、当前错误分类
- 取证备注：清单中原先把 Target `1be22307d0001001` 也列到详情接口；回读确认该 Target 实际为 `GET /clusters/names`，本条详情接口只更新 `25559f1fb8c069`
- 主要错误点：JWT 失败由中间件返回 `401/403`；目标 Cluster 不存在返回 `404 code=3004`；Kubernetes client 读取 Cluster 非 NotFound 错误返回 `500 code=5000`；operator 异步错误通过 `data.status.reason/message` 展示，不改变详情查询 HTTP 状态

## PATCH /apis/cluster.testudo.softcdata.com/v1/clusters/:name

- RunAPI Target ID：`1bd502f9fe001000`
- RunAPI 状态：已存在，已更新详细说明，URL 已由固定 `/clusters/master01` 修正为 `/clusters/:name`，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，已补齐必填 `Authorization` 与 `Content-Type` header、path `name`、当前支持的 JSON body 字段和 `200/400/404/500` 响应示例，原说明已保留到 `## 原有说明`，已回读验证；2026-06-01 追加 `veleroInstall.username` 回显成功响应示例，并追加 `username=""` 清空凭据、`imageRegistry=""` 清空配置两个成功响应示例
- server 路由：`internal/apis/disaster_cluster/v1/router.go`，`PATCH /apis/cluster.testudo.softcdata.com/v1/clusters/:name`
- server handler：`internal/apis/disaster_cluster/v1/handler.go`，`ClusterHandler.patchCluster`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> 读取 path `name` -> `BindJSON(PatchDisasterClusterRequest)` -> `Clusters().Get` -> 按指针字段局部更新 `token/tag/description/imageSources/veleroInstall` -> `imageRegistry=""` 清空整段 Velero 安装配置 -> `username=""` 清空 registry 凭据并保留镜像源 -> 可选创建/更新/删除管理面 registry Secret -> 无支持字段时通过 `convertToDisasterClusterDTO` 直接返回当前 DTO -> 有更新时写入 trace/user annotation -> `Clusters().Update` -> `convertToDisasterClusterDTO` -> 若引用 server 管理的 Velero registry Secret 则解析 username -> `transport.WriteSuccess(200)`
- operator 链路：本接口更新 CR 后由 `ClusterReconciler` 异步处理 generation 或 metadata 变化，发射编辑集群事件，重新构建目标集群客户端、同步 Velero registry Secret 到目标集群、必要时执行 Velero install/upgrade 检查，随后刷新状态、token 过期时间和统计字段；HTTP 响应不等待 reconcile
- 下层资源链路：server 同步读写管理集群 `Cluster` CR 和可选 `disaster-system/cluster-velero-regcred-<clusterName>` Secret；operator 后续访问目标 Kubernetes 集群和 Velero 资源进行同步与状态刷新
- 已写入内容：五段详细说明、JWT/JSON header、path `name`、`token/tag/description/imageSources/veleroInstall` 全字段更新语义、`endpoint/kubeConfig` 不支持 PATCH 的事实、Secret 轮换/移除规则、`imageRegistry=""` 和 `username=""` 显式清空语义、`username` 可回显和 `password` 不回显的 DTO 脱敏返回、operator 异步编辑链路、当前错误分类
- 取证备注：`tag=""` 会删除 cluster tag label，`description=""` 会删除备注 annotation，`imageSources: []` 会清空镜像源目录，`veleroInstall.imageRegistry=""` 会清空整段 Velero 安装配置，`veleroInstall.username=""` 会清空 registry 凭据；请求体没有任何支持字段时返回 200 当前 DTO，但不会执行 Update，也不会写 trace/user annotation
- 主要错误点：JWT 失败由中间件返回 `401/403`；JSON、imageSources、veleroInstall 或 Kubernetes 对象校验失败返回 `400 code=1000`；目标 Cluster 不存在返回 `404 code=3004`；读取 Cluster、管理面 registry Secret 或更新 Cluster CR 的下层错误返回 `500 code=5000`

## POST /apis/cluster.testudo.softcdata.com/v1/clusters/:name/actions/refresh-namespaces

- RunAPI Target ID：`1c613a8d76001001`
- RunAPI 状态：已存在，已更新详细说明，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，已补齐必填 `Authorization` 与 `Content-Type` header、path `name`、必填 body `type` 和 `202/400/404/409` 响应示例，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_cluster/v1/router.go`，`POST /apis/cluster.testudo.softcdata.com/v1/clusters/:name/actions/refresh-namespaces`
- server handler：`internal/apis/disaster_cluster/v1/handler.go`，`ClusterHandler.refreshNamespaces`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> trim path `name` -> `BindJSON(RefreshNamespacesRequest)` -> trim `type` -> `metadata.IsValidClusterStatsRefreshType` 校验 `namespaceStats/workloadNamespaceStats/all` -> `retry.RetryOnConflict` 读取 Cluster 并写入 annotation `testudo.softcdata.com/refresh-cluster-stats=<type>` -> `ConvertToDisasterClusterDTO` -> `transport.WriteSuccess(202)`
- operator 链路：`ClusterReconciler.processRefreshClusterStatsSignal` 在后续 reconcile 中读取 annotation；`namespaceStats` 调用 `collectNamespaceStats`，`workloadNamespaceStats` 调用 `collectWorkloadNamespaceStats`，`all` 调用 `collectClusterStats`；成功后更新 `status.lastCheckTime`、status 统计字段、统计 labels，并清除 refresh annotation
- 下层资源链路：server 只更新管理集群 `Cluster` CR annotation；operator 异步访问目标 Kubernetes 集群 discovery、Namespace、namespaced 可保护资源以及 running `Deployment/StatefulSet` 工作负载命名空间，本 HTTP 请求不直接访问目标集群
- 已写入内容：五段详细说明、JWT/JSON header、path `name`、必填 `type` 三种可选值、每种刷新类型控制的 status 字段、`202` 只表示接收请求、DTO 返回快照和统计口径、operator 异步清 annotation 链路、当前错误分类
- 取证备注：`data.cluster.status.*` 是写入 annotation 后的集群快照，可能仍是刷新前旧值；`workloadTotalCount` 是 running workload namespace 子集上的 namespace 级可保护资源总数，不是 workload 对象数量
- 主要错误点：JWT 失败由中间件返回 `401/403`；path `name` 为空、JSON 绑定失败、缺少或非法 `type` 返回 `400 code=1000`；目标 Cluster 不存在返回 `404 code=3004`；重试后仍发生更新冲突返回 `409 code=3009`；读取/更新 Cluster 的其他错误返回 `500 code=5000`

## GET /apis/cluster.testudo.softcdata.com/v1/clusters/:name/protected-namespaces

- RunAPI Target ID：`1c61734794401001`
- RunAPI 状态：已存在，已更新详细说明，URL 已由固定 `/clusters/ip170-test-001/protected-namespaces` 修正为 `/clusters/:name/protected-namespaces`，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，已补齐必填 `Authorization` header、path `name`、分页/排序/keyword query 和 `200/404/500` 响应示例，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_cluster/v1/router.go`，`GET /apis/cluster.testudo.softcdata.com/v1/clusters/:name/protected-namespaces`
- server handler：`internal/apis/disaster_cluster/v1/handler.go`，`ClusterHandler.protectedNamespaces`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> trim path `name` -> 校验 Cluster 存在 -> `common.BuildProtectedNamespaceIndex` 列出 `DisasterConfig` 与 `DisasterInstance` -> 按 `DisasterConfig.spec.sourceCluster == name` 和 `DisasterInstance.spec.config/spec.namespaces` 建索引 -> `keyword` 匹配 namespace/instanceName/instanceNamespace/configName -> 默认 `sort=namespace, order=asc` -> 未传 limit 时返回全部记录（空结果 limit=1）-> `transport.Paginate` -> `transport.BuildCollectionResponse(resourceType=clusterProtectedNamespace)` -> `transport.WriteSuccess(200)`
- operator 链路：本接口不触发 operator；它读取 operator/业务接口已创建和维护的 `DisasterConfig`、`DisasterInstance` CR 快照，间接反映当前容灾实例对命名空间的占用关系
- 下层资源链路：只读管理集群 informer cache 中的 `Cluster`、`DisasterConfig`、`DisasterInstance`；不访问目标 Kubernetes 集群、Velero、StorageRepository 或对象存储
- 已写入内容：五段详细说明、JWT header、path `name`、`keyword/limit/page/sort/order` 参数、索引构建规则、owner 字段来源、默认分页与排序行为、collection envelope、当前错误分类
- 取证备注：`meta.filters` 只会包含 `transport.ParseOptions` 收集的额外 query 参数；当前 handler 实际只使用 `keyword` 做过滤，其他自定义 filter 不参与筛选
- 主要错误点：JWT 失败由中间件返回 `401/403`；path `name` 为空返回 `400 code=1000`；目标 Cluster 不存在返回 `404 code=3004`；读取 Cluster、列出 `DisasterConfig` 或列出 `DisasterInstance` 失败返回 `500 code=5000`

## GET /apis/cluster.testudo.softcdata.com/v1/clusters/:name/restore-classes

- RunAPI Target ID：`1c4bb4e50d001001`
- RunAPI 状态：已存在，已更新详细说明，URL 已由固定 `/clusters/cluster-ip170-1774332377/restore-classes` 修正为 `/clusters/:name/restore-classes`，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，已补齐必填 `Authorization` header、path `name` 和 `200/400/404/500` 响应示例，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_cluster/v1/router.go`，`GET /apis/cluster.testudo.softcdata.com/v1/clusters/:name/restore-classes`
- server handler：`internal/apis/disaster_cluster/v1/handler.go`，`ClusterHandler.listRestoreClasses`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> trim path `name` -> `getClusterClient` 读取 Cluster 并使用 `spec.kubeConfig` 或 `spec.token/spec.endpoint` 创建目标集群 controller-runtime client -> list 目标集群 `StorageClassList` -> list 目标集群 `IngressClassList` -> 根据默认 class annotation 判断 `isDefault` -> 两个列表分别按 name 升序排序 -> `transport.WriteSuccess(200)`
- operator 链路：本接口不触发 operator；它依赖 Cluster CR 中由创建/更新接口写入、operator 维护可用性的访问凭据，并实时访问目标集群读取 class 资源
- 下层资源链路：只读管理集群 `Cluster` CR；实时访问目标 Kubernetes 集群的 `storage.k8s.io/v1 StorageClass` 与 `networking.k8s.io/v1 IngressClass`；不访问 Velero、StorageRepository、对象存储、DisasterConfig 或 DisasterInstance
- 已写入内容：五段详细说明、JWT header、path `name`、目标集群 client 构建规则、StorageClass/IngressClass 返回字段、默认 class annotation 判断规则、排序规则、当前错误分类
- 取证备注：默认 StorageClass 同时兼容 `storageclass.kubernetes.io/is-default-class` 和 beta annotation；默认 IngressClass 使用 `ingressclass.kubernetes.io/is-default-class`；annotation 值 trim 后大小写不敏感等于 `true` 才返回 `isDefault=true`
- 主要错误点：JWT 失败由中间件返回 `401/403`；path `name` 为空、缺少目标集群访问凭据、kubeconfig 解析失败或目标集群 client 创建失败返回 `400 code=1000`；Cluster 不存在返回 `404 code=3004`；目标集群 list StorageClass 或 IngressClass 失败返回 `500 code=5000`

## GET /apis/cluster.testudo.softcdata.com/v1/clusters/:name/validate

- RunAPI Target ID：`25559f2038c072`
- RunAPI 状态：已存在，已更新详细说明，URL 已由固定 `/clusters/master/validate` 修正为 `/clusters/:name/validate`，鉴权类型保持继承项目鉴权，已补齐必填 `Authorization` header、path `name` 参数，已移除历史不符合当前实现的 404 失败示例，改为 `Ready` 与“不存在或非 Ready”两个 `200` 示例，原说明为空因此未追加 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_cluster/v1/router.go`，`GET /apis/cluster.testudo.softcdata.com/v1/clusters/:name/validate`
- server handler：`internal/apis/disaster_cluster/v1/handler.go`，`ClusterHandler.validateCluster`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> 读取 path `name` -> `DisasterClient.DisasterV1().Clusters().Get` -> NotFound 返回 `transport.WriteSuccess(200, false)` -> 任意其他读取错误返回 `transport.WriteSuccess(200, false)` -> `status.status != Ready` 返回 false -> Ready 返回 true
- operator 链路：本接口不触发 operator；`status.status` 由 `ClusterReconciler` 异步维护
- 下层资源链路：只读管理集群 `Cluster` CR；不访问目标 Kubernetes 集群、Velero、StorageRepository、DisasterConfig、DisasterInstance 或对象存储
- 已写入内容：五段详细说明、JWT header、path `name`、`data=true/false` 判定规则、NotFound/读取错误不返回错误 envelope 的当前实现、当前错误分类
- 取证备注：该接口不返回 NotFound 或 NotReady 具体原因；需要原因时应查询 `GET /clusters/:name` 的 `data.status.reason/message`
- 主要错误点：JWT 失败由中间件返回 `401/403`；Cluster NotFound、Kubernetes 读取错误和非 Ready 状态都返回 `200 code=0 data=false`

## POST /apis/cluster.testudo.softcdata.com/v1/clusters/kubeconfig/validate

- RunAPI Target ID：`67d3bab78c00b`（验证连接 kubeconfig）与 `69dbb8ef8c07f`（验证连接 token）
- RunAPI 状态：已存在两个同 URL/同 handler 的请求样例，已分别更新详细说明，鉴权类型保持继承项目鉴权，已补齐必填 `Authorization` 与 `Content-Type` header，已按 kubeconfig/token 两种样例修正 JSON body、schema required、字段中文说明和全 `200` 业务结果示例，原说明均为空因此未追加 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_cluster/v1/router.go`，`POST /apis/cluster.testudo.softcdata.com/v1/clusters/kubeconfig/validate`
- server handler：`internal/apis/disaster_cluster/v1/handler.go`，`ClusterHandler.validateKubeConfig`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> `BindJSON(ValidateKubeConfigRequest)` -> 若 `kubeConfig` 非空则 `tools.GetRestConfig` -> 否则若 `token/endpoint` 非空则 token 非 `eyJ` 前缀时尝试 Base64 decode，再 `tools.GetRestConfigFromToken` -> 否则返回 `200 false meta.error` -> `kubernetes.NewForConfig` -> `clientset.ServerVersion` -> 成功返回 `200 true`
- operator 链路：无；本接口只验证传入凭据，不读取或更新 `Cluster` CR，不触发 disaster-operator reconcile
- 下层资源链路：仅在内存中构造 Kubernetes rest config，并调用目标 Kubernetes API Server discovery/version；不访问 Velero、StorageRepository、DisasterConfig、DisasterInstance、Cluster CR 或对象存储
- 已写入内容：五段详细说明、kubeconfig 与 token 两种样例差异、JWT/JSON header、`kubeConfig/token/endpoint` 字段说明、token Base64 解码规则、凭据组合优先级、`data=true/false` 返回语义、所有业务校验失败均为 `200 code=0 data=false meta.error` 的当前实现
- 取证备注：请求同时包含 `kubeConfig` 与 `token/endpoint` 时优先使用 `kubeConfig`；`token` 不以 `eyJ` 开头且 Base64 解码成功时使用解码后内容，解码失败则按原字符串使用
- 主要错误点：JWT 失败由中间件返回 `401/403`；JSON 绑定失败、缺少凭据、kubeconfig 解析失败、clientset 创建失败、目标集群不可达、认证失败或 `ServerVersion` 失败都返回 `200 code=0 data=false meta.error=<原因>`

## GET /apis/cluster.testudo.softcdata.com/v1/clusters/names

- RunAPI Target ID：`1be22307d0001001`
- RunAPI 状态：已存在，已更新详细说明，URL 已由固定 `/clusters/names?keyword=importance` 修正为 `/clusters/names`，鉴权类型已从历史 `noauth` 修正为继承项目鉴权，已补齐必填 `Authorization` header、`keyword/tag/label/limit/page` query 和 `200/400` 响应示例，原说明已保留到 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_cluster/v1/router.go`，`GET /apis/cluster.testudo.softcdata.com/v1/clusters/names`
- server handler：`internal/apis/disaster_cluster/v1/handler.go`，`ClusterHandler.clusterNames`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> `transport.ParseOptions` -> `tag` 映射为 `testudo.softcdata.com/cluster-tag` -> `BuildLabelSelector` -> `ClusterLister.List` -> 对 filters 执行 label 包含匹配 -> `keyword` 对 `metadata.name` 和 cluster tag 包含匹配 -> 转换为 `DisasterClusterNameDTO` -> `transport.WriteSuccess(200)`
- operator 链路：本接口不触发 operator；返回的统计摘要字段来自 `ClusterReconciler` 已写入的 `Cluster.status`
- 下层资源链路：只读管理集群 Cluster informer cache；不访问目标 Kubernetes 集群、Velero、StorageRepository、DisasterConfig、DisasterInstance 或对象存储
- 已写入内容：五段详细说明、JWT header、`keyword/tag/label/limit/page` 参数、`limit/page/sort/order` 不实际截断或排序的当前行为、返回简单数组而非 collection envelope、各摘要字段来源、当前错误分类
- 取证备注：本接口虽然调用 `transport.ParseOptions`，但注释明确忽略 `limit/page` 且没有调用 `transport.Sort/Paginate`，因此会返回全部匹配名称摘要
- 主要错误点：JWT 失败由中间件返回 `401/403`；`ClusterLister.List` 失败时当前 handler 使用 `400 code=1000`

## GET /apis/cluster.testudo.softcdata.com/v1/watch/clusters

- RunAPI Target ID：`2667344638c320`
- RunAPI 状态：已存在，已更新详细说明，保持 RunAPI `websocket2` 类型，URL 已确认是 `/watch/clusters`，已补充 `Authorization`、`Sec-WebSocket-Protocol` 和 query `token` 鉴权说明，原说明为空因此未追加 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_cluster/v1/router.go`，`GET /apis/cluster.testudo.softcdata.com/v1/watch/clusters`
- server handler：`internal/apis/disaster_cluster/v1/handler.go`，`ClusterHandler.watchClusters`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` 支持 `?token=` 或 `Sec-WebSocket-Protocol` 转 Authorization -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> `watchutils.StreamWatch` -> WebSocket upgrade -> `Clusters().Watch(ctx, metav1.ListOptions{})` -> 连接成功消息 -> 30 秒心跳 -> Kubernetes watch event 转 `WatchEventDTO` -> `convertToDisasterClusterDTO` -> 若引用 server 管理的 Velero registry Secret 则解析 username -> WebSocket JSON envelope
- operator 链路：本接口不触发 operator；实时推送 operator 和其他接口导致的 Cluster CR 创建、修改、删除事件
- 下层资源链路：只读管理集群 Kubernetes watch 流中的 Cluster CR；当 Cluster 引用 server 管理的 Velero registry Secret 时，会只读管理面 `disaster-system/cluster-velero-regcred-<clusterName>` Secret 以解析 username；不访问目标 Kubernetes 集群、Velero、StorageRepository、DisasterConfig、DisasterInstance 或对象存储
- 已写入内容：五段详细说明、WebSocket 而非 SSE 的事实、三种鉴权方式、连接成功/心跳/watch event/closed/timeout 消息 envelope、事件 DTO 字段、`veleroInstall.username` 可回显和 `password` 不回显、默认 30 分钟超时与 30 秒心跳、当前错误分类
- 取证备注：ApiPost `websocket2` 目标回读时不保留普通 HTTP API 的 `auth/restful` 结构；本条通过 URL、header/query 参数和 description 记录鉴权与消息格式
- 主要错误点：握手前 JWT 失败返回 `401/403`；WebSocket upgrade 失败返回普通 `500 {"message":"WebSocket 升级失败: <原因>"}`；已连接后 watcher 创建失败会通过 WebSocket 发送 `code=5000` error envelope

## GET /apis/cluster.testudo.softcdata.com/v1/watch/clusters/:name

- RunAPI Target ID：`2667d19538c36d`
- RunAPI 状态：已存在，已更新详细说明，保持 RunAPI `websocket2` 类型，URL 已由固定 `/watch/clusters/cluster-1` 修正为 `/watch/clusters/:name`，已补充 `Authorization`、`Sec-WebSocket-Protocol` 和 query `token` 鉴权说明，原说明为空因此未追加 `## 原有说明`，已回读验证
- server 路由：`internal/apis/disaster_cluster/v1/router.go`，`GET /apis/cluster.testudo.softcdata.com/v1/watch/clusters/:name`
- server handler：`internal/apis/disaster_cluster/v1/handler.go`，`ClusterHandler.watchCluster`
- 请求链路：`/apis` 分组 `WebSocketTokenAdapter` 支持 `?token=` 或 `Sec-WebSocket-Protocol` 转 Authorization -> 非 `dev` 环境 JWT 中间件 -> `TraceMiddleware` -> 读取 path `name` -> `watchutils.StreamWatch` -> WebSocket upgrade -> `Clusters().Watch(ctx, metav1.ListOptions{FieldSelector: metadata.name=<name>})` -> 连接成功消息 -> 30 秒心跳 -> 匹配名称的 Kubernetes watch event 转 `WatchEventDTO` -> `convertToDisasterClusterDTO` -> 若引用 server 管理的 Velero registry Secret 则解析 username -> WebSocket JSON envelope
- operator 链路：本接口不触发 operator；实时推送指定 Cluster CR 的创建、修改、删除事件；若资源当前不存在，watch 仍可建立并等待后续匹配事件
- 下层资源链路：只读管理集群 Kubernetes watch 流中的指定 Cluster CR；当 Cluster 引用 server 管理的 Velero registry Secret 时，会只读管理面 `disaster-system/cluster-velero-regcred-<clusterName>` Secret 以解析 username；不访问目标 Kubernetes 集群、Velero、StorageRepository、DisasterConfig、DisasterInstance 或对象存储
- 已写入内容：五段详细说明、WebSocket 而非 SSE 的事实、path `name` 语义、三种鉴权方式、field selector、连接成功/心跳/watch event/closed/timeout 消息 envelope、`veleroInstall.username` 可回显和 `password` 不回显、当前错误分类
- 取证备注：ApiPost `websocket2` 目标回读时未保留普通 restful parameter 结构；path 参数语义已体现在 URL `:name` 和 description 中
- 主要错误点：path `name` 为空时 handler 返回 `400 code=1000`；握手前 JWT 失败返回 `401/403`；WebSocket upgrade 失败返回普通 `500 {"message":"WebSocket 升级失败: <原因>"}`；已连接后 watcher 创建失败会通过 WebSocket 发送 `code=5000` error envelope

## POST /disasterjobs.testudo.softcdata.com/v1/jobs

- RunAPI Target ID：`3ee6850678c074`
- RunAPI 状态：已存在，但属于 RunAPI 额外/待人工复核项；已回读确认 URL 为 `{{baseUrl}}/disasterjobs.testudo.softcdata.com/v1/jobs`，缺少 server 实际 `/apis` 前缀，且使用 `baseUrl` 变量名，与本项目已处理的标准 `{{baseurl}}` 目标不一致
- server 路由核对：`internal/router/router.go` 只把业务模块注册到 `bashPath := sh.Group("/apis")`；`internal/apis/disaster_jobs/v1/router.go` 在该分组下注册 `POST /jobs`，完整路径为 `POST /apis/disasterjobs.testudo.softcdata.com/v1/jobs`
- 已存在正确 RunAPI 目标：`POST /apis/disasterjobs.testudo.softcdata.com/v1/jobs`，Target ID `25559f21f8c0aa`，已在前面任务模块完成详细说明更新和回读验证
- 复核结论：server 清单不需要补录裸路径接口；该 RunAPI 目标应视为历史重复/路径漏 `/apis` 的额外目标，建议人工删除、移动到废弃目录，或统一迁移到已存在的正确 `/apis` 目标
- 处理说明：未按正式接口写五段详细说明，避免把不存在的裸路径发布为有效接口文档；本项只作为差异对账证据记录

## GET /apis/v1/platform-license/status

- RunAPI Target ID：`1c7dbf4632001001`
- RunAPI 状态：已存在，已更新详细说明，新增 `LicenseEnvironmentInvalid` 与“License 内容错误优先于部署指纹/CA path 错误”的实时评价语义；本次追加当前指纹返回字段说明，原说明已保留到 `## 原有说明`，已新增“免费版状态含当前指纹”`200` 响应示例并回读验证
- server 路由：`internal/apis/platform_license/v1/router.go`，`GET /apis/v1/platform-license/status`
- server handler/service：`h.status` -> `Service.Status` -> `Service.withCurrentFingerprint` -> `KubernetesStore.Fingerprint`
- 请求链路：`/apis` 分组鉴权 -> 统计当前未删除 Cluster -> 优先读取 status ConfigMap；ConfigMap 不存在时通过 direct reader 读取 License Secret 并实时评价；返回前统一读取 Namespace、install-id 和 CA bundle 计算当前指纹，install-id Secret 不存在时由 `KubernetesStore.EnsureInstallID` 创建
- 已写入内容：`data.fingerprint`、`data.fingerprintVersion`、`data.fingerprintRequest` 字段说明；说明 malformed/UnknownKey/InvalidSignature 不被 CA path 缺失掩盖；部署环境错误以 `200 code=0 data.state=Unknown` 返回并回退免费额度；RunAPI Target ID `1c7dbf4632001001` 已回读确认新增说明与当前指纹示例存在

## POST /apis/v1/platform-license/install

- RunAPI Target ID：`1c7dbf57bd001001`
- RunAPI 状态：已存在，已更新详细说明，新增安装后基于本次写入内容实时评价、避免旧 `Free/NoLicense`、内容错误优先于部署指纹错误的语义；本次追加安装响应当前指纹字段说明，原说明已保留到 `## 原有说明`，已新增“安装 malformed License 含当前指纹”`200` 响应示例并回读验证
- server 路由：`internal/apis/platform_license/v1/router.go`，`POST /apis/v1/platform-license/install`
- server handler/service：`h.install` -> `Service.Install` -> 写入 `disaster-platform-license` Secret -> `Service.liveStatus` -> `Service.withCurrentFingerprint`
- 请求链路：请求体 `license` 必须是合法 JSON 字符串；写入 Secret 后返回 `source=liveEvaluation`；`{}` 返回 `Malformed/LicenseInvalid`；未知 key 返回 `UnknownKey/LicenseUnknownKey`；返回前统一计算当前指纹并填充 `fingerprintRequest`；环境无法计算指纹时，内容错误优先保留，内容有效时返回 `Unknown/LicenseEnvironmentInvalid`
- 已写入内容：响应字段语义、错误优先级、环境错误 reason、当前指纹字段、当前响应示例和 RunAPI 文档同步状态；RunAPI Target ID `1c7dbf57bd001001` 已回读确认新增说明与当前指纹示例存在

## 全局：server DTO 时间字段本地时区序列化

- RunAPI 状态：本次属于跨模块响应字段语义统一调整，未逐个覆盖所有接口说明；已在 Apipost 项目新增 Markdown 文档《server DTO 时间字段本地时区序列化说明》（Target ID：`1c8551dde7001000`），用于记录全局约定
- server 变更入口：`internal/common/local_time.go`，新增 `LocalTime`、`LocalCondition` 与转换函数；各 API DTO 在转换层使用 `common.NewLocalTime` / `common.NewLocalTimePtr`
- 影响模块：Cluster、StorageRepository、DisasterConfig、DisasterPolicy、DisasterInstance、DisasterDrill、DisasterGroup、DisasterBackup、DisasterJob、AppBackup、AppRestore、BackupRestoreStatistics、TaskEvent，以及相关 watch/历史/下载响应
- 行为说明：server 对外响应 DTO 中的 Kubernetes/Velero/operator 时间字段统一序列化为 RFC3339 offset 字符串，并使用进程本地时区 `time.Local`；容器设置 `TZ=Asia/Shanghai` 时示例为 `2026-05-15T11:08:19+08:00`
- 边界说明：Kubernetes apiserver 存储与 `kubectl get -o yaml` 展示的 `metadata.creationTimestamp` 仍为 UTC `Z`，本次仅改变 server API JSON DTO 的序列化结果；请求体时间字段仍按原 `metav1.Time` 解析，不改变 CRD 写入契约
- OpenAPI 状态：已在 `openspec/specs/disaster-server-openapi.yaml` 增加 `LocalRFC3339Time` 共享说明，并在 `info.description` 与主要创建时间字段描述中说明本地时区输出

## 全局：管理面 namespace 可配置

- RunAPI 状态：本次属于跨模块资源范围修复，涉及大量已存在接口；未逐个通过 Apipost live 更新目标说明，已在本地证据记录并更新 OpenAPI 中“固定 disaster-system”的描述，待后续 RunAPI 批量同步。
- server 变更入口：`configs/config.go` 读取 `server.namespace` 并设置 `common.DisasterSystemNamespace`；本次补齐 Group、Instance、Drill 剩余 handler，统一使用 `common.DisasterSystemNamespace`。
- operator 链路：`cmd/main.go` 新增 `--management-namespace`，并将其传入全局 controller 管理命名空间、license namespace 默认值、cluster-scoped event 默认落点、dependency backfill；StorageRepository 依赖读取从硬编码 `disaster-system` 改为配置的管理命名空间。
- chart 链路：`templates/server-install.yaml` 渲染 `server.namespace`；`templates/operator-install.yaml` 渲染 `--management-namespace`、`--license-namespace`、`--license-ca-path`。
- 影响模块：DisasterGroup、DisasterInstance、DisasterDrill、AppBackup/AppRestore 的 StorageRepository 链路、DisasterConfig/DisasterPolicy 依赖链路、DataSync/ResourceSync StorageRepository ready 检查、Cluster ensure-storage、license status runnable、cluster-scoped event 默认 namespace。
- 行为说明：安装到非默认 namespace 时，server 和 operator 使用 chart 渲染的管理命名空间读写管理面 namespaced CR/Secret/ConfigMap/Event；默认值仍为 `disaster-system`。
- OpenAPI 状态：已将 `openspec/specs/disaster-server-openapi.yaml` 中与固定 `disaster-system` 资源范围相关的描述改为“配置的管理命名空间（默认 `disaster-system`）”。

## Velero Hook 透传 API 契约

- RunAPI 状态：已通过 Apipost MCP 回读受影响 API target，并完成 live 同步。共享 Markdown 文档 `Velero Hook 透传 API 契约` 已重建为字段释义版当前契约，Target ID `1ca8437f2a401000`；为 AppBackup/AppRestore/DisasterInstance/DisasterDrill 相关接口新增当前 hooks、hookStatus、敏感参数拒绝响应示例。2026-06-09 已为 AppRestore list/detail/watch 四个 target 追加 `PartiallyFailed` 响应或事件帧示例。为避免覆盖既有超长接口说明，未批量重写原接口 description。
- 2026-06-10 补充：因 Apipost `update_target` 对 Markdown doc 返回成功但回读正文和标题均未变，已删除旧文档 Target ID `1ca486eeb4001000`、临时当前版 Target ID `1ca6eaf0e8001000`、上一版 Target ID `1ca6ecb4e7401000` 和字段释义版 Target ID `1ca6ef895f001000`，并使用正确内容重新创建同名文档 `Velero Hook 透传 API 契约`，最终 Target ID `1ca8437f2a401000`。该文档明确 Velero 原生字段名：Backup 使用 `resources[].pre/post` 且只支持 exec；Restore 使用 `resources[].postHooks` 且支持 exec/init、没有 `preHooks`；同时补充多个 `resources[]`、多个 `pre[]/post[]` exec、前端表单映射、DisasterInstance 创建/更新传参、DisasterDrill 覆盖/清空继承传参、敏感参数约束，以及每个 Hook 字段的中文释义。
- 受影响接口与 Target ID：
  - AppBackup：`GET /apis/appbackups.testudo.softcdata.com/v1/appbackups` (`2a25b664f8c042`)，`POST /apis/appbackups.testudo.softcdata.com/v1/appbackups` (`3ee68503b8c051`、`3efbb43238c57f`)，`GET /apis/appbackups.testudo.softcdata.com/v1/appbackups/:name` (`2a25b664f8c043`)，`PUT /apis/appbackups.testudo.softcdata.com/v1/appbackups/:name` (`3ee6850438c053`)，`GET /apis/appbackups.testudo.softcdata.com/v1/appbackups/:name/history` (`1bd4dbb7f4401001`)
  - AppRestore：`GET /apis/apprestores.testudo.softcdata.com/v1/apprestores` (`34a0ae4978c001`)，`POST /apis/apprestores.testudo.softcdata.com/v1/apprestores` (`3ee6850478c056`)，`GET /apis/apprestores.testudo.softcdata.com/v1/apprestores/:name` (`34a0ae49b8c002`)，`PUT /apis/apprestores.testudo.softcdata.com/v1/apprestores/:name` (`3ee68504b8c058`)，`GET /apis/apprestores.testudo.softcdata.com/v1/watch/apprestores` (`34a10209f8c00c`)，`GET /apis/apprestores.testudo.softcdata.com/v1/watch/apprestores/:name` (`34a12bb3b8c041`)
  - DisasterInstance：`POST /apis/disasterinstances.testudo.softcdata.com/v1/instances` (`1c01f46f55801000`)，`PUT /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name` (`1c01f4718c001000`)，`GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/sync-status` (`1c01f74a5e401000`)，`GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/sync-history` (`1c8428770dc01001`)
  - DisasterDrill：`POST /apis/disasterdrills.testudo.softcdata.com/v1/drills` (`1c0aec0a8f401001`)
- server 变更入口：
  - `internal/apis/velero_hooks/hooks.go`：BackupHooks/RestoreHooks 校验、presence/clear patch、敏感参数错误 meta。
  - `internal/apis/app_backup/v1/types.go` / `handler.go`：create/update/detail/list 支持 `hooks`。
  - `internal/apis/app_restore/v1/types.go` / `handler.go`：create/update/detail/list 支持 `hooks`。
  - `internal/apis/disaster_instance/v1/types.go` / `handler.go`：create/update/detail/list 支持 `veleroHooks`，sync-status/sync-history 回显 `backupHookStatus` / `restoreHookStatus`。
  - `internal/apis/disaster_drill/v1/types.go` / `handler.go`：create/detail/list/watch 回显演练级 `veleroHooks.dataRestore`，create 拒绝 `veleroHooks.dataBackup`。
- operator/resource 链路：
  - `DisasterInstance.spec.veleroHooks.dataBackup` -> DataSync 生成/对齐的 `AppBackup.spec.template.hooks` -> Velero Backup/Schedule。
  - `DisasterInstance.spec.veleroHooks.dataRestore` -> DataSync 生成的 `AppRestore.spec.template.hooks` -> Velero Restore。
  - `DisasterDrill.spec.veleroHooks.dataRestore` -> `DisasterOperation.spec.drillConfig.veleroHooks` -> drill data restore AppRestore template.hooks；Drill 不创建数据备份，`dataBackup` 被 server 拒绝。
  - `SyncHistoryRecord.backupHookStatus` / `restoreHookStatus` 由 operator 从 AppBackup/AppRestore Velero hookStatus 复制，server DTO 稳定回显。
  - Velero Restore `PartiallyFailed` -> operator `AppRestore.status.status=PartiallyFailed` -> server AppRestore list/detail/watch DTO `status.phase=PartiallyFailed`；该状态为非成功终态，不能再映射为 `Succeeded`。
- 字段契约：
  - AppBackup/AppRestore `hooks`：create 写入；update 未出现保持原值，传对象整体替换，传 null 或 `{}` 清空。
  - DisasterInstance `veleroHooks`：顶层未出现保持原值，传 null 或 `{}` 清空全部；`dataBackup`/`dataRestore` 子字段分别按 presence 处理。
  - DisasterDrill `veleroHooks`：仅允许 `dataRestore`；请求出现 `dataBackup` 返回 `400 code=1000`。
- 校验与错误：
  - includedResources 为空或包含 pods；exec command 必须非空；onError 仅允许 Fail/Continue。
  - timeout 上限：Backup exec `timeout<=10m`，Restore exec `execTimeout<=10m`，Restore exec `waitTimeout<=30m`，Restore init `timeout<=30m`。
  - 明显敏感明文参数硬拒绝，错误 meta 包含 `errorCode=VeleroHookSensitiveParameter` 和 `fieldPath`；敏感值应通过 Secret env、Secret volume、valueFrom 或 envFrom 传入。
- OpenAPI 状态：已在 `openspec/specs/disaster-server-openapi.yaml` 新增 `VeleroBackupHooks`、`VeleroRestoreHooks`、`DisasterVeleroHooks`、`DisasterDrillVeleroHooks`、`VeleroHookStatus`、`SyncHistoryHookStatusDTO` 以及 AppBackup/AppRestore request/response schema，并更新 DisasterInstance/DisasterDrill 相关说明；AppRestore list/detail/watch 已声明 `status.phase=PartiallyFailed` 为非成功终态，server 不做成功映射。

## DisasterInstance bulkModifierActions rewriteImage 动态镜像重写

- RunAPI 状态：已通过 Apipost MCP 回读 `POST /apis/disasterinstances.testudo.softcdata.com/v1/instances` Target ID `1c01f46f55801000` 与 `PUT /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name` Target ID `1c01f4718c001000`；由于两个目标 description 历史内容很长且工具回读被截断，本次没有调用 `update_target` 覆盖说明，避免丢失原有长说明；已为两个目标各新增当前 `rewriteImage` 成功示例和缺少 `imageRewrite.sourcePrefix` 的 `400 ModifierRuleRejected` 示例。
- server 路由：`internal/apis/disaster_instance/v1/router.go`，`POST /apis/disasterinstances.testudo.softcdata.com/v1/instances` 与 `PUT /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name`
- server handler：`internal/apis/disaster_instance/v1/handler.go`，`h.createInstance` / `h.updateInstance`
- 请求链路：`BindJSON` -> `RestorePolicyRequest.ToCRD` 解析 `bulkModifierActions` 或 `bulkModifierActionsText` -> `prepareRestorePolicyForPersist` -> `normalizeBulkModifierActions` 校验 `action=rewriteImage`、`imageRewrite.sourcePrefix`、`imageRewrite.targetPrefix`、`unmatchedPolicy=Keep|Fail`、`digestPolicy=Preserve` -> 仅对 `replaceExactValue/removeKey` 等静态 action 构建 `modifierRuleSnapshot` -> 写入 `DisasterInstance.spec.restorePolicy.bulkModifierActions`
- operator/resource 链路：`rewriteImage` 不在 server 提交期展开长期快照；operator 在 ResourceSync/Drill 恢复构建阶段读取源集群当前 workload/Pod spec 镜像，按 `sourcePrefix -> targetPrefix` 动态编译为现有 `reversible pair` 规则，并继续跳过 `/status/**`、`/metadata/finalizers/**`、`/metadata/ownerReferences/**`。
- 字段契约：`bulkModifierActions[].action` 支持 `replaceExactValue`、`removeKey`、`rewriteImage`；`replaceExactValue` 仍要求 `sourceValue/targetValue`；`removeKey` 仍要求 `key`；`rewriteImage` 要求 `imageRewrite.sourcePrefix/targetPrefix`，不要求完整镜像 `sourceValue/targetValue`，默认 `unmatchedPolicy=Keep`、`digestPolicy=Preserve`、`directionPolicy=Auto`。
- 快照语义：纯 `rewriteImage` 场景 `modifierRuleSnapshot` 和 `modifierRuleSnapshotHash` 为空；混合静态 action 与 `rewriteImage` 时仅静态 action 生成快照，`rewriteImage` 原始 DSL 保留在 `bulkModifierActions` 和 `bulkModifierActionsText` 中。
- OpenAPI 状态：已更新 `openspec/specs/disaster-server-openapi.yaml`，新增 `DisasterInstanceBulkModifierAction`、`DisasterInstanceDynamicImageRewrite`、`DisasterInstanceBulkModifierActionType` 等 schema，并更新创建/更新接口 restorePolicy 与 `modifierRuleSnapshot` 说明。
- 主要错误点：缺少 `imageRewrite`、缺少 `imageRewrite.sourcePrefix`、缺少 `imageRewrite.targetPrefix`、非法 `unmatchedPolicy`、非法 `digestPolicy` 均返回 `400 code=1000` 且 message 包含 `ModifierRuleRejected`；不再返回 `unsupported action=rewriteImage`。
