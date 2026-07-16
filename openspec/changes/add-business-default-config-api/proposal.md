# Change: 新增业务默认配置管理接口

## Why
前端需要一个稳定入口来读取当前业务默认值、字段含义、适用范围以及生效方式，并且在同一页面修改这些参数。
当前 server 只有通用 settings CRUD，缺少面向业务默认配置的强类型契约，前端必须自行拼装字段，容易遗漏说明、范围与生效方式，也容易与 operator 侧字段表脱节。
此外，配置管理页面需要支持按关键字搜索、按分组筛选以及分页浏览字段列表，不能只靠一次性拉全量数据。

## What Changes
- 新增单例业务默认配置快照查询接口，返回完整分组视图。
- 新增业务默认配置字段列表接口，返回可搜索、可分页、可排序的字段集合。
- 新增前端传参字段值接口，返回前端应从业务默认配置读取并写入业务请求体/CRD `spec` 的参数 `key`、`value`、字段层级、接口用途以及按 key 索引的字段 map，支持搜索、分页与排序。
- 新增单例业务默认配置更新接口，支持前端按字段提交修改。
- 每个变量必须返回 `key`、`name`、`description`、`value`、`defaultValue`、`dataType`、`editable`、`effectMode`、`min`、`max`、`enumValues` 等元数据。
- 快照接口以分组结构返回，前端可以直接按功能区渲染。
- 字段列表接口采用集合响应规范，支持 `keyword` 搜索、`groupKey` 筛选、`editable` 筛选、`effectMode` 筛选以及标准分页。
- 前端传参字段值接口采用集合响应规范，支持 `keyword`、`q` 搜索，并支持标准分页与按 `key`、`value` 排序。
- 前端传参字段值接口保留 `key` 和 `value`，并返回 `fieldMap`、`requestPathSegments`、`specPathSegments`、`apiUsages`，前端可直接定位每个字段应写入哪个 server 接口、哪个请求字段、最终对应哪个 CRD 字段。
- 更新接口只接受可编辑字段，服务端在写入前完成类型、范围、枚举与必填校验。
- 底层存储可以复用现有 settings 文档，但对外契约单独定义为业务默认配置资源。

## Impact
- 受影响规范：
  - `business_default_config`
  - `api-standards`（集合响应、分页、过滤、排序、响应信封与错误语义需要补充说明）
- 受影响代码：
  - `internal/apis/business_default_config/v1/*`
  - `internal/apis/system_settings/v1/*`（若复用现有 settings 文档作为底层存储）
  - `internal/router/router.go`
  - `openspec/specs/*`
- 跨仓影响：
  - `cluster-disaster-web`：新增业务默认配置页面与编辑态表单
  - `disaster-operator`：字段目录与含义需要与 `add-dynamic-runtime-configuration` 保持一致

## Non-Goals
- 不修改 operator 内部 runtime config 的热加载语义。
- 不将通用 settings CRUD 直接暴露给业务默认配置页面。
- 不在本变更中实现前端页面。
