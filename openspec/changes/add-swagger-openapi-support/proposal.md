# Change: 增加 Swagger 与 OpenAPI 支持

## Why
项目准备发版，RunAPI 已经开始按完整调用链补充详细说明，但当前服务缺少可在线访问、可校验、可被前端工具消费的 Swagger/OpenAPI 契约。

`disaster-server` 的接口语义依赖 server handler、请求结构、响应 DTO、operator CRD、controller 状态机、Velero 资源以及 Kubernetes 原生资源。仅依赖 RunAPI 页面无法建立发版门禁，也无法让前端基于统一契约进行稳定集成。

## What Changes
### 1. 建立 OpenAPI 单一事实源
新增 OpenAPI 3.0.3 契约文件，固定路径为 `openspec/specs/disaster-server-openapi.yaml`。该文件作为 Swagger 展示、接口对账、发版校验的事实源。

### 2. 提供 Swagger 在线访问能力
服务在配置 `swagger.enabled=true` 时注册以下文档路由：
- `GET /openapi.yaml`
- `GET /openapi.json`
- `GET /swagger/`

这些文档路由不进入业务 JWT 鉴权链路，是否注册完全由 `swagger.enabled` 决定。

### 3. 建立 RunAPI、server、OpenAPI 三方对账
沿用 RunAPI 详细说明治理中的 `METHOD + 标准化路径` 对账键，分别整理 server 路由清单、RunAPI 接口清单、OpenAPI 路径清单，并输出差异清单。

### 4. 继承 RunAPI 五段详细说明标准
OpenAPI 的 `operation.description` 必须使用 RunAPI 治理中定义的五段结构：
- 接口用来干什么
- 控制哪些资源
- 入参详细说明
- 返回详细说明
- 可能返回的错误

### 5. 固化字段取证来源
每个接口的 OpenAPI schema 与说明必须基于完整调用链确认：
- server router 与 handler
- server request struct、校验、默认值、转换逻辑
- server DTO 与响应转换函数
- operator CRD Spec 与 Status 类型
- operator controller 对字段的使用方式、状态机、失败 reason 与 message
- Velero 以及 Kubernetes 原生资源的实际影响

### 6. 支持 WebSocket 接口表达
WebSocket watch 接口必须作为 `GET` upgrade 接口进入 OpenAPI，并通过扩展字段标记协议、消息结构、鉴权方式以及事件数据 schema。

### 7. 建立发版前校验门禁
新增 OpenAPI 校验、server 路由对账、RunAPI 对账、Swagger UI 可访问性检查。校验失败时，发版检查结果必须标记失败。

## Non-Goals
- 不修改现有业务接口路径。
- 不修改现有请求结构、响应结构、错误码语义。
- 不改变 RunAPI 目录体系。
- 不删除 RunAPI 已有说明。
- 不在本变更中生成前端 SDK。
- 不在本变更中生成 server handler 骨架。

## Impact
- Affected specs:
  - `api-documentation-standards`
- Affected repository files:
  - `openspec/specs/disaster-server-openapi.yaml`
  - `internal/router/router.go`
  - Swagger 静态文件嵌入代码
  - OpenAPI 校验脚本
  - server 路由导出脚本
  - RunAPI 对账脚本
  - CI 发版校验脚本
- External documentation scope:
  - RunAPI/ApiPost 项目中的接口、目录、详细说明、请求示例与响应示例

## Acceptance
- `openspec/specs/disaster-server-openapi.yaml` 使用 OpenAPI 3.0.3，并通过严格校验。
- `GET /openapi.yaml` 返回 YAML 契约。
- `GET /openapi.json` 返回 JSON 契约。
- `GET /swagger/` 可以打开 Swagger UI，并加载同一份 OpenAPI 契约。
- server 全量接口均进入 OpenAPI，包括 `/apis`、`/api`、`/login`、`/refresh_token`、`/healthz`、`/readyz` 与 WebSocket watch 接口。
- OpenAPI、RunAPI、server 三方清单按 `METHOD + 标准化路径` 对账后差异为 0。
- 所有需要 JWT 的接口在 OpenAPI 中标注 `bearerAuth`。
- 所有公开接口在 OpenAPI 中显式标注 `security: []`。
- 所有 WebSocket 接口标注 `x-disaster-protocol: websocket`，并声明服务端推送消息结构。
- 每个接口包含 `summary`、`description`、`operationId`、`tags`、`parameters`、`requestBody`、`responses`、`x-runapi-target-id`、`x-controlled-resources`、`x-operator-chain`。
- 每个接口的 `description` 使用五段结构，并说明同步 HTTP 错误与 operator 后续回写失败。
- 发版校验脚本覆盖 OpenAPI 校验、server 路由对账、RunAPI 对账、Swagger UI 可访问性检查。
