# Refactor AppBackup Action API

## Status
Implemented

## Summary
本提案旨在重构 `AppBackup` 的动作触发机制，从当前的通过 `PUT` 修改 `Spec` 字段的方式，改为符合 `api-style.md` 规范的专用动作接口。

## Motivation
当前 `disaster-server` 触发 `AppBackup` 的手动备份或重试操作时，需要客户端通过 `PUT` 请求更新整个 `AppBackup` 对象的 `spec.action` 字段。这种方式存在以下问题：
1.  **不符合 RESTful 风格**：动作应该通过特定的动词或子资源触发，而不是修改资源状态。
2.  **并发冲突风险**：客户端需要先 GET 再 PUT，如果期间有其他修改（如 Controller 更新 Status），可能导致版本冲突。
3.  **易用性差**：客户端需要构造完整的 `Spec` 结构，只需触发一个动作却需要发送整个对象。
4.  **不符合 API 规范**：`openspec/guides/api-style.md` 明确建议使用 `POST /resources/{id}/actions/{action}` 形式。

## Proposed Changes

### 1. 新增 API 路由
在 `AppBackupHandler` 中新增处理动作的路由：
- `POST /appbackups/:name/actions/backup`：触发立即备份。
- `POST /appbackups/:name/actions/retry`：触发重试。

或者统一为一个接口（推荐）：
- `POST /appbackups/:name/actions`
- Body: `{"type": "Backup"}` 或 `{"type": "Retry"}`

根据 `api-style.md` 的示例 `POST /backups/{id}/actions/cancel`，建议采用 URL 路径参数或 Body 参数均可。考虑到 `BackupAction` 结构体目前只有 `Type` 和 `RequestAt`，且 `RequestAt` 由服务端生成，使用 Body 传递 `type` 更具扩展性，或者直接使用路径参数。

**建议方案**：
为了保持与 `api-style.md` 高度一致，采用子资源路径方式：
- `POST /appbackups/:name/actions/:type`
  - `:type` 可选值：`backup`, `retry`

### 2. 后端逻辑实现
在 `AppBackupHandler` 中实现 `handleAction` 方法：
1.  解析 URL 中的 `name` 和 `type`。
2.  校验 `type` 是否合法（`Backup` 或 `Retry`，注意大小写转换）。
3.  获取当前的 `AppBackup` 对象。
4.  修改对象：
    ```go
    existing.Spec.Action = &dapisv1.BackupAction{
        Type:      actionType, // "Backup" or "Retry"
        RequestAt: metav1.Now(),
    }
    ```
5.  执行 `Update` 操作（包含重试机制以处理冲突）。
6.  返回成功响应（200 OK），可返回更新后的对象或仅返回操作状态。

### 3. 废弃旧方式
虽然短期内可能保留 `PUT` 接口的兼容性，但应标记通过 `PUT` 修改 `action` 的方式为不推荐，并最终移除。

## API Design

### Trigger Action
**Request**
`POST /api/v1/appbackups/{name}/actions/{type}`

**Parameters**
- `name`: AppBackup 资源名称
- `type`: 动作类型，支持 `backup`, `retry` (不区分大小写)

**Response**
```json
{
  "code": 0,
  "message": "Action triggered successfully",
  "data": {
    "type": "Backup",
    "request_at": "2023-10-27T10:00:00Z"
  },
  "trace_id": "..."
}
```

## Implementation Plan
1.  修改 `internal/apis/app_backup/v1/handler.go`，添加 `executeAction` 方法。
2.  在 `internal/router/router.go` 或 handler 注册处添加新的路由映射。
3.  测试新接口。
