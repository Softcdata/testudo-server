# Change: 将 AppBackup 下载链接切换为平台代理地址

## Why
当前 `AppBackup` 下载接口把对象存储地址直接返回给客户端。客户端所在网络如果无法访问对象存储，下载就会失败。下载能力应该由平台服务端对外暴露下载地址，再由服务端访问对象存储并把内容转发给客户端，这样客户端只需要访问平台服务地址。

## What Changes
- 将 `GET /appbackups/:name/backups/:backupName/download` 的返回值改为平台服务端签发的同源下载地址。
- 新增下载流入口，服务端校验下载票据后再读取对象存储内容并流式返回。
- `type=resource` 走单文件代理下载，`type=data` 与 `type=all` 走流式归档下载。
- 扩展 storage 抽象，补齐对象流读取能力以及流式输出能力。
- 保持前端现有 `window.open(download_url)` 的调用方式不变。
- 同步 OpenAPI、RunAPI 以及本地证据说明，避免接口契约与实现脱节。

## Impact
- 受影响的规范：`specs/app-backup`
- 受影响的代码：
  - `internal/apis/app_backup/v1/handler.go`
  - `internal/apis/app_backup/v1/router.go`
  - `internal/apis/app_backup/v1/types.go`
  - `internal/storage/`
  - `openspec/specs/disaster-server-openapi.yaml`
- 受影响的前端使用方式：
  - `src/api/ApiBackupRecovery/ApiBackup.ts`
  - `src/views/BackupRecovery/ResourceBackup/components/BackupHistoryExpanded.vue`
  - `src/views/BackupRecovery/ResourceBackup/utils/hook.tsx`
