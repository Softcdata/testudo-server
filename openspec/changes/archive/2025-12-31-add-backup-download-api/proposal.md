# Change: 添加备份数据下载 API

## Why
目前系统支持创建和管理应用备份，但用户无法直接下载备份数据。用户需要能够选择指定应用备份下的某个 Velero 备份进行下载，以便本地保存或迁移到其他环境。备份数据存储在 MinIO 上，需要提供一个 API 来生成预签名下载链接或代理下载。

## What Changes
- 新增 `GET /appbackups/:name/backups/:backupName/download` API 端点
- 实现从 MinIO 获取备份数据的预签名 URL 或流式下载
- 支持指定 Velero 备份名称进行下载
- 添加下载权限校验

## Impact
- 受影响的规范：`specs/app-backup`
- 受影响的代码：
  - `internal/apis/app_backup/v1/handler.go` - 新增下载处理函数
  - `internal/apis/app_backup/v1/router.go` - 新增路由
  - `internal/kube/` - 可能需要 MinIO 客户端封装

## 关键测试场景
1. 成功获取指定备份的下载链接
2. 备份不存在时返回 404
3. MinIO 连接失败时的错误处理
4. 权限校验失败时返回 403
