# Change: 新增容灾配置名称列表接口并返回状态

## Why
当前前端在创建或编辑其他资源时，需要从下拉框选择一个 `DisasterConfig`（容灾配置）。
如果仅返回名称列表而不包含配置状态，用户无法区分配置是否处于可用状态（例如 `Ready`）或异常状态（例如 `Error`、`NotReady`），容易在后续提交或执行操作时才发现失败。
由于 `disaster-operator` 已在 `DisasterConfig.status.status` 中维护状态信息，`disaster-server` 可以直接复用该状态，以提升选择器的可用性与可观测性。

## What Changes
- 新增接口：`GET /apis/disasterconfigs.testudo.softcdata.com/v1/configs/names`
- 返回数据为对象数组，每项包含：
  - `id`：对应 `metadata.uid`
  - `name`：对应 `metadata.name`
  - `status`：对应 `status.status`（枚举：`Pending` / `Ready` / `NotReady` / `Error`）

## Impact
- 受影响规范：
  - 新增增量规范 `specs/disaster_config/spec.md`
- 受影响代码：
  - `internal/apis/disaster_config/v1/router.go`
  - `internal/apis/disaster_config/v1/handler.go`
  - `internal/apis/disaster_config/v1/types.go`
  - `internal/apis/disaster_config/v1/handler_test.go`
