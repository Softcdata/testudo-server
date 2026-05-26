# 设计细节 (Design Details)

### 3.1 目录结构
在 `internal/apis` 下的各个模块中（如 `app_backup`, `app_restore`, `disaster_cluster` 等），新增 `types.go` 或 `dto.go` 文件，用于定义 API 专属的结构体。

### 3.2 类型定义示例

所有资源应遵循以下命名规范：
- 创建请求结构体：`Create<ResourceName>Request`
- 更新请求结构体：`Update<ResourceName>Request`
- 转换方法：`func (r *Create<ResourceName>Request) ToCRD() <CRDType>`
- 更新转换方法：`func (r *Update<ResourceName>Request) MergeToCRD(existing *<CRDType>)`

以 `AppRestore` 为例：

```go
// internal/apis/app_restore/types.go

// CreateAppRestoreRequest 定义创建恢复任务的请求体
type CreateAppRestoreRequest struct {
    Name              string                `json:"name" binding:"required"`
    BackupSource      string                `json:"backupSource" binding:"required"`
    Cluster           string                `json:"cluster" binding:"required"`
    TargetNamespace   []string              `json:"targetNamespace,omitempty"` // 简化字段，对应 Template.IncludedNamespaces
    ResourceModifiers []ResourceModifierDTO `json:"resourceModifiers,omitempty"`
    // ... 其他字段
}

type ResourceModifierDTO struct {
    GroupResource string `json:"groupResource" binding:"required"`
    Operation     string `json:"operation" binding:"required,oneof=add remove replace"`
    Path          string `json:"path" binding:"required"`
    Value         string `json:"value,omitempty"`
}

// ToCRD 将请求转换为 Operator 的 CRD Spec
func (r *CreateAppRestoreRequest) ToCRD() disasterv1.AppRestoreSpec {
    // ... 转换逻辑
}
```

### 3.3 处理流程 (Handler Workflow)

1. **BindJSON**: 将 HTTP Body 绑定到 `CreateAppRestoreRequest`。
2. **Validate**: 框架自动执行 `binding` 标签定义的验证，或手动调用验证逻辑。
3. **Convert**: 调用 `req.ToCRD()` 转换为 `disasterv1.AppRestore` 对象。
4. **Process**: 将 CRD 对象传递给后续的 Service/DAO 层进行处理。

### 3.4 验证规则
- 必填字段检查。
- 枚举值检查 (如 `Operation` 类型)。
- 格式检查 (如 Kubernetes 资源名称规范)。
