package v1

import (
	"encoding/json"
	"testing"
)

func TestGroupMemberStatusDTOAlwaysSerializesReasonAndMessage(t *testing.T) {
	dto := GroupMemberInstanceDTO{
		Name: "inst-a",
		Status: GroupMemberStatusDTO{
			State: "Protected",
		},
		FsmState: "Protected",
	}

	raw, err := json.Marshal(dto)
	if err != nil {
		t.Fatalf("marshal dto failed: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal dto failed: %v", err)
	}

	status, ok := decoded["status"].(map[string]any)
	if !ok {
		t.Fatalf("status should be an object, got %T", decoded["status"])
	}

	if _, ok := status["reason"]; !ok {
		t.Fatalf("status.reason should always exist")
	}
	if _, ok := status["message"]; !ok {
		t.Fatalf("status.message should always exist")
	}
}
