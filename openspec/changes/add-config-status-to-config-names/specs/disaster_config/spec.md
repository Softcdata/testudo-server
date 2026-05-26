## ADDED Requirements

### Requirement: 列出所有容灾配置名称（含状态）
系统必须 (SHALL) 提供一个 API 接口来获取所有用于选择目的的容灾配置名称列表，并返回每个配置的状态。

#### Scenario: 获取容灾配置名称列表
- **WHEN** 客户端请求 `GET /apis/disasterconfigs.testudo.softcdata.com/v1/configs/names`
- **THEN** 返回所有容灾配置的对象列表，每项包含 `id`、`name` 以及 `status`
- **AND** `status` 字段取值为 `Pending` / `Ready` / `NotReady` / `Error`，语义来自 `DisasterConfig.status.status`
- **AND** 列表不需要分页

