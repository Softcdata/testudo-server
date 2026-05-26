# Tasks: 新增基于 Secret 的用户管理能力

## 1. Implementation
- [x] 1.1 新增用户存储层：定义包含 `username/email/role/status/createdAt` 的 `Secret/disaster-server-users` 文档结构与读写接口。
- [x] 1.2 在用户存储层实现 `RetryOnConflict` 更新流程，保证并发新增账号不会覆盖写入。
- [x] 1.3 在启动路径中增加 `EnsureUserSecret`，在服务监听端口之前完成内置管理员账号 `admin/123456` 初始化。
- [x] 1.4 在 Secret 已存在但缺少 `admin` 账号时补齐内置管理员账号。
- [x] 1.5 改造 `internal/middleware/jwt.go` 的 `Authenticator`，将硬编码账号校验替换为 Secret 账号校验。
- [x] 1.6 引入密码哈希策略：新增账号写入哈希值，登录流程执行哈希比对和 `status` 状态校验。
- [x] 1.7 新增 `POST /api/v1/users` 接口，支持新增账号并返回脱敏结果。
- [x] 1.8 新增 `GET /api/v1/users` 接口，支持返回用户列表并对齐 `data.items` 与 `meta.pagination`。
- [x] 1.9 新增 `PATCH /api/v1/users/:username/status` 接口，支持将账号设置为 `disabled` 和 `active`。
- [x] 1.10 新增 `DELETE /api/v1/users/:username` 接口，支持删除普通账号并拒绝删除内置 `admin`。
- [x] 1.11 新增 `PATCH /api/v1/users/:username/password` 接口，支持更新用户密码哈希。
- [x] 1.12 角色固定规则：新增账号和查询返回中 `role` 字段固定为 `admin`。
- [x] 1.13 用户管理接口不做角色权限校验，保持实现焦点在用户数据管理逻辑。

## 2. Tests
- [x] 2.1 新增用户存储层单元测试：覆盖 Secret 不存在、创建成功、并发冲突重试成功。
- [x] 2.2 新增登录单元测试：覆盖正确密码登录、密码错误、账号不存在、账号禁用。
- [x] 2.3 新增用户接口单元测试：覆盖新增成功、用户名重复、请求参数不合法、`role=admin` 固定值。
- [x] 2.4 新增用户状态接口单元测试：覆盖禁用成功、启用成功、账号不存在。
- [x] 2.5 新增用户列表接口单元测试：覆盖列表回显字段、分页、排序、关键字过滤。
- [x] 2.6 新增用户删除与改密测试：覆盖删除成功、删除内置管理员失败、改密成功、改密参数非法。

## 3. Validation
- [x] 3.1 运行 `go test ./internal/middleware ./internal/apis/...` 并记录结果。
- [x] 3.2 运行 `go test ./internal/userstore` 并记录结果。
- [x] 3.3 运行 `openspec validate add-secret-backed-user-management --strict`。
