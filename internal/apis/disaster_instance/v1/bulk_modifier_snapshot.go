package instance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

const bulkGeneratedRulePriority int32 = -100

type bulkModifierSnapshotBuildResult struct {
	Actions                  []dapisv1.BulkModifierAction
	ModifierRuleSnapshot     []dapisv1.RestoreModifierRule
	ModifierRuleSnapshotHash string
}

// BulkModifierSnapshotBuildResult exports the snapshot builder result type for
// other API modules that need to plug into the same bulk snapshot pipeline.
type BulkModifierSnapshotBuildResult = bulkModifierSnapshotBuildResult

type bulkModifierScanner struct {
	dynamicClient      dynamic.Interface
	preferredResources []*metav1.APIResourceList
}

type bulkScanScope struct {
	namespaces        []string
	namespaceAll      bool
	labelSelector     labels.Selector
	labelSelectorText string
	scopedFilters     bool
	selection         *dapisv1.RestoreResourceSelectionPolicy
}

type bulkScannedResource struct {
	GroupResource string
	Namespace     string
	Name          string
	Object        map[string]any
}

type bulkModifierMatch struct {
	GroupResource string
	Namespace     string
	ResourceName  string
	Path          string
}

func buildBulkModifierSnapshot(
	ctx context.Context,
	spec *dapisv1.DisasterInstanceSpec,
	restConfig *rest.Config,
) (*bulkModifierSnapshotBuildResult, error) {
	result := &bulkModifierSnapshotBuildResult{}
	if spec == nil || spec.RestorePolicy == nil {
		return result, nil
	}

	normalizedActions, err := normalizeBulkModifierActions(spec.RestorePolicy.BulkModifierActions)
	if err != nil {
		return nil, err
	}
	result.Actions = cloneBulkModifierActions(normalizedActions)
	if len(enabledBulkModifierActions(normalizedActions)) == 0 {
		return result, nil
	}
	if restConfig == nil {
		return nil, fmt.Errorf("%s: rest config is required for bulk modifier snapshot generation", modifierRuleRejectedCode)
	}

	scanner, err := newBulkModifierScanner(restConfig)
	if err != nil {
		return nil, fmt.Errorf("%s: initialize bulk modifier scanner failed: %v", modifierRuleRejectedCode, err)
	}
	resources, err := scanner.listScopedResources(ctx, spec)
	if err != nil {
		return nil, err
	}

	return buildBulkModifierSnapshotFromResources(spec, normalizedActions, resources)
}

func buildBulkModifierSnapshotFromResources(
	spec *dapisv1.DisasterInstanceSpec,
	normalizedActions []dapisv1.BulkModifierAction,
	resources []bulkScannedResource,
) (*bulkModifierSnapshotBuildResult, error) {
	result := &bulkModifierSnapshotBuildResult{
		Actions: cloneBulkModifierActions(normalizedActions),
	}
	if spec == nil || spec.RestorePolicy == nil {
		return result, nil
	}

	effectiveActions := enabledBulkModifierActions(normalizedActions)
	if len(effectiveActions) == 0 {
		return result, nil
	}

	generated := make([]dapisv1.RestoreModifierRule, 0)
	for actionIdx := range effectiveActions {
		action := effectiveActions[actionIdx]
		actionMatches := collectBulkModifierMatches(resources, action)
		if len(actionMatches) == 0 {
			return nil, fmt.Errorf(
				"%s: action=%s matched zero resources",
				modifierRuleRejectedCode,
				bulkModifierActionID(action, actionIdx),
			)
		}
		actionRules, err := buildRulesForBulkAction(action, actionIdx, actionMatches)
		if err != nil {
			return nil, err
		}
		generated = append(generated, actionRules...)
	}
	if err := validateBulkGeneratedRuleConflicts(generated); err != nil {
		return nil, err
	}

	snapshot := append([]dapisv1.RestoreModifierRule{}, generated...)
	snapshot = append(snapshot, cloneRestoreModifierRules(spec.RestorePolicy.ModifierRules)...)
	hash, err := hashRestoreModifierRules(snapshot)
	if err != nil {
		return nil, err
	}

	result.ModifierRuleSnapshot = snapshot
	result.ModifierRuleSnapshotHash = hash
	return result, nil
}

