# Capability: 测试标准

## Description
定义 `disaster-server` 的单元测试和集成测试标准，确保代码质量和功能的正确性。

### Requirement: 测试框架
所有 Go 代码的测试必须使用 `github.com/stretchr/testify` 库进行断言和 Mock。

#### Scenario: 使用断言库
- **WHEN** 编写测试用例时
- **THEN** 必须使用 `testify/assert` 或 `testify/require` 包进行结果验证
- **AND** 避免使用 Go 原生的 `if got != want` 模式，以提高测试代码的可读性

```go
import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestSomething(t *testing.T) {
    result := SomeFunction()
    assert.Equal(t, "expected", result, "should match expected value")
    assert.NotNil(t, result)
}
```

### Requirement: 接口测试覆盖
所有 API Handler 的实现完成后，必须编写对应的单元测试，且必须覆盖所有实现的接口方法（如 List, Get, Create, Update, Delete, Watch 等）。

#### Scenario: Handler 测试
- **WHEN** 完成一个新的 API Handler (如 `AppBackupHandler`)
- **THEN** 必须在同级目录下创建 `handler_test.go` 文件
- **AND** 必须为每一个实现的接口方法编写测试用例
- **AND** 使用 `testify` 验证 HTTP 状态码和响应体内容

```go
func TestCreateAppBackup(t *testing.T) {
    // Setup
    // ...
    
    // Execute
    // ...

    // Assert
    assert.Equal(t, 201, resp.StatusCode())
}
```

### Requirement: Hertz Handler 测试注意事项

#### Scenario: JSON 数据绑定
- **WHEN** 在 Handler 中解析 JSON 请求体
- **THEN** 建议使用 `ctx.BindJSON()` 替代 `ctx.Bind()`
- **REASON** 在单元测试环境中，`ctx.Bind()` 可能会因为 `sonic` 库的版本兼容性问题导致解析失败（报错 `WARNING:(ast) sonic only supports...`），而 `BindJSON()` 在测试中表现更稳定。

### Requirement: Kubernetes 资源 Mock
在测试涉及 Kubernetes 资源操作的 Handler 时，必须使用 `client-go` 提供的 `fake` 包来模拟 Kubernetes 客户端，而不是依赖真实的集群。

#### Scenario: 使用 Fake Clientset
- **WHEN** 测试需要与 Kubernetes API 交互的代码
- **THEN** 使用 `fake.NewSimpleClientset(objects...)` 创建模拟客户端
- **AND** 将预置的资源对象传入构造函数以模拟初始状态

```go
func newMockHandler(objects ...runtime.Object) *AppBackupHandler {
    fakeClient := fake.NewSimpleClientset(objects...)
    kc := &kube.KubeClient{
        DisasterClient: fakeClient,
    }
    
    h := server.Default()
    rg := h.Group("/v1")
    
    return NewAppBackupHandler(kc, rg)
}
```

