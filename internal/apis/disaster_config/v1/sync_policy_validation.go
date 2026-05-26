package config

import (
	"encoding/json"
	"fmt"
	"strings"
)

func rejectUnsupportedSyncPolicyField(body []byte) error {
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}
	if _, exists := payload["syncPolicy"]; !exists {
		return nil
	}

	return fmt.Errorf("syncPolicy is not supported; use dataSyncPolicy and resourcesSyncPolicy")
}
