## ADDED Requirements

### Requirement: 用户账号必须存储在 Kubernetes Secret
系统必须 (MUST) 将可登录账号存储在 `disaster-system` 命名空间的 `Secret/disaster-server-users` 中。

#### Scenario: 服务首次启动创建账号 Secret
- **WHEN** 服务进入启动流程且读取到 `Secret/disaster-server-users` 不存在
- **THEN** 服务必须创建该 Secret
- **AND** Secret 的 `data["users.json"]` 必须包含 `admin` 初始账号

### Requirement: 系统必须内置当前管理员账户
系统必须 (MUST) 内置当前管理员账户 `admin/123456`，并在启动时确保该账户存在。

#### Scenario: Secret 缺失管理员账号时补齐内置管理员
- **WHEN** 服务启动时读取到 `Secret/disaster-server-users` 存在但缺少 `admin` 用户
- **THEN** 服务必须补齐 `admin` 用户记录
- **AND** 补齐时密码 `123456` 必须以哈希形式写入

### Requirement: 登录接口必须基于 Secret 账号进行鉴权
系统必须 (MUST) 在处理 `POST /login` 时读取账号 Secret 并校验密码哈希。

#### Scenario: 账号密码正确时登录成功
- **WHEN** 客户端提交存在于 Secret 中的 `username` 且密码哈希比对成功
- **THEN** 服务返回 `200`
- **AND** 响应 `data` 中包含 `accessToken`、`refreshToken`、`expire`

#### Scenario: 账号不存在时登录失败
- **WHEN** 客户端提交的 `username` 在 Secret 文档中不存在
- **THEN** 服务返回 `401`
- **AND** 响应不得泄露 Secret 文档细节

#### Scenario: 账号被禁用时登录失败
- **WHEN** 客户端提交的 `username` 存在且该账号 `status=disabled`
- **THEN** 服务返回 `403`
- **AND** 服务不得签发 `accessToken`

### Requirement: 系统必须提供新增账号接口
系统必须 (MUST) 提供 `POST /api/v1/users` 接口用于新增账号。

#### Scenario: 新增账号成功
- **WHEN** 客户端调用 `POST /api/v1/users` 且请求体校验通过
- **THEN** 服务返回 `201`
- **AND** 新账号被写入 `Secret/disaster-server-users`
- **AND** 响应不包含明文密码与密码哈希

#### Scenario: 用户名重复时新增失败
- **WHEN** 客户端调用 `POST /api/v1/users` 且用户名已存在
- **THEN** 服务返回 `409`

#### Scenario: 邮箱重复时新增失败
- **WHEN** 客户端调用 `POST /api/v1/users` 且邮箱已存在
- **THEN** 服务返回 `409`

### Requirement: 系统必须提供用户列表接口
系统必须 (MUST) 提供 `GET /api/v1/users` 接口用于查询用户列表。

#### Scenario: 查询用户列表成功
- **WHEN** 客户端调用 `GET /api/v1/users`
- **THEN** 服务返回 `200`
- **AND** 响应 `data.items` 必须为用户数组
- **AND** 每个用户项必须包含 `username`、`email`、`role`、`status`、`createdAt`
- **AND** 响应 `meta.pagination` 必须包含 `limit`、`total`、`partial`

#### Scenario: 查询用户列表支持分页排序和关键字过滤
- **WHEN** 客户端调用 `GET /api/v1/users?page=2&limit=10&sort=username&order=asc&keyword=alice`
- **THEN** 服务返回第二页结果集且结果数量不超过 `limit`
- **AND** 响应 `meta.links.self` 必须回显本次查询参数

### Requirement: 系统必须提供用户禁用接口
系统必须 (MUST) 提供 `PATCH /api/v1/users/:username/status` 接口更新用户状态。

#### Scenario: 禁用用户成功
- **WHEN** 客户端调用 `PATCH /api/v1/users/:username/status` 且请求 `status=disabled`
- **THEN** 服务返回 `200`
- **AND** Secret 中该用户的 `status` 被更新为 `disabled`

#### Scenario: 启用用户成功
- **WHEN** 客户端调用 `PATCH /api/v1/users/:username/status` 且请求 `status=active`
- **THEN** 服务返回 `200`
- **AND** Secret 中该用户的 `status` 被更新为 `active`

### Requirement: 系统必须提供删除用户接口
系统必须 (MUST) 提供 `DELETE /api/v1/users/:username` 接口用于删除用户。

#### Scenario: 删除普通用户成功
- **WHEN** 客户端调用 `DELETE /api/v1/users/:username` 且目标用户存在
- **THEN** 服务返回 `200`
- **AND** Secret 中对应用户记录被删除

#### Scenario: 删除内置管理员被拒绝
- **WHEN** 客户端调用 `DELETE /api/v1/users/admin`
- **THEN** 服务返回 `400`
- **AND** Secret 中 `admin` 账号必须保持存在

### Requirement: 系统必须提供修改密码接口
系统必须 (MUST) 提供 `PATCH /api/v1/users/:username/password` 接口用于更新用户密码。

#### Scenario: 修改密码成功
- **WHEN** 客户端调用 `PATCH /api/v1/users/:username/password` 且目标用户存在且密码满足长度规则
- **THEN** 服务返回 `200`
- **AND** Secret 中目标用户 `passwordHash` 被更新
- **AND** 响应不包含明文密码与密码哈希

#### Scenario: 用户不存在时修改密码失败
- **WHEN** 客户端调用 `PATCH /api/v1/users/:username/password` 且目标用户不存在
- **THEN** 服务返回 `404`

### Requirement: 用户角色字段固定为管理员
系统必须 (MUST) 在用户数据中保留 `role` 字段并固定写入 `admin`。

#### Scenario: 新增账号时角色自动固定
- **WHEN** 客户端调用 `POST /api/v1/users` 创建账号
- **THEN** 服务写入 Secret 的用户记录中 `role` 固定为 `admin`

### Requirement: 当前阶段不做用户管理角色权限校验
系统必须 (MUST) 在本阶段不对用户管理接口引入角色权限校验逻辑。

#### Scenario: 新增账号接口按数据规则处理请求
- **WHEN** 客户端调用 `POST /api/v1/users`
- **THEN** 服务只执行参数校验、用户名重复校验和 Secret 持久化规则

#### Scenario: 用户状态接口按数据规则处理请求
- **WHEN** 客户端调用 `PATCH /api/v1/users/:username/status`
- **THEN** 服务只执行参数校验、用户存在性校验和 Secret 持久化规则

#### Scenario: 删除用户接口按数据规则处理请求
- **WHEN** 客户端调用 `DELETE /api/v1/users/:username`
- **THEN** 服务只执行参数校验、用户存在性校验和 Secret 持久化规则

#### Scenario: 修改密码接口按数据规则处理请求
- **WHEN** 客户端调用 `PATCH /api/v1/users/:username/password`
- **THEN** 服务只执行参数校验、用户存在性校验和 Secret 持久化规则

### Requirement: 新增账号写入必须具备并发安全
系统必须 (MUST) 在新增账号写入 Secret 时使用冲突重试机制。

#### Scenario: 并发新增触发资源版本冲突
- **WHEN** 两个请求同时更新 `Secret/disaster-server-users` 且出现 `resourceVersion` 冲突
- **THEN** 服务必须在重试循环中重新读取最新 Secret 并重新计算写入内容
- **AND** 成功写入后返回 `201`
