# Change: Add Scale-to-Zero Support for AppRestore

## Why
在灾难恢复场景（特别是 Warm Standby 或 Pilot Light 模式）中，恢复到目标集群的应用通常不应立即开始处理流量，或者为了节省资源应保持关闭状态。
目前用户可以在恢复后手动缩容，但缺乏自动化的 "恢复即缩容" (Restore-time Scale-to-Zero) 能力。
用户希望提供一种类似于 `SCMapping` 的配置方式，允许指定特定工作负载（Deployment/StatefulSet）在恢复时自动将副本数设置为 0。

## What Changes
- **API 变更**: `CreateAppRestoreRequest` 和 `UpdateAppRestoreRequest` 增加 `ScaleToZeroList` 字段 (类型 `[]string`)。
- **逻辑变更**: 后端在处理该字段时，自动生成对应的 `ResourceModifierRule`，将目标资源的 `.spec.replicas` Patch 为 `0`。
- **范围**: 支持 `Deployments` 和 `StatefulSets`。

## Impact
- **disaster-server**: API 接口和 Handler 逻辑。
- **用户**: 可以通过 API 控制恢复后的副本数。
