package event

import (
	"encoding/json"

	"github.com/softcdata/testudo-server/internal/common"
)

func (n EventNode) MarshalJSON() ([]byte, error) {
	type eventNodeJSON struct {
		Time    common.LocalTime `json:"time"`
		Code    string           `json:"code,omitempty"`
		Status  string           `json:"status"`
		Message string           `json:"message"`
		Reason  string           `json:"reason"`
	}
	return json.Marshal(eventNodeJSON{
		Time:    common.NewLocalTimeFromTime(n.Time),
		Code:    n.Code,
		Status:  n.Status,
		Message: n.Message,
		Reason:  n.Reason,
	})
}

func (t TaskEvent) MarshalJSON() ([]byte, error) {
	type taskEventJSON struct {
		ID           string            `json:"id"`
		Time         common.LocalTime  `json:"time"`
		TaskType     string            `json:"taskType"`
		TaskName     string            `json:"taskName"`
		Namespace    string            `json:"namespace"`
		Cluster      string            `json:"cluster"`
		Status       string            `json:"status"`
		Duration     string            `json:"duration"`
		TriggeredBy  string            `json:"triggeredBy"`
		TraceID      string            `json:"traceId"`
		Message      string            `json:"message"`
		ExtraMessage string            `json:"extraMessage,omitempty"`
		StartTime    *common.LocalTime `json:"startTime,omitempty"`
		EndTime      *common.LocalTime `json:"endTime,omitempty"`
		Reason       string            `json:"reason"`
		Code         string            `json:"code,omitempty"`
		Timeline     []EventNode       `json:"timeline"`
	}
	return json.Marshal(taskEventJSON{
		ID:           t.ID,
		Time:         common.NewLocalTimeFromTime(t.Time),
		TaskType:     t.TaskType,
		TaskName:     t.TaskName,
		Namespace:    t.Namespace,
		Cluster:      t.Cluster,
		Status:       t.Status,
		Duration:     t.Duration,
		TriggeredBy:  t.TriggeredBy,
		TraceID:      t.TraceID,
		Message:      t.Message,
		ExtraMessage: t.ExtraMessage,
		StartTime:    common.NewLocalTimePtrFromTime(t.StartTime),
		EndTime:      common.NewLocalTimePtrFromTime(t.EndTime),
		Reason:       t.Reason,
		Code:         t.Code,
		Timeline:     t.Timeline,
	})
}
