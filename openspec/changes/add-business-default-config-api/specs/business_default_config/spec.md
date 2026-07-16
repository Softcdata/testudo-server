# Capability: 业务默认配置

## ADDED Requirements

### Requirement: 查询业务默认配置快照
系统必须 (MUST) 提供一个接口，用于返回当前业务默认配置快照、每个变量的含义以及每个变量的当前值。

#### Scenario: 前端获取配置页面数据
- **WHEN** 客户端请求 `GET /api/v1/business-default-config`
- **THEN** 服务端必须返回所有业务默认配置分组以及分组内字段
- **AND** 每个字段必须包含 `key`、`name`、`description`、`value`、`defaultValue`、`dataType`、`editable`、`effectMode`
- **AND** 每个字段必须携带可直接展示给前端的含义说明
- **AND** 响应必须包含当前快照的更新时间信息

### Requirement: 搜索业务默认配置字段
系统必须 (MUST) 提供一个接口，用于按关键字搜索业务默认配置字段，并按标准集合响应返回结果。

#### Scenario: 前端按关键字搜索字段
- **WHEN** 客户端请求 `GET /api/v1/business-default-config/fields?keyword=timeout`
- **THEN** 服务端必须返回匹配的字段集合
- **AND** `keyword` 必须对 `key`、`name`、`description`、`value`、`groupKey`、`groupName` 执行模糊匹配
- **AND** 响应必须包含分页元数据
- **AND** 响应必须包含集合链接信息

#### Scenario: 前端按分组和状态筛选字段
- **WHEN** 客户端请求 `GET /api/v1/business-default-config/fields?groupKey=backupRuntime&editable=true`
- **THEN** 服务端必须仅返回匹配该分组且可编辑的字段集合
- **AND** 返回结果必须支持继续按 `effectMode`、`page`、`limit`、`sort` 与 `order` 进行控制

### Requirement: 搜索前端传参字段值目录
系统必须 (MUST) 提供一个只读接口，用于返回前端应从业务默认配置读取并写入业务请求体或 CRD `spec` 的参数 `key`、`value`、字段层级和接口用途。

#### Scenario: 前端搜索 timeout 相关传参字段
- **WHEN** 客户端请求 `GET /api/v1/business-default-config/frontend-fields?keyword=timeout`
- **THEN** 服务端必须返回匹配的前端传参字段集合
- **AND** `keyword` 必须对 `key`、`name`、`description`、`value`、`resourceKind`、`requestPath`、`specPath` 和接口用途执行模糊匹配
- **AND** 每个条目必须包含 `key`、`value`、`requestPathSegments`、`specPathSegments`、`apiUsages`
- **AND** 响应 `data.fieldMap` 必须按字段 `key` 返回当前结果集的字段详情
- **AND** 响应必须包含分页元数据和集合链接信息

#### Scenario: 前端获取全部传参字段 map
- **WHEN** 客户端请求 `GET /api/v1/business-default-config/frontend-fields?limit=-1`
- **THEN** 服务端必须返回全部前端传参字段集合
- **AND** 每个条目必须保留 `key` 与 `value`
- **AND** 每个条目必须返回 `keySegments`、`requestPathSegments`、`specPathSegments`
- **AND** 每个条目必须返回 `apiUsages`，用于说明字段应在哪个 server 接口和哪个请求字段中使用
- **AND** `sort=key` 与 `sort=value` 必须按对应字段排序
- **AND** `data.fieldMap` 必须允许前端通过 `fieldMap["operation.timeoutMinutes"]` 直接读取字段层级和接口用途

### Requirement: 更新业务默认配置
系统必须 (MUST) 提供一个接口，用于更新可编辑的业务默认配置字段。

#### Scenario: 更新一个可编辑字段
- **WHEN** 客户端请求 `PATCH /api/v1/business-default-config`
- **AND** 请求体只携带一个可编辑字段的修改值
- **THEN** 服务端必须合并当前快照并持久化新值
- **AND** 服务端必须返回更新后的完整快照

#### Scenario: 更新只读字段失败
- **WHEN** 客户端请求 `PATCH /api/v1/business-default-config`
- **AND** 请求体携带只读字段
- **THEN** 服务端必须拒绝请求
- **AND** 错误响应必须标明字段路径与拒绝原因

#### Scenario: 更新非法值失败
- **WHEN** 客户端请求 `PATCH /api/v1/business-default-config`
- **AND** 请求体中的字段类型不匹配、枚举值不合法、数值越界
- **THEN** 服务端必须拒绝请求
- **AND** 错误响应必须标明字段路径、期望类型与约束边界
