# Change: 跨集群应用恢复前置存储校验

## Why
当前 `POST /apis/apprestores.testudo.softcdata.com/v1/apprestores` 在创建 `AppRestore` 前，仅校验目标集群就绪状态，未校验跨集群恢复依赖的目标侧 BSL 可用状态。
在跨集群恢复场景中，真正依赖的 BSL 名称是 `<StorageRepository>-<SourceCluster>`。当该 BSL 缺失、处于 `Unavailable` 阶段时，恢复任务会进入长时间等待，用户无法在创建前得到明确结果。  
需要新增一个可显式调用的前置校验接口，并在创建接口内同步执行硬校验，保证恢复请求进入系统前已经满足存储前置条件。

## What Changes

### 1. 新增恢复前置校验接口
新增接口：
`POST /apis/apprestores.testudo.softcdata.com/v1/apprestores/preflight/validate`

请求参数：
- `backupSource`：源 `AppBackup` 名称
- `targetCluster`：目标集群名称
- `waitSeconds`：等待秒数，默认 `20`，最大 `60`

校验流程：
1. 读取 `backupSource` 对应 `AppBackup`。
2. 从 `AppBackup.Spec` 固定提取：
`sourceCluster = spec.cluster`，`storageRepository = spec.template.storageLocation`。
3. 计算恢复必需 BSL：
`requiredBSL = <storageRepository>-<sourceCluster>`。
4. 在目标集群 `velero` 命名空间读取 `requiredBSL`。
5. 当 BSL 不存在时，给目标 `Cluster` 打信号注解，触发 Operator 创建。
6. 按固定节奏轮询：每 `1` 秒读取一次，总次数等于 `waitSeconds`。
7. 任意一次读取到 `phase=Available` 即返回成功。
8. 在最大次数内未达到 `Available` 即返回失败以及明确原因。

### 2. 创建应用恢复接口增加硬闸门
在 `createAppRestore` 流程中，新增与前置校验接口一致的硬校验步骤。  
`createAppRestore` 在执行 `Create(AppRestore)` 前，必须通过 `requiredBSL` 可用性校验。  
当校验失败时，接口直接返回失败，不进入创建流程。

### 3. 复用统一 Service
新增并复用统一的恢复前置校验 Service（建议命名 `RestorePreflightVerifier`），由以下入口共用：
- `apprestores/preflight/validate`
- `createAppRestore` 硬闸门

Service 对外输出统一结构：
- `valid`
- `requiredBSL`
- `sourceCluster`
- `targetCluster`
- `storageRepository`
- `phase`
- `reason`

## Impact
- 受影响规范：
  - `openspec/specs/disaster_storage/spec.md`
- 受影响代码：
  - `internal/apis/app_restore/v1/handler.go`
  - `internal/apis/app_restore/v1/router.go`
  - `internal/apis/app_restore/v1/types.go`
  - `internal/service/verifier/*`
- 关联依赖：
  - 依赖 `disaster-operator` 的 `Cluster ensure-storage` 信号处理能力扩展，以支持 `sourceCluster` 后缀语义
