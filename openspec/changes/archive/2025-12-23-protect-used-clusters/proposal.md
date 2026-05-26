# Proposal: 集群删除保护 (Server Side)

## Summary
在服务端 API 层拦截对已被使用集群的删除请求。

## Motivation
配合 Operator 的 Finalizer 保护机制，服务端应在用户发起删除请求时立即进行检查，提供快速反馈，而不是让用户等待 Operator 的异步处理结果或面对一直处于 "Terminating" 状态的资源。

## Proposed Changes

### 1. 依赖检查服务
- 在 `internal/apis/disaster_cluster` 模块中增加依赖检查逻辑。
- 需要查询 Kubernetes API (通过 `controller-runtime` client 或 `client-go`) 来检查是否存在引用该集群的 CR。
- **使用 Label Selector 优化查询**:
    - `AppBackup`: `testudo.softcdata.com/app-backup-cluster=<cluster-name>`
    - `AppRestore`: `testudo.softcdata.com/app-restore-cluster=<cluster-name>`
    - `DisasterConfig`: 遍历检查 `spec.sourceCluster` / `spec.targetCluster` (或建议添加 Label)。
    - `StorageRepository`: 检查是否存在关联配置。

### 2. Delete Handler 修改
- 在处理 `DELETE /clusters/:name` 请求时：
    1. 调用依赖检查服务。
    2. 如果发现引用，立即返回 HTTP 400 错误，并在响应消息中列出具体的引用资源（例如："Cluster is used by AppBackup: backup-1"）。
    3. 如果无引用，继续执行删除操作。

## Tasks
- [ ] 实现 `CheckClusterDependencies` 函数。
- [ ] 集成到 `DeleteCluster` Handler。
