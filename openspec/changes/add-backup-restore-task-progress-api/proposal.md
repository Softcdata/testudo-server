# Change: 新增备份恢复任务进度趋势 API

## Why
首页“备份/恢复任务进度”需要按日期展示成功与失败数量。现有备份恢复统计接口已经能提供总量概览、来源过滤与自动备份成功率，但缺少按执行日期分桶的趋势数据。

本变更在现有 `statistics` 能力上增量扩展一个只读趋势接口，复用已有统计口径，不改变现有 `/backups`、`/restores`、`/backups/success-rate` 与 `/autobackups/execution-summary` 的响应语义。

## What Changes

### 1. 新增任务进度趋势接口
- 新增只读接口：
  - `GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/tasks/progress`
- 查询参数：
  - `type=backup|restore`
  - `scope=all|disaster|app`
  - `range=7d|30d|90d`
  - `namespace`
  - `cluster`
  - `timezone`
- 响应返回：
  - 查询窗口
  - 汇总计数
  - 连续日期分桶 `buckets[]`
  - 前端图例 `series[]`
  - 来源拆分 `sources[]`

### 2. 复用现有统计与来源能力
- 复用 `AppBackupLister`、`AppRestoreLister`、`PolicyLister`。
- 复用 `parseAppResourceOriginFilter`、`inferAppBackupOrigin`、`inferAppRestoreOrigin` 的来源判定。
- 复用 `backupRecordExecutionTime` 与 `backupRecordExecutionResult` 的备份历史判定。
- 保留 `BackupRestoreStatistics` 聚合接口作为总量概览入口，不把它改造成趋势数据源。

### 3. 明确任务范围
- `scope=app` 只统计用户创建的 `AppBackup` 与 `AppRestore`。
- `scope=disaster` 只统计容灾实例链路生成的 `AppBackup` 与 `AppRestore`。
- `scope=all` 同时统计 `app` 与 `disaster` 两个来源。
- `DisasterOperation` 继续由 `/operations` 与 `/operations/by-time` 统计，不纳入本接口，避免与底层备份恢复任务重复计数。

### 4. 明确时间分桶
- `range=7d` 返回 7 个连续自然日分桶。
- `range=30d` 返回 30 个连续自然日分桶。
- `range=90d` 返回 90 个连续自然日分桶。
- 分桶使用 `timezone` 参数，默认 `Asia/Shanghai`。
- 无数据日期必须返回零值分桶。

### 5. 明确恢复任务当前限制
- 当前 `AppRestore` 没有多次执行历史数组。
- 本变更按一个 `AppRestore` 统计一次恢复任务。
- 后续若需要统计同一恢复任务的多次重试执行，需由 `disaster-operator` 为 `AppRestore` 增加恢复历史事实，再启动独立变更。

## Non-Goals
- 不修改 `disaster-operator` CRD。
- 不改变现有统计接口响应结构。
- 不把 `DisasterOperation` 混入备份恢复任务趋势。
- 不返回 ECharts 配置。
- 不提供无限制自定义时间跨度。

## Impact
- Affected specs:
  - `statistics`
- Affected code:
  - `internal/apis/statistics/v1/router.go`
  - `internal/apis/statistics/v1/handler.go`
  - `internal/apis/statistics/v1/types.go`
  - `internal/apis/statistics/v1/*_test.go`
- Cross-repo impact:
  - `cluster-disaster-web` 首页将从 mock `/super/dashboard/tasks` 切换到新趋势接口。
  - `disaster-operator` 当前无需变更。

## Acceptance
- 前端查询 `type=backup&scope=all&range=7d` 时，能拿到 7 个连续日期分桶，每个分桶包含 `completed` 与 `failed`。
- 前端切换 `type=restore` 时，接口返回恢复任务的同结构分桶。
- 前端切换 `scope=app` 与 `scope=disaster` 时，接口能按来源标签与历史兜底规则过滤。
- 查询参数非法时，服务端返回标准 Envelope，业务码为 `1000`。
- 新接口不改变现有 `/backups`、`/restores`、`/backups/success-rate`、`/autobackups/execution-summary` 行为。
