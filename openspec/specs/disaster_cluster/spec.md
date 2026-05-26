# disaster_cluster Specification

## Purpose
待定 - 通过归档变更 `add-all-option-to-lists` 创建。归档后更新 Purpose。

## Requirements
### Requirement: 列出所有集群名称
系统必须 (SHALL) 提供一个 API 接口来获取所有用于选择目的的集群名称列表。

#### Scenario: 获取集群名称
- **WHEN** 客户端请求 `GET /disaster/v1/clusters/names`
- **THEN** 仅返回包含集群 ID 和 Name 的对象，以及必要的统计信息（命名空间数量、资源总数）
- **AND** 列表不需要分页

### Requirement: 集群模糊搜索
系统必须 (SHALL) 提供通过关键字（匹配集群名称或标签）过滤集群的能力。

#### Scenario: 按关键字搜索
- **WHEN** 客户端请求 `GET /disaster/v1/clusters?keyword={value}`
- **THEN** 返回名称包含 `{value}` 或标签包含 `{value}` 的集群列表
- **AND** 过滤操作在获取数据后的内存中进行
