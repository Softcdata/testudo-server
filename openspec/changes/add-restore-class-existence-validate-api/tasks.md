# Tasks: 恢复 Class 存在性预检接口

## 1. OpenSpec 文档

- [x] 1.1 完成 proposal/spec/tasks 文档并通过评审
- [x] 1.2 与 operator 侧失败码语义做一致性核对

## 2. API 契约与路由

- [x] 2.1 在 `types.go` 新增预检请求 DTO 与响应 DTO
- [x] 2.2 在 `router.go` 注册 `POST /instances/:name/restore-classes/validate`
- [x] 2.3 在 handler 中新增预检处理函数

## 3. 解析与校验逻辑

- [x] 3.1 实现目标集群解析优先级：请求字段 > 实例状态 > 配置目标集群
- [x] 3.2 实现映射策略输入校验：`storageClassMapping` 与 `ingressClassMapping` 不能同时缺失
- [x] 3.3 查询目标集群 `StorageClass` 列表并生成存在性结果
- [x] 3.4 查询目标集群 `IngressClass` 列表并生成存在性结果
- [x] 3.5 按 strictTargetValidation 生成最终 `valid` 结果
- [x] 3.6 缺失时返回标准失败码 `StorageClassTargetNotFound` / `IngressClassTargetNotFound`
- [x] 3.7 输入冲突与非法字段返回 `ClassMappingInvalid`

## 4. 副作用与兼容性约束

- [x] 4.1 预检接口不创建任何 CR
- [x] 4.2 预检接口不更新任何 CR
- [x] 4.3 预检接口不触发恢复流程
- [x] 4.4 现有实例创建、更新、操作接口回归保持一致

## 5. 测试与验收

- [x] 5.1 单测：目标类全部存在时返回 `valid=true`
- [x] 5.2 单测：SC 缺失且 strict=true 返回 `StorageClassTargetNotFound`
- [x] 5.3 单测：IngressClass 缺失且 strict=true 返回 `IngressClassTargetNotFound`
- [x] 5.4 单测：缺失但 strict=false 返回 `valid=true` 且包含缺失列表
- [x] 5.5 单测：请求缺失映射时返回 `ClassMappingInvalid`
- [x] 5.6 单测：目标集群解析优先级正确
- [x] 5.7 执行 `go test ./internal/apis/disaster_instance/v1`
- [x] 5.8 执行 `openspec validate add-restore-class-existence-validate-api --strict`
