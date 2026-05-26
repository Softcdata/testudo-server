# 任务清单：Cluster 与 Storage 全局事件上报 (Server)

## 前置依赖
- [x] **等待 operator 提案完成**: `disaster-operator` 的 `add-cluster-storage-events` 提案已完成，使用标准 Label `testudo.softcdata.com/task-event=true`

## 1. Server 端事件聚合扩展

- [x] 1.1 修改 `internal/apis/event/v1/handler.go`
  - 经核查，Server 端已默认监听所有带 `testudo.softcdata.com/task-event=true` Label 的事件，无需额外过滤逻辑即可支持 Cluster/Storage 事件
- [x] 1.2 更新事件 DTO 结构
  - 现有的 `HistoricalEventDTO` 是通用的，能够兼容新的事件类型

## 2. 验证

- [x] 2.1 验证 Server 能正确聚合和展示
  - Operator 发射的事件带有标准 Label，Server 端通用 Watcher (`watchEvents`) 能够自动捕获并推送

