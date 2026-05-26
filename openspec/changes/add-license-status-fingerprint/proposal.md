# Change: License 状态接口返回当前集群指纹

## Why

当前发放 License 需要管理员单独执行 `disasterctl license fingerprint` 生成指纹请求文件，页面拿不到当前集群指纹，联调与客户现场交付步骤偏重。

## What Changes

- 扩展 `GET /apis/v1/platform-license/status` 的 `data` 响应，返回当前管理面集群指纹、指纹算法版本，以及可直接用于签发 License 的指纹请求对象。
- 指纹请求对象与 `disasterctl license fingerprint` 输出字段保持一致，包含 `product`、`fingerprintVersion`、`fingerprint`、`namespace`、`generatedAt`。
- 状态接口在每次请求处理期间生成当前指纹；无 License、License 过期、License 指纹不匹配、License 内容错误时，仍返回可用于重新签发的当前指纹。
- 当当前指纹生成失败且不存在更高优先级的 License 内容错误时，状态接口以 `data.state=Unknown`、`data.reason=LicenseEnvironmentInvalid` 表达环境错误，HTTP 状态仍保持 200；当 License 内容错误已经完成分类时，保留该状态和 reason。
- 补充 Swagger/OpenAPI、RunAPI 证据、单元测试，以及 Chart RBAC 检查项。

## Impact

- 受影响的规范：`platform-license`
- 受影响的接口：`GET /apis/v1/platform-license/status`、`POST /apis/v1/platform-license/install`
- 受影响的代码：`internal/apis/platform_license/v1/types.go`、`internal/apis/platform_license/v1/service.go`、`internal/apis/platform_license/v1/service_test.go`
- 受影响的文档：`openspec/specs/disaster-server-openapi.yaml`、RunAPI 本地证据文件
- 受影响的部署：Chart 中 `disaster-server` 对 `disaster-platform-install-id` Secret 的读取与创建权限需要核对
