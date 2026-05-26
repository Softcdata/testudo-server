## ADDED Requirements

### Requirement: Server 必须支持实例级批量修改动作
系统必须 (MUST) 支持在实例创建或更新时接收 `restorePolicy.bulkModifierActions`，用于表达整实例批量替换值或批量删除 key。

#### Scenario: 创建实例时声明批量动作
- **Given** 客户端在实例请求中声明了 `restorePolicy.bulkModifierActions`
- **When** Server 处理实例创建请求
- **Then** Server 必须保存这些批量动作
- **And** 不得要求用户逐个资源手写底层 `modifierRules`

#### Scenario: enabled 缺省时按启用处理
- **Given** 一个 `bulkModifierActions` 条目未显式填写 `enabled`
- **When** Server 处理实例创建或更新请求
- **Then** Server 必须将它按 `enabled=true` 解释

#### Scenario: applyTo 默认 resourceSync
- **Given** 一个 `bulkModifierActions` 条目未显式填写 `applyTo`
- **When** Server 处理实例创建或更新请求
- **Then** Server 必须将它按 `["resourceSync"]` 解释

#### Scenario: dataSync 作为 applyTo 被拒绝
- **Given** 一个 `bulkModifierActions` 条目把 `dataSync` 写入 `applyTo`
- **When** Server 处理实例创建或更新请求
- **Then** Server 必须拒绝该请求

#### Scenario: removeKey 默认方向必须落成 ForwardOnly
- **Given** 一个 `removeKey` bulk action 未显式填写 `directionPolicy`
- **When** Server 展开它为 `modifierRuleSnapshot`
- **Then** Server 必须将展开后的规则显式写成 `directionPolicy=ForwardOnly`

### Requirement: Server 只将已启用批量动作视为有效 bulk 动作
系统必须 (MUST) 只将 `enabled != false` 的 `bulkModifierActions` 视为有效 bulk 动作。

#### Scenario: enabled=false 的动作跳过展开
- **Given** 一个 `bulkModifierActions` 条目显式填写 `enabled=false`
- **When** Server 处理实例创建或更新请求
- **Then** Server 不得为该动作生成 `modifierRuleSnapshot`
- **And** 不得因该动作触发 0 命中、受保护路径或不支持结构这类展开期失败

#### Scenario: 全部动作禁用时清理 snapshot
- **Given** 一个实例之前已经存在 `modifierRuleSnapshot` 与 `modifierRuleSnapshotHash`
- **And** 本次请求中的 `bulkModifierActions` 全部为 `enabled=false`
- **When** Server 处理实例创建或更新请求
- **Then** Server 必须清理 `modifierRuleSnapshot`
- **And** 必须清理 `modifierRuleSnapshotHash`
- **And** 必须保留用户手写 `modifierRules`

### Requirement: Server 必须将已启用批量动作展开为 modifierRuleSnapshot
系统必须 (MUST) 在 bulk 相关输入变更时把已启用批量动作展开为可执行规则快照。

#### Scenario: replaceExactValue 展开为 pair-only 快照
- **Given** 一个已启用批量动作类型为 `replaceExactValue`
- **When** Server 在实例保护范围内扫描 live 资源
- **Then** 命中的字符串叶子节点必须展开为 `modifierRuleSnapshot`
- **And** 其中的 `reversible` 规则必须 (MUST) 使用 `pair.path`、`pair.sourceValue`、`pair.targetValue`
- **And** 这些规则必须使用低于手写规则默认值的优先级

#### Scenario: removeKey 展开为 veleroNative remove 快照
- **Given** 一个已启用批量动作类型为 `removeKey`
- **When** Server 在实例保护范围内扫描 live 资源
- **Then** 命中的对象 / map 键必须展开为 `modifierRuleSnapshot`
- **And** 展开结果必须 (MUST) 使用当前正式 `veleroNative` remove patch 结构

#### Scenario: 批量展开后必须写入快照哈希
- **Given** 一个实例请求包含至少一个已启用 `bulkModifierAction`
- **When** Server 完成快照展开
- **Then** Server 必须写入 `modifierRuleSnapshotHash`

