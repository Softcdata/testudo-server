# RunAPI 全量接口清单

- 项目：容灾平台 (`5650333c5c52000`)
- 接口数量（api + websocket2）：144
- 说明：该清单来自 Apipost Open API `/open/apis/list`；详细说明是否为空需要逐接口详情阶段确认。

## 容灾云平台 / V1

| 类型 | 方法 | 路径 | 名称 | Target ID |
|---|---|---|---|---|
| websocket2 | `GET` | `/apis/v1/watch/events` | 获取全局事件 | `2bc6c1dd38c073` |

## 容灾云平台 / V1 / K8S资源

| 类型 | 方法 | 路径 | 名称 | Target ID |
|---|---|---|---|---|
| api | `GET` | `/apis/resources.testudo.softcdata.com/v1/:resource` | 获取 | `25559f2338c0c8` |

## 容灾云平台 / V1 / 任务

| 类型 | 方法 | 路径 | 名称 | Target ID |
|---|---|---|---|---|
| api | `GET` | `/apis/disasterjobs.testudo.softcdata.com/v1/jobs` | 列表查询 | `25559f21f8c0a5` |
| api | `POST` | `/apis/disasterjobs.testudo.softcdata.com/v1/jobs` | 创建 | `25559f21f8c0aa` |
| api | `GET` | `/apis/disasterjobs.testudo.softcdata.com/v1/jobs/:name` | 获取详情 | `25559f21f8c0a7` |
| api | `DELETE` | `/apis/disasterjobs.testudo.softcdata.com/v1/jobs/job-12` | 删除 | `25559f2238c0ad` |
| api | `GET` | `/apis/disasterjobs.testudo.softcdata.com/v1/watch/jobs` | 事件流(全部) | `25559f2238c0b0` |
| api | `GET` | `/apis/disasterjobs.testudo.softcdata.com/v1/watch/jobs/:name` | 事件流(单个) | `25559f2238c0b2` |
| api | `POST` | `/disasterjobs.testudo.softcdata.com/v1/jobs` | 创建容灾任务 | `3ee6850678c074` |

## 容灾云平台 / V1 / 健康检查与鉴权

| 类型 | 方法 | 路径 | 名称 | Target ID |
|---|---|---|---|---|
| api | `GET` | `/healthz` | 健康检查 | `25559f1e78c062` |
| api | `POST` | `/login` | 登录获取令牌 | `25559f1e78c064` |
| api | `GET` | `/readyz` | 就绪检查 | `25559f1e78c063` |
| api | `POST` | `/refresh_token` | 刷新令牌 (Refresh Token) | `1bf8e6af53c01001` |

## 容灾云平台 / V1 / 历史事件

| 类型 | 方法 | 路径 | 名称 | Target ID |
|---|---|---|---|---|
| api | `GET` | `/apis/v1/appbackups/app-backup-1768290527617ms/history` | 获取指定资源历史记录 | `1be82b9e38801001` |
| api | `GET` | `/apis/v1/events` | 获取全局历史事件记录 | `1be82b9e13c01001` |

## 容灾云平台 / V1 / 存储仓库配置管理

