package event

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/i18n"
	"github.com/softcdata/testudo-server/internal/transport"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	tagRegex = regexp.MustCompile(`\[(Task|Status|Duration|Cluster|User|TraceID): ([^\]]+)\]`)
)

type EventNode struct {
	Time    time.Time `json:"time"`
	Code    string    `json:"code,omitempty"`
	Status  string    `json:"status"`
	Message string    `json:"message"`
	Reason  string    `json:"reason"`
}

type TaskEvent struct {
	ID           string      `json:"id"`
	Time         time.Time   `json:"time"`
	TaskType     string      `json:"taskType"`
	TaskName     string      `json:"taskName"`
	Namespace    string      `json:"namespace"`
	Cluster      string      `json:"cluster"`
	Status       string      `json:"status"`
	Duration     string      `json:"duration"`
	TriggeredBy  string      `json:"triggeredBy"`
	TraceID      string      `json:"traceId"`
	Message      string      `json:"message"`
	ExtraMessage string      `json:"extraMessage,omitempty"` // 用于存储最新的详细操作消息
	StartTime    *time.Time  `json:"startTime,omitempty"`
	EndTime      *time.Time  `json:"endTime,omitempty"`
	Reason       string      `json:"reason"`
	Code         string      `json:"code,omitempty"`
	Timeline     []EventNode `json:"timeline"`
}

// listEvents 获取历史事件列表
func (h *EventHandler) listEvents(c context.Context, ctx *app.RequestContext) {
	namespace := ctx.Query("namespace")
	if namespace == "" {
		namespace = common.DisasterSystemNamespace
	}

	qParams := transport.ParseOptions(c, ctx)

	// 1. List events from K8s with label selector
	listOpts := metav1.ListOptions{
		LabelSelector: "testudo.softcdata.com/task-event=true",
	}
	events, err := h.KubeClient.K8sClient.CoreV1().Events(namespace).List(c, listOpts)
	if err != nil {
		transport.WriteErrorKey(ctx, transport.CodeInternalServerError, i18n.KeyEventsListFailed, nil, err)
		return
	}

	// 2. Aggregate and Convert
	taskEvents := h.aggregateEvents(events.Items)

	// 3. Filter
	taskType := qParams.Filters["taskType"]
	status := qParams.Filters["status"]
	ownerName := qParams.Filters["ownerName"]
	ownerUID := qParams.Filters["ownerUID"]
	startTimeStr := qParams.Filters["startTime"]
	endTimeStr := qParams.Filters["endTime"]
	traceIDSearch := qParams.Filters["traceId"]
	keyword := qParams.Keyword

	filtered := make([]TaskEvent, 0)
	for _, te := range taskEvents {
		// 基本属性筛选
		if taskType != "" && !strings.Contains(te.TaskType, taskType) {
			continue
		}
		if status != "" && te.Status != status {
			continue
		}
		if ownerUID != "" && te.ID != ownerUID {
			continue
		}
		if ownerName != "" && !strings.Contains(te.TaskName, ownerName) {
			continue
		}
		if traceIDSearch != "" && te.TraceID != traceIDSearch {
			continue
		}

		// 时间筛选
		if startTimeStr != "" {
			if st, err := time.Parse(time.RFC3339, startTimeStr); err == nil && te.Time.Before(st) {
				continue
			}
		}
		if endTimeStr != "" {
			if et, err := time.Parse(time.RFC3339, endTimeStr); err == nil && te.Time.After(et) {
				continue
			}
		}

		// 关键字搜索 (任务名称, 集群, 命名空间)
		if keyword != "" {
			if !strings.Contains(te.TaskName, keyword) &&
				!strings.Contains(te.Cluster, keyword) &&
				!strings.Contains(te.Namespace, keyword) &&
				!strings.Contains(te.TraceID, keyword) {
				continue
			}
		}

		filtered = append(filtered, te)
	}

	// 4. Sort
	// 默认按时间倒序
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Time.After(filtered[j].Time)
	})

	// 5. Paginate
	pagedItems, total := transport.Paginate(filtered, qParams)

	// 6. Build Response
	requestPath := string(ctx.URI().Path())
	data, meta := transport.BuildCollectionResponse(
		requestPath,
		"TaskEvent",
		pagedItems,
		qParams,
		total,
		nil,
		nil,
	)

	transport.WriteSuccess(ctx, 200, data, meta)
}

