# Change: Add Standby Workload Support

## Why
在恢复有状态应用（特别是使用 Velero 文件系统备份）时，直接 Scale-to-Zero 会导致 Pod 无法启动，从而使数据恢复失败。
为了解决这个问题，同时满足“恢复后暂不对外提供服务”的需求，我们需要一种 "Pseudo Scale-to-Zero" (伪缩容) 机制。
该机制通过将工作负载的主容器镜像替换为极简镜像（如 `busybox`）并执行休眠命令（`sleep infinity`），使 Pod 保持运行状态（以挂载 PVC 供 Velero 注入数据），但不执行实际业务逻辑。

## What Changes
- **API 变更**: `CreateAppRestoreRequest` 和 `UpdateAppRestoreRequest` 增加 `StandbyList` 字段 (类型 `[]string`)。
- **逻辑变更**: 后端自动生成 `ResourceModifierRule`：
    - 匹配 `Deployments` 和 `StatefulSets`。
    - 将所有容器的 `image` 替换为 `busybox:latest` (可配置)。
    - 将所有容器的 `command` 替换为 `["/bin/sh", "-c", "sleep infinity"]`。
    - 移除 `readinessProbe`, `livenessProbe`, `startupProbe` 防止反复重启。
- **范围**: 用于 `DataSync` 场景下的有状态应用。

## Impact
- **disaster-server**: 支持新的 Standby 模式。
- **disaster-operator**: V2 DataSync 将默认使用此模式。
