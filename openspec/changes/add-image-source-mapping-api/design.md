## Context
异地容灾场景中，源集群与目标集群使用不同镜像仓库是常态。  
Operator 已引入镜像替换执行能力，但 server API 与 Apipost 文档尚未提供完整字段入口，前端无法按标准接口提交镜像源映射。

## Goals
- 在 Cluster API 暴露 `imageSources` 字段定义。
- 在 DisasterConfig API 暴露 `imageRewrite` 字段定义。
- 明确 DisasterInstance API 不再作为 `imageRewrite` 配置入口。
- 在 Apipost 详细设计中明确新增字段、默认值、枚举、校验与示例。

## Non-Goals
- 不实现镜像仓库同步、预热、复制。
- 不改变已有容灾流程路由与 URL。
- 不修改 Velero/BSL 行为。

## Decisions

### Decision 1: Cluster 新增字段结构
`Cluster` 请求与响应补充：

```json
{
  "imageSources": [
    {
      "name": "prod-main",
      "registry": "harbor.prod.local"
    }
  ]
}
```

字段约束：
- `imageSources`：可选，数组。
- `imageSources[].name`：必填，去空格后非空；同一 Cluster 内唯一。
- `imageSources[].registry`：必填，去空格后非空；表达仓库前缀，不包含通配规则。

### Decision 2: DisasterConfig 新增字段结构
`DisasterConfig` 请求与响应补充：

```json
{
  "imageRewrite": {
    "enabled": true,
    "applyTo": ["resourceSync", "drill"],
    "unmatchedPolicy": "Fail",
    "mappings": [
      {
        "sourceImageSource": "prod-main",
        "targetImageSource": "dr-main"
      }
    ]
  }
}
```

字段约束：
- `enabled`：布尔值，默认 `false`。
- `applyTo`：可选数组；元素仅允许 `resourceSync`、`drill`。
- `unmatchedPolicy`：可选；仅允许 `Fail`、`Keep`；默认 `Fail`。
- `mappings`：当 `enabled=true` 时必须提供至少 1 项。
- `mappings[].sourceImageSource` 与 `mappings[].targetImageSource`：必填，去空格后非空。
- `mappings` 中 `sourceImageSource` 不允许重复。

### Decision 3: 服务端校验语义
对 `imageRewrite` 的校验执行顺序固定为：
1. 校验结构合法性（空值、枚举、重复 source）。
2. 基于 `config.spec.sourceCluster/targetCluster` 解析当前主备集群。
3. 校验每一条映射别名满足以下任一组合：
   - 正向组合：`sourceImageSource` 存在于 source cluster，`targetImageSource` 存在于 target cluster。
   - 反向组合：`sourceImageSource` 存在于 target cluster，`targetImageSource` 存在于 source cluster。
4. 校验通过后落盘。

### Decision 4: Apipost 文档同步策略
Apipost 同步范围固定为 11 个接口：
- Cluster: `POST /clusters`(token), `POST /clusters`(kubeconfig), `PATCH /clusters/:name`, `GET /clusters`, `GET /clusters/:name`
- Config: `POST /configs`, `PATCH /configs/:name`, `GET /configs`, `GET /configs/:name`
- Instance: `POST /instances`, `PUT /instances/:name`（标注不再承载 `imageRewrite`）

每个接口文档必须包含：
- 新增字段名称与类型
- 必填性与默认值
- 枚举取值
- 最小可用示例 JSON

## Risks
- 文档先行但代码未同步，前端提交后返回校验错误。
- 历史示例数据未更新，导致字段说明与示例不一致。

## Mitigation
- 在提案任务中分离“文档更新”和“代码实现”，并通过任务状态显式区分。
- 对每个受影响接口追加字段说明，不复用旧描述文本。
