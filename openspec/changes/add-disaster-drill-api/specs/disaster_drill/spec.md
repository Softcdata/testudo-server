## ADDED Requirements

### Requirement: DisasterDrill CRUD API

系统必须 (MUST) 提供容灾演练 (DisasterDrill) 的 CRUD API，允许用户创建、查询、确认执行和删除演练。

#### Scenario: 创建容灾演练
- **GIVEN** 一个处于 `Protected` 状态的 DisasterInstance
- **WHEN** 客户端发送 `POST /apis/v1/drills` 请求，包含 `instanceName` 字段
- **THEN** 系统应创建一个 DisasterDrill CR
- **AND** 返回状态码 `201 Created`
- **AND** 响应包含新创建的 DisasterDrillDTO

#### Scenario: 列出所有演练
- **WHEN** 客户端发送 `GET /apis/v1/drills` 请求
- **THEN** 系统应返回所有 DisasterDrill 资源的列表
- **AND** 返回状态码 `200 OK`
- **AND** 响应符合集合响应标准 (包含 `data` 数组和 `pagination` 元数据)

#### Scenario: 按实例过滤演练列表
- **WHEN** 客户端发送 `GET /apis/v1/drills?instanceName=my-app-dr` 请求
- **THEN** 系统应仅返回 `instanceName` 匹配的演练
- **AND** 返回状态码 `200 OK`

#### Scenario: 获取演练详情
- **GIVEN** 存在一个名为 `drill-001` 的 DisasterDrill
- **WHEN** 客户端发送 `GET /apis/v1/drills/drill-001` 请求
- **THEN** 系统应返回该演练的完整信息
- **AND** 返回状态码 `200 OK`
- **AND** 响应包含 DisasterDrillDTO

#### Scenario: 获取不存在的演练
- **WHEN** 客户端发送 `GET /apis/v1/drills/not-exist` 请求
- **THEN** 系统应返回状态码 `404 Not Found`
- **AND** 响应 `code` 字段为 `3004` (CodeNotFound)

#### Scenario: 删除演练
- **GIVEN** 存在一个演练 `drill-001`
- **WHEN** 客户端发送 `DELETE /apis/v1/drills/drill-001` 请求
- **THEN** 系统应删除该 DisasterDrill CR
- **AND** 返回状态码 `200 OK`
- **AND** 关联的 DisasterOperation 应被级联删除

### Requirement: 演练确认执行 API

系统必须 (MUST) 提供确认执行演练的 API，支持两阶段演练流程。

#### Scenario: 确认执行处于 Ready 状态的演练
- **GIVEN** 存在一个处于 `Ready` 状态的 DisasterDrill `drill-001`
- **WHEN** 客户端发送 `POST /apis/v1/drills/drill-001/confirm` 请求
- **THEN** 系统应 Patch DisasterDrill 的 `spec.confirmed` 为 `true`
- **AND** 返回状态码 `200 OK`
- **AND** 演练应进入 `Executing` 状态

#### Scenario: 确认执行非 Ready 状态的演练
- **GIVEN** 存在一个处于 `Pending` 状态的 DisasterDrill
- **WHEN** 客户端发送 `POST /apis/v1/drills/{name}/confirm` 请求
- **THEN** 系统应拒绝请求
- **AND** 返回状态码 `400 Bad Request`
- **AND** 响应消息说明演练必须在 Ready 状态才能确认

### Requirement: 演练 DTO 格式

API 返回的演练数据必须 (MUST) 使用 DTO 格式，仅包含必要的业务字段。

#### Scenario: DisasterDrillDTO 结构
- **WHEN** 返回 DisasterDrillDTO
- **THEN** 必须包含以下字段:
  - `name`: 演练名称
  - `instanceName`: 关联的容灾实例名称
  - `state`: 当前状态 (Pending, Ready, Executing, Completed, Failed)
  - `message`: 状态消息
- **AND** 应包含以下可选字段:
  - `targetCluster`: 目标集群
  - `namespaceMapping`: 命名空间映射
  - `restoreMode`: 恢复模式 (FullRestore)
  - `validationResults`: 校验结果
  - `startTime`, `readyTime`, `executionTime`, `completionTime`: 时间戳

### Requirement: 创建演练请求验证

系统必须 (MUST) 验证创建演练的请求参数。

#### Scenario: 缺少必填字段
- **WHEN** 客户端发送 `POST /apis/v1/drills` 请求，但不包含 `instanceName`
- **THEN** 系统应拒绝请求
- **AND** 返回状态码 `400 Bad Request`
- **AND** 响应 `code` 字段为 `1000` (CodeBadRequest)

#### Scenario: 关联实例不存在
- **WHEN** 客户端发送 `POST /apis/v1/drills` 请求，`instanceName` 指向不存在的实例
- **THEN** 系统应拒绝请求
- **AND** 返回状态码 `404 Not Found`
- **AND** 响应消息说明实例不存在

### Requirement: 受保护命名空间查询 API

系统必须 (MUST) 提供在创建演练前查询受保护命名空间的接口，支持按容灾实例或容灾组查询。

#### Scenario: 按容灾实例查询受保护命名空间
- **WHEN** 客户端发送 `GET /apis/v1/drills/actions/protected-namespaces?instanceName=my-app-dr`
- **THEN** 系统应读取该 `DisasterInstance.spec.namespaces`
- **AND** 返回去重后的命名空间列表
- **AND** 返回状态码 `200 OK`

#### Scenario: 按容灾组查询受保护命名空间
- **GIVEN** `DisasterGroup.spec.levels` 包含多个实例
- **WHEN** 客户端发送 `GET /apis/v1/drills/actions/protected-namespaces?groupName=my-group`
- **THEN** 系统应遍历组内实例并读取各实例 `spec.namespaces`
- **AND** 返回聚合去重后的命名空间列表
- **AND** 返回状态码 `200 OK`

#### Scenario: 请求参数互斥校验
- **WHEN** 客户端同时不传或同时传入 `instanceName` 与 `groupName`
- **THEN** 系统应拒绝请求
- **AND** 返回状态码 `400 Bad Request`
