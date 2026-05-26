# Change: 修正实例选择器标签字段并新增容灾组成员列表接口

## Why

1. **修正字段语义**：现有 `GET .../groups/instance-picker` 接口的 `InstancePickerItemDTO` 中，`Labels` 字段返回的是 Kubernetes 原生 Labels（`metadata.labels`），但业务需要的"标签"实为存储在 Annotation `testudo.softcdata.com/description` 中的说明文字（即 Description）。需要将 `labels` 字段替换为 `description` 字段，并同步修正关键词搜索逻辑。

2. **新增接口**：前端编辑容灾组时，需要展示该组**已选定的**容灾实例列表（即 `spec.levels` 中所有实例的摘要），包含名称、说明（description）、命名空间、状态（fsmState）四个字段。现有接口不支持此场景。

## What Changes

### 变更一：修正 `instance-picker` DTO 的 `labels` → `description`

- `InstancePickerItemDTO.Labels map[string]string` → `Description string`
  - 数据来源：`metadata.annotations["testudo.softcdata.com/description"]`
- `instancePicker` handler：构建 DTO 时从 Annotation 读取 Description
- `instancePickerMatchKeyword`：关键词搜索范围改为 `name / namespace / description`（string Contains）

### 变更二：新增 `GET /groups/:name/instances` 接口

- 路径：`GET /groups/:name/instances`
- 功能：读取指定组 `spec.levels` 中的所有实例名称，并发查询各实例的详情，返回轻量 `GroupMemberInstanceDTO`
- 字段：`name`、`description`（来自 Annotation）、`namespace`、`fsmState`
- 支持 `keyword` 模糊搜索（name / namespace / description）
- 支持 `status` 精确过滤

## Impact

- 受影响的规范：`specs/disaster-group/spec.md`（MODIFIED: 实例选择器；ADDED: 组成员列表）
- 受影响的代码：
  - `internal/apis/disaster_group/v1/types.go`（修改 `InstancePickerItemDTO`；新增 `GroupMemberInstanceDTO`）
  - `internal/apis/disaster_group/v1/handler.go`（修改 `instancePicker`；新增 `listGroupInstances`；修改 `instancePickerMatchKeyword`）
  - `internal/apis/disaster_group/v1/router.go`（注册新路由）
