# Design: syncPolicy 接口收敛到双字段契约

## 关键决策
- server 不新增对外 `syncPolicy` 字段
- `DisasterConfig` 写接口继续使用双字段。主契约保持 `dataSyncPolicy` 与 `resourcesSyncPolicy`，同时保留 `resourceSyncPolicy` 兼容别名并统一写入 CRD 的 `spec.resourceSyncPolicy`
- `DisasterInstance` 写接口继续使用 `dataSyncPolicy` 与 `resourceSyncPolicy`
- server 对包含顶层 `syncPolicy` 的 `DisasterConfig` 与 `DisasterInstance` 请求直接返回 4xx，避免客户端误以为统一字段已成为正式契约
- 实例 override 继续持久化到 `DisasterInstance.spec.dataSyncPolicy` 与 `spec.resourceSyncPolicy`
- `DisasterInstance` detail/list 必须统一返回 `effectiveDataSyncPolicy`、`effectiveResourceSyncPolicy`、`dataSyncPolicySource`、`resourceSyncPolicySource`
- `DisasterInstance` detail/list 不再额外回显统一 `syncPolicy`。即使两个 effective 值相同，也只返回双字段与 effective/source 衍生字段
- 前端“正在选择策略”的候选数据源可以按统一列表提供，但提交与回显契约仍然是双字段
- cron 回显必须反映真实引用策略；当没有有效策略时不得伪造默认 cron

## 读写语义
- `DisasterConfig` 读取时继续返回双字段；当代码路径需要兼容历史客户端时，`resourcesSyncPolicy` 与 `resourceSyncPolicy` 返回相同值
- `DisasterConfig` 写入时把资源同步字段合并到 CRD `spec.resourceSyncPolicy`
- `DisasterInstance` 读取时返回原始 override 双字段，以及按字段独立计算的 effective/source 衍生字段
- `DisasterInstance` 写入时仅更新请求中明确提供的字段；未提供的字段保持现值
- 单字段 override 的继承语义按字段独立生效：实例字段非空则取实例值，实例字段为空则取基础配置对应字段
