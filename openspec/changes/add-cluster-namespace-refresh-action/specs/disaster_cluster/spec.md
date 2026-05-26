## ADDED Requirements

### Requirement: 集群 API 必须提供带 type 的 refresh-namespaces action
系统必须 (MUST) 提供一个正式 action，用于触发指定 `Cluster` 的指定统计类型重算。

#### Scenario: 调用 action 触发 workload namespace 统计刷新
- **When** 客户端调用 `refresh-namespaces` action，并提交 `{"type":"workloadNamespaceStats"}`
- **Then** Server 必须向对应 `Cluster` 写入注解 `testudo.softcdata.com/refresh-cluster-stats=workloadNamespaceStats`
- **And** 必须返回 `202 Accepted`
- **And** 响应必须回显目标 `Cluster` 与已接受的 `type`
- **And** `202 Accepted` 只表示 refresh signal 已被接收并写入
- **And** 客户端必须通过 Cluster 读取接口观察刷新结果
- **And** 不得只通过重新查询列表来伪装刷新成功

#### Scenario: 调用 action 触发全部统计刷新
- **When** 客户端调用 `refresh-namespaces` action，并提交 `{"type":"all"}`
- **Then** Server 必须向对应 `Cluster` 写入注解 `testudo.softcdata.com/refresh-cluster-stats=all`
- **And** 必须返回 `202 Accepted`

### Requirement: refresh-namespaces action 必须校验 type
系统必须 (MUST) 对 `refresh-namespaces` action 的 `type` 执行显式枚举校验，拒绝不受支持的值。

#### Scenario: 提交不支持的 type
- **When** 客户端调用 `refresh-namespaces` action，并提交不受支持的 `type`
- **Then** Server 必须返回校验错误
- **And** 不得向 `Cluster` 写入 refresh signal

### Requirement: 集群 API 必须暴露 workload namespace 统计读取字段
系统必须 (MUST) 在 Cluster 详情、列表、watch 响应，以及 cluster names 摘要响应中暴露 workload namespace 统计，使客户端能够读取 refresh 结果。

#### Scenario: 返回 Cluster 详情或列表
- **Given** 对应 `Cluster.status` 已包含 `workloadNamespaceCount`、`workloadNamespaceStats`、`workloadTotalCount`
- **When** 客户端请求 Cluster 详情接口或列表接口
- **Then** Server 返回的 Cluster DTO 必须透传上述三个字段
- **And** `workloadNamespaceCount` 必须表示存在 running `Deployment/StatefulSet` 的命名空间数量
- **And** `workloadNamespaceStats` 与 `workloadTotalCount` 必须表示这些命名空间内的 namespace 级备份资源总数，而不是 `Deployment/StatefulSet` 对象数

#### Scenario: 推送 Cluster watch 事件
- **Given** 对应 `Cluster.status` 已包含 `workloadNamespaceCount`、`workloadNamespaceStats`、`workloadTotalCount`
- **When** 客户端监听 Cluster watch 接口
- **Then** Server 推送的 DTO 必须透传上述三个字段

#### Scenario: 返回 cluster names 摘要
- **Given** 对应 `Cluster.status` 已包含 `workloadNamespaceCount`、`workloadTotalCount`
- **When** 客户端请求 `GET /clusters/names`
- **Then** 每个摘要 DTO 必须包含 `workloadNamespaceCount` 与 `workloadTotalCount`
- **And** `workloadTotalCount` 必须表示存在 running `Deployment/StatefulSet` 的命名空间内 namespace 级备份资源总数

### Requirement: 集群 API 必须读取统一的 namespace 级备份统计口径
系统必须 (MUST) 将 Cluster 的旧统计字段与 workload 统计字段视为同一套 namespace 级备份统计读取契约。

#### Scenario: 返回通用 namespace 统计字段
- **Given** 对应 `Cluster.status` 已包含 `namespaceCount`、`namespaceStats`、`resourceTotalCount`
- **When** 客户端请求 Cluster 详情、列表、watch 或 `GET /clusters/names`
- **Then** Server 返回的 DTO 必须透传这些字段
- **And** `namespaceCount` 必须表示纳入统计口径的非系统 namespace 数量
- **And** `namespaceStats` 与 `resourceTotalCount` 必须表示 namespace 级备份资源统计结果

### Requirement: refresh-namespaces action 写入 signal 时必须遵循并发更新标准
系统必须 (MUST) 在写入 typed refresh signal 时遵循现有 API 并发更新标准，避免覆盖其他 metadata 变更。

#### Scenario: 写入 refresh signal 时遇到资源版本冲突
- **When** Server 在写入 `Cluster.metadata.annotations["testudo.softcdata.com/refresh-cluster-stats"]` 时遇到 `409 Conflict`
- **Then** 必须使用 `RetryOnConflict`
- **And** 必须在每次重试中重新获取最新 `Cluster`
- **And** 必须在最新 `metadata.annotations` 基础上写入 refresh signal
- **And** 不得覆盖无关 annotation 与 label 变更

### Requirement: refresh-namespaces action 不得写入额外审计 annotation
系统必须 (MUST) 将 `refresh-namespaces` action 实现为 signal-only metadata 变更，不得通过该 action 引入额外的编辑审计 annotation。

#### Scenario: 写入 refresh signal
- **When** Server 处理 `refresh-namespaces` action，并向 `Cluster.metadata.annotations` 写入 `testudo.softcdata.com/refresh-cluster-stats=<type>`
- **Then** 本次 action 只允许新增、更新或清理 `testudo.softcdata.com/refresh-cluster-stats`
- **And** 不得在同一次 action 中写入 `testudo.softcdata.com/user`
- **And** 不得在同一次 action 中写入其他仅用于通用 cluster edit 审计的 annotation
- **And** 不得复用会顺带写入上述审计 annotation 的通用 cluster update 路径
