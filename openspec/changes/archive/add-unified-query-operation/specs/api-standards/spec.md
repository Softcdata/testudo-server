## ADDED Requirements

### Requirement: 查询操作 (Query Operation)
查询操作允许客户端列出集合中的资源。查询操作不得产生副作用。

#### Scenario: 成功查询集合
- **WHEN** 客户端发送 GET 请求到集合 URL (例如 `/v1/folders`)
- **THEN** 返回状态码 200 OK
- **AND** 响应体包含 `type: "collection"`
- **AND** 响应体包含 `resourceType` 字段
- **AND** 响应体包含 `data` 数组

### Requirement: 分页 (Pagination)
可能返回大量结果的集合必须支持分页。

#### Scenario: 使用 Page 分页
- **WHEN** 客户端请求包含 `page={integer}` 参数
- **THEN** 服务端返回指定页码的结果集
- **AND** 响应包含 `pagination` 对象

#### Scenario: 设置每页数量
- **WHEN** 客户端请求包含 `limit={integer}` 参数
- **THEN** 服务端返回不超过指定数量的结果
- **AND** 服务端应强制执行并记录允许的上限（建议至少 1000）
- **AND** 服务端应支持 `limit=0` 以仅获取元数据

#### Scenario: 分页响应结构
- **WHEN** 集合支持分页
- **THEN** 响应必须包含 `pagination` 对象
- **AND** `pagination` 对象包含 `limit` (每页数量), `total` (总数) 和 `partial` (是否截断)
- **AND** `pagination` 对象应包含 `first`, `previous`, `next`, `last` 链接（如果适用）

### Requirement: 排序 (Sorting)
服务应允许客户端请求排序后的资源列表。

#### Scenario: 请求排序
- **WHEN** 客户端请求包含 `sort={sort name}` 和 `order={asc or desc}` 参数
- **THEN** 服务端返回按指定字段和顺序排序的结果
- **AND** 排序必须是“稳定”的（Stable）

#### Scenario: 排序响应结构
- **WHEN** 集合支持排序
- **THEN** 响应应包含 `sortLinks` 对象（可排序字段映射）
- **AND** 如果结果已排序，响应应包含 `sort` 对象（当前排序状态）

### Requirement: 过滤 (Filtering)
服务应支持集合的搜索/过滤。

#### Scenario: 请求过滤
- **WHEN** 客户端请求包含 `{filter name}{"_"+modifier}={value}` 参数
- **THEN** 服务端返回满足所有过滤条件的资源（AND 逻辑）
- **AND** 特殊字符必须进行 URL 编码

#### Scenario: 过滤响应结构
- **WHEN** 集合支持过滤
- **THEN** 响应应包含 `filters` 对象，描述每个字段应用的过滤器

### Requirement: 链接 (Links)
集合响应必须包含链接以支持 HATEOAS (Hypermedia as the Engine of Application State)。

#### Scenario: 包含 Self 链接
- **WHEN** 返回集合响应
- **THEN** `links` 对象必须包含 `self` 字段，指向当前查询的完整 URL（包含查询参数）

#### Scenario: 包含其他相关链接
- **WHEN** 集合有关联的资源（如 API Schema）
- **THEN** `links` 对象应包含相应的链接（如 `schemas`）
- **AND** 链接必须是绝对 URL

#### Scenario: 集合包含资源详情链接
- **WHEN** 返回集合中的资源列表
- **THEN** 集合的 `links` 对象应包含列表中每个资源的详情链接
- **AND** 链接的键应为资源的唯一标识符（如名称）


