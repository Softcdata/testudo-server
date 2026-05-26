## Context

server 需要把上层请求稳定映射到 operator CRD。

当前 `AppBackup` API 缺少 cluster-scoped 两个字段，`DisasterInstance` 缺少明确的恢复侧优先级语义。

## Goals / Non-Goals

- Goals:
  - 补齐 `AppBackup` scoped 四字段 API 契约。
  - 对齐 `DisasterInstance.restorePolicy.resourceSelection` scoped 四字段。
  - 在 server 入口执行确定性校验与优先级策略。
- Non-Goals:
  - 不改动 operator 控制器执行流程。
  - 不改动已有路由路径。

## Decisions

### Decision 1: API 字段与 operator 保持同名

`AppBackup` 与 `DisasterInstance` 采用与 operator 一致的字段名，避免别名增殖。

### Decision 2: DisasterInstance 恢复侧采用优先级语义

`DisasterInstance.restorePolicy.resourceSelection` 采用确定性优先级：

- 当 `includeClusterResources=true` 时，恢复侧忽略 scoped 四字段。
- 当 `includeClusterResources` 非 true 且配置了 scoped 字段时，进入 scoped 路径。

不再把 old/scoped 并存作为提交期硬拒绝条件。

### Decision 3: AppBackup 与 DisasterInstance 分别校验

- AppBackup：保持 fail-fast 冲突校验。
- DisasterInstance：按优先级选择生效路径，仅对生效路径执行冲突校验。

## Risks / Trade-offs

- 风险：字段并存时调用方误读生效结果。
  - 缓解：在文档与错误消息中明确 `includeClusterResources=true` 的覆盖语义。
- 风险：server 与 operator 版本窗口不一致。
  - 缓解：以 operator 变更为前置依赖并联动发布。

## Migration Plan

1. 完成 DTO 与请求结构字段补齐。
2. 接入统一优先级与校验函数。
3. 增加测试覆盖。
4. 更新 Apipost 与文档示例。

## Open Questions

- 是否新增 `resolvedMode` 回显字段用于前端排障。
