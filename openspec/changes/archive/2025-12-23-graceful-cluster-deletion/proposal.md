# Proposal: 集群删除选项 (Server Side)

## Summary
更新删除集群的 API，支持传递 `uninstall_velero` 选项，以便用户决定是否清理目标集群上的 Velero。

## Motivation
配合 Operator 的优雅删除逻辑，服务端 API 需要提供接口让用户表达“卸载 Velero”的意图。

## Proposed Changes

### 1. API 变更
- **Endpoint**: `DELETE /clusters/:name`
- **Parameters**:
    - Query Param: `uninstall_velero` (boolean, default: false)
    - 或者 Request Body (如果 RESTful 风格允许 DELETE 带 Body，或者改为 POST /delete-actions)
    - *建议使用 Query Param*。

### 2. Handler 逻辑 (`DeleteCluster`)
- 解析 `uninstall_velero` 参数。
- 如果为 `true`：
    1. 获取 Cluster CR。
    2. Patch Cluster CR，添加 Annotation `testudo.softcdata.com/uninstall-velero: "true"`。
- 执行删除操作 (`client.Delete`)。

## Tasks
- [ ] 更新 `DeleteCluster` Handler 以支持 `uninstall_velero` 参数。
- [ ] 实现添加 Annotation 的逻辑。
