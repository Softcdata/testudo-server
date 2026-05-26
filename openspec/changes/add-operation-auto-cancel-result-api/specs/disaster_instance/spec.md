## ADDED Requirements

### Requirement: 实例 detail/list 必须暴露自动补偿结果摘要
系统必须 (MUST) 在实例 detail 和 list 中返回自动补偿是否触发、是否成功以及是否仍需人工介入。

#### Scenario: 查询详情时看到自动补偿成功结果
- **Given** 一个失败后的 `Failover` 触发了自动补偿且补偿成功
- **When** 客户端查询该实例或操作详情
- **Then** 响应中必须包含“自动补偿已触发且成功”的稳定摘要字段

#### Scenario: 查询详情时看到自动补偿失败结果
- **Given** 一个失败后的 `Failover` 触发了自动补偿但补偿失败
- **When** 客户端查询该实例或操作详情
- **Then** 响应中必须包含自动补偿失败原因
- **And** 必须标识仍需人工介入

#### Scenario: 列表只返回摘要不展开补偿步骤
- **Given** 一个实例的最新 failover operation 写入了自动补偿状态
- **When** 客户端查询实例列表
- **Then** 列表项必须包含自动补偿摘要
- **And** 不得要求前端自行解析原始 operation message
- **And** 不得在 list 中展开完整补偿步骤明细

### Requirement: 自动补偿摘要必须直接消费 operator 稳定状态字段
系统必须 (MUST) 直接读取 operator 的稳定状态字段生成自动补偿摘要，不得通过自由文本推断。

#### Scenario: Server 不通过 message 猜测补偿结果
- **Given** 一个 `DisasterOperation` 只在状态字段中写入自动补偿结果
- **When** server 生成 detail/list/history/watch DTO
- **Then** DTO 必须正确返回自动补偿摘要
- **And** 不得依赖 message 中是否包含“自动补偿成功/失败”文本

### Requirement: 时间线必须展示自动补偿节点
系统必须 (MUST) 在时间线数据中展示自动补偿的触发和最终结果。

#### Scenario: 时间线包含自动补偿节点
- **Given** 一个 `Failover` 在失败后进入自动补偿路径
- **When** 客户端查询时间线
- **Then** 时间线必须至少包含“失败”“自动补偿触发”“自动补偿成功/失败”三个语义节点

### Requirement: 历史与 watch 视图必须共享同一自动补偿字段口径
系统必须 (MUST) 在实例 history 与组操作 watch 中使用同一套自动补偿字段命名和枚举值。

#### Scenario: History 与 Watch 返回一致的 autoCancel 摘要
- **Given** 一个 group 或 instance failover operation 写入了自动补偿状态
- **When** 客户端分别查询 history 和 watch 视图
- **Then** 两个响应中的自动补偿字段命名和枚举值必须保持一致
