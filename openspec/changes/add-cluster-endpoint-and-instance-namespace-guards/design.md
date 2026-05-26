# 设计：集群 endpoint 唯一与实例命名空间唯一门禁

## 1. 背景

当前 `disaster-server` 有两个典型写入口：

1. `ClusterHandler.createCluster`
2. `InstanceHandler.createInstance` / `InstanceHandler.updateInstance`

这三个入口都直接把请求落成 CR，没有做“同一对象是否已经被系统占用”的前置门禁。

本提案的目标不是把校验职责下沉到 `disaster-operator`，而是在 server 提交前建立稳定门禁，并提供与门禁同口径的只读查询接口给前端。

## 2. 归属边界

本轮能力归属固定如下：

- `disaster-server`：唯一性校验、冲突响应、只读查询接口
- `disaster-operator`：保持不变
- `cluster-disaster-web`：后续消费新增接口与冲突详情，本轮不改代码

原因：

- 用户入口与风险入口都在 REST API。
- 当前 operator 没有对应 admission contract。
- 先在 server 层收敛用户写入口，可以最小代价解决前端与提交期风险。

## 3. Cluster endpoint 唯一性

### 3.1 请求侧 endpoint 解析

创建集群时，服务端按以下顺序解析本次请求的有效 endpoint：

1. 读取 `CreateDisasterClusterRequest.Endpoint`
2. 对该字段执行 `strings.TrimSpace`
3. 若结果非空，直接进入归一化
4. 若结果为空，读取 `CreateDisasterClusterRequest.KubeConfig`
5. 对 kubeconfig 调用 `tools.GetRestConfig`
6. 使用返回的 `rest.Config.Host` 作为原始 endpoint
7. 若 kubeconfig 解析失败，直接返回 `400 Bad Request`

这一定义保证：

- token/endpoint 创建路径可直接判定
- kubeconfig 创建路径也能得到同一套 endpoint 键

### 3.2 endpoint 归一化规则

所有 endpoint 对比都必须使用同一套归一化结果。归一化步骤固定为：

1. 去掉前后空格
2. 使用 URL 解析器解析
3. `scheme` 转为小写
4. `hostname` 转为小写
5. 若 `scheme=https` 且端口为 `443`，删除端口
6. 若 `scheme=http` 且端口为 `80`，删除端口
7. `path` 去掉末尾 `/`
8. 丢弃 query 与 fragment
9. 重建为 `scheme://host[:port][path]`

示例：

- ` HTTPS://192.0.2.170:6443/ ` -> `https://192.0.2.170:6443`
- `https://api.demo.local/` -> `https://api.demo.local`
- `https://api.demo.local:443` -> `https://api.demo.local`

### 3.3 既有集群 endpoint 解析

扫描现有 `Cluster` 时，比较值按以下顺序解析：

1. `cluster.Spec.Endpoint`
2. `cluster.Status.Endpoint`
3. `cluster.Spec.KubeConfig` 派生的 `rest.Config.Host`

命中第一个可解析值后立即停止。

若三者都无法得到可归一化的 endpoint，则该已有 `Cluster` 不参与 endpoint 冲突集合。

### 3.4 冲突语义

若请求侧归一化 endpoint 命中已有集群的归一化 endpoint，则创建必须返回：

- HTTP `409`
- `code=3009`
- `message` 明确包含已有集群名与 endpoint
- `meta` 固定包含：
  - `conflictType=clusterEndpoint`
  - `conflictCluster=<已有集群名>`
  - `conflictEndpoint=<归一化 endpoint>`

## 4. 实例命名空间唯一性

### 4.1 唯一性作用域

命名空间唯一性不按实例 CR 命名空间判定，不按全局判定，固定按 `DisasterConfig.spec.sourceCluster` 判定。

原因：

- 实例创建页当前就是按 `sourceCluster` 读取候选命名空间。
- 用户描述的风险是“同一集群下多个实例选同一命名空间”。
- `sourceCluster` 才是受保护工作负载所在集群的稳定语义。

### 4.2 共享聚合器

新增一个共享聚合器，供以下三处复用：

1. `createInstance`
2. `updateInstance`
3. `GET /clusters/:name/protected-namespaces`

聚合器固定执行以下步骤：

1. 读取全部 `DisasterConfig`
2. 建立 `configName -> sourceCluster` 映射
3. 读取全部 `DisasterInstance`
4. 对每个实例取 `instance.Spec.Config`
5. 用映射表解析其 `sourceCluster`
6. 归一化 `instance.Spec.Namespaces`
7. 产出 `sourceCluster -> protected namespace owners` 聚合结果