| 类型 | 方法 | 路径 | 名称 | Target ID |
|---|---|---|---|---|
| api | `GET` | `/apis/storage.testudo.softcdata.com/v1/storages` | 列表查询 | `25559f2238c0b6` |
| api | `POST` | `/apis/storage.testudo.softcdata.com/v1/storages` | 添加存储仓库 | `3ee6850638c06f` |
| api | `GET` | `/apis/storage.testudo.softcdata.com/v1/storages/123/validate` | 验证 | `1bdf92f4e4801000` |
| api | `POST` | `/apis/storage.testudo.softcdata.com/v1/storages/connectivity/validate` | 验证存储连通性 (BSL) | `1bfa8b208a801001` |
| api | `GET` | `/apis/storage.testudo.softcdata.com/v1/storages/names` | 获取存储名称列表 | `1be2230920001001` |
| api | `GET` | `/apis/storage.testudo.softcdata.com/v1/storages/storage-1` | 获取详情 | `25559f2238c0b8` |
| api | `PATCH` | `/apis/storage.testudo.softcdata.com/v1/storages/storage-1` | 更新存储仓库鉴权 | `1bd50e8b7d801000` |
| api | `PUT` | `/apis/storage.testudo.softcdata.com/v1/storages/storage-1` | 编辑存储仓库 | `3ee6850678c071` |
| api | `DELETE` | `/apis/storage.testudo.softcdata.com/v1/storages/storage-12` | 删除 | `25559f2338c0c2` |
| api | `POST` | `/apis/storage.testudo.softcdata.com/v1/storages/validate/connection` | 验证连接 | `6c722f5f8c74b` |
| websocket2 | `GET` | `/apis/storage.testudo.softcdata.com/v1/watch/storages` | 事件流(全部) | `263ef8a0b8c07c` |

## 容灾云平台 / V1 / 应用备份

| 类型 | 方法 | 路径 | 名称 | Target ID |
|---|---|---|---|---|
| api | `GET` | `/apis/appbackups.testudo.softcdata.com/v1/appbackups` | 列表查询 | `2a25b664f8c042` |
| api | `POST` | `/apis/appbackups.testudo.softcdata.com/v1/appbackups` | 创建应用备份（使用策略配置） | `3ee68503b8c051` |
| api | `POST` | `/apis/appbackups.testudo.softcdata.com/v1/appbackups` | 创建应用备份（手动） | `3efbb43238c57f` |
| api | `PUT` | `/apis/appbackups.testudo.softcdata.com/v1/appbackups/appback-test001` | 更新应用备份 | `3ee6850438c053` |
| api | `DELETE` | `/apis/appbackups.testudo.softcdata.com/v1/appbackups/backup-demo-1` | 删除 | `2a25b66578c049` |
| api | `GET` | `/apis/appbackups.testudo.softcdata.com/v1/appbackups/backup-demo-1` | 获取详情 | `2a25b664f8c043` |
| api | `GET` | `/apis/appbackups.testudo.softcdata.com/v1/appbackups/backup-demo-1/backups/app-backup-backup-demo-1-1767929222/download` | 下载备份文件 | `1bd135d66a001001` |
| api | `GET` | `/apis/appbackups.testudo.softcdata.com/v1/appbackups/backup-demo-1234/history` | 获取备份历史 | `1bd4dbb7f4401001` |
| api | `GET` | `/apis/appbackups.testudo.softcdata.com/v1/appbackups/clusters` | 获取应用备份关联集群 | `1be2239988801001` |
| api | `GET` | `/apis/appbackups.testudo.softcdata.com/v1/velero/backups/bak-rs-dr-rs-ds01-69dc9953/includes` | 查询Velero备份包含项(includedNamespaces/includedResources) | `1c30a700d3001001` |
| websocket2 | `GET` | `/apis/appbackups.testudo.softcdata.com/v1/watch/appbackups` | 事件流(全部)  | `2a418fe378c0a9` |
| websocket2 | `GET` | `/apis/appbackups.testudo.softcdata.com/v1/watch/appbackups/daily-app-backup` | 事件流(单个) | `2aab0af7b0c0a6` |

## 容灾云平台 / V1 / 应用备份 / 应用备份动作

