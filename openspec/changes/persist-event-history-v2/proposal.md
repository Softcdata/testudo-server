# Change: 为全局任务事件引入可持久化的历史记录能力

## Why
当前 server 的事件审计链路仍直接读取 Kubernetes Events：
- `/apis/v1/events`
- `/apis/v1/:resource/:name/history`

这条链路虽然已经具备结构化 JSON 聚合、复合聚合键和 Kind 隔离，但仍受 Kubernetes Event TTL 限制，无法稳定支撑“跨天/跨周/跨月”的审计查询。

已有 active change `persist-events-dm` 直接把实现收敛到 GORM + 达梦表结构，这个切入点过早绑定了存储后端，也没有先把“历史事件能力”和“实时事件能力”的边界抽象清楚。

条目 9 需要的不是“先选数据库”，而是先正式定义：
- 什么是 durable event history
- durable history 与实时 watch 的关系
- execution identity / source snapshot / timeline node 的最小契约
- 去重、回放、查询一致性的边界

## What Changes

### 1. 在 server 正式化 `global-events` capability
- 新增 server 侧 `global-events` capability。
- capability 负责定义全局任务事件的历史持久化、查询契约和实时/历史边界。

### 2. 引入 durable history 抽象，而不是先绑定具体数据库
- server 增加 `HistoryStore` / `HistoryProjector` 这类逻辑抽象。
- proposal 第一层不锁定达梦、MySQL 或其他实现后端。
- 存储选型、索引、TTL 和迁移策略下沉到 `design.md` 与后续实现阶段。

### 3. 定义 execution history 记录模型
- durable history 必须区分“执行实例身份”和“资源身份”。
- 历史记录必须保留：
  - execution identity
  - owner resource snapshot (`uid/kind/name/namespace`)
  - `task/status/message/traceId/cluster/user/reason`
  - `startTime/endTime/duration`
  - timeline nodes
  - source event identity（用于去重）

### 4. 明确实时事件与历史事件的关系
- `watch` 路径继续消费实时 Kubernetes Events。
- 历史查询路径改为消费 durable history。
- 两者必须共享同一套结构化聚合语义，但不能互相阻塞。

### 5. 定义 API 兼容策略
- 首期保持现有事件列表/历史路由不变。
- 为 durable history 增加稳定的 `executionId` 概念，避免继续把资源 UID 误当作一次执行的唯一标识。
- 兼容期内保留现有客户端可消费字段，避免前端审计页和资源详情页一次性断裂。

## Non-Goals
- 不在本 proposal 第一层锁定具体数据库产品。
- 不在本 proposal 中重写实时 watch 协议。
- 不在本 proposal 中重写 operator 的事件格式、task naming 或 trace 传播机制。
- 不要求补历史全量回灌到 feature 启用之前。

## Impact
- Affected specs:
  - `global-events`（server 新 capability）
- Affected code:
  - `internal/apis/event/v1/*`
  - `internal/router/router.go`
  - `cmd/app/disaster.go`
  - `configs/*`
- Cross-repo impact:
  - `disaster-operator`：需要 companion proposal 锁定 durable history 可依赖的发射契约
  - `cluster-disaster-web`：事件审计页、实例/组历史记录页需要适配 `executionId` 与新字段

## Relationship to Existing Changes
- 参考 active change：`persist-events-dm`
- 本 change 不沿用其“达梦优先”建模方式，而是上升一层先定义能力边界。
- 若本 proposal 获批，`persist-events-dm` 后续应被收敛为实现素材或废弃，不应继续作为最终规范入口。
