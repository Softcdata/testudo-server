# Change: Add Drill Cleanup API

## Why
目前容灾演练已在底层 (Operator) 支持了针对无映射和有映射两种环境的演练清理（Drill Cleanup）。前端控制台需要在用户“手动验证容灾环境完毕后”，提供一个点击“清理资源”触发清理的交互入口。对应的后端需要提供专用的清理 API 供上游调用。

## What Changes
- 在路由层增加 `POST /apis/v1/drills/:name/cleanup` API 节点。
- 对接收到的特定演练（DisasterDrill），向 Kubernetes 更新对应 CRD 的 `spec.cleanup: true` 标识。
- 扩展 `DisasterDrillDTO`，在展示侧兼容返回最新的状态 (`CleaningUp` 与 `CleanedUp`)，并将 `cleanup` flag 暴露给前端（以便前端判断是否处于或者完成过清理流程）。
- **BREAKING**: 无。

## Impact
- Affected specs: `disaster-server/openspec/specs/disaster_drill/spec.md` (新增 Cleanup API 接口说明)
- Affected code:
  - `biz/model/disaster_drill.go` (新增清理请求与更新 DTO 状态)
  - `biz/handler/disaster_drill.go` (新增 `Cleanup` 接口实现)
  - `biz/router/disaster_drill.go` (注册 `cleanup` 路由)
