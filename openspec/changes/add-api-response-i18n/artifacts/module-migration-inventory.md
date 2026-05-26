# 模块迁移清单

## 说明
本清单记录需要逐步迁移到消息键的代码点。统计口径包含 `transport.WriteError`、直接 `ctx.JSON`/`c.JSON` 返回用户可见消息、以及成功响应 `meta.error`/`meta.message` 中承载用户可见消息的位置。

## 2026-05-16 执行状态

- 已完成：P0 公共出口全部落地，包括语言中间件、统一响应信封、JWT、登录、刷新令牌、Recovery、Tenant 与 WebSocket 错误。
- 已完成首批高频错误：`disaster_cluster`、`disaster_instance`、`app_backup`、`app_restore`、`system_settings`、`user`、`event`、`platform_license`、`disaster_backup`、`disaster_config`、`disaster_jobs`、`disaster_policy`、`deletion_check`、`kubernetes_resources` 中的本地校验和直接 JSON 错误出口。
- 未完成：`disaster_storage`、`disaster_drill`、`statistics`、`disaster_group` 以及多个模块内的上游 Kubernetes/S3/运行时原始 `err.Error()` 仍按兼容路径返回，后续按模块继续迁移到通用消息键加 `meta.details`。

迁移原则：

- 公共出口先迁移，业务模块后迁移。
- 本地校验错误改为稳定 `message_key` 和参数。
- 上游 Kubernetes、S3、运行时客户端错误不得按字符串反查翻译；应返回通用消息键，并将原始错误放入 `meta.details`、日志、trace 关联信息。
- `data` 中的状态枚举、CRD 原始字段保持原值。

## P0 公共出口

| 模块 | 文件 | 需要迁移的位置 | 迁移内容 |
| --- | --- | --- | --- |
| 统一响应 | `internal/transport/response.go` | `Success`、`Error`、`WriteError` | 增加 `message_key`、`WriteErrorKey`、`WriteErrorFrom`，旧 `WriteError` 保持兼容 |
| 语言中间件 | `internal/middleware/locale.go` | 新增文件 | 解析 `X-Language`、`Accept-Language`，设置 `Content-Language`、`Vary` |
| JWT | `internal/middleware/jwt.go` | 123、198、206、222、231、240、253 | 直接 `msg` 响应改为统一 `Envelope`，补 `auth.*` 消息键 |
| 登录成功 | `internal/middleware/jwt.go` | 90 | 登录成功直接 `c.JSON` 改为统一 `Envelope` |
| Recovery | `internal/middleware/recovery.go` | 16 | `error: internal server error` 改为统一 `Envelope` 和 `common.internal_error` |
| WebSocket | `internal/utils/watch.go` | 84、93、185、196 | 握手失败、watcher 创建失败、事件流错误改为消息键 |
| Tenant | `internal/middleware/tenant.go` | 27 | `X-Tenant-ID header is required` 改为 `tenant.header_required` |

## P1 首批业务模块

| 模块 | 候选数量 | 文件 | 代表位置 | 需要迁移的消息类型 |
| --- | ---: | --- | --- | --- |
| `disaster_instance/v1` | 71 | `internal/apis/disaster_instance/v1/handler.go`、`handler_action.go`、`handler_operation.go`、`namespace_guard.go`、`modifier_rule_validation.go`、`modifier_rule_live_validation.go`、`bulk_modifier_snapshot.go`、`sync_policy.go`、`types.go` | `handler.go:414`、`handler.go:671`、`handler.go:1556`、`handler_operation.go:42`、`handler_action.go:146` | config 未找到、class mapping 校验、sync history 参数非法、操作不存在、操作状态不允许、modifier rule 校验 |
| `disaster_cluster/v1` | 47 | `internal/apis/disaster_cluster/v1/handler.go`、`guard.go` | `handler.go:389`、`handler.go:405`、`handler.go:686`、`handler.go:697`、`handler.go:800`、`guard.go:107` | name 必填、恢复类列表失败、License 限制、删除参数非法、kubeconfig 校验 `meta.error`、endpoint 校验 |
| `app_restore/v1` | 41 | `internal/apis/app_restore/v1/handler.go` | `handler.go:274`、`handler.go:283`、`handler.go:353`、`handler.go:419`、`handler.go:457`、`handler.go:662` | AppBackup 不存在、集群客户端失败、备份字段缺失、恢复策略非法、preflight `meta.error`、watch 直接 JSON |
| `app_backup/v1` | 51 | `internal/apis/app_backup/v1/handler.go`、`velero_backup_includes.go`、`remote_client.go` | `handler.go:239`、`handler.go:500`、`handler.go:507`、`handler.go:514`、`handler.go:718`、`velero_backup_includes.go:76` | name 必填、历史备份不存在、StorageLocation 缺失、StorageRepository 不存在、动作类型不支持、Velero 备份不存在 |
| `system_settings/v1` | 31 | `internal/apis/system_settings/v1/handler.go` | `handler.go:113`、`handler.go:194`、`handler.go:302`、`handler.go:335`、`handler.go:535`、`handler.go:426` | keys 必填、至少一个字段、文件必填、资产超限、仅 admin 可改、config key 非法 |
| `user/v1` | 21 | `internal/apis/user/v1/handler.go`、`internal/userstore/store.go`、`internal/userstore/password.go` | `handler.go:36`、`handler.go:223`、`handler.go:230`、`handler.go:241`、`store.go:33`、`password.go:15` | user store 未初始化、用户名非法、邮箱必填、密码必填、用户不存在、用户已存在 |

