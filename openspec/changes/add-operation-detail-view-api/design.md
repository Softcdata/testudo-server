# Design: 容灾操作详情查看 API

## 背景
当前 server 的可视化能力呈现三段式断裂：
- drill detail 已经具备步骤 DTO
- group watch 已经具备步骤 DTO
- instance/group history 仍然只有摘要，无法打开某一次操作的完整详情

因此本 change 的核心不是“再加一张列表”，而是建立一条稳定链路：
`history row -> operationName -> detail route -> optional single-operation watch`

## Goals
- 让 instance/group 的历史记录都能打开某一次操作的完整详情。
- 让 detail DTO、group watch DTO、instance watch DTO 共用同一套字段口径。
- 让 P3 页面可以用轻量 history list 承载大多数记录。

## Non-Goals
- 不实现 durable history。
- 不新增全局 operation 列表。
- 不在 server 侧重新推断步骤顺序。

## 决策

### D1. history list 保持轻量，detail 采用按需加载
- history route 继续返回摘要。
- history item 增加 `operationName`、`operationUID`、`hasDetail`。
- 页面点击某条记录以后，再请求 detail route。

### D2. detail route 采用 owner-scoped 路径
- instance 页面使用：
  - `GET /disasterinstances.testudo.softcdata.com/v1/instances/:name/operations/:operationName`
- group 页面使用：
  - `GET /disastergroups.testudo.softcdata.com/v1/groups/:name/operations/:operationName`
- 这样可以直接校验 owner 归属，不必先引入全局 operations collection。

### D3. `operationName` 用于取详情，`operationUID` 预留给 P4
- `operationName` 是当前 live `DisasterOperation` 的稳定读取键。
- `operationUID` 用于后续 durable history / execution identity 对齐。
- 当前 detail route 仍以 `operationName` 为路径参数。

### D4. detail DTO 与 watch DTO 共享字段口径
- detail DTO、instance single-operation watch DTO、group single-operation watch DTO 必须共享：
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

### D5. P1 直接复用 drill detail
- drill 页面不等待本 change 完成。
- 现有 `/drills/:name` 已经足够支撑“先把演练步骤展示出来”。

### D6. P4 只记录依赖，不进入当前实施批次
- timeline 合流依赖 `persist-event-history-v2`。
- operator 事件完整性依赖 `add-v2-event-emission-coverage`。
- 当前 change 只确保未来可以用 `operationUID` 以及稳定 DTO 接入 P4。

## Route Contract

### 1. History Item
instance 与 group history item 新增：

```go
type OperationHistoryItemDTO struct {
    Time          metav1.Time            `json:"time"`
    Type          string                 `json:"type"`
    Status        HistoryStatusDTO       `json:"status"`
    AutoCancel    *AutoCancelSummaryDTO  `json:"autoCancel,omitempty"`
    OperationName string                 `json:"operationName,omitempty"`
    OperationUID  string                 `json:"operationUID,omitempty"`
    HasDetail     bool                   `json:"hasDetail"`
    Result        string                 `json:"result,omitempty"`
    Reason        string                 `json:"reason,omitempty"`
    Operator      string                 `json:"operator"`
    Note          string                 `json:"note,omitempty"`
}
```

规则：
- 当前 history item 由 live `DisasterOperation` 投影时，`hasDetail=true`。
- detail route 找不到对象、或者 owner 不匹配时，返回 `404`。

字段来源：

| DTO 字段 | 来源 | 说明 |
| --- | --- | --- |
| `time` | `metadata.creationTimestamp` | 当前与现有 history 行为保持一致 |
| `type` | `spec.operationType` | 操作类型 |
| `status.state` | `status.state` | 操作状态 |
| `status.reason` | `status.reason` | 机器可读原因 |
| `status.message` | `status.message` | 展示消息 |
| `autoCancel` | `status.autoCancel*` | 复用现有 auto-cancel summary 投影 |
| `operationName` | `metadata.name` | P2 查询 detail 的主键 |
| `operationUID` | `metadata.uid` | P4 execution identity 对齐预留 |
| `hasDetail` | server 投影结果 | live operation 可查时为 `true` |

## 接口规范对齐

### 1. 路径风格
- 本 change 严格沿用当前项目的 owner-scoped 子资源风格：
  - `/instances/:name/history`
  - `/instances/:name/sync-status`
  - `/groups/:name/history`
- 因此 operation detail 固定采用：
  - `/instances/:name/operations/:operationName`
  - `/groups/:name/operations/:operationName`
- 不新增全局 `/operations/:operationName`，避免脱离 owner 语义。

### 2. HTTP 方法
- detail route 与 history route 都属于查询接口，固定使用 `GET`。
- 不在查询接口中触发任何补写、重建、纠偏动作。

