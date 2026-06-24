## MODIFIED Requirements
### Requirement: 备份数据下载
系统必须 (MUST) 提供 API 允许用户下载指定应用备份下的 Velero 备份数据，且客户端返回结果不得直接暴露对象存储地址。

#### Scenario: 成功获取资源文件下载链接
- **GIVEN** 用户已认证且具有访问权限
- **AND** 指定的 AppBackup 存在
- **AND** 指定的 Velero 备份存在于该 AppBackup 的历史记录中
- **WHEN** 用户请求 `GET /appbackups/:name/backups/:backupName/download`，请求类型为 `resource`
- **THEN** 系统返回平台服务端签发的同源下载地址
- **AND** 该地址可由浏览器直接打开并开始下载
- **AND** 响应中不包含对象存储地址
- **AND** 下载地址有效期默认为 1 小时

#### Scenario: 成功下载持久卷业务数据文件
- **GIVEN** 用户已认证且具有访问权限
- **AND** 指定的 AppBackup 存在
- **AND** 指定的 Velero 备份存在于该 AppBackup 的历史记录中
- **WHEN** 用户请求 `GET /appbackups/:name/backups/:backupName/download?type=data`
- **THEN** 系统返回平台服务端签发的同源下载地址
- **AND** 用户打开该地址后，系统开始流式转发底层的持久卷存储数据文件
- **AND** 客户端只需要访问平台服务端地址

#### Scenario: 成功下载完整应用备份全集
- **GIVEN** 用户已认证且具有访问权限
- **AND** 指定的 AppBackup 存在
- **AND** 指定的 Velero 备份存在于该 AppBackup 的历史记录中
- **WHEN** 用户请求 `GET /appbackups/:name/backups/:backupName/download?type=all`
- **THEN** 系统返回平台服务端签发的同源下载地址
- **AND** 用户打开该地址后，系统开始流式发送包含 Kubernetes 资源文件与持久卷文件的完整归档流
- **AND** 客户端只需要访问平台服务端地址

## ADDED Requirements
### Requirement: 下载票据校验
系统必须 (MUST) 校验下载地址携带的票据是否有效，并在票据失效、篡改、目标资源不匹配时拒绝转发对象存储内容。

#### Scenario: 票据失效拒绝下载
- **GIVEN** 客户端访问的下载地址票据已经过期
- **WHEN** 客户端打开下载地址
- **THEN** 系统返回 403 Forbidden
- **AND** 系统不向对象存储发起读取

#### Scenario: 目标资源不匹配拒绝下载
- **GIVEN** 下载地址票据对应的 AppBackup 名称与请求路径中的名称不一致
- **WHEN** 客户端打开下载地址
- **THEN** 系统返回 403 Forbidden
- **AND** 系统不向对象存储发起读取
