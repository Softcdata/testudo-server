# Change: 使用 RetryOnConflict 优化资源更新 (Use RetryOnConflict for Updates)

## Why
目前 `disaster-server` 中的更新操作（Update）采用简单的 "Get then Update" 模式。在高并发场景下，如果资源在 Get 和 Update 之间被其他客户端修改，Kubernetes API Server 会返回 `409 Conflict` 错误，导致更新失败。为了提高系统的健壮性和用户体验，应该引入 Kubernetes 官方推荐的 `RetryOnConflict` 机制来自动处理版本冲突。

## What Changes
- 引入 `k8s.io/client-go/util/retry` 包。
- 重构所有资源的 `Update` 接口（如 `AppBackup`, `DisasterBackup`, `DisasterCluster` 等）。
- 将更新逻辑封装在 `retry.RetryOnConflict` 回调函数中：
    1. 在回调中重新 Get 最新资源。
    2. 应用修改（如更新 Spec）。
    3. 执行 Update 操作。
    4. 如果返回 Conflict 错误，`RetryOnConflict` 会自动重试。

## Impact
- **受影响的代码**: `internal/apis/` 下所有涉及 Update 操作的 Handler。
- **预期效果**: 显著减少因并发修改导致的更新失败，提升 API 的可靠性。

## Implementation Plan
- [x] 1. 修改 `AppBackup` 的 Update 接口使用 `RetryOnConflict`。
- [x] 2. 修改 `DisasterBackup` 的 Update 接口使用 `RetryOnConflict`。
- [x] 3. 修改 `DisasterCluster` 的 Update 接口使用 `RetryOnConflict` (Note: No Update interface found for Cluster).
- [x] 4. 修改 `DisasterConfig` 的 Update 接口使用 `RetryOnConflict`。
- [x] 5. 修改 `DisasterStorage` 的 Update 接口使用 `RetryOnConflict`。
