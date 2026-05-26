## 任务列表 (Tasks)
1. [x] 在 `disaster-operator/pkg/apis/disaster/v1/storagerepository_types.go` 中更新 `StorageRepositorySpec` 和 `StorageRepositoryStatus`:
   - 添加字段 `QuotaBytes int64 \`json:"quotaBytes,omitempty"\``。该字段定义存储配置的存储配额限额限制。
   - 在 `StorageRepositoryStatus` 中添加字段 `UsedSpaceBytes int64 \`json:"usedSpaceBytes,omitempty"\`` 和 `TotalBackupCount int64 \`json:"totalBackupCount,omitempty"\``，用于记录控制器的固定扫描探测结果。
2. [x] 在 `disaster-operator/internal/controller/storagerepository` 内的 `StorageRepositoryController.Reconcile` 方法中追加 S3 用量监控统计逻辑。
   - 确定标准：在 S3 连通性测试 `ValidateS3Configuration` 成功之后，并且每次返回 `ctrl.Result{RequeueAfter: 10 * time.Minute}` 之前。
   - 触发方式：同步调用利用 `minio-go/v7` 的 `ListObjects` 获取对象字节总大小（已用空间），并且基于前缀子级目录 `backups/` 统计实际总备份数。
   - 最后，直接将统计指标保存到 `sr.Status.UsedSpaceBytes` 和 `sr.Status.TotalBackupCount` 并持久化记录。
3. [x] 更新 Server API 模型 (`disaster-server/internal/apis/disaster_storage/v1/types.go`):
   - 修改 `DisasterStorageSpecDTO`，加入 `QuotaBytes int64`，以便前端能够在新建和编辑时传递该限额设置。
   - 修改 `DisasterStorageStatusDTO`，加入对应参数 `UsedSpaceBytes` 及 `TotalBackupCount` 反向返回状态全貌详情。
   - 调整 `CreateDisasterStorageRequest` 与 `UpdateDisasterStorageRequest` 中的转换函数接口，无缝接合配置。
4. [x] 定义统计相关的聚合数据展现专用模型 (`StorageUsageDTO`)，该模型放入 `disaster-server/internal/apis/statistics/v1` 内，包含 `TotalBackupCount`, `UsedSpace`, `AvailableSpace`, `UsageRate` 和 `QuotaBytes` 整体分析数据。
5. [x] 实现新的统计分析接口 HTTP Handler 并完成在 `disaster-server/internal/router` 上的 `GET /apis/statistics/v1/storage` 路由绑定过程。
6. [x] 落实统计服务端 (`Server`) 内核端点的聚合指标计算:
   - 确定标准：直接通过 Kubernetes Client 拿取 `StorageRepository` 队列配置列表；不允许从控制面再次发起 S3 网络分析请求；仅依靠取出的 `Spec.QuotaBytes`, `Status.UsedSpaceBytes`, `Status.TotalBackupCount` 这三个常量数据进行无状态汇总。
   - 对于空或零限额 (`QuotaBytes = 0`)，制定异常防御措施与默认的返回值判定 (不抛出除以零报错)。
7. [x] 添加完善有关聚合状态处理以及新限额传参保存行为的集成性单元测试套件代码段。
8. [x] 使用 MCP 指令或直接按照当前系统标准验证与构建前后端 API 通信格式变更规格约定文档。
