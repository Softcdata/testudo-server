# Design: Cluster Typed Namespace Refresh Action

## 设计目标
- 给前端一个正式 action，而不是“重新请求试试看”
- action 只负责触发指定统计类型的重算，不承诺长时任务编排
- 让前端通过现有 Cluster 读取接口直接看到 workload namespace 刷新结果
- 写入 refresh signal 时遵循现有 API 并发更新标准，避免覆盖无关 metadata 变更
- refresh action 不污染“编辑集群”事件流，不写 `testudo.softcdata.com/user` 等审计注解

## 读取路径
- Cluster 详情、列表、watch DTO 必须透传 `status.workloadNamespaceCount`、`status.workloadNamespaceStats`、`status.workloadTotalCount`
- 现有 `GET /clusters/names` 摘要 DTO 必须透传 `workloadNamespaceCount`、`workloadTotalCount`
- `status.namespaceCount`、`status.namespaceStats`、`status.resourceTotalCount` 与新增 workload 统计字段必须被视为同一套 namespace 级备份统计口径下的读取结果
- 上述字段语义为：
  - `namespaceCount`：纳入统计口径的非系统 namespace 数量
  - `namespaceStats` / `resourceTotalCount`：namespace 级备份资源统计结果
  - `workloadNamespaceCount`：存在 running `Deployment/StatefulSet` 的命名空间数量
  - `workloadNamespaceStats`：这些命名空间各自的 namespace 级备份资源总数
  - `workloadTotalCount`：`workloadNamespaceStats` 的总和
- `202 Accepted` 只表达“server 已接受刷新请求并成功写入 signal”；客户端必须通过上述读取接口观察刷新结果

## 写入路径
- `POST /clusters/:name/actions/refresh-namespaces`
- request body: `{"type":"namespaceStats|workloadNamespaceStats|all"}`
- server 先校验 `type`
- server 使用 `RetryOnConflict` 只写入 `Cluster.metadata.annotations["testudo.softcdata.com/refresh-cluster-stats"]=<type>`
- 每次重试必须重新获取最新 `Cluster`
- 每次重试必须在最新 `metadata.annotations` 基础上写入 refresh signal
- refresh action 不得复用会附带写入 `testudo.softcdata.com/user` 等审计 annotation 的通用 cluster update 路径
- refresh action 完成写入时，不得新增除 `testudo.softcdata.com/refresh-cluster-stats` 之外的 metadata 变更
- server 返回 `202 Accepted`，并回显 `cluster` 与 `type`
- operator 在统计写回成功后清理 signal
- operator 在 `type` 不受支持并被判定为终态失败后清理 signal
- operator 在瞬时错误场景保留 signal，等待后续调谐继续处理
