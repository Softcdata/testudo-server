## ADDED Requirements

### Requirement: 过滤器的容错与模糊匹配降级 (Filter Fault Tolerance & Fuzzy Matching)

API 必须 (MUST) 具备对非法过滤值的容错处理能力。当请求的过滤值包含通配符或因格式不规范无法直接使用 Kubernetes Label Selector 处理时，服务端应降级为内存中的模糊匹配，而不是静默丢弃过滤器。

#### Scenario: 非法标签值自动转为模糊匹配
- **GIVEN** 客户端发送包含 `key=val*` 的过滤请求
- **AND** `val*` 不是合法的 Kubernetes 标签值（包含 `*`）
- **WHEN** 服务端处理请求
- **THEN** 服务端不应报错，也不应丢弃该过滤条件
- **AND** 服务端应在获取初步结果后，在内存中对 `key` 标签进行前缀匹配匹配（`val` 开头）
- **AND** 最终返回的结果必须满足该匹配条件
