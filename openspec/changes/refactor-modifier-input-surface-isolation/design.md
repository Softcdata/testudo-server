# 设计：资源修改 Text 输入面隔离

## 1. 设计目标
1. 保证 `modifierRulesText`、`bulkModifierActionsText` 是两个独立用户输入面。
2. 保证 `modifierRuleSnapshot` 是执行快照，不进入用户输入面。
3. 保证手写规则和批量动作可以同时生效。
4. 保证查询回显、创建、更新、演练覆盖使用同一套语义。

## 2. 字段职责

| 字段 | 职责 | 可编辑 | 持久化来源 | 回显来源 |
|---|---|---|---|---|
| `modifierRulesText` | 手写规则文本入口 | 是 | 请求文本解析 | `modifierRules` 编码 |
| `modifierRules` | 手写规则结构化输入 | 是 | 文本解析结果以及结构化入参 | 用户输入 |
| `bulkModifierActionsText` | 批量动作文本入口 | 是 | 请求文本解析 | `bulkModifierActions` 编码 |
| `bulkModifierActions` | 批量动作结构化输入 | 是 | 文本解析结果以及结构化入参 | 用户输入 |
| `modifierRuleSnapshot` | 执行快照 | 否 | server 生成 | 仅审计展示 |
| `modifierRuleSnapshotHash` | 快照哈希 | 否 | server 生成 | 仅审计展示 |

核心规则：输入字段可以生成快照，快照不得生成输入字段。

## 3. 请求解析模型

### 3.1 创建和修改实例
`RestorePolicyRequest.ToCRD()` 必须按以下顺序处理：

1. 解析 `bulkModifierActionsText`。
2. 如果同时传入 `bulkModifierActions`，必须校验 Text 解析结果和结构化字段语义一致。
3. 将结果写入 `policy.BulkModifierActions`。
4. 解析 `modifierRulesText`。
5. 如果同时传入 `modifierRules`，必须校验 Text 解析结果和结构化字段语义一致。
6. 将结果写入 `policy.ModifierRules`。
7. 不接受请求传入 `modifierRuleSnapshot` 和 `modifierRuleSnapshotHash` 作为用户输入。

说明：Text 输入和结构化输入的冲突校验继续保留，避免同一输入面出现两个不同真值。

### 3.2 空字符串语义
- `modifierRulesText=""` 表示没有手写规则。
- `bulkModifierActionsText=""` 表示没有批量动作。
- 空字符串不得从旧 snapshot 推导出历史内容。

## 4. 持久化准备模型

`prepareRestorePolicyForPersist()` 必须保持如下不变量：

1. `policy.ModifierRules` 在函数返回后仍然只包含用户手写规则。
2. `policy.BulkModifierActions` 在函数返回后仍然只包含用户批量动作。
3. `policy.ModifierRuleSnapshot` 可以包含 bulk 生成规则和手写规则。
4. `policy.ModifierRuleSnapshotHash` 只覆盖最终 `modifierRuleSnapshot`。
5. 当没有已启用 bulk action 时，snapshot/hash 清空，手写规则保留。

伪代码：

```go
manualRules := clone(policy.ModifierRules)
actions := normalize(policy.BulkModifierActions)
policy.BulkModifierActions = actions

if len(enabled(actions)) == 0 {
    policy.ModifierRules = manualRules
    policy.ModifierRuleSnapshot = nil
    policy.ModifierRuleSnapshotHash = ""
    validateEffective(policy)
    return nil
}

snapshot := expandBulk(actions)
snapshot = append(snapshot, manualRules...)
policy.ModifierRules = manualRules
policy.ModifierRuleSnapshot = snapshot
policy.ModifierRuleSnapshotHash = hash(snapshot)
validateEffective(policy)
```

关键点：任何 bulk 展开结果都只能进入 `modifierRuleSnapshot`，不能覆盖 `policy.ModifierRules`。

## 5. 查询回显模型

`convertRestorePolicyDTO()` 必须保持如下不变量：

```go
dto.BulkModifierActions = clone(policy.BulkModifierActions)
dto.ModifierRules = clone(policy.ModifierRules)
dto.ModifierRuleSnapshot = clone(policy.ModifierRuleSnapshot)
dto.BulkModifierActionsText = encode(dto.BulkModifierActions)
dto.ModifierRulesText = encode(dto.ModifierRules)
```

禁止逻辑：

