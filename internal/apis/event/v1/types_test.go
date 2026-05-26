package event

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
)

func TestConvertToTaskEventDTO_ReturnsNilForUnsupportedReason(t *testing.T) {
	event := &corev1.Event{
		InvolvedObject: corev1.ObjectReference{
			Kind:      "DataSync",
			Name:      "dr-ds-web-instance-001",
			Namespace: "disaster-system",
			UID:       types.UID("uid-1"),
		},
		Reason:         "SyncCompleted",
		Message:        `{"task":"执行数据同步 dr-ds-web-instance-001","status":"Success","message":"完成"}`,
		FirstTimestamp: metav1.NewTime(time.Now()),
	}

	if got := ConvertToTaskEventDTO(event); got != nil {
		t.Fatalf("ConvertToTaskEventDTO()=%v, want nil for unsupported reason", got)
	}
}

func TestConvertToTaskEventDTO_ReturnsTaskEventForSupportedReason(t *testing.T) {
	event := &corev1.Event{
		InvolvedObject: corev1.ObjectReference{
			Kind:      "DataSync",
			Name:      "dr-ds-web-instance-001",
			Namespace: "disaster-system",
			UID:       types.UID("uid-1"),
		},
		Reason:         "ExecutionProgress",
		Message:        `{"task":"执行数据同步 dr-ds-web-instance-001","status":"InProgress","message":"步骤A","traceId":"trace-1","user":"system","cluster":"c1->c2"}`,
		FirstTimestamp: metav1.NewTime(time.Now()),
	}

	got := ConvertToTaskEventDTO(event)
	if got == nil {
		t.Fatalf("ConvertToTaskEventDTO() returned nil for supported reason")
	}
	if _, ok := got.(TaskEvent); !ok {
		t.Fatalf("ConvertToTaskEventDTO() type=%T, want TaskEvent", got)
	}
}