func normalizeBulkModifierActions(actions []dapisv1.BulkModifierAction) ([]dapisv1.BulkModifierAction, error) {
	if len(actions) == 0 {
		return nil, nil
	}
	out := make([]dapisv1.BulkModifierAction, 0, len(actions))
	for idx := range actions {
		action := actions[idx]
		normalized, err := normalizeBulkModifierAction(action, idx)
		if err != nil {
			return nil, err
		}
		out = append(out, normalized)
	}
	return out, nil
}

func normalizeBulkModifierAction(action dapisv1.BulkModifierAction, idx int) (dapisv1.BulkModifierAction, error) {
	normalized := action
	if normalized.Enabled == nil {
		normalized.Enabled = bulkBoolPtr(true)
	}

	applyTo, err := normalizeBulkModifierApplyTargets(normalized.ApplyTo, bulkModifierActionID(normalized, idx))
	if err != nil {
		return dapisv1.BulkModifierAction{}, err
	}
	normalized.ApplyTo = applyTo

	switch strings.TrimSpace(string(normalized.Action)) {
	case string(dapisv1.BulkModifierActionReplaceExactValue):
		normalized.Action = dapisv1.BulkModifierActionReplaceExactValue
		normalized.DirectionPolicy = normalizeRestoreDirectionPolicy(normalized.DirectionPolicy)
		normalized.SourceValue = strings.TrimSpace(normalized.SourceValue)
		normalized.TargetValue = strings.TrimSpace(normalized.TargetValue)
		if normalized.SourceValue == "" {
			return dapisv1.BulkModifierAction{}, fmt.Errorf("%s: action=%s sourceValue is required", modifierRuleRejectedCode, bulkModifierActionID(normalized, idx))
		}
		if normalized.TargetValue == "" {
			return dapisv1.BulkModifierAction{}, fmt.Errorf("%s: action=%s targetValue is required", modifierRuleRejectedCode, bulkModifierActionID(normalized, idx))
		}
	case string(dapisv1.BulkModifierActionRemoveKey):
		normalized.Action = dapisv1.BulkModifierActionRemoveKey
		normalized.Key = strings.TrimSpace(normalized.Key)
		if normalized.Key == "" {
			return dapisv1.BulkModifierAction{}, fmt.Errorf("%s: action=%s key is required", modifierRuleRejectedCode, bulkModifierActionID(normalized, idx))
		}
		if strings.TrimSpace(string(normalized.DirectionPolicy)) == "" {
			normalized.DirectionPolicy = dapisv1.RestoreModifierDirectionPolicyForwardOnly
		} else {
			normalized.DirectionPolicy = normalizeRestoreDirectionPolicy(normalized.DirectionPolicy)
		}
	default:
		return dapisv1.BulkModifierAction{}, fmt.Errorf(
			"%s: action=%s unsupported action=%s",
			modifierRuleRejectedCode,
			bulkModifierActionID(normalized, idx),
			normalized.Action,
		)
	}
	return normalized, nil
}

func normalizeBulkModifierApplyTargets(
	applyTo []dapisv1.RestoreModifierApplyTarget,
	actionID string,
) ([]dapisv1.RestoreModifierApplyTarget, error) {
	if len(applyTo) == 0 {
		return []dapisv1.RestoreModifierApplyTarget{dapisv1.RestoreModifierApplyResourceSync}, nil
	}
	seen := make(map[dapisv1.RestoreModifierApplyTarget]struct{}, len(applyTo))
	out := make([]dapisv1.RestoreModifierApplyTarget, 0, len(applyTo))
	for _, target := range applyTo {
		switch target {
		case dapisv1.RestoreModifierApplyResourceSync, dapisv1.RestoreModifierApplyDrill:
		case dapisv1.RestoreModifierApplyDataSync:
			return nil, fmt.Errorf("%s: action=%s applyTo=dataSync is not supported", modifierRuleRejectedCode, actionID)
		default:
			return nil, fmt.Errorf("%s: action=%s unsupported applyTo=%s", modifierRuleRejectedCode, actionID, target)
		}
		if _, exists := seen[target]; exists {
			continue
		}
		seen[target] = struct{}{}
		out = append(out, target)
	}
	return out, nil
}

