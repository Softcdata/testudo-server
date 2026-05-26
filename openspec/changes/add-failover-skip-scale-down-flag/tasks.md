## 1. 参数透传实现

- [x] 1.1 在实例 action handler 中解析 `skipScaleDownSource` / `SkipScaleDownSource`
- [x] 1.2 仅在 `operation=failover` 时将该参数写入 `DisasterOperationSpec.SkipScaleDownSource`
- [x] 1.3 在组 action handler 中解析并透传同名参数
- [x] 1.4 在 failover 场景写入 annotation `testudo.softcdata.com/skip-scale-down-source=true` 作为跨版本兼容兜底

## 2. 测试

- [x] 2.1 新增实例 action 单元测试，验证 failover 透传成功
- [x] 2.2 新增实例 action 回归测试，验证非 failover 不写入该参数
- [x] 2.3 新增组 action 单元测试，验证 failover 透传成功

## 3. 质量门禁

- [x] 3.1 运行 `go test ./internal/apis/disaster_instance/v1 ./internal/apis/disaster_group/v1`
- [x] 3.2 运行 `openspec validate add-failover-skip-scale-down-flag --strict`
