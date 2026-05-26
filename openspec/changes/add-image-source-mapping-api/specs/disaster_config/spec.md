## ADDED Requirements

### Requirement: 基础配置接口必须支持镜像映射配置字段
系统必须 (MUST) 在 DisasterConfig 的创建、更新与查询接口中暴露 `imageRewrite` 字段，用于定义容灾恢复路径中的镜像替换策略。

#### Scenario: 创建基础配置时提交镜像映射配置
- **WHEN** 客户端调用 `POST /apis/disasterconfigs.testudo.softcdata.com/v1/configs`
- **AND** 请求体包含 `imageRewrite`
- **THEN** 服务端必须校验并持久化 `imageRewrite`
- **AND** 当 `imageRewrite.enabled=true` 时，`mappings` 必须至少包含 1 条映射

#### Scenario: 更新基础配置时校验映射字段
- **WHEN** 客户端调用 `PATCH /apis/disasterconfigs.testudo.softcdata.com/v1/configs/:name`
- **AND** 请求体包含 `imageRewrite`
- **THEN** 服务端必须校验 `unmatchedPolicy` 仅允许 `Fail` 或 `Keep`
- **AND** 服务端必须校验 `applyTo` 仅允许 `resourceSync` 或 `drill`
- **AND** 服务端必须拒绝重复的 `sourceImageSource`
- **AND** 服务端必须校验映射别名满足正向或反向别名组合存在性
- **AND** 正向组合定义为 `sourceImageSource` 在 source Cluster 且 `targetImageSource` 在 target Cluster
- **AND** 反向组合定义为 `sourceImageSource` 在 target Cluster 且 `targetImageSource` 在 source Cluster

#### Scenario: 查询基础配置返回镜像映射配置
- **WHEN** 客户端调用 `GET /apis/disasterconfigs.testudo.softcdata.com/v1/configs` 或 `GET /apis/disasterconfigs.testudo.softcdata.com/v1/configs/:name`
- **THEN** 响应 `data.*.spec` 中必须包含 `imageRewrite` 字段
- **AND** 未显式配置 `unmatchedPolicy` 时按 `Fail` 解释
