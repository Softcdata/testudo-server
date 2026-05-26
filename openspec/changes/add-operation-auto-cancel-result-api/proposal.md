# Change: 为实例/操作详情补齐自动补偿结果契约

## Why
条目 21 在 operator 侧会引入“失败后自动补偿”的正式语义。如果 server 侧没有稳定的详情/列表回显契约，上层仍然只能通过事件文本猜测到底有没有触发自动补偿、是否成功、还需不需要人工介入。

## What Changes

### 1. 实例 detail/list 增加统一自动补偿摘要对象
- 首期不新增新的 detail route，而是在现有实例 detail/list 响应中增加统一摘要对象：
  - `autoCancel.triggered`
  - `autoCancel.status`
  - `autoCancel.reason`
  - `autoCancel.triggerStep`
  - `autoCancel.manualInterventionRequired`
  - `autoCancel.triggeredAt`
  - `autoCancel.completionTime`
- 该摘要对象来自 operator 的稳定状态字段，不由 server 自由推断。

### 2. 历史与时间线接口补齐自动补偿节点
- 当操作失败后触发自动补偿时，历史/时间线必须能稳定展示：
  - 失败
  - 自动补偿触发
  - 自动补偿成功/失败
- 首期优先复用现有实例 history / 组操作 watch / 事件 timeline 投影逻辑，不新建动作路由。

### 3. 列表接口只暴露摘要，不展开补偿步骤全文
- 列表只需要足够展示是否发生自动补偿和最终是否成功。
- 补偿步骤明细留在 history/watch/timeline 视图，不在 list 中展开。

## Non-Goals
- 不新增新的动作路由。
- 不替代 operator 侧状态记录。
- 不在 server 侧重新定义 auto-cancel 的状态机。

## Impact
- Affected specs:
  - `disaster_instance`
- Affected code:
  - `internal/apis/disaster_instance/v1/*`
  - `internal/apis/disaster_group/v1/*`
  - `internal/apis/event/v1/*`
  - 相关 DTO 聚合逻辑
- Cross-repo impact:
  - `disaster-operator`：提供稳定状态字段
  - `cluster-disaster-web`：展示时间线和详情摘要