func (h *EventHandler) aggregateEvents(items []corev1.Event) []TaskEvent {
	// Pre-sort events by timestamp to ensure chronological processing
	sort.Slice(items, func(i, j int) bool {
		return items[i].FirstTimestamp.Time.Before(items[j].FirstTimestamp.Time)
	})

	taskMap := make(map[string]*TaskEvent)
	anchorByTask := make(map[string]string)

	// DisasterEventPayload 镜像结构，用于解析 (这里不需要完全匹配，只需解析关键字段)
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

	for _, e := range items {
		// 仅处理我们自定义的结构化事件
		// 支持 ExecutionProgress
		if e.Reason != "ExecutionStarted" && e.Reason != "ExecutionFinished" && e.Reason != "ExecutionProgress" {
			continue
		}

		var payload DisasterEventPayload

		// 严格 JSON 解析，不再回退到正则
		if err := json.Unmarshal([]byte(e.Message), &payload); err != nil {
			// 解析失败则忽略此事件，或者记录 Error
			// 目前这里简单改为 continue，或者您可以选择记录一个 "ParseError"
			continue
		}

		cleanMsg := payload.Message
		if cleanMsg == "" && payload.Status == "" {
			// 有可能是空 JSON 或解析异常，兜底
			cleanMsg = e.Message
		}

		taskName := payload.Task
		if taskName == "" {
			taskName = fmt.Sprintf("%s/%s", e.InvolvedObject.Kind, e.InvolvedObject.Name)
		}
		taskID := buildAggregateTaskID(e, taskName, payload.TraceID, anchorByTask)

		te, exists := taskMap[taskID]
		if !exists {
			te = &TaskEvent{
				ID:        string(e.InvolvedObject.UID),
				Time:      e.FirstTimestamp.Time,
				TaskType:  e.InvolvedObject.Kind,
				TaskName:  payload.Task,
				Namespace: e.InvolvedObject.Namespace,
				Cluster:   payload.Cluster,
				Status:    "Unknown",
				Message:   cleanMsg,
				Timeline:  make([]EventNode, 0),
			}
			if te.TaskName == "" {
				te.TaskName = e.InvolvedObject.Name
			}
			taskMap[taskID] = te
		}

		// Append to Timeline
		status := payload.Status
		node := EventNode{
			Time:    e.FirstTimestamp.Time,
			Code:    payload.Code,
			Status:  status,
			Message: cleanMsg,
			Reason:  e.Reason,
		}
		te.Timeline = append(te.Timeline, node)

		// Update Aggregated State (based on latest event)
		if status != "" {
			// 如果当前状态已经是终态，则不更新
			if te.Status == "Success" || te.Status == "Failed" || te.Status == "Canceled" {
				// do nothing
			} else {
				te.Status = status
			}
		}
		// Always update message to reflect current step
		te.Message = cleanMsg
		te.ExtraMessage = cleanMsg

		// Update Metadata
		if te.TriggeredBy == "" {
			te.TriggeredBy = payload.User
		}
		if te.TraceID == "" {
			te.TraceID = payload.TraceID
		}
		if te.Cluster == "" || te.Cluster == "-" {
			te.Cluster = payload.Cluster
		}

		// Time calculations
		if e.Reason == "ExecutionFinished" {
			te.Duration = payload.Duration
			t := e.FirstTimestamp.Time
			te.EndTime = &t

			// Fix: For single-shot events (where no Started event exists), StartTime would be nil.
			// Back-calculate StartTime from Duration or default to EndTime.
			if te.StartTime == nil {
				if d, err := time.ParseDuration(payload.Duration); err == nil {
					st := t.Add(-d)
					te.StartTime = &st
				} else {
					te.StartTime = &t
				}
			}
		} else if e.Reason == "ExecutionStarted" {
			t := e.FirstTimestamp.Time
			te.StartTime = &t
		}

		// Duration calculation for InProgress (if Duration is empty from payload)
		if te.Status == "InProgress" && te.StartTime != nil && payload.Duration == "" {
			te.Duration = time.Since(*te.StartTime).Round(time.Second).String()
		}

		// Update main Time to be the latest activity time (for sorting results)
		te.Time = e.FirstTimestamp.Time
		// Update Reason to reflect the latest action
		te.Reason = e.Reason
		te.Code = payload.Code
	}

	result := make([]TaskEvent, 0, len(taskMap))
	for _, v := range taskMap {
		result = append(result, *v)
	}
	return result
}

