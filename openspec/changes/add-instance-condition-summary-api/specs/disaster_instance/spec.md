## ADDED Requirements

### Requirement: 实例 detail/list 必须输出 condition summary
系统必须 (MUST) 在实例 detail 和 list 接口中输出基于 `status.conditions` 的 condition summary。

#### Scenario: 列表返回高优先级 condition 摘要
- **Given** 一个 `DisasterInstance` 带有高优先级异常 condition
- **When** 客户端查询实例列表
- **Then** Server 必须返回对应的高优先级 condition 摘要
- **And** 不得要求前端自己解析原始 message 才能得出结论
