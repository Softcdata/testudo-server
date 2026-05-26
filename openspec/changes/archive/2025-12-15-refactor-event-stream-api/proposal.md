# Proposal: 重构事件流 API 以符合 API 标准

## Why
目前的事件流 API (WebSocket) 直接返回原始的 Kubernetes 事件对象或自定义的 `WatchMessage` 结构，不符合全局定义的 API 标准（统一响应信封 `Envelope` 和 DTO 规范）。这导致前端处理 WebSocket 消息时需要编写特殊的逻辑，增加了维护成本和不一致性。

## What Changes
本提案旨在重构所有使用 WebSocket 的事件流接口，使其遵循统一的 API 交互标准。

主要变更包括：
1.  **统一消息结构**：WebSocket 推送的消息必须使用标准的 `Envelope` 结构（包含 `code`, `message`, `data`, `meta`, `trace_id`）。
2.  **DTO 转换**：`data` 字段中的事件对象必须转换为 DTO，而不是直接返回原始 Kubernetes 对象。
3.  **标准化 WatchEvent**：定义标准的 `WatchEventDTO`，包含 `type` (ADDED, MODIFIED, DELETED, ERROR) 和 `object` (DTO)。

受影响的组件：
- `internal/utils/watch.go`: `StreamWatch` 工具函数
- `internal/apis/event`: 事件监听接口
- `internal/apis/disaster_cluster`: 集群监听接口