#### Scenario: 扫描范围继承实例保护范围
- **Given** 一个实例请求包含至少一个已启用 `bulkModifierAction`
- **And** 该实例声明了 `namespaces`、`labelSelector` 或 `restorePolicy.resourceSelection`
- **When** Server 进行 live 资源扫描
- **Then** Server 必须只扫描该实例有效保护范围内的资源
- **And** 不得把扫描范围扩大到整集群无关资源

#### Scenario: 无关字段更新不得重算 snapshot
- **Given** 一个实例已经存在 `modifierRuleSnapshot` 与 `modifierRuleSnapshotHash`
- **And** 本次更新未改变 `bulkModifierActions`、`modifierRules`、`resourceSelection`、`namespaces`、`labelSelector` 或 `config`
- **When** Server 处理实例更新请求
- **Then** Server 不得重算 `modifierRuleSnapshot`
- **And** 不得重写 `modifierRuleSnapshotHash`

#### Scenario: 清空 bulkModifierActions 时清理 snapshot
- **Given** 一个实例之前已经存在 `bulkModifierActions`、`modifierRuleSnapshot` 与 `modifierRuleSnapshotHash`
- **And** 客户端本次将 `bulkModifierActions` 清空或显式移除
- **When** Server 处理实例更新请求
- **Then** Server 必须清理 `modifierRuleSnapshot`
- **And** 必须清理 `modifierRuleSnapshotHash`

### Requirement: 已启用批量动作与手写规则必须有确定性合并顺序
系统必须 (MUST) 为已启用 `bulkModifierActions` 与手写 `modifierRules` 定义稳定的合并顺序。

#### Scenario: 手写规则追加到快照末尾
- **Given** 一个实例请求同时包含至少一个已启用 `bulkModifierAction` 和手写 `modifierRules`
- **When** Server 生成 `modifierRuleSnapshot`
- **Then** Server 必须先展开已启用的 `bulkModifierActions`
- **And** 必须再把手写 `modifierRules` 追加到 `modifierRuleSnapshot` 末尾
- **And** bulk 生成规则必须使用更低默认优先级，以便手写规则稳定覆盖

#### Scenario: 同路径不同值的 bulk 展开必须失败
- **Given** 多个已启用 `bulkModifierActions` 展开到了同一个冲突键
- **And** 它们的目标值不同
- **When** Server 生成 `modifierRuleSnapshot`
- **Then** Server 必须拒绝该请求

### Requirement: 已启用批量动作必须失败关闭，不得静默空跑
系统必须 (MUST) 在已启用批量动作无命中或展开非法时拒绝实例写入。

#### Scenario: 批量动作命中 0 个修改点
- **Given** 一个实例请求包含至少一个已启用 `bulkModifierAction`
- **And** 其中一个已启用动作最终没有命中任何具体资源路径
- **When** Server 处理实例创建或更新请求
- **Then** Server 必须拒绝该请求

#### Scenario: 批量动作命中受保护路径
- **Given** 一个已启用批量动作在展开过程中命中了受保护路径
- **When** Server 处理实例创建或更新请求
- **Then** Server 必须拒绝该请求

#### Scenario: removeKey 命中不支持的结构
- **Given** 一个已启用 `removeKey` 动作命中了列表元素或其他 Phase 1 不支持的结构
- **When** Server 处理实例创建或更新请求
- **Then** Server 必须拒绝该请求

### Requirement: Server 不得用批量动作重新生成旧 reversible 结构
系统必须 (MUST) 保证批量动作展开结果只使用当前正式资源修改器 contract。

#### Scenario: 批量快照拒绝旧 transform
- **Given** 一个实例请求包含至少一个已启用 `bulkModifierAction`
- **When** Server 生成 `modifierRuleSnapshot`
- **Then** 快照不得 (MUST NOT) 包含旧 `transform/map/template` 结构

#### Scenario: 最终 snapshot 必须通过现有规则校验
- **Given** 一个实例请求包含至少一个已启用 `bulkModifierAction`
- **When** Server 完成 `modifierRuleSnapshot` 展开
- **Then** Server 必须对最终 snapshot 复用现有的 submission validation
- **And** 必须对最终 snapshot 复用现有的 live validation
