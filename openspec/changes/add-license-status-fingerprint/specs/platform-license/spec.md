## ADDED Requirements

### Requirement: License 状态接口必须返回当前集群指纹

系统必须 (MUST) 在 `GET /apis/v1/platform-license/status` 的 `data` 中返回当前管理面集群指纹信息。该指纹必须由 server 在处理本次请求期间通过 `KubernetesStore.Fingerprint(ctx)` 计算得到，不得直接信任 `disaster-platform-license-status` ConfigMap 中的展示缓存字段。

#### Scenario: 无 License 时返回指纹

- **GIVEN** 当前不存在 `disaster-platform-license` Secret
- **WHEN** 客户端请求 `GET /apis/v1/platform-license/status`
- **THEN** 响应 HTTP 状态必须为 200
- **AND** `data.state` 必须为 `Free`
- **AND** `data.reason` 必须为 `NoLicense`
- **AND** `data.fingerprint` 必须为当前管理面集群指纹
- **AND** `data.fingerprintVersion` 必须为 `k8s-v1`
- **AND** `data.fingerprintRequest.product` 必须为 `disaster-platform`
- **AND** `data.fingerprintRequest.fingerprint` 必须等于 `data.fingerprint`
- **AND** `data.fingerprintRequest.fingerprintVersion` 必须等于 `data.fingerprintVersion`

#### Scenario: 状态 ConfigMap 存在时返回当前指纹

- **GIVEN** `disaster-platform-license-status` ConfigMap 存在
- **WHEN** 客户端请求 `GET /apis/v1/platform-license/status`
- **THEN** server 必须读取展示状态并使用当前未删除 Cluster 数量覆盖缓存中的 `clusterCount`
- **AND** server 必须在写出响应以前计算当前管理面集群指纹
- **AND** `data.source` 必须为 `statusConfigMap`
- **AND** `data.fingerprint` 必须为当前管理面集群指纹
- **AND** `data.fingerprintRequest.namespace` 必须为 server 生效的 License Namespace

#### Scenario: License 内容错误时仍返回当前指纹

- **GIVEN** `disaster-platform-license` Secret 已存在且内容格式错误
- **WHEN** 客户端请求 `GET /apis/v1/platform-license/status`
- **THEN** 响应 HTTP 状态必须为 200
- **AND** `data.state` 必须为 `Malformed`
- **AND** `data.reason` 必须为 `LicenseInvalid`
- **AND** `data.fingerprint` 必须为当前管理面集群指纹
- **AND** `data.fingerprintRequest.generatedAt` 必须为本次请求期间生成的 UTC RFC3339 时间

#### Scenario: 无优先 License 内容错误且当前指纹生成失败

- **GIVEN** server 无法读取生成当前指纹所需的 Kubernetes 环境信息
- **AND** 当前状态不存在已分类的 License 内容错误
- **WHEN** 客户端请求 `GET /apis/v1/platform-license/status`
- **THEN** 响应 HTTP 状态必须为 200
- **AND** `data.state` 必须为 `Unknown`
- **AND** `data.reason` 必须为 `LicenseEnvironmentInvalid`
- **AND** `data.fingerprint` 必须为空字符串
- **AND** `data.fingerprintVersion` 必须为 `k8s-v1`
- **AND** 响应不得包含 `data.fingerprintRequest`

#### Scenario: License 内容错误且当前指纹生成失败

- **GIVEN** `disaster-platform-license` Secret 已存在且内容格式错误
- **AND** server 无法读取生成当前指纹所需的 Kubernetes 环境信息
- **WHEN** 客户端请求 `GET /apis/v1/platform-license/status`
- **THEN** 响应 HTTP 状态必须为 200
- **AND** `data.state` 必须为 `Malformed`
- **AND** `data.reason` 必须为 `LicenseInvalid`
- **AND** `data.fingerprint` 必须为空字符串
- **AND** `data.fingerprintVersion` 必须为 `k8s-v1`
- **AND** 响应不得包含 `data.fingerprintRequest`

### Requirement: License 状态响应必须包含可签发请求对象

当当前指纹生成成功时，系统必须 (MUST) 在 `GET /apis/v1/platform-license/status` 的 `data.fingerprintRequest` 中返回可用于签发 License 的请求对象。请求对象字段必须包含 `product`、`fingerprintVersion`、`fingerprint`、`namespace`、`generatedAt`。

#### Scenario: 前端复制签发请求

- **GIVEN** 当前指纹生成成功
- **WHEN** 客户端请求 `GET /apis/v1/platform-license/status`
- **THEN** `data.fingerprintRequest.product` 必须为 `disaster-platform`
- **AND** `data.fingerprintRequest.fingerprintVersion` 必须为 `k8s-v1`
- **AND** `data.fingerprintRequest.fingerprint` 必须为当前管理面集群指纹
- **AND** `data.fingerprintRequest.namespace` 必须为 server 生效的 License Namespace
- **AND** `data.fingerprintRequest.generatedAt` 必须为 UTC RFC3339 时间
