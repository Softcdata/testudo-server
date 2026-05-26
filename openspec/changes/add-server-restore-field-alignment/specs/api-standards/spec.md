## ADDED Requirements

### Requirement: DisasterInstance 接口必须支持实例级恢复字段透传

系统必须 (MUST) 在实例创建、更新、查询接口中支持并回显恢复相关字段，保证 server 与 operator 字段语义一致。

#### Scenario: 创建实例时提交恢复字段
- **WHEN** 客户端请求 `POST /apis/disasterinstances.testudo.softcdata.com/v1/instances`
- **AND** 请求体包含顶层字段 `skipPodReadyCheck` 以及 `restorePolicy`
- **THEN** 服务端必须 (MUST) 将字段写入 `DisasterInstance.spec`
- **AND** 创建响应中的 `data.spec` 必须 (MUST) 包含相同字段

#### Scenario: 更新实例时修改恢复字段
- **WHEN** 客户端请求 `PUT /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name`
- **AND** 请求体包含顶层字段 `skipPodReadyCheck` 或 `restorePolicy`
- **THEN** 服务端必须 (MUST) 更新对应字段
- **AND** 查询接口返回值必须 (MUST) 反映更新结果

#### Scenario: 创建实例时通过文本传入 modifierRules
- **WHEN** 客户端请求 `POST /apis/disasterinstances.testudo.softcdata.com/v1/instances`
- **AND** `restorePolicy.modifierRulesText` 包含 JSON 数组字符串（如 `"[{\"id\":\"rule1\"}]"`）
- **THEN** 服务端必须 (MUST) 解析文本为结构化 `restorePolicy.modifierRules`
- **AND** 当文本中包含 `reversible` 规则时，必须 (MUST) 使用 `pair.path`、`pair.sourceValue`、`pair.targetValue`
- **AND** 解析失败必须 (MUST) 返回 400

#### Scenario: 更新实例时通过文本传入 modifierRules
- **WHEN** 客户端请求 `PUT /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name`
- **AND** 请求体包含 `restorePolicy.modifierRulesText`
- **THEN** 服务端必须 (MUST) 在更新前解析文本
- **AND** 当文本中包含 `reversible` 规则时，必须 (MUST) 使用 `pair.path`、`pair.sourceValue`、`pair.targetValue`
- **AND** 解析失败必须 (MUST) 返回 400

#### Scenario: modifierRules 与 modifierRulesText 冲突
- **WHEN** 客户端请求实例创建或更新接口
- **AND** `restorePolicy.modifierRules` 与 `restorePolicy.modifierRulesText` 同时存在
- **AND** 两者语义不一致
- **THEN** 服务端必须 (MUST) 返回 400
- **AND** 错误消息必须 (MUST) 明确指出冲突（如 `ModifierRulesInputConflict`）

### Requirement: 实例与组 action 接口必须提供确定性的就绪校验参数透传

系统必须 (MUST) 在实例 action 与组 action 接口中支持 `skipPodReadyCheck`，并与 `waitUntilReady` 保持确定性映射。

#### Scenario: action 请求显式传入 skipPodReadyCheck
- **WHEN** 客户端请求实例 action 接口或组 action 接口
- **AND** 请求体包含 `config.skipPodReadyCheck=true`
- **THEN** 服务端创建的 `DisasterOperation.spec.skipPodReadyCheck` 必须 (MUST) 为 `true`
- **AND** 服务端创建的 `DisasterOperation.spec.waitUntilReady` 必须 (MUST) 为 `false`

#### Scenario: action 请求仅传入 waitUntilReady
- **WHEN** 客户端请求实例 action 接口或组 action 接口
- **AND** 请求体包含 `config.waitUntilReady=true`
- **AND** 请求体不包含 `config.skipPodReadyCheck`
- **THEN** 服务端必须 (MUST) 计算 `skipPodReadyCheck=false`
- **AND** 服务端创建的 `DisasterOperation.spec.waitUntilReady` 必须 (MUST) 为 `true`

#### Scenario: skipPodReadyCheck 与 waitUntilReady 同时传入
- **WHEN** 客户端请求实例 action 接口或组 action 接口
- **AND** 请求体同时包含 `config.skipPodReadyCheck` 与 `config.waitUntilReady`
- **THEN** 服务端必须 (MUST) 以 `config.skipPodReadyCheck` 作为最终覆盖输入

### Requirement: AppRestore 接口必须提供一致的 Class 映射字段契约

系统必须 (MUST) 在 `AppRestore` 创建与更新接口中使用一致的 SC 映射语义，避免同义字段分裂。

#### Scenario: 使用规范字段 storageClassMapping 创建恢复
- **WHEN** 客户端请求 `POST /apis/apprestores.testudo.softcdata.com/v1/apprestores`
- **AND** 请求体包含 `storageClassMapping`
- **THEN** 服务端必须 (MUST) 接收该字段并转换为 `resourceModifierRules`

#### Scenario: 使用兼容字段 scMapping 创建恢复
- **WHEN** 客户端请求 `POST /apis/apprestores.testudo.softcdata.com/v1/apprestores`
- **AND** 请求体包含 `scMapping`
- **AND** 请求体不包含 `storageClassMapping`
- **THEN** 服务端必须 (MUST) 按与 `storageClassMapping` 相同语义处理

#### Scenario: 两个 SC 映射字段冲突
- **WHEN** 客户端请求 `POST` 或 `PUT` 到 `AppRestore` 接口
- **AND** 请求体同时包含 `storageClassMapping` 与 `scMapping`
- **AND** 两个字段值不一致
- **THEN** 服务端必须 (MUST) 返回 400
- **AND** 错误消息必须 (MUST) 明确指出字段冲突

### Requirement: 新增字段不得破坏既有恢复与实例链路

系统必须 (MUST) 保持存量字段行为兼容，确保升级过程可平滑进行。

#### Scenario: 历史调用仅使用 waitUntilReady 与 scMapping
- **WHEN** 客户端继续发送仅包含 `waitUntilReady` 的 action 请求，以及仅包含 `scMapping` 的 AppRestore 请求
- **THEN** 服务端必须 (MUST) 接受请求
- **AND** 处理结果必须 (MUST) 与当前版本保持一致
