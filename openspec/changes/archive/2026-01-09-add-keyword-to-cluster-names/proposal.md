# 提案：clusterNames 接口支持关键字过滤

## 背景
此前我们已经为 `/disaster/v1/clusters` 接口添加了 `keyword` 参数支持，用于模糊搜索集群名称或标签。
用户反馈 `/disaster/v1/clusters/names` 接口（通常用于下拉列表选择）也需要同样的功能，以便在选项较多时进行快速筛选。

## 目标
1.  使 `/disaster/v1/clusters/names` 接口支持 `keyword` 查询参数。
2.  过滤逻辑应与 `/disaster/v1/clusters` 保持一致：匹配集群名称或 `testudo.softcdata.com/cluster-tag` 标签。

## 变更设计

### 1. Handler 逻辑变更
文件：`internal/apis/disaster_cluster/v1/handler.go`

在 `clusterNames` 方法中，获取到 `items` 后，增加内存过滤逻辑：

```go
    // ... Lister.List(selector) ...

    // 内存关键字过滤
    if qParams.Keyword != "" {
        var matched []*dapisv1.Cluster
        keyword := qParams.Keyword
        for _, item := range items {
            // 匹配名称
            if strings.Contains(item.Name, keyword) {
                matched = append(matched, item)
                continue
            }
            // 匹配 Tag
            if tag, ok := item.Labels[ClusterTagLabel]; ok && strings.Contains(tag, keyword) {
                matched = append(matched, item)
                continue
            }
        }
type DisasterClusterNameDTO struct {
    Name               string `json:"name"`
    ID                 string `json:"id"`
    NamespaceCount     int    `json:"namespaceCount,omitempty"`
    ResourceTotalCount int    `json:"resourceTotalCount,omitempty"`
    Tag                string `json:"tag,omitempty"` // 新增
}
```

## 影响范围
- **API**: `/disaster/v1/clusters/names` 行为变更，支持 `keyword` 参数。
- **Clients**: 前端可以在调用此接口时传递 `keyword` 进行搜索。
