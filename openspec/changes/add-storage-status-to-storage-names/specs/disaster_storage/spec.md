## MODIFIED Requirements

### Requirement: 列出所有存储名称
系统必须 (SHALL) 提供一个 API 接口来获取所有用于选择目的的存储名称列表。

#### Scenario: 获取存储名称
- **WHEN** 客户端请求 `GET /apis/storage.<group>/<version>/storages/names`
- **THEN** 返回所有存储的对象列表，每项包含 `id`、`name` 以及 `status`
- **AND** `status` 字段取值为 `Available` 以及 `Unavailable`，语义来自 `StorageRepository.status.status`
- **AND** 列表不需要分页

