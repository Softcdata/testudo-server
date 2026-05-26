# Capability: API 交互标准

## Purpose
定义 `disaster-server` 对外暴露 API 的交互标准，包括请求参数、响应格式、分页、排序、过滤和 HATEOAS 链接等，以确保 API 的一致性和易用性。
## Requirements
### Requirement: 查询操作 (Query Operation)
查询操作允许客户端列出集合中的资源。查询操作必须 (MUST) 不产生副作用。

#### Scenario: 成功查询集合
- **WHEN** 客户端发送 GET 请求到集合 URL (例如 `/v1/folders`)
- **THEN** 返回状态码 200 OK
- **AND** 响应体包含 `type: "collection"`
- **AND** 响应体包含 `resourceType` 字段
- **AND** 响应体包含 `data` 数组

### Requirement: 更新操作 (Update Operation)
更新操作允许客户端修改现有资源。为了处理并发修改，更新操作必须 (MUST) 具备冲突处理机制。

#### Scenario: 使用 RetryOnConflict 处理并发更新
- **WHEN** 客户端发送 PUT 请求更新资源
- **THEN** 服务端必须使用 `RetryOnConflict` 机制
- **AND** 在重试循环中重新获取（Get）最新资源
- **AND** 在最新资源基础上应用修改
- **AND** 执行更新操作
- **AND** 如果遇到 `409 Conflict` 错误，自动重试直到成功或超时

### Requirement: 分页 (Pagination)
可能返回大量结果的集合必须 (MUST) 支持分页。

#### Scenario: 使用 Page 分页
- **WHEN** 客户端请求包含 `page={integer}` 参数
- **THEN** 服务端返回指定页码的结果集
- **AND** 响应包含 `pagination` 对象

#### Scenario: 设置每页数量
- **WHEN** 客户端请求包含 `limit={integer}` 参数
- **THEN** 服务端返回不超过指定数量的结果
- **AND** 服务端应强制执行并记录允许的上限（建议至少 1000）
- **AND** 服务端应支持 `limit=0` 以仅获取元数据

#### Scenario: 分页响应结构
- **WHEN** 集合支持分页
- **THEN** 响应必须包含 `pagination` 对象
- **AND** `pagination` 对象包含 `limit` (每页数量), `total` (总数) 和 `partial` (是否截断)
- **AND** `pagination` 对象应包含 `first`, `previous`, `next`, `last` 链接（如果适用）

### Requirement: 排序 (Sorting)
服务必须 (MUST) 允许客户端请求排序后的资源列表。

#### Scenario: 请求排序
- **WHEN** 客户端请求包含 `sort={sort name}` 和 `order={asc or desc}` 参数
- **THEN** 服务端返回按指定字段和顺序排序的结果
- **AND** 排序必须是“稳定”的（Stable）

#### Scenario: 排序响应结构
- **WHEN** 集合支持排序
- **THEN** 响应应包含 `sortLinks` 对象（可排序字段映射）
- **AND** 如果结果已排序，响应应包含 `sort` 对象（当前排序状态）

### Requirement: 过滤 (Filtering)
服务必须 (MUST) 支持集合的搜索/过滤。为了提升用户体验，API 的过滤逻辑采用“一律模糊匹配”原则。

#### Scenario: 请求过滤 (模糊匹配)
- **WHEN** 客户端发送包含 `{label_key}={value}` 的查询参数
- **THEN** 服务端必须 (MUST) 在内存中对该标签执行 `Contains` 匹配
- **AND** 匹配过程不区分是否传有通配符 `*`（通配符将被视作空位或直接移除）
- **AND** 返回满足所有过滤条件的资源（AND 逻辑）
- **AND** 特殊字符必须进行 URL 编码

#### Scenario: 过滤逻辑的平滑降级
- **GIVEN** 被过滤的字段不支持 Kubernetes 的 Label Selector 精确匹配（例如包含非法字符或需要模糊筛选）
- **WHEN** 服务端处理请求
- **THEN** 服务端应先利用 Selector 进行广义筛选，再在内存中执行精确的模糊过滤匹配
- **AND** 确保最终返回给客户端的结果集准确符合模糊匹配预期

### Requirement: 链接 (Links)
集合响应必须 (MUST) 包含链接以支持 HATEOAS (Hypermedia as the Engine of Application State)。

#### Scenario: 包含 Self 链接
- **WHEN** 返回集合响应
- **THEN** `links` 对象必须包含 `self` 字段，指向当前查询的完整 URL（包含查询参数）

#### Scenario: 包含其他相关链接
- **WHEN** 集合有关联的资源（如 API Schema）
- **THEN** `links` 对象应包含相应的链接（如 `schemas`）
- **AND** 链接必须是绝对 URL

#### Scenario: 集合包含资源详情链接
- **WHEN** 返回集合中的资源列表
- **THEN** 集合的 `links` 对象应包含列表中每个资源的详情链接
- **AND** 链接的键应为资源的唯一标识符（如名称）

### Requirement: 统一响应信封 (Unified Response Envelope)
所有 API 响应（无论是成功还是错误）都必须 (MUST) 使用标准的 `Envelope` 结构。

#### Scenario: 成功响应
给定一个对 `GET /apis/v1/backups/1` 的请求
当请求被成功处理时
那么响应体应匹配：
```json
{
  "code": 0,
  "message": "OK",
  "data": { ... },
  "meta": { ... },
  "trace_id": "..."
}
```

