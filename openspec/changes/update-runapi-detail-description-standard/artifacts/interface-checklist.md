# RunAPI 接口逐项处理清单

> 2026-05-16 API 响应消息国际化备注：本地 server 与 OpenAPI 已声明
> `X-Language`、`Accept-Language`、`Content-Language`、`Vary` 和
> `message_key`。Apipost 项目 `5650333c5c52000` 已新增共享 Markdown 文档
> `API 响应消息国际化协议`，Target ID `1c8686a538401000`；`POST /login`
> 与 `POST /refresh_token` 已更新统一信封说明、新增 `X-Language` 请求头和当前错误响应示例。
> 后续逐接口详情补写时继续保留原有说明。

> 2026-05-15 CRD group 迁移备注：本地清单中的接口路径已从
> `disaster.wuxs.vip` 批量替换为 `testudo.softcdata.com`，用于对齐当前
> server 路由和 OpenAPI。Apipost MCP 已批量回写 live RunAPI 中 `api` 类型目标的
> URL 元数据；复查旧 group 后剩余项均为 `websocket2` 类型。当前 MCP
> `update_target` 对 `websocket2` 返回成功但不实际修改 URL，这部分仍需通过
> Apipost UI 或支持 websocket 目标的专用接口补迁移。本清单下方勾选项的
> “已回读验证”是此前逐接口说明补充时的历史状态。

处理规则：每完成一个接口，必须已经完成 server handler/request/response/error 取证、operator CRD/controller/status 调用链取证、RunAPI 新增或详细说明更新、RunAPI 回读验证，之后才能把该项勾选。

状态说明：
- 已存在：RunAPI 已有同方法同路由或参数化样例匹配项，目标动作是补充五段详细说明并保留原说明。
- 缺失：RunAPI 未找到对应接口，目标动作是新增接口并写入五段详细说明。
- 模块疑似错位：RunAPI 有匹配接口，但目录与 server 模块不一致，需要处理该接口时复核目录。
- 待人工复核：RunAPI 额外接口无法匹配到 server 路由，暂不勾选到 server 清单内，单独列在末尾。

## Kubernetes 资源

- [x] `GET /apis/resources.testudo.softcdata.com/v1/:resource` - RunAPI 状态：已存在；Target ID：`25559f2338c0c8`；Handler：`k.getResources`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证

## 事件与历史

- [x] `GET /apis/v1/:resource/:name/history` - RunAPI 状态：已存在；Target ID：`1be82b9e38801001`；Handler：`h.listResourceEvents`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/v1/events` - RunAPI 状态：已存在；Target ID：`1be82b9e13c01001`；Handler：`h.listEvents`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/v1/watch/:resource/:name/events` - RunAPI 状态：缺失；Target ID：`1c7aff33ad001001`；Handler：`h.watchResourceEvents`；类型：WebSocket；目标动作：新增接口并写入五段详细说明；处理状态：已完成，已新增并回读验证
- [x] `GET /apis/v1/watch/events` - RunAPI 状态：已存在；Target ID：`2bc6c1dd38c073`；Handler：`h.watchEvents`；类型：WebSocket；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证

## 删除检查

- [x] `POST /api/v1/deletion/check` - RunAPI 状态：已存在；Target ID：`1c2f51b2b9801001`；Handler：`h.check`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `POST /apis/v1/deletion/check` - RunAPI 状态：缺失；Target ID：`1c7b02d04a401001`；Handler：`h.check`；类型：HTTP；目标动作：新增接口并写入五段详细说明；处理状态：已完成，已新增并回读验证

## 存储仓库

- [x] `GET /apis/storage.testudo.softcdata.com/v1/storages` - RunAPI 状态：已存在；Target ID：`25559f2238c0b6`；Handler：`c.storages`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `POST /apis/storage.testudo.softcdata.com/v1/storages` - RunAPI 状态：已存在；Target ID：`3ee6850638c06f`；Handler：`c.createStorage`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `DELETE /apis/storage.testudo.softcdata.com/v1/storages/:name` - RunAPI 状态：已存在；Target ID：`25559f2338c0c2`；Handler：`c.deleteStorage`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/storage.testudo.softcdata.com/v1/storages/:name` - RunAPI 状态：已存在；Target ID：`25559f2238c0b8`；Handler：`c.storage`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证；备注：候选 `1be2230920001001` 实际为 `GET /apis/storage.testudo.softcdata.com/v1/storages/names`
- [x] `PATCH /apis/storage.testudo.softcdata.com/v1/storages/:name` - RunAPI 状态：已存在；Target ID：`1bd50e8b7d801000`；Handler：`c.patchStorage`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `PUT /apis/storage.testudo.softcdata.com/v1/storages/:name` - RunAPI 状态：已存在；Target ID：`3ee6850678c071`；Handler：`c.updateStorage`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/storage.testudo.softcdata.com/v1/storages/:name/validate` - RunAPI 状态：已存在；Target ID：`1bdf92f4e4801000`；Handler：`c.validateStorage`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `POST /apis/storage.testudo.softcdata.com/v1/storages/connectivity/validate` - RunAPI 状态：已存在；Target ID：`1bfa8b208a801001`；Handler：`c.validateBSLConnectivity`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/storage.testudo.softcdata.com/v1/storages/names` - RunAPI 状态：已存在；Target ID：`1be2230920001001`；Handler：`c.storageNames`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `POST /apis/storage.testudo.softcdata.com/v1/storages/validate/connection` - RunAPI 状态：已存在；Target ID：`6c722f5f8c74b`；Handler：`c.validateS3Connection`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/storage.testudo.softcdata.com/v1/watch/storages` - RunAPI 状态：已存在；Target ID：`263ef8a0b8c07c`；Handler：`c.watchStorages`；类型：WebSocket；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证

