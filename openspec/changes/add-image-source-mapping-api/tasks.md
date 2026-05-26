# Tasks: 基础配置级镜像映射 API 与 Apipost 文档补齐

## 1. 详细设计
- [x] 1.1 在 `design.md` 固化 `imageSources` 与 `imageRewrite` 字段结构
- [x] 1.2 明确 `unmatchedPolicy` 默认值 `Fail` 与可选值 `Keep|Fail`
- [x] 1.3 明确 `applyTo` 可选值 `resourceSync|drill`
- [x] 1.4 明确接口级校验规则（别名存在性、重复映射、空值约束）

## 2. Cluster 接口（Apipost）
- [x] 2.1 更新 `POST /clusters`（kubeconfig）新增 `imageSources` 字段说明
- [x] 2.2 更新 `POST /clusters`（token）新增 `imageSources` 字段说明
- [x] 2.3 更新 `PATCH /clusters/:name` 新增 `imageSources` 更新语义说明
- [x] 2.4 更新 `GET /clusters` 与 `GET /clusters/:name` 响应字段说明（`spec.imageSources`）

## 3. DisasterConfig 接口（Apipost）
- [x] 3.1 更新 `POST /configs` 新增 `imageRewrite` 字段说明
- [x] 3.2 更新 `PATCH /configs/:name` 新增 `imageRewrite` 字段说明
- [x] 3.3 更新 `GET /configs` 与 `GET /configs/:name` 响应字段说明（`spec.imageRewrite`）

## 4. DisasterInstance 接口（Apipost）
- [x] 4.1 更新 `POST /instances` 与 `PUT /instances/:name`，明确 `imageRewrite` 不再作为配置入口
- [x] 4.2 更新 `GET /instances` 与 `GET /instances/:name` 响应字段说明（移除 `spec.imageRewrite`）

## 5. Apipost 详细设计文档
- [x] 5.1 新增“镜像源映射字段详细设计”文档
- [x] 5.2 文档包含新增字段、默认值、枚举、校验与示例 JSON

## 6. 验证
- [x] 6.1 执行 `openspec validate add-image-source-mapping-api --strict`

## 7. server 实现（补充）
- [x] 7.1 对齐 operator 依赖类型，补齐 `ClusterSpec.imageSources` 与 `DisasterConfigSpec.imageRewrite` 的 vendored 字段
- [x] 7.2 实现 Cluster API 对 `imageSources` 的创建/更新/查询透传
- [x] 7.3 实现 Cluster API 对 `imageSources` 的字段校验（非空、别名唯一）
- [x] 7.4 实现 DisasterConfig API 对 `imageRewrite` 的创建/更新/查询透传
- [x] 7.5 实现 DisasterConfig API 对 `imageRewrite` 的枚举/默认值/重复映射校验
- [x] 7.6 实现 DisasterConfig API 映射别名存在性校验（基于 source/target cluster）
- [x] 7.7 调整 DisasterInstance API，移除 `imageRewrite` 配置入口
- [x] 7.8 新增并通过 Config/Instance 接口定向单测
