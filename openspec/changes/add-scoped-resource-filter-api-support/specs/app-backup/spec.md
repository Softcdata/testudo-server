## ADDED Requirements
### Requirement: AppBackup 支持完整 scoped 资源过滤字段
Server 在 `AppBackup` 创建、更新、查询接口中必须 (MUST) 支持并回显完整 scoped 资源过滤四字段。

#### Scenario: 创建请求携带 scoped 四字段
- **WHEN** 客户端发送 `POST AppBackup` 请求，包含 `includedNamespaceScopedResources`、`excludedNamespaceScopedResources`、`includedClusterScopedResources`、`excludedClusterScopedResources`
- **THEN** Server 必须将四字段完整写入 `AppBackup.Spec.Template`
- **AND** 创建响应必须回显同名字段

#### Scenario: 更新请求携带 scoped 四字段
- **WHEN** 客户端发送 `PUT AppBackup` 请求，包含 scoped 四字段
- **THEN** Server 必须把字段更新到目标资源
- **AND** 更新响应必须回显同名字段

### Requirement: AppBackup 资源过滤冲突在提交期拒绝
Server 在 `AppBackup` 创建与更新入口必须 (MUST) 对资源过滤字段执行冲突校验并在提交期拒绝非法请求。

#### Scenario: old 与 scoped 混用被拒绝
- **WHEN** 请求同时包含 old 字段（`includedResources`、`excludedResources`、`includeClusterResources`）与 scoped 字段
- **THEN** Server 必须返回 400
- **AND** 错误消息必须标识混用冲突

#### Scenario: scoped include/exclude 冲突被拒绝
- **WHEN** scoped include 与 scoped exclude 存在同项冲突，或者 `exclude=["*"]` 且 include 非空
- **THEN** Server 必须返回 400
- **AND** 错误消息必须标识冲突项