## 容灾任务

- [x] `GET /apis/disasterjobs.testudo.softcdata.com/v1/jobs` - RunAPI 状态：已存在；Target ID：`25559f21f8c0a5`；Handler：`c.configs`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `POST /apis/disasterjobs.testudo.softcdata.com/v1/jobs` - RunAPI 状态：已存在；Target ID：`25559f21f8c0aa`；Handler：`c.createConfig`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `DELETE /apis/disasterjobs.testudo.softcdata.com/v1/jobs/:name` - RunAPI 状态：已存在；Target ID：`25559f2238c0ad`；Handler：`c.deleteConfig`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/disasterjobs.testudo.softcdata.com/v1/jobs/:name` - RunAPI 状态：已存在；Target ID：`25559f21f8c0a7`；Handler：`c.config`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/disasterjobs.testudo.softcdata.com/v1/watch/jobs` - RunAPI 状态：已存在；Target ID：`25559f2238c0b0`；Handler：`c.watchJobs`；类型：WebSocket；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证；备注：RunAPI 历史目标类型仍为 `api`，已按 WebSocket 写入详细说明
- [x] `GET /apis/disasterjobs.testudo.softcdata.com/v1/watch/jobs/:name` - RunAPI 状态：已存在；Target ID：`25559f2238c0b2`；Handler：`c.watchJob`；类型：WebSocket；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证；备注：RunAPI 历史目标类型仍为 `api`，已按 WebSocket 写入详细说明

## 容灾备份

- [x] `GET /apis/disasterbackups.testudo.softcdata.com/v1/backups` - RunAPI 状态：缺失；Target ID：`1c7b0e63d9401001`；Handler：`c.backups`；类型：HTTP；目标动作：新增接口并写入五段详细说明；处理状态：已完成，已新增并回读验证
- [x] `POST /apis/disasterbackups.testudo.softcdata.com/v1/backups` - RunAPI 状态：缺失；Target ID：`1c7b0ed711801001`；Handler：`c.createBackup`；类型：HTTP；目标动作：新增接口并写入五段详细说明；处理状态：已完成，已新增并回读验证
- [x] `DELETE /apis/disasterbackups.testudo.softcdata.com/v1/backups/:name` - RunAPI 状态：缺失；Target ID：`1c7b0f565a801001`；Handler：`c.deleteBackup`；类型：HTTP；目标动作：新增接口并写入五段详细说明；处理状态：已完成，已新增并回读验证
- [x] `GET /apis/disasterbackups.testudo.softcdata.com/v1/backups/:name` - RunAPI 状态：缺失；Target ID：`1c7b0f91e8c01001`；Handler：`c.backup`；类型：HTTP；目标动作：新增接口并写入五段详细说明；处理状态：已完成，已新增并回读验证
- [x] `PUT /apis/disasterbackups.testudo.softcdata.com/v1/backups/:name` - RunAPI 状态：缺失；Target ID：`1c7b0fcf32001001`；Handler：`c.updateBackup`；类型：HTTP；目标动作：新增接口并写入五段详细说明；处理状态：已完成，已新增并回读验证
- [x] `GET /apis/disasterbackups.testudo.softcdata.com/v1/watch/backups` - RunAPI 状态：缺失；Target ID：`6345b0ecacc6000`；Handler：`c.watchBackups`；类型：WebSocket；目标动作：新增接口并写入五段详细说明；处理状态：已完成，已新增并回读验证
- [x] `GET /apis/disasterbackups.testudo.softcdata.com/v1/watch/backups/:name` - RunAPI 状态：缺失；Target ID：`6345b54530c6000`；Handler：`c.watchBackup`；类型：WebSocket；目标动作：新增接口并写入五段详细说明；处理状态：已完成，已新增并回读验证

## 容灾实例

