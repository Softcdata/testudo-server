## MODIFIED Requirements
### Requirement: 创建集群必须执行平台 License 前置门禁
系统必须 (MUST) 在 `POST /apis/cluster.testudo.softcdata.com/v1/clusters` 创建 Cluster CR 前执行平台 License 门禁。门禁必须读取 License Secret、重新计算当前平台指纹并校验签名、产品、版本、公钥、时间窗口、指纹与额度字段，不得依赖展示用 License 状态 ConfigMap。

#### Scenario: 免费版创建前数量未超过限制
- **GIVEN** 当前不存在有效企业 License
- **AND** 创建前未删除 Cluster 数量小于 2
- **WHEN** 客户端创建 Cluster
- **THEN** server 必须继续执行原有创建流程

#### Scenario: 免费版创建前数量达到限制
- **GIVEN** 当前不存在有效企业 License
- **AND** 创建前未删除 Cluster 数量等于 2
- **WHEN** 客户端创建 Cluster
- **THEN** server 必须拒绝请求
- **AND** HTTP 状态必须为 403
- **AND** 响应 `meta.reason` 必须为 `LicenseLimitExceeded`
- **AND** 响应 `meta.maxClusters` 必须为 2
- **AND** 响应 `meta.currentClusters` 必须为 2

#### Scenario: 有效企业 License 不限制集群数量
- **GIVEN** 当前存在有效企业 License
- **AND** License 中 `limits.maxClusters` 为 -1
- **WHEN** 客户端创建 Cluster
- **THEN** server 必须允许创建流程继续

#### Scenario: License 状态缓存被篡改
- **GIVEN** License 状态 ConfigMap 被改成 `Active` 与无限制
- **AND** License Secret 缺失或无效
- **AND** 创建前未删除 Cluster 数量等于 2
- **WHEN** 客户端创建 Cluster
- **THEN** server 必须按免费版限制拒绝请求
