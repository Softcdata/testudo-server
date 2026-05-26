# Auth Specification

## MODIFIED Requirements

#### Requirement: 用户登录接口返回 AccessToken
登录接口必须返回明确的 `accessToken` 字段，而不是通用的 `token`。

#### Scenario: 成功登录
- **Given**: 用户提供正确的用户名和密码
- **When**: 发送 POST 请求到 `/login`
- **Then**: 响应体应包含 `accessToken` 字段
- **Then**: 响应体应包含过期时间 `expire`
- **And**: 响应状态码为 200

## ADDED Requirements

#### Requirement: Token 刷新接口
系统必须提供刷新 Token 的接口，允许在 Token 即将过期时获取新的 AccessToken。

#### Scenario: 刷新 Token
- **Given**: 用户拥有有效的（或在刷新窗口期的）JWT Token
- **When**: 发送请求到 `/refresh_token` (携带 Authorization Bearer Token)
- **Then**: 响应体应包含新的 `accessToken`
- **Then**: 响应体应包含新的过期时间 `expire`
- **And**: 旧 Token (视策略而定) 可能失效或仅作为刷新凭证
