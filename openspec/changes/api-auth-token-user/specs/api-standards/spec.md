# Specification Change: API Authentication Standards

## Purpose
该变更定义了 API 认证和用户信息传递的标准。我们将废弃基于 header 的 `X-Tenant-ID` 传递方式，转而完全依赖标准的 JWT `Authorization` header 来提取用户身份和租户上下文（如果适用）。

## Requirements

### Authentication Source
#### Scenario: Client Request with Token
*   **GIVEN** a client sends a request with a valid `Authorization: Bearer <token>` header
*   **WHEN** the server processes the request
*   **THEN** it SHALL extract the `username` and `userID` from the token claims.
*   **AND** it SHALL make this user information available to the API context.

### Legacy Header Deprecation
#### Scenario: Request without X-Tenant-ID
*   **GIVEN** a client sends a request with a valid Token but NO `X-Tenant-ID` header
*   **WHEN** the server processes authentication
*   **THEN** it SHALL NOT reject the request based on missing tenant header.
*   **AND** it SHALL default to the user's implicit tenant or system default context.

### Event Auditing
#### Scenario: Audit Log Generation
*   **GIVEN** an operation triggers a system event (e.g., Create Policy)
*   **WHEN** the event is recorded
*   **THEN** the `User` field of the event SHALL be populated with the `username` extracted from the JWT Token.
