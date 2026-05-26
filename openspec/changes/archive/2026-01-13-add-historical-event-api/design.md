# 设计方案：历史事件列表接口

## 系统架构
本功能通过直接查询 Kubernetes 的 `Event` 资源实现。由于 K8s 事件是点位记录（Point-in-time），我们需要通过特定的规则解析和聚合这些事件，以满足业务对“任务历史”的定义。

## API 定义

### 1. 全局历史记录查询
`GET /apis/v1/events`

**查询参数**:
- `taskType`: 任务类型筛选（如 AppBackup, AppRestore）。
- `status`: 状态筛选（Success, Failed, InProgress）。
- `startTime`/`endTime`: 时间范围筛选 (RFC3339 格式)。
- `keyword`: 模糊搜索，匹配 `TaskName`、`Cluster` 或 `Namespace`。
- `ownerName`: 按关联资源名称精确/模糊筛选。
- `ownerUID`: 按关联资源 UID 精确筛选。

### 2. 指定资源执行记录查询
`GET /apis/v1/:resource/:name/history`

### 3. 全局事件实时流 (WebSocket)
`GET /apis/v1/events/watch`

**Query Params**:
- 与列表接口一致，支持 `taskType`、`status`、`keyword` 等筛选。
- 必须支持 `LabelSelector` 对应的查询参数（如下文实现逻辑所述）。

### 4. 指定资源事件实时流 (WebSocket)
`GET /apis/v1/:resource/:name/events/watch`

## 数据模型 (DTO)

```go
type TaskEventDTO struct {
    ID           string    `json:"id"`           // 业务对象 UID
    Time         time.Time `json:"time"`
    TaskType     string    `json:"taskType"`     // 备份, 恢复, 容灾切换, 等
    TaskName     string    `json:"taskName"`
    Namespace    string    `json:"namespace"`
    Cluster      string    `json:"cluster"`
    Status       string    `json:"status"`       // Success, Failed, InProgress, Canceled
    Duration     string    `json:"duration"`     // 耗时，例如 "5m30s"
    TriggeredBy  string    `json:"triggeredBy"`  // 触发人
    TraceID      string    `json:"traceId"`      // 追踪ID
    Message      string    `json:"message"`
}
```

响应必须遵循统一信封格式，且 `type` 为 `collection`。

## 实现逻辑

### 1. 事件聚合与状态判定
Server 需要遍历获取到的 Events，并按“任务实例”（通常基于 InvolvedObject 和时间点/名称）进行分组：

- **InProgress 判定**:
    - 如果发现了 `Reason: ExecutionStarted` 且没有后续的 `ExecutionFinished`。
    - **耗时计算**: `Duration = Now() - Event.Time`。这是动态计算的。
- **终态判定**:
    - 如果发现了 `Reason: ExecutionFinished`。
    - **耗时计算**: 直接从 Event Message 的 `[Duration: ...]` 标签中解析。

### 2. 任务类型映射
- `involvedObject.kind == "AppBackup"` -> `备份`
- `involvedObject.kind == "AppRestore"` -> `恢复`

### 3. 触发人提取
- 优先从 Event Message 的 `[User: ...]` 标签解析。
- 兜底：从关联 CRD 资源的 Annotations 获取。

## Operator 协同
Operator 必须确保实施“起止双事件”模式：
1. 任务开始时发射 `ExecutionStarted`。
2. 任务结束时发射 `ExecutionFinished`。
