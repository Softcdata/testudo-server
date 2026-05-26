# Tasks: 新增容灾配置名称列表接口并返回状态

## 1. Implementation
- [x] 1.1 新增 `DisasterConfigNameDTO`（包含 `id`/`name`/`status`）
- [x] 1.2 实现 `GET /configs/names`（基于 Lister，返回全量不分页）
- [x] 1.3 单元测试覆盖 `status` 字段
- [x] 1.4 更新增量规范 `specs/disaster_config/spec.md`
- [x] 1.5 写入 Apipost（新增接口文档 + 成功响应示例）
- [x] 1.6 验证：`go test ./internal/apis/disaster_config/v1`，`openspec validate add-config-status-to-config-names --strict`
