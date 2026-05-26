# Capability: Statistics API 规范

## Purpose
This capability defines the API interface specifications for the Statistics feature, which provides aggregated statistical data for disaster recovery systems (including backup and restore operations). It is designed to expose the underlying BackupRestoreStatistics data securely and efficiently to front-end consumers.
## Requirements
### Requirement: 统计信息查询
系统必须 (MUST) 提供 RESTful API 以获取聚合的备份和恢复统计数据。

#### Scenario: 获取备份统计概览
- **WHEN** 发送 GET 请求到 `/apis/statistics/v1/backups`
- **AND** (可选) 提供查询参数 `namespace` 指定命名空间
- **THEN** 查询 Kubernetes 中所有带有标签 `testudo.softcdata.com/owner-kind=AppBackup` 的 `BackupRestoreStatistics` 资源
- **AND** 聚合所有记录的 `Total`, `InProgress`, `Completed`, `Failed` 等字段
- **AND** 返回聚合后的统计数据 JSON 对象
- **AND** 响应状态码为 200

#### Scenario: 获取恢复统计概览
- **WHEN** 发送 GET 请求到 `/apis/statistics/v1/restores`
- **AND** (可选) 提供查询参数 `namespace` 指定命名空间
- **THEN** 查询 Kubernetes 中所有带有标签 `testudo.softcdata.com/owner-kind=AppRestore` 的 `BackupRestoreStatistics` 资源
- **AND** 聚合所有记录的 `Total`, `InProgress`, `Completed`, `Failed` 等字段
- **AND** 返回聚合后的统计数据 JSON 对象
- **AND** 响应状态码为 200

### Requirement: 数据模型
API 响应的数据模型 MUST 包含以下字段以确保一致性：

#### Scenario: 返回标准聚合结构
- **WHEN** 客户端成功发起统计请求
- **THEN** 服务端返回定义的标准模型格式

```json
{
  "total": 10,
  "inProgress": 2,
  "completed": 7,
  "failed": 1,
  "canceled": 0,
  "unknown": 0
}
```



## Implementation Notes

### 1. 依赖资源
后端服务需要访问 Kubernetes 中的 `BackupRestoreStatistics` (Group: `testudo.softcdata.com`, Version: `v1`) 资源。

### 2. 聚合逻辑
由于 Operator 采用客户端聚合策略，Server 端需要实现以下逻辑：
1.  使用 Controller-Runtime Client 或 Client-Go。
2.  List `BackupRestoreStatistics`。
3.  LabelSelector: `testudo.softcdata.com/owner-kind` = `AppBackup` (或 `AppRestore`)。
4.  遍历 List 结果，累加 `status.statistics` 中的各项数值。

### 3. 路由设计
建议在 `internal/apis/statistics` 包中实现 Handler，并在 `internal/router` 中注册路由。
