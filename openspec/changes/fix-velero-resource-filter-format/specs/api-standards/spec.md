## ADDED Requirements

### Requirement: Velero 资源筛选字段必须使用 group-resource 格式

Server MUST 在 Velero 资源清单以及 AppRestore 写入边界把资源筛选值规范为 group-resource 格式，例如 `apps/v1/Deployment` 必须转换为 `deployments.apps`，`v1/ConfigMap` 必须转换为 `configmaps`。转换必须使用备份源集群 RESTMapper，禁止通过 Kind 文本手工推断资源复数。

#### Scenario: 资源清单转换 Deployment

- **WHEN** 备份资源清单包含键 `apps/v1/Deployment`
- **THEN** `GET /apis/appbackups.testudo.softcdata.com/v1/velero/backups/{backupName}/includes` 返回的 `includedResources` 必须包含 `deployments.apps`
- **AND** 返回值不得包含 `apps/v1/Deployment`

#### Scenario: 创建恢复兼容 GVK 包含项以及排除项

- **WHEN** 客户端创建 AppRestore
- **AND** `includedResources` 包含 `apps/v1/Deployment`
- **AND** `excludedResources` 包含 `v1/ConfigMap`
- **THEN** Server 必须在 AppRestore CR 的 `spec.template` 中保存 `deployments.apps` 以及 `configmaps`
- **AND** Operator 接收到的 RestoreSpec 必须继续使用这两个 group-resource 值

#### Scenario: 更新恢复兼容 GVK 包含项以及排除项

- **WHEN** 客户端更新 AppRestore
- **AND** 请求的 `includedResources` 以及 `excludedResources` 含有 GVK 值
- **THEN** Server 必须在更新后的 AppRestore CR 中保存对应 group-resource 值

#### Scenario: 恢复源变更或历史恢复缺少源集群字段时按备份源映射

- **WHEN** 更新 AppRestore，且其 `spec.sourceCluster` 为空
- **OR** 本次请求将 `backupSource` 更改为另一个 AppBackup
- **AND** 请求的包含项或排除项使用 GVK 格式
- **THEN** Server 必须读取更新后 `backupSource` 指向的 AppBackup
- **AND** 使用 AppBackup 的 `spec.cluster` 对 GVK 执行 RESTMapper 映射
- **AND** 不得使用恢复目标 `spec.cluster` 代替备份源集群

#### Scenario: 更新恢复无法确定源集群时拒绝 GVK

- **WHEN** 更新 AppRestore 的包含项或排除项使用 GVK 格式
- **AND** Server 无法从 AppRestore 或关联 AppBackup 确定源集群
- **THEN** 更新接口必须返回 HTTP 400
- **AND** 不得更新 AppRestore CR

#### Scenario: 已规范值保持兼容

- **WHEN** 客户端提交 `deployments.apps`、`configmaps` 以及 `*`
- **THEN** Server 必须原样保存这些值
- **AND** 不得因为不需要 RESTMapper 查询而拒绝请求

#### Scenario: GVK 无法解析时拒绝请求

- **WHEN** 客户端提交 RESTMapper 无法解析的 GVK
- **THEN** AppRestore 创建以及更新接口必须返回 HTTP 400
- **AND** 错误信息必须包含字段名以及原始资源值
- **AND** Server 不得创建以及更新 AppRestore CR

#### Scenario: 资源清单中的未知 GVK 不得静默透传

- **WHEN** 备份资源清单包含 RESTMapper 无法解析的 GVK
- **THEN** 资源清单接口必须返回明确的内部错误
- **AND** 响应不得把未知 GVK 放入 `includedResources`
