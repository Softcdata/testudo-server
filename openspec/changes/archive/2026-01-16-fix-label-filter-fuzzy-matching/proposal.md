# Change: 标签过滤器统一降级为内存模糊搜索

## Why

目前 API 的标签过滤逻辑（如 `?testudo.softcdata.com/app-backup-name=app-`）依赖于 Kubernetes 的 Label Selector，而 Label Selector 仅支持精确匹配。

根据用户反馈，搜索 `app-` 时期望能匹配到所有包含该字符串的资源。目前如果传入非法字符（如通配符 `*`），Selector 会创建失败导致返回全量数据；如果不传通配符，则只能精确匹配。

为了提升交互体验，API 标准调整为：**所有基于查询参数的过滤操作一律支持模糊匹配（包含匹配）**。

## What Changes

- **标准变更**: 不再要求客户端传递 `*` 号，服务端默认对所有过滤条件执行 `strings.Contains` 匹配。
- **技术实现**: 
    - `transport.BuildLabelSelector` 不再负责这些需要模糊匹配的字段（或作为初步筛选）。
    - 服务端在获取列表后，统一在内存中进行二次过滤。
- **性能优化**: 针对小规模资源（<1000个），内存过滤的延迟完全可控且逻辑更灵活。

## Impact

- **受影响文件**: 
    - `internal/transport/query.go` (匹配算法更新)
    - `internal/apis/*/v1/handler.go` (所有列表接口)
- **受影响规范**: 
    - `api-standards` (更新过滤章节，明确“一律模糊匹配”原则)
