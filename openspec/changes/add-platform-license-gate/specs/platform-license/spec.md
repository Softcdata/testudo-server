## ADDED Requirements
### Requirement: 查询平台 License 状态
系统必须 (MUST) 提供 `GET /apis/v1/platform-license/status` 接口，用于返回平台 License 展示状态、当前集群数量与集群额度。

#### Scenario: 状态 ConfigMap 存在
- **GIVEN** `disaster-platform-license-status` ConfigMap 存在
- **WHEN** 客户端查询 License 状态
- **THEN** server 必须返回 ConfigMap 中的展示字段
- **AND** 响应必须包含当前未删除 Cluster 数量
- **AND** 响应 `source` 必须为 `statusConfigMap`

#### Scenario: 状态 ConfigMap 不存在
- **GIVEN** `disaster-platform-license-status` ConfigMap 不存在
- **WHEN** 客户端查询 License 状态
- **THEN** server 必须基于 License Secret 与当前指纹实时计算状态
- **AND** 响应 `source` 必须为 `liveEvaluation`

### Requirement: 安装平台 License
系统必须 (MUST) 提供 `POST /apis/v1/platform-license/install` 接口，用于安装平台 License。接口必须将 License 内容写入 `disaster-platform-license` Secret，Secret 类型必须为 `testudo.softcdata.com/license`，数据键必须为 `license.lic`。

#### Scenario: 安装有效格式 License
- **GIVEN** 请求体包含非空 `license` 字段
- **WHEN** 客户端安装 License
- **THEN** server 必须创建或更新 License Secret
- **AND** 响应必须返回基于当前 Secret 与指纹实时计算的 License 状态
- **AND** 响应不得返回安装前缓存的 `NoLicense` 状态

#### Scenario: License 内容错误优先于部署指纹错误
- **GIVEN** License Secret 已存在但内容为 malformed JSON schema、未知 `keyId` 或无效签名
- **AND** server 本地运行环境无法读取默认 ServiceAccount CA path
- **WHEN** 客户端查询或安装 License 后触发实时评价
- **THEN** server 必须优先返回 License 内容对应的稳定状态和 reason
- **AND** 不得用 CA path 或部署指纹计算错误掩盖 malformed、unknown key 或 invalid signature

#### Scenario: 部署指纹计算失败返回环境错误
- **GIVEN** License 内容、key、签名和时间窗口均有效
- **AND** server 无法从 kubeconfig/rest.Config CA bundle、显式 CA bundle 或 `license.caPath` 计算部署指纹
- **WHEN** 客户端查询或安装 License 后触发实时评价
- **THEN** server 必须返回 `state=Unknown`
- **AND** reason 必须为 `LicenseEnvironmentInvalid`
- **AND** message 必须包含部署指纹计算失败的上下文

#### Scenario: 请求体为空
- **GIVEN** 请求体缺失 `license` 字段或字段为空
- **WHEN** 客户端安装 License
- **THEN** server 必须返回 400
- **AND** 响应业务码必须为 `CodeBadRequest`

### Requirement: License API 统一响应
平台 License API 必须 (MUST) 使用统一 Envelope 响应结构，并在错误场景中返回稳定的 `meta.reason`。

#### Scenario: License 无效
- **GIVEN** 已安装 License 存在签名、产品、版本、公钥、时间窗口、指纹或额度错误
- **WHEN** License 影响创建集群或状态展示
- **THEN** server 必须返回或展示稳定 reason
- **AND** reason 必须为 `LicenseInvalid`、`LicenseExpired`、`LicenseFingerprintMismatch`、`LicenseEnvironmentInvalid`、`LicenseUnsupportedVersion`、`LicenseUnknownKey`、`LicenseProductMismatch` 或 `LicenseLimitExceeded` 之一
