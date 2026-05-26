## ADDED Requirements

### Requirement: Cluster API 必须提供恢复 Class 列表查询接口

系统必须 (MUST) 提供按集群名返回恢复可用 `StorageClass` 与 `IngressClass` 列表的只读接口。

#### Scenario: 成功查询恢复 Class 列表
- **WHEN** 客户端请求 `GET /apis/cluster.testudo.softcdata.com/v1/clusters/:name/restore-classes`
- **THEN** 服务端必须 (MUST) 返回 200
- **AND** 响应数据必须 (MUST) 包含 `targetCluster`、`storageClasses`、`ingressClasses`
- **AND** `storageClasses` 与 `ingressClasses` 必须 (MUST) 按 `name` 升序返回

#### Scenario: 集群不存在时返回 404
- **GIVEN** `:name` 对应集群不存在
- **WHEN** 服务端处理列表请求
- **THEN** 服务端必须 (MUST) 返回 404

### Requirement: 恢复 Class 列表接口必须返回默认 Class 标记

系统必须 (MUST) 在列表项中返回默认类标记，支持前端直接展示默认候选项。

#### Scenario: 返回默认 StorageClass 与 IngressClass 标记
- **GIVEN** 目标集群中存在默认 `StorageClass` 与默认 `IngressClass`
- **WHEN** 服务端完成列表查询
- **THEN** 对应列表项必须 (MUST) 返回 `isDefault=true`

### Requirement: 恢复 Class 列表接口不得产生副作用

系统必须 (MUST) 保证该接口仅执行读取与响应组装，不得影响业务资源状态。

#### Scenario: 列表查询后业务资源保持不变
- **WHEN** 客户端调用恢复 Class 列表接口
- **THEN** 服务端不得创建、更新、删除 `DisasterInstance`、`DisasterOperation`、`AppRestore`
- **AND** 不得触发任何恢复任务
