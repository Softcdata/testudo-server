# disaster_policy Specification

## Purpose
待定 - 通过归档变更 `add-all-option-to-lists` 创建。归档后更新 Purpose。

## Requirements
### Requirement: 列出所有策略名称
系统必须 (SHALL) 提供一个 API 接口来获取所有用于选择目的的备份策略名称列表。

#### Scenario: 获取策略名称
- **WHEN** 客户端请求 `GET /disaster/v1/policies/names`
- **THEN** 返回所有策略的对象列表，仅包含 `name` 字段
- **AND** 列表不需要分页
