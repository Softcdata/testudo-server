# Change: 隔离资源修改 Text 输入面与批量快照执行面

## Why
当前实例恢复策略同时支持两类用户输入：

- `restorePolicy.modifierRulesText`：用户手写的精确资源修改规则。
- `restorePolicy.bulkModifierActionsText`：用户声明的实例级批量替换和批量删除动作。

系统还会在写入实例时生成 `modifierRuleSnapshot`，该字段是执行层快照，里面会包含批量动作展开后的规则以及手写规则。这个模型本身成立，但前端只把两个 Text 字段作为用户可编辑面时，必须明确边界：

- `modifierRulesText` 只能回显用户手写 `modifierRules`。
- `bulkModifierActionsText` 只能回显用户声明的 `bulkModifierActions`。
- `modifierRuleSnapshot` 只能作为执行快照和审计证据，不能反灌到任一 Text 字段。

如果 bulk 展开的规则被写回 `modifierRules`，前端资源修改 Text 会显示大量 `bulk-*` 生成规则，用户会误以为这些规则是自己手写配置，后续编辑保存还可能把执行快照永久固化为手写输入，造成输入层污染。

## What Changes

### 1. 明确三层字段职责
`restorePolicy` 必须按三层职责处理：

- 手写输入层：`modifierRules`、`modifierRulesText`
- 批量输入层：`bulkModifierActions`、`bulkModifierActionsText`
- 执行快照层：`modifierRuleSnapshot`、`modifierRuleSnapshotHash`

字段职责必须保持单向流动：

1. 请求进入时，Text 字段只解析到对应结构化输入字段。
2. server 准备持久化时，使用批量输入和手写输入生成执行快照。
3. 查询回显时，Text 字段只从对应结构化输入字段编码，不能从快照编码。

### 2. `modifierRuleSnapshot` 不得反灌到 `modifierRules`
server 在生成快照时可以把手写规则追加进 `modifierRuleSnapshot`，但不得把 bulk 生成规则写入 `modifierRules`。

正确持久化形态：

- `modifierRules`：只保存用户手写规则。
- `bulkModifierActions`：只保存用户批量动作。
- `modifierRuleSnapshot`：保存最终执行规则集合，包含 bulk 展开规则和手写规则。

### 3. 两个 Text 字段必须相互独立
查询实例详情和列表时：

- `modifierRulesText` 必须只由 `modifierRules` 编码得到。
- `bulkModifierActionsText` 必须只由 `bulkModifierActions` 编码得到。
- 当实例只有 bulk 输入且无手写规则时，`modifierRulesText` 必须为空。
- 当实例同时存在 bulk 输入和手写规则时，两个 Text 字段必须分别回显自己的输入，不得交叉混入。

### 4. 两个输入面必须可以同时启用
同时配置 `modifierRulesText` 和 `bulkModifierActionsText` 是合法主路径。

server 必须：

1. 分别解析两个 Text 字段。
2. 分别持久化到 `modifierRules` 和 `bulkModifierActions`。
3. 生成同时包含 bulk 展开规则和手写规则的 `modifierRuleSnapshot`。
4. 保持 bulk 默认优先级低于手写规则默认优先级。
5. 在资源同步和演练链路中让两个配置同时生效。

### 5. 更新路径必须保持输入面隔离
更新实例时：

- 仅更新 bulk 输入，不得生成 `modifierRulesText` 污染。
- 仅更新手写规则，不得清空 bulk 输入。
- 更新普通字段，不得把 snapshot 写回 `modifierRules`。
- 清空 bulk 输入时，只清理 snapshot/hash 中的 bulk 快照效果，同时保留手写规则。
- 清空手写规则时，只清理 `modifierRules`，不得清理 bulk 输入。

## Non-Goals

- 不新增第三种资源修改 DSL。
- 不改变 `bulkModifierActions` 当前支持的动作类型。
- 不改变 operator 的资源修改执行引擎。
- 不要求前端展示 `modifierRuleSnapshot` 为可编辑内容。
- 不把 `modifierRuleSnapshot` 作为用户输入字段开放编辑。

## Impact

### Affected specs
- `modifier-input-surface-isolation`

### Affected code
- `internal/apis/disaster_instance/v1/types.go`
- `internal/apis/disaster_instance/v1/handler.go`
- `internal/apis/disaster_instance/v1/bulk_modifier_prepare.go`
- `internal/apis/disaster_instance/v1/bulk_modifier_snapshot.go`
- `internal/apis/disaster_drill/v1/*`，当演练复用实例恢复策略 DTO 时需要保持同样语义

### Cross-repo impact
- `disaster-operator`：无需新增字段；继续按 `modifierRuleSnapshot` 作为执行输入。需要回归确认 snapshot 模式下不会再次追加 `modifierRules`。
- `cluster-disaster-web`：两个 Text 编辑器可以同时展示和提交，但不得把 snapshot 展示为用户可编辑的手写规则。
- RunAPI/Apipost：需要在创建和修改实例文档中说明两个 Text 字段独立输入，并补充同时启用示例。

## Relationship to Existing Changes

本变更建立在以下已完成变更之上：

- `add-instance-bulk-modifier-api`
- `add-instance-bulk-modifier-contract`
- `refactor-server-modifier-pair-only-alignment`
- `add-generic-reversible-resource-modifier-engine`

本变更不是替代 bulk modifier，而是补齐 bulk modifier 与手写 modifier 在 API 回显和持久化层面的边界契约。
