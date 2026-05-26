# Proposal: 添加存储用量统计 API

## Why
灾备平台控制台需要一个“存储用量统计”组件，以展示各个存储仓库的整体存储消耗情况。所需的指标包括：
1. 总备份数 (Total Backup Count): 目标存储桶中按物理目录隔离的备份条目数量。
2. 已用空间 (Used Space): 备份对象消耗的总字节数。
3. 可用空间 (Available Space): 存储配额/限额减去已用空间。
4. 使用率 (Usage Rate): 已用空间 / (已用空间 + 限额)。

目前，`disaster-operator` 中的 `StorageRepository` CRD 并没有定义配额或限额规范的字段，也没有记录存储桶的实际使用量，且缺乏一个可以聚合所有已配置存储仓库使用情况的 API 接口。

## What Changes
本提案引入了 `Storage Usage Statistics API`（存储用量统计 API）能力。采用**Operator 同步更新状态**的方案：
- 修改 `disaster-operator` 中的 `StorageRepository` CRD：
  - 在 `Spec` 中增加 `QuotaBytes` 字段，用于支持系统限额。
  - 在 `Status` 中增加 `UsedSpaceBytes` 和 `TotalBackupCount` 字段。
- 在 `disaster-operator` 内的 `StorageRepositoryController` 中补充 S3 用量监控逻辑。每次成功的存储重试或验证（即 Reconcile 间隔 10 分钟轮询时），使用 MinIO SDK 遍历目标桶内部计算已用空间，并利用 `backups/` 目录前缀统计总备份数，随后更新到 CRD 的 `Status` 中。
- 在 `disaster-server` 中创建一个轻量级 API 路由端点（`/apis/statistics/v1/storage`），该接口将：
  - 支持通过查询参数 `storageName` 筛选特定的存储名获取单例详情。
  - 直接读取当前所有匹配的 `StorageRepository` 中的 `Spec.QuotaBytes` 和 `Status` 内数据（不执行任何实际的 S3 连接代码）。
  - 将获取的指标进行安全聚合计算生成总体使用率，并将最终数据结构层返回给前端仪表盘展示。

## 方案对比与决策 (Alternatives & Decision)
- **在 Server 端按需实时聚合计算 (Server-side On-demand Aggregation)**: 
  - *缺点*: 对于包含大量碎文件的存储桶，API 实时请求延迟极高，且每次前端看板刷新都会造成 S3 大量访问压力。
- **在 Operator 端持续更新状态 (Operator-side Status Updates)**: 
  - *优点*: 前端请求统计 API 服务响应极快（直接从 Local Cache 或 API 服务器拿取元数据，实现底座读写分离）；能够将耗时的 S3 I/O 隔离在 Kubernetes 后台底层。
  - *决策 (Decision)*: 采纳此标准方案。统一依托 Operator 将计算好指标写回到 CRD 状态栏，而 Server 统计分析 API 退化为单纯针对状态栏的累加计算及聚合分发服务出口。

## 范围之外 (Out of Scope)
- 本提案不涵盖前端控制台视图界面的具体编码，只涉及后端 API 及 Operator CRD 增修与发布。
