## 1. Proposal
- [x] 1.1 锁定接口路径、查询参数、响应 DTO 与错误语义
- [x] 1.2 明确复用现有统计能力的范围
- [x] 1.3 明确 `BackupRestoreStatistics` 不作为趋势主数据源的原因

## 2. API Contract
- [x] 2.1 在 `internal/apis/statistics/v1/router.go` 注册 `GET /tasks/progress`
- [x] 2.2 在 `types.go` 增加任务进度趋势 DTO
- [x] 2.3 在 handler 中实现参数解析与严格枚举校验
- [x] 2.4 返回标准 `Envelope`，非法参数返回业务码 `1000`

## 3. Aggregation
- [x] 3.1 实现 `range=7d|30d|90d` 连续日期分桶
- [x] 3.2 实现 `timezone` 解析与默认值 `Asia/Shanghai`
- [x] 3.3 实现 `type=backup` 的 `AppBackup.Status.History` 聚合
- [x] 3.4 实现 `type=restore` 的 `AppRestore.Status` 聚合
- [x] 3.5 复用现有来源识别，支持 `scope=all|disaster|app`
- [x] 3.6 支持 `namespace` 与 `cluster` 过滤

## 4. Tests
- [x] 4.1 覆盖非法参数
- [x] 4.2 覆盖空数据返回完整零值分桶
- [x] 4.3 覆盖备份成功、失败、取消、执行中状态映射
- [x] 4.4 覆盖恢复成功、失败、取消、执行中状态映射
- [x] 4.5 覆盖 `scope=all|disaster|app`
- [x] 4.6 覆盖现有统计接口回归

## 5. Documentation
- [x] 5.1 补充 Apipost 接口文档
- [x] 5.2 示例响应使用 `code=0` 与 `message=OK`
- [x] 5.3 与前端确认 `buckets[]`、`series[]`、`sources[]` 字段消费方式

## 6. Verification
- [x] 6.1 `openspec validate add-backup-restore-task-progress-api --strict`
- [x] 6.2 `go test ./internal/apis/statistics/v1`
