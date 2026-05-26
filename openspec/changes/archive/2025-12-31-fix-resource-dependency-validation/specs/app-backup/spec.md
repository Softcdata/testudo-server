## ADDED Requirements

### Requirement: 前置依赖验证
Server 端在创建 AppBackup 资源前,必须 (MUST) 验证依赖资源的就绪状态,确保备份操作能够成功执行。

#### Scenario: Cluster 不存在时拒绝创建
- **WHEN** 发送 POST 请求创建 AppBackup
- **AND** `Spec.Cluster` 引用的 Cluster 资源不存在
- **THEN** 返回 400 Bad Request
- **AND** 错误信息包含 "Cluster not found: {cluster_name}"

#### Scenario: Cluster 未就绪时拒绝创建
- **WHEN** 发送 POST 请求创建 AppBackup
- **AND** `Spec.Cluster` 引用的 Cluster 资源存在但 `Status.Status != "Ready"`
- **THEN** 返回 400 Bad Request
- **AND** 错误信息包含 "Cluster {name} is not ready (current: {status})"

#### Scenario: StorageRepository 不存在时拒绝创建
- **WHEN** 发送 POST 请求创建 AppBackup
- **AND** `Spec.Template.StorageLocation` 引用的 StorageRepository 资源不存在
- **THEN** 返回 400 Bad Request
- **AND** 错误信息包含 "StorageRepository not found: {repo_name}"

#### Scenario: StorageRepository 未就绪时拒绝创建
- **WHEN** 发送 POST 请求创建 AppBackup
- **AND** `Spec.Template.StorageLocation` 引用的 StorageRepository 资源存在但 `Status.Status != "Available"`
- **THEN** 返回 400 Bad Request
- **AND** 错误信息包含 "StorageRepository {name} is not available (current: {status})"

#### Scenario: 所有依赖就绪时允许创建
- **WHEN** 发送 POST 请求创建 AppBackup
- **AND** `Spec.Cluster` 引用的 Cluster 资源存在且 `Status.Status == "Ready"`
- **AND** `Spec.Template.StorageLocation` 引用的 StorageRepository 资源存在且 `Status.Status == "Available"`
- **THEN** 成功创建 AppBackup 资源
- **AND** 返回 201 Created