#### Scenario: 错误响应
给定一个包含无效参数的请求
当验证失败时
那么响应体应匹配：
```json
{
  "code": 1000,
  "message": "Validation failed",
  "data": null,
  "meta": { "details": [...] },
  "trace_id": "..."
}
```

### Requirement: 双层错误码 (Double Layer Error Codes)
系统必须 (MUST) 使用“双层”错误码方案，其中 HTTP 状态码表示传输层状态，`code` 字段表示业务逻辑状态。

#### Scenario: 资源未找到
给定一个对不存在资源的请求
当处理程序处理该请求时
那么 HTTP 状态码应为 404
并且响应体中的 `code` 应为 3004 (CodeNotFound)

### Requirement: 数据传输对象 (DTOs)
API 必须 (MUST) 返回仅包含必要业务字段的 DTO，而不是原始的 Kubernetes CRD。

#### Scenario: 备份列表响应
给定一个对 `GET /apis/v1/backups` 的请求
当获取列表时
- **THEN** `data` 中的条目应为简化的 DTO
- **AND** 不应包含 `managedFields` 或完整的 `metadata`

### Requirement: 租户上下文 (Tenant Context)
所有请求必须 (MUST) 包含有效的 `X-Tenant-ID` 请求头，该请求头由中间件进行验证。

#### Scenario: 缺少租户 ID
- **Given** 一个没有 `X-Tenant-ID` 请求头的请求
- **When** 请求到达中间件时
- **Then** 应被拒绝，返回 HTTP 400 和错误码 1000

### Requirement: 分页和过滤
集合 API 必须 (MUST) 支持标准分页和过滤参数，并返回标准化的元数据。

#### Scenario: 分页元数据
- **Given** 对 `GET /apis/v1/backups?page=1&limit=10` 的请求
- **When** 生成响应时
- **Then** `meta` 字段应包含：
```json
{
  "pagination": {
    "limit": 10,
    "total": 50,
    "page": 1
  }
}
```

### Requirement: WebSocket 事件流标准 (WebSocket Event Stream Standards)
所有基于 WebSocket 的事件流接口必须 (MUST) 遵循统一的响应格式和数据传输对象 (DTO) 规范。

#### Scenario: 标准化消息结构
- **WHEN** 服务端通过 WebSocket 推送事件
- **THEN** 消息体必须 (MUST) 是一个标准的 `Envelope` JSON 对象
- **AND** `code` 字段应为 0 (成功) 或错误码
- **AND** `data` 字段应包含 `WatchEventDTO`

#### Scenario: WatchEventDTO 结构
- **WHEN** 返回 `WatchEventDTO`
- **THEN** 它必须 (MUST) 包含 `type` 字段 (ADDED, MODIFIED, DELETED, ERROR)
- **AND** 它必须 (MUST) 包含 `object` 字段，该字段是资源的 DTO 表示，而不是原始 Kubernetes 对象

#### Scenario: 心跳消息
- **WHEN** 发送心跳保活消息
- **THEN** 应发送一个特殊的 `Envelope`
- **AND** `meta` 字段中包含 `type: "heartbeat"`

### Requirement: 历史事件聚合 API (Historical Event Aggregation)
系统必须 (MUST) 提供能够聚合、解析并展示结构化后台任务历史记录的 API。

#### Scenario: 聚合结构化事件
- **WHEN** 客户端请求历史事件列表（如 `/apis/v1/events` 或 `/apis/v1/:resource/:name/history`）
- **THEN** 服务端必须从 Kubernetes Events 中筛选 Reason 为 `ExecutionStarted`、`ExecutionProgress`、`ExecutionFinished` 的记录
- **And** 必须将 `Event.message` 按 JSON 结构化载荷解析（字段至少包含 `task`、`status`、`message`）
- **And** 对于同一任务应聚合为单一 `TaskEvent` 对象，任务键必须使用复合键 `task + traceId + involvedObject.uid`
- **And** 当 `traceId` 缺失时，必须使用 `task + involvedObject.uid + startedAtAnchor` 作为兜底键，避免跨批次误合并
- **And** 响应中的 `id` 必须映射为关联业务对象（InvolvedObject）的 `UID`
- **And** 对于无法解析为 JSON 的事件，服务端应忽略且不进入聚合结果

#### Scenario: 资源历史与资源流 Kind 隔离
- **WHEN** 客户端请求 `GET /apis/v1/:resource/:name/history` 或 `GET /apis/v1/watch/:resource/:name/events`
- **THEN** 服务端必须将 `:resource` 解析为目标 Kind，并只处理 `involvedObject.kind=目标Kind` 的事件
- **And** 对于不支持的 `:resource`，服务端必须返回 400
- **And** 不得仅依赖 `involvedObject.name` 进行事件归属判定

#### Scenario: 动态耗时计算
- **WHEN** 一个任务的最新状态为 `InProgress`
- **THEN** 服务端必须在 `StartTime` 可用时动态计算耗时：`Duration = time.Since(StartTime)`
- **And** 返回格式应与静态耗时保持一致（如 `45s`, `5m10s`）
- **And** 当 `StartTime` 不可用时应返回占位值（如 `-`）

#### Scenario: 过滤与搜索
- **WHEN** 客户端提供 `ownerUID`, `ownerName`, `taskType`, `status`, `traceId`, `keyword`, `startTime`, `endTime` 参数
- **THEN** 服务端必须支持对聚合后的 `TaskEvent` 进行相应的过滤
- **And** `ownerUID` 对应业务对象的 `UID`
