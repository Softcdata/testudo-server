## Context
RunAPI 当前缺少项目级详细说明规范。发版前补全文档时，核心风险不是格式不一致，而是字段语义写错：server 的入参通常会转换为 operator CRD，operator controller 再基于 CRD 字段创建、更新、读取 Velero 与 Kubernetes 原生资源，并把状态与失败原因写回 Status。

因此 RunAPI 详细说明必须从“接口能调通”提升为“调用方能准确理解字段目的、资源影响、返回字段来源与错误原因”。

## Goals
- 固定每个接口详细说明的五段结构。
- 固定入参字段与返回字段的说明粒度。
- 固定 server 与 operator 调用链取证要求。
- 固定 RunAPI 已有说明的保留方式。
- 固定 server 接口盘点、RunAPI 接口盘点、差异对账、逐接口查询 operator 调用链、逐接口更新、更新后验证的执行顺序。

## Non-Goals
- 不补充与接口无关的产品背景长文。
- 不把详细说明写成需求说明书。
- 不复述请求参数表中已经完整表达的类型信息。
- 不在证据不足时猜测字段含义。

## Decision 1: 详细说明固定为五段
每个接口详细说明必须包含并只围绕以下五段组织：

```markdown
## 1. 接口用来干什么

## 2. 控制哪些资源

## 3. 入参详细说明

## 4. 返回详细说明

## 5. 可能返回的错误
```

### 五段内容要求
- `接口用来干什么`：说明接口业务用途与典型使用场景。
- `控制哪些资源`：说明 server 直接操作资源、operator 下层影响资源、Velero 与 Kubernetes 原生资源影响、资源作用范围。
- `入参详细说明`：按字段列出中文含义、可传入值、传入目的、是否必传、约束与默认值。
- `返回详细说明`：按字段列出中文含义、可能取值、字段来源、字段用途、为空条件。
- `可能返回的错误`：列出 server 当场返回错误，并补充 operator 后续回写失败的状态字段。

## Decision 2: 入参字段说明表固定列
入参说明必须使用以下表格列：

```markdown
| 字段 | 类型 | 是否必传 | 可传入值 | 中文含义 | 传入目的 | 约束与默认值 |
|---|---|---|---|---|---|---|
```

### 入参字段硬性规范
- 字段路径必须使用真实 JSON 路径，例如 `name`、`spec.config`、`restorePolicy.execution.restorePVs`。
- `类型` 必须以 server 对外请求结构为准，并结合 operator CRD 类型确认。
- `是否必传` 必须基于 server 校验、CRD schema、controller 前置检查三处证据判断。
- `可传入值` 必须列出枚举值、布尔值含义、数组元素含义、对象结构。
- `中文含义` 必须说明业务含义，不能只翻译字段名。
- `传入目的` 必须说明该值会控制什么行为，以及会影响哪些下层资源。
- `约束与默认值` 必须说明唯一性、引用资源存在性、空值语义、默认值来源。

## Decision 3: 返回字段说明表固定列
返回说明必须使用以下表格列：

```markdown
| 字段 | 类型 | 可能取值 | 中文含义 | 字段来源 | 说明 |
|---|---|---|---|---|---|
```

### 返回字段硬性规范
- 返回结构以 server 实际 DTO 与转换函数为准。
- `字段来源` 必须明确写为 server 计算、CRD Spec、CRD Status、Velero Status、Kubernetes metadata、Kubernetes Event、存储系统返回。
- CRD Status 字段必须说明由 operator reconcile 后回写。
- 状态字段必须列出真实可能取值，取值来源为 operator 类型常量、controller 状态机分支、Velero 状态常量。
- 空字段必须说明为空条件，例如尚未 reconcile、下层资源未创建、任务未完成、失败前未产生结果。

## Decision 4: 控制资源说明必须区分直接资源与下层资源
`控制哪些资源` 必须至少包含：

```markdown
直接操作资源：
- ...

下层影响资源：
- ...

资源作用范围：
- ...
```

### 资源说明硬性规范
- server 通过 Kubernetes client 创建、查询、更新、删除的资源写入直接操作资源。
- operator controller 基于该资源创建、查询、更新、删除的资源写入下层影响资源。
- 影响 Velero `Backup`、`Restore`、`Schedule`、`BackupStorageLocation`、`VolumeSnapshotLocation`、`ResourceModifier` 时必须写明。
- 影响 Kubernetes `Secret`、`ConfigMap`、`ServiceAccount`、`Namespace`、工作负载资源时必须写明。
- 资源命名空间固定为 `disaster-system` 时必须写明。
- 涉及源集群与目标集群时必须写明跨集群作用范围。

## Decision 5: 单接口字段取证顺序固定
完成 server 接口清单、RunAPI 接口清单与差异对账后，每个接口必须按以下顺序取证：

1. server router：确认方法、路径、模块、公开范围、WebSocket 属性。
2. server handler：确认接口用途、直接操作资源、错误返回点。
3. server request struct：确认入参 JSON 字段、类型、基础必填性。
4. server 校验与默认值逻辑：确认必填、枚举、引用资源、默认值、空值语义。
5. server 转换逻辑：确认请求字段如何写入 CRD Spec、metadata、annotation、label。
6. server DTO 与转换函数：确认响应 JSON 字段、字段来源、隐藏字段。
7. operator CRD 类型：确认 Spec、Status、枚举、kubebuilder 校验。
8. operator controller：确认字段如何驱动下层资源、状态机、失败 reason 与 message。
9. Velero 与 Kubernetes 资源：确认最终被创建、更新、读取、删除的真实对象。

