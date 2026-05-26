# Disaster Server API 风格指南（初版）

本指南定义 `disaster-server` 的统一请求/响应规范、资源设计、异步作业、分页筛选、错误码、版本化、鉴权与可观测等约定，参考 Kubernetes、Velero、Argo CD 等云原生项目最佳实践。

## 1. 统一请求与响应

- 头部约定：
  - `X-Request-ID`：请求唯一标识（服务端生成与透传）
  - `X-Tenant-ID`：多租户标识（必选，管理员可跨租户）
  - `Authorization: Bearer <jwt>`：认证令牌
  - `Content-Type: application/json`
  - `Idempotency-Key`：幂等写操作键（可选）

- 响应包裹（Envelope）：
  - 成功：
    ```json
    {"code":0,"message":"OK","data":{...},"meta":{...},"trace_id":"..."}
    ```
  - 失败：
    ```json
    {"code":1234,"message":"校验失败","data":null,"meta":{"details":[...]},"trace_id":"..."}
    ```

- 严格校验与默认值：后端按 JSON Schema 校验入参，拒绝未定义字段（`additionalProperties:false`），并进行默认值填充。

## 2. 资源与动作建模

- 遵循 REST：资源使用复数名，如 `/backups`, `/restores`, `/storages`, `/policies`, `/clusters`。
- 动作使用子资源或 action 后缀：
  - `POST /backups/{id}:cancel` 取消备份
  - 或 `POST /backups/{id}/actions/cancel`
- 写操作返回异步 `Operation`，前端通过 `GET /operations/{operation_id}` 轮询或订阅状态。

### 验证接口规范

- 路径约定：所有验证类接口统一使用 `/validate` 后缀，例如 `POST /storages/connection/validate` 或 `POST /clusters/kubeconfig/validate`。
- 响应规范：
  - 验证成功：`data` 为 `true`，`meta` 为 `null`。
  - 验证失败：`data` 为 `false`，`meta` 中包含错误详情（如 `error` 字段）。
  - 示例：
    ```json
    // 成功
    {"code":0,"message":"OK","data":true,"meta":null,"trace_id":"..."}
    
    // 失败
    {"code":0,"message":"OK","data":false,"meta":{"error":"Invalid credentials"},"trace_id":"..."}
    ```

## 3. 异步作业与状态机

- `Operation` 字段：
  - `operation_id`、`type`（Backup|Restore|...）、`status`（Pending|Running|Succeeded|Failed|Canceled）
  - `progress`（0-100）、`started_at`、`finished_at`
  - `owner_resource`（资源类型与ID）、`error`（结构化错误）

- 生命周期：创建→排队→执行→完成/失败/取消；支持重试策略（次数、退避）。
- 事件与订阅：支持 SSE/WS 订阅 `operations`，或注册 `webhooks` 接收状态变更。
- 与 Velero 对齐：`phase`、`warnings`、`errors` 映射并拉平为统一字段。

## 4. 分页、筛选、排序

- 请求参数（沿用当前模式）：`page`、`limit`、`sort_by`、`sort_order`、`filters`
- 过滤表达：`filters` 使用 `key:op:value` 多项逗号分隔，如 `status:eq:Completed,cluster_id:eq:abc`
- 响应 `meta`（集合响应统一结构）：
  ```json
  {
    "pagination": {
      "limit": 10,
      "total": 50,
      "partial": true,
      "first": "/apis/...?page=1&limit=10",
      "previous": "/apis/...?page=1&limit=10",
      "next": "/apis/...?page=3&limit=10",
      "last": "/apis/...?page=5&limit=10"
    },
    "sort": {
      "name": "creationTimestamp",
      "order": "asc"
    },
    "filters": {
      "region": "us-east-1"
    },
    "links": {
      "self": "/apis/...?page=2&limit=10&sort=creationTimestamp",
      "cluster-1": "/apis/.../clusters/cluster-1"
    },
    "type": "collection",
    "resourceType": "disasterCluster"
  }
  ```

## 5. 错误码与问题定位

- 错误码分层：
  - 0：成功
  - 1xxx：客户端错误（校验失败、语义不合法）
  - 2xxx：认证鉴权错误
  - 3xxx：资源错误（不存在、冲突、配额超限）
  - 4xxx：依赖/外部错误（K8s/Velero 调用失败、网络）
  - 5xxx：服务内部错误（未知、存储、序列化）

- 错误结构：`code`、`message`（用户可读）、`details`（字段错误列表或上游错误码）、`trace_id`。
- 定位：统一 `trace_id` 贯穿日志、事件、操作记录；日志包含 `request_id`、用户、租户、资源ID、集群ID。

### HTTP 状态与业务 code 的关系

- 建议采用“双层编码”模式：HTTP 状态用于网络/协议层语义，`code` 用于业务语义与细分错误。
- 映射原则：
  - 成功：HTTP `200/201/204`，业务 `code=0`。
  - 客户端入参错误：HTTP `400`，业务 `code=1xxx`（例如校验失败、语义不合法）。
  - 未认证：HTTP `401`，业务 `code=2xxx`。
  - 无权限：HTTP `403`，业务 `code=2xxx`。
  - 资源不存在：HTTP `404`，业务 `code=3xxx`。
  - 冲突（如幂等冲突/版本冲突）：HTTP `409`，业务 `code=3xxx`。
  - 依赖/上游错误（K8s/Velero 调用失败）：HTTP `502/503/504`，业务 `code=4xxx`。
  - 服务内部错误：HTTP `500`，业务 `code=5xxx`。

