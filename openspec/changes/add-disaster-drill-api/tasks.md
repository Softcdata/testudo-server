# Tasks: 容灾演练 API 实施任务

## 1. 实现准备

### 1.1 CRD 依赖
- [x] 1.1.1 确认 disaster-operator 中 DisasterDrill CRD 已发布
- [x] 1.1.2 更新 disaster-server 依赖的 operator 版本
- [x] 1.1.3 运行 `go mod tidy` 确保依赖正确

## 2. 代码实现

### 2.1 类型定义
- [x] 2.1.1 创建 `internal/apis/disaster_drill/v1/types.go`:
  - DisasterDrillDTO
  - DrillValidationDTO
  - CreateDrillRequest
  - ListDrillsQuery

### 2.2 Handler 实现
- [x] 2.2.1 创建 `internal/apis/disaster_drill/v1/handler.go`:
  - NewDrillHandler 构造函数
  - listDrills 方法
  - getDrill 方法
  - createDrill 方法
  - confirmDrill 方法
  - deleteDrill 方法
  - convertToDTO 辅助函数

### 2.3 路由注册
- [x] 2.3.1 创建 `internal/apis/disaster_drill/v1/router.go`:
  - RegisterRoutes 函数
- [x] 2.3.2 修改 `internal/router/router.go`:
  - 导入 drill 包
  - 注册 `/drills` 路由组

## 3. 功能实现详情

### 3.1 listDrills
- [x] 3.1.1 解析查询参数 (instanceName, state, namespace, limit, page)
- [x] 3.1.2 调用 DisasterClient.DisasterV1().DisasterDrills().List()
- [x] 3.1.3 内存过滤 (按 instanceName, state)
- [x] 3.1.4 分页处理
- [x] 3.1.5 构建响应 (符合 API 标准)

### 3.2 getDrill
- [x] 3.2.1 解析 name 路径参数
- [x] 3.2.2 解析 namespace 查询参数
- [x] 3.2.3 调用 DisasterClient.DisasterV1().DisasterDrills().Get()
- [x] 3.2.4 转换为 DTO 并返回

### 3.3 createDrill
- [x] 3.3.1 解析 CreateDrillRequest
- [x] 3.3.2 校验必填字段 (instanceName)
- [x] 3.3.3 检查 DisasterInstance 是否存在
- [x] 3.3.4 生成演练名称 (如未指定)
- [x] 3.3.5 构建 DisasterDrill CR
- [x] 3.3.6 注入 TraceID
- [x] 3.3.7 调用 Create API
- [x] 3.3.8 返回创建结果

### 3.4 confirmDrill
- [x] 3.4.1 解析 name 路径参数
- [x] 3.4.2 获取当前 DisasterDrill
- [x] 3.4.3 校验状态 (必须是 Ready)
- [x] 3.4.4 使用 Update 设置 spec.confirmed = true
- [x] 3.4.5 返回更新后的状态

### 3.5 deleteDrill
- [x] 3.5.1 解析 name 路径参数
- [x] 3.5.2 调用 Delete API (级联删除 DisasterOperation)
- [x] 3.5.3 返回成功响应

## 4. API 标准合规

### 4.1 响应格式
- [x] 4.1.1 使用 transport.WriteSuccess / transport.WriteError
- [x] 4.1.2 返回标准 Envelope 结构
- [x] 4.1.3 包含 trace_id

### 4.2 分页
- [x] 4.2.1 使用 transport.ParseOptions
- [x] 4.2.2 使用 transport.BuildCollectionResponse
- [x] 4.2.3 返回 pagination 元数据

### 4.3 错误处理
- [x] 4.3.1 未找到资源返回 404 + CodeNotFound
- [x] 4.3.2 参数校验失败返回 400 + CodeBadRequest
- [x] 4.3.3 内部错误返回 500 + CodeInternalServerError

## 5. 测试验证

### 5.1 单元测试
- [x] 5.1.1 创建 handler_test.go (15 个测试用例)
- [x] 5.1.2 测试 listDrills 过滤逻辑
- [x] 5.1.3 测试 createDrill 参数校验
- [x] 5.1.4 测试 confirmDrill 状态校验

### 5.2 集成测试
- [ ] 5.2.1 测试完整演练创建流程
- [ ] 5.2.2 测试确认执行后状态变化
- [ ] 5.2.3 测试删除演练级联删除 Operation

## 6. 文档更新

### 6.1 API 文档
- [x] 6.1.1 在 Apipost 中添加 DisasterDrill API 分组
- [x] 6.1.2 添加各端点的请求/响应示例
- [x] 6.1.3 添加错误码说明

### 6.2 前端对接
- [ ] 6.2.1 通知前端团队 API 变更
- [ ] 6.2.2 提供接口调用示例

## 7. 容灾组演练支持 (新增)

### 7.1 类型扩展 (已实现)
- [x] 7.1.1 更新 `types.go`:
  - 添加 GroupName 字段到 DisasterDrillDTO
  - 添加 DrillGroupProgressDTO 类型
  - 添加 DrillInstanceResultDTO 类型
  - 更新 CreateDrillRequest 支持 groupName

### 7.2 Handler 扩展 (已实现)
- [x] 7.2.1 更新 `createDrill`:
  - 校验 instanceName 和 groupName 互斥
  - 支持容灾组演练创建
  - 校验 DisasterGroup 存在性
- [x] 7.2.2 更新 `convertToDTO`:
  - 处理 GroupProgress 字段映射
  - 处理 InstanceResults 聚合

### 7.3 查询支持 (已实现)
- [x] 7.3.1 更新 `listDrills`:
  - 添加 groupName 过滤参数
  - 支持按容灾组过滤

### 7.4 测试扩展
- [ ] 7.4.1 添加容灾组演练单元测试
- [ ] 7.4.2 测试 groupName 过滤逻辑
- [ ] 7.4.3 测试 instanceName/groupName 互斥校验

## 8. Operator 端容灾组演练支持 (已在 Operator 项目实现)

### 8.1 CRD 扩展
- [x] 8.1.1 更新 DisasterDrill CRD:
  - 添加 spec.groupName 字段
  - 添加 status.groupProgress 字段

### 8.2 Controller 扩展
- [x] 8.2.1 更新 DisasterDrill Controller:
  - 支持 groupName 模式
  - 创建 GroupOperation 类型的 DisasterOperation
- [x] 8.2.2 添加 groupProgress 状态聚合逻辑

## 9. 受保护命名空间查询接口 (Drill 前置辅助)

### 9.1 接口实现
- [x] 9.1.1 新增 `GET /apis/v1/drills/protected-namespaces` 路由
- [x] 9.1.2 支持 `instanceName` / `groupName` 二选一查询
- [x] 9.1.3 容灾组查询时遍历 `DisasterGroup.spec.levels` 并聚合实例命名空间
- [x] 9.1.4 返回去重后的命名空间列表

### 9.2 测试验证
- [x] 9.2.1 新增实例查询成功用例
- [x] 9.2.2 新增容灾组查询成功用例
- [x] 9.2.3 新增参数校验错误用例（同时不传/同时传）
