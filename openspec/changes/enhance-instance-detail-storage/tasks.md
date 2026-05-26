# 任务列表：容灾实例详情回显存储仓信息

## 开发任务

### 1. DTO 定义更新
- [ ] 修改 `internal/apis/disaster_instance/v1/types.go`
    - [ ] 在 `DisasterInstanceDTO` 中添加 `Storage *storagev1.DisasterStorageDTO` 字段。
    - [ ] 确保正确引入 `github.com/softcdata/testudo-server/internal/apis/disaster_storage/v1` (作为 `storagev1`)。
    - [ ] 更新 `ConvertToDisasterInstanceDTO` 函数签名，增加 `storage *dapisv1.StorageRepository` 参数。
    - [ ] 在 `ConvertToDisasterInstanceDTO` 内部调用 `storagev1.ConvertToDisasterStorageDTO` 进行转换。

### 2. Handler 逻辑实现
- [ ] 修改 `internal/apis/disaster_instance/v1/handler.go`
    - [ ] 在 `getInstance` 函数中，获取 `DisasterConfig` 后，增加对 `StorageRepository` 的查询逻辑。
    - [ ] 在 `listInstances` 函数中，调用修改后的 `ConvertToDisasterInstanceDTO`（传入 `nil` 存储，保持列表简洁）。
    - [ ] 在 `createInstance`, `updateInstance` 等涉及 DTO 转换的地方，适配新的函数签名。
    - [ ] 在 `watchInstances`, `watchInstance` 等 WS 相关转换逻辑中，如果有需要，获取存储信息（由于 WS 触发频繁，建议此处保持 nil 或只在 Single Watch 时加载）。

### 3. 功能测试
- [ ] 启动 `disaster-server` 进行集成测试。
- [ ] 调用 `GET /apis/testudo.softcdata.com/v1/namespaces/disaster-system/disasterinstances/:name` 确认回显了存储信息。
- [ ] 检查返回的 `storage` 字段，确认不包含 `accessKey` 和 `secretKey`。

## 验收完成情况
- [ ] 详情接口成功回显存储仓详情。
- [ ] 敏感信息已过滤。
- [ ] 现有存量应用兼容性正常（无破坏性变更）。