func enabledBulkModifierActions(actions []dapisv1.BulkModifierAction) []dapisv1.BulkModifierAction {
	if len(actions) == 0 {
		return nil
	}
	out := make([]dapisv1.BulkModifierAction, 0, len(actions))
	for idx := range actions {
		action := actions[idx]
		if action.Enabled != nil && !*action.Enabled {
			continue
		}
		copied := action.DeepCopy()
		if copied == nil {
			continue
		}
		out = append(out, *copied)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func newBulkModifierScanner(restConfig *rest.Config) (*bulkModifierScanner, error) {
	dyn, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	disco, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	preferred, err := disco.ServerPreferredResources()
	if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
		return nil, err
	}
	sort.SliceStable(preferred, func(i, j int) bool {
		return preferred[i].GroupVersion < preferred[j].GroupVersion
	})
	return &bulkModifierScanner{
		dynamicClient:      dyn,
		preferredResources: preferred,
	}, nil
}

func (s *bulkModifierScanner) listScopedResources(
	ctx context.Context,
	spec *dapisv1.DisasterInstanceSpec,
) ([]bulkScannedResource, error) {
	if spec == nil {
		return nil, nil
	}
	scope, err := resolveBulkScanScope(spec)
	if err != nil {
		return nil, err
	}
	listOptions := metav1.ListOptions{LabelSelector: scope.labelSelectorText}
	collected := make([]bulkScannedResource, 0)

	for _, resourceList := range s.preferredResources {
		gv, err := schema.ParseGroupVersion(resourceList.GroupVersion)
		if err != nil {
			continue
		}
		for _, apiResource := range resourceList.APIResources {
			if !bulkAPIResourceListable(apiResource) {
				continue
			}
			gvr := gv.WithResource(apiResource.Name)
			groupResource := formatGroupResource(gvr.Group, gvr.Resource)
			if !scope.allowsResource(apiResource.Name, groupResource, apiResource.Namespaced) {
				continue
			}
			dynResource := s.dynamicClient.Resource(gvr)

			if apiResource.Namespaced {
				if !scope.namespaceAll && len(scope.namespaces) == 0 {
					continue
				}
				namespaces := scope.namespaces
				if scope.namespaceAll {
					namespaces = []string{metav1.NamespaceAll}
				}
				for _, namespace := range namespaces {
					items, err := dynResource.Namespace(namespace).List(ctx, listOptions)
					if err != nil {
						return nil, fmt.Errorf(
							"%s: list groupResource=%s namespace=%s failed: %v",
							modifierRuleRejectedCode,
							groupResource,
							namespace,
							err,
						)
					}
					collected = append(collected, filterBulkScannedResources(groupResource, items.Items, scope.labelSelector)...)
				}
				continue
			}

			items, err := dynResource.List(ctx, listOptions)
			if err != nil {
				return nil, fmt.Errorf(
					"%s: list groupResource=%s failed: %v",
					modifierRuleRejectedCode,
					groupResource,
					err,
				)
			}
			collected = append(collected, filterBulkScannedResources(groupResource, items.Items, scope.labelSelector)...)
		}
	}

	sort.SliceStable(collected, func(i, j int) bool {
		if collected[i].GroupResource != collected[j].GroupResource {
			return collected[i].GroupResource < collected[j].GroupResource
		}
		if collected[i].Namespace != collected[j].Namespace {
			return collected[i].Namespace < collected[j].Namespace
		}
		return collected[i].Name < collected[j].Name
	})
	return collected, nil
}

func resolveBulkScanScope(spec *dapisv1.DisasterInstanceSpec) (bulkScanScope, error) {
	scope := bulkScanScope{
		namespaceAll:  true,
		labelSelector: labels.Everything(),
	}
	if spec == nil {
		return scope, nil
	}

	selection := (*dapisv1.RestoreResourceSelectionPolicy)(nil)
	if spec.RestorePolicy != nil {
		selection = spec.RestorePolicy.ResourceSelection
	}
	scope.selection = selection
	scope.scopedFilters = hasRestoreScopedResourceFilters(selection)

	namespaces := trimRestoreResourceFilterValues(spec.Namespaces)
	includedNamespaces := trimRestoreResourceFilterValues(nil)
	excludedNamespaces := trimRestoreResourceFilterValues(nil)
	if selection != nil {
		includedNamespaces = trimRestoreResourceFilterValues(selection.IncludedNamespaces)
		excludedNamespaces = trimRestoreResourceFilterValues(selection.ExcludedNamespaces)
	}

	switch {
	case len(namespaces) > 0 && len(includedNamespaces) > 0:
		namespaces = intersectStringSlices(namespaces, includedNamespaces)
		scope.namespaceAll = false
	case len(namespaces) == 0 && len(includedNamespaces) > 0:
		namespaces = includedNamespaces
		scope.namespaceAll = false
	case len(namespaces) > 0:
		scope.namespaceAll = false
	}

	if len(excludedNamespaces) > 0 {
		namespaces = subtractStringSlices(namespaces, excludedNamespaces)
		if !scope.namespaceAll {
			scope.namespaces = namespaces
		}
	} else if !scope.namespaceAll {
		scope.namespaces = namespaces
	}

	if selection != nil && selection.LabelSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(selection.LabelSelector)
		if err != nil {
			return bulkScanScope{}, fmt.Errorf("%s: invalid resourceSelection.labelSelector: %v", modifierRuleRejectedCode, err)
		}
		scope.labelSelector = selector
		if !selector.Empty() {
			scope.labelSelectorText = selector.String()
		}
		return scope, nil
	}
	if spec.LabelSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(spec.LabelSelector)
		if err != nil {
			return bulkScanScope{}, fmt.Errorf("%s: invalid spec.labelSelector: %v", modifierRuleRejectedCode, err)
		}
		scope.labelSelector = selector
		if !selector.Empty() {
			scope.labelSelectorText = selector.String()
		}
	}
	return scope, nil
}

