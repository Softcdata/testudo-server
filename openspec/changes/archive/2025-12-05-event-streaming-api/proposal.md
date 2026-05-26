# Change: 事件流式查询接口 (Event Streaming API)

## Why
目前系统缺乏对资源运行状态的实时反馈机制。用户只能通过查询资源状态（Status）来获取有限的信息，无法像 `kubectl get events -w` 那样实时看到 Kubernetes 产生的详细事件（如调度失败、镜像拉取错误、控制器处理日志等）。

为了提升可观测性和用户体验，需要提供流式接口来实时推送 Kubernetes Events。

## What Changes
新增两个流式 API 接口，用于实时推送 Kubernetes 事件。

### 1. 全局事件流 (Global Events Stream)
监听并推送 `disaster-system` 命名空间下，所有由 Disaster Operator 管理的 CRD 资源产生的事件。

- **URL**: `/v1/events/watch`
- **Method**: `GET`
- **Query Params**:
    - `namespace`: (可选) 默认为 `disaster-system`
- **Behavior**: 建立长连接，实时推送所有相关事件。

### 2. 指定资源事件流 (Resource Specific Events Stream)
监听并推送指定特定资源实例（如某个 AppBackup）产生的事件。

- **URL**: `/v1/:resource/:name/events/watch`
    - e.g., `/v1/appbackups/daily-backup/events/watch`
- **Method**: `GET`
- **Behavior**: 建立长连接，仅推送 `involvedObject` 匹配该资源的事件。

## Implementation Plan

### 1. 技术方案
- 使用 `client-go` 的 `CoreV1().Events(namespace).Watch(...)` 接口。
- 利用 `FieldSelector` 进行服务端过滤：
    - **指定资源**: `involvedObject.name=<name>,involvedObject.kind=<Kind>,involvedObject.uid=<UID>` (UID 可选，但更精确)
    - **全局**: 监听 Namespace 下所有 Event，在内存中过滤 `involvedObject.apiGroup=testudo.softcdata.com` (或者不过滤，直接推该 NS 下所有事件)。
- 复用现有的 `watchutils.StreamWatch` 工具（如果适用）或封装新的 Event Watcher。

### 2. 接口定义

#### 接口 1: 全局事件流
```http
GET /v1/events/watch
```
**Response (Stream)**:
```json
{"type":"ADDED", "object": {...Event...}}
{"type":"ADDED", "object": {...Event...}}
...
```

#### 接口 2: 指定资源事件流
```http
GET /v1/:resource/:name/events/watch
```
**Path Params**:
- `resource`: 资源类型 (e.g., `appbackups`, `disasterclusters`)
- `name`: 资源名称

**Response (Stream)**:
同上，但仅包含该资源的事件。

## Impact
- **新增 Handler**: `internal/apis/event/v1/handler.go`
- **新增 Router**: 注册新的路由组。
- **前端对接**: 前端需要使用 `EventSource` 或长轮询来消费数据。
