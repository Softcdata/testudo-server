# 任务：实现集群模糊搜索

## 1. 实现 (Implementation)
- [x] 1.1 **更新 Transport**: 在 `internal/transport/query.go` 的 `Options` 中添加 `Keyword` 字段，并在 `ParseOptions` 中绑定。
- [x] 1.2 **更新 Cluster Handler**: 修改 `internal/apis/disaster_cluster/v1/handler.go` 的 `clusters` 方法，实现关键词过滤逻辑。
- [x] 1.3 **更新 OpenSpec**: 记录 API 变更。
