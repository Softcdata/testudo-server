# Tasks: 增加 Swagger 与 OpenAPI 支持

## 1. 清单与对账
- [x] 1.1 从 `internal/router/router.go` 与 `internal/apis/**/router.go` 整理 server 全量接口清单，记录方法、标准化路径、模块、handler、认证范围、WebSocket 属性。
- [x] 1.2 拉取 RunAPI 全量接口清单，记录项目、目录、接口 ID、方法、标准化路径、接口类型、已有详细说明状态。
- [x] 1.3 读取 OpenAPI 当前清单。首次实施时该清单为空，并记录为空的原因。
- [x] 1.4 使用 `METHOD + 标准化路径` 完成 server、RunAPI、OpenAPI 三方对账。
- [x] 1.5 输出差异清单，包含 server 缺失、RunAPI 缺失、OpenAPI 缺失、模块错位、WebSocket 类型不一致、历史裸路径。
- [x] 1.6 输出按模块组织的 Swagger 逐接口勾选清单。

## 2. OpenAPI 骨架
- [x] 2.1 新建 `openspec/specs/disaster-server-openapi.yaml`，版本固定为 OpenAPI 3.0.3。
- [x] 2.2 添加基础 `info`、`servers`、`tags`、`securitySchemes.bearerAuth`。
- [x] 2.3 添加通用 `Envelope`、`ErrorEnvelope`、`CollectionMeta`、`PaginationMeta`、`WatchEnvelope`、`WatchEventDTO` schema。
- [x] 2.4 按 server 清单写入全量 paths 空壳，并保留 WebSocket 标记。
- [x] 2.5 校验 OpenAPI YAML 可解析。

## 3. 逐接口调用链取证与 schema 补齐
- [ ] 3.1 按逐接口勾选清单顺序读取 server router、handler、request struct、校验、默认值、转换逻辑。
- [ ] 3.2 按逐接口勾选清单顺序读取 server DTO 与响应转换函数。
- [ ] 3.3 按逐接口勾选清单顺序读取 operator CRD Spec、Status、枚举、kubebuilder 校验。
- [ ] 3.4 按逐接口勾选清单顺序读取 operator controller，确认下层资源、状态机、失败 reason 与 message。
- [ ] 3.5 逐接口补齐 path、query、header、requestBody schema。
- [ ] 3.6 逐接口补齐 response schema 与成功响应示例。
- [ ] 3.7 逐接口补齐 HTTP 错误响应、业务 code、错误原因、触发条件。
- [ ] 3.8 逐接口补齐 `x-controlled-resources`、`x-operator-chain`、`x-async-failure-status`。
- [ ] 3.9 逐接口补齐五段 `description`。
- [ ] 3.10 单接口通过 OpenAPI 校验与 Swagger 渲染检查后，立即勾选 Swagger 逐接口清单。

## 4. Swagger UI 服务端支持
- [x] 4.1 确认现有配置结构，增加 `swagger.enabled` 配置项。
- [x] 4.2 生产配置设置 `swagger.enabled=false`。
- [ ] 4.3 发版验收配置设置 `swagger.enabled=true`。
- [x] 4.4 注册 `GET /openapi.yaml`。
- [x] 4.5 注册 `GET /openapi.json`。
- [x] 4.6 注册 `GET /swagger/`。
- [x] 4.7 确认 Swagger 文档路由不挂载业务 JWT 中间件。
- [x] 4.8 确认 Swagger UI 加载的是 `openspec/specs/disaster-server-openapi.yaml` 的同源转换结果。

## 5. 校验脚本与发版门禁
- [x] 5.1 增加 OpenAPI 3.0.3 schema 校验脚本。
- [x] 5.2 增加 OpenAPI YAML 到 JSON 转换校验。
- [x] 5.3 增加 `operationId` 全局唯一校验。
- [x] 5.4 增加 server 路由清单导出脚本。
- [ ] 5.5 增加 RunAPI 清单导出脚本。
- [x] 5.6 增加 server、RunAPI、OpenAPI 三方对账脚本。
- [x] 5.7 增加 WebSocket 接口扩展字段校验。
- [ ] 5.8 增加 Swagger UI 可访问性检查。
- [ ] 5.9 将全部校验接入发版检查命令。

## 6. 验收报告
- [ ] 6.1 输出 OpenAPI 全量接口清单。
- [ ] 6.2 输出 server、RunAPI、OpenAPI 三方差异为 0 的对账报告。
- [x] 6.3 输出 WebSocket 接口清单与消息 schema 检查结果。
- [ ] 6.4 输出未确认字段清单。所有接口勾选完成时该清单必须为空。
- [ ] 6.5 输出 Swagger UI 截图与访问路径。
- [x] 6.6 运行 `openspec validate add-swagger-openapi-support --strict` 并通过。
