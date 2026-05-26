## ADDED Requirements

### Requirement: 恢复 Class 列表接口必须遵循统一响应信封与双层错误码

系统必须 (MUST) 对
`GET /apis/cluster.testudo.softcdata.com/v1/clusters/:name/restore-classes`
返回统一 `Envelope`，并遵循双层错误码规范。

#### Scenario: 成功响应使用统一 Envelope
- **WHEN** 客户端成功请求恢复 Class 列表接口
- **THEN** HTTP 状态码必须 (MUST) 为 200
- **AND** 响应体必须 (MUST) 包含 `code=0`、`message`、`data`、`trace_id`
- **AND** `data` 必须 (MUST) 包含 `targetCluster`、`storageClasses`、`ingressClasses`

#### Scenario: 集群不存在时返回双层错误码
- **GIVEN** 请求中的 `:name` 对应集群不存在
- **WHEN** 服务端处理该请求
- **THEN** HTTP 状态码必须 (MUST) 为 404
- **AND** 业务错误码必须 (MUST) 为 `CodeNotFound`

#### Scenario: 参数不合法时返回双层错误码
- **GIVEN** 请求参数不合法
- **WHEN** 服务端处理该请求
- **THEN** HTTP 状态码必须 (MUST) 为 400
- **AND** 业务错误码必须 (MUST) 为 `CodeBadRequest`
