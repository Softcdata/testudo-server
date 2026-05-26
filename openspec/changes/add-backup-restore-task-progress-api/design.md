# Design: 备份恢复任务进度趋势 API

## Context
现有 server 已经具备三类相关能力：

1. `GET /backups` 与 `GET /restores` 聚合 `BackupRestoreStatistics` 总量。
2. `origin=user|instance|all` 能区分用户应用任务与容灾实例链路任务。
3. `GET /autobackups/execution-summary` 已经按 `AppBackup.Status.History` 统计自动备份窗口内成功与失败数量。

首页柱状图需要的是按日期分桶的成功与失败趋势。`BackupRestoreStatistics` 保存的是聚合计数，不保存每次执行的日期序列，因此趋势接口必须读取业务 CR 的执行事实。

## Goals
- 提供一个稳定的备份恢复任务趋势接口。
- 保持现有统计接口语义不变。
- 复用现有来源识别与备份执行结果判定。
- 让前端直接消费 `buckets[]` 渲染柱状图。

## Non-Goals
- 不生成图表配置。
- 不改 `BackupRestoreStatistics` CRD。
- 不在查询接口中写入 Kubernetes 对象。
- 不支持超过 90 天的查询窗口。

## Route

```http
GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/tasks/progress
```

### Query Parameters

| 参数 | 默认值 | 合法值 | 说明 |
| --- | --- | --- | --- |
| `type` | 无 | `backup`, `restore` | 任务类型 |
| `scope` | `all` | `all`, `disaster`, `app` | 来源范围 |
| `range` | `7d` | `7d`, `30d`, `90d` | 日期窗口 |
| `namespace` | 空 | 任意合法 namespace | 管理侧命名空间过滤 |
| `cluster` | 空 | 任意集群名 | `backup` 表示源集群，`restore` 表示目标集群 |
| `timezone` | `Asia/Shanghai` | IANA timezone | 日期分桶时区 |

非法参数返回 `400`，业务码固定为 `1000`。

## DTO

```go
type TaskProgressTrendDTO struct {
    Type      string                       `json:"type"`
    Scope     string                       `json:"scope"`
    Range     string                       `json:"range"`
    Timezone  string                       `json:"timezone"`
    StartTime metav1.Time                  `json:"startTime"`
    EndTime   metav1.Time                  `json:"endTime"`
    Summary   TaskProgressCountDTO         `json:"summary"`
    Buckets   []TaskProgressBucketDTO      `json:"buckets"`
    Series    []TaskProgressSeriesDTO      `json:"series"`
    Sources   []TaskProgressSourceDTO      `json:"sources"`
}

type TaskProgressCountDTO struct {
    Total      int32 `json:"total"`
    InProgress int32 `json:"inProgress"`
    Completed  int32 `json:"completed"`
    Failed     int32 `json:"failed"`
    Canceled   int32 `json:"canceled"`
    Unknown    int32 `json:"unknown"`
}

type TaskProgressBucketDTO struct {
    Date       string `json:"date"`
    Total      int32  `json:"total"`
    InProgress int32  `json:"inProgress"`
    Completed  int32  `json:"completed"`
    Failed     int32  `json:"failed"`
    Canceled   int32  `json:"canceled"`
    Unknown    int32  `json:"unknown"`
}

type TaskProgressSeriesDTO struct {
    Key   string `json:"key"`
    Label string `json:"label"`
}

type TaskProgressSourceDTO struct {
    Scope     string `json:"scope"`
    Label     string `json:"label"`
    Total     int32  `json:"total"`
    Completed int32  `json:"completed"`
    Failed    int32  `json:"failed"`
}
```

### Response Example

```json
{
  "code": 0,
  "message": "OK",
  "data": {
    "type": "backup",
    "scope": "all",
    "range": "7d",
    "timezone": "Asia/Shanghai",
    "startTime": "2026-04-01T00:00:00+08:00",
    "endTime": "2026-04-07T23:59:59+08:00",
    "summary": {
      "total": 190,
      "inProgress": 0,
      "completed": 142,
      "failed": 48,
      "canceled": 0,
      "unknown": 0
    },
    "buckets": [
      {
        "date": "2026-04-01",
        "total": 52,
        "inProgress": 0,
        "completed": 36,
        "failed": 16,
        "canceled": 0,
        "unknown": 0
      }
    ],
    "series": [
      { "key": "completed", "label": "备份成功" },
      { "key": "failed", "label": "备份失败" }
    ],
    "sources": [
      { "scope": "disaster", "label": "容灾备份", "total": 126, "completed": 96, "failed": 30 },
      { "scope": "app", "label": "应用备份", "total": 64, "completed": 46, "failed": 18 }
    ]
  },
  "meta": {
    "resourceType": "backupRestoreTaskProgress"
  },
  "trace_id": "..."
}
```

