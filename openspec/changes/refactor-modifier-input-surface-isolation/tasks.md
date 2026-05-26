## 1. Server API 模型与转换
- [ ] 1.1 确认 `RestorePolicyRequest` 不接收 `modifierRuleSnapshot` 和 `modifierRuleSnapshotHash` 用户输入
- [ ] 1.2 确认 `modifierRulesText` 只解析到 `modifierRules`
- [ ] 1.3 确认 `bulkModifierActionsText` 只解析到 `bulkModifierActions`
- [ ] 1.4 保留 Text 与结构化字段同时传入时的等价校验

## 2. Server 持久化准备
- [ ] 2.1 修改并加固 `prepareRestorePolicyForPersist`，保证 bulk 生成规则只写入 `modifierRuleSnapshot`
- [ ] 2.2 保证 `modifierRules` 在 snapshot 生成前后保持手写输入集合
- [ ] 2.3 保证无已启用 bulk action 时清理 snapshot/hash，同时保留手写规则
- [x] 2.4 保证清空 hand-written rules 不影响 bulk 输入
- [x] 2.5 保证清空 bulk 输入不影响 hand-written rules

## 3. Server 查询回显
- [ ] 3.1 加固 `convertRestorePolicyDTO`，只从 `modifierRules` 编码 `modifierRulesText`
- [ ] 3.2 加固 `convertRestorePolicyDTO`，只从 `bulkModifierActions` 编码 `bulkModifierActionsText`
- [ ] 3.3 增加防回归断言：`modifierRulesText` 不得包含 `bulk-` 生成 rule ID
- [ ] 3.4 增加防回归断言：bulk-only 实例回显 `modifierRulesText` 为空

## 4. Drill 复用链路
- [ ] 4.1 检查演练创建的 `restorePolicy` 覆盖逻辑复用同一套转换语义
- [ ] 4.2 增加演练级 bulk + manual 同时启用测试
- [ ] 4.3 确认演练级 snapshot 不反灌实例 `modifierRules` 和 `modifierRulesText`

## 5. Operator 回归确认
- [x] 5.1 确认存在已启用 bulk action 时 operator 只消费 `modifierRuleSnapshot`
- [x] 5.2 确认 operator 不会再次追加 `modifierRules`
- [x] 5.3 确认 snapshot 中包含 manual rule 时，manual rule 能随 ResourceSync 生效
- [x] 5.4 确认 snapshot 中包含 manual rule 时，manual rule 能随 Drill 生效

## 6. 测试与文档
- [ ] 6.1 增加 server 单测覆盖 bulk-only、manual-only、bulk+manual 三类 DTO 回显
- [ ] 6.2 增加 server 单测覆盖普通字段更新不污染 Text
- [x] 6.3 增加 server 单测覆盖清空 bulk 和清空 manual 的隔离语义
- [x] 6.4 更新 RunAPI 创建实例和修改实例文档，补充两个 Text 同时启用示例
- [ ] 6.5 更新 E2E 测试文档，增加 Text 隔离和同时生效用例
