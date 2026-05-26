## 1. 修正 instance-picker（labels → description）

- [x] 1.1 修改 `internal/apis/disaster_group/v1/types.go`
  - [x] 将 `InstancePickerItemDTO.Labels map[string]string` 改为 `Description string`
- [x] 1.2 修改 `internal/apis/disaster_group/v1/handler.go`
  - [x] `instancePicker`：从 `inst.Annotations["testudo.softcdata.com/description"]` 读取 Description
  - [x] `instancePickerMatchKeyword`：参数改为 `description string`，匹配范围改为 name / namespace / description

## 2. 新增 listGroupInstances 接口

- [x] 2.1 新增 `GroupMemberInstanceDTO` 到 `internal/apis/disaster_group/v1/types.go`
  - [x] 字段：`Name`、`Description`、`Namespace`、`FsmState`
- [x] 2.2 实现 `listGroupInstances` handler
  - [x] 获取 DisasterGroup（不存在返回 404）
  - [x] 展平 spec.levels，去重收集实例名称
  - [x] 循环查询各 DisasterInstance，构建 GroupMemberInstanceDTO
  - [x] keyword 模糊过滤 / status 精确过滤
  - [x] transport.Paginate + transport.BuildCollectionResponse 标准响应
- [x] 2.3 注册 `GET /groups/:name/instances` 路由

## 3. 验证

- [x] 3.1 `go build ./...` 编译通过
