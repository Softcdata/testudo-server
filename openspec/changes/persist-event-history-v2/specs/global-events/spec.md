## ADDED Requirements

### Requirement: Server 必须提供 durable task event history
系统必须 (MUST) 为结构化任务事件提供可持久化的历史记录能力，使历史查询不再受 Kubernetes Event TTL 限制。

#### Scenario: 查询 30 天前的任务历史
- **Given** 一次结构化任务事件执行已在历史存储中完成持久化
- **When** 用户查询 30 天前的任务历史
- **Then** Server 必须从 durable history 返回该执行记录
- **And** 不得依赖当前仍存活的 Kubernetes Events 才能返回结果

### Requirement: Server 必须区分执行身份与资源身份
系统必须 (MUST) 在历史记录与查询结果中区分“一次执行身份”和“资源身份”。

#### Scenario: 同一资源存在多次任务执行
- **Given** 同一个资源在不同时间触发了多次任务执行
- **When** Server 投影并查询这些历史事件
- **Then** Server 必须为每次执行生成稳定的 `executionId`
- **And** 不得继续仅用资源 UID 充当一次执行的唯一标识
- **And** 查询结果必须同时保留资源身份快照

### Requirement: Server 必须以幂等方式投影结构化事件到历史记录
系统必须 (MUST) 以幂等方式将结构化任务事件投影为 durable history，并保留 source event identity 以阻止重复 timeline node。

#### Scenario: 同一条源事件被重复投递
- **Given** history projector 重复收到同一条结构化 Kubernetes Event
- **When** Server 执行历史投影
- **Then** Server 必须只写入一次对应的 timeline node
- **And** 不得因为重复投递生成重复历史节点

### Requirement: 历史查询与实时 watch 必须职责分离
系统必须 (MUST) 保持历史查询与实时事件流职责分离：历史查询读取 durable history，实时 watch 继续读取 Kubernetes Events。

#### Scenario: 历史存储暂时不可用
- **Given** durable history store 暂时不可用
- **When** 客户端发起实时 watch 请求
- **Then** Server 不得因为历史存储故障而中断实时事件流
- **And** 历史查询路径必须显式返回失败或降级状态
- **And** 不得静默回退为“只查询最近仍存活的 Kubernetes Events”
