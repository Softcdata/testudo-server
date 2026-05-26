## MODIFIED Requirements

### Requirement: 实例接口不得作为镜像映射配置入口
系统必须 (MUST) 保持 DisasterInstance 接口仅承载实例运行范围配置（如 config、namespaces、labelSelector），镜像映射配置入口必须统一在 DisasterConfig。

#### Scenario: 创建或更新实例时提交 imageRewrite
- **WHEN** 客户端调用 `POST /apis/disasterinstances.testudo.softcdata.com/v1/instances` 或 `PUT /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name`
- **AND** 请求体包含 `imageRewrite`
- **THEN** 服务端不得将该字段持久化到 `DisasterInstance.spec`
- **AND** 实例创建与更新行为保持与未提交该字段时一致

#### Scenario: 查询实例接口
- **WHEN** 客户端调用 `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances` 或 `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name`
- **THEN** 响应 `data.*.spec` 中不应包含 `imageRewrite` 字段
