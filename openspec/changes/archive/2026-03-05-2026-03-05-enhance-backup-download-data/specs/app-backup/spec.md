## MODIFIED Requirements

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
