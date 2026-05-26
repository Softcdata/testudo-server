# Change: 新增集群 endpoint 唯一与实例命名空间唯一门禁

## Why

当前存在两个高风险缺口：

1. `POST /apis/cluster.testudo.softcdata.com/v1/clusters` 允许把同一个 Kubernetes API Server 以不同集群名重复接入，导致前端误选、后端重复探测、运维边界混乱。
2. `POST /apis/disasterinstances.testudo.softcdata.com/v1/instances` 当前允许同一源集群下多个 `DisasterInstance` 保护同一个业务命名空间，极易导致同步、恢复、切换流程串扰。

同时，实例创建页当前只能读取集群的 `namespaceStats`，无法直接知道“哪些命名空间已经被其他实例占用”。前端需要一个按集群返回“已受保护命名空间”的只读接口，用于禁用高风险选项。

## What Changes

### 1. 为集群创建接口增加 endpoint 唯一性门禁

服务端在创建 `Cluster` 前必须先解析本次请求的“有效 endpoint”，并基于归一化后的 endpoint 做唯一性校验。

有效 endpoint 的来源固定为：

1. 请求体 `endpoint` 非空时，直接使用该字段。
2. 请求体 `endpoint` 为空且 `kubeConfig` 非空时，解析 kubeconfig 并使用其 `server` 地址。

命中重复时：

- 返回 `409 Conflict`
- 业务码固定为 `3009`
- 错误信息中回显冲突的已有集群名与 endpoint

### 2. 为实例创建与更新接口增加命名空间唯一性门禁

服务端在创建、更新 `DisasterInstance` 前，必须按 `DisasterConfig.spec.sourceCluster` 建立保护范围，并检查本次提交的 `spec.namespaces` 是否与同一源集群下其他实例发生交集。

冲突判定固定为：

- 作用域：`sourceCluster` 相同
- 对象集：除当前实例自身之外的全部 `DisasterInstance`
- 冲突条件：归一化后的命名空间集合存在交集

命中冲突时：

- 返回 `409 Conflict`
- 业务码固定为 `3009`
- 错误信息必须说明冲突命名空间
- 响应元数据必须可让前端定位冲突实例

### 3. 新增按集群查询已受保护命名空间接口

新增只读接口：

- `GET /apis/cluster.testudo.softcdata.com/v1/clusters/:name/protected-namespaces`

接口职责：

- 按 `:name` 作为源集群名聚合全部受保护命名空间
- 返回去重后的命名空间列表
- 返回每个命名空间的占用实例明细，供前端禁用与提示

### 4. 三条能力共享同一套聚合口径

以下三处逻辑必须共享同一套“实例归属集群 + 受保护命名空间”解析口径：

1. 实例创建门禁
2. 实例更新门禁
3. `clusters/:name/protected-namespaces` 查询接口

这样才能保证前端查询结果与服务端最终提交门禁完全一致。

## Non-Goals

- 本提案不改 `disaster-operator` webhook、controller、CRD schema。
- 本提案不处理历史重复数据的自动清理或迁移。
- 本提案不改 `cluster-disaster-web` 页面实现，只提供稳定后端契约。
- 本提案不把命名空间唯一性扩大为全局唯一；唯一范围只限定在同一 `sourceCluster`。

## Compatibility Commitment

- 新增查询接口是纯增量能力，不影响存量客户端。
- 现有创建、更新接口的成功路径保持不变。
- 存量客户端若继续提交重复 endpoint、重复受保护命名空间，将从“写入成功”变为“稳定 409 拒绝”，这是本提案的预期收敛。

## Impact

### Affected specs

- `disaster_cluster`
- `disaster_instance`

### Affected code

- `internal/apis/disaster_cluster/v1/router.go`
- `internal/apis/disaster_cluster/v1/types.go`
- `internal/apis/disaster_cluster/v1/handler.go`
- `internal/apis/disaster_cluster/v1/handler_test.go`
- `internal/apis/disaster_instance/v1/handler.go`
- `internal/apis/disaster_instance/v1/types.go`
- `internal/apis/disaster_instance/v1/handler_test.go`
- 共享保护范围解析辅助模块（新增）
