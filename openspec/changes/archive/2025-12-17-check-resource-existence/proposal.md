# 提案：创建前检查资源是否存在

## 元数据
- **状态**: 提议中
- **日期**: 2025-12-17
- **作者**: GitHub Copilot
- **评审人**: 待定

## 背景
目前，通过 API 创建资源（例如 `AppRestore`、`AppBackup`）时，服务器可能依赖底层 Kubernetes 客户端来处理冲突，或者不同处理程序的行为可能不一致。

用户要求采用一种标准化的方法，即服务器在尝试创建资源之前，显式检查具有相同名称的资源是否已经存在。如果存在，服务器应返回 `CodeConflict` (HTTP 409) 错误。

## 问题
- 重复资源的错误处理不一致。
- 如果底层存储返回通用错误，可能会导致竞争条件或错误消息不明确。
- 需要在所有创建端点提供统一的用户体验。

## 解决方案
修改服务器中的所有 `create` 处理程序，包含一个预检查步骤：

1.  **提取名称**：从请求体中获取资源名称。
2.  **检查存在性**：使用 Kubernetes 客户端（或适当的 lister/getter）在目标命名空间中按名称查询资源。
3.  **处理结果**：
    - 如果资源**存在**（Get 未返回错误），则返回 `transport.CodeConflict` (HTTP 409)，并附带指示资源已存在的消息。
    - 如果错误是 **NotFound**，则继续执行创建逻辑。
    - 如果检查期间发生任何**其他错误**，则返回 `transport.CodeInternalServerError`（或适当的错误代码）。

## 受影响组件
此更改将影响以下 API 模块中的 `create` 处理程序函数：

- `internal/apis/app_backup`
- `internal/apis/app_restore`
- `internal/apis/disaster_backup`
- `internal/apis/disaster_cluster`
- `internal/apis/disaster_config`
- `internal/apis/disaster_policy`
- `internal/apis/disaster_storage`
- `internal/apis/disaster_jobs` (如果适用)

## 实施计划
1.  遍历每个 API 模块。
2.  定位 `create` 处理程序（例如 `createAppRestore`、`createAppBackup`）。
3.  在处理程序开头（绑定请求之后）插入存在性检查逻辑。
4.  确保使用 `transport.CodeConflict` 进行错误响应。

## 示例 (AppRestore)

```go
func (h *AppRestoreHandler) createAppRestore(c context.Context, ctx *app.RequestContext) {
    var req CreateAppRestoreRequest
    if err := ctx.BindJSON(&req); err != nil {
        transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
        return
    }

    // 新增：检查资源是否存在
    existing, err := h.DisasterClient.DisasterV1().AppRestores(common.DisasterSystemNamespace).Get(c, req.Name, metav1.GetOptions{})
    if err == nil && existing != nil {
        transport.WriteError(ctx, transport.CodeConflict, fmt.Sprintf("AppRestore %s already exists", req.Name), nil)
        return
    }
    if err != nil && !errors.IsNotFound(err) {
         transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
         return
    }

    // ... 现有的创建逻辑 ...
}
```
