# Change: 统一查询操作封装 (Unified Query Operation)

## Why
目前 `disaster-server` 中的列表查询接口（如 `DisasterCluster`, `DisasterBackup`）存在大量重复代码，且响应格式不统一。此外，用户需要获取资源总数 (Total Count) 并支持随机页码访问 (Page/Limit)，而 Kubernetes API 原生的 `continue` token 分页机制无法直接满足这些需求。因此，需要引入基于 Informer 的内存分页机制。

## What Changes
- 创建 `internal/pkg/query` 包，提供通用的查询参数解析和响应构建工具。
- 定义标准的 `BaseQuery` 结构体用于解析 `limit`, `page`, `sort`, `order` 以及动态过滤参数。
- 定义标准的 `CollectionResponse` 结构体，确保所有列表接口返回一致的 JSON 结构（包含 `data`, `pagination`, `links`, `sort`, `filters`）。
- 实现 `ParseOptions` 函数，解析 HTTP 查询参数。
- 实现 `BuildLabelSelector` 函数，将过滤参数转换为 Kubernetes Label Selector。
- 实现内存中的 `Sort` 和 `Paginate` 辅助函数。
- 实现 `BuildCollectionResponse` 函数，自动生成分页链接和 HATEOAS 资源链接。
- 将数据获取方式从直接 API 调用迁移到 `Informer/Lister`，以支持高效的内存分页和总数统计。

## Impact
- **受影响的规范**: 新增 `api-standards` 功能规范。
- **受影响的代码**: 
  - 新增 `internal/pkg/query`。
  - 修改 `internal/apis/` 下的所有 List Handler（逐步迁移）。
