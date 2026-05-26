# Informer/Lister 内存分页与排序实现指南

## 1. 概述
本文档描述了 `disaster-server` 如何利用 Kubernetes Informer/Lister 机制结合内存处理来实现高效的资源列表查询、排序和分页。

目前所有的 List 接口（如 `AppBackup`, `DisasterCluster` 等）均采用此方案，而非直接透传请求给 Kubernetes API Server。

## 2. 架构设计

核心思想是 **"缓存读取 + 内存计算"**。

### 数据流向
1.  **Informer Sync (后台)**: 
    *   Server 启动时，SharedInformerFactory 创建 Informer。
    *   Informer 与 K8s API Server 建立长连接 (Watch)，将资源实时同步到本地内存缓存 (Store/Indexer)。
2.  **Client Request (前台)**:
    *   客户端发起 HTTP GET 请求，携带查询参数 (e.g., `?page=1&limit=10&sort=creationTimestamp`).
3.  **Handler Processing**:
    *   **Step 1: Lister Fetch**: Handler 通过 `Lister` 接口从本地缓存中获取数据。支持 LabelSelector 过滤。
    *   **Step 2: Memory Filter**: (可选) 对 Lister 返回的数据进行非 Label 字段的二次过滤。
    *   **Step 3: Memory Sort**: 使用 `internal/pkg/query` 对全量结果进行稳定排序。
    *   **Step 4: Memory Paginate**: 根据分页参数对排序后的切片进行截取 (Slicing)。
4.  **Response**:
    *   返回分页后的数据子集以及总条数 (`total`) 等元数据。

## 3. 详细实现步骤

### 3.1 数据获取 (Lister)
使用 `client-go` 生成的 Lister 接口。这是最高效的步骤，因为它只是从内存 Map 中查找。

```go
// 1. 解析查询参数
qParams := query.ParseOptions(c, ctx)
selector := query.BuildLabelSelector(qParams)

// 2. 从缓存获取 (只支持 Label 过滤)
filteredItems, err := h.AppBackupLister.AppBackups(namespace).List(selector)
```

### 3.2 内存排序 (In-Memory Sorting)
由于 Lister 返回的列表顺序是不确定的（取决于 Map 遍历顺序），必须在分页前进行**稳定排序**。

我们使用 `internal/pkg/query.Sort` 通用函数：

```go
sortedItems := query.Sort(filteredItems, qParams, func(a, b *dapisv1.AppBackup, field string) int {
    switch field {
    case "name":
        return strings.Compare(a.Name, b.Name)
    case "creationTimestamp":
        // 时间比较逻辑
        if a.CreationTimestamp.Before(&b.CreationTimestamp) { return -1 }
        return 1
    default:
        return 0
    }
})
```

### 3.3 内存分页 (In-Memory Pagination)
在全量有序列表的基础上，计算切片的 Start 和 End 索引。

```go
// items[offset : offset+limit]
pagedItems, total := query.Paginate(sortedItems, qParams)
```

## 4. 方案优缺点分析

### 优点 (Pros)
1.  **高性能 (High Performance)**: 
    *   读取操作完全在内存中完成，无网络 I/O 开销。
    *   响应速度通常在微秒/毫秒级。
2.  **保护 API Server (Protect API Server)**: 
    *   避免了客户端频繁的 List 请求直接打到 K8s API Server（K8s List 请求如果不带 ResourceVersion="0" 且数据量大，开销很高）。
    *   Informer 只需要一个 Watch 连接即可维护数据。
3.  **功能灵活 (Flexibility)**: 
    *   K8s API Server 的原生排序和过滤能力非常有限（仅支持部分 FieldSelector）。
    *   内存实现允许我们支持任意字段的排序、复杂的组合过滤逻辑。

### 缺点 (Cons)
1.  **内存消耗 (Memory Usage)**: 
    *   所有资源对象都驻留在 Server 内存中。
    *   *缓解*: 对于 CRD 资源，通常数量级在几千到几万，内存占用可控。如果是 Pod/Event 等海量资源需谨慎。
2.  **最终一致性 (Eventual Consistency)**: 
    *   Informer 缓存与 API Server 之间存在微小的同步延迟。
    *   *影响*: 刚创建的资源可能无法立即在 List 接口中查到（通常延迟 < 100ms）。
3.  **CPU 开销 (CPU Overhead)**: 
    *   每次请求都需要对全量数据进行排序。如果单次 List 返回 10万条数据，排序会消耗 CPU。
    *   *缓解*: Go 的排序算法效率很高，且通常配合 LabelSelector 先过滤掉大部分数据。

## 5. 最佳实践
1.  **优先使用 LabelSelector**: 在调用 Lister 时尽量传入 LabelSelector，利用 Indexer 索引减少初次返回的数据量。
2.  **限制最大 Limit**: 防止客户端请求 `limit=100000` 导致的大包传输。
3.  **只读操作**: 此方案仅适用于 `List` / `Get` (Get 也可以走 Lister)。`Create`/`Update`/`Delete` 必须直接调用 KubeClient 直连 API Server。
