package instance

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

type submissionRulePatch struct {
	operation string
	path      string
}

type liveModifierRuleValidator struct {
	dynamicClient dynamic.Interface
	restMapper    meta.RESTMapper
}

func validateRestorePolicyModifierRulesLive(
	ctx context.Context,
	policy *dapisv1.RestorePolicy,
	defaultNamespaces []string,
	restConfig *rest.Config,
) error {
	if policy == nil || len(policy.ModifierRules) == 0 {
		return nil
	}
	if restConfig == nil {
		return fmt.Errorf("%s: rest config is required for live rule validation", modifierRuleRejectedCode)
	}

	validator, err := newLiveModifierRuleValidator(restConfig)
	if err != nil {
		return fmt.Errorf("%s: initialize live rule validator failed: %v", modifierRuleRejectedCode, err)
	}

	for idx := range policy.ModifierRules {
		rule := policy.ModifierRules[idx]
		if !modifierRuleEnabled(rule) {
			continue
		}
		ruleID := normalizedRestoreModifierRuleID(rule, idx)
		patches, err := extractSubmissionRulePatches(ruleID, rule)
		if err != nil {
			return err
		}
		objects, err := validator.listMatchingResources(ctx, rule.Conditions, defaultNamespaces)
		if err != nil {
			return fmt.Errorf(
				"%s: rule=%s groupResource=%s list resources failed: %v",
				modifierRuleRejectedCode,
				ruleID,
				strings.TrimSpace(rule.Conditions.GroupResource),
				err,
			)
		}
		if len(objects) == 0 {
			return fmt.Errorf(
				"%s: rule=%s groupResource=%s matched zero resources",
				modifierRuleRejectedCode,
				ruleID,
				strings.TrimSpace(rule.Conditions.GroupResource),
			)
		}
		for _, obj := range objects {
			resourceRef := obj.GetName()
			if ns := strings.TrimSpace(obj.GetNamespace()); ns != "" {
				resourceRef = ns + "/" + resourceRef
			}
			for _, patch := range patches {
				if err := ensureJSONPointerLocatable(obj.Object, patch.path, patch.operation); err != nil {
					return fmt.Errorf(
						"%s: rule=%s groupResource=%s resource=%s path=%s: %v",
						modifierRuleRejectedCode,
						ruleID,
						strings.TrimSpace(rule.Conditions.GroupResource),
						resourceRef,
						patch.path,
						err,
					)
				}
				if normalizeRestoreModifierMode(rule.Mode) == dapisv1.RestoreModifierModeReversible {
					if err := validateReversiblePairValueCompatibility(patch.path, rule.Pair, obj.Object); err != nil {
						return fmt.Errorf(
							"%s: rule=%s groupResource=%s resource=%s path=%s: %v",
							modifierRuleRejectedCode,
							ruleID,
							strings.TrimSpace(rule.Conditions.GroupResource),
							resourceRef,
							patch.path,
							err,
						)
					}
				}
			}
		}
	}

	return nil
}

func newLiveModifierRuleValidator(restConfig *rest.Config) (*liveModifierRuleValidator, error) {
	dyn, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	disco, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	groupResources, err := restmapper.GetAPIGroupResources(disco)
	if err != nil {
		return nil, err
	}
	return &liveModifierRuleValidator{
		dynamicClient: dyn,
		restMapper:    restmapper.NewDiscoveryRESTMapper(groupResources),
	}, nil
}

