## ADDED Requirements
### Requirement: OpenAPI 单一事实源
系统必须 (MUST) 提供 OpenAPI 3.0.3 契约文件，文件路径固定为 `openspec/specs/disaster-server-openapi.yaml`。Swagger UI、OpenAPI YAML 输出、OpenAPI JSON 输出、发版校验必须读取同一份契约。

#### Scenario: 读取统一契约
- **WHEN** 发版校验读取 OpenAPI 契约
- **THEN** 必须读取 `openspec/specs/disaster-server-openapi.yaml`
- **AND** 契约中的 `openapi` 字段必须为 `3.0.3`
- **AND** Swagger UI 必须展示同一份契约内容

### Requirement: Swagger 文档路由
系统必须 (MUST) 在 `swagger.enabled=true` 时注册 Swagger 文档路由，并在 `swagger.enabled=false` 时不注册这些路由。

#### Scenario: 启用 Swagger
- **GIVEN** 配置项 `swagger.enabled=true`
- **WHEN** 服务启动并完成路由注册
- **THEN** `GET /openapi.yaml` 必须返回 OpenAPI YAML
- **AND** `GET /openapi.json` 必须返回 OpenAPI JSON
- **AND** `GET /swagger/` 必须返回 Swagger UI 页面
- **AND** 这些路由不得挂载业务 JWT 中间件

#### Scenario: 关闭 Swagger
- **GIVEN** 配置项 `swagger.enabled=false`
- **WHEN** 服务启动并完成路由注册
- **THEN** `GET /openapi.yaml` 不得返回 OpenAPI 契约
- **AND** `GET /openapi.json` 不得返回 OpenAPI 契约
- **AND** `GET /swagger/` 不得返回 Swagger UI 页面

### Requirement: OpenAPI 与 RunAPI 详细说明一致
OpenAPI 每个 operation 的 `description` 必须 (MUST) 使用 RunAPI 详细说明治理中的五段结构，并基于 server 与 operator 调用链确认字段语义。

#### Scenario: 生成接口说明
- **WHEN** 为任一接口补充 OpenAPI operation
- **THEN** `description` 必须包含 `接口用来干什么`
- **AND** `description` 必须包含 `控制哪些资源`
- **AND** `description` 必须包含 `入参详细说明`
- **AND** `description` 必须包含 `返回详细说明`
- **AND** `description` 必须包含 `可能返回的错误`
- **AND** 字段含义必须基于 server handler、request struct、DTO、operator CRD、operator controller 证据确认

### Requirement: 三方接口清单对账
系统必须 (MUST) 建立 server、RunAPI、OpenAPI 三方接口清单对账机制，对账主键固定为 `METHOD + 标准化路径`。

#### Scenario: 发版前对账
- **WHEN** 执行发版接口文档校验
- **THEN** 必须导出 server 全量接口清单
- **AND** 必须导出 RunAPI 全量接口清单
- **AND** 必须导出 OpenAPI 全量接口清单
- **AND** 必须按 `METHOD + 标准化路径` 对账
- **AND** 差异不为 0 时校验结果必须失败

### Requirement: OpenAPI operation 完整性
OpenAPI 中每个 operation 必须 (MUST) 包含可供 Swagger UI 展示、前端集成与发版校验使用的完整字段。

#### Scenario: 校验 operation 字段
- **WHEN** 校验任一 OpenAPI operation
- **THEN** 必须存在 `tags`
- **AND** 必须存在 `summary`
- **AND** 必须存在 `description`
- **AND** 必须存在 `operationId`
- **AND** 必须存在 `parameters`
- **AND** 必须存在 `responses`
- **AND** 必须存在 `security`
- **AND** 必须存在 `x-runapi-target-id`
- **AND** 必须存在 `x-controlled-resources`
- **AND** 必须存在 `x-operator-chain`

### Requirement: WebSocket 接口 OpenAPI 表达
所有 WebSocket watch 接口必须 (MUST) 作为 `GET` upgrade operation 写入 OpenAPI，并使用固定扩展字段声明协议与消息 schema。

#### Scenario: 校验 WebSocket operation
- **WHEN** OpenAPI operation 表示 WebSocket watch 接口
- **THEN** 必须存在 `x-disaster-protocol: websocket`
- **AND** 必须存在 `x-message-schema`
- **AND** `responses` 必须包含 `101`
- **AND** `description` 必须说明监听资源范围、事件类型、心跳消息、认证方式、断开条件

### Requirement: OpenAPI 发版门禁
发版检查必须 (MUST) 包含 OpenAPI 契约校验、Swagger UI 可访问性检查、三方接口清单对账与 WebSocket 扩展字段校验。

#### Scenario: 发版校验失败
- **WHEN** OpenAPI 契约校验失败
- **THEN** 发版检查结果必须失败

#### Scenario: 三方对账失败
- **WHEN** server、RunAPI、OpenAPI 三方接口清单差异不为 0
- **THEN** 发版检查结果必须失败

#### Scenario: Swagger UI 不可访问
- **WHEN** `swagger.enabled=true` 且 `GET /swagger/` 无法返回 Swagger UI 页面
- **THEN** 发版检查结果必须失败