- **Kubernetes 错误映射规范**：
  - `errors.IsNotFound(err)` -> HTTP `404` / Code `3004`
  - `errors.IsAlreadyExists(err)` -> HTTP `409` / Code `3009`
  - `errors.IsConflict(err)` -> HTTP `409` / Code `3009`
  - 其他未知 K8s 错误 -> HTTP `500` / Code `5000`
  - 服务内部错误：HTTP `500`，业务 `code=5xxx`。

- 统一响应包：无论成功或失败，均返回 Envelope，失败时 `data` 通常为 `null`，`meta.details` 可携带字段级错误或上游错误信息。
- 写操作的异步语义：创建成功但业务执行异步（如备份创建返回 operation），HTTP 使用 `200/201`，`data` 中携带 `operation_id`，后续失败将体现在 `GET /operations/{id}` 的 `status` 与 `error` 字段中，而非创建请求的 HTTP 码。

## 6. 版本化与演进

- 路径版本：`/api/v1/...`，保留 `v2` 进行非兼容演进。
- 兼容策略：新增字段向后兼容；删除/语义变更遵循 deprecation 流程，并在文档标注移除计划。
- Schema 版本：资源对象包含 `schema_version` 以支持数据迁移。

## 7. 鉴权、RBAC、多租户

- 认证：JWT/OIDC；短期内可使用内置用户。
- 授权（RBAC）：
  - 动词：`get`, `list`, `create`, `update`, `delete`, `operate`
  - 资源：`backups`, `restores`, `storages`, `policies`, `clusters`
  - 作用域：`tenant` / `cluster` / `namespace`
- 多租户：必须带 `X-Tenant-ID`；管理员可跨租户操作但需审计。

## 8. 可观测与审计

- 日志：结构化 JSON，统一字段（`trace_id`, `request_id`, `user`, `tenant`, `resource`, `operation_id`）。
- 指标（Prometheus）：
  - `disaster_server_requests_total{route,method,status}`
  - `disaster_server_operation_duration_seconds{type,status}`
  - `disaster_server_velero_calls_total{action,result}`
- 追踪：W3C TraceContext，建议接入 OpenTelemetry。
- 审计：记录敏感操作（创建、删除、策略与存储修改）。

## 9. OpenAPI 契约与生成链路

- 单一事实源：OpenAPI 3.0 规范文件 `openspec/specs/disaster-server-openapi.yaml`。
- 代码生成：前端/后端从 OpenAPI 生成 SDK 与接口骨架，禁止手写偏差。
- 文档与校验：整合 Swagger UI 与 Postman；CI 校验请求/响应与 OpenAPI 一致性。

## 10. 资源建模示例

- Backup：
  - `id`, `name`, `cluster_id`, `namespace`, `labels`, `annotations`
  - `spec`: `included_namespaces`, `included_resources`, `storage_id`, `schedule_id`, `ttl`, `hooks`
  - `status`: `phase`, `progress`, `warnings`, `errors`, `start_time`, `completion_time`, `operation_id`
- Restore：结构与 Backup 类似。
- Storage：后端存储配置。
- Policy：策略与配额。
- Operation：异步作业统一结构。

## 11. 接口示例

- 创建备份：
  - `POST /api/v1/backups`
  - Request Body：`name`, `cluster_id`, `spec:{...}`；Header：`Idempotency-Key`, `X-Tenant-ID`
  - Response：
    ```json
    {"code":0,"message":"OK","data":{"backup":{"id":"...","name":"...","status":{"phase":"Pending","operation_id":"..."}}},"trace_id":"..."}
    ```

- 查询作业：
  - `GET /api/v1/operations/{id}`
  - Response：
    ```json
    {"code":0,"data":{"operation":{"status":"Running","progress":35,"error":null}},"trace_id":"..."}
    ```

- 列表统一：
  - `GET /api/v1/backups?page=1&limit=20&sort_by=created_at&sort_order=desc&filters=status:eq:Completed,cluster_id:eq:abc`
  - Response：
    ```json
    {
      "code": 0,
      "data": {"items": [...]},
      "meta": {
        "pagination": {"limit": 20, "total": 123, "partial": true, "first": "/api/v1/backups?page=1&limit=20", "previous": "/api/v1/backups?page=1&limit=20", "next": "/api/v1/backups?page=2&limit=20", "last": "/api/v1/backups?page=7&limit=20"},
        "sort": {"name": "created_at", "order": "desc"},
        "filters": {"status": "Completed", "cluster_id": "abc"},
        "links": {"self": "/api/v1/backups?page=1&limit=20&sort_by=created_at&sort_order=desc"},
        "type": "collection",
        "resourceType": "backup"
      },
      "trace_id": "..."
    }
    ```

## 12. 后续工作

- 按本指南补充中间件、错误处理器、统一查询解析与 OpenAPI 文件；CI 接入一致性校验。
