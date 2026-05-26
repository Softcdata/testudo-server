## ADDED Requirements

### Requirement: DisasterInstance modifierRules 输入必须对齐 pair-only reversible 契约
系统必须 (MUST) 在实例创建、更新接口中将 `reversible` 规则的正式输入模型收敛为 pair-only，并与 operator 当前契约保持一致。

#### Scenario: 创建实例时提交 pair-only reversible 规则
- **WHEN** 客户端请求 `POST /apis/disasterinstances.testudo.softcdata.com/v1/instances`
- **AND** 请求体中的 `restorePolicy.modifierRules` 包含 `mode=reversible`
- **THEN** reversible 规则必须 (MUST) 使用 `pair.path`、`pair.sourceValue`、`pair.targetValue`

#### Scenario: 通过 modifierRulesText 提交 pair-only reversible 规则
- **WHEN** 客户端请求实例创建或更新接口
- **AND** `restorePolicy.modifierRulesText` 包含 JSON 数组字符串
- **THEN** 服务端必须 (MUST) 将其中的 pair-only reversible 规则解析为结构化 `restorePolicy.modifierRules`

#### Scenario: 提交旧 transform 写法
- **WHEN** 客户端请求实例创建或更新接口
- **AND** reversible 规则仍使用旧 `transform.type=map/template/pair`
- **THEN** 服务端必须 (MUST) 返回 400
- **AND** 错误消息必须 (MUST) 明确指向 `pair.path/sourceValue/targetValue`

### Requirement: DisasterInstance 提交期 live validation 必须读取 pair.path
系统必须 (MUST) 在提交期 live validation 中以 `pair.path` 作为 reversible 规则的目标 JSON Pointer 路径。

#### Scenario: reversible 规则参与 live validation
- **WHEN** 服务端对实例请求执行 live rule validation
- **AND** 规则为 `mode=reversible`
- **THEN** 服务端必须 (MUST) 从 `pair.path` 提取待校验路径
- **AND** 不得 (MUST NOT) 再要求 `transform.path`
