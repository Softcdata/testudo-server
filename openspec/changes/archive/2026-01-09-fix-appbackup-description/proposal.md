# 提案：修复 AppBackup 描述字段处理逻辑

## 问题描述
用户反馈在 `AppBackup` 的入参中添加了描述字段，但似乎没有生效。经代码审查发现：

1.  **Update 逻辑缺陷**: 在 `UpdateAppBackupRequest` 中，`Description` 字段定义为 `string` 类型。在 `handler.go` 中存在逻辑矛盾：
    ```go
    if req.Description != "" {
        // ...
        if req.Description == "" { // 永远不可达
            delete(...)
        }
    }
    ```
    这导致如果不传递 `description` (默认为空串) 会跳过更新（正确），但如果用户想显式清空描述传空串，也会被跳过（错误）。且无法区分“未传”和“传空”。

2.  **Create 逻辑**: 虽然逻辑看似正确，但建议再次核查。

## 解决方案

### 1. 修改 DTO
将 `UpdateAppBackupRequest` 中的 `Description` 字段改为指针类型 `*string`，以便区分零值（未传）和空值（清空）。

```go
type UpdateAppBackupRequest struct {
    // ...
    Description *string `json:"description,omitempty"`
    // ...
}
```

### 2. 修复 Handler
修改 `internal/apis/app_backup/v1/handler.go` 中的 `updateAppBackup` 方法：

```go
if req.Description != nil {
    if existing.Annotations == nil {
        existing.Annotations = make(map[string]string)
    }
    if *req.Description == "" {
        delete(existing.Annotations, AppBackupDescriptionAnnotation)
    } else {
        existing.Annotations[AppBackupDescriptionAnnotation] = *req.Description
    }
}
```

## 验证计划
1.  启动 Server。
2.  创建带 Description 的 AppBackup -> 验证成功。
3.  更新 AppBackup 修改 Description -> 验证成功。
4.  更新 AppBackup 清空 Description (`""`) -> 验证成功。
