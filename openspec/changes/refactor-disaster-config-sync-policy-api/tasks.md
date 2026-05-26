# Tasks

## 1. Proposal
- [x] 1.1 收敛 server 提案口径，明确 API 继续保持双字段契约
- [x] 1.2 明确前端可复用统一策略候选数据源，但不改变 create/update/detail/list 字段

## 2. Server
- [x] 2.1 收敛 `DisasterConfig` DTO 与写路径：继续使用双字段，并拒绝顶层 `syncPolicy`
- [x] 2.2 收敛 `DisasterInstance` DTO 与写路径：继续使用双字段，并拒绝顶层 `syncPolicy`
- [x] 2.3 保留实例 detail/list 的 `effectiveDataSyncPolicy`、`effectiveResourceSyncPolicy`、`dataSyncPolicySource`、`resourceSyncPolicySource`
- [x] 2.4 去除无有效策略时的伪默认 cron 回显
- [x] 2.5 补拒绝 `syncPolicy`、继承、单字段 override、双字段 override 场景测试

## 3. Alignment
- [x] 3.1 与 operator 对齐字段名和继承链
- [x] 3.2 与 API 文档口径对齐，不把统一 `syncPolicy` 暴露为正式接口字段

## 4. Verification
- [x] 4.1 `openspec validate refactor-disaster-config-sync-policy-api --strict`