### 3. 响应格式
- 所有新增 route 必须继续使用标准 `Envelope`。
- 返回体中的 `data.object` 或 `data` 必须是 DTO，而不是原始 Kubernetes 对象。
- DTO 字段集由本 change 明确列出，前端不得依赖 CR 透传字段。

### 4. 错误语义
- `operationName` 不存在：返回 `404`。
- `operationName` 存在但不属于路径中的 owner：返回 `404`。
- 不允许先按 `operationName` 命中，再忽略 owner 不匹配继续返回数据。

### 5. Watch 风格一致性
- group 现有单操作 watch 保持不变：
  - `/watch/groups/operations/:operationName`
- instance 新增单操作 watch 必须采用同风格：
  - `/watch/instances/operations/:operationName`
- 不引入 `/instances/:name/operations/:operationName/watch` 这类第二种风格。

### 2. Detail DTO

```go
type OperationDetailDTO struct {
    Name              string                            `json:"name"`
    UID               string                            `json:"uid"`
    Namespace         string                            `json:"namespace"`
    OwnerKind         string                            `json:"ownerKind"`
    OwnerName         string                            `json:"ownerName"`
    OperationType     string                            `json:"operationType"`
    State             string                            `json:"state"`
    Reason            string                            `json:"reason,omitempty"`
    CurrentStep       string                            `json:"currentStep,omitempty"`
    Message           string                            `json:"message,omitempty"`
    Steps             []StepStatusDTO                   `json:"steps,omitempty"`
    AutoCancel        *instance.AutoCancelSummaryDTO    `json:"autoCancel,omitempty"`
    RoleStatus        *RoleStatusDTO                    `json:"roleStatus,omitempty"`
    GroupStatus       *GroupStatusDTO                   `json:"groupStatus,omitempty"`
    StartTime         *metav1.Time                      `json:"startTime,omitempty"`
    CompletionTime    *metav1.Time                      `json:"completionTime,omitempty"`
    CreationTimestamp metav1.Time                       `json:"creationTimestamp"`
}
```

说明：
- detail DTO 直接读取 operator 状态字段，不通过 `message` 推断步骤。
- `roleStatus` 主要用于 `failover`、`reprotect`、`undo`。
- `groupStatus` 只在 group operation 场景有值。

字段来源：

| DTO 字段 | 来源 |
| --- | --- |
| `name` | `metadata.name` |
| `uid` | `metadata.uid` |
| `namespace` | `metadata.namespace` |
| `ownerKind` | `spec.instanceName` / `spec.groupName` 推断 |
| `ownerName` | `spec.instanceName` / `spec.groupName` |
| `operationType` | `spec.operationType` |
| `state` | `status.state` |
| `reason` | `status.reason` |
| `currentStep` | `status.currentStep` |
| `message` | `status.message` |
| `steps[]` | `status.steps[]` |
| `autoCancel` | `status.autoCancel*` |
| `roleStatus` | `status.roleStatus` |
| `groupStatus` | `status.groupStatus` |
| `startTime` | `status.startTime` |
| `completionTime` | `status.completionTime` |
| `creationTimestamp` | `metadata.creationTimestamp` |

### 3. Watch Strategy
- instance running operation：使用新增的 instance single-operation watch。
- group running operation：继续使用现有 group single-operation watch。
- completed / failed history item：只请求 detail route，不再建立 watch。

### 4. 兼容策略
- `HistoryDTO` 旧字段 `result`、`reason`、`operator`、`note` 继续保留。
- 新字段只做增量添加：
  - `operationName`
  - `operationUID`
  - `hasDetail`
- `OperationDetailDTO` 与 group single-operation watch DTO 共享字段命名，避免 web 写两份 adapter。

### 5. 验收矩阵
| 场景 | 输入 | 期望输出 |
| --- | --- | --- |
| 历史列表 | 查询 instance history | 每条记录包含 `operationName` |
| 详情查询 | 用 history row 的 `operationName` 查 detail | 返回完整 `steps[]` |
| owner 校验 | 用错误的 owner 查 detail | 返回 `404` |
| 运行中操作 | history row 状态为 `Running` | 可切到 single-operation watch，字段集不漂移 |

## P0-P4 映射
| 阶段 | 目标 | server 交付 |
| --- | --- | --- |
| P1 | drill 步骤先可见 | 复用现有 drill detail，无新增 route |
| P2 | 能查看某一次 instance/group 操作详情 | history 标识 + detail route |
| P3 | 历史列表点击行后打开详情抽屉 | instance single-operation watch + 共享 DTO |
| P4 | 时间线与 durable history 合流 | 本 change 只保留 `operationUID` 与依赖说明 |