| 类型 | 方法 | 路径 | 名称 | Target ID |
|---|---|---|---|---|
| api | `POST` | `/apis/appbackups.testudo.softcdata.com/v1/appbackups/:name/actions/cancel` | 触发取消 | `1bd135d4c5c01001` |
| api | `POST` | `/apis/appbackups.testudo.softcdata.com/v1/appbackups/:name/actions/retry` | 触发重试 | `1bd135d316401001` |
| api | `POST` | `/apis/appbackups.testudo.softcdata.com/v1/appbackups/app-backup-1767941589125/actions/pause` | 暂停pause恢复resume自动备份 | `3f0947a838c682` |
| api | `POST` | `/apis/appbackups.testudo.softcdata.com/v1/appbackups/app-backup-1768290527617ms/actions/backup` | 触发立即备份 | `3c1cb82178c001` |
| api | `POST` | `/apis/appbackups.testudo.softcdata.com/v1/appbackups/backup-demo-1/actions/delete` | 删除Velero备份 | `1bd135d64b401001` |

## 容灾云平台 / V1 / 应用备份恢复统计

| 类型 | 方法 | 路径 | 名称 | Target ID |
|---|---|---|---|---|
| api | `GET` | `/apis/backuprestorestatistics.testudo.softcdata.com/v1/autobackups/execution-summary` | 获取自动备份执行统计 | `1c7215aa3e401001` |
| api | `GET` | `/apis/backuprestorestatistics.testudo.softcdata.com/v1/backups` | 获取备份统计 | `1a7322a38c001` |
| api | `GET` | `/apis/backuprestorestatistics.testudo.softcdata.com/v1/backups/success-rate` | 获取备份成功率统计 | `1c239ca7d4801000` |
| api | `GET` | `/apis/backuprestorestatistics.testudo.softcdata.com/v1/instances` | 获取容灾实例统计 (概览) | `1c239ac3a6001000` |
| api | `GET` | `/apis/backuprestorestatistics.testudo.softcdata.com/v1/operations` | 获取容灾操作统计 | `1c1fe2a799801001` |
| api | `GET` | `/apis/backuprestorestatistics.testudo.softcdata.com/v1/operations/by-time` | 获取容灾操作统计 (按时间) | `1c2398ad1d001000` |
| api | `GET` | `/apis/backuprestorestatistics.testudo.softcdata.com/v1/restores` | 获取恢复统计 | `1a7322a38c003` |
| api | `GET` | `/apis/backuprestorestatistics.testudo.softcdata.com/v1/storage` | 获取存储用量统计 | `1c2620b6d7801001` |
| api | `GET` | `/apis/backuprestorestatistics.testudo.softcdata.com/v1/tasks/progress` | 获取备份恢复任务进度趋势 | `1c71f52b8b401001` |

## 容灾云平台 / V1 / 应用恢复

| 类型 | 方法 | 路径 | 名称 | Target ID |
|---|---|---|---|---|
| api | `GET` | `/apis/apprestores.testudo.softcdata.com/v1/apprestores` | 列表查询 | `34a0ae4978c001` |
| api | `POST` | `/apis/apprestores.testudo.softcdata.com/v1/apprestores` | 创建应用恢复 | `3ee6850478c056` |
| api | `PUT` | `/apis/apprestores.testudo.softcdata.com/v1/apprestores/:name` | 更新应用恢复 | `3ee68504b8c058` |
| api | `POST` | `/apis/apprestores.testudo.softcdata.com/v1/apprestores/preflight/validate` | 恢复前置校验，并加载BLS | `1c385ce4c7801001` |
| api | `GET` | `/apis/apprestores.testudo.softcdata.com/v1/apprestores/restore-172` | 获取详情 | `34a0ae49b8c002` |
| api | `DELETE` | `/apis/apprestores.testudo.softcdata.com/v1/apprestores/restore-55567` | 删除 | `34a0ae4b38c008` |
| websocket2 | `GET` | `/apis/apprestores.testudo.softcdata.com/v1/watch/apprestores` | 事件流(全部)  | `34a10209f8c00c` |
| websocket2 | `GET` | `/apis/apprestores.testudo.softcdata.com/v1/watch/apprestores/:name` | 事件流(单个)  | `34a12bb3b8c041` |

## 容灾云平台 / V1 / 应用恢复 / 应用恢复动作

