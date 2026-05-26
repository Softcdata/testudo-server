## Context
当前服务端鉴权逻辑依赖硬编码管理员账号，账号无法持久化，也无法由 API 进行维护。
项目已经具备 Kubernetes 客户端能力，并且现有模块已经采用 `ConfigMap` 存储系统设置；账号数据可以采用同样的“文档化资源 + 乐观并发更新”模式，但载体改为 `Secret`。

## Goals
- 提供可落地的用户管理闭环：新增账号、列表查询、禁用账号、删除账号、修改密码、登录鉴权。
- 账号数据存储在 Kubernetes `Secret`，避免硬编码账号。
- 登录和用户管理流程具备并发安全与确定性行为。
- 当前阶段实现聚焦用户数据管理，不引入角色权限控制逻辑。

## Non-Goals
- 不引入外部数据库。
- 不改动现有业务 API 的资源读写语义。
- 不在本阶段设计用户角色模型和权限矩阵。

## Decisions

### Decision 1: 账号数据载体
账号数据统一存储在 `disaster-system` 命名空间的 `Secret` 中：
- 资源名：`disaster-server-users`
- 资源类型：`Opaque`
- 数据键：`users.json`

`users.json` 结构定义如下：

```json
{
  "schemaVersion": 1,
  "nextUserID": 2,
  "updatedAt": "2026-03-23T10:00:00Z",
  "updatedBy": "system",
  "users": {
    "admin": {
      "id": 1,
      "username": "admin",
      "email": "admin@example.com",
      "role": "admin",
      "status": "active",
      "passwordHash": "<bcrypt>",
      "createdAt": "2026-03-23T10:00:00Z",
      "updatedAt": "2026-03-23T10:00:00Z"
    }
  }
}
```

字段规则固定为：
- `username`：唯一。
- `email`：唯一。
- `role`：固定值 `admin`。
- `status`：仅允许 `active` 和 `disabled`。
- `createdAt`：创建后保持不变。

### Decision 2: 启动初始化触发时机
在 `ApiServer.Run` 启动流程中，执行顺序固定为：
1. 创建 Hertz Server 实例。
2. 调用 `EnsureUserSecret`。
3. 完成路由注册。
4. 开始监听端口。

`EnsureUserSecret` 行为固定为：
- 当 `Secret/disaster-server-users` 不存在时，创建包含 `admin` 账号的初始文档。
- 当 Secret 存在且缺少 `admin` 账号时，补齐内置管理员账号。
- 当 Secret 存在且已有 `admin` 账号时，仅做结构校验，不覆盖已有管理员密码与状态。

内置管理员账号固定为：
- `username=admin`
- `password=123456`（持久化时写入 bcrypt 哈希，不写入明文）
- `email=admin@example.com`
- `role=admin`
- `status=active`

### Decision 3: 新增账号写入流程
`POST /api/v1/users` 的写入顺序固定为：
1. 校验请求体（用户名、邮箱、密码长度、用户名字符集）。
2. 进入 `RetryOnConflict` 循环。
3. 在循环内执行 `Get Secret -> 反序列化 -> 重复用户名检查 -> 重复邮箱检查 -> 生成 bcrypt 哈希 -> 组装用户记录 -> 写回 Secret`。
4. 返回 `201 Created`。

冲突语义固定为：
- 用户名已存在返回 `409 Conflict`。
- 邮箱已存在返回 `409 Conflict`。
- Secret 更新冲突触发重试，重试结束仍失败则返回 `500`。

### Decision 4: 查询、状态更新、删除、改密流程
`GET /api/v1/users` 的执行顺序固定为：
1. 解析查询参数 `page`、`limit`、`sort`、`order`、`keyword`。
2. 读取并校验 `Secret/disaster-server-users`。
3. 转换为用户 DTO 列表，字段固定为 `username`、`email`、`role`、`status`、`createdAt`。
4. 按查询参数执行关键字过滤、稳定排序、内存分页。
5. 返回 `200 OK`，响应结构固定为 `data.items` 与 `meta.pagination`。

`PATCH /api/v1/users/:username/status` 的写入顺序固定为：
1. 校验路径参数 `username`。
2. 校验请求体 `status` 只允许 `active` 和 `disabled`。
3. 进入 `RetryOnConflict` 循环。
4. 在循环内执行 `Get Secret -> 反序列化 -> 查找用户 -> 更新 status 和 updatedAt -> 写回 Secret`。
5. 返回 `200 OK`。

`DELETE /api/v1/users/:username` 的写入顺序固定为：
1. 校验路径参数 `username`。
2. 当 `username=admin` 时直接返回 `400`。
3. 进入 `RetryOnConflict` 循环。
4. 在循环内执行 `Get Secret -> 反序列化 -> 查找用户 -> 删除用户记录 -> 写回 Secret`。
5. 返回 `200 OK`。

`PATCH /api/v1/users/:username/password` 的写入顺序固定为：
1. 校验路径参数 `username`。
2. 校验请求体密码长度规则。
3. 进入 `RetryOnConflict` 循环。
4. 在循环内执行 `Get Secret -> 反序列化 -> 查找用户 -> 生成 bcrypt 哈希 -> 更新 passwordHash 和 updatedAt -> 写回 Secret`。
5. 返回 `200 OK`。

### Decision 5: 用户管理接口权限范围
`GET /api/v1/users`、`POST /api/v1/users`、`PATCH /api/v1/users/:username/status`、`DELETE /api/v1/users/:username`、`PATCH /api/v1/users/:username/password` 在本阶段不做角色权限校验，服务端固定执行参数校验、存在性校验、重复性校验和 Secret 写入流程。

### Decision 6: 登录鉴权流程
`POST /login` 的鉴权顺序固定为：
1. 解析 `username` 和 `password`。
2. 读取 `Secret/disaster-server-users`。
3. 根据 `username` 获取账号记录。
4. 校验账号 `status` 是否为 `active`。
5. 使用 bcrypt 对输入密码与 `passwordHash` 做比对。
6. 比对成功后签发 AccessToken 与 RefreshToken。

### Decision 7: 密码安全策略
- Secret 中不得写入明文密码。
- 新增账号和改密时必须使用 bcrypt 哈希，默认 cost 为 `12`。
- API 响应中不得返回 `passwordHash` 字段。

## Risks
- 启动时 Secret 初始化失败会导致服务无法按预期提供登录能力。
- Secret 文档被手工改坏会导致账号解析失败。

## Mitigation
- 启动日志明确输出初始化失败原因，并在失败时中止启动，避免服务进入“可访问但无法登录”的不一致状态。
- 读取 Secret 时执行结构校验，发现非法结构即返回明确错误码并记录错误日志。

## Migration Plan
1. 发布新版本后，服务启动时自动创建 `Secret/disaster-server-users`。
2. 使用默认管理员账号登录。
3. 通过 `POST /api/v1/users` 创建业务账号。
4. 通过 `GET /api/v1/users` 核对账号列表。
5. 通过 `PATCH /api/v1/users/:username/password` 执行密码轮换。
6. 通过 `PATCH /api/v1/users/:username/status` 执行账号禁用与启用。
7. 通过 `DELETE /api/v1/users/:username` 删除不再使用的账号。
