# Change: syncPolicy 接口收敛到双字段契约

## Why
最终产品口径已经确定：
- `DisasterConfig` 对外接口继续使用双字段表达基础配置策略
- `DisasterInstance` 对外接口继续使用双字段表达实例 override
- operator 与 CRD 继续保持原有双字段模型，不新增统一 `syncPolicy` 字段

此前提案中“统一 `syncPolicy` 输入/回显”的方向与最终口径冲突，必须在 server 契约层收敛回双字段，避免 server、operator、RunAPI 与前端理解继续分叉。

## What Changes
- `DisasterConfig` create/update/detail/list 继续使用 `dataSyncPolicy` 与资源同步双字段契约，不新增对外 `syncPolicy`
- `DisasterConfig` 写接口若收到顶层 `syncPolicy`，必须直接返回 4xx，避免旧错误提案继续渗透到客户端
- `DisasterInstance` create/update/detail/list 继续使用 `dataSyncPolicy` 与 `resourceSyncPolicy`，不新增对外 `syncPolicy`
- `DisasterInstance` detail/list 继续返回 `effectiveDataSyncPolicy`、`effectiveResourceSyncPolicy`、`dataSyncPolicySource`、`resourceSyncPolicySource`，覆盖继承、单字段 override、双字段 override 场景
- 前端在“选择策略候选项”时可以复用统一策略数据源，但这不改变 server API 字段
- 配置详情中的 cron 回显需要与 operator 真实策略一致，避免继续伪造默认 cron

## Non-Goals
- 不修改 operator CRD 或 controller 的双字段语义
- 不引入新的统一 `syncPolicy` API 字段
- 不修改 operator 执行 CRD 语义
- 不修改前端展示实现本身

## Impact
- Affected specs:
  - `disaster_config`
  - `disaster_instance`
- Affected code:
  - `internal/apis/disaster_config/v1/*`
  - `internal/apis/disaster_instance/v1/*`
