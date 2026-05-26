# Change: Failover 操作透传 skipScaleDownSource 参数

## Why

当前 server 在实例/组操作接口中仅透传 `force`、`skipFinalSync`、`timeoutMinutes` 等参数，未透传 failover 新增能力 `skipScaleDownSource`。

这会导致前端即使传入了 `skipScaleDownSource=true`，Operator 侧仍收不到参数，无法触发“跳过源集群缩零”行为。

## What Changes

- 在实例操作接口 `POST /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/actions` 中支持解析并透传 `config.skipScaleDownSource`。
- 在容灾组操作接口 `POST /apis/disastergroups.testudo.softcdata.com/v1/groups/:name/actions` 中支持解析并透传 `config.skipScaleDownSource`。
- 兼容首字母大写键名 `SkipScaleDownSource`。
- 限定该参数仅在 `operation=failover` 时写入 `DisasterOperationSpec`，其它操作类型保持现状。
- 为兼容 server 与 operator 版本不一致场景，同步写入 annotation `testudo.softcdata.com/skip-scale-down-source=true` 作为兜底信号。
- 增加单元测试覆盖实例与组两条透传路径。

## Impact

- Affected specs: `api-standards`
- Affected code:
  - `internal/apis/disaster_instance/v1/handler_action.go`
  - `internal/apis/disaster_group/v1/handler.go`
  - `internal/apis/disaster_instance/v1/handler_test.go`
  - `internal/apis/disaster_group/v1/handler_test.go`