`owner` 记录的最小字段固定为：

- `namespace`
- `instanceName`
- `instanceNamespace`
- `configName`

### 4.3 命名空间归一化规则

实例命名空间比较与查询输出都必须执行以下归一化：

1. 对每个 namespace 执行 `strings.TrimSpace`
2. 丢弃空字符串
3. 按值去重
4. 最终输出按字典序排序

### 4.4 createInstance 门禁

`createInstance` 固定按以下顺序执行：

1. 绑定请求
2. 解析 `spec`
3. 读取请求指定的 `DisasterConfig`
4. 取 `config.Spec.SourceCluster`
5. 用共享聚合器读取该 `sourceCluster` 下的现有 namespace owners
6. 计算本次请求归一化命名空间与 owners 的交集
7. 若交集非空，直接返回 `409`
8. 若交集为空，再继续原有 CR 创建流程

### 4.5 updateInstance 门禁

`updateInstance` 固定按以下顺序执行：

1. 解析目标实例
2. 生成更新后的有效 `spec`
3. 读取该实例有效 `spec.config` 对应的 `DisasterConfig`
4. 取 `config.Spec.SourceCluster`
5. 用共享聚合器读取该 `sourceCluster` 下的现有 namespace owners
6. 在冲突集合中排除当前实例自身 `(instanceNamespace, instanceName)`
7. 计算更新后命名空间与剩余 owners 的交集
8. 若交集非空，直接返回 `409`
9. 若交集为空，再继续原有更新流程

这保证更新链路不能绕过唯一性门禁。

### 4.6 命名空间冲突响应

若实例写入命中命名空间冲突，则返回：

- HTTP `409`
- `code=3009`
- `message` 明确包含 `sourceCluster` 与冲突命名空间
- `meta` 固定包含：
  - `conflictType=protectedNamespaces`
  - `sourceCluster=<cluster>`
  - `conflictNamespaces=<string[]>`
  - `owners=<owner[]>`

其中 `owners` 只返回本次冲突涉及到的实例明细，不返回无关命名空间。

## 5. 按集群查询已受保护命名空间接口

### 5.1 路由位置

新增路由：

- `GET /apis/cluster.testudo.softcdata.com/v1/clusters/:name/protected-namespaces`

挂在 Cluster API 的原因是：

- 路由语义与查询维度一致
- 前端当前实例创建页已经先拿 `sourceCluster`
- 接口结果天然属于 cluster 维度只读信息

### 5.2 查询流程

请求到达后固定执行：

1. 读取 `:name`
2. 先校验该 `Cluster` 存在，不存在直接返回 `404`
3. 调用共享聚合器读取 `:name` 对应的聚合结果
4. 产出响应 DTO

### 5.3 响应 DTO

响应必须遵循现有 API 统一响应信封与 collection 元数据标准。

```json
{
  "code": 0,
  "message": "OK",
  "data": {
    "items": [
      {
        "namespace": "app-a",
        "cluster": "cluster-a",
        "owners": [
          {
            "instanceName": "inst-a",
            "instanceNamespace": "disaster-system",
            "configName": "cfg-a"
          }
        ]
      }
    ]
  },
  "meta": {
    "type": "collection",
    "resourceType": "clusterProtectedNamespace"
  },
  "trace_id": "..."
}
```

字段约束固定为：

- `data.items[].namespace`：已受保护命名空间，按字典序排序
- `data.items[].cluster`：请求路径中的集群名
- `data.items[].owners[]`：该命名空间的占用实例明细
- `meta.type`：固定为 `collection`
- `meta.resourceType`：固定为 `clusterProtectedNamespace`
- `meta.pagination`、`meta.sort`、`meta.links`：按现有 collection 响应标准生成

### 5.4 无副作用约束

该接口只允许：

- 读取 `Cluster`
- 读取 `DisasterConfig`
- 读取 `DisasterInstance`
- 组装响应

禁止创建、更新、删除任何 CR。

## 6. 测试矩阵

最小测试覆盖固定为：

1. `createCluster`：直接 endpoint 命中重复，返回 `409`
2. `createCluster`：kubeconfig 派生 endpoint 命中重复，返回 `409`
3. `createInstance`：同一 `sourceCluster` 命中重复 namespace，返回 `409`
4. `createInstance`：不同 `sourceCluster` 允许同名 namespace
5. `updateInstance`：保留自身 namespace 不被误判
6. `updateInstance`：改成他人已占用 namespace，返回 `409`
7. `GET /clusters/:name/protected-namespaces`：返回聚合后的 `namespaces + items`
8. `GET /clusters/:name/protected-namespaces`：不存在的 cluster 返回 `404`
