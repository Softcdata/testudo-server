# 设计：全局任务事件的 durable history

## 1. 背景与现状
当前 server 侧全局事件链路已经具备三项基础能力：
- 结构化 JSON 事件解析：`internal/apis/event/v1/types.go`
- 复合聚合键：`task + traceId + involvedObject.uid`，无 trace 时回退启动锚点
- 资源历史 Kind 隔离：`listResourceEvents` / `watchResourceEvents`

但历史查询仍直接读取 Kubernetes Events：
- `GET /apis/v1/events`
- `GET /apis/v1/:resource/:name/history`

因此它天然受以下限制：
- Event TTL 到期后无法查询历史
- 审计结果与 K8s Event 生命周期耦合
- 同一资源多次执行的“执行身份”仍未在 API 中正式建模

## 2. 设计目标
- 为任务事件提供可跨 TTL 的 durable history。
- 把“执行身份”与“资源身份”正式拆开。
- 保持实时 watch 与历史查询的职责分离。
- 保持与 operator 现有结构化事件契约兼容，不重写发射端协议。

## 3. 非目标
- 不在本设计中绑定具体数据库产品。
- 不改变现有 watch endpoint 的 SSE / WS 行为。
- 不要求一次性补 feature 启用前的全部历史数据。

## 4. 逻辑模型

### 4.1 Execution History
每一次任务执行在 history store 中对应一个 durable execution record，至少包含：
- `executionId`：一次执行的稳定身份
- `ownerRef`：`uid/kind/name/namespace`
- `taskName`
- `taskType`
- `traceId`
- `cluster`
- `triggeredBy`
- `status`
- `latestReason`
- `latestMessage`
- `startTime`
- `endTime`
- `duration`
- `timeline[]`

### 4.2 Timeline Node
每个 timeline node 对应一条被接受的结构化任务事件，至少包含：
- `sourceEventUID`
- `time`
- `reason`
- `status`
- `message`
- `rawPayload`（可选，供排障）

### 4.3 身份模型
- `executionId` 表示“一次执行实例”
- `ownerRef.uid` 表示“资源实例”

同一资源可以有多次 execution history；因此 API 不得再仅用资源 UID 充当执行实例主键。

## 5. 投影与去重

### 5.1 Source of Truth
- durable history 的源数据仍然是 operator 发出的结构化任务事件。
- server 不自己推导 task lifecycle，只做接收、归并和持久化。

### 5.2 Aggregation Key
history projector 复用当前实时聚合语义：
- 主键：`task + traceId + involvedObject.uid`
- 无 trace 时：`task + involvedObject.uid + startedAtAnchor`

### 5.3 Idempotency
由于 K8s watch / informer 可能重复投递，history projector 必须按 `sourceEventUID` 做 node 级幂等保护。

约束：
- 同一个 `sourceEventUID` 只允许写入一个 timeline node
- 重放同一事件不得生成重复 timeline
- 同一 execution record 的终态更新必须幂等

## 6. 读写边界

### 6.1 实时路径
以下接口继续直接消费 Kubernetes Events：
- `GET /apis/v1/watch/events`
- `GET /apis/v1/watch/:resource/:name/events`

原因：
- 保持最低延迟
- 不让持久化链路阻塞实时通知

### 6.2 历史路径
以下接口切换为读取 durable history：
- `GET /apis/v1/events`
- `GET /apis/v1/:resource/:name/history`

兼容策略：
- 保持原路由
- 维持现有常用字段
- 增加 `executionId`
- 保留资源身份信息，避免前端把 `executionId` 与 `owner uid` 混淆

### 6.3 失败语义
- realtime watch 与 durable history 解耦
- 当 history store 不可用时：
  - watch 路径不得受影响
  - 历史查询路径应显式失败或返回明确降级状态
  - 不得静默回退到“只查最近 1 小时 K8s Events”，否则会破坏查询语义

## 7. 存储抽象
第一阶段仅要求逻辑抽象，不锁定后端。

推荐抽象：
- `HistoryStore`
  - `UpsertExecution(...)`
  - `AppendTimelineNode(...)`
  - `ListExecutions(...)`
  - `ListExecutionsByOwner(...)`
- `HistoryProjector`
  - 监听结构化任务事件
  - 解析 payload
  - 计算 execution identity
  - 执行幂等投影

配置建议：
- `history.enabled`
- `history.driver`
- `history.dsn`
- `history.retentionDays`
- `history.projector.resyncPeriod`

## 8. API 兼容与前端影响
当前前端事件审计页主要消费：
- `id`
- `time`
- `taskType`
- `taskName`
- `cluster`
- `status`
- `duration`
- `triggeredBy`
- `message`

为避免断裂：
- 首期保留这些字段
- 新增 `executionId`
- 资源详情页和事件审计页后续迁移到以 `executionId` 作为一次执行的稳定身份

## 9. 风险
- 如果继续复用 `id=ownerUID`，多次执行仍无法稳定区分。
- 如果历史查询静默回退到 K8s Events，会导致跨时间范围查询语义不稳定。
- 如果 operator 未持续输出终态事件，durable history 会出现悬挂记录。

## 10. 验证策略
- 单测：
  - execution aggregation
  - sourceEventUID 幂等
  - ownerRef + executionId 区分
- 集成测试：
  - 事件进入 store 后，`/events` 与 `/:resource/:name/history` 返回 durable history
  - history store 故障时，watch 路径仍可工作
- 联调：
  - 前端审计页可展示跨 TTL 的历史记录
