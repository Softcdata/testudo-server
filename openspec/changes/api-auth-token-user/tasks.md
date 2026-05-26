# Tasks: 使用 Token 提取用户信息并移除 X-Tenant-ID

## 1. Middleware Enhancement
- [ ] 修改 `internal/middleware/jwt.go`: 确保 `IdentityHandler` 或成功回调中将 UserInfo 注入到 `c` (Set keys like "userID", "userName")。
- [ ] 修改 `internal/middleware/trace.go` (or `tenant.go` if exists): 移除对 `X-Tenant-ID` 的强校验，或者将其改为从 Context/Token 获取（如果租户信息也在 Token 中）。

## 2. API Handler Updates
- [ ] 全局搜索 `X-Tenant-ID` 的使用处。
- [ ] 修改 `internal/apis/event/v1` 及其他模块：将获取 User 的逻辑从 `c.Request.Header.Get("X-Tenant-ID")` 改为 `c.GetString("userName")`。
- [ ] 确保 `helper.ReportTask*` 系列函数传递正确的 User 参数。

## 3. Verification
- [ ] 测试 `/login` 获取 Token。
- [ ] 使用 Token 调用业务接口（不带 Tenant Header）。
- [ ] 验证操作成功且 Event 记录中包含正确的 Username。
