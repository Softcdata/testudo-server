# Change: 新增按集群返回恢复 Class 列表接口

## Why

当前前端配置恢复映射时，流程是“用户先手工输入目标 Class，再调用 `restore-classes/validate` 做存在性预检”。该流程存在三个问题：

1. 用户需要提前知道目标集群中可用的 `StorageClass` 与 `IngressClass`，输入成本高。
2. 输入错误只能在预检阶段暴露，界面无法直接提供候选值。
3. 预检接口职责偏向“校验”，不适合承担“可选项发现”。

## What Changes

### 1. 新增按集群查询恢复 Class 列表接口

新增接口：

- `GET /apis/cluster.testudo.softcdata.com/v1/clusters/:name/restore-classes`

接口语义：

- `:name` 作为目标集群名。
- 服务端直接查询该集群中的 `StorageClass` 列表与 `IngressClass` 列表。
- 返回结构化列表，供前端直接渲染映射候选项。

### 2. 返回结构标准化

响应体至少包含：

- `targetCluster`
- `storageClasses[]`
- `ingressClasses[]`

列表项至少包含：

- `name`
- `isDefault`

排序约束：

- `storageClasses` 与 `ingressClasses` 均按 `name` 升序返回，保证同一输入下响应稳定。

### 3. 现有预检接口保持兼容

现有接口
`POST /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/restore-classes/validate`
保持可用。

职责划分：

- 新接口负责“发现目标集群可用 Class 列表”。
- 现有接口负责“校验映射策略在 strict 语义下是否可通过”。

### 4. 副作用约束

新接口仅执行读取与组装响应，禁止创建、更新、删除任何业务 CR，禁止触发恢复流程。

### 5. API 规范对齐

新接口必须遵循 `api-standards` 既有约束：

1. 成功响应使用统一 `Envelope` 结构，`code=0`，`data` 包含列表结果。
2. 错误响应使用双层错误码，HTTP 状态码表达传输语义，`code` 表达业务语义。
3. 不返回原始 Kubernetes 对象，返回稳定 DTO。

## Non-Goals

- 本提案不修改 `disaster-operator` 的恢复执行逻辑。
- 本提案不替换现有 `restore-classes/validate` 严格校验接口。
- 本提案不引入恢复映射自动推荐算法。

## Compatibility Commitment

- 本提案为新增只读接口，不改变现有接口行为。
- 未调用新接口的存量客户端行为保持不变。

## Impact

### Affected specs

- `disaster_cluster`
- `api-standards`

### Affected code

- `internal/apis/disaster_cluster/v1/router.go`
- `internal/apis/disaster_cluster/v1/types.go`
- `internal/apis/disaster_cluster/v1/handler.go`
- `internal/apis/disaster_cluster/v1/handler_test.go`
