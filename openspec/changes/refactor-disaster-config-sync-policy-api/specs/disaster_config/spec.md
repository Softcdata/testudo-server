## ADDED Requirements

### Requirement: DisasterConfig API 必须保持双字段策略契约
系统必须 (MUST) 对外继续使用 `dataSyncPolicy` 与资源同步双字段表达基础配置策略，不得把统一 `syncPolicy` 作为正式接口字段。

#### Scenario: 使用双字段创建或更新基础配置
- **Given** 客户端提交一个包含 `dataSyncPolicy` 与资源同步字段的基础配置请求
- **When** Server 将该请求写入 operator CRD
- **Then** Server 必须把 `dataSyncPolicy` 写入 `spec.dataSyncPolicy`
- **And** 必须把资源同步字段写入 `spec.resourceSyncPolicy`

#### Scenario: 请求包含统一 syncPolicy 时拒绝写入
- **Given** 客户端提交一个包含顶层 `syncPolicy` 的基础配置请求
- **When** Server 校验该请求
- **Then** Server 必须将该请求视为无效并拒绝写入

### Requirement: DisasterConfig API 必须稳定回显双字段
系统必须 (MUST) 在 detail/list 路径继续回显双字段契约，避免客户端读取到未约定的统一字段。

#### Scenario: 读取基础配置详情时返回双字段
- **Given** 一个 `DisasterConfig` 已写入 operator CRD
- **When** 客户端读取该资源详情或列表项
- **Then** Server 必须返回 `dataSyncPolicy`
- **And** 必须返回资源同步双字段契约
- **And** 不得额外回显统一 `syncPolicy`

### Requirement: 配置 cron 回显必须反映真实策略
系统必须 (MUST) 只回显真实存在且可解析的策略 cron，不得在无有效策略时伪造默认值。

#### Scenario: 未配置策略时不返回伪默认 cron
- **Given** 一个 `DisasterConfig` 未绑定有效的同步策略
- **When** 客户端读取该资源详情
- **Then** Server 不得伪造默认 `dataSyncCron` 或 `resourceSyncCron`
