## ADDED Requirements

### Requirement: Server MUST keep modifier Text input surfaces independent
Server MUST 保持 `modifierRulesText` 与 `bulkModifierActionsText` 两个用户输入面相互独立。

#### Scenario: bulk-only instance does not echo bulk snapshot as modifierRulesText
- **Given** 实例只配置了 `restorePolicy.bulkModifierActionsText`
- **And** Server 已生成 `modifierRuleSnapshot`
- **When** 客户端查询实例详情
- **Then** 响应中的 `restorePolicy.bulkModifierActionsText` 必须包含原始批量动作
- **And** 响应中的 `restorePolicy.modifierRulesText` 必须为空
- **And** 响应中的 `restorePolicy.modifierRules` 必须为空

#### Scenario: manual-only instance does not create bulk text
- **Given** 实例只配置了 `restorePolicy.modifierRulesText`
- **When** 客户端查询实例详情
- **Then** 响应中的 `restorePolicy.modifierRulesText` 必须包含原始手写规则
- **And** 响应中的 `restorePolicy.bulkModifierActionsText` 必须为空
- **And** 响应中的 `restorePolicy.bulkModifierActions` 必须为空

#### Scenario: bulk and manual text fields echo separately
- **Given** 实例同时配置了 `restorePolicy.modifierRulesText` 和 `restorePolicy.bulkModifierActionsText`
- **When** 客户端查询实例详情
- **Then** `modifierRulesText` 必须只包含手写规则
- **And** `bulkModifierActionsText` 必须只包含批量动作
- **And** `modifierRulesText` 不得包含 `bulk-` 生成规则 ID

### Requirement: Server MUST treat modifierRuleSnapshot as execution snapshot only
Server MUST 把 `modifierRuleSnapshot` 视为执行快照，不得把快照内容反灌到用户输入字段。

#### Scenario: snapshot generated from bulk does not persist into modifierRules
- **Given** 实例存在已启用 `bulkModifierActions`
- **When** Server 为该实例生成 `modifierRuleSnapshot`
- **Then** bulk 展开规则必须只写入 `modifierRuleSnapshot`
- **And** bulk 展开规则不得写入 `modifierRules`

#### Scenario: list response does not encode modifierRulesText from snapshot
- **Given** 实例存在 `modifierRuleSnapshot`
- **And** 实例不存在手写 `modifierRules`
- **When** 客户端查询实例列表
- **Then** 列表项中的 `modifierRulesText` 必须为空

#### Scenario: ordinary update does not pollute modifierRulesText
- **Given** 实例只存在 bulk 输入和已生成 snapshot
- **When** 客户端只更新实例描述字段
- **Then** Server 必须保留原有 `bulkModifierActions`
- **And** Server 不得把 snapshot 写入 `modifierRules`
- **And** 再次查询时 `modifierRulesText` 必须为空

### Requirement: Server MUST allow manual rules and bulk actions to be enabled together
Server MUST 允许手写规则和批量动作同时启用，并生成包含两类规则的执行快照。

#### Scenario: create instance with both text fields
- **Given** 创建实例请求同时包含 `modifierRulesText` 和 `bulkModifierActionsText`
- **When** Server 成功创建实例
- **Then** CR 中 `spec.restorePolicy.modifierRules` 必须等于手写规则解析结果
- **And** CR 中 `spec.restorePolicy.bulkModifierActions` 必须等于批量动作解析结果
- **And** CR 中 `spec.restorePolicy.modifierRuleSnapshot` 必须包含 bulk 展开规则和手写规则
- **And** CR 中 `spec.restorePolicy.modifierRuleSnapshotHash` 必须非空

#### Scenario: update instance with both text fields
- **Given** 实例已经存在
- **When** 客户端通过更新接口同时提交 `modifierRulesText` 和 `bulkModifierActionsText`
- **Then** Server 必须分别更新 `modifierRules` 和 `bulkModifierActions`
- **And** Server 必须重算 `modifierRuleSnapshot`
- **And** 查询回显必须保持两个 Text 字段独立

#### Scenario: runtime effect includes both manual and bulk modifications
- **Given** 实例同时存在已启用 bulk action 和手写 modifier rule
- **When** 资源同步恢复构建 AppRestore
- **Then** AppRestore 使用的资源修改规则必须同时包含 bulk 展开规则和手写规则
- **And** 目标集群资源必须体现两类修改结果

### Requirement: Server MUST isolate clear operations between the two input surfaces
Server MUST 隔离两个输入面的清空操作。

#### Scenario: clear bulk keeps manual rules
- **Given** 实例同时存在手写规则和批量动作
- **When** 客户端清空 `bulkModifierActionsText`
- **Then** Server 必须清空 `bulkModifierActions`
- **And** Server 必须清理 `modifierRuleSnapshot` 和 `modifierRuleSnapshotHash`
- **And** Server 必须保留 `modifierRules`
- **And** 查询回显中 `modifierRulesText` 必须仍然包含手写规则

#### Scenario: clear manual keeps bulk actions
- **Given** 实例同时存在手写规则和批量动作
- **When** 客户端清空 `modifierRulesText`
- **Then** Server 必须清空 `modifierRules`
- **And** Server 必须保留 `bulkModifierActions`
- **And** Server 必须重算只包含 bulk 展开规则的 `modifierRuleSnapshot`
- **And** 查询回显中 `bulkModifierActionsText` 必须仍然包含批量动作
