# Proposal: 添加集群 Token 认证支持

## Summary
除了现有的 KubeConfig 方式外，增加使用 Token 连接和验证 Kubernetes 集群的支持。这将涉及修改 CRD 以包含 Endpoint 字段，并在服务端和 Operator 端实现相应的连接逻辑。

## Motivation
当前的实现仅支持通过 KubeConfig 连接集群。CRD 中已经包含了 `token` 字段，但服务端尚未利用它进行验证或连接，且缺少必要的 `endpoint` 字段来定位集群。支持基于 Token 的认证提供了一种更灵活、更安全的集群连接方式，特别是在 KubeConfig 文件不易获取或管理的场景中。

## Proposed Changes

### 1. API & CRD 变更
- 修改 `ClusterSpec` (在 `pkg/apis/disaster/v1/cluster_types.go`)：
    - 添加 `Endpoint` 字段 (string, optional)。
    - 说明：当 `KubeConfig` 为空时，`Token` 和 `Endpoint` 为必填项。

### 2. Server 端实现 (`disaster-server`)
- **DTO 变更**：
    - 修改或新增请求结构体 `ValidateConnectionRequest` (替代 `ValidateKubeConfigRequest`)。
    - 字段：`KubeConfig` (bytes, optional), `Token` (string, optional), `Endpoint` (string, optional)。
- **Handler 变更** (`internal/apis/disaster_cluster/v1/handler.go`)：
    - 将 `validateKubeConfig` 重构为 `validateConnection`。
    - **逻辑流程**：
        1. 接收请求参数。
        2. 如果 `KubeConfig` 不为空：
           - 调用 `tools.GetRestConfig(req.KubeConfig)`。
        3. 如果 `KubeConfig` 为空，但 `Token` 和 `Endpoint` 不为空：
           - 构造 `rest.Config`：
             ```go
             &rest.Config{
                 Host: req.Endpoint,
                 BearerToken: req.Token,
                 TLSClientConfig: rest.TLSClientConfig{Insecure: true},
             }
             ```
        4. 使用生成的 config 创建 `kubernetes.Clientset`。
        5. 调用 `clientset.Discovery().ServerVersion()` 验证连接。
        6. 如果成功，返回 true；否则返回错误信息。

### 4. 错误处理优化
- **API 响应**：
    - 即使验证失败，API 也应返回 HTTP 200 OK。
    - 响应体结构：
      ```json
      {
        "code": 0,
        "msg": "error message if failed",
        "data": false // true if success
      }
      ```
    - 避免直接返回 HTTP 400/500 导致前端处理复杂。

### 5. 依赖更新
- 更新 `disaster-server` 对 `disaster-operator` 的依赖，以获取最新的 CRD 定义。