| 类型 | 方法 | 路径 | 名称 | Target ID |
|---|---|---|---|---|
| api | `POST` | `/apis/apprestores.testudo.softcdata.com/v1/apprestores/restore001/actions/cancel` | 触发取消 | `3c211cc3f8c006` |
| api | `POST` | `/apis/apprestores.testudo.softcdata.com/v1/apprestores/restore001/actions/retry` | 触发重试 | `3c211cc3f8c008` |

## 容灾云平台 / V1 / 用户管理

| 类型 | 方法 | 路径 | 名称 | Target ID |
|---|---|---|---|---|
| api | `GET` | `/api/v1/users` | 用户列表 | `1c4ce9c769001001` |
| api | `POST` | `/api/v1/users` | 新增用户 | `1c4cd93b5fc01001` |
| api | `DELETE` | `/api/v1/users/:username` | 删除用户 | `1c4ce9c787001001` |
| api | `PATCH` | `/api/v1/users/:username/password` | 修改用户密码 | `1c4ce9c7a7401001` |
| api | `PATCH` | `/api/v1/users/:username/status` | 更新用户状态（启用/禁用） | `1c4cd93b7b001001` |

## 容灾云平台 / V1 / 策略配置

| 类型 | 方法 | 路径 | 名称 | Target ID |
|---|---|---|---|---|
| api | `GET` | `/apis/policies.testudo.softcdata.com/v1/policies` | 列表查询 | `35ec1a7278c00b` |
| api | `POST` | `/apis/policies.testudo.softcdata.com/v1/policies` | 创建容灾策略 | `3ee6850638c06a` |
| api | `PUT` | `/apis/policies.testudo.softcdata.com/v1/policies/auto-daily-backup` | 更新 AutoBackup 策略（传播到 Velero Schedule） | `1c721610dc401001` |
| api | `GET` | `/apis/policies.testudo.softcdata.com/v1/policies/disasterpolicy-sample` | 获取详情 | `35ec1a72b8c00c` |
| api | `DELETE` | `/apis/policies.testudo.softcdata.com/v1/policies/example-policy001` | 删除 | `35ec1a7378c012` |
| api | `PUT` | `/apis/policies.testudo.softcdata.com/v1/policies/example-policy001` | 更新容灾策略 | `3ee6850638c06c` |
| api | `GET` | `/apis/policies.testudo.softcdata.com/v1/policies/names` | 获取策略名称列表 | `1be2230900401001` |

## 容灾云平台 / V1 / 系统设置（配置管理）

| 类型 | 方法 | 路径 | 名称 | Target ID |
|---|---|---|---|---|
| api | `GET` | `/api/v1/system-settings` | 列表查询 | `1c31b7bf28801000` |
| api | `POST` | `/api/v1/system-settings` | 创建配置项 | `1c31b7bf87401001` |
| api | `DELETE` | `/api/v1/system-settings/:config_key` | 删除配置项 | `1c31b7ca57001001` |
| api | `PUT` | `/api/v1/system-settings/:config_key` | 更新配置项 | `1c31b7bfa1c01001` |
| api | `GET` | `/api/v1/system-settings/assets/:config_key` | 获取资产 | `1c31b7ca8b001000` |
| api | `POST` | `/api/v1/system-settings/assets/:config_key` | 上传资产 | `1c31b7ca73001000` |
| api | `GET` | `/api/v1/system-settings/public` | public按keys读取 | `1c31b7bf6e801000` |

## 容灾云平台 / V1 / 平台许可

| 类型 | 方法 | 路径 | 名称 | Target ID |
|---|---|---|---|---|
| api | `GET` | `/apis/v1/platform-license/status` | 查询平台许可状态 | `1c7dbf4632001001` |
| api | `POST` | `/apis/v1/platform-license/install` | 安装平台许可 | `1c7dbf57bd001001` |

## 容灾云平台 / V1 / 统一删除保护

