# Change: 为实例 API 增加 condition summary 聚合契约

## Why
随着条目 23 引入正式的 role drift condition，server 需要稳定把 `status.conditions` 聚合成 detail/list 可消费的摘要，否则前端仍会回退到各接口拼 message 的旧模式。

## What Changes
- 为实例 detail/list 增加 condition summary 聚合
- 聚合结果优先消费 operator condition，而不是自行拼接 message

## Non-Goals
- 不新增第二套状态源
- 不在 server 侧自行判定 role drift

## Impact
- Affected specs:
  - `disaster_instance`
- Affected code:
  - `internal/apis/disaster_instance/v1/*`
