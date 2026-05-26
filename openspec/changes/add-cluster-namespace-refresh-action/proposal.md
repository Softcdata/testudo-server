# Change: 为集群 API 增加支持 type 的 refresh-namespaces action

## Why
前端当前没有稳定的服务端 action 契约来触发 namespace 重算，只能依赖重新拉详情或列表。这无法保证 operator 真正执行了新的扫描。

此外，刷新动作当前没有“按类型刷新指定统计”的能力，无法单独触发“存在 running `Deployment/StatefulSet` 的命名空间资源统计”刷新。

## What Changes
- 新增 `refresh-namespaces` action route
- action request 显式携带 `type`，支持 `namespaceStats`、`workloadNamespaceStats`、`all`
- server 通过 typed one-shot signal 请求 operator 立即重算指定统计
- refresh action 只允许变更 `Cluster.metadata.annotations["testudo.softcdata.com/refresh-cluster-stats"]`，不得复用会顺带写入 `testudo.softcdata.com/user` 等审计注解的通用集群更新路径
- Cluster 详情、列表、watch DTO 与现有 `GET /clusters/names` 摘要 DTO 必须暴露 workload namespace 统计读取字段，使前端能够读取刷新结果
- Cluster 现有 `namespaceCount/namespaceStats/resourceTotalCount` 与新增 `workloadNamespaceCount/workloadNamespaceStats/workloadTotalCount` 必须统一采用 namespace 级备份统计口径
- 其中 `workloadNamespaceCount` 表示存在 running `Deployment/StatefulSet` 的命名空间数量；`workloadNamespaceStats/workloadTotalCount` 表示这些命名空间内的 namespace 级备份资源总数，而不是 `Deployment/StatefulSet` 对象数
- server 写入 refresh signal 时必须继承现有 API 并发更新标准：`RetryOnConflict`、重取最新对象、在最新 metadata 基础上更新注解
- action 响应需向前端返回已接受的 `type` 与目标 `Cluster`；`202 Accepted` 仅表示请求已被接收，不表示统计已经完成刷新

## Non-Goals
- 不在本 proposal 中定义长期任务状态机
- 不在 server 侧直接计算 cluster stats
- 不新增独立的 cluster names route；只扩展现有 `GET /clusters/names` 摘要 DTO 字段

## Impact
- Affected specs:
  - `disaster_cluster`
  - `api-standards`
- Affected code:
  - `internal/apis/disaster_cluster/v1/router.go`
  - `internal/apis/disaster_cluster/v1/handler*.go`
  - `internal/apis/disaster_cluster/v1/types.go`
