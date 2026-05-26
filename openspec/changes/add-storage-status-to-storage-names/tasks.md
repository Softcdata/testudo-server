# Tasks: 为存储名称列表增加状态字段

## 1. Implementation
- [x] 1.1 扩展 `DisasterStorageNameDTO` 增加 `status`
- [x] 1.2 `GET /storages/names` 返回每项 `status`
- [x] 1.3 单元测试覆盖 `status` 字段
- [x] 1.4 更新 OpenSpec 增量规范 `disaster_storage`
- [x] 1.5 验证：`go test ./internal/apis/disaster_storage/v1`，`openspec validate add-storage-status-to-storage-names --strict`
