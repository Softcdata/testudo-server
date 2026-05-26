# Spec: 集群认证

## MODIFIED Requirements

### Cluster Connection
系统必须支持使用 KubeConfig 或 Token 连接到 Kubernetes 集群。

#### Scenario: 使用 Token 连接
Given 具有有效 API 端点和 Token 的集群配置
When 系统尝试连接到集群
Then 连接必须成功建立
And 系统必须能够获取集群版本

#### Scenario: 验证 Token
Given 具有无效 Token 的集群配置
When 系统尝试验证集群连接
Then 验证必须失败
And API 必须返回 HTTP 200 OK
And 响应体中的 `data` 字段必须为 `false`
And 响应体中的 `msg` 字段必须包含错误详情
