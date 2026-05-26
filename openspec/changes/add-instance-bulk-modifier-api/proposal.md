# Change: 为实例恢复策略提供批量资源修改 API

## Why
模板目录方案并没有解决当前核心用户故事：

- 我想把整个容灾实例里出现的某个 IP 批量替换成另一个 IP。
- 我想把整个容灾实例里某个 key 批量删除。
- 我不想逐个资源、逐条路径手写 `modifierRules`。

因此 server 侧应当提供的不是“模板复用”，而是“实例级批量修改动作”：

- 用户声明高层动作。
- server 在实例写入时扫描该实例保护范围内的资源。
- server 把命中的结果展开成具体 `modifierRuleSnapshot`。

这样用户看到的是“整实例批量替换 / 批量删除”，执行层仍复用现有规则引擎。

## What Changes

### 1. 为实例 API 增加 `bulkModifierActions`
`restorePolicy` 新增 `bulkModifierActions` 作为正式产品层输入。

该字段用于表达实例级批量修改意图，而不是底层 JSON Pointer 规则。

Phase 1 推荐字段形状：

- 公共字段
  - `id`
  - `action`
  - `enabled`
  - `applyTo`
  - `directionPolicy`
- `replaceExactValue`
  - `sourceValue`
  - `targetValue`
- `removeKey`
  - `key`

其中：

- `enabled` 省略时按 `true` 处理
- 只有已启用动作参与 live 扫描、snapshot 生成、失败关闭与摘要统计
- `enabled=false` 的动作保留在用户输入中，但按“无效 bulk 动作”跳过展开

### 2. Phase 1 只支持两类实例级批量动作
Phase 1 只支持：

- `replaceExactValue`
  - 在实例保护范围内扫描字符串叶子节点
  - 仅命中与 `sourceValue` 精确相等的值
  - 展开为当前正式 `reversible pair-only` 规则
- `removeKey`
  - 在实例保护范围内扫描对象 / map 成员键
  - 仅命中与 `key` 精确相等的键
  - 展开为当前正式 `veleroNative` remove patch

Phase 1 额外约束：

- `applyTo` 只允许 `resourceSync`、`drill`
- 省略 `applyTo` 时默认按 `["resourceSync"]`
- 不支持 `dataSync`
- `replaceExactValue.directionPolicy` 默认 `Auto`
- `removeKey.directionPolicy` 默认 `ForwardOnly`
- `removeKey` 生成的 snapshot 规则必须显式写入 `directionPolicy=ForwardOnly`

### 3. server 在 bulk 相关输入变更时为已启用动作生成 `modifierRuleSnapshot`
server 不是在任意实例更新时都重扫 live 资源，而是在“影响 bulk 展开结果的输入”变更时，为已启用动作生成或重算 `modifierRuleSnapshot`。

bulk 相关输入至少包括：

- `restorePolicy.bulkModifierActions`
- `restorePolicy.modifierRules`
- `restorePolicy.resourceSelection`
- `spec.namespaces`
- `spec.labelSelector`
- `spec.config`

在这些输入变更时，server 会：

1. 读取 `bulkModifierActions`
2. 先过滤出 `enabled != false` 的有效动作
3. 若有效动作为空，则清理 `modifierRuleSnapshot` / `modifierRuleSnapshotHash` 并保留手写 `modifierRules`
4. 若有效动作非空，根据实例有效保护范围扫描 live 资源
5. 把命中结果展开为具体规则
6. 与用户手写 `modifierRules` 合并
7. 写入：
   - `modifierRuleSnapshot`
   - `modifierRuleSnapshotHash`

其中：

- `modifierRules` 继续保留为用户手写输入
- `modifierRuleSnapshot` 表示最终执行快照

具体要求：

- 扫描集群固定使用 `config.spec.sourceCluster`
- 保护范围固定继承实例 restore 语义，而不是“扫整集群”
- `modifierRuleSnapshotHash` 使用最终 snapshot 的 canonical JSON 做 `sha256`
- 与 bulk 无关的实例更新不得触发 snapshot 重算

### 3.1 清空或全部禁用 bulkModifierActions 时必须清理 snapshot
当用户移除、清空 `bulkModifierActions`，或把它们全部改成 `enabled=false` 时，server 必须同步清理：

- `modifierRuleSnapshot`
- `modifierRuleSnapshotHash`

不允许保留旧 bulk snapshot 继续让 operator 执行。

### 4. 定义批量动作与手写规则的合并顺序
Phase 1 必须明确：

1. 先按声明顺序展开已启用 `bulkModifierActions`
2. 再把用户手写 `modifierRules` 追加到 snapshot 末尾

但因为现有编译器是按 `priority` 不是按数组顺序做冲突决胜，所以 server 还必须：

- 给 bulk 生成规则写入更低默认优先级，推荐 `priority=-100`
- 手写规则保持当前语义，默认 `priority=0`

这样用户仍可在少数路径上用手写规则稳定覆盖批量展开结果。

### 5. 批量动作必须失败关闭，不允许静默空跑
以下情况必须拒绝实例写入：

- 已启用动作没有命中任何资源 / 路径
- 已启用动作试图修改受保护路径
- 已启用 `removeKey` 命中的是不支持的结构
- 已启用动作展开结果生成了不符合正式 contract 的规则
- 同一路径被多个已启用 bulk action 展开成不同值
- 任一动作 `applyTo` 包含 `dataSync`

`enabled=false` 的动作不得触发 zero-match / protected-path / unsupported-structure 这类展开期失败。

## Non-Goals

- 不再做模板 CRUD、模板目录、模板绑定。
- 不在 Phase 1 支持正则替换、子串替换、脚本替换。
- 不在 Phase 1 支持按列表元素名称批量删除。
- 不让 server 生成旧 `transform/map/template` 结构。
- 不让 operator 在运行时重新扫描资源。

## Impact
- Affected specs:
  - `bulk-modifier-api`
- Affected code:
  - `internal/apis/disaster_instance/v1/*`
  - 可能新增批量扫描 / 展开辅助模块
- Cross-repo impact:
  - `disaster-operator`：消费 `modifierRuleSnapshot`
  - `cluster-disaster-web`：实例级批量替换 / 删除 UI

## Relationship to Existing Changes
- 建立在 `refactor-server-modifier-pair-only-alignment` 之上。
- 依赖 operator 当前正式 `restore-modifier` contract。
- 替代已删除的模板 API proposal，不再沿用模板目录方案。
- 复用现有 `modifierRules` 的 schema、submission validation 和 live validation，不再另写第二套执行 DSL。
