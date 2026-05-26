## ADDED Requirements

### Requirement: WebSocket 鉴权支持
由于浏览器 WebSocket API 限制，无法自定义 HTTP Header，系统必须 (MUST) 支持通过其他方式传递认证 Token。

#### Scenario: 通过 Sec-WebSocket-Protocol 传递 Token
- **WHEN** 客户端发起 WebSocket 连接
- **AND** 在 `Sec-WebSocket-Protocol` Header 中携带 JWT Token
- **THEN** 服务端必须能够提取并验证该 Token
- **AND** 握手成功后建立连接

#### Scenario: 通过 URL Query 传递 Token
- **WHEN** 客户端发起 WebSocket 连接
- **AND** 在 URL Query 参数 `token` 中携带 JWT Token
- **THEN** 服务端必须能够提取并验证该 Token