- [x] `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances` - RunAPI 状态：已存在；Target ID：`1c01f46e4f001000`；Handler：`h.listInstances`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证；2026-05-21 已修正 `query.namespace` 为实例保护的业务 namespace 过滤，不再表示 CR 存储 namespace；2026-05-25 已改为支持 namespace 片段模糊搜索
- [x] `POST /apis/disasterinstances.testudo.softcdata.com/v1/instances` - RunAPI 状态：已存在；Target ID：`1c01f46f55801000`；Handler：`h.createInstance`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `DELETE /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name` - RunAPI 状态：已存在；Target ID：`1c01f472a2c01000`；Handler：`h.deleteInstance`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name` - RunAPI 状态：已存在；Target ID：`1c01f470ba401000`；Handler：`h.getInstance`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `PUT /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name` - RunAPI 状态：已存在；Target ID：`1c01f4718c001000`；Handler：`h.updateInstance`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `POST /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/actions` - RunAPI 状态：已存在；Target ID：`1c01fb364ac01000`；Handler：`h.executeAction`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证；2026-05-18 已补充可恢复同步失败 Failed 下 sync-data/sync-resource 重试兼容语义；2026-05-22 已补充 FailingOver 下 cancel 兼容语义，本地证据已更新，RunAPI live 已补充 202 响应示例
- [x] `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/groups` - RunAPI 状态：缺失；Target ID：`1c7b14c8c7401001`；Handler：`h.getInstanceGroups`；类型：HTTP；目标动作：新增接口并写入五段详细说明；处理状态：已完成，已新增并回读验证
- [x] `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/history` - RunAPI 状态：已存在；Target ID：`1c01f7948bc01000`；Handler：`h.getHistory`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/operations/:operationName` - RunAPI 状态：已存在；Target ID：`1c5ead7bab401001`；Handler：`h.getOperationDetail`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `POST /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/restore-classes/validate` - RunAPI 状态：已存在；Target ID：`1c49f6d741801001`；Handler：`h.validateRestoreClasses`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/sync-status` - RunAPI 状态：已存在；Target ID：`1c01f74a5e401000`；Handler：`h.getSyncStatus`；类型：HTTP；目标动作：补充 `lastSyncStatus` 字段说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/sync-history` - RunAPI 状态：缺失，已新增；Target ID：`1c8428770dc01001`；Handler：`h.getSyncHistory`；类型：HTTP；目标动作：新增同步历史接口并写入五段详细说明；处理状态：已完成，已新增并回读验证
- [x] `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/validate-target` - RunAPI 状态：缺失；Target ID：`1c7b171878001001`；Handler：`h.validateTarget`；类型：HTTP；目标动作：新增接口并写入五段详细说明；处理状态：已完成，已新增并回读验证；2026-05-18 已补充可恢复同步失败 Failed 下 sync-data/sync-resource 重试兼容语义；2026-05-22 已补充 FailingOver 下 cancel 兼容语义，本地证据已更新，RunAPI live 已补充 200 响应示例并更新 operation query 说明
- [x] `GET /apis/disasterinstances.testudo.softcdata.com/v1/watch/instances` - RunAPI 状态：已存在；Target ID：`3b8699c1b8c14b`；Handler：`h.watchInstances`；类型：WebSocket；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/disasterinstances.testudo.softcdata.com/v1/watch/instances/:name` - RunAPI 状态：已存在；Target ID：`3b86fa34f8c221`；Handler：`h.watchInstance`；类型：WebSocket；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/disasterinstances.testudo.softcdata.com/v1/watch/instances/operations/:operationName` - RunAPI 状态：已存在；Target ID：`1c5eadf4f1801001`；Handler：`h.watchOperation`；类型：WebSocket；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证

## 容灾演练

