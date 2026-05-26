# Change: 对齐 server 侧自定义资源修改器 pair-only 契约

## Why

`disaster-operator` 已将 `reversible` 的正式契约收敛为 pair-only：

- `pair.path`
- `pair.sourceValue`
- `pair.targetValue`

但 `disaster-server` 当前仍存在三类旧口径残留：

1. 实例创建/更新校验仍按 `transform.type=map/template/pair` 检查。
2. `modifierRulesText` 示例与测试仍生成旧 `transform + forwardValue/reverseValue` 结构。
3. vendored `disaster-operator` 类型副本仍停留在旧 `Transform` 模型，导致本地 server 即使重启也会继续拒绝新的 pair 输入。

这会直接造成 server API 与 operator 真实契约分裂：用户提交合法的 pair-only 请求，却在 server 层被旧 `transform` 校验提前拒绝。

## What Changes

### 1. 对齐 server 的 modifierRules 输入模型

- 实例创建/更新接口继续支持：
  - `restorePolicy.modifierRules`
  - `restorePolicy.modifierRulesText`
- 其中 `reversible` 规则必须使用 pair-only 正式结构：
  - `pair.path`
  - `pair.sourceValue`
  - `pair.targetValue`

### 2. 旧 transform 输入改为失败关闭

- server 不再接受旧 `transform.type=map/template/pair` 作为正式输入。
- 当用户仍提交旧 `transform` 时，server 必须返回 400。
- 错误消息必须明确指向 `pair.path/sourceValue/targetValue`。

### 3. 对齐提交期静态校验与 live validation

- 静态校验改为检查 `rule.Pair`。
- live validation 提取提交期 patch 路径时改为读取 `rule.Pair.Path`。
- 错误口径与 operator 保持一致，不再输出 `reversible rule missing transform`。

### 4. 同步 server 中 vendored operator 类型副本

- server 当前编译优先使用 `vendor/` 中的 operator 类型。
- 必须同步 vendored `RestoreModifierRule` / `RestoreModifierPair` 相关定义，使其与当前 operator pair-only 契约一致。

### 5. 更新测试与示例

- 所有实例 API handler tests 中的 reversible 示例改为 pair-only。
- `modifierRulesText` 示例改为 pair-only JSON 文本。
- 增加旧 `transform` 写法被拒绝的回归测试。

## Non-Goals

- 本提案不改动 operator 执行链路。
- 本提案不实现模板 CRUD 或模板绑定能力。
- 本提案不改变 `veleroNative` 透传语义。

## Dependencies

- 依赖 `disaster-operator` 当前 pair-only 契约：
  - `refactor-reversible-modifier-pair-only`
  - `restore-modifier` 当前规范

## Impact

- Affected specs:
  - `api-standards`
- Affected code:
  - `vendor/github.com/softcdata/testudo-operator/pkg/apis/disaster/v1/*`
  - `internal/apis/disaster_instance/v1/types.go`
  - `internal/apis/disaster_instance/v1/modifier_rule_validation.go`
  - `internal/apis/disaster_instance/v1/modifier_rule_live_validation.go`
  - `internal/apis/disaster_instance/v1/handler_test.go`
