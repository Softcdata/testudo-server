# 任务：修复 AppBackup 描述字段

## 1. 接口定义 (Type Definition)
- [x] 1.1 **Update DTO**: 修改 `internal/apis/app_backup/v1/types.go`，将 `UpdateAppBackupRequest` 的 `Description` 字段改为 `*string`。

## 2. 逻辑修复 (Logic Fix)
- [x] 2.1 **Update Handler**: 修改 `internal/apis/app_backup/v1/handler.go` 中的 `updateAppBackup` 方法，正确处理指针类型的 `Description`。

## 3. 验证 (Validation)
- [ ] 3.1 **Manual Test**: 验证创建和更新（包括清空）描述字段的功能。
