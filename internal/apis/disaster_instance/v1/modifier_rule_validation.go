package instance

import (
	"fmt"
	"regexp"
	"strings"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	modifierRuleRejectedCode = "ModifierRuleRejected"

	maxModifierRulesPerInstance = 200
	maxPatchesPerModifierRule   = 50
	maxJSONPointerDepth         = 32
)

func validateRestorePolicyModifierRules(policy *dapisv1.RestorePolicy, instanceNamespaces []string) error {
	if policy == nil || len(policy.ModifierRules) == 0 {
		return nil
	}
	if len(policy.ModifierRules) > maxModifierRulesPerInstance {
		return fmt.Errorf(
			"%s: modifier rule count %d exceeds limit %d",
			modifierRuleRejectedCode,
			len(policy.ModifierRules),
			maxModifierRulesPerInstance,
		)
	}

	for idx := range policy.ModifierRules {
		rule := policy.ModifierRules[idx]
		ruleID := normalizedRestoreModifierRuleID(rule, idx)

		if err := validateRestoreModifierConditions(ruleID, rule.Conditions); err != nil {
			return err
		}
		if err := validateRestoreRuleNamespaceScope(ruleID, rule.Conditions.Namespaces, instanceNamespaces); err != nil {
			return err
		}
		if err := validateRestoreModifierApplyTargets(ruleID, rule.ApplyTo); err != nil {
			return err
		}
		if err := validateRestoreModifierDirectionPolicy(ruleID, rule.DirectionPolicy); err != nil {
			return err
		}
		if err := validateRestoreModifierConflictPolicy(ruleID, rule.OnConflict); err != nil {
			return err
		}

		switch normalizeRestoreModifierMode(rule.Mode) {
		case dapisv1.RestoreModifierModeVeleroNative:
			if err := validateVeleroNativeModifierRule(ruleID, rule); err != nil {
				return err
			}
		case dapisv1.RestoreModifierModeReversible:
			if err := validateReversibleModifierRule(ruleID, rule); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%s: rule=%s unsupported mode=%s", modifierRuleRejectedCode, ruleID, rule.Mode)
		}
	}

	return nil
}

func validateRestoreModifierConditions(ruleID string, conditions dapisv1.Conditions) error {
	if err := validateGroupResource(conditions.GroupResource); err != nil {
		return fmt.Errorf("%s: rule=%s %v", modifierRuleRejectedCode, ruleID, err)
	}

	if conditions.LabelSelector != nil {
		if _, err := metav1.LabelSelectorAsSelector(conditions.LabelSelector); err != nil {
			return fmt.Errorf("%s: rule=%s invalid labelSelector: %v", modifierRuleRejectedCode, ruleID, err)
		}
	}

	if regex := strings.TrimSpace(conditions.ResourceNameRegex); regex != "" {
		if _, err := regexp.Compile(regex); err != nil {
			return fmt.Errorf("%s: rule=%s invalid resourceNameRegex: %v", modifierRuleRejectedCode, ruleID, err)
		}
	}

	return nil
}

func validateVeleroNativeModifierRule(ruleID string, rule dapisv1.RestoreModifierRule) error {
	if rule.VeleroRule == nil {
		return fmt.Errorf("%s: rule=%s veleroNative rule missing veleroRule", modifierRuleRejectedCode, ruleID)
	}
	if len(rule.VeleroRule.MergePatches) > 0 || len(rule.VeleroRule.StrategicPatches) > 0 {
		return fmt.Errorf(
			"%s: rule=%s mergePatches/strategicPatches are not supported in phase 1",
			modifierRuleRejectedCode,
			ruleID,
		)
	}
	if len(rule.VeleroRule.Patches) == 0 {
		return fmt.Errorf("%s: rule=%s veleroNative rule patches cannot be empty", modifierRuleRejectedCode, ruleID)
	}
	if len(rule.VeleroRule.Patches) > maxPatchesPerModifierRule {
		return fmt.Errorf(
			"%s: rule=%s patch count %d exceeds limit %d",
			modifierRuleRejectedCode,
			ruleID,
			len(rule.VeleroRule.Patches),
			maxPatchesPerModifierRule,
		)
	}
	for i := range rule.VeleroRule.Patches {
		if err := validateModifierPatchPath(rule.VeleroRule.Patches[i].Path); err != nil {
			return fmt.Errorf("%s: rule=%s patch[%d] %v", modifierRuleRejectedCode, ruleID, i, err)
		}
	}
	return nil
}