| 类型 | 方法 | 路径 | 名称 | Target ID |
|---|---|---|---|---|
| api | `POST` | `/api/v1/deletion/check` | 删除前依赖检查 | `1c2f51b2b9801001` |

## 容灾云平台 / V1 / 集群配置管理

| 类型 | 方法 | 路径 | 名称 | Target ID |
|---|---|---|---|---|
| api | `GET` | `/apis/cluster.testudo.softcdata.com/v1/clusters` | 列表查询 | `25559f1f78c067` |
| api | `POST` | `/apis/cluster.testudo.softcdata.com/v1/clusters` | 添加集群(kubeconfig添加) | `3ee6850538c060` |
| api | `POST` | `/apis/cluster.testudo.softcdata.com/v1/clusters` | 添加集群(token添加) | `69ee1a0f8c0b7` |
| api | `POST` | `/apis/cluster.testudo.softcdata.com/v1/clusters/:name/actions/refresh-namespaces` | 刷新命名空间统计 | `1c613a8d76001001` |
| api | `GET` | `/apis/cluster.testudo.softcdata.com/v1/clusters/cluster-ip170-1774332377/restore-classes` | 获取恢复Class列表 | `1c4bb4e50d001001` |
| api | `GET` | `/apis/cluster.testudo.softcdata.com/v1/clusters/cluster-ip172` | 获取详情 | `25559f1fb8c069` |
| api | `GET` | `/apis/cluster.testudo.softcdata.com/v1/clusters/ip170-test-001/protected-namespaces` | 查询指定集群下已受保护命名空间 | `1c61734794401001` |
| api | `DELETE` | `/apis/cluster.testudo.softcdata.com/v1/clusters/ip172` | 删除 | `25559f2038c06f` |
| api | `POST` | `/apis/cluster.testudo.softcdata.com/v1/clusters/kubeconfig/validate` | 验证连接(kubeconfig) | `67d3bab78c00b` |
| api | `POST` | `/apis/cluster.testudo.softcdata.com/v1/clusters/kubeconfig/validate` | 验证连接(token) | `69dbb8ef8c07f` |
| api | `GET` | `/apis/cluster.testudo.softcdata.com/v1/clusters/master/validate` | 校验 | `25559f2038c072` |
| api | `PATCH` | `/apis/cluster.testudo.softcdata.com/v1/clusters/master01` | 更新集群 | `1bd502f9fe001000` |
| api | `GET` | `/apis/cluster.testudo.softcdata.com/v1/clusters/names` | 获取集群名称列表 | `1be22307d0001001` |
| websocket2 | `GET` | `/apis/cluster.testudo.softcdata.com/v1/watch/clusters` | 事件流(全部) | `2667344638c320` |
| websocket2 | `GET` | `/apis/cluster.testudo.softcdata.com/v1/watch/clusters/cluster-1` | 事件流(单个) | `2667d19538c36d` |

## 容灾云平台 / V2 / 容灾实例配置

