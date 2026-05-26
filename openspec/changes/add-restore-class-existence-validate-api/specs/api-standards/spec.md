## ADDED Requirements

### Requirement: DisasterInstance 必须提供恢复 Class 存在性预检接口

系统必须 (MUST) 提供实例级接口，用于在触发恢复前检查目标集群中 `StorageClass` 与 `IngressClass` 的目标值存在性。

#### Scenario: 调用恢复 Class 预检接口
- **WHEN** 客户端请求 `POST /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/restore-classes/validate`
- **THEN** 服务端必须 (MUST) 返回结构化预检结果
- **AND** 结果必须包含 `storageClassCheck` 与 `ingressClassCheck`

#### Scenario: 输入映射校验
- **GIVEN** 请求体缺失 `storageClassMapping` 且缺失 `ingressClassMapping`
- **WHEN** 服务端执行预检
- **THEN** 必须 (MUST) 返回 400
- **AND** 错误码必须 (MUST) 为 `ClassMappingInvalid`

#### Scenario: 目标集群优先级
- **GIVEN** 请求体提供 `targetCluster`
- **WHEN** 服务端执行预检
- **THEN** 必须 (MUST) 使用该值作为目标集群
- **AND** 请求体缺失时必须 (MUST) 依次回退到 `DisasterInstance.status.secondaryCluster` 与 `DisasterConfig.spec.targetCluster`

### Requirement: 预检结果必须与 strictTargetValidation 语义一致

系统必须 (MUST) 根据映射策略中的 `strictTargetValidation` 计算最终 `valid` 状态，并返回可直接提示的失败码。

#### Scenario: strict 模式缺失 StorageClass
- **GIVEN** `storageClassMapping.strictTargetValidation=true`
- **AND** 至少一个 `targetClass` 在目标集群不存在
- **WHEN** 服务端完成检查
- **THEN** 返回 `valid=false`
- **AND** 返回 `code=StorageClassTargetNotFound`
- **AND** 在 `storageClassCheck.missingTargets` 中返回缺失项

#### Scenario: strict 模式缺失 IngressClass
- **GIVEN** `ingressClassMapping.strictTargetValidation=true`
- **AND** 至少一个 `targetClass` 在目标集群不存在
- **WHEN** 服务端完成检查
- **THEN** 返回 `valid=false`
- **AND** 返回 `code=IngressClassTargetNotFound`
- **AND** 在 `ingressClassCheck.missingTargets` 中返回缺失项

#### Scenario: 非 strict 模式缺失目标 Class
- **GIVEN** 映射策略 `strictTargetValidation=false`
- **AND** 存在缺失目标 Class
- **WHEN** 服务端完成检查
- **THEN** 返回 `valid=true`
- **AND** 必须 (MUST) 在缺失列表中返回明细

### Requirement: 预检接口不得产生副作用

系统必须 (MUST) 保证预检接口只执行读取与计算，不得影响现有恢复链路与容灾实例链路。

#### Scenario: 预检调用后的资源状态保持不变
- **WHEN** 客户端调用恢复 Class 预检接口
- **THEN** 服务端不得创建、更新、删除 `DisasterInstance`、`DisasterOperation`、`AppRestore`
- **AND** 不得触发任何恢复任务
