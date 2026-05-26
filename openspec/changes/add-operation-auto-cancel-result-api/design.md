# Design: Operation Auto Cancel Result API

## 设计目标
- 上层必须能稳定回答四个问题：
  - 有没有触发自动补偿
  - 自动补偿是否成功
  - 为什么触发
  - 是否仍需人工介入

## DTO 契约

### 统一摘要对象
server 侧引入统一的 `AutoCancelSummaryDTO`：

```go
type AutoCancelSummaryDTO struct {
    Triggered                  bool         `json:"triggered"`
    Status                     string       `json:"status"` // NotTriggered|Running|Succeeded|Failed
    Reason                     string       `json:"reason,omitempty"`
    TriggerStep                string       `json:"triggerStep,omitempty"`
    ManualInterventionRequired bool         `json:"manualInterventionRequired"`
    TriggeredAt                *metav1.Time `json:"triggeredAt,omitempty"`
    CompletionTime             *metav1.Time `json:"completionTime,omitempty"`
}
```

### 投影位置
- instance detail：返回完整 `autoCancel` 摘要
- instance list：返回同名摘要对象，但不展开补偿步骤
- instance history：对每条 operation history 记录补 `autoCancel` 摘要
- group operation watch DTO：直接镜像 operator status 中的同名摘要字段

## 聚合规则

### detail/list
- 对实例 detail/list，选择“最新的、属于该实例的 failover `DisasterOperation`”作为摘要来源。
- 若该 operation 未写入任何 auto-cancel 字段，则 `autoCancel` 置空。
- 若最新 operation 不是 failover 且不携带 auto-cancel 字段，server 不向前追溯更老 operation 填充当前摘要，避免展示过期补偿状态。

### history
- `GET /instances/:name/history` 中每条 history record 直接读取对应 `DisasterOperation.status` 的 auto-cancel 字段。
- 不通过 `message` 关键词猜测补偿结果。

### watch
- 组操作 watch DTO 直接镜像 `DisasterOperation.status` 的 auto-cancel 字段，避免前端再拼接 status/reason/message。

## 时间线投影
- 时间线节点来源优先采用 operator 的稳定状态字段和时间戳：
  - failover 失败节点：来自 operation 失败步骤或 operation 失败时刻
  - auto-cancel triggered 节点：来自 `autoCancelTriggeredAt`
  - auto-cancel success/failure 节点：来自 `autoCancelCompletionTime + autoCancelStatus`
- 首期 timeline node 语义固定为：
  - `FailoverFailed`
  - `AutoCancelTriggered`
  - `AutoCancelSucceeded` 或 `AutoCancelFailed`
- 若现有 event timeline 同时存在结构化事件，则允许复用展示，但不得以事件文本作为唯一数据源。

## 边界
- server 不自行推断补偿结果，只消费 operator 提供的稳定状态字段。
- server 不新增新的 action route。
- server 不改变 operator `status.state` 的语义，只额外投影 auto-cancel 摘要。
