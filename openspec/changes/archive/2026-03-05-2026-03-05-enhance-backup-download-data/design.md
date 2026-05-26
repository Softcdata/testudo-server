# 增强备份数据下载 - 详细设计

## 1. 架构目标
强化 `GET /appbackups/:name/backups/:backupName/download` API 的能力。
不仅提供原有的 Kubernetes 资源打包文件（`<backup-name>.tar.gz`）下载，还需要支持底层业务数据文件（Restic/Kopia 的持久卷备份数据）的下载。

## 2. 问题分析与难点
目前系统中通过预签名 URL 仅能直接获取单个 S3 对象：`/<cluster>/backups/<backup-name>/<backup-name>.tar.gz`。

对于**数据文件下载**面临以下挑战：
1. **多对象且依赖解析**：Velero 备份除了 `tar.gz` 文件外，其核心的资源清单被保存在 `velero-backup.json` 中。
2. **Kopia/Restic 数据池机制及路径结构**：
   底层真实的 PV 业务数据是被拆分为多个离散的 Chunk，基于 Namespace 隔离存放在特定的 Prefix 下。
   实际 MinIO 中观察到的目录结构如下：
   - **备份资源元信息**：`/<cluster>/backups/<backup-name>/` (包含 `velero-backup.json`, `<backup-name>.tar.gz`, `*volumebackups.json.gz` 等)
   - **底层的 PV 数据块**：`/<cluster>/kopia/<namespace>/` 或 `/<cluster>/restic/<namespace>/` 
   无法直接通过一个 S3 Presigned URL 导出以上全部分散的对象。
3. **大文件代理内存溢出风险**：将多个存储对象打包下载如果直接读入内存会导致 Server OOM 崩毁。

## 3. 设计方案

### 3.1 扩展 API 参数
增强现有的 Download 端点，增加 `type` 查询参数控制返回数据的内容。

请求路径：`GET /appbackups/:name/backups/:backupName/download?type={resource|data|all}`

**行为定义：**
- `type=resource`（或不传）：保留现有逻辑，默认只返回资源文件 `/<cluster>/backups/<backup-name>/<backup-name>.tar.gz` 的预签名 URL。
- `type=data`：流式打包底层的持久卷 PV 数据。需根据该备份涵盖的 Namespaces，打包所有关联的 `/<cluster>/kopia/<namespace>/` 和 `/<cluster>/restic/<namespace>/` 数据。
- `type=all`：流式打包下载。包含该备份在 `/<cluster>/backups/<backup-name>/` 下的所有文件，以及附带所有的 `/<cluster>/kopia/<namespace>/` PV 业务数据全集。

### 3.2 引入 `S3StreamZipper` (流式代理打包人)
在 `internal/kube/storage/` 包下，引入 `S3StreamZipper` 模块增强 `StorageService` 接口：

```go
type StorageService interface {
    GetDownloadURL(ctx context.Context, endpoint, accessKey, secretKey, bucket, region, objectKey string, expires time.Duration) (string, error)
    
    // 新增：流式打包给定的多个前缀的 S3 Objects 并实时写入 io.Writer 
    StreamDownloadPrefixes(ctx context.Context, endpoint, accessKey, secretKey, bucket, region string, prefixes []string, w io.Writer) error
}
```

**StreamDownloadPrefixes 实现逻辑：**
1. 获取 `minio.Client` 实例。
2. 针对 HTTP Response，设置 Header `Content-Type: application/x-tar` 与 `Content-Disposition: attachment; filename="backup-{name}-{type}.tar"`。
3. 创建以 `io.PipeWriter` 结合 `archive/tar` 零内存缓存流式推送给 Hertz 的 `SetBodyStream(pr, -1)`。
4. 针对需下载的 `prefixes` 列表，使用 `client.ListObjects(ctx, bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: true})` 递归获取对应的多级对象。
5. 遍历 S3 Object：
   - 写入文件的 Tar/Zip 头部（剥离绝对层级，保留相对结构避免深层级空目录）。
   - 初始化 `objStream, err := client.GetObject(...)` 获取对象数据。
   - 使用 `io.Copy(tarWriter, objStream)` 零拷贝转发 S3 数据流给前端浏览器/客户端，期间不占用过多机器内存。
   - 完成单独对象拷贝后，关闭此时的 `objStream`，直到迭代完所有文件。
6. 最终完成流的收尾封包并关闭 Writer。

### 3.3 数据对应关系 (Prefix 解析规则)
当前端触发 `type=data` 或 `type=all`，Server 该打包哪些具体的 Prefix 路径？

需要遵循以下前缀映射表规则：
1. **备份元数据主目录**：
   - 路径：`/<cluster>/backups/<backupName>/`
   - 此目录必须始终被打包（若 `type=all`），内部必然含有 `<backupName>.tar.gz` 和声明该次备份目标涵盖了哪些 Namespaces 的 `velero-backup.json` 文件。
2. **锁定 Namespaces 获取 Kopia/Restic 数据池路径**：
   - 读取 Kubernetes 的 `AppBackup` CRD 本身的 `Spec.Template.IncludedNamespaces` 字段，以确定本次备份的目标。
   - 对于每个目标 Namespace：
     添加路径 `/<cluster>/kopia/<namespace>/` 的遍历规则。
     添加路径 `/<cluster>/restic/<namespace>/` 的遍历规则（兼容两种后端引擎）。
     
*(注意: 这里基于简化实现，我们不再去强解析 json，而是直接利用 AppBackup CRD 上的 IncludedNamespaces 信息即可是最高效的方案)*

### 4. 异常与回滚设计
- **客户端断连或中断下载**：`io.Copy` 返回 `write: broken pipe` 的错误。由于完全采用流式处理，只需取消或退出当前读取的 S3 对象即可释放协程池及网络，无僵尸进程残留和资源卡死风险。
- **MinIO 认证/授权失效**：提前下放获取集群 StorageRepository 凭据和解析，如果无法建立通信返回 503 Service Unavailable 给客户端，停止引发后续读操作。

### 5. 测试策略 (BDD)
- `type=resource` 时，保持向下兼容不变，确保存活和返回了临时签发的 Presigned URL；
- `type=data/all` 时，使用 BDD 打桩出伪造的 minio 目录，验证流式 Tar 能够包含 Restic/Kopia 的块文件目录和数据。模拟一次极大内存占用操作，必须断言监控 Goroutine 或者 Memory 内存处于正常阈值无泄漏现象，并且确认 CPU 不会因为二次解压/压缩而居高不下。
