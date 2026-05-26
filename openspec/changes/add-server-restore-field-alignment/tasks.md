# Tasks: server 端恢复字段一致性

## 1. 规范与契约

- [x] 1.1 完成本提案的 OpenSpec 文档并通过评审
- [x] 1.2 与 operator 提案 `add-restore-policy-and-sc-mapping` 做字段清单对齐

## 2. DisasterInstance API 对齐

- [x] 2.1 扩展 `CreateDisasterInstanceRequest` 与 `UpdateDisasterInstanceRequest`，支持 `skipPodReadyCheck`
- [x] 2.2 扩展实例 DTO 回传 `spec.skipPodReadyCheck`
- [x] 2.3 扩展实例 API 透传 `spec.restorePolicy`
- [x] 2.4 补充实例创建、更新、查询单元测试
- [x] 2.5 创建/更新接口支持 `restorePolicy.modifierRulesText` 文本入参并解析为结构化规则
- [x] 2.6 定义并实现冲突规则：`modifierRules` 与 `modifierRulesText` 不一致时返回 400

## 3. 实例/组 Action API 对齐

- [x] 3.1 在实例 action 接口支持解析 `config.skipPodReadyCheck`
- [x] 3.2 在组 action 接口支持解析 `config.skipPodReadyCheck`
- [x] 3.3 实现优先级规则：`skipPodReadyCheck` 覆盖 `waitUntilReady`
- [x] 3.4 创建 `DisasterOperation` 时写入 `spec.skipPodReadyCheck` 与兼容字段 `spec.waitUntilReady`
- [x] 3.5 补充实例 action 与组 action 透传测试

## 4. AppRestore API 字段命名对齐

- [x] 4.1 在创建接口支持 `storageClassMapping`
- [x] 4.2 在更新接口支持 `storageClassMapping`
- [x] 4.3 保持 `scMapping` 兼容输入
- [x] 4.4 增加 `storageClassMapping` 与 `scMapping` 冲突校验
- [x] 4.5 补充 AppRestore 创建、更新单元测试

## 5. 回归与质量门禁

- [x] 5.1 回归验证旧请求：仅 `waitUntilReady` 的实例与组 action
- [x] 5.2 回归验证旧请求：仅 `scMapping` 的 AppRestore 创建与更新
- [x] 5.3 执行 `go test ./internal/apis/disaster_instance/v1 ./internal/apis/disaster_group/v1 ./internal/apis/app_restore/v1`
- [x] 5.4 执行 `openspec validate add-server-restore-field-alignment --strict`