func (s bulkScanScope) allowsResource(resourceName string, groupResource string, namespaced bool) bool {
	resourceNames := normalizeBulkResourceIdentifiers(resourceName, groupResource)
	if len(resourceNames) == 0 {
		return false
	}
	if s.selection == nil {
		return namespaced
	}

	if namespaced {
		if s.scopedFilters {
			return resourceAllowedByLists(
				resourceNames,
				trimRestoreResourceFilterValues(s.selection.IncludedNamespaceScopedResources),
				trimRestoreResourceFilterValues(s.selection.ExcludedNamespaceScopedResources),
				true,
			)
		}
		return resourceAllowedByLists(
			resourceNames,
			trimRestoreResourceFilterValues(s.selection.IncludedResources),
			trimRestoreResourceFilterValues(s.selection.ExcludedResources),
			true,
		)
	}

	if s.scopedFilters {
		included := trimRestoreResourceFilterValues(s.selection.IncludedClusterScopedResources)
		excluded := trimRestoreResourceFilterValues(s.selection.ExcludedClusterScopedResources)
		if len(excluded) == 1 && excluded[0] == "*" {
			return false
		}
		if len(included) == 0 && len(excluded) == 0 {
			return false
		}
		return resourceAllowedByLists(resourceNames, included, excluded, false)
	}

	if s.selection.IncludeClusterResources == nil || !*s.selection.IncludeClusterResources {
		return false
	}
	return resourceAllowedByLists(
		resourceNames,
		trimRestoreResourceFilterValues(s.selection.IncludedResources),
		trimRestoreResourceFilterValues(s.selection.ExcludedResources),
		false,
	)
}

