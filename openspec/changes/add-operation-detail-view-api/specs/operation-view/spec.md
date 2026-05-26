## ADDED Requirements

### Requirement: history 记录必须携带稳定的操作标识
系统必须 (MUST) 在 instance history 与 group history 的每条记录中返回可用于打开详情的稳定标识。

#### Scenario: 查询 instance history
- **Given** 某个 instance 已经产生多条 `DisasterOperation`
- **When** 客户端查询 `GET /disasterinstances.testudo.softcdata.com/v1/instances/:name/history`
- **Then** 每条记录必须返回 `operationName`
- **And** 每条记录必须返回 `operationUID`
- **And** 每条记录必须返回 `hasDetail`

#### Scenario: 查询 group history
- **Given** 某个 group 已经产生多条 `DisasterOperation`
- **When** 客户端查询 `GET /disastergroups.testudo.softcdata.com/v1/groups/:name/history`
- **Then** 每条记录必须返回 `operationName`
- **And** 每条记录必须返回 `operationUID`
- **And** 每条记录必须返回 `hasDetail`

### Requirement: 系统必须提供 owner-scoped operation detail route
系统必须 (MUST) 提供按 owner 归属查询单次操作详情的 route，并直接返回步骤级详情。

#### Scenario: 查询 instance operation detail
- **Given** 某条 instance history 记录带有 `operationName`
- **When** 客户端查询 `GET /disasterinstances.testudo.softcdata.com/v1/instances/:name/operations/:operationName`
- **Then** 响应必须包含 `currentStep`
- **And** 响应必须包含 `steps[]`
- **And** 响应必须包含 `state`
- **And** 响应必须直接读取 operator 状态字段，不得通过 `message` 文本推断步骤

#### Scenario: 查询 group operation detail
- **Given** 某条 group history 记录带有 `operationName`
- **When** 客户端查询 `GET /disastergroups.testudo.softcdata.com/v1/groups/:name/operations/:operationName`
- **Then** 响应必须包含 `groupStatus`
- **And** 响应必须包含 `currentStep`
- **And** 响应必须包含 `steps[]`

#### Scenario: owner 不匹配时返回 404
- **Given** `operationName` 存在，但并不属于路径中的 owner
- **When** 客户端请求对应 detail route
- **Then** 服务端必须返回 `404`

#### Scenario: 详情接口遵循既有子资源路径风格
- **Given** 服务端需要为 instance 或 group 提供 operation detail
- **When** 定义 route
- **Then** 必须采用 owner-scoped 子资源路径
- **And** 不得新增全局 `/operations/:operationName` 查询接口

### Requirement: 新增查询接口必须遵循现有 API 标准
系统必须 (MUST) 让新增的 operation history/detail/watch 查询接口遵循现有 API 标准，包括 `GET` 查询语义、标准 `Envelope` 与 DTO 返回。

#### Scenario: 查询 operation detail
- **Given** 客户端调用 operation detail route
- **When** 请求成功
- **Then** 服务端必须使用标准 `Envelope` 返回 DTO
- **And** 不得直接透传原始 `DisasterOperation` CR

#### Scenario: 查询接口不得产生副作用
- **Given** 客户端调用 operation detail route 或 history route
- **When** 服务端处理请求
- **Then** 服务端不得在该查询过程中修改 `DisasterOperation`、`DisasterInstance` 或 `DisasterGroup`

### Requirement: 运行中的操作必须支持单操作 watch
系统必须 (MUST) 为 running 的 instance 操作提供单操作 watch，并与现有 group watch 共享同一字段口径。

#### Scenario: 订阅 running 的 instance operation
- **Given** 某条 instance history 记录的状态为 `Pending`
- **When** 客户端连接 `GET /disasterinstances.testudo.softcdata.com/v1/watch/instances/operations/:operationName`
- **Then** 推送 DTO 必须包含 `currentStep`
- **And** 推送 DTO 必须包含 `steps[]`
- **And** 推送 DTO 的字段命名必须与 group single-operation watch 保持一致

### Requirement: P1 Drill 详情必须被视为现有可复用能力
系统必须 (MUST) 把现有 drill detail 的步骤字段明确标记为 P1 可直接复用能力，不得重复设计第二套 drill detail 接口。

#### Scenario: 评审 P1 快速收益
- **Given** 评审者查看“容灾操作步骤可查看”提案
- **When** 对照 P1
- **Then** 提案必须明确说明现有 drill detail 已包含 `status.currentStep` 与 `steps[]`
- **And** 提案不得要求新增第二套 drill detail route
