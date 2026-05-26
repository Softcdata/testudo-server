## 1. Implementation
- [x] 1.1 增加 License 配置项，默认命名空间为 `disaster-system`，默认 CA 路径为 ServiceAccount CA。
- [x] 1.2 新增 platform-license handler、router、DTO 与 Kubernetes 存取逻辑。
- [x] 1.3 在 Cluster 创建接口中加入基于 Secret 与当前指纹的创建前门禁。
- [x] 1.4 返回稳定 License 错误 reason 与额度元信息。
- [x] 1.5 注册新增 License API 路由。

## 2. Documentation
- [x] 2.1 更新 Swagger/OpenAPI，新增 License 状态与安装接口。
- [x] 2.2 更新 Swagger/OpenAPI，补充 Cluster 创建接口 License 错误说明。
- [x] 2.3 更新 RunAPI，新增 License 状态与安装接口文档。
- [x] 2.4 更新 RunAPI，补充 Cluster 创建接口 License 错误说明。

## 3. Validation
- [x] 3.1 增加 Cluster 创建 License 门禁测试。
- [x] 3.2 增加 License 状态与安装接口测试。
- [x] 3.3 运行目标 Go 测试。
- [x] 3.4 运行 OpenSpec 严格校验。
- [x] 3.5 运行 OpenAPI 校验。