```go
// 禁止：从 snapshot 生成手写 Text
dto.ModifierRulesText = encode(policy.ModifierRuleSnapshot)

// 禁止：把 bulk 展开规则回写成手写规则
dto.ModifierRules = filterBulkGenerated(policy.ModifierRuleSnapshot)
```

## 6. 同时启用语义

当请求同时包含：

```json
{
  "restorePolicy": {
    "modifierRulesText": "[...]",
    "bulkModifierActionsText": "[...]"
  }
}
```

持久化后必须满足：

- `modifierRules` 等于 `modifierRulesText` 解析结果。
- `bulkModifierActions` 等于 `bulkModifierActionsText` 解析结果。
- `modifierRuleSnapshot` 包含 bulk 展开规则和手写规则。
- `modifierRulesText` 回显只包含手写规则。
- `bulkModifierActionsText` 回显只包含批量动作。

执行后必须满足：

- ResourceSync 使用 snapshot 执行。
- Drill 使用 snapshot 执行。
- 目标资源同时体现 bulk 批量修改结果和手写规则修改结果。

## 7. 更新语义矩阵

| 更新场景 | 预期 |
|---|---|
| 只更新普通字段 | 保留两个输入面和 snapshot/hash，不重算 |
| 只更新 `bulkModifierActionsText` | 重算 snapshot/hash，`modifierRules` 不变 |
| 只更新 `modifierRulesText` | 重算 snapshot/hash，`bulkModifierActions` 不变 |
| 清空 `bulkModifierActionsText` | 清理 snapshot/hash，保留 `modifierRules` |
| 清空 `modifierRulesText` | 清空 `modifierRules`，bulk 输入保留，snapshot 只含 bulk 生成规则 |
| 同时清空两个 Text | `modifierRules`、`bulkModifierActions`、snapshot/hash 均为空 |
| 更新 `resourceSelection/namespaces/labelSelector/config` | 重算 snapshot/hash，输入面不被 snapshot 污染 |

## 8. 演练覆盖策略

演练创建支持独立 `restorePolicy` 覆盖时，必须复用同一套 DTO 转换和持久化准备逻辑。

演练策略覆盖必须满足：

- 未传演练级 `restorePolicy` 时，继承实例策略。
- 传入演练级 `modifierRulesText` 时，只覆盖演练本次手写规则。
- 传入演练级 `bulkModifierActionsText` 时，只覆盖演练本次批量动作。
- 演练级 snapshot 仍然只作为本次演练执行快照，不反灌实例输入面。

## 9. 测试策略

### 9.1 单元测试
- bulk-only 创建：`modifierRulesText` 为空，`bulkModifierActionsText` 非空，snapshot 非空。
- manual-only 创建：`modifierRulesText` 非空，`bulkModifierActionsText` 为空，snapshot 为空。
- bulk + manual 创建：两个 Text 独立回显，snapshot 包含两类规则。
- bulk-only 更新普通字段：Text 不被 snapshot 污染。
- 清空 bulk：清理 snapshot/hash，保留 manual。
- 清空 manual：保留 bulk，snapshot 只含 bulk 展开结果。

### 9.2 API 集成测试
- 创建实例后 GET 详情校验两个 Text 字段。
- 修改实例后 GET 详情校验两个 Text 字段。
- 列表接口校验不从 snapshot 生成 `modifierRulesText`。
- RunAPI 示例请求可直接复现。

### 9.3 E2E 测试
- 使用专用 namespace，创建 Deployment、Service、Ingress、ConfigMap。
- bulk 替换固定 IP。
- manual 修改 NodePort、Ingress host、annotation。
- 触发 ResourceSync，检查目标集群资源同时体现两类修改。
- 创建 Drill，检查演练 namespace 资源同时体现两类修改。

## 10. 验收标准
1. 任何 GET/List 响应中，`modifierRulesText` 不得出现 `bulk-` 生成规则 ID。
2. bulk-only 实例的 `modifierRulesText` 必须为空。
3. bulk + manual 实例的 `modifierRulesText` 只能包含 manual rule ID。
4. bulk + manual 实例的 `bulkModifierActionsText` 只能包含 bulk action ID。
5. CR 中 `spec.restorePolicy.modifierRules` 不得包含 `bulk-` 生成规则。
6. CR 中 `spec.restorePolicy.modifierRuleSnapshot` 必须包含最终执行规则集合。
7. ResourceSync 和 Drill 目标资源必须同时体现 bulk 和 manual 的修改效果。