## P2 第二批业务模块

| 模块 | 候选数量 | 文件 | 代表位置 | 需要迁移的消息类型 |
| --- | ---: | --- | --- | --- |
| `disaster_storage/v1` | 39 | `internal/apis/disaster_storage/v1/handler.go` | `handler.go:250`、`handler.go:579`、`handler.go:599`、`handler.go:655`、`handler.go:667`、`handler.go:671` | endpoint 格式、S3 校验 `meta.error`、AWS 配置加载失败、集群客户端失败、BSL 连接校验消息 |
| `disaster_drill/v1` | 38 | `internal/apis/disaster_drill/v1/handler.go` | `handler.go:132`、`handler.go:524`、`handler.go:534`、`handler.go:567`、`handler.go:677`、`handler.go:820` | 演练目标互斥、目标必填、名称长度、实例与组不存在、状态不允许、清理已触发 |
| `disaster_config/v1` | 20 | `internal/apis/disaster_config/v1/handler.go`、`sync_policy_validation.go` | `handler.go:316`、`handler.go:416`、`handler.go:437`、`handler.go:480`、`handler.go:489`、`sync_policy_validation.go:22` | path name 必填、imageRewrite applyTo、unmatchedPolicy、source/target cluster 必填和不存在、旧 syncPolicy 拒绝 |
| `disaster_group/v1` | 19 | `internal/apis/disaster_group/v1/handler.go`、`handler_operation.go` | `handler.go:272`、`handler.go:366`、`handler.go:756`、`handler.go:801`、`handler.go:891`、`handler_operation.go:30` | 组不存在、操作启动 message、组操作不存在、成员实例不存在、配置不存在 |
| `disaster_policy/v1` | 16 | `internal/apis/disaster_policy/v1/handler.go` | `handler.go:49`、`handler.go:125`、`handler.go:257`、`handler.go:277` | 直接 JSON、URL 名称与 body 名称不一致、策略被 AppBackup 引用 |
| `disaster_backup/v1` | 14 | `internal/apis/disaster_backup/v1/handler.go` | `handler.go:122`、`handler.go:247` | name 必填、创建更新删除中的校验与资源错误 |

## P3 第三批业务模块

| 模块 | 候选数量 | 文件 | 代表位置 | 需要迁移的消息类型 |
| --- | ---: | --- | --- | --- |
| `statistics/v1` | 13 | `internal/apis/statistics/v1/handler.go` | `handler.go:388`、`handler.go:413`、`handler.go:474`、`handler.go:504`、`handler.go:896` | 时间格式非法、周期非法、任务类型必填、时区非法、origin 非法 |
| `event/v1` | 10 | `internal/apis/event/v1/handler.go`、`list.go`、`resource_kind.go` | `handler.go:55`、`handler.go:85`、`list.go:66`、`list.go:328`、`resource_kind.go:46` | watcher 初始化失败、resource/name 必填、事件列表失败、资源类型非法 |
| `disaster_jobs/v1` | 9 | `internal/apis/disaster_jobs/v1/handler.go` | `handler.go:165`、`handler.go:197` | name 必填、watch 直接 JSON |
| `platform_license/v1` | 7 | `internal/apis/platform_license/v1/handler.go`、`service.go` | `handler.go:42`、`handler.go:66`、`handler.go:70`、`service.go:45` | license service 未初始化、license 必填、license JSON 格式非法 |
| `deletion_check/v1` | 6 | `internal/apis/deletion_check/v1/handler.go`、`cleanup_plan.go` | `handler.go:71`、`handler.go:76`、`handler.go:104`、`cleanup_plan.go:363` | resource_kind 不支持、name 必填、dependency-token 生成失败、kind 不支持 |
| `kubernetes_resources` | 1 | `internal/apis/kubernetes_resources/handler.go` | `handler.go:32` | 直接 JSON 返回资源列表错误 |

## 低优先级与特殊处理

| 模块 | 文件 | 说明 |
| --- | --- | --- |
| OpenAPI 静态资源 | `internal/openapi/handler.go` | 仅文档服务失败消息，正式 API i18n 可不作为首批 |
| Kubernetes 上游错误 | `internal/kube/methods.go`、各 handler 的 `err.Error()` | 改为通用消息键加 `meta.details`，不翻译上游错误正文 |
| Restore preflight | `internal/service/verifier/restore_preflight_verifier.go` | `result.Reason` 进入 `message` 与 `meta.error`，建议改为结构化 reason key |
| Common validator | `internal/common/validator.go` | 集群与存储校验错误会被多个模块复用，建议抽公共消息键 |

## 建议执行顺序

1. P0 公共出口。
2. P1 中的 `user/v1` 与 JWT，覆盖登录、权限、用户管理。
3. P1 中的 `disaster_cluster/v1` 与 `disaster_instance/v1`，覆盖核心资源管理。
4. P1 中的 `app_backup/v1` 与 `app_restore/v1`，覆盖备份恢复主流程。
5. P2 与 P3 模块按接口访问频率推进。
