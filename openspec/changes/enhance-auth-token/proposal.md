# Proposal: 增强认证接口 (Enhance Auth Token)

## 1. 背景
用户需要更安全的双 Token 认证机制。目前的单一 Token 机制不足以满足长期会话维持和安全性的平衡需求。需要引入 Access Token (短效) 和 Refresh Token (长效) 分离的机制。

## 2. 目标
1.  **双 Token 机制**: 登录成功返回 `accessToken` (短效) 和 `refreshToken` (长效)。
2.  **时长可配置**: 两个 Token 的过期时间均可通过配置文件调整。
    *   `accessToken`: 默认 24 小时。
    *   `refreshToken`: 默认较长时间 (如 7 天)。
3.  **刷新接口调整**: `/refresh_token` 接口通过 Body 传入 `refreshToken` 来获取新的 `accessToken`。

## 3. 设计方案

### 3.1 配置变更
在配置文件 (如 `config.yaml`) 中增加或修改 JWT 相关配置：
```yaml
jwt:
  secret: "..."
  access_expire: 24h  # AccessToken 过期时间
  refresh_expire: 168h # RefreshToken 过期时间 (7天)
```

### 3.2 接口变更
*   **Login API (`POST /login`)**:
    *   Response JSON:
        ```json
        {
          "code": 0,
          "data": {
             "accessToken": "eyJ...",
             "refreshToken": "eyJ...",
             "expire": "2026-01-27T11:00:00+08:00" // AccessToken 的过期时间
          }
        }
        ```
*   **Refresh API (`POST /refresh_token`)**:
    *   Request JSON: `{ "refreshToken": "..." }`
    *   Response JSON:
        ```json
        {
          "code": 0,
          "data": {
             "accessToken": "eyJ...",
             "expire": "2026-01-27T12:00:00+08:00"
          }
        }
        ```

### 3.3 实现细节
*   **Token 生成**:
    *   `accessToken`: 包含用户身份信息，有效期短。
    *   `refreshToken`: 包含用于刷新的必要信息 (如用户ID)，有效期长，可以使用不同的 Secret 或 Claim 标识。
*   **Middleware 调整**:
    *   `LoginResponse` 中需要额外生成 `refreshToken` 并返回。
    *   自定义 `/refresh_token` 处理逻辑，不使用 `hertz-contrib/jwt` 默认的 `RefreshHandler` (因为它通常基于 Authorization Header 中的 expired token)，而是解析 Body 中的 `refreshToken` 并验证，验证通过后签发新的 `accessToken`。

## 4. 影响范围
*   `configs/config.go` & `config.yaml`: 新增配置项。
*   `internal/middleware/jwt.go`:
    *   修改 `NewJWT` 初始化逻辑。
    *   修改 `LoginResponse`。
    *   实现新的 Refresh Logic。
*   `internal/router/router.go`: 绑定新的 `/refresh_token` handler。

## 5. 验证计划
1.  **配置检查**: 确认时间配置生效。
2.  **登录测试**: 调用 `/login`，检查是否返回两个 Token。
3.  **Token 解析**: 验证 `accessToken` 和 `refreshToken` 的过期时间是否符合配置。
4.  **刷新测试**:
    *   使用 `refreshToken` 调用接口，验证能否获取新 `accessToken`。
    *   使用无效/过期的 `refreshToken`，验证是否拒绝。