func buildAggregateTaskID(e corev1.Event, taskName, traceID string, anchorByTask map[string]string) string {
	uid := string(e.InvolvedObject.UID)
	if uid == "" {
		uid = fmt.Sprintf("%s/%s/%s", e.InvolvedObject.Namespace, e.InvolvedObject.Kind, e.InvolvedObject.Name)
	}
	baseTaskKey := fmt.Sprintf("%s|%s", taskName, uid)
	normalizedTraceID := strings.TrimSpace(traceID)
	if normalizedTraceID != "" && normalizedTraceID != "-" {
		return fmt.Sprintf("%s|trace:%s", baseTaskKey, normalizedTraceID)
	}

	if e.Reason == "ExecutionStarted" {
		anchorByTask[baseTaskKey] = e.FirstTimestamp.Time.UTC().Format(time.RFC3339Nano)
	}
	anchor, ok := anchorByTask[baseTaskKey]
	if !ok {
		anchor = e.FirstTimestamp.Time.UTC().Format(time.RFC3339Nano)
		anchorByTask[baseTaskKey] = anchor
	}
	if e.Reason == "ExecutionFinished" {
		defer delete(anchorByTask, baseTaskKey)
	}
	return fmt.Sprintf("%s|trace:-|anchor:%s", baseTaskKey, anchor)
}

// listResourceEvents 获取指定资源的历史事件
func (h *EventHandler) listResourceEvents(c context.Context, ctx *app.RequestContext) {
	resourceType := ctx.Param("resource")
	name := ctx.Param("name")
	namespace := ctx.Query("namespace")
	if namespace == "" {
		namespace = common.DisasterSystemNamespace
	}
	if resourceType == "" || name == "" {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationResourceNameRequired, nil, nil)
		return
	}
	expectedKind, err := resolveEventResourceKind(resourceType)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	qParams := transport.ParseOptions(c, ctx)
	startTimeStr := qParams.Filters["startTime"]
	endTimeStr := qParams.Filters["endTime"]

	fieldSelector := fmt.Sprintf("involvedObject.name=%s", name)

	events, err := h.KubeClient.K8sClient.CoreV1().Events(namespace).List(c, metav1.ListOptions{
		FieldSelector: fieldSelector,
		LabelSelector: "testudo.softcdata.com/task-event=true",
	})
	if err != nil {
		transport.WriteErrorKey(ctx, transport.CodeInternalServerError, i18n.KeyEventsListFailed, nil, err)
		return
	}
	taskEvents := h.aggregateEvents(filterEventsByKind(events.Items, expectedKind))

	// Filter by time
	filtered := make([]TaskEvent, 0)
	for _, te := range taskEvents {
		if startTimeStr != "" {
			if st, err := time.Parse(time.RFC3339, startTimeStr); err == nil && te.Time.Before(st) {
				continue
			}
		}
		if endTimeStr != "" {
			if et, err := time.Parse(time.RFC3339, endTimeStr); err == nil && te.Time.After(et) {
				continue
			}
		}
		filtered = append(filtered, te)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Time.After(filtered[j].Time)
	})

	// 5. Paginate
	pagedItems, total := transport.Paginate(filtered, qParams)

	// 6. Build Response
	requestPath := string(ctx.URI().Path())
	data, meta := transport.BuildCollectionResponse(
		requestPath,
		"TaskEvent",
		pagedItems,
		qParams,
		total,
		nil,
		nil,
	)

	transport.WriteSuccess(ctx, 200, data, meta)
}
