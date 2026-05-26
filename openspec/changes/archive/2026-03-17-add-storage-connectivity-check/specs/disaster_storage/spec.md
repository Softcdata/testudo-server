## ADDED Requirements

### Requirement: 跨集群恢复前置存储校验
系统必须 (MUST) 提供恢复前置校验接口，在创建 `AppRestore` 前验证跨集群恢复必需 BSL 的可用状态。

#### Scenario: 必需 BSL 已可用
- **GIVEN** 请求体包含 `backupSource` 与 `targetCluster`
- **WHEN** 客户端调用 `POST /apis/apprestores.testudo.softcdata.com/v1/apprestores/preflight/validate`
- **THEN** 服务端必须从 `backupSource` 对应 `AppBackup.Spec` 提取 `sourceCluster` 与 `storageRepository`
- **AND** 服务端必须计算 `requiredBSL=<storageRepository>-<sourceCluster>`
- **AND** 服务端必须在目标集群 `velero` 命名空间读取 `requiredBSL`
- **AND** 当 `requiredBSL.status.phase=Available` 时返回 `data=true`

#### Scenario: 必需 BSL 缺失触发创建
- **GIVEN** 目标集群不存在 `requiredBSL`
- **WHEN** 服务端执行前置校验
- **THEN** 服务端必须给目标 `Cluster` 写入注解 `testudo.softcdata.com/ensure-storage=<storageRepository>`
- **AND** 服务端必须给目标 `Cluster` 写入注解 `testudo.softcdata.com/ensure-storage-source-cluster=<sourceCluster>`
- **AND** 服务端必须按固定节奏轮询目标集群 BSL：每 `1` 秒读取一次，总次数等于 `waitSeconds`
- **AND** 在轮询窗口内读到 `phase=Available` 时返回 `data=true`

#### Scenario: 必需 BSL 未在窗口内就绪
- **GIVEN** 前置校验进入轮询阶段
- **WHEN** 达到最大轮询次数仍未出现 `phase=Available`
- **THEN** 服务端返回 `data=false`
- **AND** 响应 `meta` 必须包含 `required_bsl`、`source_cluster`、`target_cluster`、`storage_repository` 与失败原因

### Requirement: 创建应用恢复硬闸门
系统必须 (MUST) 在 `createAppRestore` 流程中复用前置校验逻辑，未通过校验时禁止创建 `AppRestore`。

#### Scenario: 创建前校验失败
- **GIVEN** 客户端调用 `POST /apis/apprestores.testudo.softcdata.com/v1/apprestores`
- **WHEN** 前置校验结果为失败
- **THEN** 服务端不得创建 `AppRestore`
- **AND** 服务端必须直接返回失败响应并附带失败原因
