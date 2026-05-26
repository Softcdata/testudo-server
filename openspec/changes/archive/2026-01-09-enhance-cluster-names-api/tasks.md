# 任务：增强 clusterNames 接口

## 1. 接口定义 (Type Definition)
- [x] 1.1 **Update DTO**: 修改 `internal/apis/disaster_cluster/v1/types.go` 中的 `DisasterClusterNameDTO`，添加 `NamespaceCount` and `ResourceTotalCount`。

## 2. 逻辑实现 (Logic Implementation)
- [x] 2.1 **Update Handler**: 修改 `internal/apis/disaster_cluster/v1/handler.go` 中的 `clusterNames` 方法，将 `Cluster.Status` 中的统计数据填充到 DTO 中。

## 3. 文档与验证 (Docs & Validation)
- [ ] 3.1 **Update OpenSpec**: 更新相关的 API 规范文档。
- [ ] 3.2 **Manual Verification**: 启动 Server，调用接口验证返回字段。
