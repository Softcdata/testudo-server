## Context
跨集群应用恢复依赖目标集群可访问源侧备份路径。  
当前创建 `AppRestore` 前仅检查目标集群状态，未检查恢复必需 BSL 是否可用。  
当必需 BSL 缺失、处于 `Unavailable` 阶段时，用户会在创建后进入长时间等待，定位成本高。

## Goals
- 提供显式可调用的恢复前置校验接口。
- 将同一校验逻辑嵌入 `createAppRestore`，形成后端硬闸门。
- 固定触发步骤、固定轮询节奏、固定超时边界，输出可诊断结果。

## Non-Goals
- 不改动前端页面流程。
- 不改动 `AppBackup` 创建与动作接口。
- 不改动 Velero Controller 行为。

## Decisions

### Decision 1: 必需 BSL 计算规则
- 输入：`backupSource`、`targetCluster`
- 读取 `AppBackup`：
  - `sourceCluster = appBackup.spec.cluster`
  - `storageRepository = appBackup.spec.template.storageLocation`
- 计算：
  - `requiredBSL = <storageRepository>-<sourceCluster>`
- 结论：前置校验与创建硬闸门全部使用该规则，禁止使用 `<storageRepository>-<targetCluster>` 代替。

### Decision 2: 缺失 BSL 的触发方式
- 当目标集群读取 `requiredBSL` 返回 `NotFound`：
  - Patch 目标 `Cluster` 注解：
    - `testudo.softcdata.com/ensure-storage=<storageRepository>`
    - `testudo.softcdata.com/ensure-storage-source-cluster=<sourceCluster>`
- 由 Operator 消费注解并创建目标侧 BSL。

### Decision 3: 固定轮询节奏
- 轮询间隔：`1` 秒。
- 轮询总次数：`waitSeconds`。
- 默认 `waitSeconds=20`，最大 `60`。
- 判定条件：
  - 任意一次 `phase=Available`：成功。
  - 达到最大次数仍非 `Available`：失败。

### Decision 4: 统一校验 Service
- 新增 `RestorePreflightVerifier`，封装以下步骤：
  - 计算 `requiredBSL`
  - 检查目标侧 BSL
  - 触发信号
  - 轮询等待
  - 返回结构化结果
- 调用入口：
  - `POST /apprestores/preflight/validate`
  - `POST /apprestores` 创建前硬闸门

## Risks
- Operator 未在超时窗口内消费信号，导致前置校验超时。
- 目标集群网络抖动导致短时读取失败。
- BSL 状态从 `New` 到 `Available` 超过默认窗口。

## Mitigation
- 提供 `waitSeconds` 参数，允许调用方按场景放大窗口。
- 失败响应返回 `requiredBSL`、`phase`、`reason`，便于快速定位。
- 创建接口复用同一校验逻辑，避免前置接口与创建流程判定不一致。
