# Tasks: 业务默认配置管理接口

## 1. 规格与字段目录
- [x] 1.1 固化业务默认配置的分组目录、字段路径、字段说明与默认值。
- [x] 1.2 约定每个字段的 `dataType`、`effectMode`、`editable`、`min`、`max`、`enumValues`。
- [x] 1.3 明确查询接口返回的分组结构与更新接口的合并语义。

## 2. 服务端实现
- [x] 2.1 新增 `internal/apis/business_default_config/v1` handler、router、字段目录、ConfigMap 存储和校验逻辑。
- [x] 2.2 注册 `/api/v1/business-default-config`、`/api/v1/business-default-config/fields` 以及 `/apis/v1` 兼容前缀。
- [x] 2.3 实现 `GET /api/v1/business-default-config` 快照响应。
- [x] 2.4 实现 `GET /api/v1/business-default-config/fields` 搜索、筛选、排序和分页。
- [x] 2.5 实现 `GET /api/v1/business-default-config/frontend-fields` 前端传参字段目录搜索、排序和分页，并返回 `fieldMap`、字段层级与接口用途。
- [x] 2.6 实现 `PATCH /api/v1/business-default-config` 分组结构和扁平 key 部分更新。
- [x] 2.7 实现未知字段、只读字段、类型错误、范围错误、跨字段关系错误的 400 响应和字段级 `meta`。

## 3. 存储与兼容
- [x] 3.1 使用管理命名空间 ConfigMap `disaster-business-default-config` 作为单例存储。
- [x] 3.2 保持通用 `system-settings` CRUD 与业务默认配置 API 的外部契约隔离。
- [x] 3.3 对齐 disaster-operator runtime config 字段目录、默认值和校验边界。

## 4. 文档与 RunAPI
- [x] 4.1 编写 OpenSpec 变更规范。
- [x] 4.2 更新 `openspec/specs/disaster-server-openapi.yaml`，补齐 schema、参数、响应、错误和兼容前缀说明。
- [x] 4.3 通过 Apipost MCP 新增 RunAPI `业务默认配置` 文件夹和 6 个接口目标。
- [x] 4.4 通过 Apipost MCP 新增 RunAPI 前端传参字段目录 2 个接口目标。
- [x] 4.5 更新本地 RunAPI evidence/checklist 与 Swagger/OpenAPI checklist。

## 5. 验证
- [x] 5.1 运行 `go test ./internal/apis/business_default_config/v1 ./internal/router`。
- [x] 5.2 运行 `go run ./tools/openapi validate --spec openspec/specs/disaster-server-openapi.yaml`。
- [x] 5.3 运行 `openspec validate add-business-default-config-api --strict`。
- [x] 5.4 运行 `git diff --check`。
