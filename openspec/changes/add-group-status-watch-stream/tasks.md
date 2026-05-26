## 1. 路由与处理器

- [x] 1.1 在容灾组路由中注册 `GET /watch/groups/status`
- [x] 1.2 在容灾组路由中注册 `GET /watch/groups/status/:name`
- [x] 1.3 新增 `watchGroupStatuses`，使用 List+Watch 获取 `DisasterGroup` 状态事件流
- [x] 1.4 新增 `watchGroupStatus`，使用 `metadata.name=<name>` 监听单个容灾组状态事件流
- [x] 1.5 统一 `DisasterGroupDTO` 构建逻辑，确保 HTTP 查询接口与 WS 事件流输出字段一致

## 2. 测试与回归

- [ ] 2.1 新增状态事件流处理器单测，覆盖列表流与详情流
- [x] 2.2 新增 DTO 转换单测，覆盖 `status.summary`、`status.fsmState`、`status.availableOperations`
- [x] 2.3 回归验证现有 `/watch/groups/operations*` 接口无行为变化

## 3. 规范与校验

- [x] 3.1 编写 `disaster-group` 增量规范，定义状态事件流接口契约
- [x] 3.2 执行 `openspec validate add-group-status-watch-stream --strict`
- [x] 3.3 在 Apipost 的「容灾组管理」目录新增状态事件流接口文档（`/watch/groups/status` 与 `/watch/groups/status/:name`）