func (l *liveModifierRuleValidator) listMatchingResources(
	ctx context.Context,
	conditions dapisv1.Conditions,
	defaultNamespaces []string,
) ([]unstructured.Unstructured, error) {
	groupResource, err := parseRuleGroupResource(conditions.GroupResource)
	if err != nil {
		return nil, err
	}
	gvr, err := l.restMapper.ResourceFor(schema.GroupVersionResource{
		Group:    groupResource.Group,
		Resource: groupResource.Resource,
	})
	if err != nil {
		return nil, err
	}

	gvk, err := l.restMapper.KindFor(gvr)
	if err != nil {
		return nil, err
	}
	mapping, err := l.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}

	labelSelector := labels.Everything()
	labelSelectorText := ""
	if conditions.LabelSelector != nil {
		selector, selectorErr := metav1.LabelSelectorAsSelector(conditions.LabelSelector)
		if selectorErr != nil {
			return nil, selectorErr
		}
		labelSelector = selector
		if !selector.Empty() {
			labelSelectorText = selector.String()
		}
	}

	var nameRegex *regexp.Regexp
	if rawRegex := strings.TrimSpace(conditions.ResourceNameRegex); rawRegex != "" {
		compiledRegex, regexErr := regexp.Compile(rawRegex)
		if regexErr != nil {
			return nil, regexErr
		}
		nameRegex = compiledRegex
	}

	matches := make([]unstructured.Unstructured, 0)
	dynResource := l.dynamicClient.Resource(gvr)
	listOptions := metav1.ListOptions{LabelSelector: labelSelectorText}

	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		namespaces := effectiveRuleNamespaces(conditions.Namespaces, defaultNamespaces)
		if len(namespaces) == 0 {
			namespaces = []string{metav1.NamespaceAll}
		}
		for _, namespace := range namespaces {
			items, listErr := dynResource.Namespace(namespace).List(ctx, listOptions)
			if listErr != nil {
				return nil, listErr
			}
			matches = append(matches, filterMatchedItems(items.Items, labelSelector, nameRegex)...)
		}
		return matches, nil
	}

	items, listErr := dynResource.List(ctx, listOptions)
	if listErr != nil {
		return nil, listErr
	}
	matches = append(matches, filterMatchedItems(items.Items, labelSelector, nameRegex)...)
	return matches, nil
}

func parseRuleGroupResource(raw string) (schema.GroupResource, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return schema.GroupResource{}, fmt.Errorf("conditions.groupResource is required")
	}
	parts := strings.Split(raw, ".")
	resource := strings.TrimSpace(parts[0])
	if resource == "" {
		return schema.GroupResource{}, fmt.Errorf("invalid groupResource=%s", raw)
	}
	group := ""
	if len(parts) > 1 {
		group = strings.TrimSpace(strings.Join(parts[1:], "."))
	}
	return schema.GroupResource{Group: group, Resource: resource}, nil
}

func effectiveRuleNamespaces(ruleNamespaces []string, defaultNamespaces []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(ruleNamespaces)+len(defaultNamespaces))

	appendAll := func(items []string) {
		for _, item := range items {
			ns := strings.TrimSpace(item)
			if ns == "" {
				continue
			}
			if _, ok := seen[ns]; ok {
				continue
			}
			seen[ns] = struct{}{}
			out = append(out, ns)
		}
	}

	appendAll(ruleNamespaces)
	if len(out) > 0 {
		return out
	}
	appendAll(defaultNamespaces)
	return out
}

func filterMatchedItems(
	items []unstructured.Unstructured,
	selector labels.Selector,
	nameRegex *regexp.Regexp,
) []unstructured.Unstructured {
	matches := make([]unstructured.Unstructured, 0, len(items))
	for _, item := range items {
		if selector != nil && !selector.Empty() && !selector.Matches(labels.Set(item.GetLabels())) {
			continue
		}
		if nameRegex != nil && !nameRegex.MatchString(item.GetName()) {
			continue
		}
		matches = append(matches, item)
	}
	return matches
}

func extractSubmissionRulePatches(ruleID string, rule dapisv1.RestoreModifierRule) ([]submissionRulePatch, error) {
	switch normalizeRestoreModifierMode(rule.Mode) {
	case dapisv1.RestoreModifierModeVeleroNative:
		if rule.VeleroRule == nil {
			return nil, fmt.Errorf("%s: rule=%s veleroNative rule missing veleroRule", modifierRuleRejectedCode, ruleID)
		}
		patches := make([]submissionRulePatch, 0, len(rule.VeleroRule.Patches))
		for _, p := range rule.VeleroRule.Patches {
			patches = append(patches, submissionRulePatch{
				operation: strings.TrimSpace(p.Operation),
				path:      strings.TrimSpace(p.Path),
			})
		}
		return patches, nil
	case dapisv1.RestoreModifierModeReversible:
		if rule.Pair == nil {
			return nil, fmt.Errorf(
				"%s: rule=%s reversible rule must use pair canonical form (pair.path, pair.sourceValue, pair.targetValue)",
				modifierRuleRejectedCode,
				ruleID,
			)
		}
		return []submissionRulePatch{{
			operation: "add",
			path:      strings.TrimSpace(rule.Pair.Path),
		}}, nil
	default:
		return nil, fmt.Errorf("%s: rule=%s unsupported mode=%s", modifierRuleRejectedCode, ruleID, rule.Mode)
	}
}

func modifierRuleEnabled(rule dapisv1.RestoreModifierRule) bool {
	if rule.Enabled == nil {
		return true
	}
	return *rule.Enabled
}
