## ADDED Requirements

### Requirement: 容灾组聚合必须将 ConfigError 计入 Degraded 触发条件

系统 MUST 在组聚合计算阶段将 `ConfigError` 识别为错误成员状态。

#### Scenario: 组内成员存在 ConfigError 时输出 Degraded

- **GIVEN** 容灾组包含两个成员
- **AND** 成员 A 的 `fsmState == ConfigError`
- **AND** 成员 B 的 `fsmState == Protected`
- **WHEN** 客户端请求 `GET /groups`
- **THEN** 该组 `status.fsmState` 必须为 `Degraded`
- **AND** `status.availableOperations` 必须为空数组

### Requirement: 组成员接口必须保留 ConfigError 语义

系统 MUST 在组成员派生状态中输出 `ConfigError`，不得把配置异常重写成 `Failed`。

#### Scenario: 配置 NotReady 时成员状态为 ConfigError

- **GIVEN** 实例原始 `fsmState == Protected`
- **AND** 实例引用配置 `status == NotReady`
- **WHEN** 客户端请求 `GET /groups/instance-picker`
- **THEN** 响应成员 `status.state` 必须为 `ConfigError`
- **AND** `status.reason` 与 `status.message` 必须包含配置异常语义

### Requirement: error 状态筛选必须命中 ConfigError

系统 MUST 在组列表状态筛选 `status=error` 中命中 `ConfigError` 成员。

#### Scenario: status=error 时返回包含 ConfigError 成员的组

- **GIVEN** 存在一个容灾组，其任意成员 `fsmState == ConfigError`
- **WHEN** 客户端请求 `GET /groups?status=error`
- **THEN** 该容灾组必须出现在结果集
