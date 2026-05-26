# API 规范变更：历史事件列表

## ADDED Requirements

### 事件列表访问
#### Scenario: 用户请求获取全局历史事件列表
- **Requirement**: 系统必须提供 `GET /apis/v1/events/list` 接口。
- **Requirement**: 接口必须支持分页参数 `page` 和 `pageSize`。
- **Requirement**: 接口必须支持按 `taskType` (backup, restore...) 筛选。
- **Requirement**: 接口必须支持按 `status` (success, failed, processing) 筛选。
- **Requirement**: 响应内容必须包含耗时 (duration) 和 触发人 (triggeredBy) 字段。

#### Scenario: 用户请求获取特定资源的历史事件列表
- **Requirement**: 系统必须提供 `GET /apis/v1/:resource/:name/events/list` 接口。
- **Requirement**: 接口响应应仅包含与该资源关联的事件。