证据不足时，详细说明中必须写 `待确认`，并在 Todo 中留下复核任务。不得以字段名猜测业务含义。

## Decision 6: 错误说明必须分为同步错误与异步失败
`可能返回的错误` 必须包含两类内容。

### server 当场返回错误
使用表格：

```markdown
| HTTP 状态码 | 业务码 | 错误原因 | 触发条件 |
|---|---|---|---|
```

必须覆盖当前接口实际可能出现的参数错误、认证错误、权限错误、资源不存在、资源冲突、Kubernetes client 调用失败。

### operator 后续回写失败
当接口成功创建以及更新 CRD 后，后续执行由 operator 完成时，必须补充：

```markdown
下层执行失败不会表现为本接口 HTTP 错误。接口成功后，operator 执行失败会回写到 `[status 字段路径]`、`reason`、`message`。调用方需要通过详情接口、列表接口以及事件流接口查看最终状态。
```

## Decision 7: 已有 RunAPI 说明必须保留
更新已存在接口时：
- 新增五段说明放在最前面。
- 原详细说明整体放在新增说明之后。
- 原详细说明标题固定为 `## 原有说明`。
- 原详细说明不得删减、改写、重排。

格式：

```markdown
## 1. 接口用来干什么

...

## 5. 可能返回的错误

...

---

## 原有说明

[原说明原样保留]
```

## Decision 8: RunAPI 对账键固定
接口对账主键为：

```text
METHOD + 标准化路径
```

标准化路径规则：
- `{{baseurl}}` 不参与对账。
- `/api` 与 `/apis` 前缀按实际 server 注册路径分别记录。
- `:name`、`{name}`、`{{name}}` 统一为 `:name`。
- 查询参数不参与主键，进入接口详情字段说明。
- WebSocket 接口仍按 HTTP upgrade 的 `GET` 路径记录，并标记 `WebSocket=true`。

## Decision 9: 执行顺序固定
实施阶段必须按以下顺序推进：

1. 完成 server 全量接口清单。
2. 完成 RunAPI 全量接口清单。
3. 完成差异清单。
4. 完成模块级 Todo。
5. 生成按模块组织的逐接口勾选清单。
6. 按逐接口勾选清单顺序查询 operator 调用链。
7. 逐接口编写五段说明草稿。
8. 逐接口新增以及更新 RunAPI。
9. 逐接口读取 RunAPI 详情并验证。
10. 汇总发版前剩余风险。

在第 1 至第 5 步完成前，不得查询接口级 operator 调用链并修改 RunAPI。第 6 至第 9 步必须以单个接口为最小闭环推进。

## Decision 10: 逐接口勾选清单固定
差异对账完成后，必须生成一份按模块组织的逐接口勾选清单。清单项格式固定为：

```markdown
- [ ] `METHOD /path` - RunAPI 状态：[缺失/已存在/模块疑似错位/待人工复核]；目标动作：[新增/补充说明/移动目录/复核]
```

### 勾选规则
- 接口未完成调用链取证、详细说明写入、RunAPI 回读验证前，清单项必须保持未勾选。
- 单个接口完成后，必须立即将该接口清单项改为已勾选。
- 不得在一批接口全部完成后再集中勾选。
- 存在 `待确认` 字段的接口不得勾选完成，必须保留为未勾选并注明阻塞原因。
- 清单必须保留 RunAPI 缺失接口、已存在接口、模块疑似错位接口、需要人工复核接口，不得只列需要修改的接口。

## Decision 11: 单接口完成标准固定
单个接口只有同时满足以下条件，才能标记完成并勾选清单：
- 已确认 server 方法、路径、handler、认证范围。
- 已确认 RunAPI 中的接口 ID、目录、方法、路径。
- 已确认该接口在差异清单中的类型：RunAPI 缺失、已存在、模块疑似错位、需要人工复核。
- 已确认直接操作资源与下层影响资源。
- 已确认全部入参字段说明。
- 已确认全部返回字段说明。
- 已确认 server 当场错误。
- 已确认 operator 后续回写失败字段。
- 已按五段结构写入 RunAPI。
- 已保留原有说明。
- 已重新读取 RunAPI 详情并检查内容。

## Risks
- server 与 operator 当前依赖版本存在差异，字段说明可能需要同时参考 go.mod 依赖版本与本机 operator 仓库。
- RunAPI 中部分接口可能放在错误目录，单纯按目录匹配会漏判。
- 部分字段只在 controller 分支中使用，单看类型定义无法确认传入目的。
- 部分状态字段来自 Velero，需继续追溯 Velero 类型与 operator 转换逻辑。

## Mitigation
- 以 server `go.mod` 中的 operator 依赖为版本基准，同时读取本机 `/home/chenxi/YS/disaster-operator` 调用链。
- 对账时使用方法与标准化路径为主，目录名称只作为辅助判断。
- 每个字段必须记录证据来源，证据不足标记 `待确认`。
- 状态字段优先从 operator controller 与 Velero 状态类型确认。
