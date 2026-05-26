# 提案：存储与策略配置的删除保护及优雅删除

## 摘要
本提案旨在将 `StorageRepository` (存储) 和 `DisasterPolicy` (策略) 的删除保护逻辑下沉至 Operator 层，通过 Kubernetes Finalizer 机制实现。同时移除 Server 端硬编码的删除拦截逻辑，确保无论是通过 API 还是 `kubectl` 删除资源，都能保证数据安全和依赖完整性。

## 动机
目前 `DisasterPolicy` 的删除保护逻辑硬编码在 `disaster-server` 的 API Handler 中，这导致通过 `kubectl` 直接删除资源时无法触发保护，存在误删风险。`StorageRepository` 目前完全没有删除保护，一旦被删除，依赖该存储的备份任务将失败。

为了统一行为并增强安全性，我们需要参考 `Cluster` 资源的实现，引入 Finalizer 和 `Deleting` 状态。

## 提议的变更

### 1. Operator 层变更

#### StorageRepository
- **Status 字段扩展**:
  - 新增 `Phase` (或复用 `Status`): 增加 `Deleting` 状态。
  - 新增 `Reason`, `Message`: 用于记录删除阻塞的原因。
- **Controller 逻辑**:
  - **Finalizer**: 创建时自动添加 `testudo.softcdata.com/storage-finalizer`。
  - **HandleDelete**:
    - 检查是否被 `DisasterConfig` 引用 (通过 Label `testudo.softcdata.com/storage-repository` 检索)。
    - 检查是否被 `AppBackup` / `AppRestore` 引用 (通过 Label `testudo.softcdata.com/storage-repository` 检索)。
    - 如果存在依赖，更新状态为 `Deleting`，设置 Reason 为 `DeletionBlocked`，并 Requeue。
    - 如果无依赖，移除 Finalizer 允许删除。

#### DisasterPolicy
- **Status 字段扩展**:
  - 新增 `Phase` (或复用 `Status` 字段，目前 Policy Status 结构较简单，可能需要重构): 增加 `Deleting` 状态。
  - 新增 `Reason`, `Message`。
- **Controller 逻辑**:
  - **Finalizer**: 创建时自动添加 `testudo.softcdata.com/policy-finalizer`。
  - **HandleDelete**:
    - 检查是否被 `DisasterConfig` 引用 (通过 Label `testudo.softcdata.com/disaster-policy-name` 检索)。
    - 检查是否被 `AppBackup` (自动备份产生的记录) 引用 (通过 Label `testudo.softcdata.com/disaster-policy-name` 检索)。
    - 如果存在依赖，更新状态为 `Deleting`，设置 Reason 为 `DeletionBlocked`，并 Requeue。
    - 如果无依赖，移除 Finalizer 允许删除。

### 2. Server 层变更

- **移除拦截逻辑**:
  - 删除 `internal/apis/disaster_policy/v1/handler.go` 中 `deletePolicy` 方法里的 `AppBackup` 查询拦截代码。
  - 删除 `internal/apis/disaster_storage/v1/handler.go` 中的任何潜在拦截（目前没有，但需确认）。
- **API 行为**:
  - 删除接口调用 K8s Delete 后，如果资源有 Finalizer 且被阻塞，K8s 会返回成功（200/202），但资源会进入 `Terminating` 状态。
  - Server 端无需特殊处理，前端可通过查询详情接口获取 `Deleting` 状态和阻塞原因展示给用户。

## 影响范围
- `disaster-operator`: `StorageRepositoryController`, `DisasterPolicyController`, CRD 定义。
- `disaster-server`: `PolicyHandler`, `StorageHandler`。

## 依赖关系
- 需要先更新 Operator CRD 和 Controller，再更新 Server 去除拦截，否则在 Operator 更新前 Server 去除拦截会导致保护真空期。

## 修改规划

### 1. 定义与补充标签 (pkg/metadata/labels.go)
- 新增 `LabelStorageRepositoryName = "testudo.softcdata.com/storage-repository-name"`
- 确认 `LabelDisasterPolicyName` 已存在。
- 新增 Finalizer 常量:
  - `LabelStorageFinalizer = "testudo.softcdata.com/storage-finalizer"`
  - `LabelPolicyFinalizer = "testudo.softcdata.com/policy-finalizer"`

### 2. 标签注入逻辑 (Controller/Webhook)
为了支持高效检索，需要在依赖资源创建或更新时自动注入标签：
- **DisasterConfig Controller**:
  - 当 Spec 中引用了 `StorageRepository` 时，自动添加 `testudo.softcdata.com/storage-repository-name` 标签。
  - 当 Spec 中引用了 `DisasterPolicy` 时，自动添加 `testudo.softcdata.com/disaster-policy-name` 标签。
- **AppBackup Controller**:
  - 自动添加 `testudo.softcdata.com/storage-repository-name` 标签（源自 Spec.StorageLocation）。
  - 自动添加 `testudo.softcdata.com/disaster-policy-name` 标签（如果是由策略触发）。
- **AppRestore Controller**:
  - 自动添加 `testudo.softcdata.com/storage-repository-name` 标签（源自 Backup 的 StorageLocation）。

### 3. 删除保护实现 (Controller)
- **StorageRepositoryController**:
  - `Reconcile`: 确保 Finalizer 存在。
  - `handleDelete`:
    - List `DisasterConfig` where label `storage-repository-name` == current.Name
    - List `AppBackup` where label `storage-repository-name` == current.Name
    - List `AppRestore` where label `storage-repository-name` == current.Name
    - 若有结果 -> 阻塞删除，更新 Status。
- **DisasterPolicyController**:
  - `Reconcile`: 确保 Finalizer 存在。
  - `handleDelete`:
    - List `DisasterConfig` where label `disaster-policy-name` == current.Name
### 4. Server 响应适配 (Types/DTO)
- **DisasterStorageStatusDTO**:
  - 新增 `Reason` (string)
  - 新增 `Message` (string)
- **DisasterPolicyStatusDTO**:
  - 新增 `Reason` (string)
  - 新增 `Message` (string)
- **Handler**:
  - 确保 `ConvertToDTO` 方法映射了这些新字段，以便前端能展示删除阻塞的原因。

