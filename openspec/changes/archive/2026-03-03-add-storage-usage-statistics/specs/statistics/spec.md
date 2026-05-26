# Capability: Storage Statistics

## Purpose
The purpose of this capability is to provide comprehensive API endpoints for obtaining storage usage statistics across the disaster recovery platform. This includes total backup counts, used space, available space, and overall usage rates, leveraging a read-write separation strategy between the backend server and the underlying operator.

## ADDED Requirements

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
