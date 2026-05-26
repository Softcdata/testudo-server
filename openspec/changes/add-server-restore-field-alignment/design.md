# 设计文档：server 端恢复字段一致性

## 1. 上下文

本设计覆盖 server 三条入口：

1. `DisasterInstance` 创建、更新、查询接口。
2. 实例操作与容灾组操作 action 接口。
3. `AppRestore` 创建、更新接口。

目标是让字段语义在“实例默认策略 + 操作覆盖 + 应用恢复参数”之间保持单一解释，不出现同义不同名。

## 2. 目标

1. `DisasterInstance` API 可配置并可回显 `skipPodReadyCheck` 以及 `restorePolicy`。
2. 操作接口对 `skipPodReadyCheck` 与 `waitUntilReady` 提供确定性优先级。
3. `AppRestore` 接口对 SC 映射字段提供统一规范名。
4. 保持现有调用链路兼容，不破坏存量调用。

## 3. 参数优先级与写入规则

### 3.1 操作接口输入解析

在处理 action 请求时，按以下顺序计算 `effectiveSkipPodReadyCheck`：

1. 请求体包含 `config.skipPodReadyCheck` 时，直接使用该值。
2. 请求体未包含 `config.skipPodReadyCheck` 且包含 `config.waitUntilReady` 时，使用 `!waitUntilReady`。
3. 两个字段都不存在时，不生成操作级覆盖值。

### 3.2 操作对象写入

当存在 `effectiveSkipPodReadyCheck` 时：

1. 写入 `DisasterOperation.spec.skipPodReadyCheck = effectiveSkipPodReadyCheck`。
2. 写入 `DisasterOperation.spec.waitUntilReady = !effectiveSkipPodReadyCheck`。

当不存在覆盖值时：

1. 保持当前逻辑，不额外写入覆盖语义。

该规则同时应用在实例 action 与容灾组 action 两条接口。

## 4. AppRestore 字段对齐策略

### 4.1 输入字段

创建与更新 `AppRestore` 时支持：

1. `storageClassMapping`（规范字段）。
2. `scMapping`（兼容字段）。
3. `ingressClassMapping`（保持不变）。

### 4.2 冲突处理

当 `storageClassMapping` 与 `scMapping` 同时出现且键值对不一致时：

1. 服务端直接返回 400。
2. 错误信息明确指出字段冲突。

### 4.3 规则落地

1. 使用规范字段计算最终 SC 映射输入。
2. 最终映射继续转译为 `AppRestore.spec.resourceModifierRules`。

## 5. 兼容性策略

1. 仅传 `waitUntilReady` 的旧请求继续可用。
2. 仅传 `scMapping` 的旧请求继续可用。
3. 未传新字段时，行为与当前版本一致。
4. `restorePolicy.hooks` 不在本提案处理范围。

## 6. 测试设计

1. 实例 API：创建、更新、查询的字段透传测试。
2. 操作 API：
   - 仅 `skipPodReadyCheck`。
   - 仅 `waitUntilReady`。
   - 两个字段同时传且值冲突。
   - 组操作透传一致性。
3. AppRestore API：
   - 仅 `storageClassMapping`。
   - 仅 `scMapping`。
   - 两字段冲突拒绝。
4. 回归测试：不传新字段的历史请求路径。
