## 1. 资源格式转换

- [x] 1.1 增加共享资源筛选转换函数，识别 GVK、group-resource 以及通配符。
- [x] 1.2 使用源集群 RESTMapper 将 GVK 转换为 group-resource，并返回字段路径化错误。
- [x] 1.3 在备份资源清单接口转换 `includedResources`，覆盖真实清单以及回退到 BackupSpec 的路径。

## 2. AppRestore 兼容

- [x] 2.1 在创建 AppRestore 时转换 `includedResources` 以及 `excludedResources`。
- [x] 2.2 在更新 AppRestore 时转换 `includedResources` 以及 `excludedResources`。
- [x] 2.3 对无法解析的 GVK 返回 400，确保不创建以及不更新 CR。
- [x] 2.4 历史 AppRestore 缺少 sourceCluster 时从最终 backupSource 指向的 AppBackup 获取映射集群；不可确定时拒绝请求，不使用目标集群映射。

## 3. 回归测试

- [x] 3.1 增加 `apps/v1/Deployment`、`v1/ConfigMap` 到 group-resource 的 RESTMapper 单元测试。
- [x] 3.2 增加资源清单接口返回 `deployments.apps` 的测试。
- [x] 3.3 增加 AppRestore 创建以及更新包含项、排除项兼容测试。
- [x] 3.4 增加无法解析资源的错误测试以及已是 group-resource 的兼容测试。
- [x] 3.5 使用隔离 Deployment 复现 `existingResourcePolicy=update`，记录恢复前后业务标签、`velero.io/backup-name`、`velero.io/restore-name`。

## 4. 接口文档和验证

- [x] 4.1 更新 Swagger/OpenAPI 三个受影响接口的请求、响应以及错误说明。
- [x] 4.2 调用 RunAPI `get_api_detail` 读取现有内容，追加新说明并保留原说明，更新三个接口的请求组件以及响应示例。
- [x] 4.3 更新本地接口证据和检查清单。
- [x] 4.4 运行目标 Go 测试、OpenAPI 校验、`git diff --check` 以及 OpenSpec 严格校验。
