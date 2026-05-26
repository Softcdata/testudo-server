## ADDED Requirements

### Requirement: 系统必须提供备份恢复任务进度趋势接口
系统必须 (MUST) 提供一个只读统计接口，用于返回备份任务与恢复任务在固定日期窗口内的成功、失败、取消、执行中与未知计数趋势。

#### Scenario: 查询备份任务 7 天趋势
- **Given** 系统存在多条 `AppBackup.Status.History` 记录
- **When** 客户端请求 `GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/tasks/progress?type=backup&scope=all&range=7d`
- **Then** 响应必须使用标准 `Envelope`
- **And** `data.type` 必须为 `backup`
- **And** `data.buckets` 必须包含 7 个连续日期分桶
- **And** 每个分桶必须包含 `total`、`completed`、`failed`、`inProgress`、`canceled`、`unknown`
- **And** 没有数据的日期必须返回零值分桶

#### Scenario: 查询恢复任务 7 天趋势
- **Given** 系统存在多个 `AppRestore`
- **When** 客户端请求 `GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/tasks/progress?type=restore&scope=all&range=7d`
- **Then** 响应必须使用标准 `Envelope`
- **And** `data.type` 必须为 `restore`
- **And** `data.buckets` 必须包含 7 个连续日期分桶
- **And** `data.series` 必须返回恢复成功与恢复失败两个图例项

#### Scenario: 查询 30 天与 90 天趋势
- **Given** 系统存在跨多个日期的备份历史记录
- **When** 客户端分别请求 `range=30d` 与 `range=90d`
- **Then** `range=30d` 响应必须包含 30 个连续日期分桶
- **And** `range=90d` 响应必须包含 90 个连续日期分桶
- **And** 响应必须包含 `startTime` 与 `endTime`

### Requirement: 趋势接口必须复用现有来源识别口径
系统必须 (MUST) 复用现有 `AppBackup` 与 `AppRestore` 来源判定能力，按 `scope` 参数裁剪用户应用任务与容灾实例链路任务。

#### Scenario: 查询应用任务趋势
- **Given** 系统同时存在用户创建任务与容灾实例链路任务
- **When** 客户端请求 `scope=app`
- **Then** 响应只能统计来源不是 `disaster-instance` 的 `AppBackup` 与 `AppRestore`

#### Scenario: 查询容灾任务趋势
- **Given** 系统同时存在用户创建任务与容灾实例链路任务
- **When** 客户端请求 `scope=disaster`
- **Then** 响应只能统计来源为 `disaster-instance` 的 `AppBackup` 与 `AppRestore`

#### Scenario: 查询全部任务趋势
- **Given** 系统同时存在用户创建任务与容灾实例链路任务
- **When** 客户端请求 `scope=all`
- **Then** 响应必须同时统计用户应用任务与容灾实例链路任务
- **And** `data.sources` 必须按 `app` 与 `disaster` 返回来源拆分汇总

### Requirement: 趋势接口必须按执行时间分桶
系统必须 (MUST) 基于任务执行时间进行日期分桶，不得使用 `BackupRestoreStatistics.metadata.creationTimestamp` 代替单次任务执行时间。

#### Scenario: 备份历史使用执行时间
- **Given** 某条 `AppBackup.Status.History` 记录存在 `completionTimestamp`
- **When** 服务端计算备份趋势
- **Then** 该记录必须按 `completionTimestamp` 落入日期分桶

#### Scenario: 执行中备份使用开始时间
- **Given** 某条 `AppBackup.Status.History` 记录处于执行中状态
- **And** 该记录存在 `startTimestamp`
- **When** 服务端计算备份趋势
- **Then** 该记录必须按 `startTimestamp` 落入日期分桶

#### Scenario: 恢复任务使用恢复状态时间
- **Given** 某个 `AppRestore` 存在 `status.restoreStatus.completionTimestamp`
- **When** 服务端计算恢复趋势
- **Then** 该恢复任务必须按 `completionTimestamp` 落入日期分桶

#### Scenario: 分桶时区固定
- **Given** 客户端没有提供 `timezone`
- **When** 服务端计算日期分桶
- **Then** 服务端必须使用 `Asia/Shanghai`

### Requirement: 趋势接口必须严格校验查询参数
系统必须 (MUST) 对 `type`、`scope`、`range` 与 `timezone` 执行严格校验，并在参数非法时返回标准错误响应。

#### Scenario: 非法任务类型
- **When** 客户端请求 `type=job`
- **Then** 服务端必须返回 HTTP 400
- **And** 响应业务码必须为 `1000`

#### Scenario: 非法来源范围
- **When** 客户端请求 `scope=system`
- **Then** 服务端必须返回 HTTP 400
- **And** 响应业务码必须为 `1000`

#### Scenario: 非法日期窗口
- **When** 客户端请求 `range=365d`
- **Then** 服务端必须返回 HTTP 400
- **And** 响应业务码必须为 `1000`

#### Scenario: 非法时区
- **When** 客户端请求 `timezone=bad-zone`
- **Then** 服务端必须返回 HTTP 400
- **And** 响应业务码必须为 `1000`

### Requirement: 趋势接口不得改变现有统计接口语义
系统必须 (MUST) 保持现有统计接口的路径、响应结构与统计口径稳定。

#### Scenario: 现有备份统计保持不变
- **Given** 客户端请求 `GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/backups`
- **When** 本变更实施后
- **Then** 该接口仍必须返回现有总量统计结构

#### Scenario: 现有恢复统计保持不变
- **Given** 客户端请求 `GET /apis/backuprestorestatistics.testudo.softcdata.com/v1/restores`
- **When** 本变更实施后
- **Then** 该接口仍必须返回现有总量统计结构

#### Scenario: 查询接口无副作用
- **When** 客户端请求任务进度趋势接口
- **Then** 服务端不得修改任何 Kubernetes 对象
