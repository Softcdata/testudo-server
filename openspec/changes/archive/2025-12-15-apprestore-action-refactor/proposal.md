# Refactor AppRestore Action API

## Status
Implemented

## Summary
本提案旨在重构 `AppRestore` 的动作触发机制，从当前的通过 `PUT` 修改 `Spec` 字段的方式，改为符合 `api-style.md` 规范的专用动作接口。

## Motivation
当前 `disaster-server` 触发 `AppRestore` 的取消或重试操作时，需要客户端通过 `PUT` 请求更新整个 `AppRestore` 对象的 `spec.action` 字段。这种方式存在以下问题：
1.  **不符合 RESTful 风格**：动作应该通过特定的动词或子资源触发，而不是修改资源状态。
2.  **并发冲突风险**：客户端需要先 GET 再 PUT，如果期间有其他修改（如 Controller 更新 Status），可能导致版本冲突。
3.  **易用性差**：客户端需要构造完整的 `Spec` 结构，只需触发一个动作却需要发送整个对象。
4.  **不符合 API 规范**：`openspec/guides/api-style.md` 明确建议使用 `POST /resources/{id}/actions/{action}` 形式。

## Proposed Changes

### 1. 新增 API 路由
在 `AppRestoreHandler` 中新增处理动作的路由：
- `POST /apprestores/:name/actions/:type`
  - `:type` 可选值：`cancel`, `retry`

### 2. 后端逻辑实现
在 `AppRestoreHandler` 中实现 `executeAction` 方法：
1.  解析 URL 中的 `name` 和 `type`。
2.  校验 `type` 是否合法（`cancel` 或 `retry`，注意大小写转换）。
3.  获取当前的 `AppRestore` 对象。
4.  修改对象：
    ```go
    existing.Spec.Action = &dapisv1.RestoreAction{
        Type:      actionType, // "cancel" or "retry"
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
`POST /api/v1/apprestores/{name}/actions/{type}`

**Parameters**
- `name`: AppRestore 资源名称
- `type`: 动作类型，支持 `cancel`, `retry` (不区分大小写)

**Response**
```json
{
  "code": 0,
  "message": "Action triggered successfully",
  "data": {
    "type": "retry",
    "request_at": "2023-10-27T10:00:00Z"
  },
  "trace_id": "..."
}
```

## Implementation Plan
1.  修改 `internal/apis/app_restore/v1/handler.go`，添加 `executeAction` 方法。
2.  在 `internal/apis/app_restore/v1/router.go` 添加新的路由映射。
3.  测试新接口。
