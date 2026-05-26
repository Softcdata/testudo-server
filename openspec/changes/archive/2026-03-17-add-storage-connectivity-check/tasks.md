# Tasks: 跨集群应用恢复前置存储校验

## 1. 校验 Service
- [x] 1.1 新增 `RestorePreflightVerifier` 接口与实现
- [x] 1.2 固定提取 `sourceCluster` 与 `storageRepository`（来源 `AppBackup.Spec`）
- [x] 1.3 固定计算 `requiredBSL=<storageRepository>-<sourceCluster>`
- [x] 1.4 实现缺失 BSL 信号注解写入：
  - [x] `testudo.softcdata.com/ensure-storage`
  - [x] `testudo.softcdata.com/ensure-storage-source-cluster`
- [x] 1.5 实现固定轮询：
  - [x] 间隔 `1` 秒
  - [x] 总次数 `waitSeconds`
  - [x] 默认 `20`
  - [x] 最大 `60`

## 2. 前置校验接口
- [x] 2.1 在 `app_restore` 模块新增 `POST /apprestores/preflight/validate`
- [x] 2.2 新增请求结构：
  - [x] `backupSource`
  - [x] `targetCluster`
  - [x] `waitSeconds`
- [x] 2.3 新增响应 `meta` 结构：
  - [x] `required_bsl`
  - [x] `source_cluster`
  - [x] `target_cluster`
  - [x] `storage_repository`
  - [x] `state`
  - [x] `error`

## 3. 创建接口硬闸门
- [x] 3.1 在 `createAppRestore` 中接入 `RestorePreflightVerifier`
- [x] 3.2 将硬闸门放在 `Create(AppRestore)` 之前
- [x] 3.3 校验失败直接返回失败响应，不进入创建流程

## 4. 与 Operator 联动校验
- [x] 4.1 增加对 `ensure-storage-source-cluster` 注解的兼容调用
- [x] 4.2 与 operator 提案 `support-storage-connectivity-signal` 对齐字段名

## 5. 测试
- [x] 5.1 单测：`requiredBSL` 计算规则
- [x] 5.2 单测：目标侧 BSL 已 `Available` 直接成功
- [x] 5.3 单测：目标侧 BSL 缺失时写入双注解并等待成功
- [x] 5.4 单测：目标侧 BSL `Unavailable` 返回失败
- [x] 5.5 单测：超时返回失败
- [x] 5.6 集成测试：`preflight/validate` 成功路径
- [x] 5.7 集成测试：`createAppRestore` 在校验失败时被阻断
