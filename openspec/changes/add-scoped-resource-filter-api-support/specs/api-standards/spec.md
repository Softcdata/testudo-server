## ADDED Requirements
### Requirement: DisasterInstance 恢复策略 scoped 字段透传一致性
Server 在 `DisasterInstance` 创建、更新、详情接口中必须 (MUST) 透传并回显 `restorePolicy.resourceSelection` 的 scoped 四字段。

#### Scenario: 创建实例时透传 scoped 四字段
- **WHEN** 客户端发送 `POST DisasterInstance` 请求，`restorePolicy.resourceSelection` 包含 scoped 四字段
- **THEN** Server 必须把字段写入目标 `DisasterInstance.Spec`
- **AND** 创建响应必须回显同名字段

#### Scenario: 查询实例时回显 scoped 四字段
- **WHEN** 客户端发送 `GET DisasterInstance` 请求
- **THEN** 响应 `data.spec.restorePolicy.resourceSelection` 必须包含 scoped 四字段当前值

### Requirement: DisasterInstance 恢复侧字段优先级规则
Server 在 `DisasterInstance` 创建与更新入口必须 (MUST) 执行恢复侧字段优先级规则。

#### Scenario: includeClusterResources=true 时忽略 scoped 四字段
- **WHEN** `resourceSelection` 同时包含 `includeClusterResources=true` 与 scoped 四字段
- **THEN** Server 必须按 old 路径处理恢复侧配置
- **AND** scoped 四字段不得触发冲突拒绝

#### Scenario: scoped 路径冲突被拒绝
- **WHEN** 进入 scoped 路径后，scoped include 与 scoped exclude 存在同项冲突，或者 `exclude=["*"]` 且 include 非空
- **THEN** Server 必须返回 400
- **AND** 错误消息必须标识冲突项