## Data Source

### Backup
`type=backup` 读取 `AppBackup.Status.History`。

计数规则：
- `Completed` 计入 `completed`。
- `Failed`、`PartiallyFailed`、`FailedValidation` 计入 `failed`。
- `Canceled` 计入 `canceled`。
- `InProgress`、`New` 计入 `inProgress`。
- 其余状态计入 `unknown`。

时间规则：
- 终态记录使用 `CompletionTimestamp`。
- 执行中记录使用 `StartTimestamp`。
- 缺失执行时间的记录不进入分桶。

### Restore
`type=restore` 读取 `AppRestore`。

计数规则：
- `Succeeded` 计入 `completed`。
- `Failed` 计入 `failed`。
- `Cancelled` 计入 `canceled`。
- `Pending`、`Initiating`、`Restoring`、`Deleting` 计入 `inProgress`。
- 其余状态计入 `unknown`。

时间规则：
- `RestoreStatus.CompletionTimestamp` 有值时作为终态分桶时间。
- `RestoreStatus.StartTimestamp` 有值时作为执行中分桶时间。
- 两者都缺失时使用 `metadata.creationTimestamp`。

## Scope Mapping

`scope` 与来源映射如下：

| scope | 来源 |
| --- | --- |
| `app` | `inferAppBackupOrigin` 与 `inferAppRestoreOrigin` 结果不是 `disaster-instance` |
| `disaster` | `inferAppBackupOrigin` 与 `inferAppRestoreOrigin` 结果是 `disaster-instance` |
| `all` | 不按来源裁剪 |

容灾来源识别继续复用：
- `DataSync` controller owner
- `ResourceSync` controller owner
- `testudo.softcdata.com/app-resource-origin=disaster-instance`
- 历史命名前缀兜底规则

## Time Window

服务端根据 `timezone` 加载时区。`timezone` 为空时使用 `Asia/Shanghai`。

`range=7d` 的窗口规则：
- `endTime` 为服务端当前时间。
- `startTime` 为 `endTime` 所在自然日向前 6 天的 00:00:00。
- 返回 7 个日期分桶。

`range=30d` 与 `range=90d` 使用同一规则，分别返回 30 个与 90 个日期分桶。

每个分桶以 `YYYY-MM-DD` 字符串表示。没有任务落入的日期必须返回零值。

## Existing API Reuse

复用内容：
- `parseAppResourceOriginFilter`
- `inferAppBackupOrigin`
- `inferAppRestoreOrigin`
- `backupRecordExecutionTime`
- `backupRecordExecutionResult`
- `StatisticsDTO` 的计数字段命名
- 标准 `Envelope`

不直接复用内容：
- `GetBackupSuccessRate` 按 `BackupRestoreStatistics.CreationTimestamp` 过滤，不满足按每次执行时间分桶。
- `GetOperationStatisticsByTime` 统计 `DisasterOperation`，不属于备份恢复任务趋势。
- `BackupRestoreStatistics` 聚合结果缺少每次执行时间序列，不能作为趋势接口主数据源。

## Test Strategy

- 参数校验：
  - 非法 `type`
  - 非法 `scope`
  - 非法 `range`
  - 非法 `timezone`
- 备份趋势：
  - 按 `AppBackup.Status.History` 生成连续分桶
  - `scope=app`
  - `scope=disaster`
  - `scope=all`
- 恢复趋势：
  - `Succeeded`
  - `Failed`
  - `Cancelled`
  - 执行中状态
- 空数据：
  - 返回完整日期分桶
  - 所有计数为 `0`
- 兼容性：
  - 现有 `/backups`、`/restores`、`/backups/success-rate`、`/autobackups/execution-summary` 测试保持通过。
