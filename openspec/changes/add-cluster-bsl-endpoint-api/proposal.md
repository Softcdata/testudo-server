# 变更提案：通过集群 API 配置专用 BSL endpoint

## 背景

Operator 已支持从 `Cluster.spec.veleroInstall.bslEndpoint` 读取物理集群专用的 BSL S3/MinIO 入口，但 server 的 Cluster API 尚未暴露该字段。通过 API 创建或修改集群时，字段会被忽略，导致同城和异地集群无法保存不同的访问入口。

## 变更内容

- Cluster 创建请求支持 `veleroInstall.bslEndpoint`。
- Cluster 详情、列表、watch、创建和 PATCH 响应返回非敏感的 `bslEndpoint`。
- PATCH 支持设置、保持和清除专用 endpoint。
- server 按 Operator 相同规则校验 endpoint：必须是带 host 的绝对 `http` 或 `https` URL，禁止 userinfo、query 和 fragment。
- 同步更新 OpenAPI、RunAPI 本地证据和 Cluster handler 单元测试。

## 语义

- 创建时未传或传空值表示不配置专用 endpoint，使用 Operator 默认 endpoint。
- PATCH 未传 `bslEndpoint` 表示保持当前值；传非空值表示覆盖；传空字符串表示清除。
- `bslEndpoint` 与 `imageRegistry`、registry credential 独立。清空 `imageRegistry` 时保留已有专用 endpoint；只有所有 VeleroInstall 字段都为空时才清除整段配置。

## 影响范围

- `internal/apis/disaster_cluster/v1/types.go`
- `internal/apis/disaster_cluster/v1/handler.go`
- `internal/apis/disaster_cluster/v1/handler_test.go`
- `openspec/specs/disaster-server-openapi.yaml`
- RunAPI/Apipost 本地接口证据
