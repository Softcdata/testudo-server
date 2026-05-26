# Change: 增强备份下载接口以支持数据文件下载

## Why
在早期的提案中（已归档的 2025-12-31-add-backup-download-api），我们提供了备份下载，但该接口仅下发了 `<backup-name>.tar.gz` 资源文件组成的 S3 预签名 URL。用户无法获取由 Restic/Kopia 生成的底层 Volume 持久卷数据，导致无法获取真正的业务代码和数据备份。因此需要对下载接口进行增强，支持下载具体的备份持久卷数据文件。

## What Changes
- 修改 `GET /appbackups/:name/backups/:backupName/download` API 端点，增加 `type` 查询参数（可选值：`resource`, `data`, `all`，默认 `resource`）。
- **BREAKING** `type=data` 或 `type=all` 机制无法依赖单一对象的 S3 预签名 URL，因为数据呈分布式多对象存储状态（如：kopia 目录内的多重块文件）。Server 端需要新增支持将多个 S3 对象实时打包（采用 Streaming Raw Tar 处理，**严禁再次 Gzip 压缩否则会引发 CPU 100% 打死系统**）并通过 `io.Pipe` 代理流式下发给客户端，避免 OOM（内存溢出）。
- 升级 `StorageService` 接口，增加支持拉取和扫描含有特定 S3 prefix（如备份相关数据块）的对象并流式转发。

## Impact
- Affected specs: `specs/app-backup`
- Affected code:
  - `internal/apis/app_backup/v1/handler.go` 
  - `internal/kube/storage/` (Storage 服务抽象层新增打包下载能力)
