# 任务列表：增强实例同步状态与同步历史接口

## 1. 规范与文档
- [x] 1.1 评审 `sync-status` 的 `lastSyncStatus` 字段结构和来源规则。
- [x] 1.2 评审 `sync-history` 的路径、查询参数、DTO、排序、分页和 summary 规则。
- [x] 1.3 更新 Swagger/OpenAPI：同步状态接口、同步历史接口、相关 DTO 和响应示例。
- [x] 1.4 更新 RunAPI/Apipost：更新 `GET /instances/:name/sync-status`，新增 `GET /instances/:name/sync-history`。
- [x] 1.5 更新本地 RunAPI 证据文件，记录接口链路、字段语义、错误语义和目标 ID。

## 2. Server 实现
- [x] 2.1 在 `SubResourceStatusDTO` 增加 `lastSyncStatus` 字段。
- [x] 2.2 新增 `LastSyncStatusDTO` 与 `SyncHistoryItemDTO`。
- [x] 2.3 从 DataSync/ResourceSync `status.history` 中选择最新记录并投影为 `lastSyncStatus`。
- [x] 2.4 新增 `GET /instances/:name/sync-history` 路由。
- [x] 2.5 实现 `source=syncRecord`，读取 DataSync/ResourceSync `status.history`。
- [x] 2.6 实现 `source=operation`，复用现有 DisasterOperation 历史并筛选 `syncdata`、`syncresource`、`synconce`。
- [x] 2.7 实现 `syncType`、`status`、`source` 参数校验和过滤。
- [x] 2.8 实现固定排序、分页和 `meta.summary`。

## 3. 测试
- [x] 3.1 单测覆盖 `sync-status` 返回 `dataSync.lastSyncStatus`。
- [x] 3.2 单测覆盖 `sync-status` 返回 `resourceSync.lastSyncStatus`。
- [x] 3.3 单测覆盖无 history 时省略 `lastSyncStatus`。
- [x] 3.4 单测覆盖 `sync-history` 默认读取 sync record。
- [x] 3.5 单测覆盖 `source=operation` 只返回同步相关 operation。
- [x] 3.6 单测覆盖非法 `syncType`、`status`、`source` 返回 `400 code=1000`。
- [x] 3.7 单测覆盖排序、分页和 summary 基于过滤后、分页前集合计算。

## 4. 验证
- [x] 4.1 `go test ./internal/apis/disaster_instance/v1`
- [x] 4.2 `openspec validate add-instance-sync-history-api --strict`
- [x] 4.3 `go run ./tools/openapi validate --spec openspec/specs/disaster-server-openapi.yaml`
- [x] 4.4 `git diff --check`
