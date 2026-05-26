# Tasks: 添加备份数据下载 API

## 0. 测试设计（测试先行）
- [x] 0.1 设计下载 API 的测试场景（成功/404/403/500）
- [x] 0.2 在 `handler_test.go` 中实现 `downloadBackup` 的单元测试框架

## 1. 基础设施准备
- [x] 1.1 调研 MinIO SDK 的预签名 URL 生成方式
- [x] 1.2 确认备份数据在 MinIO 上的存储路径结构（通常为 `backups/<backup-name>/`）
- [x] 1.3 在 `internal/kube/` 或新建 `internal/storage/` 封装 MinIO 客户端

## 2. API 实现
- [x] 2.1 在 `types.go` 中定义下载响应结构体 `BackupDownloadResponse`
- [x] 2.2 在 `handler.go` 中实现 `downloadBackup` 处理函数
  - [x] 2.2.1 解析路径参数 `:name` (AppBackup) 和 `:backupName` (Velero Backup)
  - [x] 2.2.2 验证 AppBackup 存在且包含指定的备份记录
  - [x] 2.2.3 从 StorageRepository 获取 MinIO 连接信息
  - [x] 2.2.4 生成预签名下载 URL（有效期可配置，默认 1 小时）
  - [x] 2.2.5 返回下载链接或重定向
- [x] 2.3 在 `router.go` 中注册新路由 `GET /appbackups/:name/backups/:backupName/download`

## 3. 测试与验证
- [x] 3.1 运行单元测试并确保通过
- [x] 3.2 手动测试完整流程（创建备份 → 下载备份）
- [x] 3.3 验证预签名 URL 的有效性和安全性

## 4. 文档
- [x] 4.1 更新 API 文档（如果存在）
