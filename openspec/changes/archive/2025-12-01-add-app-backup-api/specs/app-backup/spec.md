## ADDED Requirements

### Requirement: AppBackup 资源管理
系统 SHALL 提供 RESTful API 以管理 `AppBackup` Kubernetes 自定义资源。

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
API 交互的数据模型 SHALL 与 `disaster-operator` 定义的 `AppBackup` CRD 保持一致。

#### Scenario: 使用 Operator 类型
- **WHEN** 处理 API 请求和响应
- **THEN** 必须使用 `github.com/softcdata/testudo-operator/pkg/apis/disaster/v1` 包中的 `AppBackup` 和 `AppBackupList` 类型