- [x] `GET /apis/disasterdrills.testudo.softcdata.com/v1/drills` - RunAPI 状态：已存在；Target ID：`1c0aec042dc01001`；Handler：`h.listDrills`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `POST /apis/disasterdrills.testudo.softcdata.com/v1/drills` - RunAPI 状态：已存在；Target ID：`1c0aec0a8f401001`；Handler：`h.createDrill`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `DELETE /apis/disasterdrills.testudo.softcdata.com/v1/drills/:name` - RunAPI 状态：已存在；Target ID：`1c0aec15e5001001`；Handler：`h.deleteDrill`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/disasterdrills.testudo.softcdata.com/v1/drills/:name` - RunAPI 状态：已存在；Target ID：`1c0aec0a6cc01001`；Handler：`h.getDrill`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `POST /apis/disasterdrills.testudo.softcdata.com/v1/drills/:name/cleanup` - RunAPI 状态：已存在；Target ID：`1c28bf2907c01000`；Handler：`h.cleanupDrill`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `POST /apis/disasterdrills.testudo.softcdata.com/v1/drills/:name/confirm` - RunAPI 状态：已存在；Target ID：`1c0aec15bc001001`；Handler：`h.confirmDrill`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `POST /apis/disasterdrills.testudo.softcdata.com/v1/drills/:name/restart` - RunAPI 状态：已存在；Target ID：`1c0c8731f1401001`；Handler：`h.restartDrill`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/disasterdrills.testudo.softcdata.com/v1/drills/actions/protected-namespaces` - RunAPI 状态：已存在；Target ID：`1c380061ca801001`；Handler：`h.getProtectedNamespaces`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证；2026-05-21 已修正 namespace 为空时使用配置的管理命名空间且不再全 namespace 回退查找实例/组
- [x] `GET /apis/disasterdrills.testudo.softcdata.com/v1/watch/drills` - RunAPI 状态：已存在；Target ID：`477a33c78c1c8`；Handler：`h.watchDrills`；类型：WebSocket；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/disasterdrills.testudo.softcdata.com/v1/watch/drills/:name` - RunAPI 状态：已存在；Target ID：`477bda678c1cb`；Handler：`h.watchDrill`；类型：WebSocket；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证

## 容灾策略

- [x] `GET /apis/policies.testudo.softcdata.com/v1/policies` - RunAPI 状态：已存在；Target ID：`35ec1a7278c00b`；Handler：`h.policies`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `POST /apis/policies.testudo.softcdata.com/v1/policies` - RunAPI 状态：已存在；Target ID：`3ee6850638c06a`；Handler：`h.createPolicy`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `DELETE /apis/policies.testudo.softcdata.com/v1/policies/:name` - RunAPI 状态：已存在；Target ID：`35ec1a7378c012`；Handler：`h.deletePolicy`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/policies.testudo.softcdata.com/v1/policies/:name` - RunAPI 状态：已存在；Target ID：`35ec1a72b8c00c`（原候选 `1be2230900401001` 实为 `GET /policies/names`）；Handler：`h.policy`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `PUT /apis/policies.testudo.softcdata.com/v1/policies/:name` - RunAPI 状态：已存在；Target ID：`3ee6850638c06c | 1c721610dc401001`（同一路由两条 RunAPI 记录，通用更新与 AutoBackup 专项说明）；Handler：`h.updatePolicy`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/policies.testudo.softcdata.com/v1/policies/names` - RunAPI 状态：已存在；Target ID：`1be2230900401001`；Handler：`h.policyNames`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证

## 容灾组

- [x] `GET /apis/disastergroups.testudo.softcdata.com/v1/groups` - RunAPI 状态：已存在；Target ID：`1c048e6aa6401000`；Handler：`h.listGroups`；类型：HTTP；目标动作：补充 `meta.summary.abnormalCount` 与 `status=error` 语义并保留原说明；处理状态：已完成，已回读验证
- [x] `POST /apis/disastergroups.testudo.softcdata.com/v1/groups` - RunAPI 状态：已存在；Target ID：`1c048e6950c01000`；Handler：`h.createGroup`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `DELETE /apis/disastergroups.testudo.softcdata.com/v1/groups/:name` - RunAPI 状态：已存在；Target ID：`1c04bcb222801001`；Handler：`h.deleteGroup`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/disastergroups.testudo.softcdata.com/v1/groups/:name` - RunAPI 状态：已存在；Target ID：`1c048e6d34001000`（原候选 `1c20cb43ab001001` 实为 `GET /groups/instance-picker`）；Handler：`h.getGroup`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `PATCH /apis/disastergroups.testudo.softcdata.com/v1/groups/:name` - RunAPI 状态：原缺失，已新增；Target ID：`1c7b252d8bc01001`；Handler：`h.updateGroup`；类型：HTTP；目标动作：新增接口并写入五段详细说明；处理状态：已完成，已回读验证
- [x] `PUT /apis/disastergroups.testudo.softcdata.com/v1/groups/:name` - RunAPI 状态：已存在；Target ID：`1c239548ddc01000`；Handler：`h.updateGroup`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `POST /apis/disastergroups.testudo.softcdata.com/v1/groups/:name/actions` - RunAPI 状态：已存在；Target ID：`1c048e6d67c01000`；Handler：`h.executeAction`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/disastergroups.testudo.softcdata.com/v1/groups/:name/history` - RunAPI 状态：已存在；Target ID：`1c04e21a5c801001`；Handler：`h.getHistory`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/disastergroups.testudo.softcdata.com/v1/groups/:name/instances` - RunAPI 状态：已存在；Target ID：`1c2100948fc01001`；Handler：`h.listGroupInstances`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/disastergroups.testudo.softcdata.com/v1/groups/:name/operations/:operationName` - RunAPI 状态：已存在；Target ID：`1c5ead7bcb801001`；Handler：`h.getOperationDetail`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/disastergroups.testudo.softcdata.com/v1/groups/instance-picker` - RunAPI 状态：已存在；Target ID：`1c20cb43ab001001`；Handler：`h.instancePicker`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/disastergroups.testudo.softcdata.com/v1/watch/groups/operations` - RunAPI 状态：已存在；Target ID：`1a44c9e9f8c0a8`；Handler：`h.watchGroupOperations`；类型：WebSocket；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/disastergroups.testudo.softcdata.com/v1/watch/groups/operations/:operationName` - RunAPI 状态：已存在；Target ID：`1c5eadf517801001 | 1a44ffeb38c0ab`；Handler：`h.watchGroupOperation`；类型：WebSocket；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/disastergroups.testudo.softcdata.com/v1/watch/groups/status` - RunAPI 状态：已存在；Target ID：`39a5775f8c2ec`；Handler：`h.watchGroupStatuses`；类型：WebSocket；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/disastergroups.testudo.softcdata.com/v1/watch/groups/status/:name` - RunAPI 状态：已存在；Target ID：`39a699978c2f0`；Handler：`h.watchGroupStatus`；类型：WebSocket；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证

## 容灾配置

- [x] `GET /apis/disasterconfigs.testudo.softcdata.com/v1/configs` - RunAPI 状态：已存在；Target ID：`25559f2138c090`；Handler：`c.configs`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `POST /apis/disasterconfigs.testudo.softcdata.com/v1/configs` - RunAPI 状态：已存在；Target ID：`3ee68505b8c065`；Handler：`c.createConfig`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `DELETE /apis/disasterconfigs.testudo.softcdata.com/v1/configs/:name` - RunAPI 状态：已存在；Target ID：`25559f21b8c098`；Handler：`c.deleteConfig`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/disasterconfigs.testudo.softcdata.com/v1/configs/:name` - RunAPI 状态：已存在；Target ID：`25559f2138c092`（原候选 `1c304687b0801001` 实为 `GET /configs/names`）；Handler：`c.config`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `PUT /apis/disasterconfigs.testudo.softcdata.com/v1/configs/:name` - RunAPI 状态：已存在；Target ID：`3ee68505f8c067`；Handler：`c.updateConfig`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/disasterconfigs.testudo.softcdata.com/v1/configs/names` - RunAPI 状态：已存在；Target ID：`1c304687b0801001`；Handler：`c.configNames`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/disasterconfigs.testudo.softcdata.com/v1/watch/configs` - RunAPI 状态：已存在；Target ID：`267c3f73f8c40a | 3b831dfe78c08c`；Handler：`c.watchConfigs`；类型：WebSocket；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/disasterconfigs.testudo.softcdata.com/v1/watch/configs/:name` - RunAPI 状态：已存在；Target ID：`267c641278c417`；Handler：`c.watchConfig`；类型：WebSocket；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证

## 应用备份

