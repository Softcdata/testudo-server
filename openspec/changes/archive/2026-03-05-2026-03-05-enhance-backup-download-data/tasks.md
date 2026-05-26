## 1. Implementation
- [x] 1.1 设计增强备份下载（支持持久卷数据文件）的 BDD 测试场景
- [x] 1.2 更新 `StorageService` 接口，支持遍历前缀和打包流式下载多个对象
- [x] 1.3 实现 S3 客户端前缀文件拉取与 `tar.gz` 实时打包压缩功能
- [x] 1.4 修改 `AppBackupHandler.downloadBackup`，处理 `type` 查询参数（`resource`, `data`, `all`）并正确响应（预签名或流式）
- [x] 1.5 验证下载数据文件的完整性与正确性
- [x] 1.6 验证代码测试覆盖率达标
