# Tasks: 标签过滤器统一降级为内存模糊搜索

## 1. 底层工具改进

- [x] 1.1 修改 `internal/transport/query.go`
    - 更新 `BuildLabelSelector` 签名，返回 `(labels.Selector, map[string]string)`。
    - 将失败的过滤器搜集到返回的 map 中。
- [x] 1.2 增加内存过滤工具函数
    - 在 `transport` 包下增加 `FuzzyFilter[T any](items []T, filters map[string]string, getter func(T, string) string) []T`。
    - 支持 `*` 语法的匹配逻辑。

## 2. API 适配

- [x] 2.1 更新 `internal/apis/app_backup/v1/handler.go`
    - 在 `appBackups` 方法中接收 `rejectedFilters`。
    - 调用 `FuzzyFilter` 进行二次过滤。
- [x] 2.2 更新 `internal/apis/disaster_cluster/v1/handler.go` (示例)
    - 同样适配新的匹配逻辑。

## 3. 规范与验证

- [x] 3.1 更新 `openspec/specs/api-standards/spec.md`
    - 增加“非法标签值降级为模糊匹配”的场景描述。
- [x] 3.2 运行验证脚本
    - 使用 `app-*` 进行查询，确认不再返回不匹配的结果。

## 3. 规范更新

- [x] 3.1 修改 `openspec/specs/api-standards/spec.md`
    - 明确“所有过滤参数默认为模糊匹配”的 API 原则。