- [x] `GET /apis/appbackups.testudo.softcdata.com/v1/appbackups` - RunAPI 状态：已存在；Target ID：`2a25b664f8c042`；Handler：`c.appBackups`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `POST /apis/appbackups.testudo.softcdata.com/v1/appbackups` - RunAPI 状态：已存在；Target ID：`3ee68503b8c051 | 3efbb43238c57f`；Handler：`c.createAppBackup`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `DELETE /apis/appbackups.testudo.softcdata.com/v1/appbackups/:name` - RunAPI 状态：已存在；Target ID：`2a25b66578c049`；Handler：`c.deleteAppBackup`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/appbackups.testudo.softcdata.com/v1/appbackups/:name` - RunAPI 状态：已存在；Target ID：`2a25b664f8c043`；Handler：`c.appBackup`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证；备注：候选 `1be2239988801001` 实际为 `GET /apis/appbackups.testudo.softcdata.com/v1/appbackups/clusters`
- [x] `PUT /apis/appbackups.testudo.softcdata.com/v1/appbackups/:name` - RunAPI 状态：已存在；Target ID：`3ee6850438c053`；Handler：`c.updateAppBackup`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `POST /apis/appbackups.testudo.softcdata.com/v1/appbackups/:name/actions/:type` - RunAPI 状态：已存在；Target ID：`1bd135d4c5c01001 | 1bd135d316401001 | 3f0947a838c682 | 3c1cb82178c001 | 1bd135d64b401001`；Handler：`c.executeAction`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/appbackups.testudo.softcdata.com/v1/appbackups/:name/backups/:backupName/download` - RunAPI 状态：已存在；Target ID：`1bd135d66a001001`；Handler：`c.downloadBackup`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/appbackups.testudo.softcdata.com/v1/appbackups/:name/history` - RunAPI 状态：已存在；Target ID：`1bd4dbb7f4401001`；Handler：`c.getBackupHistory`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/appbackups.testudo.softcdata.com/v1/appbackups/clusters` - RunAPI 状态：已存在；Target ID：`1be2239988801001`；Handler：`c.getAppBackupClusters`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/appbackups.testudo.softcdata.com/v1/velero/backups/:backupName/includes` - RunAPI 状态：已存在；Target ID：`1c30a700d3001001`；Handler：`c.getVeleroBackupIncludes`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/appbackups.testudo.softcdata.com/v1/watch/appbackups` - RunAPI 状态：已存在；Target ID：`2a418fe378c0a9`；Handler：`c.watchAppBackups`；类型：WebSocket；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/appbackups.testudo.softcdata.com/v1/watch/appbackups/:name` - RunAPI 状态：已存在；Target ID：`2aab0af7b0c0a6`；Handler：`c.watchAppBackup`；类型：WebSocket；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证

## 应用恢复

- [x] `GET /apis/apprestores.testudo.softcdata.com/v1/apprestores` - RunAPI 状态：已存在；Target ID：`34a0ae4978c001`；Handler：`c.appRestores`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `POST /apis/apprestores.testudo.softcdata.com/v1/apprestores` - RunAPI 状态：已存在；Target ID：`3ee6850478c056`；Handler：`c.createAppRestore`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `DELETE /apis/apprestores.testudo.softcdata.com/v1/apprestores/:name` - RunAPI 状态：已存在；Target ID：`34a0ae4b38c008`；Handler：`c.deleteAppRestore`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/apprestores.testudo.softcdata.com/v1/apprestores/:name` - RunAPI 状态：已存在；Target ID：`34a0ae49b8c002`；Handler：`c.appRestore`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `PUT /apis/apprestores.testudo.softcdata.com/v1/apprestores/:name` - RunAPI 状态：已存在；Target ID：`3ee68504b8c058`；Handler：`c.updateAppRestore`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `POST /apis/apprestores.testudo.softcdata.com/v1/apprestores/:name/actions/:type` - RunAPI 状态：已存在；Target ID：`3c211cc3f8c006 | 3c211cc3f8c008`；Handler：`c.executeAction`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，两个目标均已回读验证
- [x] `POST /apis/apprestores.testudo.softcdata.com/v1/apprestores/preflight/validate` - RunAPI 状态：已存在；Target ID：`1c385ce4c7801001`；Handler：`c.validateRestorePreflight`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/apprestores.testudo.softcdata.com/v1/watch/apprestores` - RunAPI 状态：已存在；Target ID：`34a10209f8c00c`；Handler：`c.watchAppRestores`；类型：WebSocket；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/apprestores.testudo.softcdata.com/v1/watch/apprestores/:name` - RunAPI 状态：已存在；Target ID：`34a12bb3b8c041`；Handler：`c.watchAppRestore`；类型：WebSocket；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证

## 根接口

