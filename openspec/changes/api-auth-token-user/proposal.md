# Proposal: 使用 Token 提取用户信息并移除 X-Tenant-ID

## 1. 背景
目前 `disaster-server` 的 API 依赖 `X-Tenant-ID` Header 来传递租户信息。但现在系统已集成了 JWT Authentication，Token 中已经包含用户信息（ID 和 Username）。为了简化前端调用并统一认证机制，我们计划从 Token 中直接提取用户信息，并移除对 `X-Tenant-ID` 的依赖，同时确保 Event 中的操作者字段能正确记录 Token 中的用户。

## 2. 目标
1.  **移除 X-Tenant-ID**: API 不再强制要求客户端传递 `X-Tenant-ID` Header（或将其设为可选/废弃）。
2.  **提取 Token 用户**: 中间件从 JWT Token 中解析 `username` 和 `userID`。
3.  **注入 Context**: 将解析出的用户信息注入 Request Context，供后续 Handler 使用。
4.  **事件记录**: 确保 `EventHandler` 和其它业务逻辑在记录操作日志时，使用 Token 中的 `username` 作为操作者 (`TriggeredBy` / `User`)。

## 3. 设计方案

### 3.1 Middleware 变更
*   修改 `internal/middleware/jwt.go` (或相关中间件)：
    *   确保 JWT 解析成功后，将 UserInfo (`id`, `username`) 放入 Hertz 上下文 (例如使用 `c.Set("user_id", ...)` 和 `c.Set("username", ...)` )。
    *   移除或弱化 `TenantID` 中间件中对 Header 的强制检查。如果 Header 缺失，尝试使用 Token 中的 Default Tenant (如果适用) 或者直接忽略 Tenant 概念（如果是单租户系统）。

### 3.2 Handler 变更
*   在 Handler 中（例如 `RecordAction` 或业务函数），不再从 Header 取 `X-Tenant-ID`。
*   改为从 Context 取 `username`，并将其作为 Event 的 `User` 字段。

### 3.3 规范变更
*   更新 API 规范文档，说明 Authorization Header 是必需的，而 X-Tenant-ID 不再需要。

## 4. 影响范围
*   `internal/middleware/`: 认证与上下文注入逻辑。
*   `internal/apis/`: 所有需要记录 Audit Log 或 Event 的 Handler。
*   前端/客户端: 调用 API 时需确保 Header 包含 Bearer Token，可移除 X-Tenant-ID。

## 5. 验证计划
*   调用 API (如 Create Policy) 不带 X-Tenant-ID，带有效 Token。
*   检查生成的 Event，确认 `User` 字段正确记录了 Token 中的用户名。
