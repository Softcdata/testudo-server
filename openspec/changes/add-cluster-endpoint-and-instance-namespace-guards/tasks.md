## 1. 规范

- [x] 1.1 完成 proposal、design、spec delta 自检，确认三处能力使用同一套聚合口径。

## 2. Cluster 门禁与查询接口

- [x] 2.1 在 Cluster create 链路实现“请求侧有效 endpoint 解析 + 归一化”。
- [x] 2.2 在 Cluster create 链路实现 endpoint 唯一性冲突检测与 `409/3009` 响应。
- [x] 2.3 新增 `GET /apis/cluster.testudo.softcdata.com/v1/clusters/:name/protected-namespaces` 路由、DTO 与 handler。
- [x] 2.4 为 Cluster endpoint 唯一性与 protected-namespaces 查询补充单测。

## 3. Instance 门禁

- [x] 3.1 实现按 `DisasterConfig.spec.sourceCluster` 聚合已受保护命名空间的共享辅助模块。
- [x] 3.2 在 `createInstance` 中接入命名空间唯一性校验并返回结构化冲突详情。
- [x] 3.3 在 `updateInstance` 中接入同口径校验，并排除当前实例自身。
- [x] 3.4 为实例创建、实例更新的命名空间冲突门禁补充单测。

## 4. 验证

- [x] 4.1 执行 `go test ./internal/apis/disaster_cluster/v1 ./internal/apis/disaster_instance/v1`
- [x] 4.2 执行 `openspec validate add-cluster-endpoint-and-instance-namespace-guards --strict`
