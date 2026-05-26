## 1. API Design & Models
- [x] 1.1 在 `biz/model/disaster_drill.go` 中，增加 `CleanUp bool` 到请求体内。
- [x] 1.2 在 `biz/model/disaster_drill.go` 的 DTO 中，补充暴露 `Cleanup` 和新状态 (`CleaningUp`, `CleanedUp`) 信息供前端渲染按钮和列表。

## 2. API Handlers & Routing
- [x] 2.1 修改 `biz/router/disaster_drill.go`，注入并注册 `POST /apis/v1/drills/:name/cleanup` 端点。
- [x] 2.2 在 `biz/handler/disaster_drill.go` 实现处理逻辑：
     - 若当前 Drill 状态不属于 `Completed` 或者已有 `Cleanup: true`，拒绝（400）。
     - 通过与 `kube-apiserver` 交互，修改 K8s 对象中 `DisasterDrill` 对应的 `spec.cleanup` 为并落库 `true`。

## 3. Testing 
- [x] 3.1 为 API handler 增加单元测试。
- [x] 3.2 通过 E2E / API Postman 集成测试确保这个调用成功通过授权，并且资源字段修改生效。
