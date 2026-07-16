# Release 1.0.2

发布日期：2026-07-16

基线版本：`1.0.1`（`2adf84fdae433623e1f45596abec8128f894c6e5`）

功能代码截至：`bdd05367ed8ce6bf496db2da3f79d7711a6c5a8a`

## 版本概览

本版本补齐容灾实例操作超时参数，修复集群私有仓库凭据 PATCH 的空字段语义，将 modifier 容量与 Operator 对齐，并新增业务默认配置查询、字段目录和更新 API。

## 已实现功能与修复

### 1. 集群 Velero 私仓凭据 PATCH 修复

1.0.1 已支持通过 `veleroInstall` 管理镜像仓库和 registry credential，本版本修复编辑态空字段被误解释为删除的问题：

- PATCH 未携带 username/password 时保持现有 Secret。
- PATCH 携带 `password=""` 时保持现有 Secret，不轮换、不删除。
- PATCH 携带空 username 和空 password 时保持现有 Secret。
- 仅携带空 username 时保持现有 Secret。
- 只有 `removeCredential=true` 才显式删除托管 Secret 和 credential ref。
- 显式 `imageRegistry=""` 继续表示删除整个 Velero install registry 配置及托管 Secret。
- 仅修改非空 imageRegistry 时保留现有 credential。
- detail/list/create/patch 响应只回显 imageRegistry、credentialConfigured 和可解析 username，不返回 password 或 dockerconfigjson。

### 2. 容灾实例 operationTimeoutMinutes

- 创建 DisasterInstance 时支持提交 `operationTimeoutMinutes`。
- 更新 DisasterInstance 时支持部分修改 `operationTimeoutMinutes`。
- detail/list DTO 回显当前值。
- 字段写入 `DisasterInstance.spec.operationTimeoutMinutes`。
- 配套 Operator 1.0.2 会把该值用于实例操作超时，并投递到 DataSync/ResourceSync 创建的 AppBackup/AppRestore。

### 3. modifier 容量对齐

- Server 的单实例 modifier rule 上限从 200 提高到 1000。
- 与 Operator 1.0.2 的治理上限一致。
- 超出上限仍返回 `ModifierRuleRejected`。

### 4. 业务默认配置 API

新增以下认证接口，并同时支持 `/api/v1` 和 `/apis/v1` 前缀：

- `GET /api/v1/business-default-config`
- `PATCH /api/v1/business-default-config`
- `GET /api/v1/business-default-config/fields`
- `GET /api/v1/business-default-config/frontend-fields`

能力包括：

- 按 backupRuntime、restoreRuntime、operationRuntime、instanceRuntime、syncRuntime、storageRuntime、clusterRuntime 分组返回字段。
- 返回 key、name、description、value、defaultValue、dataType、editable、effectMode、min、max 等元数据。
- 字段列表支持关键字、分组、editable、effectMode、分页和排序。
- frontend-fields 返回 requestPath、specPath、字段层级、API usages 和按 key 索引的 fieldMap。
- PATCH 支持扁平 key 和分组对象的部分更新。
- 校验未知字段、只读字段、duration/int/bool 类型、范围和跨字段关系。
- 存储使用 `disaster-system/disaster-business-default-config` ConfigMap 的 `config` data key。
- 使用 Kubernetes resourceVersion 冲突重试，单文档最大 256 KiB。

### 5. Swagger/OpenAPI 和 RunAPI 文档

- OpenAPI 增加 business-default-config 路由、请求、响应、错误和字段 schema。
- DisasterInstance API 增加 operationTimeoutMinutes。
- modifierRules 文档上限更新为 1000。
- 更新集群 registry credential PATCH 保持/轮换/删除语义。
- 更新 Swagger 接口清单和 RunAPI 证据。

## 行为变化

### 临时放宽受保护 namespace 冲突检查

当前版本的 `validateProtectedNamespaces` 直接返回成功，因此允许同一源集群的同一 namespace 被多个 DisasterInstance 同时保护。旧冲突检测代码仍保留在 return 后，作为测试窗口后的快速恢复参考。

这是一项高风险临时行为，可能导致重复 DataSync/ResourceSync、资源归属冲突和多个实例同时恢复同一 namespace。正式生产使用前应明确业务决策：恢复冲突保护，或者用正式 OpenSpec、开关和冲突模型替代临时绕过。

## 跨仓依赖和升级要求

- 配套 Operator 版本应为 1.0.2，以消费 operationTimeoutMinutes、1000 条 modifier 和新增 Drill 模式。
- 新业务默认配置 API 需要 Server ServiceAccount 对 `disaster-system` ConfigMap 具备 get/create/update 权限。
- 本版本没有数据库迁移。
- 仓库中的历史 `dist/disaster-server-20260707-095356-amd64.tar` 不属于 1.0.2 源码和 tag。

## 当前未闭环的契约

### 1. ResourceOnly/Mixed Drill 输出契约未同步

Operator 1.0.2 新增 ResourceOnly/Mixed 和逐实例 `instanceRestoreModes`，但当前 Server：

- Drill OpenAPI 仍声明 `enum: [Reuse, FullRestore]`。
- DisasterDrillStatusDTO 没有 `instanceRestoreModes`。
- OpenAPI 文案仍描述 Operator 始终使用 FullRestore。

Server 使用 string 读取 restoreMode，因此运行时值可能透传，但 OpenAPI、生成客户端和前端类型仍不完整。

### 2. 业务默认配置尚未形成实际生效闭环

- PATCH 只写 `disaster-business-default-config` ConfigMap。
- Operator 监听的是 `OperatorRuntimeConfig/default` CR，不监听该 ConfigMap。
- Server 不会把 ConfigMap 转换或同步为 OperatorRuntimeConfig。
- `/frontend-fields` 当前从静态 DefaultValue 构造响应，不读取已持久化 ConfigMap。
- Server 也不会自动把这些默认值注入业务创建/更新请求。

因此该 API 当前完成的是配置存储、字段目录和校验，不应宣称修改后已经热更新 Operator 或自动影响后续业务请求。

## 验证状态

已通过：

- `go test ./internal/apis/disaster_cluster/v1`
- `go test ./internal/apis/disaster_instance/v1`
- `go test ./internal/apis/business_default_config/v1`
- `go test ./internal/apis/disaster_drill/v1`
- `go test ./internal/router`
- `openspec validate add-disaster-cluster-registry-credential-api --strict`
- `openspec validate add-business-default-config-api --strict`
- `go run ./tools/openapi validate --spec openspec/specs/disaster-server-openapi.yaml`
- `git diff --check`

说明：Server `go.mod` 使用 `replace github.com/softcdata/testudo-operator => ../disaster-operator`，上述测试证明当前 Server 与本地 Operator 1.0.2 源码组合可编译，不是独立远程 module 解析证据。

## 已知问题和发布限制

- namespace 重复保护检查处于临时关闭状态。
- ResourceOnly/Mixed Server/OpenAPI 契约未完整同步。
- 业务默认配置尚未写入 OperatorRuntimeConfig，也不会改变 frontend-fields 当前值。
- 当前 tag 代表源码版本冻结，不等同于无条件生产准入结论。

## 功能提交

- `bdd0536`：新增业务默认配置 API，修复私仓凭据 PATCH，补齐实例超时和 modifier 上限。
