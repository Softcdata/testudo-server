package instance

import (
	"encoding/json"
	"fmt"
	"strings"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
)

const (
	syncPolicySourceConfig   = "config"
	syncPolicySourceInstance = "instance"
)

func (r *CreateDisasterInstanceRequest) applyPolicyFields(spec *dapisv1.DisasterInstanceSpec) {
	if spec == nil {
		return
	}

	spec.DataSyncPolicy = normalizeOptionalStringPtr(r.DataSyncPolicy)
	spec.ResourceSyncPolicy = normalizeOptionalStringPtr(r.ResourceSyncPolicy)
}

func (r *UpdateDisasterInstanceRequest) applyPolicyFields(spec *dapisv1.DisasterInstanceSpec) {
	if spec == nil {
		return
	}

	if r.DataSyncPolicy != nil {
		spec.DataSyncPolicy = normalizeOptionalStringPtr(r.DataSyncPolicy)
	}
	if r.ResourceSyncPolicy != nil {
		spec.ResourceSyncPolicy = normalizeOptionalStringPtr(r.ResourceSyncPolicy)
	}
}

func populateInstanceSyncPolicyDTO(dto *DisasterInstanceSpecDTO, item *dapisv1.DisasterInstance, config *dapisv1.DisasterConfig) {
	if dto == nil || item == nil {
		return
	}

	effectiveDataSyncPolicy, dataSyncPolicySource := resolveEffectiveInstanceSyncPolicy(
		strings.TrimSpace(item.Spec.DataSyncPolicy),
		effectiveConfigDataSyncPolicy(config),
	)
	effectiveResourceSyncPolicy, resourceSyncPolicySource := resolveEffectiveInstanceSyncPolicy(
		strings.TrimSpace(item.Spec.ResourceSyncPolicy),
		effectiveConfigResourceSyncPolicy(config),
	)

	dto.EffectiveDataSyncPolicy = effectiveDataSyncPolicy
	dto.EffectiveResourceSyncPolicy = effectiveResourceSyncPolicy
	dto.DataSyncPolicySource = dataSyncPolicySource
	dto.ResourceSyncPolicySource = resourceSyncPolicySource
}

func resolveEffectiveInstanceSyncPolicy(instancePolicy, configPolicy string) (effectivePolicy, source string) {
	if instancePolicy = strings.TrimSpace(instancePolicy); instancePolicy != "" {
		return instancePolicy, syncPolicySourceInstance
	}
	if configPolicy = strings.TrimSpace(configPolicy); configPolicy != "" {
		return configPolicy, syncPolicySourceConfig
	}
	return "", ""
}

func effectiveConfigDataSyncPolicy(config *dapisv1.DisasterConfig) string {
	if config == nil {
		return ""
	}
	return strings.TrimSpace(config.Spec.DataSyncPolicy)
}

func effectiveConfigResourceSyncPolicy(config *dapisv1.DisasterConfig) string {
	if config == nil {
		return ""
	}
	return strings.TrimSpace(config.Spec.ResourceSyncPolicy)
}

func normalizeOptionalStringPtr(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

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

	return fmt.Errorf("syncPolicy is not supported; use dataSyncPolicy and resourceSyncPolicy")
}