func resourceAllowedByLists(resourceNames []string, include []string, exclude []string, defaultAllow bool) bool {
	if containsString(exclude, "*") {
		return false
	}
	if containsAnyString(exclude, resourceNames) {
		return false
	}
	if len(include) == 0 {
		return defaultAllow
	}
	if containsString(include, "*") {
		return true
	}
	return containsAnyString(include, resourceNames)
}

func bulkAPIResourceListable(resource metav1.APIResource) bool {
	if strings.Contains(resource.Name, "/") {
		return false
	}
	for _, verb := range resource.Verbs {
		if verb == "list" {
			return true
		}
	}
	return false
}

func filterBulkScannedResources(
	groupResource string,
	items []unstructured.Unstructured,
	selector labels.Selector,
) []bulkScannedResource {
	out := make([]bulkScannedResource, 0, len(items))
	for _, item := range items {
		if selector != nil && !selector.Empty() && !selector.Matches(labels.Set(item.GetLabels())) {
			continue
		}
		out = append(out, bulkScannedResource{
			GroupResource: groupResource,
			Namespace:     strings.TrimSpace(item.GetNamespace()),
			Name:          strings.TrimSpace(item.GetName()),
			Object:        item.DeepCopy().Object,
		})
	}
	return out
}

func collectBulkModifierMatches(
	resources []bulkScannedResource,
	action dapisv1.BulkModifierAction,
) []bulkModifierMatch {
	matches := make([]bulkModifierMatch, 0)
	for _, resource := range resources {
		switch action.Action {
		case dapisv1.BulkModifierActionReplaceExactValue:
			matches = append(matches, collectReplaceExactValueMatches(resource, action.SourceValue)...)
		case dapisv1.BulkModifierActionRemoveKey:
			matches = append(matches, collectRemoveKeyMatches(resource, action.Key)...)
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].GroupResource != matches[j].GroupResource {
			return matches[i].GroupResource < matches[j].GroupResource
		}
		if matches[i].Namespace != matches[j].Namespace {
			return matches[i].Namespace < matches[j].Namespace
		}
		if matches[i].ResourceName != matches[j].ResourceName {
			return matches[i].ResourceName < matches[j].ResourceName
		}
		return matches[i].Path < matches[j].Path
	})
	return matches
}

func collectReplaceExactValueMatches(resource bulkScannedResource, sourceValue string) []bulkModifierMatch {
	matches := make([]bulkModifierMatch, 0)
	var walk func(node any, path string)
	walk = func(node any, path string) {
		switch typed := node.(type) {
		case map[string]any:
			keys := sortedMapKeys(typed)
			for _, key := range keys {
				childPath := path + "/" + escapeJSONPointerToken(key)
				walk(typed[key], childPath)
			}
		case []any:
			for idx, item := range typed {
				childPath := path + "/" + strconv.Itoa(idx)
				walk(item, childPath)
			}
		case string:
			if typed == sourceValue {
				matches = append(matches, bulkModifierMatch{
					GroupResource: resource.GroupResource,
					Namespace:     resource.Namespace,
					ResourceName:  resource.Name,
					Path:          path,
				})
			}
		}
	}
	walk(resource.Object, "")
	return matches
}

func collectRemoveKeyMatches(resource bulkScannedResource, targetKey string) []bulkModifierMatch {
	matches := make([]bulkModifierMatch, 0)
	var walk func(node any, path string)
	walk = func(node any, path string) {
		switch typed := node.(type) {
		case map[string]any:
			keys := sortedMapKeys(typed)
			for _, key := range keys {
				childPath := path + "/" + escapeJSONPointerToken(key)
				if key == targetKey {
					matches = append(matches, bulkModifierMatch{
						GroupResource: resource.GroupResource,
						Namespace:     resource.Namespace,
						ResourceName:  resource.Name,
						Path:          childPath,
					})
				}
				walk(typed[key], childPath)
			}
		case []any:
			for idx, item := range typed {
				childPath := path + "/" + strconv.Itoa(idx)
				walk(item, childPath)
			}
		}
	}
	walk(resource.Object, "")
	return matches
}

