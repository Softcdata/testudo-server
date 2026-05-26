# Change: 新增恢复 Class 存在性预检接口

## Why

当前 `disaster-operator` 已在恢复构建阶段执行 `StorageClass` 与 `IngressClass` 存在性校验。`disaster-server` 仍缺少一个可直接调用的预检接口，前端与运维脚本在“保存实例策略前”无法提前确认目标集群 Class 可用性。

这会带来两个问题：

1. 失败发现时机偏后，用户通常在触发恢复任务后才看到失败。
2. 失败原因定位成本偏高，排障需要进入控制器日志。

## What Changes

### 1. 新增实例级恢复 Class 预检接口

新增接口：

- `POST /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/restore-classes/validate`

接口用途：

- 针对目标集群执行 `StorageClass` 与 `IngressClass` 目标值存在性检查。
- 返回逐项检查结果与缺失列表。
- 返回与 operator 对齐的失败码，用于前端直接提示。

### 2. 预检输入与目标集群解析规则

请求体支持：

- `targetCluster`：目标集群名称（可选）。
- `storageClassMapping`：检查用 SC 映射策略（可选，二选一）。
- `ingressClassMapping`：检查用 IngressClass 映射策略（可选，二选一）。

映射输入规则：

1. 接口使用请求体映射策略作为检查输入。
2. 请求体中 `storageClassMapping` 与 `ingressClassMapping` 不能同时缺失。

目标集群解析优先级：

1. 优先使用请求体 `targetCluster`。
2. 请求体缺失时，使用 `DisasterInstance.status.secondaryCluster`。
3. 若上一步为空，读取 `DisasterConfig.spec.targetCluster`。

### 3. 结果判定与错误码

接口返回结构包含：

- `storageClassCheck.checkedTargets`
- `storageClassCheck.missingTargets`
- `ingressClassCheck.checkedTargets`
- `ingressClassCheck.missingTargets`
- `valid`
- `code`
- `message`

判定规则：

1. 映射中每个 `targetClass` 均进入检查集合。
2. 检查集合在目标集群资源列表中全部存在时，`valid=true`。
3. 出现缺失时：
   - 对应映射策略 `strictTargetValidation=true`：`valid=false`，并返回 `StorageClassTargetNotFound` 或 `IngressClassTargetNotFound`。
   - 对应映射策略 `strictTargetValidation=false`：`valid=true`，同时在缺失列表中给出明细。

### 4. 执行时机与副作用约束

执行时机必须具备确定性：

- 仅在收到该接口请求后，同步执行检查并立即返回。

副作用约束：

- 接口不得创建、更新、删除任何业务 CR。
- 接口不得触发恢复任务。

## Non-Goals

- 本提案不修改 `disaster-operator` 的恢复执行逻辑。
- 本提案不变更现有实例创建、实例更新、AppRestore 创建流程。
- 本提案不引入 Velero hooks 字段。

## Compatibility Commitment

- 本提案是新增接口，不替换现有接口。
- 现有恢复流程、容灾实例流程、容灾操作流程保持不变。
- 未调用新接口的存量客户端行为保持不变。

## Dependencies

- 依赖 operator 已落地映射失败码语义：`StorageClassTargetNotFound`、`IngressClassTargetNotFound`、`ClassMappingInvalid`。

## Impact

### Affected specs

- `api-standards`

### Affected code

- `internal/apis/disaster_instance/v1/router.go`
- `internal/apis/disaster_instance/v1/types.go`
- `internal/apis/disaster_instance/v1/handler.go`
- `internal/apis/disaster_instance/v1/handler_test.go`
- `internal/apis/disaster_instance/v1/*restore*validate*.go`（新增文件）
