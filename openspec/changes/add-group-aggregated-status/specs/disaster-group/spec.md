## ADDED Requirements

### Requirement: 容灾组聚合状态

系统 SHALL 在 `getGroup`（详情）和 `listGroups`（列表）接口的响应中，通过 Server 侧实时计算提供 `status.fsmState` 聚合状态字段，使调用方无需自行遍历 `instances` 数组即可判断整组的健康情况。

聚合状态 SHALL 取自以下枚举值，计算采用**优先级投票**规则（高优先级覆盖低优先级）：

| 优先级 | `fsmState` | 触发条件 |
|--------|-----------|---------|
| 1（最高）| `FailingOver` | 有任意实例 `fsmState == "FailingOver"` |
| 1（最高）| `FailingBack` | 有任意实例 `fsmState == "FailingBack"` |
| 2 | `Degraded` | 有任意实例 `fsmState == "Failed"`，且无操作进行中 |
| 3 | `Active` | 所有实例均为 `Active` |
| 3 | `Paused` | 所有实例均为 `Paused` |
| 3 | `Protected` | 所有实例均为 `Protected` |
| 4 | `Initializing` | 有实例为 `Pending` 或 `Initializing` |
| 5 | `PartialProtected` | 以上均不满足的混合状态 |
| 6（最低）| `Unknown` | 组内无实例，或全部实例查询失败 |

#### Scenario: 全部实例 Protected 时组聚合为 Protected

- **WHEN** 调用 `GET /groups/:name` 或 `GET /groups`
- **AND** 组内所有 `DisasterInstance.status.fsmState` 均为 `Protected`
- **THEN** 响应中 `status.fsmState` SHALL 为 `"Protected"`

#### Scenario: 有 FailingOver 实例时组优先聚合为 FailingOver

- **WHEN** 组内有任意一个实例的 `fsmState` 为 `"FailingOver"`（其余实例为任意状态）
- **THEN** 响应中 `status.fsmState` SHALL 为 `"FailingOver"`

#### Scenario: 有 Failed 实例且无操作进行中时组聚合为 Degraded

- **WHEN** 组内有任意实例 `fsmState == "Failed"`
- **AND** 无实例处于 `"FailingOver"` 或 `"FailingBack"`
- **THEN** 响应中 `status.fsmState` SHALL 为 `"Degraded"`

#### Scenario: 混合状态时组聚合为 PartialProtected

- **WHEN** 组内实例状态混合（如部分 Protected、部分 Paused），且无操作进行中、无 Failed
- **THEN** 响应中 `status.fsmState` SHALL 为 `"PartialProtected"`

#### Scenario: 空组或无法获取实例时组聚合为 Unknown

- **WHEN** `spec.levels` 展平后为空，或所有实例均查询失败
- **THEN** 响应中 `status.fsmState` SHALL 为 `"Unknown"`

---

### Requirement: 容灾组可用操作自动推导

系统 SHALL 在响应中提供 `status.availableOperations` 字段，根据 `status.fsmState` 自动推导当前整组可执行的操作列表，供前端 enable/disable 按钮判断。

| `fsmState` | `availableOperations` |
|------------|-----------------------|
| `Protected` | `["failover", "pause", "synconce", "syncdata", "syncresource"]` |
| `PartialProtected` | `["failover", "pause", "synconce"]` |
| `Paused` | `["resume"]` |
| `Active` | `["reprotect"]` |
| `Degraded` | `[]`（需人工介入各实例） |
| `FailingOver` / `FailingBack` | `[]`（操作进行中，防止重入） |
| `Initializing` / `Unknown` | `[]` |

#### Scenario: Protected 状态下可用操作正确

- **WHEN** `status.fsmState == "Protected"`
- **THEN** `status.availableOperations` SHALL 等于 `["failover", "pause", "synconce", "syncdata", "syncresource"]`

#### Scenario: FailingOver 状态下可用操作为空

- **WHEN** `status.fsmState == "FailingOver"`
- **THEN** `status.availableOperations` SHALL 为空列表，前端 SHALL 禁用所有操作按钮以防止重入
