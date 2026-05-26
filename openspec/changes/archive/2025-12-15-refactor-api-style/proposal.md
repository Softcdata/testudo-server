# Refactor API Style to Match Guidelines

## Status
Implemented

## Summary
本提案旨在重构 `disaster-server` 的 API 实现，使其完全符合 `openspec/guides/api-style.md` 中定义的风格指南。主要变更包括统一请求/响应格式、引入多租户上下文、标准化异步作业流程以及规范化错误码和分页过滤机制。

## Motivation
当前 `disaster-server` 的 API 实现与新制定的风格指南存在部分差异，主要体现在：
- 缺少统一的多租户 (`X-Tenant-ID`) 处理。
- 响应结构 (`Envelope`) 与错误码定义需要严格对齐。
- 直接返回 CRD 结构，暴露了过多底层细节且数据冗余。
- 分页和过滤的参数及响应元数据 (`meta`) 格式不一致。
- 部分 HTTP 头（如 `X-Request-ID`）的大小写不规范。

通过本次重构，我们将提升 API 的一致性、可观测性和可维护性，并为后续的多租户和大规模部署打下基础。

## Proposed Changes

### 1. 中间件增强 (Middleware)
- **TenantID Middleware**: 新增中间件以解析和校验 `X-Tenant-ID` 头，并将其注入上下文。
- **RequestID Middleware**: 修正 Header 名称为 `X-Request-ID` (大写 D)，确保全链路透传。
- **Trace Middleware**: 确保 `trace_id` 在所有响应（包括错误响应）中正确返回。

### 2. 统一响应、数据模型与错误处理 (Response, Data Modeling & Error)
- **Envelope**: 确保所有接口（成功/失败）均返回标准的 `Envelope` 结构。
- **Data Modeling (DTOs)**: 避免直接返回 Kubernetes CRD 完整结构。定义专门的 API 响应结构体 (DTO)，仅包含前端业务所需的必要字段，屏蔽底层实现细节（如 `managedFields`, `metadata` 中的冗余信息）。
- **Error Codes**: 重构 `transport` 包中的错误码定义，实现“双层编码”模式（HTTP Status + Business Code）。
- **Validation**: 引入 JSON Schema 校验或增强现有的绑定校验，确保返回 `400` 及详细的字段错误信息。

### 3. 分页与过滤 (Pagination & Filtering)
- **Query Params**: 统一使用 `page`, `limit`, `sort_by`, `sort_order`, `filters` 参数。
- **Meta Response**: 在集合类响应的 `meta` 字段中包含标准的分页、排序和过滤信息。

## Alternatives Considered
- **维持现状**: 导致前后端对接成本高，错误排查困难，不支持多租户，且直接暴露 CRD 结构导致前端依赖底层实现细节。

## Timeline
- **Phase 1**: 基础架构调整 (Middleware, Transport, Error Codes).
- **Phase 2**: 数据模型定义 (DTOs) 与转换逻辑实现.
- **Phase 3**: 核心资源 (Backup, Restore) 的接口重构与迁移.
- **Phase 4**: 其他资源的列表/查询接口标准化.

## Implementation Plan for Remaining APIs

Following the successful refactoring of `AppBackup`, the following APIs will be refactored to match the new standards (DTOs, Pagination, Filtering, Standard Response):

### 1. AppRestore (`internal/apis/app_restore`)
- **DTO Definition**: Create `AppRestoreDTO` to expose only necessary fields (e.g., `name`, `status`, `backupSource`, `cluster`, `restoreStatus`).
- **List API**: Implement standard pagination, sorting, and filtering.
- **Response**: Wrap responses in standard `Envelope`.

### 2. DisasterCluster (`internal/apis/disaster_cluster`)
- **DTO Definition**: Create `DisasterClusterDTO`.
- **List API**: Implement standard pagination, sorting, and filtering.
- **Response**: Wrap responses in standard `Envelope`.

### 3. DisasterPolicy (`internal/apis/disaster_policy`)
- **DTO Definition**: Create `DisasterPolicyDTO`.
- **List API**: Implement standard pagination, sorting, and filtering.
- **Response**: Wrap responses in standard `Envelope`.

### 4. DisasterStorage (`internal/apis/disaster_storage`)
- **DTO Definition**: Create `DisasterStorageDTO`.
- **List API**: Implement standard pagination, sorting, and filtering.
- **Response**: Wrap responses in standard `Envelope`.

### 5. DisasterJob (`internal/apis/disaster_jobs`)
- **DTO Definition**: Create `DisasterJobDTO`.
- **List API**: Implement standard pagination, sorting, and filtering.
- **Response**: Wrap responses in standard `Envelope`.

### 6. DisasterConfig (`internal/apis/disaster_config`)
- **DTO Definition**: Create `DisasterConfigDTO`.
- **Response**: Wrap responses in standard `Envelope`.

### 7. Events (`internal/apis/event`)
- **DTO Definition**: Create `EventDTO`.
- **List API**: Implement standard pagination, sorting, and filtering.
- **Response**: Wrap responses in standard `Envelope`.

### 8. KubernetesResources (`internal/apis/kubernetes_resources`)
- **Response**: Ensure standard `Envelope` format.

### 9. DisasterBackup (`internal/apis/disaster_backup`)
- **DTO Definition**: Create `DisasterBackupDTO`.
- **List API**: Implement standard pagination, sorting, and filtering.
- **Response**: Wrap responses in standard `Envelope`.

