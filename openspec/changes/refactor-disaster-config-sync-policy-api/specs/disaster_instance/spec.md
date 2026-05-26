## ADDED Requirements

### Requirement: DisasterInstance API 必须保持双字段 override 契约
系统必须 (MUST) 对外继续使用 `dataSyncPolicy` 与 `resourceSyncPolicy` 表达实例级 override，不得把统一 `syncPolicy` 作为正式接口字段。

#### Scenario: 使用双字段创建或更新实例 override
- **Given** 客户端提交一个包含实例级双字段的请求
- **When** Server 将该请求写入 operator CRD
- **Then** Server 必须把 `dataSyncPolicy` 写入 `DisasterInstance.spec.dataSyncPolicy`
- **And** 必须把 `resourceSyncPolicy` 写入 `DisasterInstance.spec.resourceSyncPolicy`

#### Scenario: 请求包含统一 syncPolicy 时拒绝写入
- **Given** 客户端提交一个包含顶层 `syncPolicy` 的实例请求
- **When** Server 校验该请求
- **Then** Server 必须将该请求视为无效并拒绝写入

### Requirement: DisasterInstance API 必须在 detail/list 中统一回显有效策略与字段来源
系统必须 (MUST) 在实例详情与列表项中统一返回 `effectiveDataSyncPolicy`、`effectiveResourceSyncPolicy`、`dataSyncPolicySource`、`resourceSyncPolicySource`，并继续保留原始 override 双字段。

#### Scenario: 实例未设置 override 时 detail/list 回显继承值
- **Given** 一个 `DisasterInstance` 未设置实例级同步策略字段
- **And** 其关联基础配置已提供有效同步策略
- **When** 客户端查询该实例详情或列表项
- **Then** Server 必须返回继承后的 `effectiveDataSyncPolicy` 与 `effectiveResourceSyncPolicy`
- **And** `dataSyncPolicySource` 与 `resourceSyncPolicySource` 必须标明来源为 `config`

#### Scenario: 实例只覆盖单字段时 detail/list 回显部分 override
- **Given** 一个 `DisasterInstance` 仅设置了单个实例级同步策略字段
- **And** 另一个维度仍从基础配置继承
- **When** 客户端查询该实例详情或列表项
- **Then** Server 必须返回两个维度各自的 effective 策略值
- **And** override 的维度来源必须标明为 `instance`
- **And** 继承的维度来源必须标明为 `config`
- **And** 不得额外回显统一 `syncPolicy`

#### Scenario: 实例设置双字段 override 时回显为实例来源
- **Given** 一个 `DisasterInstance` 已设置实例级同步策略字段
- **When** 客户端查询该实例详情或列表项
- **Then** Server 必须把两个维度的来源都标明为 `instance`
- **And** 必须返回实例级 override 的 effective 值
- **And** 不得额外回显统一 `syncPolicy`
