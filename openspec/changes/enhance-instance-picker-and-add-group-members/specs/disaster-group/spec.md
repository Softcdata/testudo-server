## MODIFIED Requirements

### Requirement: 容灾实例选择器接口 (Instance Picker API)

系统 SHALL 将 `InstancePickerItemDTO` 中原有的 `labels`（`metadata.labels` map 类型）字段替换为 `description`（字符串类型），数据来源改为 `metadata.annotations["testudo.softcdata.com/description"]`，以正确返回业务说明文字而非 Kubernetes 原生标签映射。

该接口 SHALL 返回所有 `DisasterInstance` 资源的简要信息，包含以下字段：

| 字段 | 来源 | 说明 |
|------|------|------|
| `name` | `metadata.name` | 实例名称 |
| `namespace` | `metadata.namespace` | 实例所在命名空间 |
| `description` | `metadata.annotations["testudo.softcdata.com/description"]` | 说明标签（业务描述） |
| `fsmState` | `status.fsmState` | 当前状态 |

#### Scenario: 返回 description 而非 labels

- **WHEN** 客户端请求 `GET /apis/disastergroups.testudo.softcdata.com/v1/groups/instance-picker`
- **THEN** 响应中每个 `InstancePickerItemDTO` 包含 `description` 字段（字符串）
- **AND** `description` 的值来自实例的 `metadata.annotations["testudo.softcdata.com/description"]`
- **AND** 响应中不再包含 `labels` 字段

#### Scenario: 关键词搜索范围修正

- **WHEN** 客户端请求携带 `keyword=nginx`
- **THEN** 服务端在内存中对每个实例执行 Contains 匹配（不区分大小写），匹配范围为：
  - `metadata.name`
  - `metadata.namespace`
  - `metadata.annotations["testudo.softcdata.com/description"]`
- **AND** 不再对 `metadata.labels` 的值进行匹配

---

## ADDED Requirements

### Requirement: 容灾组已选实例列表接口 (Group Members API)

系统 SHALL 在容灾组路由组下提供 `GET /groups/:name/instances` 接口，返回指定容灾组（`DisasterGroup`）当前 `spec.levels` 中所有已选定的容灾实例摘要，供前端编辑容灾组时展示已选成员。

该接口 SHALL 返回 `GroupMemberInstanceDTO` 数组，仅包含：

| 字段 | 来源 | 说明 |
|------|------|------|
| `name` | `metadata.name` | 实例名称 |
| `description` | `metadata.annotations["testudo.softcdata.com/description"]` | 说明标签 |
| `namespace` | `metadata.namespace` | 实例所在命名空间 |
| `fsmState` | `status.fsmState` | 当前状态 |

#### Scenario: 返回容灾组所有已选实例

- **WHEN** 客户端 `GET /apis/disastergroups.testudo.softcdata.com/v1/groups/my-group/instances`
- **THEN** 服务端读取 `DisasterGroup.spec.levels`（二维数组，每项为实例名称列表）
- **AND** 并发查询各实例的 `DisasterInstance` 资源
- **AND** 返回所有实例的 `GroupMemberInstanceDTO` 数组（展平 levels，不含层级结构）
- **AND** `data.type` 为 `"collection"`，`data.resourceType` 为 `"groupMemberInstance"`

#### Scenario: 容灾组不存在

- **WHEN** 指定的 `:name` 对应的 `DisasterGroup` 不存在
- **THEN** 返回 HTTP 404，`code` 为 3004（CodeNotFound）

#### Scenario: 按关键词搜索

- **WHEN** 请求携带 `keyword=app`
- **THEN** 服务端对返回结果进行内存过滤，匹配范围：`name` / `namespace` / `description`（Contains，不区分大小写）

#### Scenario: 按状态过滤

- **WHEN** 请求携带 `status=Protected`
- **THEN** 仅返回 `fsmState == "Protected"` 的实例

#### Scenario: 标准分页

- **WHEN** 请求携带 `page=1&limit=10`
- **THEN** 响应 `meta.pagination` 包含 `limit`、`total`、`page` 字段
