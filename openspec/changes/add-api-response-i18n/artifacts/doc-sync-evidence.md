# 文档同步证据

## 2026-05-16 执行记录

- Swagger/OpenAPI：已更新 `openspec/specs/disaster-server-openapi.yaml`。
- 共享请求头：新增 `components.parameters.LanguageHeader`，声明 `X-Language` 支持 `zh-CN` 与 `en-US`，非法值降级为 `zh-CN`。
- WebSocket 语言参数：新增 `components.parameters.WebSocketLangQuery`，声明握手查询参数 `lang`。
- 响应头：新增 `components.headers.ContentLanguage` 与 `components.headers.VaryLanguage`，声明 `Content-Language` 和 `Vary: X-Language, Accept-Language`。
- 统一响应信封：`Envelope` 与 `ErrorEnvelope` 新增 `message_key` 描述，`message` 描述改为按语言协议返回本地化消息。
- 鉴权错误：`MiddlewareAuthError` 改为复用 `ErrorEnvelope`，不再描述旧 `code/msg` 响应。
- Panic 错误：`PanicErrorResponse` 改为复用 `ErrorEnvelope`，`message_key` 固定为 `common.internal_error`。
- 示例：新增中文与英文 `validation.name_required` 错误响应示例。

## RunAPI/Apipost 状态

- 已在 Apipost 项目 `5650333c5c52000` 新增 Markdown 文档 `API 响应消息国际化协议`，Target ID 为 `1c8686a538401000`。
- 已回读并更新 `POST /login`，Target ID 为 `25559f1e78c064`；已新增 `X-Language` 请求头，更新当前统一信封说明，并补充当前登录成功信封、当前用户名或密码错误 `en-US` 响应示例。
- 已回读并更新 `POST /refresh_token`，Target ID 为 `1bf8e6af53c01001`；已新增 `X-Language` 请求头，更新当前统一信封说明，并补充当前缺少 `refreshToken`、当前无效 `refreshToken` 的 `en-US` 响应示例。
- 本地证据文件同步记录共享协议变化，后续逐接口详情补写时继续保留原有说明。

## 受影响接口范围

- HTTP：所有经过全局中间件的 REST 接口都会解析 `X-Language` 与 `Accept-Language`，响应头携带 `Content-Language` 与 `Vary`。
- WebSocket：通过 `internal/utils/watch.go` 的 watch 接口在握手阶段支持 `lang` 查询参数，事件流错误信封携带本地化 `message` 与 `message_key`。
- 首批代码迁移模块：JWT、登录、刷新令牌、Recovery、Tenant、WebSocket、公用响应、`disaster_cluster`、`disaster_instance`、`disaster_backup`、`disaster_config`、`disaster_jobs`、`disaster_policy`、`deletion_check`、`event`、`platform_license`、`app_backup`、`app_restore`、`system_settings`、`user`、`kubernetes_resources` 错误出口。
