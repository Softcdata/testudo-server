# Proposal: V2 链路追踪 (Trace ID Propagation for V2)

## 1. 背景与问题 (Background)
V1 版本已经实现了 `trace_id` 的全链路传播（从 CRD 传递到 Velero 资源）。
V2 版本引入了新的 CRD 体系 (`DisasterOperation`, `DisasterInstance`)，目前 Server 端在创建 V2 操作时，可能尚未从 HTTP 请求中提取 Trace ID 并注入到 CRD 中。

## 2. 目标 (Goals)
确保 `disaster-server` 在处理 V2 版本的实例操作（如 `ExecuteAction`）时，能够：
1.  从 HTTP 请求头（如 `X-Trace-ID`）中获取 Trace ID。
2.  如果没有 Trace ID，则生成一个新的。
3.  将 Trace ID 注入到创建的 `DisasterOperation` 资源的 Annotations 中，Key 为 `testudo.softcdata.com/trace-id`。
4.  确保 `DisasterInstance` 和 `DisasterGroup` 的相关操作 API 都支持此机制。

## 3. 设计方案 (Design)

### 3.1 识别 Trace ID
### 3.1 识别与注入
在 `ExecuteAction` (DisasterInstance) 和 `ExecuteGroupAction` (DisasterGroup) 等 Handler 中：
1.  检查 Request Header `X-Trace-ID`。
2.  如果为空，生成一个 UUID。

### 3.2 注入 Annotation
在构造 `DisasterOperation` 对象时：
```go
op := &disasterv1.DisasterOperation{
    ObjectMeta: metav1.ObjectMeta{
        // ...
        Annotations: map[string]string{
            "testudo.softcdata.com/trace-id": traceID,
        },
    },
    // ...
}
```

## 4. 任务分解 (Tasks)
1.  修改 `internal/apis/disaster_instance/v1/handler_action.go` 中的 `ExecuteAction` 逻辑。
2.  修改 `internal/apis/disaster_group/v1/handler.go` 中的相关 Action 逻辑 (如 `Failover`, `Reprotect` 等，如果有创建 DisasterOperation 的地方)。
3.  添加单元测试验证 Annotation 注入。
