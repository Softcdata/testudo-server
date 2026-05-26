# 提案：容灾实例详情回显存储仓信息

## 状态
- **ID**: `enhance-instance-detail-storage`
- **作者**: Antigravity
- **创建日期**: 2026-02-11

## 1. 摘要
增强 `disaster-server` 的容灾实例（Disaster Instance）详情接口，在返回数据中聚合关联的存储仓（Storage Repository）详细信息，以便前端完整展示容灾拓扑中的存储资源配置。

## 2. 背景与动机
目前容灾实例详情仅包含 `DisasterConfig` 的信息，而 `DisasterConfig` 中的存储仅反映为一个名称字符串。前端在展示详情页时，通常需要展示存储的具体类型、桶名、Endpoint 等核心属性，目前需要额外请求存储列表或详情接口进行手动匹配。

通过在详情接口中直接聚合存储信息，可以：
1. 减少前端请求次数。
2. 简化前端状态管理逻辑。
3. 提升用户查看容灾完整链路时的体验。

## 3. 设计方案

### 3.1 DTO 变更
在 `internal/apis/disaster_instance/v1/types.go` 中：
- `DisasterInstanceDTO` 结构体增加 `Storage` 字段，类型为 `*storagev1.DisasterStorageDTO`。

### 3.2 聚合逻辑增强
在 `internal/apis/disaster_instance/v1/handler.go` 的 `getInstance` 中：
1. 调用 `h.DisasterClient.DisasterV1().DisasterInstances(namespace).Get(...)` 获取实例。
2. 调用 `h.DisasterClient.DisasterV1().DisasterConfigs().Get(...)` 获取配置。
3. 若配置存在，提取 `spec.storageRepository` 名称。
4. 调用 `h.DisasterClient.DisasterV1().StorageRepositories().Get(...)` 获取存储详情。
5. 使用 `instance.ConvertToDisasterInstanceDTO` 进行聚合。

### 3.3 安全性
聚合时必须使用 `storagev1.ConvertToDisasterStorageDTO`，该转换器已实现过滤敏感字段（`accessKey`, `secretKey`）的逻辑。

## 4. 影响范围
- **API**: `GET /apis/testudo.softcdata.com/v1/namespaces/:namespace/disasterinstances/:name`
- **DTO**: `DisasterInstanceDTO`
- **内部调用**: 增加了对 `StorageRepository` 的 Read 调用。

## 5. 替代方案
**方案：前端自行聚合**
- 缺点：详情页渲染前需要并发请求多个接口，代码复杂度高，网络开销增加。
