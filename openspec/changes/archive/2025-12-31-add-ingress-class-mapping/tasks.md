## 1. 实现 IngressClass 映射功能 (disaster-server)
- [x] 1.1 在 `internal/resourcemodifier/rule.go` 中添加 `IngressClassMapping` 函数
- [x] 1.2 在 `internal/apis/app_restore/v1/types.go` 中添加 `IngressClassMapping` 字段到 `CreateAppRestoreRequest`
- [x] 1.3 在 `internal/apis/app_restore/v1/types.go` 中添加 `IngressClassMapping` 字段到 `UpdateAppRestoreRequest`
- [x] 1.4 在 `internal/apis/app_restore/v1/handler.go` 的 `createAppRestore` 方法中应用 IngressClass 映射规则
- [x] 1.5 在 `internal/apis/app_restore/v1/handler.go` 的 `updateAppRestore` 方法中处理 IngressClass 映射更新
- [x] 1.6 验证编译通过

## 2. 更新文档
- [x] 2.1 在 `internal/resourcemodifier/readme.md` 中添加 IngressClass 映射示例
- [x] 2.2 添加 API 请求示例
- [x] 2.3 说明使用场景和注意事项

## 3. 测试
- [x] 3.1 编写单元测试验证 `IngressClassMapping` 函数生成正确的规则
- [x] 3.2 手动测试: 创建包含 Ingress 的备份
- [x] 3.3 手动测试: 使用 `ingressClassMapping` 参数恢复
- [x] 3.4 验证恢复后的 Ingress 资源 `ingressClassName` 已正确映射

## 4. 验证与发布
- [x] 4.1 在测试环境验证完整流程
- [x] 4.2 更新 API 文档 (如果有)
- [x] 4.3 提交代码并创建 PR
- [x] 4.4 归档变更提案
