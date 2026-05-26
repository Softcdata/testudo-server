# 提案：添加 S3 连接验证接口

## 摘要
本提案旨在为 `disaster-server` 添加一个新的 API 接口，用于验证 S3 存储配置（Endpoint, AccessKey, SecretKey, Bucket, Region）的连通性和有效性。

## 动机
目前用户在创建或更新 `StorageRepository` 时，无法即时得知提供的 S3 配置是否正确。如果配置错误，只有在 Operator 尝试调和并报错后用户才能发现，体验较差。提供一个同步的验证接口可以帮助用户在提交配置前进行自检。

## 提议的变更

### API 变更
新增一个 POST 接口 `/storages/connection/validate`。

**请求体 (JSON):**
```json
{
  "endpoint": "http://minio:9000",
  "region": "us-east-1",
  "bucket": "my-bucket",
  "accessKey": "minioadmin",
  "secretKey": "minioadmin",
  "storageType": "s3" // 可选，默认为 s3
}
```

**响应:**
- 成功 (200):
  ```json
  {
    "code": 0,
    "message": "OK",
    "data": true,
    "meta": null,
    "trace_id": "..."
  }
  ```
- 失败 (200):
  ```json
  {
    "code": 0,
    "message": "OK",
    "data": false,
    "meta": {
      "error": "Invalid credentials..."
    },
    "trace_id": "..."
  }
  ```
- 参数错误 (400)

### 实现细节
1.  在 `internal/apis/disaster_storage/v1` 包中添加新的 Handler 方法 `validateS3Connection`。
2.  使用 AWS SDK (如 `github.com/aws/aws-sdk-go` 或 `minio-go`) 创建一个临时的 S3 Client。
3.  尝试执行一个轻量级的操作，如 `HeadBucket` 或 `ListObjects` (limit 1)，以验证凭证和权限。
4.  注册新的路由。

## 影响范围
- `disaster-server` API
- 前端页面 (需要调用此接口进行表单验证)
