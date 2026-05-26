## ADDED Requirements

### Requirement: Failover 参数 skipScaleDownSource 透传

系统必须 (MUST) 在容灾操作创建接口中支持透传 `skipScaleDownSource`，用于控制 Operator 在 failover 时是否跳过源集群缩零步骤。

该参数约束如下：
- 仅当 `operation=failover` 时生效；
- 允许读取 `skipScaleDownSource` 与 `SkipScaleDownSource` 两种键名；
- 服务端必须 (MUST) 写入 annotation `testudo.softcdata.com/skip-scale-down-source=true` 作为跨版本兼容兜底；
- 未传时默认 `false`。

#### Scenario: 实例 failover 透传 skipScaleDownSource

- **WHEN** 客户端请求 `POST /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/actions`
- **AND** 请求体 `operation` 为 `failover`
- **AND** `config.skipScaleDownSource=true`
- **THEN** 服务端创建的 `DisasterOperation.spec.skipScaleDownSource` 必须 (MUST) 为 `true`

#### Scenario: 组 failover 透传 skipScaleDownSource

- **WHEN** 客户端请求 `POST /apis/disastergroups.testudo.softcdata.com/v1/groups/:name/actions`
- **AND** 请求体 `operation` 为 `failover`
- **AND** `config.skipScaleDownSource=true`
- **THEN** 服务端创建的 `DisasterOperation.spec.skipScaleDownSource` 必须 (MUST) 为 `true`

#### Scenario: 非 failover 操作忽略 skipScaleDownSource

- **WHEN** 客户端请求实例或组 action 接口
- **AND** `operation` 不为 `failover`
- **AND** `config.skipScaleDownSource=true`
- **THEN** 服务端创建的 `DisasterOperation.spec.skipScaleDownSource` 必须 (MUST) 保持默认值 `false`

#### Scenario: 兼容模式使用 annotation 透传

- **WHEN** 客户端发起 failover 并传入 `config.skipScaleDownSource=true`
- **THEN** 服务端创建的 `DisasterOperation.metadata.annotations["testudo.softcdata.com/skip-scale-down-source"]` 必须 (MUST) 为 `"true"`
