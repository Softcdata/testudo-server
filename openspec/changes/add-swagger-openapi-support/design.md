## Context
当前项目已经通过 RunAPI 详细说明治理建立了接口文档的字段说明标准，但 RunAPI 不是可被 CI 严格校验的契约源。项目指南中已经提出 OpenAPI 3.0 单一事实源、Swagger UI、Postman 集成、请求响应一致性校验的方向，本变更负责把该方向落为可执行方案。

`disaster-server` 使用 Hertz 注册路由，接口覆盖 HTTP API 与 WebSocket watch API。接口字段语义来自 server 与 operator 的组合调用链：server 接收入参并转换为 CRD，operator 根据 CRD 驱动 Velero 以及 Kubernetes 原生资源，并把状态与失败原因回写到 Status。Swagger 支持必须继承这个事实，不能只从 handler 注释生成浅层文档。

## Goals
- 固定 OpenAPI 文件位置、版本、字段规范与扩展字段。
- 固定 Swagger UI 暴露路径与注册条件。
- 固定 server、RunAPI、OpenAPI 三方对账方式。
- 固定逐接口取证、补 schema、补说明、校验、勾选的执行顺序。
- 固定 WebSocket 接口在 OpenAPI 中的表达方式。
- 固定发版校验门禁。

## Non-Goals
- 不通过 Go 注释作为唯一文档来源。
- 不生成前端 SDK。
- 不生成 server handler 骨架。
- 不重构现有接口。
- 不改变 RunAPI 详细说明保留规则。

## Decision 1: OpenAPI 文件为 Swagger 唯一契约源
OpenAPI 契约文件固定为：

```text
openspec/specs/disaster-server-openapi.yaml
```

契约版本固定为：

```yaml
openapi: 3.0.3
```

Swagger UI、`/openapi.yaml`、`/openapi.json`、CI 校验全部读取这一份契约。任何脚本生成的中间文件只能进入 `openspec/changes/add-swagger-openapi-support/artifacts/`，不得作为 Swagger 的事实源。

## Decision 2: Swagger 路由注册条件固定
服务仅在配置项满足以下条件时注册 Swagger 文档路由：

```text
swagger.enabled=true
```

注册路径固定为：

```text
GET /openapi.yaml
GET /openapi.json
GET /swagger/
```

这些路由不挂载业务 JWT 中间件。生产配置文件中的 `swagger.enabled` 必须为 `false`，发版验收环境配置文件中的 `swagger.enabled` 必须为 `true`。

## Decision 3: 不采用 Go 注释作为主生成方案
Go 注释只允许作为辅助补充，不能作为 OpenAPI 主数据源。主数据源为：
- server 路由扫描结果
- RunAPI 接口清单与详细说明
- server 请求结构、响应 DTO、校验与转换逻辑
- operator CRD 类型与 controller 调用链

原因：
- 当前接口说明需要解释下层资源影响，单个 handler 注释无法可靠表达 operator controller 行为。
- RunAPI 已经按五段结构补充大量字段语义，直接丢弃会造成重复工作与信息分叉。
- Hertz 与注释生成链路无法自动确认 operator 回写失败字段。

## Decision 4: 三方对账键固定
server、RunAPI、OpenAPI 的对账主键固定为：

```text
METHOD + 标准化路径
```

标准化规则：
- `{{baseurl}}` 不参与对账。
- `/api` 与 `/apis` 按真实 server 路径分别记录。
- OpenAPI `{name}`、RunAPI `:name`、RunAPI `{{name}}` 统一为 `:name` 后参与对账。
- 查询参数不参与主键，必须进入参数 schema。
- WebSocket 接口按 HTTP upgrade 的 `GET` 路径记录，并标记 `WebSocket=true`。
- 历史裸路径不得进入 OpenAPI，例如 `/disasterjobs.testudo.softcdata.com/v1/jobs`。

## Decision 5: OpenAPI operation 硬性字段固定
每个 operation 必须包含：
- `tags`
- `summary`
- `description`
- `operationId`
- `parameters`
- `requestBody`
- `responses`
- `security`
- `x-runapi-target-id`
- `x-controlled-resources`
- `x-operator-chain`

`operationId` 必须全局唯一，命名格式固定为：

```text
模块_动作_资源
```

示例：

```text
cluster_list_clusters
cluster_watch_clusters
storage_update_storage
```

## Decision 6: description 继承五段结构
每个 operation 的 `description` 必须包含以下五段：

```markdown
## 1. 接口用来干什么

## 2. 控制哪些资源

## 3. 入参详细说明

## 4. 返回详细说明

## 5. 可能返回的错误
```

OpenAPI 不承载 RunAPI 的 `## 原有说明`。RunAPI 已有说明继续保存在 RunAPI 内，OpenAPI 只承载当前标准说明。

## Decision 7: 入参与返回字段必须落入 schema
所有 request body、query、path、header 字段必须进入 OpenAPI schema。字段说明必须包含：
- 中文含义
- 是否必传
- 可传入值
- 值的目的
- 约束与默认值
- 字段最终写入的 CRD Spec、metadata、annotation、label 位置

所有 response 字段必须进入 OpenAPI schema。字段说明必须包含：
- 中文含义
- 可能取值
- 字段来源
- 字段用途
- 为空条件

字段来源必须从以下集合选择：
- server 计算
- CRD Spec
- CRD Status
- Velero Status
- Kubernetes metadata
- Kubernetes Event
- 存储系统返回
- 鉴权中间件返回

