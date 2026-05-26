# Tasks: scoped 资源过滤 API 契约补齐

## 1. AppBackup 字段补齐

- [x] 1.1 在 `AppBackupSpecDTO` 增加 `includedClusterScopedResources` 与 `excludedClusterScopedResources`
- [x] 1.2 在 `CreateAppBackupRequest` 与 `UpdateAppBackupRequest` 增加同名字段
- [x] 1.3 更新 `ConvertSpecToDTO`、`ToCRD`、`MergeToCRD` 的字段映射

## 2. DisasterInstance 字段对齐

- [x] 2.1 复核 `RestorePolicyRequest.ResourceSelection` 对新四字段的解析与序列化
- [x] 2.2 确保实例创建、更新、详情接口完整回显 scoped 四字段

## 3. 提交期校验与优先级

- [x] 3.1 新增 AppBackup 资源过滤冲突校验（交集冲突与通配符冲突）
- [x] 3.2 新增 DisasterInstance 资源过滤优先级判定（`includeClusterResources=true` 时忽略 scoped 四字段）
- [x] 3.3 在 `createAppBackup` 与 `updateAppBackup` 接入校验并返回 400
- [x] 3.4 在 `createInstance` 与 `updateInstance` 接入优先级与校验逻辑

## 4. 测试与文档

- [x] 4.1 增加 `app_backup` 类型转换测试与 handler 测试
- [x] 4.2 增加 `disaster_instance` 创建/更新优先级与校验测试
- [x] 4.3 更新 Apipost 与接口文档示例

## 5. 验证

- [x] 5.1 执行 `go test ./internal/apis/app_backup/v1 ./internal/apis/disaster_instance/v1`
- [x] 5.2 执行 `openspec validate add-scoped-resource-filter-api-support --strict`

## 备注

- server 使用 vendor 模式编译，本次同步更新了 vendor 中 `disaster-operator` 的 `RestoreResourceSelectionPolicy` 结构与 deepcopy，确保 scoped 四字段在当前仓可编译可测试。