func buildRulesForBulkAction(
	action dapisv1.BulkModifierAction,
	actionIdx int,
	matches []bulkModifierMatch,
) ([]dapisv1.RestoreModifierRule, error) {
	rules := make([]dapisv1.RestoreModifierRule, 0, len(matches))
	actionID := bulkModifierActionID(action, actionIdx)
	for idx := range matches {
		match := matches[idx]
		conditions := dapisv1.Conditions{
			GroupResource:     match.GroupResource,
			ResourceNameRegex: exactResourceNameRegex(match.ResourceName),
		}
		if match.Namespace != "" {
			conditions.Namespaces = []string{match.Namespace}
		}

		ruleID := fmt.Sprintf("bulk-%s-%04d", actionID, idx+1)
		switch action.Action {
		case dapisv1.BulkModifierActionReplaceExactValue:
			rules = append(rules, dapisv1.RestoreModifierRule{
				ID:              ruleID,
				Mode:            dapisv1.RestoreModifierModeReversible,
				ApplyTo:         append([]dapisv1.RestoreModifierApplyTarget{}, action.ApplyTo...),
				Priority:        bulkGeneratedRulePriority,
				Conditions:      conditions,
				DirectionPolicy: normalizeRestoreDirectionPolicy(action.DirectionPolicy),
				OnConflict:      dapisv1.RestoreModifierConflictPolicyFail,
				Pair: &dapisv1.RestoreModifierPair{
					Path:        match.Path,
					SourceValue: action.SourceValue,
					TargetValue: action.TargetValue,
				},
			})
		case dapisv1.BulkModifierActionRemoveKey:
			rules = append(rules, dapisv1.RestoreModifierRule{
				ID:              ruleID,
				Mode:            dapisv1.RestoreModifierModeVeleroNative,
				ApplyTo:         append([]dapisv1.RestoreModifierApplyTarget{}, action.ApplyTo...),
				Priority:        bulkGeneratedRulePriority,
				Conditions:      conditions,
				DirectionPolicy: normalizeRestoreDirectionPolicy(action.DirectionPolicy),
				OnConflict:      dapisv1.RestoreModifierConflictPolicyFail,
				VeleroRule: &dapisv1.RestoreModifierVeleroRule{
					Patches: []dapisv1.JSONPatch{{
						Operation: "remove",
						Path:      match.Path,
					}},
				},
			})
		default:
			return nil, fmt.Errorf("%s: action=%s unsupported action=%s", modifierRuleRejectedCode, actionID, action.Action)
		}
	}
	return rules, nil
}

func validateBulkGeneratedRuleConflicts(rules []dapisv1.RestoreModifierRule) error {
	if len(rules) == 0 {
		return nil
	}
	seen := make(map[string]string, len(rules))
	owners := make(map[string]string, len(rules))
	for idx := range rules {
		key, value, err := bulkGeneratedRuleConflictSignature(rules[idx])
		if err != nil {
			return err
		}
		ruleID := normalizedRestoreModifierRuleID(rules[idx], idx)
		if existingValue, exists := seen[key]; exists {
			if existingValue != value {
				return fmt.Errorf(
					"%s: conflict key=%s existingRule=%s currentRule=%s",
					modifierRuleRejectedCode,
					key,
					owners[key],
					ruleID,
				)
			}
			continue
		}
		seen[key] = value
		owners[key] = ruleID
	}
	return nil
}

