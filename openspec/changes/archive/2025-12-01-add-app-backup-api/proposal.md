# Change: 添加 AppBackup API 接口

## Why
目前 `disaster-server` 缺少对 `AppBackup` 资源的管理接口。为了支持前端对应用备份策略的管理，需要实现相应的 RESTful API。

## What Changes
- 新增 `internal/apis/app_backup/v1` 包。
- 实现 `AppBackupHandler`，包含标准的 CRUD 和 Watch 操作。
- 编写对应的单元测试，必须覆盖所有实现的接口方法（List, Get, Create, Update, Delete, Watch），并使用 `testify` 进行断言。
- 导出 Postman 接口定义文件，便于前端调试。
- 在 `internal/router/router.go` 中注册新的 Handler。

## Impact
- 受影响的规范：`app-backup` (新增)
- 受影响的代码：`internal/apis/app_backup/`, `internal/router/router.go`
