# Tasks

- [ ] 定义 `WatchEventDTO` 结构体 @backend
- [ ] 更新 `internal/utils/watch.go` 中的 `StreamWatch` 以使用 `Envelope` 和 `WatchEventDTO` @backend
- [ ] 重构 `internal/apis/event` 以适配新的 `StreamWatch` @backend
- [ ] 重构 `internal/apis/disaster_cluster` 以适配新的 `StreamWatch` @backend
- [ ] 验证所有 WebSocket 接口返回的数据格式符合规范 @backend
