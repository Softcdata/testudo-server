# Tasks: 按集群返回恢复 Class 列表接口

## 1. OpenSpec 文档

- [x] 1.1 完成 proposal/spec/tasks 文档并通过评审
- [x] 1.2 与现有 `restore-classes/validate` 职责边界做一致性核对

## 2. API 契约与路由

- [x] 2.1 在 `types.go` 新增恢复 Class 列表响应 DTO
- [x] 2.2 在 `router.go` 注册 `GET /clusters/:name/restore-classes`
- [x] 2.3 在 `handler.go` 新增列表查询处理函数
- [x] 2.4 成功/失败响应对齐统一 `Envelope` 与双层错误码规范

## 3. 列表查询逻辑

- [x] 3.1 按 `:name` 解析目标集群并构建目标集群客户端
- [x] 3.2 查询目标集群 `StorageClass` 列表并提取 `name` 与 `isDefault`
- [x] 3.3 查询目标集群 `IngressClass` 列表并提取 `name` 与 `isDefault`
- [x] 3.4 对两个列表按 `name` 升序排序后返回
- [x] 3.5 约束该接口为纯读取语义，不创建、不更新、不删除业务 CR

## 4. 错误语义

- [x] 4.1 集群不存在时返回 404
- [x] 4.2 目标集群客户端构建失败时返回可读错误
- [x] 4.3 目标集群资源列表查询失败时返回可读错误
- [x] 4.4 集群不存在错误返回业务码 `CodeNotFound`
- [x] 4.5 参数或请求不合法返回业务码 `CodeBadRequest`

## 5. 测试与验收

- [x] 5.1 单测：查询成功返回 `storageClasses` 与 `ingressClasses`
- [x] 5.2 单测：`storageClasses` 与 `ingressClasses` 按名称升序
- [x] 5.3 单测：集群不存在返回 404
- [x] 5.4 单测：目标集群客户端构建失败返回错误
- [x] 5.5 单测：列表接口调用后不触发业务资源变更
- [x] 5.6 执行 `go test ./internal/apis/disaster_cluster/v1`
- [x] 5.7 执行 `openspec validate add-cluster-restore-class-list-api --strict`
