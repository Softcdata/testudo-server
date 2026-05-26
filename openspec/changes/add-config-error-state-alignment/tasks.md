## 1. 实例 DTO 映射

- [x] 1.1 修改 `internal/apis/disaster_instance/v1/types.go`。
- [x] 1.2 在 `determineCurrentState` 增加 `ConfigError -> Error` 分支。
- [x] 1.3 保持 `DisasterInstanceDTO.Status.FsmState` 原值透传。

## 2. 组聚合状态

- [x] 2.1 修改 `internal/apis/disaster_group/v1/handler.go` 的 `computeGroupFsmState`。
- [x] 2.2 在错误优先级分支新增 `ConfigError` 命中。
- [x] 2.3 保持 `FailingOver`、`FailingBack` 优先级不变。

## 3. 组成员派生状态

- [x] 3.1 修改 `deriveGroupMemberStatus` 的配置异常分支。
- [x] 3.2 `config NotFound` 场景输出 `state=ConfigError`，并补齐 `ConfigNotFound` 语义。
- [x] 3.3 `config Error` 场景输出 `state=ConfigError`。
- [x] 3.4 `config NotReady` 场景输出 `state=ConfigError`。
- [x] 3.5 保持 `state=NotFound` 时不覆盖状态。

## 4. 组状态筛选

- [x] 4.1 修改 `matchStatus` 的 `error` 筛选分支。
- [x] 4.2 增加 `ConfigError` 命中条件。

## 5. 单元测试

- [x] 5.1 在 `internal/apis/disaster_instance/v1/handler_test.go` 新增 `determineCurrentState(ConfigError)` 用例。
- [x] 5.2 在 `internal/apis/disaster_group/v1/handler_test.go` 新增 `ConfigError -> Degraded` 聚合用例。
- [x] 5.3 新增 `FailingOver` 对 `ConfigError` 的优先级覆盖用例。
- [x] 5.4 新增 `deriveGroupMemberStatus(config NotReady)` 输出 `ConfigError` 用例。
- [x] 5.5 新增 `deriveGroupMemberStatus(config NotFound)` 输出 `ConfigError` 用例。
- [x] 5.6 新增 `status=error` 命中 `ConfigError` 用例。

## 6. 验证

- [x] 6.1 运行 `go test ./internal/apis/disaster_instance/v1 -count=1`。
- [x] 6.2 运行 `go test ./internal/apis/disaster_group/v1 -count=1`。
- [x] 6.3 运行 `go test ./... -count=1`。
- [x] 6.4 运行 `openspec validate add-config-error-state-alignment --strict`。
