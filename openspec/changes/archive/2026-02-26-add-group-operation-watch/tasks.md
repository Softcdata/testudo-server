## 1. 实现任务

- [x] 1.1 在 `internal/apis/disaster_group/v1/types.go` 中新增 DTO 和转换函数
  - [x] 1.1.1 新增 `DisasterOperationDTO` 结构体
  - [x] 1.1.2 新增 `StepStatusDTO` 结构体（对应 `status.steps[]`）
  - [x] 1.1.3 新增 `GroupStatusDTO` 结构体（对应 `status.groupStatus`）
  - [x] 1.1.4 新增 `LevelStatusDTO` 结构体（对应 `groupStatus.levelStatuses[]`）
  - [x] 1.1.5 实现 `ConvertToDisasterOperationDTO` 转换函数

- [x] 1.2 在 `internal/apis/disaster_group/v1/handler.go` 中实现 Watch 方法
  - [x] 1.2.1 实现 `watchGroupOperations`：按 groupName LabelSelector 过滤，无参数时监听所有
  - [x] 1.2.2 实现 `watchGroupOperation`：通过 FieldSelector 精确监听单个操作

- [x] 1.3 在 `internal/apis/disaster_group/v1/router.go` 中注册路由
  - [x] 1.3.1 注册 `GET /watch/groups/operations`
  - [x] 1.3.2 注册 `GET /watch/groups/operations/:operationName`

- [x] 1.4 编译验证（`go build ./...` 无错误）
