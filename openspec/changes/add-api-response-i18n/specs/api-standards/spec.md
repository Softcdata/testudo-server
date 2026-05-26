## ADDED Requirements

### Requirement: API 响应消息国际化
系统必须 (MUST) 根据客户端传入的语言偏好返回本地化的用户可见响应消息。

#### Scenario: HTTP 请求通过 X-Language 指定语言
- **GIVEN** 客户端发送 HTTP 请求并携带 `X-Language: en-US`
- **WHEN** 服务端返回带有用户可见消息的响应
- **THEN** 响应体中的 `message` 必须使用英文
- **AND** 响应头 `Content-Language` 必须为 `en-US`
- **AND** 响应头 `Vary` 必须包含 `X-Language` 和 `Accept-Language`

#### Scenario: HTTP 请求通过 Accept-Language 指定语言
- **GIVEN** 客户端发送 HTTP 请求，未携带 `X-Language`
- **AND** 请求头包含 `Accept-Language: en-US,en;q=0.9`
- **WHEN** 服务端返回带有用户可见消息的响应
- **THEN** 响应体中的 `message` 必须使用英文
- **AND** 响应头 `Content-Language` 必须为 `en-US`

#### Scenario: 缺少语言请求头
- **GIVEN** 客户端发送 HTTP 请求，未携带 `X-Language`
- **AND** 请求头未携带 `Accept-Language`
- **WHEN** 服务端返回带有用户可见消息的响应
- **THEN** 服务端必须使用默认语言 `zh-CN`
- **AND** 响应头 `Content-Language` 必须为 `zh-CN`

#### Scenario: 非法语言值降级
- **GIVEN** 客户端发送 HTTP 请求并携带 `X-Language: invalid`
- **WHEN** 服务端返回带有用户可见消息的响应
- **THEN** 服务端必须使用默认语言 `zh-CN`
- **AND** 响应头 `Content-Language` 必须为 `zh-CN`
- **AND** 服务端不得因为非法语言值拒绝业务请求

#### Scenario: WebSocket 握手通过 lang 指定语言
- **GIVEN** 客户端建立 WebSocket 连接并携带查询参数 `lang=en-US`
- **WHEN** 服务端通过 WebSocket 返回带有用户可见消息的 `Envelope`
- **THEN** `message` 必须使用英文
- **AND** `message_key` 必须保持稳定

#### Scenario: 原始数据字段保持稳定
- **GIVEN** 接口响应 `data` 中包含 Kubernetes 状态、CRD 枚举、业务状态机枚举
- **WHEN** 客户端通过 `X-Language` 请求不同语言
- **THEN** 原始状态字段必须保持原值
- **AND** 服务端不得用本地化文案覆盖原始状态字段

## MODIFIED Requirements

### Requirement: 统一响应信封 (Unified Response Envelope)
所有 API 响应（无论是成功还是错误）都必须 (MUST) 使用标准的 `Envelope` 结构。带有用户可见消息的错误响应必须 (MUST) 返回本地化 `message`，并应返回稳定的 `message_key`。

#### Scenario: 成功响应
给定一个对 `GET /apis/v1/backups/1` 的请求
当请求被成功处理时
那么响应体应匹配：
```json
{
  "code": 0,
  "message": "OK",
  "data": { },
  "meta": { },
  "trace_id": "..."
}
```

#### Scenario: 错误响应
给定一个包含无效参数的请求
当验证失败时
那么响应体应匹配：
```json
{
  "code": 1000,
  "message": "名称不能为空",
  "message_key": "validation.name_required",
  "data": null,
  "meta": { "details": [] },
  "trace_id": "..."
}
```

#### Scenario: 英文错误响应
给定一个包含无效参数的请求
并且请求头包含 `X-Language: en-US`
当验证失败时
那么响应体应匹配：
```json
{
  "code": 1000,
  "message": "name is required",
  "message_key": "validation.name_required",
  "data": null,
  "meta": { "details": [] },
  "trace_id": "..."
}
```

#### Scenario: 未迁移错误响应
给定一个 handler 仍通过旧 `WriteError` 返回错误
当请求被处理时
那么服务端必须保持现有 `message` 字符串
并且不得改变 HTTP 状态码、业务错误码、`trace_id`
