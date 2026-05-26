# 任务列表

## Operator 开发
- [ ] **StorageRepository CRD 更新** @operator
  - [ ] 添加 `Reason`, `Message` 字段到 Status
  - [ ] 定义 `Deleting` 状态常量
- [ ] **StorageRepository Controller 实现** @operator
  - [ ] 实现 `handleDelete` 逻辑
  - [ ] 添加 Finalizer `testudo.softcdata.com/storage-finalizer` 管理
  - [ ] 实现依赖检查 (DisasterConfig, AppBackup)
- [ ] **DisasterPolicy CRD 更新** @operator
  - [ ] 添加 `Reason`, `Message` 字段到 Status
  - [ ] 定义 `Deleting` 状态常量
- [ ] **DisasterPolicy Controller 实现** @operator
  - [ ] 实现 `handleDelete` 逻辑
  - [ ] 添加 Finalizer `testudo.softcdata.com/policy-finalizer` 管理
  - [ ] 实现依赖检查 (DisasterConfig, AppBackup)

## Server 开发
- [ ] **移除 Policy 删除拦截** @server
  - [ ] 修改 `internal/apis/disaster_policy/v1/handler.go`，移除 `deletePolicy` 中的依赖检查代码
- [ ] **验证 Storage 删除接口** @server
  - [ ] 确保 `deleteStorage` 逻辑简洁，只负责调用 K8s Delete

## 测试与验证
- [ ] 验证 Storage 有依赖时无法删除，状态变为 Deleting @qa
- [ ] 验证 Policy 有依赖时无法删除，状态变为 Deleting @qa
- [ ] 验证无依赖时可正常删除 @qa
