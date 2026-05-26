# Change: 新增容灾操作详情查看 API

## Why
当前“容灾操作步骤不可查看”的根因不是 operator 没有步骤状态，而是 server 缺少完整的查看链路：
- instance history 只有摘要字段，缺少稳定的 `operationName`
- group watch 已经有丰富 DTO，但 group history 仍然是薄记录
- drill detail 已经回显 `status.currentStep` 与 `steps[]`，页面却没有复用

因此本次 proposal 需要把 P0-P4 收敛成可落地分期：
- P1：直接消费现有 drill detail
- P2：补 history 标识与 detail API
- P3：历史列表点击记录后调用 detail API
- P4：后续再与 durable history / timeline 合流

## What Changes

### 1. P1 定义为现有 Drill DTO 的快速收益
- `GET /disasterdrills.testudo.softcdata.com/v1/drills/:name` 已经返回：
  - `status.currentStep`
  - `steps[]`
  - `status.operationName`
- 本 change 不重写 drill API，只把它明确标记为 P1 可直接落地的页面输入。

### 2. P2 新增 history 标识与 owner-scoped detail route
- 为以下 history route 的每条记录增加稳定标识：
  - `GET /disasterinstances.testudo.softcdata.com/v1/instances/:name/history`
  - `GET /disastergroups.testudo.softcdata.com/v1/groups/:name/history`
- 每条记录至少补齐：
  - `operationName`
  - `operationUID`
  - `hasDetail`
- 新增 detail route：
  - `GET /disasterinstances.testudo.softcdata.com/v1/instances/:name/operations/:operationName`
  - `GET /disastergroups.testudo.softcdata.com/v1/groups/:name/operations/:operationName`

### 3. P3 固定为“history summary + detail on demand”
- history list 继续保持轻量。
- 页面点击某一行以后，使用该行的 `operationName` 调用 P2 detail route。
- 当该行状态为 `Pending`、`Running` 时，页面再订阅单操作 watch。
- 本 change 为 instance 新增：
  - `GET /disasterinstances.testudo.softcdata.com/v1/watch/instances/operations/:operationName`
- group 继续复用现有：
  - `GET /disastergroups.testudo.softcdata.com/v1/watch/groups/operations/:operationName`

### 4. P4 只写依赖关系，不并入本 change
- P4 的 timeline / durable history 合流依赖：
  - `persist-event-history-v2`
  - `add-v2-event-emission-coverage`
- 本 change 不承诺同批实现 execution timeline 合流。

### 5. 接口规范必须在提案阶段锁定
- 所有新增详情接口都必须遵循现有“主资源 + 子资源”路径风格，不引入全局 `/operations/:name` 路由。
- 所有查询类接口固定使用 `GET`，不得在查询接口中引入副作用。
- 所有响应继续使用标准 `Envelope`，返回 DTO，不直接暴露原始 `DisasterOperation` CR。
- `operationName` 存在但 owner 不匹配时，服务端必须返回 `404`，不得回退为模糊匹配。
- instance single-operation watch 的路径风格必须与现有 group watch 保持一致，采用 `/watch/instances/operations/:operationName`。

### 6. 提案必须直接给出字段蓝图
- `history item`、`detail DTO`、`single-operation watch DTO` 的字段集必须在 proposal / design 中直接写明，不允许在实现阶段临时增删。
- `history item` 负责“定位到某次操作”：
  - `time`
  - `type`
  - `status`
  - `autoCancel`
  - `operationName`
  - `operationUID`
  - `hasDetail`
  - 兼容字段 `result/reason/operator/note`
- `detail DTO` 负责“展示某次操作完整过程”：
  - `name`
  - `uid`
  - `namespace`
  - `ownerKind`
  - `ownerName`
  - `operationType`
  - `state`
  - `reason`
  - `currentStep`
  - `message`
  - `steps[]`
  - `autoCancel`
  - `roleStatus`
  - `groupStatus`
  - `startTime`
  - `completionTime`
  - `creationTimestamp`

### 7. 提案必须给出兼容策略
- `history` 旧字段保持可读，新增字段只做增量扩展，不删除现有字段。
- `detail DTO` 与现有 group single-operation watch DTO 保持字段同名，避免 web 再做两套解析器。
- P2 落地后，P3 前端可以先只接 `operationName + detail route`，不要求和 P4 一起上线。

## Non-Goals
- 不新增全局 `DisasterOperation` 列表 route。
- 不替代已有 drill detail route。
- 不通过 message 文本推断步骤。
- 不在本 change 中实现 durable history。

## Impact
- Affected specs:
  - `operation-view`
- Affected code:
  - `internal/apis/disaster_instance/v1/*`
  - `internal/apis/disaster_group/v1/*`
- Cross-repo impact:
  - `disaster-operator`：提供稳定的 `currentStep`、`steps[]`、`autoCancel*`、`groupStatus`
  - `cluster-disaster-web`：实现 P1 页面展示以及 P3 详情抽屉

## Relationship to Existing Changes
- 复用 `add-disaster-drill-api` 已有 DTO，不重开 drill detail 能力。
- 复用 `add-group-operations-watch-route` 与现有 group watch DTO，不破坏现有 group watch 路径。
- 依赖 `add-operation-auto-cancel-result-api` 的 `autoCancelSummary` 字段口径。
- 后续与 `persist-event-history-v2` 做 P4 合流，不在当前 change 中重复建模。

## Acceptance
- 当客户端查询 instance/group history 时，每条记录都能拿到 `operationName`，前端无需再从 `note` 猜测对象。
- 当客户端用 `operationName` 查询 detail route 时，能直接拿到完整 `currentStep + steps[]`。
- 当 `operationName` 不属于当前 owner 时，接口稳定返回 `404`。
- 当同一条记录进入执行中状态时，前端可以无缝切到 single-operation watch，并复用同一套字段解析。
