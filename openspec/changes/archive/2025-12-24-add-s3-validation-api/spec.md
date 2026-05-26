# 规范变更：S3 连接验证 API

## ADDED Requirements

### S3 连接验证接口

系统必须提供一个接口来验证 S3 存储配置的有效性。

#### Scenario: 验证有效的 S3 配置
Given 一个包含正确 Endpoint, Region, Bucket, AccessKey, SecretKey 的请求
When 调用 `POST /storages/connection/validate`
Then 返回状态码 200
And 响应体中 `data` 为 `true`

#### Scenario: 验证无效的 S3 凭证
Given 一个包含错误 AccessKey 或 SecretKey 的请求
When 调用 `POST /storages/connection/validate`
Then 返回状态码 200
And 响应体中 `data` 为 `false`
And 响应体 `meta` 中包含具体的错误信息

#### Scenario: 验证不存在的 Bucket
Given 一个包含不存在 Bucket 名称的请求
When 调用 `POST /storages/connection/validate`
Then 返回状态码 200
And 响应体中 `data` 为 `false`
And 响应体 `meta` 中包含具体的错误信息
