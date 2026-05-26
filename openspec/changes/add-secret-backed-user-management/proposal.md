# Change: 新增基于 Secret 的用户管理能力

## Why
当前 `POST /login` 在 `internal/middleware/jwt.go` 中使用硬编码账号 `admin/123456` 进行鉴权，无法满足生产环境的账号管理需求。
当前服务缺少完整用户管理闭环，运维人员无法通过 API 完成账号查询、删除和密码轮换。
为了让多副本实例共享同一份账号数据，同时满足最小化改动原则，需要把账号数据落在 Kubernetes `Secret` 资源中。

## What Changes
- 新增用户管理能力：提供 `POST /api/v1/users` 接口用于新增账号。
- 新增用户列表能力：提供 `GET /api/v1/users` 接口用于返回账号列表。
- 用户列表响应对齐 API 规范：回显 `data.items` 中的 `username`、`email`、`role`、`status`、`createdAt` 字段，并返回 `meta.pagination` 分页信息。
- 新增用户状态管理能力：提供 `PATCH /api/v1/users/:username/status` 接口用于禁用和启用账号。
- 新增用户删除能力：提供 `DELETE /api/v1/users/:username` 接口用于删除账号。
- 新增密码更新能力：提供 `PATCH /api/v1/users/:username/password` 接口用于更新账号密码。
- 新增用户账号存储模型：在 `disaster-system` 命名空间维护 `Secret/disaster-server-users`。
- 用户数据模型统一包含 `username`、`email`、`role`、`status`、`createdAt` 字段，其中 `role` 固定为 `admin`。
- 内置当前管理员账户：系统启动时确保存在 `admin/123456` 对应账号（密码以哈希形式存储）。
- 登录鉴权改造：`/login` 不再使用硬编码账号，改为读取 `Secret` 中的账号数据并校验密码哈希。
- 登录状态校验改造：当用户 `status=disabled` 时，登录请求固定返回失败。
- 启动初始化改造：服务在启动流程中执行一次账号 Secret 初始化，确保存在可登录的管理员账号。
- 本阶段不引入用户角色权限控制：用户管理接口只做参数校验、重复校验和数据写入。
- 测试补充：覆盖新增账号、账号列表、删除账号、修改密码、禁用账号、登录成功、登录失败、并发写冲突重试等关键路径。

## Impact
- 受影响规范：
  - `openspec/specs/user-management/spec.md`（新能力）
- 受影响代码：
  - `internal/middleware/jwt.go`
  - `internal/router/router.go`
  - `internal/apis`（新增 `user` 模块）
  - `internal/common/common.go`（复用命名空间常量）
