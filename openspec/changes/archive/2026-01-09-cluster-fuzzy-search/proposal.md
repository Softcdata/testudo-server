# 变更：集群列表支持模糊搜索

## 为什么 (Why)
前端界面在集群列表页提供了一个统一的搜索框，用户希望输入关键字后，能够同时匹配“集群名称”和“集群标签（Tag）”。目前的过滤机制仅支持基于 Label 的精确匹配，无法满足模糊搜索和多字段匹配的需求。

## 变更内容 (What Changes)
- **Transport 层**: 更新 `internal/transport/query.go` 中的 `Options` 结构体，增加 `Keyword` 字段，并更新 `ParseOptions` 以支持从 URL 参数 `keyword` 中解析该值。
- **Cluster API**: 更新 `internal/apis/disaster_cluster/v1/handler.go` 中的 `clusters` 方法：
  - 检查 `Options.Keyword` 是否存在。
  - 如果存在，则获取所有集群后，在内存中进行过滤。
  - 匹配逻辑：`strings.Contains(cluster.Name, keyword)` || `strings.Contains(cluster.Labels[ClusterTagLabel], keyword)`。
  - 过滤后的结果再进行排序和分页。

## 影响 (Impact)
- **API 行为**: `GET /disaster/v1/clusters?keyword=xxx` 将返回名称或标签包含 `xxx` 的集群。
- **性能**: 涉及获取所有集群并在内存中过滤。考虑到集群数量级（通常在百以内），内存过滤开销可以忽略不计。
