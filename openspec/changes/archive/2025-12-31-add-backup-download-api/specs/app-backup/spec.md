## ADDED Requirements

### Requirement: 备份数据下载
系统必须 (MUST) 提供 API 允许用户下载指定应用备份下的 Velero 备份数据。

#### Scenario: 成功获取备份下载链接
- **GIVEN** 用户已认证且具有访问权限
- **AND** 指定的 AppBackup 存在
- **AND** 指定的 Velero 备份存在于该 AppBackup 的历史记录中
- **WHEN** 用户请求 `GET /appbackups/:name/backups/:backupName/download`
- **THEN** 系统返回备份数据的预签名下载 URL
- **AND** URL 有效期默认为 1 小时

#### Scenario: 备份不存在返回 404
- **GIVEN** 用户请求下载一个不存在的备份
- **WHEN** 系统查询备份记录
- **THEN** 返回 HTTP 404 Not Found
- **AND** 错误消息明确指出备份不存在

#### Scenario: 存储库不可达返回 503
- **GIVEN** MinIO 存储服务不可用
- **WHEN** 用户请求下载备份
- **THEN** 返回 HTTP 503 Service Unavailable
- **AND** 错误消息提示存储服务暂时不可用
