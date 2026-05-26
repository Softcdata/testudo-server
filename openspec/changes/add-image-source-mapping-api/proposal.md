# Change: 基础配置级镜像映射 API 与 Apipost 文档补齐

## Why
`disaster-operator` 在提案 `add-image-source-mapping` 中已明确将镜像映射入口下沉为 `DisasterConfig.spec.imageRewrite`。  
`disaster-server` 当前提案和部分接口描述仍停留在 `DisasterInstance.spec.imageRewrite`，与 operator 语义不一致，容易导致配置入口混乱。

## What Changes

### 1. Cluster API 新增镜像源字段
- `POST /clusters`（kubeconfig 与 token 场景）请求体新增 `imageSources`。
- `PATCH /clusters/:name` 支持更新 `imageSources`。
- `GET /clusters` 与 `GET /clusters/:name` 响应补充 `spec.imageSources` 字段说明。

### 2. DisasterConfig API 新增镜像映射字段
- `POST /configs` 请求体新增 `imageRewrite`。
- `PATCH /configs/:name` 支持更新 `imageRewrite`。
- `GET /configs` 与 `GET /configs/:name` 响应补充 `spec.imageRewrite` 字段说明。

### 3. DisasterInstance API 移除镜像映射配置入口
- `POST /instances` 与 `PUT /instances/:name` 不再接收 `imageRewrite` 作为配置入口。
- `GET /instances` 与 `GET /instances/:name` 的 `spec` 不再暴露 `imageRewrite` 字段。

### 4. Apipost 详细设计与接口说明同步
- 在 Apipost 项目新增“详细设计”文档，集中说明本提案新增字段、默认值与校验规则。
- 对相关接口描述补充新增字段说明与示例结构。

## Impact
- 受影响规范：
  - `openspec/specs/disaster_cluster/spec.md`
  - `openspec/specs/disaster_config/spec.md`
  - `openspec/specs/disaster_instance/spec.md`（调整为非配置入口）
- 受影响代码（实施阶段）：
  - `internal/apis/disaster_cluster/v1/*`
  - `internal/apis/disaster_config/v1/*`
  - `internal/apis/disaster_instance/v1/*`
- 受影响文档：
  - Apipost 项目 `容灾平台` 下 Cluster/Config/Instance 相关接口说明