## Decision 8: 错误说明分为同步错误与异步失败
OpenAPI `responses` 必须列出 server 当场返回的 HTTP 状态码与业务 `code`，至少覆盖：
- `200`
- `201`
- `400`
- `401`
- `403`
- `404`
- `409`
- `500`

实际接口没有对应错误分支时不得虚构状态码。异步执行接口必须在 `description` 和 `x-async-failure-status` 中说明 operator 后续失败回写字段。

扩展字段格式固定为：

```yaml
x-async-failure-status:
  statusFields:
    - status.phase
    - status.reason
    - status.message
  readVia:
    - GET 详情接口
    - GET 列表接口
    - WebSocket 事件流接口
```

## Decision 9: WebSocket 接口表达固定
WebSocket 接口在 OpenAPI 中必须保留为 `get` operation，并包含：

```yaml
x-disaster-protocol: websocket
x-message-schema:
  $ref: "#/components/schemas/WatchEnvelope"
responses:
  "101":
    description: WebSocket 连接升级成功
```

WebSocket `description` 必须说明：
- 连接用途
- 监听资源范围
- 初始快照行为
- 增量事件类型
- 心跳消息结构
- 认证方式
- 断开条件
- operator 状态回写字段

## Decision 10: 安全声明固定
需要 JWT 的业务接口必须写：

```yaml
security:
  - bearerAuth: []
```

公开接口必须写：

```yaml
security: []
```

公开接口包含：
- `GET /healthz`
- `GET /readyz`
- `POST /login`
- `POST /refresh_token`
- `GET /openapi.yaml`
- `GET /openapi.json`
- `GET /swagger/`

## Decision 11: 敏感字段示例固定脱敏
OpenAPI 示例不得出现真实敏感值。敏感字段示例固定使用以下值：
- `password`: `******`
- `token`: `eyJ***`
- `kubeconfig`: `apiVersion: v1\nclusters: []\n`
- `accessKey`: `<ACCESS_KEY_ID_REDACTED>`
- `secretKey`: `********`
- `privateKey`: `<PRIVATE_KEY_PEM_REDACTED>`

## Decision 12: 执行顺序固定
实施阶段必须按以下顺序推进：

1. 完成 server 全量接口清单。
2. 完成 RunAPI 全量接口清单。
3. 完成 OpenAPI 当前清单。
4. 完成 server、RunAPI、OpenAPI 三方差异清单。
5. 输出按模块组织的 Swagger 逐接口勾选清单。
6. 建立 OpenAPI 骨架文件。
7. 按逐接口勾选清单顺序查询 operator 调用链。
8. 逐接口补齐 request schema、response schema、错误响应、扩展字段、五段说明。
9. 单接口完成 Swagger 渲染检查与 OpenAPI 校验后，立即勾选该接口。
10. 接入 Swagger UI 路由。
11. 接入发版校验脚本。
12. 输出 Swagger/OpenAPI 发版验收报告。

在第 1 至第 5 步完成前，不得补写单接口 OpenAPI schema。第 7 至第 9 步必须以单个接口为最小闭环推进。

## Decision 13: 逐接口勾选清单固定
Swagger 逐接口清单格式固定为：

```markdown
- [ ] `METHOD /path` - RunAPI：[已存在/缺失/错位/待复核]；OpenAPI：[缺失/待补/已补]；Schema：[待确认/已确认]；错误：[待确认/已确认]；operator：[待取证/已取证]
```

勾选规则：
- 未完成调用链取证时不得勾选。
- 未完成 request schema 时不得勾选。
- 未完成 response schema 时不得勾选。
- 未完成错误响应说明时不得勾选。
- 未完成 Swagger UI 渲染检查时不得勾选。
- 单接口满足完成标准后必须立即勾选。

## Decision 14: CI 校验固定
发版校验必须执行以下检查：
- OpenAPI 3.0.3 schema 校验。
- OpenAPI YAML 解析校验。
- OpenAPI JSON 转换校验。
- `operationId` 全局唯一校验。
- server 路由清单与 OpenAPI 清单对账。
- RunAPI 清单与 OpenAPI 清单对账。
- WebSocket 接口扩展字段校验。
- Swagger UI 路由可访问性检查。

任一检查失败时，发版校验结果必须失败。

## Risks / Trade-offs
- OpenAPI schema 初期需要人工逐接口补齐，工作量大。
- WebSocket 在 OpenAPI 3.0 中没有原生完整协议建模，需要扩展字段承载消息语义。
- RunAPI 平台数据可能存在历史错位接口，必须以标准化路径对账，不能按目录名称推断。
- operator 版本与 server 依赖版本存在偏差时，字段说明必须以 server `go.mod` 依赖版本为基准，并在证据文件中记录本机 operator 路径。

## Migration Plan
1. 先生成 OpenAPI 骨架与全量 paths。
2. 逐接口补 schema 与说明。
3. 接入 Swagger UI，只在 `swagger.enabled=true` 时注册。
4. 接入 CI 校验脚本。
5. 发版验收环境启用 Swagger。
6. 生产配置保持 `swagger.enabled=false`。

## Open Questions
- `swagger.enabled` 最终放入现有配置结构的具体字段路径需要在实施阶段读取配置代码后确定。
- Swagger UI 静态资源采用嵌入二进制还是从本地目录读取，需要在实施阶段结合当前构建方式确定。
- RunAPI 清单导出脚本是否复用已有治理脚本，需要在实施阶段检查当前 artifact 与脚本状态后确定。
