## 1. API 模型
- [x] 1.1 为 `restorePolicy` 增加 `bulkModifierActions`
- [x] 1.2 为 `restorePolicy` 增加 `modifierRuleSnapshot`
- [x] 1.3 为 `restorePolicy` 增加 `modifierRuleSnapshotHash`
- [x] 1.4 为 `replaceExactValue` / `removeKey` 增加字段级校验、`enabled` 默认值归一化与有效动作过滤
- [x] 1.5 限制 `bulkModifierActions.applyTo` 只允许 `resourceSync` / `drill`

## 2. 批量展开
- [x] 2.1 基于 `config.spec.sourceCluster` 构建 live scanner
- [x] 2.2 复用实例命名空间、labelSelector、resourceSelection 计算扫描范围
- [x] 2.3 实现 `replaceExactValue` 扫描与展开
- [x] 2.4 实现 `removeKey` 扫描与展开
- [x] 2.5 对命中结果做稳定排序并生成 deterministic rule ID
- [x] 2.6 生成并写入 `modifierRuleSnapshotHash`
- [x] 2.7 `removeKey` 展开时显式写入 `directionPolicy=ForwardOnly`

## 3. 合并与校验
- [x] 3.1 bulk 生成规则统一写 `priority=-100`
- [x] 3.2 手写 `modifierRules` 按原语义追加到 snapshot
- [x] 3.3 对 zero-match / protected-path / unsupported-structure / dataSync 失败关闭
- [x] 3.4 对最终 snapshot 复用现有 submission validation
- [x] 3.5 对最终 snapshot 复用现有 live validation
- [x] 3.6 保证展开结果只生成 pair-only / veleroNative 正式结构
- [x] 3.7 仅在 bulk 相关输入变更时重算 snapshot
- [x] 3.8 用户清空或全部禁用 `bulkModifierActions` 时同步清理 snapshot/hash
- [x] 3.9 `enabled=false` 的动作跳过展开与展开期失败校验

## 4. 兼容性
- [x] 4.1 保持现有 `modifierRules` / `modifierRulesText` 继续可用
- [x] 4.2 无批量动作时允许 snapshot 回退到现有显式规则行为
- [x] 4.3 当用户清空 `bulkModifierActions` 时清理 snapshot 与 hash

## 5. 验证
- [x] 5.1 handler test 覆盖批量动作请求读写
- [x] 5.2 展开单测覆盖 `replaceExactValue`
- [x] 5.3 展开单测覆盖 `removeKey`
- [x] 5.4 展开单测覆盖 snapshot hash 稳定性
- [x] 5.5 回归测试覆盖批量动作与手写规则合并优先级
- [x] 5.6 回归测试覆盖 0 命中 / protected-path / unsupported-structure / dataSync
- [x] 5.7 回归测试覆盖无关字段更新不会重算 snapshot
- [x] 5.8 回归测试覆盖清空或全部禁用 `bulkModifierActions` 会清理 snapshot/hash
- [x] 5.9 回归测试覆盖 `enabled=false` 的动作不会触发展开期失败
