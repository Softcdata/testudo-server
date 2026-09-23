## ADDED Requirements

### Requirement: Cluster API 管理专用 BSL endpoint

Cluster API MUST 支持在 `spec.veleroInstall.bslEndpoint` 保存当前物理集群访问共享 S3/MinIO 的专用 endpoint。该字段不属于主备角色，不从源集群或 BSL 对象推断。

#### Scenario: 创建集群保存 endpoint

- **WHEN** 客户端创建 Cluster 并传入 `veleroInstall.bslEndpoint` 非空值
- **THEN** server MUST 将该值写入 `Cluster.spec.veleroInstall.bslEndpoint`
- **AND** 创建响应 MUST 返回该非敏感字段

#### Scenario: 查询回显 endpoint

- **WHEN** 客户端请求 Cluster 详情、列表或 watch
- **THEN** `spec.veleroInstall.bslEndpoint` MUST 与 Cluster CR 当前值一致
- **AND** 响应 MUST NOT 返回 registry password 或 dockerconfigjson 原文

#### Scenario: PATCH 设置和保持 endpoint

- **WHEN** PATCH 传入非空 `veleroInstall.bslEndpoint`
- **THEN** server MUST 更新 Cluster CR 的 endpoint
- **WHEN** PATCH 未传 `bslEndpoint`
- **THEN** server MUST 保持已有 endpoint 不变

#### Scenario: PATCH 清除 endpoint

- **WHEN** PATCH 传入 `veleroInstall.bslEndpoint` 空字符串
- **THEN** server MUST 清除该字段
- **AND** 若 `imageRegistry` 和凭据引用仍存在，MUST 保留其余 VeleroInstall 配置

#### Scenario: 校验 endpoint

- **WHEN** 请求中的 endpoint 不是带 host 的绝对 `http` 或 `https` URL，或包含 userinfo、query、fragment
- **THEN** server MUST 返回 HTTP 400
- **AND** MUST NOT 写入或修改 Cluster
