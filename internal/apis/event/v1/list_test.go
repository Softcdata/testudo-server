package event

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
)

func TestResolveEventResourceKind(t *testing.T) {
	tests := []struct {
		resource string
		wantKind string
		wantErr  bool
	}{
		{resource: "appbackups", wantKind: "AppBackup"},
		{resource: "AppRestore", wantKind: "AppRestore"},
		{resource: "disasterinstances", wantKind: "DisasterInstance"},
		{resource: "unknown-resource", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.resource, func(t *testing.T) {
			gotKind, err := resolveEventResourceKind(tt.resource)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("resolveEventResourceKind(%q) expected error", tt.resource)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveEventResourceKind(%q) unexpected error: %v", tt.resource, err)
			}
			if gotKind != tt.wantKind {
				t.Fatalf("resolveEventResourceKind(%q)=%q, want %q", tt.resource, gotKind, tt.wantKind)
			}
		})
	}
}

func TestFilterEventsByKind(t *testing.T) {
	items := []corev1.Event{
		{InvolvedObject: corev1.ObjectReference{Kind: "AppBackup", Name: "same-name"}},
		{InvolvedObject: corev1.ObjectReference{Kind: "DataSync", Name: "same-name"}},
		{InvolvedObject: corev1.ObjectReference{Kind: "AppBackup", Name: "same-name"}},
	}

	filtered := filterEventsByKind(items, "AppBackup")
	if len(filtered) != 2 {
		t.Fatalf("filterEventsByKind length=%d, want 2", len(filtered))
	}
	for _, item := range filtered {
		if item.InvolvedObject.Kind != "AppBackup" {
			t.Fatalf("found unexpected kind %s after filter", item.InvolvedObject.Kind)
		}
	}
}

func TestAggregateEvents_SplitByTraceID(t *testing.T) {
	now := time.Now()
	uid := types.UID("uid-1")
	messageWithTrace1 := `{"task":"执行数据同步 dr-ds-web-instance-001","status":"InProgress","cluster":"c1->c2","user":"system","traceId":"trace-1","message":"步骤A"}`
	messageWithTrace2 := `{"task":"执行数据同步 dr-ds-web-instance-001","status":"InProgress","cluster":"c1->c2","user":"system","traceId":"trace-2","message":"步骤A"}`

	items := []corev1.Event{
		{
			InvolvedObject: corev1.ObjectReference{Kind: "DataSync", Name: "dr-ds-web-instance-001", UID: uid},
			Reason:         "ExecutionProgress",
			Message:        messageWithTrace1,
			FirstTimestamp: metav1.NewTime(now),
		},
		{
			InvolvedObject: corev1.ObjectReference{Kind: "DataSync", Name: "dr-ds-web-instance-001", UID: uid},
			Reason:         "ExecutionProgress",
			Message:        messageWithTrace2,
			FirstTimestamp: metav1.NewTime(now.Add(1 * time.Second)),
		},
	}

	h := &EventHandler{}
	got := h.aggregateEvents(items)
	if len(got) != 2 {
		t.Fatalf("aggregateEvents length=%d, want 2", len(got))
	}
}
