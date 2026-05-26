## 1. Server

- [x] 1.1 在 `StatusDTO` 新增当前指纹字段与 `FingerprintRequestDTO`。
- [x] 1.2 在 `Service.Status` 返回前统一计算当前指纹，并填充响应字段。
- [x] 1.3 保持 `CheckClusterCreate` 的门禁校验链路不变，确保创建集群仍基于 Secret 与当前指纹重新评价。
- [x] 1.4 确认 `Service.Install` 返回的 `StatusDTO` 同步包含当前指纹字段。

## 2. Tests

- [x] 2.1 增加状态 ConfigMap 存在时仍返回当前指纹的单元测试。
- [x] 2.2 增加无 License 时返回当前指纹的单元测试。
- [x] 2.3 增加指纹生成失败时返回 `Unknown` 与 `LicenseEnvironmentInvalid` 的单元测试。
- [x] 2.4 增加安装 License 后响应携带当前指纹的单元测试。

## 3. Docs

- [x] 3.1 更新 `openspec/specs/disaster-server-openapi.yaml` 的 `PlatformLicenseStatus` schema。
- [x] 3.2 更新 `GET /apis/v1/platform-license/status` 的接口说明与响应示例。
- [x] 3.3 更新 RunAPI 本地证据文件；如果 Apipost MCP 可用，先读取接口详情，再更新线上 RunAPI 条目。

## 4. Deploy

- [x] 4.1 检查 Chart 中 `disaster-server` 对 `disaster-platform-install-id` Secret 的读取权限。
- [x] 4.2 检查 Chart 中 `disaster-server` 在 License Namespace 创建 Secret 的权限。

## 5. Validation

- [x] 5.1 运行 `go test ./internal/apis/platform_license/v1`。
- [x] 5.2 运行 OpenAPI 校验命令。
- [x] 5.3 运行 `openspec validate add-license-status-fingerprint --strict`。
- [x] 5.4 运行 `git diff --check`。
