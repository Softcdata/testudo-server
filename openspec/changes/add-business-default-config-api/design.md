# Design: 业务默认配置管理接口

## 背景
server 需要向前端提供一份稳定、可编辑、可解释的业务默认配置视图。
前端在创建实例、创建备份、恢复、容灾操作前读取这份配置，再把值写入对应请求体。
该能力只承担“读取当前默认值与说明、搜索字段、更新可编辑默认值”，不承载 operator 内部热加载状态。

## 关键决策

### D1. 采用单例配置资源
- 业务默认配置只存在一个逻辑对象。
- 服务端固定暴露一个配置快照，页面直接展示当前值。
- 更新请求仅作用于这一份快照，不引入按名称枚举多个配置实例的语义。

### D2. 返回结果必须包含变量说明
每个变量在响应中都必须返回以下信息：
- `key`：字段路径，形如 `backupRuntime.pollInterval`
- `name`：页面展示名称
- `description`：字段含义，必须直接给前端展示
- `value`：当前值
- `defaultValue`：默认值
- `dataType`：字段类型，形如 `duration`、`int`、`bool`、`string`、`enum`
- `editable`：是否允许页面修改
- `effectMode`：生效方式，取值固定为 `hot`、`restart`、`startup`
- `min` / `max` / `enumValues`：校验边界与候选项
- `groupKey` / `groupName`：分组信息，方便页面按功能区渲染

### D3. 字段列表接口采用集合响应规范
- 字段列表接口用于搜索、分页、排序和筛选。
- 集合响应必须使用统一响应信封。
- 响应必须包含 `items`、`meta.pagination`、`links.self`，并在适用时返回 `links.first`、`links.previous`、`links.next`、`links.last`。
- 列表项与快照字段使用同一份字段目录，避免前端展示值与搜索结果的字段说明分裂。

### D4. 搜索与筛选语义
- `keyword` 必须在 `key`、`name`、`description`、`value`、`groupKey`、`groupName` 上执行模糊匹配。
- `groupKey`、`editable`、`effectMode` 使用精确匹配。
- 搜索结果按字段条目返回，前端可以按 `groupKey` 二次分组。
- 当未提供任何过滤条件时，列表接口返回全部字段。

### D4.1 前端传参字段值目录
- 前端传参字段值目录独立于 operator 运行时配置字段目录。
- 该接口返回前端应读取的默认配置 `key` 与对应 `value`，用于前端在创建备份、恢复、实例、演练或发起容灾操作时填入对应请求字段。
- 响应条目必须包含 `resourceKind`、`specPath`、`requestPath`、`serverSupported`、`keySegments`、`requestPathSegments`、`specPathSegments`、`apiUsages` 等映射元数据。
- 响应 `data.fieldMap` 必须以字段 `key` 为 map key，便于前端直接通过 `fieldMap["operation.timeoutMinutes"]` 获取字段层级和接口用途。
- 每个 `key` 应写入哪个 server 接口、哪个请求字段、最终对应哪个 CRD 字段，以接口响应体为准；RunAPI 文档只作为人工说明与示例。
- `keyword` 和 `q` 对 `key`、`name`、`description`、`value`、`resourceKind`、`requestPath`、`specPath` 和接口用途执行大小写不敏感包含匹配。
- 该接口不支持 `resourceKind`、`configGroupKey`、`dataType`、`editable`、`serverSupported`、`operation` 等映射维度筛选；需要查看映射时读取响应字段本身。

### D5. 更新语义采用部分提交与合并写入
- `PATCH` 请求只携带变化字段。
- 未出现在请求体中的字段保持当前值不变。
- 请求体中的字段必须落在可编辑目录内。
- 服务端在持久化前完成类型校验、范围校验、枚举校验与必填校验。

### D6. 说明文本与字段目录保持单一来源
- 字段说明、默认值、类型与边界从同一份字段目录生成。
- 前端不自行拼接说明文本。
- 后端不允许仅改值不改说明目录，否则页面会出现含义漂移。

### D7. 底层存储可以复用现有 settings 文档
- 外部契约只暴露业务默认配置资源。
- 内部实现可以复用现有 settings 文档、ConfigMap 以及等价存储。
- 如果未来存储层调整，只要对外响应结构保持不变，前端无需改造。

## 接口草案

### 查询当前配置
- `GET /api/v1/business-default-config`

返回内容建议包含：
- `schemaVersion`
- `updatedAt`
- `updatedBy`
- `groups[]`
- `groups[].fields[]`

### 搜索字段列表
- `GET /api/v1/business-default-config/fields`

查询参数建议包含：
- `keyword`：模糊搜索字段
- `groupKey`：按分组筛选
- `editable`：按可编辑性筛选
- `effectMode`：按生效方式筛选
- `page`、`limit`：标准分页
- `sort`、`order`：标准排序

返回内容建议使用集合响应，字段条目包含：
- `key`
- `name`
- `description`
- `value`
- `defaultValue`
- `dataType`
- `editable`
- `effectMode`
- `groupKey`
- `groupName`

### 搜索前端传参字段值目录
- `GET /api/v1/business-default-config/frontend-fields`

查询参数建议包含：
- `keyword`：模糊搜索字段
- `q`：`keyword` 的兼容别名
- `page`、`limit`、`sort`、`order`：标准分页与排序
- `sort` 当前支持 `key` 与 `value`

返回内容使用集合响应，字段条目包含：
- `key`
- `value`
- `fieldMap`
- `requestPathSegments`
- `specPathSegments`
- `apiUsages`

RunAPI 文档必须维护每个 `key` 到业务接口字段的映射表，至少包含：
- `key`
- 前端提交的 server 接口
- 请求字段路径
- 最终 CRD 字段路径
- 当前 server 是否已支持直接透传
- 备注

### 更新当前配置
- `PATCH /api/v1/business-default-config`

请求体建议采用分组结构，示例：

```json
{
  "backupRuntime": {
    "pollInterval": "10s"
  },
  "restoreRuntime": {
    "retryBackoff": "15s"
  }
}
```

### 页面展示模型
前端页面直接渲染分组列表，每个字段展示：
- 当前值
- 默认值
- 字段含义
- 是否可编辑
- 生效方式
- 校验范围

## 更新校验与错误
- 字段不存在时返回 400，并标明具体字段路径。
- 字段属于只读目录时返回 400，并标明原因。
- 类型不匹配时返回 400，并标明期望类型。
- 取值超出边界时返回 400，并标明最小值与最大值。
- 更新成功后返回最新配置快照。

## 与 operator 的关系
- 字段目录与含义需要与 `disaster-operator` 的 `add-dynamic-runtime-configuration` 保持一致。
- server 负责给前端提供业务默认值和说明。
- operator 负责消费最终写入的业务参数。
