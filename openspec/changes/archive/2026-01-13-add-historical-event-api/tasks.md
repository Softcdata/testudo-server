# 任务清单：增加历史事件列表接口

## 准备阶段
- [x] 调研 `AppBackup` 和 `AppRestore` 发射的所有 Reason 类型。

## 核心开发
- [x] **API 结构体定义**:
    - [x] 在 `internal/apis/event/v1/types.go` 或新文件中定义 `HistoricalEventDTO`。
- [x] **List Handler 实现**:
    - [x] 实现 `listEvents` (全局) 和 `listResourceEvents` (资源相关)。
    - [x] 实现事件解析与聚合逻辑 (Aggregation Logic)。
    - [x] 支持根据 `TaskType`, `Status`, `TriggeredBy` 等进行内存过滤。
- [x] **路由注册**:
    - [x] 在 `router.go` 中挂载新接口。

## 系统规范变更
- [x] 更新 `openspec/specs/api-standards/spec.md` 包含新的事件列表 API 定义。

## 验证
- [x] 编写集成测试，模拟 K8s 事件并验证列表接口的过滤能力。
- [x] 在模拟环境中验证分页功能。
