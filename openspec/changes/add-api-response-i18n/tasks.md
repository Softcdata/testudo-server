## 1. 规范与设计
- [x] 1.1 确认支持语言首批为 `zh-CN` 与 `en-US`
- [x] 1.2 确认成功响应 `message` 保持 `OK`
- [x] 1.3 确认 WebSocket 使用 `lang` 查询参数
- [x] 1.4 评审并批准本 OpenSpec 变更

## 2. 服务端基础设施
- [x] 2.1 新增 `internal/i18n` 包，支持语言归一化、默认语言降级、翻译资源加载
- [x] 2.2 新增 `LocaleMiddleware`，解析 `X-Language` 与 `Accept-Language`
- [x] 2.3 在全局中间件链注册 `LocaleMiddleware`
- [x] 2.4 响应设置 `Content-Language` 与 `Vary`
- [x] 2.5 扩展 `transport.Envelope`，新增 `message_key`
- [x] 2.6 新增 `WriteErrorKey` 与 `WriteErrorFrom`
- [x] 2.7 保留旧 `WriteError` 兼容未迁移 handler

## 3. 公共响应出口迁移
- [x] 3.1 迁移 JWT 未授权响应到统一响应信封
- [x] 3.2 迁移登录响应到统一响应信封
- [x] 3.3 迁移刷新令牌响应到统一响应信封
- [x] 3.4 迁移恢复中间件响应到统一响应信封
- [x] 3.5 迁移 WebSocket 升级失败与事件流错误响应

## 4. 高频业务错误迁移
- [x] 4.1 维护 `artifacts/module-migration-inventory.md` 中的模块迁移清单
- [x] 4.2 迁移公共校验错误消息键
- [x] 4.3 迁移 `disaster_cluster` 高频错误
- [x] 4.4 迁移 `disaster_instance` 高频错误
- [x] 4.5 迁移 `app_backup` 与 `app_restore` 高频错误
- [x] 4.6 迁移 `system_settings` 与 `user` 高频错误

## 5. 文档同步
- [x] 5.1 更新 Swagger/OpenAPI 共享请求头参数
- [x] 5.2 更新 Swagger/OpenAPI 统一响应信封 schema
- [x] 5.3 更新 Swagger/OpenAPI 示例，覆盖中文与英文错误响应
- [x] 5.4 更新 RunAPI/Apipost 共享说明与受影响接口示例
- [x] 5.5 更新本地接口证据文件，记录语言头、响应头、响应字段变化

## 6. 测试与验证
- [x] 6.1 增加语言解析单元测试
- [x] 6.2 增加翻译目录缺失键与降级测试
- [x] 6.3 增加 `transport` 本地化响应测试
- [x] 6.4 增加 JWT 与刷新令牌响应测试
- [x] 6.5 增加 WebSocket 语言参数测试
- [x] 6.6 运行相关 Go 测试
- [x] 6.7 运行 OpenSpec 严格校验
- [x] 6.8 运行 OpenAPI 校验
- [x] 6.9 运行 `git diff --check`
