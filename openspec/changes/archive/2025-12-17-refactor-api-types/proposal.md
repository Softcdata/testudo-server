# 重构 API 类型定义 (Refactor API Types)

## 1. 摘要 (Summary)
本提案旨在将 `disaster-server` 的 API 请求/响应类型与 `disaster-operator` 的 CRD 类型解耦。我们将为 Server 端定义独立的 DTO (Data Transfer Object) 结构体，用于参数绑定、验证和文档生成，然后再将其转换为 Operator 的 CRD 类型进行后续处理。

## 2. 动机 (Motivation)
目前，Server 端直接使用 Operator 的 CRD 类型（如 `AppBackupSpec`, `AppRestoreSpec`）作为 HTTP API 的入参。
- **验证受限**：CRD 的验证逻辑通常依赖 Kubernetes Webhook，Server 端难以进行自定义的业务逻辑验证。
- **耦合过紧**：Operator 的 CRD 变更会直接影响 Server 的 API 契约，缺乏缓冲层。
- **字段冗余**：CRD 中包含许多 Status 或 Controller 内部使用的字段，不应暴露给 API 用户。
- **文档生成**：直接使用 CRD 类型生成的 Swagger/OpenAPI 文档不够友好。

## 3. 目标 (Goals)
- 为所有核心资源创建独立的 API 请求结构体 (DTO)，用于**创建**和**更新**操作，包括但不限于：
    - `AppBackup`
    - `AppRestore`
    - `DisasterBackup`
    - `DisasterCluster`
    - `DisasterConfig`
    - `DisasterPolicy`
    - `DisasterStorage`
    - `DisasterJobs` (如果适用)
- 实现 DTO 到 CRD 类型的转换函数 (`ToCRD`)。
- 在 DTO 层引入更严格的参数验证 (使用 `go-playground/validator` 或类似库)。
- 保持现有 API 行为不变（向后兼容）。

## 4. 非目标 (Non-Goals)
- 修改 Operator 的 CRD 定义。
- 改变现有的数据库存储结构。
