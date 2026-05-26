# Change: 为存储名称列表增加存储状态

## Why
当前 `GET /apis/storage.<group>/<version>/storages/names` 用于前端下拉选择存储仓库。
该接口返回字段缺少每个存储仓库对应存储桶的可用性状态，前端无法区分 `Available` 以及 `Unavailable`，只能在提交表单时再触发失败校验，用户体验较差。
`StorageRepository.status.status` 已由 `disaster-operator` 写入并更新，该状态可以直接复用。

## What Changes
- 为存储名称列表项新增 `status` 字段，取值来自 `StorageRepository.status.status`。
- 保留现有字段 `name` 以及 `id`，仅新增字段，不删除字段。

## Impact
- 受影响规范：
  - `openspec/specs/disaster_storage/spec.md`
- 受影响代码：
  - `internal/apis/disaster_storage/v1/types.go`
  - `internal/apis/disaster_storage/v1/handler.go`
  - `internal/apis/disaster_storage/v1/handler_test.go`
