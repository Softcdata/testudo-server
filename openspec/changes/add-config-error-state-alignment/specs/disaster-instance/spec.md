## ADDED Requirements

### Requirement: 实例 DTO currentState 必须识别 ConfigError

系统 MUST 在实例 DTO 映射阶段将 `fsmState=ConfigError` 映射为 `currentState=Error`。

#### Scenario: ConfigError 在 currentState 映射为 Error

- **GIVEN** `DisasterInstance.status.fsmState == ConfigError`
- **WHEN** 客户端请求实例列表
- **THEN** 响应中该实例 `currentState` 必须为 `Error`

### Requirement: 实例原始 fsmState 必须保持透传

系统 MUST 在实例 DTO 输出中保留原始 `status.fsmState`，不得将 `ConfigError` 改写为 `Failed`。

#### Scenario: ConfigError 原始状态保持不变

- **GIVEN** `DisasterInstance.status.fsmState == ConfigError`
- **WHEN** 客户端请求实例详情
- **THEN** 响应中 `status.fsmState` 必须为 `ConfigError`