func validateReversibleModifierRule(ruleID string, rule dapisv1.RestoreModifierRule) error {
	if rule.Pair == nil {
		return fmt.Errorf(
			"%s: rule=%s reversible rule must use pair canonical form (pair.path, pair.sourceValue, pair.targetValue)",
			modifierRuleRejectedCode,
			ruleID,
		)
	}
	if err := validateModifierPatchPath(rule.Pair.Path); err != nil {
		return fmt.Errorf("%s: rule=%s %v", modifierRuleRejectedCode, ruleID, err)
	}
	if strings.TrimSpace(rule.Pair.SourceValue) == "" {
		return fmt.Errorf("%s: rule=%s reversible rule missing pair.sourceValue", modifierRuleRejectedCode, ruleID)
	}
	if strings.TrimSpace(rule.Pair.TargetValue) == "" {
		return fmt.Errorf("%s: rule=%s reversible rule missing pair.targetValue", modifierRuleRejectedCode, ruleID)
	}
	return nil
}

func validateModifierPatchPath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("patch path cannot be empty")
	}
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("patch path must start with /")
	}
	if path == "/" {
		return fmt.Errorf("patch path / is not allowed")
	}
	if depth := jsonPointerDepth(path); depth > maxJSONPointerDepth {
		return fmt.Errorf("patch path depth %d exceeds limit %d", depth, maxJSONPointerDepth)
	}
	if err := validateJSONPointer(path); err != nil {
		return err
	}
	if path == "/status" || strings.HasPrefix(path, "/status/") {
		return fmt.Errorf("patch path %s is forbidden", path)
	}
	if path == "/metadata/finalizers" || strings.HasPrefix(path, "/metadata/finalizers/") {
		return fmt.Errorf("patch path %s is forbidden", path)
	}
	if path == "/metadata/ownerReferences" || strings.HasPrefix(path, "/metadata/ownerReferences/") {
		return fmt.Errorf("patch path %s is forbidden", path)
	}
	return nil
}

func validateJSONPointer(path string) error {
	tokens := strings.Split(path[1:], "/")
	for _, token := range tokens {
		if _, err := decodeJSONPointerToken(token); err != nil {
			return err
		}
	}
	return nil
}

func decodeJSONPointerToken(token string) (string, error) {
	if token == "" {
		return "", nil
	}
	var b strings.Builder
	b.Grow(len(token))
	for i := 0; i < len(token); i++ {
		if token[i] != '~' {
			b.WriteByte(token[i])
			continue
		}
		if i+1 >= len(token) {
			return "", fmt.Errorf("invalid JSON Pointer escape at end")
		}
		switch token[i+1] {
		case '0':
			b.WriteByte('~')
		case '1':
			b.WriteByte('/')
		default:
			return "", fmt.Errorf("invalid JSON Pointer escape ~%c", token[i+1])
		}
		i++
	}
	return b.String(), nil
}

func validateGroupResource(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("conditions.groupResource is required")
	}
	parts := strings.Split(raw, ".")
	resource := strings.TrimSpace(parts[0])
	if resource == "" {
		return fmt.Errorf("invalid groupResource=%s", raw)
	}
	return nil
}

func validateRestoreRuleNamespaceScope(ruleID string, ruleNamespaces []string, instanceNamespaces []string) error {
	if len(ruleNamespaces) == 0 || len(instanceNamespaces) == 0 {
		return nil
	}

	allowed := make(map[string]struct{}, len(instanceNamespaces))
	for _, ns := range instanceNamespaces {
		ns = strings.TrimSpace(ns)
		if ns == "" {
			continue
		}
		allowed[ns] = struct{}{}
	}
	if len(allowed) == 0 {
		return nil
	}

	for _, ns := range ruleNamespaces {
		ns = strings.TrimSpace(ns)
		if ns == "" {
			continue
		}
		if _, ok := allowed[ns]; ok {
			continue
		}
		return fmt.Errorf(
			"%s: rule=%s namespace=%s is outside instance namespaces",
			modifierRuleRejectedCode,
			ruleID,
			ns,
		)
	}
	return nil
}

