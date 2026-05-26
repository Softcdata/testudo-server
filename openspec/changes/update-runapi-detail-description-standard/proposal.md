# Change: 规范化 RunAPI 详细说明治理

## Why
项目准备发版，RunAPI 中已有大量接口文档，但部分接口缺少足够的详细说明。现有 `api-documentation-standards` 只约束 Postman Collection 导出、中文命名、WebSocket 标识与 `{{baseurl}}` 使用，没有定义 RunAPI 详细说明应写什么、如何核对 server 与 operator 调用链、如何处理已有说明。

`disaster-server` 对外接口大量透传、转换、读取 `disaster-operator` 的 CRD Spec 与 Status。若只看 server 的请求结构与响应 DTO，无法准确说明字段含义、可传入值、下层影响资源、异步状态以及 operator 回写的失败原因。

## What Changes
### 1. 固化 RunAPI 详细说明五段结构
每个接口的详细说明必须只围绕以下五块展开：
- 接口用来干什么
- 控制哪些资源
- 入参详细说明
- 返回详细说明
- 可能返回的错误

### 2. 建立字段取证规则
每个入参与返回字段必须基于完整调用链确认：
- server 路由与 handler
- server request struct、校验、默认值、转换逻辑
- server DTO 与响应转换函数
- operator CRD Spec 与 Status 类型
- operator controller 对字段的使用方式、下层资源创建与状态回写
- Velero 以及 Kubernetes 原生资源的实际影响

### 3. 建立 RunAPI 对账流程
先整理 server 全量接口清单，再整理 RunAPI 全量接口清单，随后按 `METHOD + 标准化路径` 对账：
- RunAPI 缺失的接口进入新增任务
- RunAPI 已存在的接口进入详细说明补充任务
- RunAPI 模块归属不一致的接口进入模块修正任务
- 路径含义接近但模块位置异常的接口必须人工复核后再更新

### 4. 保护已有 RunAPI 说明
已存在详细说明的接口不得覆盖原说明。本次新增的标准化说明必须放在最前面，原说明整体放在 `## 原有说明` 后面，保留原文本。

### 5. 按对账结果逐接口推进
任务清单必须先完成 server 接口清单、RunAPI 接口清单与差异对账，再基于对账结果逐接口推进。每个接口进入修改前必须先查询该接口的 operator 调用链，并具备 server 证据、RunAPI 当前状态、operator 证据、待写说明草稿、更新结果检查记录。

### 6. 维护逐接口勾选清单
差异对账完成后，必须输出一份按模块组织的逐接口勾选清单。后续每完成一个接口的调用链取证、详细说明写入与回读验证，必须立即勾选该接口，不能在最后批量勾选。

## Non-Goals
- 不修改 server 业务代码。
- 不修改 operator 业务代码。
- 不改变现有接口路径、请求结构、响应结构与错误码语义。
- 不重新设计 RunAPI 的整体目录体系。
- 不生成新的 OpenAPI 契约文件。

## Impact
- Affected specs:
  - `api-documentation-standards`
- Affected repository files:
  - `openspec/changes/update-runapi-detail-description-standard/*`
- Read-only analysis scope:
  - `disaster-server/internal/apis/**`
  - `disaster-server/internal/router/router.go`
  - `/home/chenxi/YS/disaster-operator/pkg/apis/disaster/v1/**`
  - `/home/chenxi/YS/disaster-operator/internal/controller/**`
- External documentation scope:
  - RunAPI/ApiPost 项目中的接口、目录、详细说明、请求示例与响应示例

## Acceptance
- 形成 server 全量接口清单，包含方法、路径、模块、handler、认证范围、是否 WebSocket、直接操作资源。
- 形成 RunAPI 全量接口清单，包含项目、目录、接口 ID、方法、路径、已有详细说明状态、模块归属。
- 形成差异清单，明确 RunAPI 缺失接口、已存在接口、模块疑似错位接口、需要人工复核接口。
- 形成按模块组织的逐接口勾选清单，每个清单项包含方法、路径、RunAPI 状态、模块归属状态与处理状态。
- 每个接口的详细说明都按五段结构编写。
- 每个入参字段都说明中文含义、可传入值、传入目的、是否必传、约束与默认值。
- 每个返回字段都说明中文含义、可能取值、字段来源、字段用途、为空条件。
- 每个接口都说明直接控制资源、下层影响资源以及资源作用范围。
- 每个接口都区分 server 当场返回错误与 operator 后续回写失败。
- 已有 RunAPI 说明被完整保留在新增说明之后。
- RunAPI 更新完成后，每个接口通过读取详情确认说明内容、模块位置、路径、方法保持符合预期。
