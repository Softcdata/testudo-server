# 提案：增强 clusterNames 接口返回统计信息

## 背景
当前的 `GET /disaster/v1/clusters/names` 接口仅返回集群名称和 ID。前端在展示集群下拉列表时，希望能够显示更多的上下文信息，特别是集群的命名空间数量和资源总数，以帮助用户更直观地识别集群规模。

用户反馈提到 "显示他标签中的命名空间数和总资源数"，经核查 `DisasterCluster` CRD 定义，`NamespaceCount` 和 `ResourceTotalCount` 实际上存储在 `Status` 字段中。本提案将把这些字段暴露在 `clusterNames` 的轻量级响应中。

## 目标
1. 修改 `DisasterClusterNameDTO` 结构体，增加统计字段。
2. 更新 `clusterNames` 接口逻辑，填充这些字段。

## 变更设计

### 1. API DTO 变更
文件：`internal/apis/disaster_cluster/v1/types.go`

```go
type DisasterClusterNameDTO struct {
    Name           string `json:"name"`
    ID             string `json:"id"`
    NamespaceCount int    `json:"namespaceCount"`       // 新增
    ResourceCount  int    `json:"resourceTotalCount"`   // 新增
}
```

### 2. Handler 逻辑变更
文件：`internal/apis/disaster_cluster/v1/handler.go`

在 `clusterNames` 方法中，在遍历 Cluster 列表转换为 DTO 时，从 `item.Status` 中读取：
- `NamespaceCount`
- `ResourceTotalCount`

并赋值给 DTO。

## 影响范围
- **API**: `/disaster/v1/clusters/names` 响应结构发生变化（向下兼容，仅增加字段）。
- **Clients**: 前端需适配新字段以展示。
