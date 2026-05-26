# Tasks for Refactor API Style

- [x] **Middleware & Transport** <!-- id: middleware-transport -->
    - [x] Implement `TenantID` middleware to handle `X-Tenant-ID`. <!-- id: impl-tenant-middleware -->
    - [x] Update `RequestID` middleware to use `X-Request-ID`. <!-- id: update-request-id -->
    - [x] Refactor `transport/response.go` to strictly match `Envelope` spec. <!-- id: refactor-response -->
    - [x] Define standard error codes in `transport` package matching the "Double Layer" spec. <!-- id: define-error-codes -->

# API 风格重构任务列表

- [x] **中间件与传输层** <!-- id: middleware-transport -->
    - [x] 实现 `TenantID` 中间件以处理 `X-Tenant-ID`。 <!-- id: impl-tenant-middleware -->
    - [x] 更新 `RequestID` 中间件以使用 `X-Request-ID`。 <!-- id: update-request-id -->
    - [x] 重构 `transport/response.go` 以严格符合 `Envelope` 规范。 <!-- id: refactor-response -->
    - [x] 在 `transport` 包中定义符合“双层错误码”规范的标准错误码。 <!-- id: define-error-codes -->

- [x] **数据建模 (DTOs)** <!-- id: data-modeling -->
    - [x] 为核心资源（Backup, Restore, Cluster 等）定义 DTO 结构体，仅暴露必要字段。 <!-- id: define-dtos -->
    - [x] 实现将内部 K8s CRD 转换为 DTO 的映射函数。 <!-- id: impl-mappers -->

- [x] **分页与过滤** <!-- id: pagination-filtering -->
    - [x] 创建用于从请求中解析 `page`, `limit`, `sort`, `filters` 的辅助函数。 <!-- id: pagination-helpers -->
    - [x] 创建用于生成标准 `meta` 响应的辅助函数。 <!-- id: meta-response-helpers -->

- [x] **API 重构 (迭代进行)** <!-- id: refactor-apis -->
    - [x] 重构 `app_backup` API 以使用新的响应/错误/异步模式。 <!-- id: refactor-app-backup -->
    - [x] 重构 `app_restore` API。 <!-- id: refactor-app-restore -->
    - [x] 重构 `disaster_cluster` API。 <!-- id: refactor-cluster -->
    - [x] 重构其他 API (`config`, `policy`, `storage`, 等)。 <!-- id: refactor-others -->

- [x] **验证** <!-- id: verification -->
    - [x] 验证所有端点返回正确的 `Envelope`。 <!-- id: verify-envelope -->
    - [x] 验证错误情况返回正确的 HTTP 状态码和业务错误码。 <!-- id: verify-errors -->
    - [x] 验证分页元数据正确。 <!-- id: verify-pagination -->


- [x] **Pagination & Filtering** <!-- id: pagination-filtering -->
    - [x] Create helper functions for parsing `page`, `limit`, `sort`, `filters` from request. <!-- id: pagination-helpers -->
    - [x] Create helper functions for generating standard `meta` response. <!-- id: meta-response-helpers -->

- [x] **Refactor APIs (Iterative)** <!-- id: refactor-apis -->
    - [x] Refactor `app_backup` APIs to use new response/error/async pattern. <!-- id: refactor-app-backup -->
    - [x] Refactor `app_restore` APIs. <!-- id: refactor-app-restore -->
    - [x] Refactor `disaster_cluster` APIs. <!-- id: refactor-cluster -->
    - [x] Refactor other APIs (`config`, `policy`, `storage`, etc.). <!-- id: refactor-others -->

- [x] **Verification** <!-- id: verification -->
    - [x] Verify all endpoints return correct `Envelope`. <!-- id: verify-envelope -->
    - [x] Verify error cases return correct HTTP status and Business code. <!-- id: verify-errors -->
    - [x] Verify pagination metadata is correct. <!-- id: verify-pagination -->
