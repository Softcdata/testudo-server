package event

import (
	"strings"
	"testing"
)

func TestBuildTaskEventLabelSelector(t *testing.T) {
	tests := []struct {
		name        string
		origin      string
		wantContain []string
		wantErr     bool
	}{
		{
			name:   "default user",
			origin: "",
			wantContain: []string{
				"testudo.softcdata.com/task-event=true",
				taskOriginLabelKey + "!=" + taskOriginDisasterInstance,
			},
		},
		{
			name:   "origin user",
			origin: "user",
			wantContain: []string{
				"testudo.softcdata.com/task-event=true",
				taskOriginLabelKey + "!=" + taskOriginDisasterInstance,
			},
		},
		{
			name:   "origin instance",
			origin: "instance",
			wantContain: []string{
				"testudo.softcdata.com/task-event=true",
				taskOriginLabelKey + "=" + taskOriginDisasterInstance,
			},
		},
		{
			name:   "origin all",
			origin: "all",
			wantContain: []string{
				"testudo.softcdata.com/task-event=true",
			},
		},
		{
			name:    "invalid origin",
			origin:  "bad",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector, err := buildTaskEventLabelSelector(tt.origin)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("buildTaskEventLabelSelector() expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("buildTaskEventLabelSelector() unexpected error: %v", err)
			}
			for _, fragment := range tt.wantContain {
				if !strings.Contains(selector, fragment) {
					t.Fatalf("selector %q does not contain %q", selector, fragment)
				}
			}
		})
	}
}
