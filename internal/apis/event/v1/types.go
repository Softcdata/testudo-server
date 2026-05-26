package event

import (
	"encoding/json"
	"time"

	corev1 "k8s.io/api/core/v1"
)

// ConvertToTaskEventDTO 将 K8s Watch 事件转换为前端所需的 TaskEvent 格式
func ConvertToTaskEventDTO(obj interface{}) interface{} {
	event, ok := obj.(*corev1.Event)
	if !ok {
		return nil
	}

	// 仅处理我们自定义的结构化事件
	if event.Reason != "ExecutionStarted" && event.Reason != "ExecutionFinished" && event.Reason != "ExecutionProgress" {
		return nil
	}

	// DisasterEventPayload 镜像结构
	type DisasterEventPayload struct {
		Task     string `json:"task"`
		Status   string `json:"status"`
		Cluster  string `json:"cluster"`
		User     string `json:"user"`
		TraceID  string `json:"traceId"`
		Duration string `json:"duration"`
		Code     string `json:"code"`
		Message  string `json:"message"`
	}

	var payload DisasterEventPayload

	// 尝试 JSON 解析
	if err := json.Unmarshal([]byte(event.Message), &payload); err != nil {
		// 解析失败 (可能是旧格式或非结构化日志)，直接返回 nil 忽略，或者降级处理
		// 既然要求“彻底删除旧逻辑”，这里就不再尝试 parseTags
		return nil
	}

	taskName := payload.Task
	if taskName == "" {
		taskName = event.InvolvedObject.Name
	}

	// 构建基础 TaskEvent
	te := TaskEvent{
		ID:           string(event.InvolvedObject.UID),
		Time:         event.FirstTimestamp.Time,
		TaskType:     event.InvolvedObject.Kind,
		TaskName:     taskName,
		Namespace:    event.InvolvedObject.Namespace,
		Cluster:      payload.Cluster,
		Status:       payload.Status,
		Duration:     payload.Duration,
		TriggeredBy:  payload.User,
		TraceID:      payload.TraceID,
		Message:      payload.Message, // 清洗后的消息
		ExtraMessage: payload.Message, // 统一填充
		Reason:       event.Reason,
		Code:         payload.Code,
		StartTime:    &event.FirstTimestamp.Time,
	}

	if te.Status == "" {
		te.Status = "Unknown"
	}
	if te.Duration == "" || te.Duration == "-" {
		te.Duration = "-"
		if te.Status == "InProgress" {
			te.Duration = time.Since(te.Time).Round(time.Second).String()
		}
	}

	// 兜底
	if te.Cluster == "" {
		te.Cluster = "-"
	}
	if te.TriggeredBy == "" {
		te.TriggeredBy = "system"
	}

	// 如果 Status 是 InProgress 且是 ExecutionStarted，修正显示
	if event.Reason == "ExecutionStarted" {
		te.Status = "InProgress"
		t := event.FirstTimestamp.Time
		te.StartTime = &t
	} else if event.Reason == "ExecutionFinished" {
		t := event.FirstTimestamp.Time
		te.EndTime = &t
		// Finished 时，StartTime 应该是之前的，但单事件流拿不到之前的。
	}

	return te
}