- [x] `GET /healthz` - RunAPI 状态：已存在；Target ID：`25559f1e78c062`；Handler：`func(ctx context.Context, c *app.RequestContext`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `POST /login` - RunAPI 状态：已存在；Target ID：`25559f1e78c064`；Handler：`jwtMiddleware.LoginHandler`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /readyz` - RunAPI 状态：已存在；Target ID：`25559f1e78c063`；Handler：`func(ctx context.Context, c *app.RequestContext`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `POST /refresh_token` - RunAPI 状态：已存在；Target ID：`1bf8e6af53c01001`；Handler：`middleware.RefreshTokenHandler`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证

## 用户管理

- [x] `GET /api/v1/users` - RunAPI 状态：已存在；Target ID：`1c4ce9c769001001`；Handler：`h.listUsers`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `POST /api/v1/users` - RunAPI 状态：已存在；Target ID：`1c4cd93b5fc01001`；Handler：`h.createUser`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `DELETE /api/v1/users/:username` - RunAPI 状态：已存在；Target ID：`1c4ce9c787001001`；Handler：`h.deleteUser`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `PATCH /api/v1/users/:username/password` - RunAPI 状态：已存在；Target ID：`1c4ce9c7a7401001`；Handler：`h.patchUserPassword`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `PATCH /api/v1/users/:username/status` - RunAPI 状态：已存在；Target ID：`1c4cd93b7b001001`；Handler：`h.patchUserStatus`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/v1/users` - RunAPI 状态：缺失，已新增；Target ID：`1c7b437084c01001`；Handler：`h.listUsers`；类型：HTTP；目标动作：新增接口并写入五段详细说明；处理状态：已完成，已新增并回读验证
- [x] `POST /apis/v1/users` - RunAPI 状态：缺失，已新增；Target ID：`1c7b43df0d401000`；Handler：`h.createUser`；类型：HTTP；目标动作：新增接口并写入五段详细说明；处理状态：已完成，已新增并回读验证
- [x] `DELETE /apis/v1/users/:username` - RunAPI 状态：缺失，已新增；Target ID：`1c7b440fdec01001`；Handler：`h.deleteUser`；类型：HTTP；目标动作：新增接口并写入五段详细说明；处理状态：已完成，已新增并回读验证
- [x] `PATCH /apis/v1/users/:username/password` - RunAPI 状态：缺失，已新增；Target ID：`1c7b4441e6801001`；Handler：`h.patchUserPassword`；类型：HTTP；目标动作：新增接口并写入五段详细说明；处理状态：已完成，已新增并回读验证
- [x] `PATCH /apis/v1/users/:username/status` - RunAPI 状态：缺失，已新增；Target ID：`1c7b4474d2001001`；Handler：`h.patchUserStatus`；类型：HTTP；目标动作：新增接口并写入五段详细说明；处理状态：已完成，已新增并回读验证

## 系统设置

- [x] `GET /api/v1/system-settings` - RunAPI 状态：已存在；Target ID：`1c31b7bf28801000`；Handler：`h.listSettings`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `POST /api/v1/system-settings` - RunAPI 状态：已存在；Target ID：`1c31b7bf87401001`；Handler：`h.createSetting`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `DELETE /api/v1/system-settings/:config_key` - RunAPI 状态：已存在；Target ID：`1c31b7ca57001001`；Handler：`h.deleteSetting`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `PUT /api/v1/system-settings/:config_key` - RunAPI 状态：已存在；Target ID：`1c31b7bfa1c01001`；Handler：`h.updateSetting`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /api/v1/system-settings/assets/:config_key` - RunAPI 状态：已存在；Target ID：`1c31b7ca8b001000`；Handler：`h.getAsset`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `POST /api/v1/system-settings/assets/:config_key` - RunAPI 状态：已存在；Target ID：`1c31b7ca73001000`；Handler：`h.uploadAsset`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /api/v1/system-settings/public` - RunAPI 状态：已存在；Target ID：`1c31b7bf6e801000`；Handler：`h.listPublicSettings`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/v1/system-settings` - RunAPI 状态：缺失，已新增；Target ID：`1c7b469bcbc01001`；Handler：`h.listSettings`；类型：HTTP；目标动作：新增接口并写入五段详细说明；处理状态：已完成，已新增并回读验证
- [x] `POST /apis/v1/system-settings` - RunAPI 状态：缺失，已新增；Target ID：`1c7b46cc9d801001`；Handler：`h.createSetting`；类型：HTTP；目标动作：新增接口并写入五段详细说明；处理状态：已完成，已新增并回读验证
- [x] `DELETE /apis/v1/system-settings/:config_key` - RunAPI 状态：缺失，已新增；Target ID：`1c7b470b1b001000`；Handler：`h.deleteSetting`；类型：HTTP；目标动作：新增接口并写入五段详细说明；处理状态：已完成，已新增并回读验证
- [x] `PUT /apis/v1/system-settings/:config_key` - RunAPI 状态：缺失，已新增；Target ID：`1c7b474554801001`；Handler：`h.updateSetting`；类型：HTTP；目标动作：新增接口并写入五段详细说明；处理状态：已完成，已新增并回读验证
- [x] `GET /apis/v1/system-settings/assets/:config_key` - RunAPI 状态：缺失，已新增；Target ID：`1c7b47c5ea801001`；Handler：`h.getAsset`；类型：HTTP；目标动作：新增接口并写入五段详细说明；处理状态：已完成，已新增并回读验证
- [x] `POST /apis/v1/system-settings/assets/:config_key` - RunAPI 状态：缺失，已新增；Target ID：`1c7b4783d4401001`；Handler：`h.uploadAsset`；类型：HTTP；目标动作：新增接口并写入五段详细说明；处理状态：已完成，已新增并回读验证
- [x] `GET /apis/v1/system-settings/public` - RunAPI 状态：缺失，已新增；Target ID：`1c7b4869a4801001`；Handler：`h.listPublicSettings`；类型：HTTP；目标动作：新增接口并写入五段详细说明；处理状态：已完成，已新增并回读验证

## 统计

- [x] `GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/autobackups/execution-summary` - RunAPI 状态：已存在；Target ID：`1c7215aa3e401001`；Handler：`h.GetAutoBackupExecutionSummary`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/backups` - RunAPI 状态：已存在；Target ID：`1a7322a38c001`；Handler：`h.GetBackupStatistics`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/backups/success-rate` - RunAPI 状态：已存在；Target ID：`1c239ca7d4801000`；Handler：`h.GetBackupSuccessRate`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/instances` - RunAPI 状态：已存在；Target ID：`1c239ac3a6001000`；Handler：`h.GetInstanceStatistics`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/operations` - RunAPI 状态：已存在；Target ID：`1c1fe2a799801001`；Handler：`h.GetOperationStatistics`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/operations/by-time` - RunAPI 状态：已存在；Target ID：`1c2398ad1d001000`；Handler：`h.GetOperationStatisticsByTime`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/restores` - RunAPI 状态：已存在；Target ID：`1a7322a38c003`；Handler：`h.GetRestoreStatistics`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/storage` - RunAPI 状态：已存在；Target ID：`1c2620b6d7801001`；Handler：`h.GetStorageStatistics`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/tasks/progress` - RunAPI 状态：已存在；Target ID：`1c71f52b8b401001`；Handler：`h.GetTaskProgressTrend`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证

## 集群管理

- [x] `GET /apis/cluster.testudo.softcdata.com/v1/clusters` - RunAPI 状态：已存在；Target ID：`25559f1f78c067`；Handler：`c.clusters`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `POST /apis/cluster.testudo.softcdata.com/v1/clusters` - RunAPI 状态：已存在；Target ID：`3ee6850538c060 | 69ee1a0f8c0b7`；Handler：`c.createCluster`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `DELETE /apis/cluster.testudo.softcdata.com/v1/clusters/:name` - RunAPI 状态：已存在；Target ID：`25559f2038c06f`；Handler：`c.deleteCluster`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/cluster.testudo.softcdata.com/v1/clusters/:name` - RunAPI 状态：已存在；Target ID：`25559f1fb8c069`；Handler：`c.cluster`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证（`1be22307d0001001` 经回读确认属于 `GET /clusters/names`）
- [x] `PATCH /apis/cluster.testudo.softcdata.com/v1/clusters/:name` - RunAPI 状态：已存在；Target ID：`1bd502f9fe001000`；Handler：`c.patchCluster`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `POST /apis/cluster.testudo.softcdata.com/v1/clusters/:name/actions/refresh-namespaces` - RunAPI 状态：已存在；Target ID：`1c613a8d76001001`；Handler：`c.refreshNamespaces`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/cluster.testudo.softcdata.com/v1/clusters/:name/protected-namespaces` - RunAPI 状态：已存在；Target ID：`1c61734794401001`；Handler：`c.protectedNamespaces`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/cluster.testudo.softcdata.com/v1/clusters/:name/restore-classes` - RunAPI 状态：已存在；Target ID：`1c4bb4e50d001001`；Handler：`c.listRestoreClasses`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/cluster.testudo.softcdata.com/v1/clusters/:name/validate` - RunAPI 状态：已存在；Target ID：`25559f2038c072`；Handler：`c.validateCluster`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `POST /apis/cluster.testudo.softcdata.com/v1/clusters/kubeconfig/validate` - RunAPI 状态：已存在；Target ID：`67d3bab78c00b | 69dbb8ef8c07f`；Handler：`c.validateKubeConfig`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/cluster.testudo.softcdata.com/v1/clusters/names` - RunAPI 状态：已存在；Target ID：`1be22307d0001001`；Handler：`c.clusterNames`；类型：HTTP；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/cluster.testudo.softcdata.com/v1/watch/clusters` - RunAPI 状态：已存在；Target ID：`2667344638c320`；Handler：`c.watchClusters`；类型：WebSocket；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证
- [x] `GET /apis/cluster.testudo.softcdata.com/v1/watch/clusters/:name` - RunAPI 状态：已存在；Target ID：`2667d19538c36d`；Handler：`c.watchCluster`；类型：WebSocket；目标动作：补充五段详细说明并保留原说明；处理状态：已完成，已回读验证

## 平台许可

- [x] `GET /apis/v1/platform-license/status` - RunAPI 状态：已存在；Target ID：`1c7dbf4632001001`；Handler：`h.status`；类型：HTTP；目标动作：补充 `LicenseEnvironmentInvalid`、内容错误优先级、liveEvaluation 语义和当前指纹返回字段；处理状态：已完成，已新增“免费版状态含当前指纹”示例并回读验证
- [x] `POST /apis/v1/platform-license/install` - RunAPI 状态：已存在；Target ID：`1c7dbf57bd001001`；Handler：`h.install`；类型：HTTP；目标动作：补充写入后基于当前 Secret/本次内容实时评价、避免旧 `NoLicense`、内容错误优先级和安装响应当前指纹字段；处理状态：已完成，已新增“安装 malformed License 含当前指纹”示例并回读验证

## RunAPI 额外/待人工复核

- [x] `POST /disasterjobs.testudo.softcdata.com/v1/jobs` - RunAPI 状态：待人工复核；Target ID：`3ee6850678c074`；目录：容灾云平台 / V1 / 任务；名称：创建容灾任务；目标动作：确认是否为废弃接口、路径漏 `/apis`，或 server 清单需要补录；处理状态：已复核，确认为 RunAPI 额外历史目标，server 无该裸路径路由；正确 `/apis` 目标已处理，建议人工删除或迁移

## 全局响应约定

- [x] `server DTO 时间字段本地时区序列化` - RunAPI 状态：[已新增 Apipost Markdown 说明文档，Target ID `1c8551dde7001000`]；OpenAPI：[已补全局 LocalRFC3339Time 说明与 info.description]；Schema：[响应 DTO 时间字段仍为 string/date-time，语义改为按 time.Local/TZ 输出 RFC3339 offset]；错误：[不涉及错误码]；operator：[不改 CRD 存储，kubectl YAML 仍为 UTC Z]
- [x] `管理面 namespace 可配置` - RunAPI 状态：[未逐个 live 批量更新，已记录待同步]；OpenAPI：[已将固定 `disaster-system` 资源范围描述改为配置的管理命名空间]；Schema：[不改变响应结构]；错误：[非默认 namespace 下避免错误访问缺失的 `disaster-system`]；operator：[新增 `--management-namespace` 并修正 StorageRepository/license/event/backfill 链路]

## Velero Hook 透传

- [x] `GET/POST/PUT/GET history /apis/appbackups.testudo.softcdata.com/v1/appbackups...` - RunAPI 状态：[已回读并新增 live 示例；共享文档 Target ID `1ca8437f2a401000`；API Target ID：`2a25b664f8c042`、`3ee68503b8c051`、`3efbb43238c57f`、`2a25b664f8c043`、`3ee6850438c053`、`1bd4dbb7f4401001`]；OpenAPI：[已新增 hooks request/response schema 与 history hookStatus]；Schema：[AppBackupCreateRequest/AppBackupUpdateRequest/AppBackupDTO/BackupRecordDTO]；错误：[VeleroHookSensitiveParameter、timeout 上限、pod scope 校验]；operator：[AppBackup template.hooks 继续透传 Velero Backup/Schedule]
- [x] `GET/POST/PUT/watch /apis/apprestores.testudo.softcdata.com/v1/apprestores...` - RunAPI 状态：[已回读并新增 live 示例；共享文档 Target ID `1ca8437f2a401000`；API Target ID：`34a0ae4978c001`、`3ee6850478c056`、`34a0ae49b8c002`、`3ee68504b8c058`、`34a10209f8c00c`、`34a12bb3b8c041`]；OpenAPI：[已新增 restore hooks request/response schema，并声明 `status.phase=PartiallyFailed` 为非成功终态]；Schema：[AppRestoreCreateRequest/AppRestoreUpdateRequest/AppRestoreDTO/WatchEventDTO]；错误：[VeleroHookSensitiveParameter、exec/wait/init timeout 上限、pod scope 校验]；operator：[AppRestore template.hooks 继续透传 Velero Restore；Velero Restore `PartiallyFailed` 映射为 AppRestore 顶层 `PartiallyFailed`]；PVC 清理：[restorePVs/cleanVolumes 自动规则使用 `add /spec/volumeName` 空值并替换 legacy remove]
- [x] `POST/PUT /apis/disasterinstances.testudo.softcdata.com/v1/instances...` - RunAPI 状态：[已回读并新增 live 示例；共享文档 Target ID `1ca8437f2a401000`；API Target ID：`1c01f46f55801000`、`1c01f4718c001000`]；OpenAPI：[已新增 veleroHooks 字段和 presence/clear 说明]；Schema：[DisasterVeleroHooks]；错误：[VeleroHookSensitiveParameter、timeout 上限、Hook 结构非法]；operator：[DataSync 投影 dataBackup/dataRestore，既有 ds-* AppBackup 对齐 hooks]
- [x] `GET sync-status/sync-history /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/...` - RunAPI 状态：[已回读并新增 hookStatus live 示例；共享文档 Target ID `1ca8437f2a401000`；API Target ID：`1c01f74a5e401000`、`1c8428770dc01001`]；OpenAPI：[已新增 backupHookStatus/restoreHookStatus]；Schema：[LastSyncStatusDTO/SyncHistoryItemDTO/SyncHistoryHookStatusDTO]；错误：[不新增错误码]；operator：[SyncHistoryRecord 从 Velero hookStatus 复制 attempted/failed]
- [x] `POST /apis/disasterdrills.testudo.softcdata.com/v1/drills` - RunAPI 状态：[已回读并新增 dataRestore hook、禁止 dataBackup、敏感参数拒绝 live 示例；共享文档 Target ID `1ca8437f2a401000`；API Target ID `1c0aec0a8f401001`]；OpenAPI：[已新增 veleroHooks.dataRestore，说明 dataBackup 禁止，并声明 `veleroHooks:{}` 清空继承]；Schema：[DisasterDrillVeleroHooks]；错误：[出现 dataBackup 返回 400，敏感参数硬拒绝]；operator：[Drill spec veleroHooks 复制到 Operation DrillConfig，dataRestore 覆盖实例级 hooks；空 veleroHooks 复制为非 nil 空对象以阻断继承]

## 动态镜像重写 bulkModifierActions

- [x] `POST /apis/disasterinstances.testudo.softcdata.com/v1/instances` - RunAPI 状态：[已回读 Target ID `1c01f46f55801000`；因长 description 回读被截断，未覆盖原说明；已新增 `rewriteImage` 成功示例和缺少 `sourcePrefix` 的 400 示例]；OpenAPI：[已新增 `rewriteImage` action、`imageRewrite` schema 与纯 rewriteImage 不生成长期 snapshot 说明]；Schema：[DisasterInstanceBulkModifierAction/DisasterInstanceDynamicImageRewrite]；错误：[ModifierRuleRejected: imageRewrite.sourcePrefix/targetPrefix required、unmatchedPolicy、digestPolicy]；operator：[ResourceSync/Drill 运行时动态编译镜像 pair 规则，跳过 forbidden path]
- [x] `PUT /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name` - RunAPI 状态：[已回读 Target ID `1c01f4718c001000`；因长 description 回读被截断，未覆盖原说明；已新增 `rewriteImage` 成功示例和缺少 `sourcePrefix` 的 400 示例]；OpenAPI：[已更新 update restorePolicy 说明，明确 `rewriteImage` 更新时只保存 DSL、不重算长期 snapshot]；Schema：[同创建接口]；错误：[同创建接口]；operator：[后续恢复构建阶段按源真实镜像动态编译]
