## ADDED Requirements

### Requirement: 容灾组列表汇总统计

系统 MUST 在 `GET /apis/disastergroups.testudo.softcdata.com/v1/groups` 的响应 `meta.summary` 中返回容灾组列表汇总统计。统计范围 MUST 是按 `keyword` 与 `status` 过滤后、分页前的容灾组集合。

`meta.summary.instanceCount` MUST 统计该集合中所有 `instances[]` 条目数量。

`meta.summary.abnormalCount` MUST 统计该集合中的异常容灾组数量，每个容灾组最多计数一次。异常容灾组定义为组自身健康异常，判定条件如下：

- `DisasterGroup.status.reason` 非空，例如 `InstanceNotFound`、`InstanceFailed`
- `DisasterGroup.status.conditions` 中存在 `type=Error` 且 `status=True`
- server 推导的组展示态 `status.fsmState` 为 `Degraded`
- 组内成员实例展示状态为 `Failed`、`ConfigError`、`NotFound`
- 组内成员实例存在非空 `reason`

`FailingOver`、`FailingBack` 表示切换动作进行中。仅存在 `FailingOver`、`FailingBack` 成员状态，且不满足以上异常条件时，该容灾组 MUST NOT 计入 `meta.summary.abnormalCount`。

#### Scenario: 返回实例总数和异常容灾组个数
- **GIVEN** 过滤后、分页前共有 3 个容灾组
- **AND** 三个容灾组的成员实例数量分别为 2、3、1
- **AND** 其中 2 个容灾组满足异常容灾组判定条件
- **WHEN** 客户端请求 `GET /apis/disastergroups.testudo.softcdata.com/v1/groups?page=1&limit=1`
- **THEN** 响应 `meta.summary.instanceCount` 必须为 `6`
- **AND** 响应 `meta.summary.abnormalCount` 必须为 `2`
- **AND** 统计结果不得受当前页只返回 1 个容灾组影响

#### Scenario: 操作中容灾组不直接计入异常
- **GIVEN** 过滤后、分页前存在 1 个容灾组
- **AND** 该容灾组内存在 `FailingOver` 成员实例
- **AND** 该容灾组 `status.reason` 为空
- **AND** 该容灾组没有 `type=Error` 且 `status=True` 的 condition
- **AND** 该容灾组内不存在 `Failed`、`ConfigError`、`NotFound` 成员实例
- **WHEN** 客户端请求 `GET /apis/disastergroups.testudo.softcdata.com/v1/groups`
- **THEN** 响应 `meta.summary.abnormalCount` 必须为 `0`

#### Scenario: 组级错误原因计入异常
- **GIVEN** 过滤后、分页前存在 1 个容灾组
- **AND** 该容灾组 `status.reason` 为 `InstanceNotFound`
- **WHEN** 客户端请求 `GET /apis/disastergroups.testudo.softcdata.com/v1/groups`
- **THEN** 响应 `meta.summary.abnormalCount` 必须为 `1`
