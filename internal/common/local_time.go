package common

import (
	"bytes"
	"encoding/json"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LocalTime keeps metav1.Time behavior for server-side code while serializing
// API responses in the process local timezone, for example Asia/Shanghai.
type LocalTime struct {
	metav1.Time
}

func NewLocalTime(t metav1.Time) LocalTime {
	return LocalTime{Time: t}
}

func NewLocalTimePtr(t *metav1.Time) *LocalTime {
	if t == nil || t.IsZero() {
		return nil
	}
	local := NewLocalTime(*t)
	return &local
}

func NewLocalTimeFromTime(t time.Time) LocalTime {
	if t.IsZero() {
		return LocalTime{}
	}
	return NewLocalTime(metav1.NewTime(t))
}

func NewLocalTimePtrFromTime(t *time.Time) *LocalTime {
	if t == nil || t.IsZero() {
		return nil
	}
	local := NewLocalTimeFromTime(*t)
	return &local
}

func FormatLocalRFC3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format(time.RFC3339)
}

func (t LocalTime) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(t.Time.Time.Local().Format(time.RFC3339Nano))
}

func (t *LocalTime) UnmarshalJSON(data []byte) error {
	if t == nil {
		return nil
	}
	if bytes.Equal(data, []byte("null")) {
		t.Time = metav1.Time{}
		return nil
	}

	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if value == "" {
		t.Time = metav1.Time{}
		return nil
	}

	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return err
	}
	t.Time = metav1.NewTime(parsed)
	return nil
}

type LocalCondition struct {
	Type               string                 `json:"type"`
	Status             metav1.ConditionStatus `json:"status"`
	ObservedGeneration int64                  `json:"observedGeneration,omitempty"`
	LastTransitionTime LocalTime              `json:"lastTransitionTime,omitempty"`
	Reason             string                 `json:"reason,omitempty"`
	Message            string                 `json:"message,omitempty"`
}

func NewLocalCondition(condition metav1.Condition) LocalCondition {
	return LocalCondition{
		Type:               condition.Type,
		Status:             condition.Status,
		ObservedGeneration: condition.ObservedGeneration,
		LastTransitionTime: NewLocalTime(condition.LastTransitionTime),
		Reason:             condition.Reason,
		Message:            condition.Message,
	}
}

func NewLocalConditions(conditions []metav1.Condition) []LocalCondition {
	if len(conditions) == 0 {
		return nil
	}
	out := make([]LocalCondition, 0, len(conditions))
	for _, condition := range conditions {
		out = append(out, NewLocalCondition(condition))
	}
	return out
}
