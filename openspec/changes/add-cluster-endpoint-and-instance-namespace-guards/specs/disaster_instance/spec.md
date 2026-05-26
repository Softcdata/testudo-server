## ADDED Requirements

### Requirement: 实例写入必须拒绝同源集群命名空间冲突

系统必须 (MUST) 在 `DisasterInstance` 创建与更新时，按 `DisasterConfig.spec.sourceCluster` 检查受保护命名空间是否与其他实例冲突。

#### Scenario: 创建实例时命中同源集群命名空间冲突
- **GIVEN** 已存在一个 `DisasterInstance`
- **AND** 该实例关联的 `DisasterConfig.spec.sourceCluster=cluster-a`
- **AND** 该实例已经保护命名空间 `app-a`
- **WHEN** 客户端提交新的 `POST /apis/disasterinstances.testudo.softcdata.com/v1/instances`
- **AND** 新实例关联的 `DisasterConfig.spec.sourceCluster=cluster-a`
- **AND** 新实例请求中的 `spec.namespaces` 包含 `app-a`
- **THEN** 服务端必须 (MUST) 返回 `409 Conflict`
- **AND** 业务码必须 (MUST) 为 `3009`
- **AND** 响应元数据必须 (MUST) 包含 `conflictType=protectedNamespaces`

#### Scenario: 不同源集群允许同名命名空间
- **GIVEN** 已存在一个 `DisasterInstance`
- **AND** 该实例关联的 `DisasterConfig.spec.sourceCluster=cluster-a`
- **AND** 该实例已经保护命名空间 `app-a`
- **WHEN** 客户端提交新的 `POST /apis/disasterinstances.testudo.softcdata.com/v1/instances`
- **AND** 新实例关联的 `DisasterConfig.spec.sourceCluster=cluster-b`
- **AND** 新实例请求中的 `spec.namespaces` 同样包含 `app-a`
- **THEN** 服务端不得 (MUST NOT) 仅因该同名命名空间拒绝请求

### Requirement: 更新实例时必须排除当前实例自身

系统必须 (MUST) 在更新 `DisasterInstance` 时排除当前实例自身，避免把自有命名空间误判为冲突。

#### Scenario: 更新实例保留自身命名空间
- **GIVEN** 存在一个 `DisasterInstance inst-a`
- **AND** `inst-a` 当前已经保护命名空间 `app-a`
- **WHEN** 客户端请求 `PUT /apis/disasterinstances.testudo.softcdata.com/v1/instances/inst-a`
- **AND** 更新后的 `spec.namespaces` 仍包含 `app-a`
- **THEN** 服务端不得 (MUST NOT) 将 `inst-a` 自身视为冲突来源

#### Scenario: 更新实例改成他人已占用命名空间
- **GIVEN** 存在 `inst-a` 与 `inst-b`
- **AND** `inst-b` 在同一 `sourceCluster` 下已经保护命名空间 `app-b`
- **WHEN** 客户端请求 `PUT /apis/disasterinstances.testudo.softcdata.com/v1/instances/inst-a`
- **AND** 更新后的 `spec.namespaces` 包含 `app-b`
- **THEN** 服务端必须 (MUST) 返回 `409 Conflict`
- **AND** 响应元数据必须 (MUST) 包含冲突命名空间与占用实例明细
