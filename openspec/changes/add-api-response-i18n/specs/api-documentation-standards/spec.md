## ADDED Requirements

### Requirement: 国际化协议文档
所有 API 文档必须 (MUST) 声明服务端响应消息国际化协议，确保前端和调试工具能够按语言验证响应。

#### Scenario: OpenAPI 声明语言请求头
- **WHEN** OpenAPI 描述 HTTP 接口
- **THEN** 必须声明共享请求头参数 `X-Language`
- **AND** `X-Language` 可传入值必须包含 `zh-CN` 和 `en-US`
- **AND** 描述必须说明默认语言为 `zh-CN`
- **AND** 描述必须说明服务端兼容 `Accept-Language`

#### Scenario: OpenAPI 声明响应语言头
- **WHEN** OpenAPI 描述 HTTP 接口响应
- **THEN** 必须声明响应头 `Content-Language`
- **AND** 必须声明响应头 `Vary`
- **AND** `Vary` 描述必须包含 `X-Language` 和 `Accept-Language`

#### Scenario: OpenAPI 声明统一响应信封
- **WHEN** OpenAPI 描述统一响应信封 schema
- **THEN** 必须包含 `message_key`
- **AND** `message_key` 描述必须说明该字段为稳定消息键
- **AND** 错误响应示例必须包含中文示例与英文示例

#### Scenario: WebSocket 文档声明语言参数
- **WHEN** 文档描述 WebSocket 接口
- **THEN** 必须声明查询参数 `lang`
- **AND** `lang` 可传入值必须包含 `zh-CN` 和 `en-US`
- **AND** 描述必须说明 WebSocket 消息沿用统一响应信封

#### Scenario: RunAPI/Apipost 同步
- **WHEN** 任意接口文档因国际化协议发生更新
- **THEN** RunAPI/Apipost 必须保留原有说明
- **AND** 新增国际化协议说明必须位于原有说明前
- **AND** 原有说明必须放入 `## 原有说明` 小节
