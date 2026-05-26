## ADDED Requirements

### Requirement: 添加集群时必须拒绝重复 endpoint

系统必须 (MUST) 在 `POST /apis/cluster.testudo.softcdata.com/v1/clusters` 持久化前解析请求的有效 endpoint，并基于归一化结果执行唯一性校验。

#### Scenario: 直接 endpoint 命中重复集群
- **GIVEN** 已存在一个 `Cluster`，其归一化 endpoint 为 `https://192.0.2.170:6443`
- **WHEN** 客户端提交 `POST /apis/cluster.testudo.softcdata.com/v1/clusters`
- **AND** 请求体 `endpoint` 归一化后同样为 `https://192.0.2.170:6443`
- **THEN** 服务端必须 (MUST) 返回 `409 Conflict`
- **AND** 业务码必须 (MUST) 为 `3009`
- **AND** 响应元数据必须 (MUST) 包含 `conflictType=clusterEndpoint`

#### Scenario: kubeconfig 派生 endpoint 命中重复集群
- **GIVEN** 已存在一个 `Cluster`，其归一化 endpoint 为 `https://api.demo.local`
- **WHEN** 客户端提交 `POST /apis/cluster.testudo.softcdata.com/v1/clusters`
- **AND** 请求体未显式提供 `endpoint`
- **AND** 请求体 `kubeConfig` 解析出的 server 归一化后为 `https://api.demo.local`
- **THEN** 服务端必须 (MUST) 返回 `409 Conflict`

### Requirement: Cluster API 必须提供按集群查询已受保护命名空间接口

系统必须 (MUST) 提供按集群返回已受保护命名空间与占用实例明细的只读接口。

#### Scenario: 成功查询指定集群下已受保护命名空间
- **GIVEN** 集群 `cluster-a` 存在
- **WHEN** 客户端请求 `GET /apis/cluster.testudo.softcdata.com/v1/clusters/cluster-a/protected-namespaces`
- **THEN** 服务端必须 (MUST) 返回 `200 OK`
- **AND** 响应必须 (MUST) 使用标准 collection 信封
- **AND** `meta.resourceType` 必须 (MUST) 为 `clusterProtectedNamespace`
- **AND** `data.items[]` 必须 (MUST) 包含 `namespace`
- **AND** `data.items[]` 必须 (MUST) 包含 `cluster`
- **AND** `data.items[]` 必须 (MUST) 包含 `owners`

#### Scenario: 查询结果只聚合同源集群实例
- **GIVEN** 同时存在属于 `cluster-a` 与 `cluster-b` 的 `DisasterInstance`
- **WHEN** 客户端请求 `GET /apis/cluster.testudo.softcdata.com/v1/clusters/cluster-a/protected-namespaces`
- **THEN** 服务端必须 (MUST) 只返回 `sourceCluster=cluster-a` 的实例占用结果

#### Scenario: 查询不存在的集群
- **WHEN** 客户端请求 `GET /apis/cluster.testudo.softcdata.com/v1/clusters/not-exist/protected-namespaces`
- **THEN** 服务端必须 (MUST) 返回 `404 Not Found`
