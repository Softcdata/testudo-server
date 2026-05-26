## Context

服务端承担状态归一化与组聚合计算职责。operator 新增 `ConfigError` 后，服务端必须在以下接口持续保持同一语义：

- `GET /instances`
- `GET /instances/:name`
- `GET /groups`
- `GET /groups/:name`
- `GET /groups/instance-picker`
- `GET /groups/:name/instances`

## 目标

- 目标 1：`ConfigError` 在实例 DTO、组 DTO、组成员 DTO 中保持可见。
- 目标 2：组聚合优先级保持确定顺序，新增状态不改变既有进行中优先级。
- 目标 3：`status=error` 筛选命中 `ConfigError`。

## 规则 A：实例 currentState 映射

文件：`internal/apis/disaster_instance/v1/types.go`

函数：`determineCurrentState`

映射规则：
1. `fsmState == ""`，返回 `Pending`。
2. `fsmState == Failed`，返回 `Error`。
3. `fsmState == ConfigError`，返回 `Error`。
4. 其余状态，返回 `Running`。

注意：
- `DisasterInstanceDTO.Status.FsmState` 必须透传原始状态值。

## 规则 B：组聚合状态计算

文件：`internal/apis/disaster_group/v1/handler.go`

函数：`computeGroupFsmState`

聚合优先级按以下固定顺序执行：
1. 统计中存在 `FailingOver`，返回 `FailingOver`，操作集为空。
2. 统计中存在 `FailingBack`，返回 `FailingBack`，操作集为空。
3. 统计中存在 `Failed`，返回 `Degraded`，操作集为空。
4. 统计中存在 `ConfigError`，返回 `Degraded`，操作集为空。
5. 全量为 `Active`，返回 `Active`，操作集为 `reprotect`。
6. 全量为 `Paused`，返回 `Paused`，操作集为 `resume`。
7. 全量为 `Protected`，返回 `Protected`，操作集为 `failover/pause/synconce/syncdata/syncresource`。
8. 统计中存在 `Pending`，返回 `Initializing`，操作集为空。
9. 统计中存在 `Initializing`，返回 `Initializing`，操作集为空。
10. 其余场景，返回 `PartialProtected`，操作集为 `failover/pause/synconce`。

## 规则 C：组成员状态派生

文件：`internal/apis/disaster_group/v1/handler.go`

函数：`deriveGroupMemberStatus`

步骤：
1. `state` 初值来自实例原始 `fsmState`。
2. `state == ""` 时改为 `Unknown`。
3. 若实例 `reason` 非空，且 `state` 不是 `Failed`，且 `state` 不是 `NotFound`，将 `state` 设为 `Failed`。
4. `configErr` 为 NotFound 时：
   - 写入 `reason=ConfigNotFound`（仅在 reason 为空时写入）。
   - 写入 `message=DisasterConfig <name> not found`（仅在 message 为空时写入）。
   - `state` 在非 `NotFound` 场景下设为 `ConfigError`。
5. `config.status.status` 为 `Error` 时：
   - reason/message 优先透传 config 状态字段。
   - 缺省文案按现有兜底规则。
   - `state` 在非 `NotFound` 场景下设为 `ConfigError`。
6. `config.status.status` 为 `NotReady` 时：
   - reason/message 优先透传 config 状态字段。
   - 缺省文案按现有兜底规则。
   - `state` 在非 `NotFound` 场景下设为 `ConfigError`。

## 规则 D：状态筛选

文件：`internal/apis/disaster_group/v1/handler.go`

函数：`matchStatus`

当筛选值为 `error` 时，命中条件更新为：
- `Failed`
- `ConfigError`
- `FailingOver`
- `FailingBack`

## 向后兼容

- 不修改接口路径。
- 不修改响应主结构。
- 旧前端读取 `currentState` 依旧可得到 `Error` 分类。
- 新前端读取 `status.fsmState` 可直接识别 `ConfigError`。

## 测试矩阵

文件：`internal/apis/disaster_instance/v1/handler_test.go`

- 新增 `determineCurrentState(ConfigError) == Error`。
- 新增 DTO 转换后 `status.fsmState == ConfigError` 保留验证。

文件：`internal/apis/disaster_group/v1/handler_test.go`

- 新增 `computeGroupFsmState`：成员包含 `ConfigError` 时输出 `Degraded`。
- 新增优先级回归：同时存在 `FailingOver` 与 `ConfigError` 时输出 `FailingOver`。
- 新增 `deriveGroupMemberStatus`：配置 NotReady 输出 `ConfigError`。
- 新增 `deriveGroupMemberStatus`：配置 NotFound 输出 `ConfigError + ConfigNotFound`。
- 新增 `status=error` 筛选命中 `ConfigError`。

## 验证命令

- `go test ./internal/apis/disaster_instance/v1 -count=1`
- `go test ./internal/apis/disaster_group/v1 -count=1`
- `go test ./... -count=1`
- `openspec validate add-config-error-state-alignment --strict`
