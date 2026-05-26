# Tasks: 添加 Token 认证支持

- [x] 在 `internal/kube/client.go`（或等效位置）中实现基于 Token 的 Kubernetes 客户端创建逻辑。
- [x] 更新 `internal/apis/disaster_cluster/` 中的集群验证服务以支持 Token 认证。
- [x] 在验证流程中验证使用 Token 的连接。
- [x] 为基于 Token 的连接添加单元测试。
- [x] 确保 API 返回 200 OK 即使验证失败（在 data 中返回 false）。
