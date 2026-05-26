## ADDED Requirements

### Requirement: 容灾实例选择器接口 (Instance Picker API)

系统 SHALL 在容灾组路由组下提供一个专用的轻量级实例选择器接口，供前端构建"选择容灾实例"UI 组件时使用。

该接口 SHALL 返回所有 `DisasterInstance` 资源的简要信息，仅包含：

| 字段 | 来源 | 说明 |
|------|------|------|
| `name` | `metadata.name` | 实例名称 |
| `namespace` | `metadata.namespace` | 实例所在命名空间 |
| `labels` | `metadata.labels` | 实例标签（map） |
| `fsmState` | `status.fsmState` | 当前状态（如 `Protected`、`Paused`、`Failed` 等） |

接口 SHALL 不返回 Config 详情、Storage 详情、DataSyncStatus、ResourceSyncStatus 等重量级字段。

#### Scenario: 无过滤条件时返回所有实例简要列表

- **WHEN** 客户端 `GET /apis/disastergroups.testudo.softcdata.com/v1/groups/instance-picker` 且不带任何查询参数
- **THEN** 返回 HTTP 200
- **AND** 响应信封 `code` 为 0
- **AND** `data.type` 为 `"collection"`
- **AND** `data.data` 为 `InstancePickerItemDTO` 数组，每项仅包含 `name`、`namespace`、`labels`、`fsmState` 四个字段

#### Scenario: 按关键词搜索实例

- **WHEN** 客户端请求携带 `keyword=nginx`
- **THEN** 服务端在内存中对每个实例执行 Contains 匹配（不区分大小写），匹配范围包括：
  - `metadata.name`
  - `metadata.namespace`
  - `metadata.labels` 的所有 **值**（value）
- **AND** 仅返回至少匹配其中一个字段的实例
- **AND** 未匹配实例不出现在结果中

#### Scenario: 按状态过滤实例

- **WHEN** 客户端请求携带 `status=Protected`
- **THEN** 服务端仅返回 `fsmState == "Protected"` 的实例
- **AND** 其他状态的实例不出现在结果中

#### Scenario: 同时使用关键词与状态过滤

- **WHEN** 客户端请求携带 `keyword=app&status=Protected`
- **THEN** 服务端先按关键词 Contains 过滤，再按状态精确过滤（AND 语义）
- **AND** 仅返回同时满足两个条件的实例

#### Scenario: 标准分页

- **WHEN** 客户端请求携带 `page=1&limit=10`
- **THEN** 响应 `meta.pagination` 包含 `limit`、`total`、`page` 字段
- **AND** `data.data` 数组长度不超过 `limit` 值
