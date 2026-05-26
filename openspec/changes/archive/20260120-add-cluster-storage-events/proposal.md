# 提案：Cluster 与 Storage 全局事件上报

## 背景
当前历史事件接口 (`/events`) 仅支持 `AppBackup` 和 `AppRestore` 的事件展示。用户反馈希望在全局事件流中也能看到 **Cluster 创建/删除/状态变更** 和 **StorageRepository 创建/删除/验证结果** 等关键运维事件。

这类事件与 AppBackup/AppRestore 不同，它们的生命周期由 **Operator 控制器** 管理，因此需要 Operator 发射相应的 Kubernetes Events，Server 端才能聚合展示。

## 目标
1. **Operator 侧**: 为 `Cluster` 和 `StorageRepository` 控制器增加结构化事件发射。
2. **Server 侧**: 在历史事件接口中支持新的事件类型筛选。

## 依赖
- **disaster-operator**: 需要先规划并实现 Cluster/StorageRepository 控制器的事件发射逻辑

## Server 端变更

### 事件类型扩展
当前事件接口支持 `taskType` 筛选，需扩展支持：
- `AppBackup` (已有)
- `AppRestore` (已有)
- `Cluster` (新增)
- `StorageRepository` (新增)

### 事件聚合
Server 的事件列表接口需要能够聚合来自不同资源类型的事件：
- 识别 `involvedObject.kind` 为 `Cluster` 或 `StorageRepository`
- 解析事件 Reason 和 Message 提取关键信息

## 影响范围
- **disaster-server**:
  - `internal/apis/event/v1/handler.go` - 扩展事件类型过滤
- **disaster-operator** (关联提案):
  - `internal/controller/cluster_controller.go` - 增加事件发射
  - `internal/controller/storagerepository_controller.go` - 增加事件发射

## 变更 ID
`add-cluster-storage-events`
