## 1. 实现任务

- [x] 1.1 在 `internal/apis/disaster_group/v1/types.go` 中新增 `InstancePickerItemDTO` 结构体
- [x] 1.2 在 `internal/apis/disaster_group/v1/handler.go` 中实现 `instancePicker` 处理函数
  - [x] 1.2.1 通过 `DisasterClient.DisasterV1().DisasterInstances("")` 列出所有命名空间的实例
  - [x] 1.2.2 实现 `keyword` 模糊过滤逻辑（Contains，不区分大小写，匹配 name / namespace / labels values）
  - [x] 1.2.3 实现 `status` 精确过滤逻辑（与 keyword 为 AND 关系）
  - [x] 1.2.4 将 CRD 对象转换为 `InstancePickerItemDTO`
  - [x] 1.2.5 调用 `transport.Paginate` 实现分页
  - [x] 1.2.6 调用 `transport.BuildCollectionResponse` + `transport.WriteSuccess` 返回标准响应
- [x] 1.3 在 `internal/apis/disaster_group/v1/router.go` 中注册新路由 `GET /groups/instance-picker`
- [x] 1.4 验证接口，确认：
  - [x] 编译通过（`go build ./...` 无错误）
  - [x] 无参数时返回所有实例
  - [x] keyword 过滤生效
  - [x] status 过滤生效
  - [x] 分页元数据正确
