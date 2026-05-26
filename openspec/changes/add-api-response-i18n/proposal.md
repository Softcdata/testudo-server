# Change: API 响应消息国际化

## Why
当前服务端响应中的用户可见消息以英文硬编码为主，前端无法通过统一协议获取中文、英文等语言版本。鉴权、刷新令牌、WebSocket 错误、业务校验错误还存在直接返回 `msg` 的路径，和统一响应信封不一致。

## What Changes
- 增加服务端语言解析机制，客户端通过 `X-Language` 请求头传入语言，服务端兼容 `Accept-Language`。
- 扩展统一响应信封，错误响应返回本地化 `message`，并返回稳定的 `message_key`。
- 增加响应头 `Content-Language` 和 `Vary`，让客户端、代理、缓存明确语言差异。
- WebSocket 握手支持 `lang` 查询参数，事件流消息沿用同一套语言上下文。
- 分阶段迁移用户可见错误消息，优先覆盖 `transport`、鉴权、刷新令牌、恢复中间件、WebSocket 和高频业务校验错误。
- 更新 Swagger/OpenAPI 与 RunAPI/Apipost 文档，声明共享语言请求头、响应语言头、`message_key` 字段和示例。

## Impact
- 受影响的规范：`api-standards`、`api-documentation-standards`
- 受影响的代码：
  - `internal/i18n/`
  - `internal/middleware/`
  - `internal/transport/`
  - `internal/utils/watch.go`
  - `internal/middleware/jwt.go`
  - 各业务 handler 中返回用户可见错误消息的位置
- 受影响的文档：
  - `openspec/specs/disaster-server-openapi.yaml`
  - RunAPI/Apipost 线上接口文档
  - 本地接口文档证据文件

## Scope
本变更不要求重写所有接口 handler。所有 HTTP 与 WebSocket 接口需要在文档层声明语言协议；代码层只改会产生用户可见消息的公共出口与错误路径。纯数据查询、DTO 字段映射、Kubernetes 原始状态枚举在本变更中保持稳定。
