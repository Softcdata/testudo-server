# 变更：修复 Velero 资源筛选格式兼容

## Why

备份资源清单中的资源键来自 Velero 资源清单，当前接口把 `apps/v1/Deployment` 直接返回给 Web。Web 将该值写入 AppRestore 的 `includedResources` 以及 `excludedResources` 后，Velero 按 `deployments.apps` 匹配，导致 Deployment 被跳过，`existingResourcePolicy=update` 也无法更新资源标签。

## What Changes

- 资源清单接口把历史 GVK 资源键转换为 Velero 使用的 group-resource 格式，例如 `apps/v1/Deployment` 转换为 `deployments.apps`。
- AppRestore 创建和更新接口兼容客户端提交的 GVK 格式，在写入 AppRestore CR 前统一转换 `includedResources` 以及 `excludedResources`。
- 转换使用备份源集群的 RESTMapper；已经是 group-resource 格式的值保持不变。
- GVK 无法解析时返回包含字段路径和原始值的明确错误，禁止把未转换值静默传给 Velero。
- 同步更新 Swagger/OpenAPI、RunAPI 说明和接口证据，说明资源筛选值的格式、兼容输入和错误行为。

## Impact

- 受影响的规范：`api-standards`。
- 受影响的接口：
  - `GET /apis/appbackups.testudo.softcdata.com/v1/velero/backups/{backupName}/includes`
  - `POST /apis/apprestores.testudo.softcdata.com/v1/apprestores`
  - `PUT /apis/apprestores.testudo.softcdata.com/v1/apprestores/{name}`
- 受影响代码：`internal/apis/app_backup/v1/velero_backup_includes.go`、`internal/apis/app_restore/v1/types.go`、`internal/apis/app_restore/v1/handler.go`，以及共享资源筛选转换测试。
- Operator 不需要改变字段转发逻辑；其接收的 AppRestore CR 将保持 Velero group-resource 契约。
