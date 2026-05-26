# Change: Support WS Token in Protocol

## Why
目前 Hertz JWT 中间件主要从 Authorization Header 提取 Token。但在 WebSocket 场景下，浏览器原生的 WebSocket API 不支持设置自定义 Header，常用的变通方案是将 Token 放入 `Sec-WebSocket-Protocol` 子协议字段中，或者放入 URL Query 参数。
为了统一支持 WebSocket 鉴权，我们需要增强 JWT 中间件的 Token 提取逻辑，使其能够从 `Sec-WebSocket-Protocol` 中识别并提取 Token。

## What Changes
- **JWT Middleware**: 更新 `TokenLookup` 配置。
  - 当前默认可能仅支持 `header:Authorization`。
  - 需要扩展为支持 `header:Authorization, header:Sec-WebSocket-Protocol, query:token, param:token, cookie:jwt` 等多种方式，或者自定义提取函数。
  - 特别针对 `Sec-WebSocket-Protocol`，它可能是一个列表（如 `token, json`），需要解析。但通常最简单的做法是将 Token 作为唯一值或第一个值传入。
  - 参考 `hertz-contrib/jwt` 文档，查看是否支持 `TokenLookup` 的组合配置。如果支持，直接配置；如果不支持，使用 `TokenExtractor` 自定义函数。

## Impact
- **受影响代码**: `internal/middleware/jwt.go`
- **使用方**: 如果 Token 存在于 Protocol 头中，Hertz 会像处理 Header 一样验证它。如果是 WS 连接，升级前的 HTTP 握手阶段就会通过此验证。

## Implementation Details
- `hertz-contrib/jwt` 的 `New` 选项中有一个 `TokenLookup` 字段。
- 格式字符串：`"header:Authorization, query:token, cookie:jwt"`
- 我们可以添加 `"header:Sec-WebSocket-Protocol"`。
- **注意**：`Sec-WebSocket-Protocol` 的值可能包含多个子协议，用逗号分隔。如果 Token 只是其中之一，标准的 `header:` 提取器可能无法正确解析（它可能把整个字符串当做 Token）。
- **更稳健的做法**: 实现自定义的 `TokenLookup` 或者如果库支持 `func`。
- 查看 `hertz-contrib/jwt` 源码或文档：它支持 `TokenLookup` (string)。
- 如果库本身不支持解析 Protocol 列表，我们可能需要 hack 一下，或者假定 Protocol 只传 Token。
- **降级方案**: 将 Token 放入 Query 参数 (`ws://host/path?token=xxx`)。这是最通用且兼容的 WS 鉴权方式。
- **本次 Proposals**: 优先尝试 Query 参数（标准支持），同时尝试支持 Protocol（如果用户强烈要求）。但用户明确提到“只能把token传到Sec-WebSocket-Protocol”，说明他们可能已经受限于某些 Clients 或不想暴露 Token 在 URL 日志中。
- 因此，我们将实现从 `Sec-WebSocket-Protocol` 提取 Token。由于 `hertz-contrib/jwt` 可能直接取 Header 值，我们需要确认能否处理。如果不能，我们需要自定义提取器逻辑。如果 `hertz-contrib/jwt` 不支持自定义提取器函数（只支持 string lookup），我们需要在 JWT 中间件**之前**加一个适配器中间件，将 Protocol 中的 Token 挪到 Authorization Header，或者 Query 参数中。

- **Plan B (Middleware Adapter)**:
  1. 新增中间件 `ProtocolToAuthAdapter`。
  2. 检查 `Sec-WebSocket-Protocol` 头。
  3. 如果存在且看起来像 Token，将其设置为 `Authorization: Bearer <token>`。
  4. 清理 Protocol 头（可选，以免干扰后续 WS 握手，但 WS 握手通常需要回传选中的 Protocol，这里可能要注意）。
  
- **Plan A (Native config)**:
  配置 `TokenLookup: "header:Authorization,header:Sec-WebSocket-Protocol,query:token"`。
