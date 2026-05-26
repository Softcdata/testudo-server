## Context
`disaster-operator` 已提供平台 License 的签名校验、Kubernetes 指纹计算、Secret/ConfigMap 读写与集群额度判断。`disaster-server` 需要复用同一套 verifier，承担 API 前置门禁和前端展示接口职责。

## Goals / Non-Goals
- Goals: server 在创建 Cluster 前按创建前计数执行 License 门禁。
- Goals: server 提供 License 状态查询与安装接口。
- Goals: server 的判权逻辑只读取 License Secret 并重新计算当前指纹。
- Non-Goals: server 不作为最终可信边界，最终兜底仍由 operator webhook 和 reconciler 执行。
- Non-Goals: server 不改变 Apache-2.0 源码授权方式。

## Decisions
- Decision: License 展示 API 路径固定为 `GET /apis/v1/platform-license/status`，安装 API 路径固定为 `POST /apis/v1/platform-license/install`。这些接口挂载在现有 `/apis` 认证链路下。
- Decision: `POST /apis/cluster.testudo.softcdata.com/v1/clusters` 在创建 CR 前调用可信 verifier，使用所有未删除 Cluster 的创建前数量作为 `currentCount`。
- Decision: 状态接口可以读取 `disaster-platform-license-status` ConfigMap 作为展示缓存；当 ConfigMap 不存在时返回基于 Secret 实时计算的状态。所有额度判断必须调用 Secret 与指纹校验，不得依赖该 ConfigMap。
- Decision: License 安装接口写入 `disaster-platform-license` Secret，Secret 类型为 `testudo.softcdata.com/license`，数据键为 `license.lic`。
- Decision: server 本地或外部集群运行时优先使用 `rest.Config` 中的 CAData/CAFile 作为部署指纹 CA bundle；仅当该来源不可用时才读取 `license.caPath`。部署指纹计算失败必须返回 `LicenseEnvironmentInvalid`，不得包装为 License 内容无效。
- Decision: License 安装接口写入 Secret 后必须基于本次写入内容和 direct API reader 做实时评价，避免 controller-runtime cache 延迟导致安装响应返回旧的 `NoLicense` 状态。
- Decision: License 失败时返回统一 Envelope，HTTP 状态为 403，业务码为 `CodeForbidden`，`meta.reason` 使用稳定错误原因。

## Risks / Trade-offs
- Risk: server 依赖的 operator 版本未发布时无法解析 `pkg/license`。
  Mitigation: 开发期使用本地 `replace github.com/softcdata/testudo-operator => ../disaster-operator`，发布时更新为包含 `pkg/license` 的正式 operator 版本。
- Risk: status ConfigMap 被用户篡改导致页面展示不准。
  Mitigation: 创建门禁不读取该 ConfigMap；状态接口返回 `source` 字段标明状态来源。

## Migration Plan
1. 部署带 License gate 的 operator。
2. 部署带 License API 与前置门禁的 server。
3. 前端调用状态接口展示免费版或企业版权益。
4. 用户上传 License 后，server 写入 Secret，operator 刷新状态 ConfigMap。

## Open Questions
- 无。