func bulkGeneratedRuleConflictSignature(rule dapisv1.RestoreModifierRule) (string, string, error) {
	switch normalizeRestoreModifierMode(rule.Mode) {
	case dapisv1.RestoreModifierModeReversible:
		if rule.Pair == nil {
			return "", "", fmt.Errorf("%s: bulk reversible rule missing pair", modifierRuleRejectedCode)
		}
		key := fmt.Sprintf(
			"%s|%s|%s|%s",
			strings.TrimSpace(rule.Conditions.GroupResource),
			strings.Join(rule.Conditions.Namespaces, ","),
			strings.TrimSpace(rule.Conditions.ResourceNameRegex),
			strings.TrimSpace(rule.Pair.Path),
		)
		value := fmt.Sprintf(
			"%s|%s|%s",
			strings.TrimSpace(rule.Pair.SourceValue),
			strings.TrimSpace(rule.Pair.TargetValue),
			normalizeRestoreDirectionPolicy(rule.DirectionPolicy),
		)
		return key, value, nil
	case dapisv1.RestoreModifierModeVeleroNative:
		if rule.VeleroRule == nil || len(rule.VeleroRule.Patches) != 1 {
			return "", "", fmt.Errorf("%s: bulk veleroNative rule must contain exactly one patch", modifierRuleRejectedCode)
		}
		patch := rule.VeleroRule.Patches[0]
		key := fmt.Sprintf(
			"%s|%s|%s|%s",
			strings.TrimSpace(rule.Conditions.GroupResource),
			strings.Join(rule.Conditions.Namespaces, ","),
			strings.TrimSpace(rule.Conditions.ResourceNameRegex),
			strings.TrimSpace(patch.Path),
		)
		value := fmt.Sprintf(
			"%s|%s|%s",
			strings.ToLower(strings.TrimSpace(patch.Operation)),
			strings.TrimSpace(patch.Value),
			normalizeRestoreDirectionPolicy(rule.DirectionPolicy),
		)
		return key, value, nil
	default:
		return "", "", fmt.Errorf("%s: unsupported bulk rule mode=%s", modifierRuleRejectedCode, rule.Mode)
	}
}

func hashRestoreModifierRules(rules []dapisv1.RestoreModifierRule) (string, error) {
	payload, err := json.Marshal(rules)
	if err != nil {
		return "", fmt.Errorf("%s: marshal modifierRuleSnapshot failed: %v", modifierRuleRejectedCode, err)
	}
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func bulkModifierActionID(action dapisv1.BulkModifierAction, idx int) string {
	if id := strings.TrimSpace(action.ID); id != "" {
		return id
	}
	return fmt.Sprintf("action-%04d", idx+1)
}

func exactResourceNameRegex(name string) string {
	return "^" + regexp.QuoteMeta(strings.TrimSpace(name)) + "$"
}

func escapeJSONPointerToken(token string) string {
	token = strings.ReplaceAll(token, "~", "~0")
	token = strings.ReplaceAll(token, "/", "~1")
	return token
}

func formatGroupResource(group string, resource string) string {
	group = strings.TrimSpace(group)
	resource = strings.TrimSpace(resource)
	if group == "" {
		return resource
	}
	return resource + "." + group
}

func sortedMapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func containsString(values []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}

func containsAnyString(values []string, targets []string) bool {
	for _, target := range targets {
		if containsString(values, target) {
			return true
		}
	}
	return false
}

func normalizeBulkResourceIdentifiers(values ...string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func intersectStringSlices(left []string, right []string) []string {
	if len(left) == 0 || len(right) == 0 {
		return nil
	}
	allowed := make(map[string]struct{}, len(right))
	for _, value := range right {
		allowed[strings.TrimSpace(value)] = struct{}{}
	}
	out := make([]string, 0, len(left))
	for _, value := range left {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := allowed[trimmed]; ok {
			out = append(out, trimmed)
		}
	}
	return out
}

func subtractStringSlices(values []string, excluded []string) []string {
	if len(values) == 0 {
		return nil
	}
	if len(excluded) == 0 {
		return append([]string{}, values...)
	}
	blocked := make(map[string]struct{}, len(excluded))
	for _, value := range excluded {
		blocked[strings.TrimSpace(value)] = struct{}{}
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, blockedValue := blocked[trimmed]; blockedValue {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func bulkBoolPtr(value bool) *bool {
	return &value
}