func validateRestoreModifierApplyTargets(ruleID string, applyTo []dapisv1.RestoreModifierApplyTarget) error {
	for _, target := range applyTo {
		switch target {
		case dapisv1.RestoreModifierApplyDataSync, dapisv1.RestoreModifierApplyResourceSync, dapisv1.RestoreModifierApplyDrill:
		default:
			return fmt.Errorf("%s: rule=%s unsupported applyTo=%s", modifierRuleRejectedCode, ruleID, target)
		}
	}
	return nil
}

func validateRestoreModifierDirectionPolicy(ruleID string, policy dapisv1.RestoreModifierDirectionPolicy) error {
	switch normalizeRestoreDirectionPolicy(policy) {
	case dapisv1.RestoreModifierDirectionPolicyAuto,
		dapisv1.RestoreModifierDirectionPolicyForwardOnly,
		dapisv1.RestoreModifierDirectionPolicyReverseOnly:
		return nil
	default:
		return fmt.Errorf("%s: rule=%s unsupported directionPolicy=%s", modifierRuleRejectedCode, ruleID, policy)
	}
}

func validateRestoreModifierConflictPolicy(ruleID string, policy dapisv1.RestoreModifierConflictPolicy) error {
	switch normalizeRestoreConflictPolicy(policy) {
	case dapisv1.RestoreModifierConflictPolicyFail, dapisv1.RestoreModifierConflictPolicySkip:
		return nil
	default:
		return fmt.Errorf("%s: rule=%s unsupported onConflict=%s", modifierRuleRejectedCode, ruleID, policy)
	}
}

func normalizeRestoreModifierMode(mode dapisv1.RestoreModifierMode) dapisv1.RestoreModifierMode {
	switch strings.TrimSpace(string(mode)) {
	case string(dapisv1.RestoreModifierModeVeleroNative):
		return dapisv1.RestoreModifierModeVeleroNative
	case string(dapisv1.RestoreModifierModeReversible):
		return dapisv1.RestoreModifierModeReversible
	default:
		return mode
	}
}

func normalizeRestoreDirectionPolicy(
	policy dapisv1.RestoreModifierDirectionPolicy,
) dapisv1.RestoreModifierDirectionPolicy {
	switch strings.TrimSpace(string(policy)) {
	case string(dapisv1.RestoreModifierDirectionPolicyForwardOnly):
		return dapisv1.RestoreModifierDirectionPolicyForwardOnly
	case string(dapisv1.RestoreModifierDirectionPolicyReverseOnly):
		return dapisv1.RestoreModifierDirectionPolicyReverseOnly
	case "":
		return dapisv1.RestoreModifierDirectionPolicyAuto
	default:
		return policy
	}
}

func normalizeRestoreConflictPolicy(
	policy dapisv1.RestoreModifierConflictPolicy,
) dapisv1.RestoreModifierConflictPolicy {
	switch strings.TrimSpace(string(policy)) {
	case string(dapisv1.RestoreModifierConflictPolicySkip):
		return dapisv1.RestoreModifierConflictPolicySkip
	case "", string(dapisv1.RestoreModifierConflictPolicyFail):
		return dapisv1.RestoreModifierConflictPolicyFail
	default:
		return policy
	}
}

func normalizedRestoreModifierRuleID(rule dapisv1.RestoreModifierRule, idx int) string {
	if id := strings.TrimSpace(rule.ID); id != "" {
		return id
	}
	return fmt.Sprintf("rule-%04d", idx)
}

func jsonPointerDepth(path string) int {
	path = strings.TrimSpace(path)
	if path == "" || path == "/" || !strings.HasPrefix(path, "/") {
		return 0
	}
	depth := 0
	for _, token := range strings.Split(path[1:], "/") {
		if token == "" {
			continue
		}
		depth++
	}
	return depth
}

func isModifierRuleValidationError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), modifierRuleRejectedCode)
}
