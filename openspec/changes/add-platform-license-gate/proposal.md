# Change: 增加平台开源许可门禁 API

## Why
平台开源版本需要在无有效企业 License 时限制最多创建 2 个集群。`disaster-server` 需要在 API 层提供前置拦截、License 上传与状态展示，避免用户在页面或 API 中先看到创建成功再被 operator 拒绝。

## What Changes
- 在 `POST /apis/cluster.testudo.softcdata.com/v1/clusters` 创建前读取可信 License Secret、重新计算平台指纹并校验权益，不信任展示用 ConfigMap。
- 新增 `GET /apis/v1/platform-license/status`，返回 License 展示状态、当前集群数量与额度信息。
- 新增 `POST /apis/v1/platform-license/install`，将上传的 License 写入固定 Secret，供 operator 与 server 共同校验。
- 补充 Swagger/OpenAPI 与 RunAPI 文档，包含新增接口和集群创建接口的 License 错误说明。

## Impact
- 受影响的规范：`disaster_cluster`、`platform-license`
- 受影响的代码：`internal/apis/disaster_cluster/v1`、`internal/apis/platform_license/v1`、`internal/router/router.go`、`configs`
- 受影响的文档：`openspec/specs/disaster-server-openapi.yaml`、RunAPI 项目 `5650333c5c52000`
