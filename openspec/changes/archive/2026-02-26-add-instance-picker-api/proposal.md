# Change: 为容灾组添加实例选择器 API

## Why

前端在创建/编辑容灾组（DisasterGroup）时，需要从现有容灾实例（DisasterInstance）列表中选择成员。现有的实例列表接口（`/apis/disasterinstances.testudo.softcdata.com/v1/instances`）返回的数据结构过于重，包含大量前端选择器不需要的字段（如 Config 详情、Storage 详情、SyncStatus 等），且前端无法方便地基于实例状态筛选或按关键词检索。需要一个专为"选择器场景"设计的轻量级接口。

## What Changes

- 在容灾组 Handler（`internal/apis/disaster_group/v1/`）下新增 `GET /groups/instance-picker` 接口
- 新增 `InstancePickerItemDTO` 轻量 DTO，仅包含：实例名称、命名空间、Labels、FsmState
- 支持 `keyword` 查询参数：对名称、命名空间、Labels 值进行模糊搜索（Contains，不区分大小写）
- 支持 `status` 查询参数：按 FsmState 精确过滤（支持多值逗号分隔）
- 支持标准分页参数 `page` / `limit`
- 遵循统一响应信封（`transport.WriteSuccess` + `transport.BuildCollectionResponse`）

## Impact

- 受影响的规范：`specs/disaster_group`（新建）、`specs/api-standards`（使用现有标准）
- 受影响的代码：
  - `internal/apis/disaster_group/v1/handler.go`（新增 handler 方法）
  - `internal/apis/disaster_group/v1/router.go`（注册新路由）
  - `internal/apis/disaster_group/v1/types.go`（新增 DTO）
