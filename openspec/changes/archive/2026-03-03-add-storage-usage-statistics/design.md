# 设计方案 (Design): 存储用量统计 (Storage Usage Statistics)

1. **新 API 端点 (New API endpoint)**: `GET /apis/statistics/v1/storage`
2. **应用资源模型更新设计 (CRD Mod)**: 
   - `StorageRepositorySpec` 增加 `QuotaBytes int64` 字段，以存放配置的全局存储限额规范。
   - `StorageRepositoryStatus` 增加 `UsedSpaceBytes int64` 和 `TotalBackupCount int64`，作为存储底盘最新运行数据的落盘位置。
3. **底层收集与状态更新 (Operator Collection)** (利用 `minio-go/v7`):
   - 确定标准：在 `disaster-operator` 的 `StorageRepositoryReconcile` 主逻辑验证流程里（当 `ValidateS3Configuration` 成功通过后返回 `RequeueAfter: 10 * time.Minute` 之前）。
   - 触发方式：同步在此 `Reconcile` 内执行 MinIO SDK 代码连接 S3 算出各项用量和文件占有数。
     - 已用空间 (Used Space): 通过 `minioClient.ListObjects(..., Recursive: true)` 一次拿到全存储桶里所有对象的字节大小后进行累积 `sum(object.Size)`。
     - 总备份数 (Total Backup Count): 设置查询参数前缀为 `Prefix="backups/"` 以及 `Recursive=false` 拿到实际目录结构列表。每个存在于桶结构内的前缀目录代表 1 个独立的物理备份集系统，进行一次总计总数累加 (`+1`)。
     - 数据回写：将上述两项计算结果保存并更新 `StorageRepository` 的 `.Status.UsedSpaceBytes` 及 `.Status.TotalBackupCount` 字段。
4. **服务器聚合计算逻辑标准 (Server Endpoint Logic)**:
   - 从 `disaster-server` 中的 k8s Client 缓存中查出满足查询条件的 `StorageRepository` 资源配置对象数组集合。
   - 不进行网络消耗（不调用 MinIO/S3 流量），而是直接提取各单独对象 Status 内更新的数据数值与 Spec 中的预设限额。
   - 快速求和统计运算：拿到全局所有资源底层的 `Quota`、`UsedSpace`、`BackupCount` 累加之和。

5. **聚合 JSON 响应结构定义 (Response Schema)**:
   ```json
   {
       "totalBackupCount": 35,
       "usedSpaceBytes": 10737418240, 
       "availableSpaceBytes": 96636764160,
       "usageRate": 0.1, // 代表 10%
       "quotaBytes": 107374182400
   }
   ```
   注：如果限额数 `quotaBytes` 为无限大限制情况 (设为 0)，统计服务判定该资源的 `availableSpaceBytes` 置位为 0，并且强制保证 `usageRate` 配置设定为 0（避免百分比异常出现超出逻辑显示值与除零异常）。
