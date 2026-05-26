# API 响应消息国际化设计

## Context
`disaster-server` 使用 Hertz 提供 REST 与 WebSocket 接口，统一响应信封位于 `internal/transport/response.go`。现有成功响应固定 `message: "OK"`，错误响应由 handler 传入字符串，鉴权与刷新令牌路径存在直接 `c.JSON` 返回 `msg` 的情况。当前 API 标准要求统一响应信封，但尚未定义语言协商、响应语言头、稳定消息键。

## Goals
- 客户端通过请求头控制 HTTP 接口返回语言。
- WebSocket 握手通过查询参数控制事件流消息语言。
- 服务端响应保留稳定业务错误码，并新增稳定消息键。
- 未迁移业务 handler 在过渡期保持原行为。
- 文档、测试、RunAPI/Apipost 示例与服务端行为一致。

## Non-Goals
- 不翻译 `data` 中的 Kubernetes 原始状态、CRD 枚举、业务状态机枚举。
- 不在本变更中重构所有 handler。
- 不通过英文原始消息反查翻译键。
- 不改变 HTTP 状态码和业务错误码映射。

## Decisions
- Decision: HTTP 语言入口使用 `X-Language`，服务端兼容 `Accept-Language`。
  Reason: 前端可控性最高，同时兼容浏览器标准语言头。
- Decision: 语言解析顺序为 `X-Language`，然后 `Accept-Language`，最后 `zh-CN`。
  Reason: 明确前端显式选择优先级，服务端默认值稳定。
- Decision: 支持语言首批为 `zh-CN` 与 `en-US`。
  Reason: 覆盖当前中文界面与英文错误消息迁移需求。
- Decision: 非法语言值降级到 `zh-CN`。
  Reason: 语言协商失败不应阻断核心业务接口。
- Decision: 响应信封新增 `message_key`，`message` 按请求语言本地化。
  Reason: 前端展示使用 `message`，自动化测试、日志关联、前端兜底使用 `message_key`。
- Decision: WebSocket 使用 `lang` 查询参数，并沿用 `transport.Success` 与 `transport.Error`。
  Reason: 浏览器原生 WebSocket 无法稳定设置自定义请求头，查询参数更适合事件流握手。
- Decision: 翻译资源使用 `go:embed` 内嵌。
  Reason: 容器运行时不依赖外部文件挂载，发布物可复现。

## Architecture
新增 `internal/i18n` 包：

```text
internal/i18n/
  locale.go
  catalog.go
  error.go
  locales/
    zh-CN.yaml
    en-US.yaml
```

`locale.go` 负责语言归一化、请求头解析、默认语言降级。`catalog.go` 负责加载内嵌翻译资源并按键查找消息。`error.go` 定义携带 `message_key`、参数和原始错误的业务错误类型。

新增 `internal/middleware/locale.go`：

- 读取 `X-Language`。
- 当 `X-Language` 为空时读取 `Accept-Language`。
- 写入 Hertz `RequestContext` 的 `locale`。
- 设置 `Content-Language`。
- 设置 `Vary: X-Language, Accept-Language`。

改造 `internal/transport/response.go`：

- `Envelope` 增加 `MessageKey string json:"message_key,omitempty"`。
- 保留 `WriteError(ctx, code, message, meta)` 兼容旧调用。
- 新增 `WriteErrorKey(ctx, code, key, args, meta)`。
- 新增 `WriteErrorFrom(ctx, code, err, meta)`，识别本地化错误类型。
- `Success` 默认继续返回 `message: "OK"`，待产品确认后再迁移为 `common.ok`。

## Migration Strategy
第一阶段建立基础设施：

- 增加 i18n 包、语言中间件、响应信封扩展。
- 注册全局语言中间件，使 `/login`、`/refresh_token`、`/apis`、`/api` 都具备语言上下文。
- 为 `transport`、语言解析、翻译降级补单元测试。

第二阶段统一公共出口：

- 改造 JWT 未授权响应。
- 改造刷新令牌响应。
- 改造恢复中间件响应。
- 改造 WebSocket 升级失败与事件流错误响应。

第三阶段迁移高频业务错误：

- 先迁移公共校验错误，例如名称为空、请求体非法、查询参数非法、资源不存在、动作不支持、权限不足。
- 再按模块迁移业务错误，顺序为 `disaster_cluster`、`disaster_instance`、`app_backup`、`app_restore`、`system_settings`、`user`、`event`、`statistics`。

第四阶段处理展示字段：

- 当产品要求数据字段展示本地化文案时，新增 `statusDisplay` 这类展示字段。
- 原始状态字段继续保持原值。

## Interface Scope
所有接口都需要在文档中声明语言协议。代码层不需要逐个接口重写：

- 已经通过 `transport.WriteSuccess` 返回纯数据的接口不需要业务代码修改。
- 通过 `transport.WriteError` 返回用户可见字符串的接口需要逐步迁移到消息键。
- 直接 `c.JSON` 返回 `msg`、`message`、`error` 的接口需要迁移到统一响应信封。
- WebSocket 事件流需要在握手阶段解析语言，并在错误消息中使用同一语言上下文。

## Risks / Trade-offs
- Risk: 一次性迁移全部 handler 会造成大范围冲突。
  Mitigation: 先保留旧 `WriteError`，新增消息键接口，新代码与高频错误先迁移。
- Risk: 测试中断言英文错误字符串会失效。
  Mitigation: 新测试断言 `code` 与 `message_key`，只在语言测试中断言 `message`。
- Risk: 第三方客户端依赖旧 `msg` 字段。
  Mitigation: 鉴权与刷新令牌迁移需在 OpenAPI、RunAPI/Apipost、发布说明中明确统一为 `message`。
- Risk: 缺失翻译键导致用户看到键名。
  Mitigation: 单元测试覆盖目录完整性，缺失键写入日志，默认语言缺失时返回键名。

## Open Questions
- 成功响应 `message: "OK"` 是否需要本地化为中文 `成功`。
- 前端是否需要服务端返回 `supported_languages` 配置接口。
- 首批业务模块迁移范围是否限定在登录、刷新令牌、集群、实例、备份、恢复。
