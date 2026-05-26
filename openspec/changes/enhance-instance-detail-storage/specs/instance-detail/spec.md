# 规范增量：容灾实例详情回显存储仓信息

## 变更描述
在容灾实例详情接口中增加存储仓详情的回显，确保前端可以一次性获取到完整的容灾拓扑配置。

## ADDED Requirements

### Requirement: 容灾实例详情聚合存储信息
接口 `GET /apis/testudo.softcdata.com/v1/namespaces/:namespace/disasterinstances/:name` 的响应数据模型 MUST 包含关联的存储仓详情。

#### 约束
1. 聚合的存储信息 MUST 来源于 `DisasterConfig` 中定义的 `storageRepository`。
2. 必须使用安全转换逻辑，禁止返回存储仓的敏感凭证（如 `accessKey`, `secretKey`）。
3. 如果关联的存储仓不存在或被删除，该字段应返回 `null`，不应导致接口报错。

#### Scenario: 成功获取包含存储信息的实例详情
- **WHEN**: 存在一个名为 `inst-1` 的容灾实例，其配置 `conf-1` 指向存储仓 `minio-1`。调用详情接口请求 `inst-1`。
- **THEN**: 响应 JSON 中的 `storage` 字段应包含 `name: "minio-1"`、`spec.bucket`、`spec.endpoint` 等非敏感信息。

#### Scenario: 过滤敏感信息
- **WHEN**: 存储仓 `minio-1` 包含 `accessKey` 为 "admin" 和 `secretKey` 为 "password"。调用详情接口获取实例详情。
- **THEN**: 响应 JSON 中的 `storage` 字段下不应出现 `accessKey` 和 `secretKey` 字段。
