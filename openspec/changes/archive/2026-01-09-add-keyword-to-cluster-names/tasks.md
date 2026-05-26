## 1. 逻辑实现 (Implementation)
- [x] 1.1 **Update Handler**: 修改 `internal/apis/disaster_cluster/v1/handler.go` 中的 `clusterNames` 方法，添加内存关键字过滤逻辑（复用或参考 `clusters` 方法的逻辑）。
- [x] 1.2 **Update DTO**: 修改 `DisasterClusterNameDTO` 添加 `Tag` 字段，并在 Handler 中赋值。

## 2. 验证 (Validation)
- [x] 2.1 **Manual Verification**: 启动 Server，调用 `/disaster/v1/clusters/names?keyword=xxx` 验证过滤效果。
