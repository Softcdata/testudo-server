## MODIFIED Requirements
### Requirement: WatchEventDTO 结构
系统必须 (MUST) 保证 Watch 事件流与历史事件列表返回一致的 `TaskEvent` 关键字段语义。

#### Scenario: TaskEvent DTO 完整性
- **WHEN** 返回 `TaskEvent` 对象
- **THEN** 必须包含 `reason` 字段，对应 Kubernetes Event 的 Reason
- **AND** `taskName` 必须优先使用 JSON 消息体中的 `task` 字段，若不存在则回退到资源名称
