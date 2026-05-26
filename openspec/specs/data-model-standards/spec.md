# Capability: 数据模型标准

## Description
规范 `disaster-server` 中数据结构的定义，确保与 `disaster-operator` 定义的 CRD (Custom Resource Definition) 保持一致，避免重复定义并确保版本兼容性。

### Requirement: 基于 Operator Spec 定义
所有涉及业务逻辑的数据对象（DAO）或数据传输对象（DTO），必须基于 `disaster-operator/pkg/apis/disaster/v1` 包下的 `xxxSpec` 类型进行定义或直接复用。

#### Scenario: 复用 CRD Spec
- **WHEN** 定义服务端的数据处理结构体时
- **THEN** 应当直接引用 `github.com/softcdata/testudo-operator/pkg/apis/disaster/v1` 中的 `xxxSpec` 类型（例如 `DisasterBackupSpec`, `DisasterClusterSpec`）
- **AND** 避免在 Server 端重新定义与 CRD Spec 相同的字段结构

#### Scenario: API 请求绑定
- **WHEN** 处理创建或更新资源的 API 请求时
- **THEN** 请求体（Body）应直接绑定到 `disaster-operator` 定义的完整 CRD 结构体（如 `DisasterBackup`）或其 `Spec` 部分
- **AND** 确保 API 接收的数据格式符合 Operator CRD 的定义要求

```go
// 示例：直接使用 Operator 定义的结构体
import dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"

func (cluster *BackupHandler) createBackup(c context.Context, ctx *app.RequestContext) {
    // 直接使用 dapisv1.DisasterBackup，其中包含 DisasterBackupSpec
    body := dapisv1.DisasterBackup{}
    if err := ctx.Bind(&body); err != nil {
        // ...
    }
    // ...
}
```
