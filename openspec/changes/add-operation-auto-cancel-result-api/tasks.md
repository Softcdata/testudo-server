# Tasks

## 1. Proposal
- [ ] 1.1 评审 detail/list/history/watch 共用的 `autoCancelSummary` 字段集
- [ ] 1.2 对齐 operator/server 的字段名、枚举值和时间戳语义

## 2. Server DTO / Projection
- [ ] 2.1 新增 `AutoCancelSummaryDTO`
- [ ] 2.2 为实例 detail DTO 增加自动补偿结果摘要
- [ ] 2.3 为实例 list DTO 增加同名简化摘要
- [ ] 2.4 为 instance history 记录增加自动补偿摘要
- [ ] 2.5 为 group operation watch DTO 增加自动补偿摘要
- [ ] 2.6 为时间线投影增加自动补偿节点
- [ ] 2.7 补 handler / DTO / projection tests

## 3. Alignment
- [ ] 3.1 与 operator 对齐状态字段名、枚举值和“补偿成功但 failover 失败”的语义口径
- [ ] 3.2 与 web 对齐 detail/list/history/watch/timeline 展示

## 4. Verification
- [ ] 4.1 `openspec validate add-operation-auto-cancel-result-api --strict`
