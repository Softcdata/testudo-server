# API Standards Delta

## ADDED Requirements

### Requirement: 统一响应信封 (Unified Response Envelope)
所有 API 响应（无论是成功还是错误）都必须 (MUST) 使用标准的 `Envelope` 结构。

#### Scenario: 成功响应
给定一个对 `GET /apis/v1/backups/1` 的请求
当请求被成功处理时
那么响应体应匹配：
```json
{
  "code": 0,
  "message": "OK",
  "data": { ... },
  "meta": { ... },
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
  "message": "Validation failed",
  "data": null,
  "meta": { "details": [...] },
  "trace_id": "..."
}
```

### Requirement: 双层错误码 (Double Layer Error Codes)
系统必须 (MUST) 使用“双层”错误码方案，其中 HTTP 状态码表示传输层状态，`code` 字段表示业务逻辑状态。

#### Scenario: 资源未找到
给定一个对不存在资源的请求
当处理程序处理该请求时
那么 HTTP 状态码应为 404
并且响应体中的 `code` 应为 3004 (CodeNotFound)

### Requirement: 数据传输对象 (DTOs)
API 必须 (MUST) 返回仅包含必要业务字段的 DTO，而不是原始的 Kubernetes CRD。

#### Scenario: 备份列表响应
给定一个对 `GET /apis/v1/backups` 的请求
当获取列表时
Then the items in `data` should be simplified DTOs
And they should NOT contain `managedFields` or full `metadata`

### Requirement: Tenant Context
All requests MUST include a valid `X-Tenant-ID` header, which is validated by middleware.

#### Scenario: Missing Tenant ID
Given a request without `X-Tenant-ID` header
When the request reaches the middleware
Then it should be rejected with HTTP 400 and error code 1000

### Requirement: Pagination and Filtering
Collection APIs MUST support standard pagination and filtering parameters and return standardized metadata.

#### Scenario: Pagination Metadata
Given a request to `GET /apis/v1/backups?page=1&limit=10`
When the response is generated
Then the `meta` field should include:
```json
{
  "pagination": {
    "limit": 10,
    "total": 50,
    "page": 1
  }
}
```
