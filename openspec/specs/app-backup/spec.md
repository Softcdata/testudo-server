# Capability: AppBackup 规范

## Purpose
定义 `AppBackup` 资源的管理规范，包括 API 接口、数据模型和交互行为。
## Requirements
### Requirement: AppBackup 资源管理
系统必须 (MUST) 提供 RESTful API 以管理 `AppBackup` Kubernetes 自定义资源。

#### Scenario: 获取 AppBackup 列表
- **WHEN** 发送 GET 请求到 `/apis/appbackups.<group>/<version>/backups`
- **THEN** 返回所有 `AppBackup` 资源的列表
- **AND** 响应状态码为 200

#### Scenario: 创建 AppBackup
- **WHEN** 发送 POST 请求到 `/apis/appbackups.<group>/<version>/backups`
- **AND** 请求体包含符合 `AppBackup` CRD 定义的 JSON 数据
- **THEN** 在 Kubernetes 中创建相应的资源
- **AND** 返回创建的资源对象和状态码 201

#### Scenario: 监听 AppBackup 变更
- **WHEN** 发送 GET 请求到 `/apis/appbackups.<group>/<version>/watch/backups`
- **THEN** 建立长连接并流式传输资源变更事件

### Requirement: 数据模型一致性
API 交互的数据模型必须 (MUST) 与 `disaster-operator` 定义的 `AppBackup` CRD 保持一致。

#### Scenario: 使用 Operator 类型
- **WHEN** 处理 API 请求和响应
- **THEN** 必须使用 `github.com/softcdata/testudo-operator/pkg/apis/disaster/v1` 包中的 `AppBackup` 和 `AppBackupList` 类型

### Requirement: 资源依赖验证
在创建 AppBackup 之前，系统必须 (MUST) 验证所有依赖资源的存在性和状态。

#### Scenario: 验证 Cluster 就绪
- **WHEN** 创建 AppBackup
- **THEN** 系统检查指定的 `Cluster` 资源是否存在
- **AND** 检查 `Cluster` 的状态是否为 `Ready`
- **AND** 如果验证失败，拒绝创建请求并返回 400 错误

#### Scenario: 验证 StorageRepository 可用
- **WHEN** 创建 AppBackup
- **THEN** 系统检查指定的 `StorageRepository` 资源是否存在
- **AND** 检查 `StorageRepository` 的状态是否为 `Available`
- **AND** 如果验证失败，拒绝创建请求并返回 400 错误

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

### Requirement: 备份数据下载
系统必须 (MUST) 提供 API 允许用户下载指定应用备份下的 Velero 备份数据，且支持下载真实的持久卷数据文件。

#### Scenario: 成功获取资源文件下载链接
- **GIVEN** 用户已认证且具有访问权限
- **AND** 指定的 AppBackup 存在
- **AND** 指定的 Velero 备份存在于该 AppBackup 的历史记录中
- **WHEN** 用户请求 `GET /appbackups/:name/backups/:backupName/download` （默认）或带有参数 `type=resource`
- **THEN** 系统返回备份元数据与清单的预签名下载 URL
- **AND** URL 有效期默认为 1 小时

#### Scenario: 成功下载持久卷业务数据文件
- **GIVEN** 用户已认证且具有访问权限
- **AND** 指定的 AppBackup 存在
- **AND** 指定的 Velero 备份存在于该 AppBackup 的历史记录中
- **WHEN** 用户请求 `GET /appbackups/:name/backups/:backupName/download?type=data`
- **THEN** 系统开始流式代理转发底层的持久卷存储数据文件（如 restic/kopia 相关数据块文件）
- **AND** 数据以打包流 (Raw Tar 不含二次压缩，避免高负载) 格式零拷贝传送至客户端

#### Scenario: 成功下载完整应用备份全集
- **GIVEN** 用户已认证且具有访问权限
- **AND** 指定的 AppBackup 存在
- **AND** 指定的 Velero 备份存在于该 AppBackup 的历史记录中
- **WHEN** 用户请求 `GET /appbackups/:name/backups/:backupName/download?type=all`
- **THEN** 系统开始流式打包并发送包含 Kubernetes 资源文件与持久卷文件的完整归档流至客户端

### Requirement: AppBackup 描述字段处理
系统必须 (MUST) 正确处理 `AppBackup` 资源的 `description` 字段，允许更新操作进行设置、修改或清除描述。

#### Scenario: 清除描述
- **WHEN** 客户端发送 `description: ""` (空字符串) 的更新请求
- **THEN** 资源上的 `testudo.softcdata.com/app-backup-description` 注解必须 (MUST) 被移除
- **AND** 响应中的 description 字段必须为空

#### Scenario: 更新描述
- **WHEN** 客户端发送带有非空 `description` 的更新请求
- **THEN** `testudo.softcdata.com/app-backup-description` 注解必须 (MUST) 更新为新值

