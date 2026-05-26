## MODIFIED Requirements

### Requirement: 集群接口必须支持镜像源目录字段
系统必须 (MUST) 在 Cluster 的创建、更新与查询接口中暴露 `imageSources` 字段，用于基础配置级镜像映射配置复用。

#### Scenario: 创建集群时提交镜像源目录
- **WHEN** 客户端调用 `POST /apis/cluster.testudo.softcdata.com/v1/clusters`
- **AND** 请求体包含 `imageSources`
- **THEN** 服务端必须接受并持久化 `imageSources`
- **AND** `imageSources[].name` 在同一集群中必须唯一
- **AND** `imageSources[].registry` 必须为非空字符串

#### Scenario: 更新集群时修改镜像源目录
- **WHEN** 客户端调用 `PATCH /apis/cluster.testudo.softcdata.com/v1/clusters/:name`
- **AND** 请求体包含 `imageSources`
- **THEN** 服务端必须按请求值更新集群镜像源目录
- **AND** 返回结果中必须包含更新后的 `spec.imageSources`

#### Scenario: 查询集群返回镜像源目录
- **WHEN** 客户端调用 `GET /apis/cluster.testudo.softcdata.com/v1/clusters` 或 `GET /apis/cluster.testudo.softcdata.com/v1/clusters/:name`
- **THEN** 响应 `data.*.spec` 中必须包含 `imageSources` 字段
