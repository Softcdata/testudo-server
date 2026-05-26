# disaster_storage Specification

## Purpose
待定 - 通过归档变更 `add-all-option-to-lists` 创建。归档后更新 Purpose。
## Requirements
### Requirement: 获取存储用量统计信息
The system MUST provide a RESTful API to retrieve storage usage statistics, including total backup counts, used space, and quota limitations.

#### Scenario: 获取所有存储资源的聚合统计
- **WHEN** 发送 GET 请求到 `/apis/statistics/v1/storage`
- **AND** 不提供任何查询参数
- **THEN** 后端 Web 服务程序将从 Kubernetes API 层面查询系统内目前所有被实例化的 `StorageRepository` 资源结构。
- **AND** 获取每个资源中预设在 `Spec.QuotaBytes` 里面的限额。
- **AND** 确定标准通过直接提取各对象底层状态空间栏的 `Status.UsedSpaceBytes` 及 `Status.TotalBackupCount` 数值（这些数值由底盘 `disaster-operator` 强制在主 Reconcile 间隔轮次运行 MinIO SDK 分析并写入）。
- **AND** 无需与底层 S3 发送流量，直接累加所有存储的限额（QuotaBytes）作为总可用空间限额，累加所有已用空间状态，聚合输出全体数据的统一总量以及整体的使用率。
- **AND** 即使不传配额，限额为 0 的情况下系统后台运算使用率时也能做到容灾兼容归零并正确展示。
- **AND** 响应状态码为 200，并返回带有整体聚合指标总计结构的 JSON 响应格式。

#### Scenario: 获取指定单一存储资源情况的统计汇总
- **WHEN** 发送 GET 请求到 `/apis/statistics/v1/storage`
- **AND** 提供指定精确存储仓库查询参数，如 `storageName=my-storage-1`
- **THEN** 后端 Web 服务仅需要匹配拉取对象池资源 `my-storage-1` 对应的特定 `StorageRepository` 资源配置项实体情况，进行安全判定其资源可用性。
- **AND** 单纯的从该实体内快速计算提取 `QuotaBytes`, `Status.UsedSpaceBytes`, 以及 `Status.TotalBackupCount` 的对应参数并返回其独有存储量占比。
- **AND** 返回仅针对特定仓库过滤目标后的统计数据 JSON 结构展示形式信息。
- **AND** 响应状态码必须保证为 200 的请求正确执行。

### Requirement: 列出所有存储名称
系统必须 (SHALL) 提供一个 API 接口来获取所有用于选择目的的存储名称列表。

#### Scenario: 获取存储名称
- **WHEN** 客户端请求 `GET /disaster/v1/storages/names`
- **THEN** 返回所有存储的对象列表，仅包含 `name` 字段
- **AND** 列表不需要分页

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

