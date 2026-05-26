package instance

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

type bulkRelevantSpecState struct {
	Config        string
	Namespaces    []string
	LabelSelector *metav1.LabelSelector
	RestorePolicy bulkRelevantRestorePolicyState
}

type bulkRelevantRestorePolicyState struct {
	BulkModifierActions []dapisv1.BulkModifierAction
	ModifierRules       []dapisv1.RestoreModifierRule
	ResourceSelection   *dapisv1.RestoreResourceSelectionPolicy
}

func (h *InstanceHandler) prepareRestorePolicyForPersist(
	ctx context.Context,
	spec *dapisv1.DisasterInstanceSpec,
	previous *dapisv1.DisasterInstance,
) error {
	if spec == nil || spec.RestorePolicy == nil {
		return nil
	}

	policy := spec.RestorePolicy
	if previous != nil && !bulkRelatedInputsChanged(&previous.Spec, spec) {
		if previous.Spec.RestorePolicy != nil {
			policy.ModifierRuleSnapshot = cloneRestoreModifierRules(previous.Spec.RestorePolicy.ModifierRuleSnapshot)
			policy.ModifierRuleSnapshotHash = strings.TrimSpace(previous.Spec.RestorePolicy.ModifierRuleSnapshotHash)
			policy.BulkModifierActions = cloneBulkModifierActions(previous.Spec.RestorePolicy.BulkModifierActions)
		}
		return nil
	}

	normalizedActions, err := normalizeBulkModifierActions(policy.BulkModifierActions)
	if err != nil {
		return err
	}
	policy.BulkModifierActions = normalizedActions

	if len(enabledBulkModifierActions(normalizedActions)) == 0 {
		policy.ModifierRuleSnapshot = nil
		policy.ModifierRuleSnapshotHash = ""
		return h.validateEffectiveModifierRules(ctx, spec, nil)
	}

	buildSnapshot := h.BuildBulkModifierSnapshotFunc
	usingDefaultBuilder := buildSnapshot == nil
	if usingDefaultBuilder {
		buildSnapshot = buildBulkModifierSnapshot
	}

	var sourceRESTConfig *rest.Config
	if h.shouldRunLiveModifierRuleValidation() {
		resolved, err := h.resolveInstanceSourceCluster(ctx, spec.Config)
		if err != nil {
			return fmt.Errorf("ModifierRuleRejected: %v", err)
		}
		sourceRESTConfig, err = h.getClusterRESTConfig(ctx, resolved)
		if err != nil {
			return fmt.Errorf(
				"ModifierRuleRejected: build rest config for source cluster %s failed: %v",
				resolved,
				err,
			)
		}
	} else if usingDefaultBuilder {
		return fmt.Errorf("ModifierRuleRejected: cluster client is not initialized for bulk modifier snapshot generation")
	}

	result, err := buildSnapshot(ctx, spec, sourceRESTConfig)
	if err != nil {
		return err
	}
	policy.BulkModifierActions = cloneBulkModifierActions(result.Actions)
	policy.ModifierRuleSnapshot = cloneRestoreModifierRules(result.ModifierRuleSnapshot)
	policy.ModifierRuleSnapshotHash = strings.TrimSpace(result.ModifierRuleSnapshotHash)

	return h.validateEffectiveModifierRules(ctx, spec, sourceRESTConfig)
}

// PrepareRestorePolicyForPersist exposes the shared bulk snapshot preparation / modifier
// validation flow so other API modules (for example drill create) can persist a policy in the
// same canonical form as instances.
func (h *InstanceHandler) PrepareRestorePolicyForPersist(
	ctx context.Context,
	spec *dapisv1.DisasterInstanceSpec,
	previous *dapisv1.DisasterInstance,
) error {
	return h.prepareRestorePolicyForPersist(ctx, spec, previous)
}

func (h *InstanceHandler) validateEffectiveModifierRules(
	ctx context.Context,
	spec *dapisv1.DisasterInstanceSpec,
	sourceRESTConfig *rest.Config,
) error {
	if spec == nil || spec.RestorePolicy == nil {
		return nil
	}

	effectivePolicy := spec.RestorePolicy.DeepCopy()
	if len(enabledBulkModifierActions(effectivePolicy.BulkModifierActions)) > 0 {
		effectivePolicy.ModifierRules = cloneRestoreModifierRules(effectivePolicy.ModifierRuleSnapshot)
	} else {
		effectivePolicy.ModifierRuleSnapshot = nil
		effectivePolicy.ModifierRuleSnapshotHash = ""
	}

	if err := validateRestorePolicyModifierRules(effectivePolicy, spec.Namespaces); err != nil {
		return err
	}
	if len(effectivePolicy.ModifierRules) == 0 || sourceRESTConfig == nil {
		return nil
	}
	return validateRestorePolicyModifierRulesLive(ctx, effectivePolicy, spec.Namespaces, sourceRESTConfig)
}

func bulkRelatedInputsChanged(current *dapisv1.DisasterInstanceSpec, desired *dapisv1.DisasterInstanceSpec) bool {
	return !reflect.DeepEqual(extractBulkRelevantSpecState(current), extractBulkRelevantSpecState(desired))
}

func extractBulkRelevantSpecState(spec *dapisv1.DisasterInstanceSpec) bulkRelevantSpecState {
	out := bulkRelevantSpecState{}
	if spec == nil {
		return out
	}
	out.Config = strings.TrimSpace(spec.Config)
	out.Namespaces = append([]string(nil), spec.Namespaces...)
	if spec.LabelSelector != nil {
		out.LabelSelector = spec.LabelSelector.DeepCopy()
	}
	if spec.RestorePolicy == nil {
		return out
	}
	out.RestorePolicy = bulkRelevantRestorePolicyState{
		BulkModifierActions: cloneBulkModifierActions(spec.RestorePolicy.BulkModifierActions),
		ModifierRules:       cloneRestoreModifierRules(spec.RestorePolicy.ModifierRules),
	}
	if spec.RestorePolicy.ResourceSelection != nil {
		out.RestorePolicy.ResourceSelection = spec.RestorePolicy.ResourceSelection.DeepCopy()
	}
	return out
}