| 类型 | 方法 | 路径 | 名称 | Target ID |
|---|---|---|---|---|
| api | `GET` | `/apis/disasterinstances.testudo.softcdata.com/v1/instances` | 列表查询 | `1c01f46e4f001000` |
| api | `POST` | `/apis/disasterinstances.testudo.softcdata.com/v1/instances` | 创建实例 | `1c01f46f55801000` |
| api | `GET` | `/apis/disasterinstances.testudo.softcdata.com/v1/instances/:name` | 获取详情 | `1c01f470ba401000` |
| api | `PUT` | `/apis/disasterinstances.testudo.softcdata.com/v1/instances/:name` | 更新实例 | `1c01f4718c001000` |
| api | `GET` | `/apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/history` | 操作历史记录 | `1c01f7948bc01000` |
| api | `POST` | `/apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/restore-classes/validate` | 校验恢复Class存在性 | `1c49f6d741801001` |
| api | `GET` | `/apis/disasterinstances.testudo.softcdata.com/v1/instances/ds01/operations/failover-ds01-1776225823995710522` | 获取操作详情 | `1c5ead7bab401001` |
| api | `DELETE` | `/apis/disasterinstances.testudo.softcdata.com/v1/instances/pppp1-dr-instance-test1` | 删除实例 | `1c01f472a2c01000` |
| api | `GET` | `/apis/disasterinstances.testudo.softcdata.com/v1/instances/self-web/sync-status` | 查询同步状态 | `1c01f74a5e401000` |
| api | `GET` | `/apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/sync-history` | 查询同步历史 | `1c8428770dc01001` |
| api | `POST` | `/apis/disasterinstances.testudo.softcdata.com/v1/instances/test-cleanup-vol/actions` | 执行操作 (增强版) | `1c01fb364ac01000` |
| websocket2 | `GET` | `/apis/disasterinstances.testudo.softcdata.com/v1/watch/instances` | 事件流(全部) | `3b8699c1b8c14b` |
| websocket2 | `GET` | `/apis/disasterinstances.testudo.softcdata.com/v1/watch/instances/app-group-1772076095894` | 事件流(单个) | `3b86fa34f8c221` |
| api | `GET` | `/apis/disasterinstances.testudo.softcdata.com/v1/watch/instances/operations/:operationName` | 操作事件流(单个) | `1c5eadf4f1801001` |

## 容灾云平台 / V2 / 容灾演练 (Drill)

| 类型 | 方法 | 路径 | 名称 | Target ID |
|---|---|---|---|---|
| api | `GET` | `/apis/disasterdrills.testudo.softcdata.com/v1/drills` | 列表查询 | `1c0aec042dc01001` |
| api | `POST` | `/apis/disasterdrills.testudo.softcdata.com/v1/drills` | 创建演练 | `1c0aec0a8f401001` |
| api | `POST` | `/apis/disasterdrills.testudo.softcdata.com/v1/drills/:name/cleanup` | 清理演练 | `1c28bf2907c01000` |
| api | `GET` | `/apis/disasterdrills.testudo.softcdata.com/v1/drills/actions/protected-namespaces` | 查询受保护命名空间 | `1c380061ca801001` |
| api | `GET` | `/apis/disasterdrills.testudo.softcdata.com/v1/drills/drill-pppp1-dr-instance-test1-20260209144531` | 获取详情 | `1c0aec0a6cc01001` |
| api | `DELETE` | `/apis/disasterdrills.testudo.softcdata.com/v1/drills/drills01` | 删除演练 | `1c0aec15e5001001` |
| api | `POST` | `/apis/disasterdrills.testudo.softcdata.com/v1/drills/drills1/confirm` | 确认执行 | `1c0aec15bc001001` |
| api | `POST` | `/apis/disasterdrills.testudo.softcdata.com/v1/drills/drills15/restart` | 重跑演练 | `1c0c8731f1401001` |
| websocket2 | `GET` | `/apis/disasterdrills.testudo.softcdata.com/v1/watch/drills` | 事件流(全部) | `477a33c78c1c8` |
| websocket2 | `GET` | `/apis/disasterdrills.testudo.softcdata.com/v1/watch/drills/:name` | 事件流(单个) | `477bda678c1cb` |

## 容灾云平台 / V2 / 容灾组管理 (Disaster Group)

