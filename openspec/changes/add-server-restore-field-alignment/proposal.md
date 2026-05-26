# Change: 对齐 server 端应用恢复与容灾操作字段契约

## Why

当前 `disaster-operator` 已在进行中的提案 `add-restore-policy-and-sc-mapping` 中引入了新的恢复与就绪校验语义，`disaster-server` 侧存在以下契约缺口：

1. `DisasterInstance` 接口尚未暴露实例级恢复策略字段与 `skipPodReadyCheck` 默认策略字段。
2. 实例与容灾组操作接口仍以 `waitUntilReady` 为主，缺少 `skipPodReadyCheck` 显式入参，且组操作链路没有统一透传。
3. `AppRestore` 创建与更新接口仍使用 `scMapping` 命名，和实例级恢复策略中的 `storageClassMapping` 命名不一致。

这会导致前端与运维脚本在“实例配置 -> 触发操作 -> 应用恢复”这条链路上出现字段语义漂移，增加误配风险。

## What Changes

### 1. 补齐 DisasterInstance API 的恢复字段透传

对以下接口补齐字段契约并保持 DTO 回传一致：

- `POST /apis/disasterinstances.testudo.softcdata.com/v1/instances`
- `PUT /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name`
- `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances`
- `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name`

新增并对齐：

- 创建/更新请求顶层字段：`skipPodReadyCheck`、`restorePolicy`
- 查询与创建响应中的 `data.spec.skipPodReadyCheck`、`data.spec.restorePolicy`（与 operator 最终字段结构一致，覆盖 `storageClassMapping` 与 `ingressClassMapping`）
- 创建/更新请求支持 `restorePolicy.modifierRulesText` 文本入参：
  - 文本格式为 JSON 数组字符串（例如 `"[{...},{...}]"`）
  - 服务端在写入 CRD 前解析为结构化 `restorePolicy.modifierRules`
  - 当文本中包含 `reversible` 规则时，必须使用当前正式的 pair-only 结构：
    - `pair.path`
    - `pair.sourceValue`
    - `pair.targetValue`
  - 当 `modifierRules` 与 `modifierRulesText` 同时提供且语义不一致时，返回 400（`ModifierRulesInputConflict`）

### 2. 统一实例/组操作接口的就绪校验参数

对以下接口统一支持 `config.skipPodReadyCheck`：

- `POST /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/actions`
- `POST /apis/disastergroups.testudo.softcdata.com/v1/groups/:name/actions`

兼容策略：

- 保留 `config.waitUntilReady` 兼容输入。
- 当 `config.skipPodReadyCheck` 与 `config.waitUntilReady` 同时出现时，服务端按 `skipPodReadyCheck` 计算最终值。
- 服务端在创建 `DisasterOperation` 时，按统一规则写入 `spec.skipPodReadyCheck` 以及兼容字段 `spec.waitUntilReady`，保证新旧 operator 版本都可消费。

### 3. 对齐 AppRestore 映射字段命名

对以下接口补齐“规范名 + 兼容别名”策略：

- `POST /apis/apprestores.testudo.softcdata.com/v1/apprestores`
- `PUT /apis/apprestores.testudo.softcdata.com/v1/apprestores/:name`

命名策略：

- 规范字段：`storageClassMapping`
- 兼容字段：`scMapping`
- 当两者同时出现且内容不一致时，服务端返回 400 拒绝请求，避免歧义。
- `ingressClassMapping` 继续沿用当前命名，并与实例恢复策略中的同名能力保持一致语义。

## Non-Goals

- 本提案不改动 operator 控制器执行逻辑。
- 本提案不引入 `restorePolicy.hooks` 字段，该能力由独立提案推进。
- 本提案不改变既有恢复执行顺序。

## Compatibility Commitment

- 对仅使用旧字段的现有调用保持兼容：`waitUntilReady` 与 `scMapping` 继续可用。
- 未传新字段时，实例创建、实例更新、应用恢复、容灾操作行为保持现状。
- 本提案为增量契约补齐，不移除现有 API 路由。

## Dependencies

- 依赖 `disaster-operator` 提案 `add-restore-policy-and-sc-mapping` 的字段落地。
- 当前 `modifierRules` 正式 contract 以后续 `refactor-server-modifier-pair-only-alignment` 与 operator `restore-modifier` 规范为准。

## Impact

- Affected specs:
  - `api-standards`
- Affected code:
  - `internal/apis/disaster_instance/v1/types.go`
  - `internal/apis/disaster_instance/v1/handler.go`
  - `internal/apis/disaster_instance/v1/handler_action.go`
  - `internal/apis/disaster_group/v1/handler.go`
  - `internal/apis/app_restore/v1/types.go`
  - `internal/apis/app_restore/v1/handler.go`
  - `internal/apis/disaster_instance/v1/handler_test.go`
  - `internal/apis/disaster_group/v1/handler_test.go`
  - `internal/apis/app_restore/v1/*_test.go`