| 类型 | 方法 | 路径 | 名称 | Target ID |
|---|---|---|---|---|
| api | `GET` | `/apis/disastergroups.testudo.softcdata.com/v1/groups` | 查询列表 | `1c048e6aa6401000` |
| api | `POST` | `/apis/disastergroups.testudo.softcdata.com/v1/groups` | 创建容灾组 | `1c048e6950c01000` |
| api | `GET` | `/apis/disastergroups.testudo.softcdata.com/v1/groups/:name/history` | 获取组操作历史 | `1c04e21a5c801001` |
| api | `GET` | `/apis/disastergroups.testudo.softcdata.com/v1/groups/:name/operations/:operationName` | 获取组操作详情 | `1c5ead7bcb801001` |
| api | `GET` | `/apis/disastergroups.testudo.softcdata.com/v1/groups/app-group-1772076095894/instances` | 容灾组已选实例列表 | `1c2100948fc01001` |
| api | `PUT` | `/apis/disastergroups.testudo.softcdata.com/v1/groups/app-group-1772244901658` | 更新容灾组 | `1c239548ddc01000` |
| api | `GET` | `/apis/disastergroups.testudo.softcdata.com/v1/groups/core-app-group` | 获取详情 | `1c048e6d34001000` |
| api | `DELETE` | `/apis/disastergroups.testudo.softcdata.com/v1/groups/core-app-group1` | 删除容灾组 | `1c04bcb222801001` |
| api | `POST` | `/apis/disastergroups.testudo.softcdata.com/v1/groups/group1/actions` | 执行组操作 | `1c048e6d67c01000` |
| api | `GET` | `/apis/disastergroups.testudo.softcdata.com/v1/groups/instance-picker` | 实例选择器列表 | `1c20cb43ab001001` |
| websocket2 | `GET` | `/apis/disastergroups.testudo.softcdata.com/v1/watch/groups/operations` | 事件流(全部) | `1a44c9e9f8c0a8` |
| api | `GET` | `/apis/disastergroups.testudo.softcdata.com/v1/watch/groups/operations/:operationName` | 组操作事件流(单个) | `1c5eadf517801001` |
| websocket2 | `GET` | `/apis/disastergroups.testudo.softcdata.com/v1/watch/groups/operations/app-group-1772076095894` | 事件流(单个) | `1a44ffeb38c0ab` |
| websocket2 | `GET` | `/apis/disastergroups.testudo.softcdata.com/v1/watch/groups/status` | 状态事件流(全部) | `39a5775f8c2ec` |
| websocket2 | `GET` | `/apis/disastergroups.testudo.softcdata.com/v1/watch/groups/status/app-group-1774505807766` | 状态事件流(单个) | `39a699978c2f0` |

## 容灾云平台 / V2 / 灾备配置

| 类型 | 方法 | 路径 | 名称 | Target ID |
|---|---|---|---|---|
| api | `GET` | `/apis/disasterconfigs.testudo.softcdata.com/v1/configs` | 列表查询 | `25559f2138c090` |
| api | `POST` | `/apis/disasterconfigs.testudo.softcdata.com/v1/configs` | 创建容灾配置 | `3ee68505b8c065` |
| api | `PUT` | `/apis/disasterconfigs.testudo.softcdata.com/v1/configs/:name` | 更新容灾配置 | `3ee68505f8c067` |
| api | `DELETE` | `/apis/disasterconfigs.testudo.softcdata.com/v1/configs/dr-config-inst-auto-trigger-1770010966` | 删除 | `25559f21b8c098` |
| api | `GET` | `/apis/disasterconfigs.testudo.softcdata.com/v1/configs/global-config1` | 获取详情 | `25559f2138c092` |
| api | `GET` | `/apis/disasterconfigs.testudo.softcdata.com/v1/configs/names` | 获取容灾配置名称列表 | `1c304687b0801001` |
| websocket2 | `GET` | `/apis/disasterconfigs.testudo.softcdata.com/v1/watch/configs` | 事件流(全部) | `267c3f73f8c40a` |
| websocket2 | `GET` | `/apis/disasterconfigs.testudo.softcdata.com/v1/watch/configs` | 事件流(全部) | `3b831dfe78c08c` |
| websocket2 | `GET` | `/apis/disasterconfigs.testudo.softcdata.com/v1/watch/configs/global-config1` | 事件流(单个) | `267c641278c417` |
