package instance

import (
	"context"
	"reflect"
	"strings"
	"testing"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/rest"
)

func TestNormalizeBulkModifierActions_DefaultsEnabledApplyToAndDirection(t *testing.T) {
	t.Parallel()

	actions := []dapisv1.BulkModifierAction{
		{
			ID:          "replace-ip",
			Action:      dapisv1.BulkModifierActionReplaceExactValue,
			SourceValue: "10.10.0.12",
			TargetValue: "10.20.0.12",
		},
		{
			ID:     "drop-key",
			Action: dapisv1.BulkModifierActionRemoveKey,
			Key:    "site-role",
		},
	}

	got, err := normalizeBulkModifierActions(actions)
	if err != nil {
		t.Fatalf("expected normalization success, got %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 normalized actions, got %d", len(got))
	}

	if got[0].Enabled == nil || !*got[0].Enabled {
		t.Fatalf("expected replaceExactValue enabled=true by default, got %v", got[0].Enabled)
	}
	if !reflect.DeepEqual(got[0].ApplyTo, []dapisv1.RestoreModifierApplyTarget{dapisv1.RestoreModifierApplyResourceSync}) {
		t.Fatalf("expected default applyTo=[resourceSync], got %v", got[0].ApplyTo)
	}
	if got[0].DirectionPolicy != dapisv1.RestoreModifierDirectionPolicyAuto {
		t.Fatalf("expected replaceExactValue directionPolicy=Auto, got %s", got[0].DirectionPolicy)
	}

	if got[1].Enabled == nil || !*got[1].Enabled {
		t.Fatalf("expected removeKey enabled=true by default, got %v", got[1].Enabled)
	}
	if !reflect.DeepEqual(got[1].ApplyTo, []dapisv1.RestoreModifierApplyTarget{dapisv1.RestoreModifierApplyResourceSync}) {
		t.Fatalf("expected default applyTo=[resourceSync], got %v", got[1].ApplyTo)
	}
	if got[1].DirectionPolicy != dapisv1.RestoreModifierDirectionPolicyForwardOnly {
		t.Fatalf("expected removeKey directionPolicy=ForwardOnly, got %s", got[1].DirectionPolicy)
	}
}

func TestNormalizeBulkModifierActions_RejectsDataSyncApplyTo(t *testing.T) {
	t.Parallel()

	_, err := normalizeBulkModifierActions([]dapisv1.BulkModifierAction{{
		ID:          "replace-ip",
		Action:      dapisv1.BulkModifierActionReplaceExactValue,
		ApplyTo:     []dapisv1.RestoreModifierApplyTarget{dapisv1.RestoreModifierApplyDataSync},
		SourceValue: "10.10.0.12",
		TargetValue: "10.20.0.12",
	}})
	if err == nil {
		t.Fatalf("expected dataSync rejection, got nil")
	}
	if !strings.Contains(err.Error(), "applyTo=dataSync is not supported") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNormalizeBulkModifierActions_RewriteImageDefaultsAndValidation(t *testing.T) {
	t.Parallel()

	got, err := normalizeBulkModifierActions([]dapisv1.BulkModifierAction{{
		ID:      "rewrite-primary-registry",
		Action:  dapisv1.BulkModifierActionRewriteImage,
		ApplyTo: []dapisv1.RestoreModifierApplyTarget{dapisv1.RestoreModifierApplyResourceSync, dapisv1.RestoreModifierApplyDrill},
		ImageRewrite: &dapisv1.DynamicImageRewriteConfig{
			SourcePrefix: " 10.11.11.1:5000/ ",
			TargetPrefix: " registry-test.xxx.xxx.com:30088/dr_images/10_11_11_1_5000/ ",
		},
	}})
	if err != nil {
		t.Fatalf("expected rewriteImage normalization success, got %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 normalized action, got %d", len(got))
	}
	action := got[0]
	if action.Action != dapisv1.BulkModifierActionRewriteImage {
		t.Fatalf("expected rewriteImage action, got %s", action.Action)
	}
	if action.DirectionPolicy != dapisv1.RestoreModifierDirectionPolicyAuto {
		t.Fatalf("expected directionPolicy=Auto, got %s", action.DirectionPolicy)
	}
	if action.ImageRewrite == nil {
		t.Fatalf("expected imageRewrite config")
	}
	if action.ImageRewrite.SourcePrefix != "10.11.11.1:5000/" {
		t.Fatalf("unexpected sourcePrefix: %s", action.ImageRewrite.SourcePrefix)
	}
	if action.ImageRewrite.TargetPrefix != "registry-test.xxx.xxx.com:30088/dr_images/10_11_11_1_5000/" {
		t.Fatalf("unexpected targetPrefix: %s", action.ImageRewrite.TargetPrefix)
	}
	if action.ImageRewrite.UnmatchedPolicy != dapisv1.ImageRewriteUnmatchedPolicyKeep {
		t.Fatalf("expected default unmatchedPolicy=Keep, got %s", action.ImageRewrite.UnmatchedPolicy)
	}
	if action.ImageRewrite.DigestPolicy != dapisv1.ImageRewriteDigestPolicyPreserve {
		t.Fatalf("expected default digestPolicy=Preserve, got %s", action.ImageRewrite.DigestPolicy)
	}
}

func TestNormalizeBulkModifierActions_RewriteImageRequiresPrefixes(t *testing.T) {
	t.Parallel()

	_, err := normalizeBulkModifierActions([]dapisv1.BulkModifierAction{{
		ID:     "rewrite-primary-registry",
		Action: dapisv1.BulkModifierActionRewriteImage,
		ImageRewrite: &dapisv1.DynamicImageRewriteConfig{
			TargetPrefix: "registry-test.xxx.xxx.com:30088/dr_images/10_11_11_1_5000/",
		},
	}})
	if err == nil {
		t.Fatalf("expected missing sourcePrefix rejection, got nil")
	}
	if !strings.Contains(err.Error(), "imageRewrite.sourcePrefix is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildBulkModifierSnapshotFromResources_ReplaceExactValueMergesManualRulesAndStableHash(t *testing.T) {
	t.Parallel()

	spec := &dapisv1.DisasterInstanceSpec{
		RestorePolicy: &dapisv1.RestorePolicy{
			ModifierRules: []dapisv1.RestoreModifierRule{{
				ID:   "manual-rule",
				Mode: dapisv1.RestoreModifierModeVeleroNative,
				Conditions: dapisv1.Conditions{
					GroupResource: "deployments.apps",
				},
				VeleroRule: &dapisv1.RestoreModifierVeleroRule{
					Patches: []dapisv1.JSONPatch{{
						Operation: "add",
						Path:      "/metadata/annotations/manual",
						Value:     "manual",
					}},
				},
			}},
		},
	}
	actions, err := normalizeBulkModifierActions([]dapisv1.BulkModifierAction{{
		ID:          "replace-ip",
		Action:      dapisv1.BulkModifierActionReplaceExactValue,
		SourceValue: "10.10.0.12",
		TargetValue: "10.20.0.12",
	}})
	if err != nil {
		t.Fatalf("expected normalization success, got %v", err)
	}

	resources := []bulkScannedResource{{
		GroupResource: "deployments.apps",
		Namespace:     "demo-ns",
		Name:          "demo",
		Object: map[string]any{
			"spec": map[string]any{
				"template": map[string]any{
					"spec": map[string]any{
						"containers": []any{
							map[string]any{
								"env": []any{
									map[string]any{
										"name":  "SERVICE_IP",
										"value": "10.10.0.12",
									},
								},
							},
						},
					},
				},
			},
		},
	}}

	got, err := buildBulkModifierSnapshotFromResources(spec, actions, resources)
	if err != nil {
		t.Fatalf("expected snapshot build success, got %v", err)
	}
	if len(got.ModifierRuleSnapshot) != 2 {
		t.Fatalf("expected 2 snapshot rules, got %d", len(got.ModifierRuleSnapshot))
	}

	bulkRule := got.ModifierRuleSnapshot[0]
	if bulkRule.ID != "bulk-replace-ip-0001" {
		t.Fatalf("expected deterministic rule ID, got %s", bulkRule.ID)
	}
	if bulkRule.Priority != bulkGeneratedRulePriority {
		t.Fatalf("expected generated rule priority=%d, got %d", bulkGeneratedRulePriority, bulkRule.Priority)
	}
	if bulkRule.Pair == nil {
		t.Fatalf("expected reversible pair rule")
	}
	if bulkRule.Pair.Path != "/spec/template/spec/containers/0/env/0/value" {
		t.Fatalf("unexpected replaceExactValue path: %s", bulkRule.Pair.Path)
	}
	if bulkRule.Pair.SourceValue != "10.10.0.12" || bulkRule.Pair.TargetValue != "10.20.0.12" {
		t.Fatalf("unexpected pair values: %#v", bulkRule.Pair)
	}
	if got.ModifierRuleSnapshot[1].ID != "manual-rule" {
		t.Fatalf("expected manual rule appended after generated rules, got %s", got.ModifierRuleSnapshot[1].ID)
	}

	gotAgain, err := buildBulkModifierSnapshotFromResources(spec, cloneBulkModifierActions(actions), resources)
	if err != nil {
		t.Fatalf("expected stable rebuild success, got %v", err)
	}
	if got.ModifierRuleSnapshotHash != gotAgain.ModifierRuleSnapshotHash {
		t.Fatalf("expected stable snapshot hash, got %s vs %s", got.ModifierRuleSnapshotHash, gotAgain.ModifierRuleSnapshotHash)
	}
}

func TestBuildBulkModifierSnapshotFromResources_RewriteImageDoesNotGenerateStaticSnapshot(t *testing.T) {
	t.Parallel()

	spec := &dapisv1.DisasterInstanceSpec{
		RestorePolicy: &dapisv1.RestorePolicy{
			BulkModifierActions: []dapisv1.BulkModifierAction{{
				ID:      "rewrite-primary-registry",
				Action:  dapisv1.BulkModifierActionRewriteImage,
				Enabled: boolPtr(true),
				ImageRewrite: &dapisv1.DynamicImageRewriteConfig{
					SourcePrefix:    "10.11.11.1:5000/",
					TargetPrefix:    "registry-test.xxx.xxx.com:30088/dr_images/10_11_11_1_5000/",
					UnmatchedPolicy: dapisv1.ImageRewriteUnmatchedPolicyKeep,
					DigestPolicy:    dapisv1.ImageRewriteDigestPolicyPreserve,
				},
			}},
		},
	}
	actions, err := normalizeBulkModifierActions(spec.RestorePolicy.BulkModifierActions)
	if err != nil {
		t.Fatalf("expected normalization success, got %v", err)
	}

	got, err := buildBulkModifierSnapshotFromResources(spec, actions, nil)
	if err != nil {
		t.Fatalf("expected rewriteImage to skip static snapshot build, got %v", err)
	}
	if len(got.Actions) != 1 || got.Actions[0].Action != dapisv1.BulkModifierActionRewriteImage {
		t.Fatalf("expected rewriteImage action to be preserved, got %#v", got.Actions)
	}
	if len(got.ModifierRuleSnapshot) != 0 {
		t.Fatalf("expected no static modifierRuleSnapshot for rewriteImage, got %d", len(got.ModifierRuleSnapshot))
	}
	if got.ModifierRuleSnapshotHash != "" {
		t.Fatalf("expected empty snapshot hash for rewriteImage, got %s", got.ModifierRuleSnapshotHash)
	}
}

func TestBuildBulkModifierSnapshotFromResources_MixedRewriteImageAndStaticOnlyExpandsStatic(t *testing.T) {
	t.Parallel()

	spec := &dapisv1.DisasterInstanceSpec{
		RestorePolicy: &dapisv1.RestorePolicy{},
	}
	actions, err := normalizeBulkModifierActions([]dapisv1.BulkModifierAction{
		{
			ID:     "rewrite-primary-registry",
			Action: dapisv1.BulkModifierActionRewriteImage,
			ImageRewrite: &dapisv1.DynamicImageRewriteConfig{
				SourcePrefix: "10.11.11.1:5000/",
				TargetPrefix: "registry-test.xxx.xxx.com:30088/dr_images/10_11_11_1_5000/",
			},
		},
		{
			ID:          "replace-ip",
			Action:      dapisv1.BulkModifierActionReplaceExactValue,
			SourceValue: "10.10.0.12",
			TargetValue: "10.20.0.12",
		},
	})
	if err != nil {
		t.Fatalf("expected normalization success, got %v", err)
	}

	got, err := buildBulkModifierSnapshotFromResources(spec, actions, []bulkScannedResource{{
		GroupResource: "configmaps",
		Namespace:     "demo-ns",
		Name:          "demo",
		Object: map[string]any{
			"data": map[string]any{
				"ip": "10.10.0.12",
			},
		},
	}})
	if err != nil {
		t.Fatalf("expected mixed snapshot build success, got %v", err)
	}
	if len(got.Actions) != 2 {
		t.Fatalf("expected both actions to be preserved, got %d", len(got.Actions))
	}
	if len(got.ModifierRuleSnapshot) != 1 {
		t.Fatalf("expected only static action to generate one rule, got %d", len(got.ModifierRuleSnapshot))
	}
	if got.ModifierRuleSnapshot[0].ID != "bulk-replace-ip-0001" {
		t.Fatalf("unexpected generated rule ID: %s", got.ModifierRuleSnapshot[0].ID)
	}
}

func TestBuildBulkModifierSnapshotFromResources_RemoveKeySetsForwardOnlyPatch(t *testing.T) {
	t.Parallel()

	spec := &dapisv1.DisasterInstanceSpec{
		RestorePolicy: &dapisv1.RestorePolicy{},
	}
	actions, err := normalizeBulkModifierActions([]dapisv1.BulkModifierAction{{
		ID:     "drop-site-role",
		Action: dapisv1.BulkModifierActionRemoveKey,
		Key:    "site-role",
	}})
	if err != nil {
		t.Fatalf("expected normalization success, got %v", err)
	}

	got, err := buildBulkModifierSnapshotFromResources(spec, actions, []bulkScannedResource{{
		GroupResource: "deployments.apps",
		Namespace:     "demo-ns",
		Name:          "demo",
		Object: map[string]any{
			"metadata": map[string]any{
				"annotations": map[string]any{
					"site-role": "primary",
				},
			},
		},
	}})
	if err != nil {
		t.Fatalf("expected snapshot build success, got %v", err)
	}
	if len(got.ModifierRuleSnapshot) != 1 {
		t.Fatalf("expected 1 snapshot rule, got %d", len(got.ModifierRuleSnapshot))
	}
	rule := got.ModifierRuleSnapshot[0]
	if rule.DirectionPolicy != dapisv1.RestoreModifierDirectionPolicyForwardOnly {
		t.Fatalf("expected ForwardOnly direction, got %s", rule.DirectionPolicy)
	}
	if rule.VeleroRule == nil || len(rule.VeleroRule.Patches) != 1 {
		t.Fatalf("expected one veleroNative patch, got %#v", rule.VeleroRule)
	}
	if patch := rule.VeleroRule.Patches[0]; patch.Operation != "remove" || patch.Path != "/metadata/annotations/site-role" {
		t.Fatalf("unexpected removeKey patch: %#v", patch)
	}
}

func TestBuildBulkModifierSnapshotFromResources_ReplaceExactValueSkipsForbiddenStatusPath(t *testing.T) {
	t.Parallel()

	const sourceImage = "10.11.11.1:5000/blueking/bcs-bkcmdb-synchronizer:v1.30.0"
	const targetImage = "registry-test.xxx.xxx.com:30088/dr_images/10_11_11_1_5000/blueking/bcs-bkcmdb-synchronizer:v1.30.0"

	spec := &dapisv1.DisasterInstanceSpec{
		RestorePolicy: &dapisv1.RestorePolicy{},
	}
	actions, err := normalizeBulkModifierActions([]dapisv1.BulkModifierAction{{
		ID:          "replace-bkcmdb-synchronizer-image",
		Action:      dapisv1.BulkModifierActionReplaceExactValue,
		SourceValue: sourceImage,
		TargetValue: targetImage,
		ApplyTo: []dapisv1.RestoreModifierApplyTarget{
			dapisv1.RestoreModifierApplyResourceSync,
			dapisv1.RestoreModifierApplyDrill,
		},
	}})
	if err != nil {
		t.Fatalf("expected normalization success, got %v", err)
	}

	got, err := buildBulkModifierSnapshotFromResources(spec, actions, []bulkScannedResource{
		{
			GroupResource: "deployments.apps",
			Namespace:     "demo-ns",
			Name:          "bkcmdb",
			Object: map[string]any{
				"spec": map[string]any{
					"template": map[string]any{
						"spec": map[string]any{
							"containers": []any{
								map[string]any{
									"name":  "synchronizer",
									"image": sourceImage,
								},
							},
						},
					},
				},
			},
		},
		{
			GroupResource: "pods",
			Namespace:     "demo-ns",
			Name:          "bkcmdb-pod",
			Object: map[string]any{
				"spec": map[string]any{
					"containers": []any{
						map[string]any{
							"name":  "synchronizer",
							"image": sourceImage,
						},
					},
				},
				"status": map[string]any{
					"containerStatuses": []any{
						map[string]any{
							"name":  "synchronizer",
							"image": sourceImage,
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("expected snapshot build success, got %v", err)
	}
	if len(got.ModifierRuleSnapshot) != 2 {
		t.Fatalf("expected 2 snapshot rules for deployment spec image and pod spec image, got %d", len(got.ModifierRuleSnapshot))
	}

	paths := make(map[string]struct{}, len(got.ModifierRuleSnapshot))
	for _, rule := range got.ModifierRuleSnapshot {
		if rule.Pair == nil {
			t.Fatalf("expected reversible pair rule, got %#v", rule)
		}
		if strings.HasPrefix(rule.Pair.Path, "/status/") || rule.Pair.Path == "/status" {
			t.Fatalf("generated forbidden status path: %s", rule.Pair.Path)
		}
		paths[rule.Conditions.GroupResource+"|"+rule.Pair.Path] = struct{}{}
	}
	expectedPaths := []string{
		"deployments.apps|/spec/template/spec/containers/0/image",
		"pods|/spec/containers/0/image",
	}
	for _, path := range expectedPaths {
		if _, ok := paths[path]; !ok {
			t.Fatalf("missing expected generated path %s; got %v", path, paths)
		}
	}
	if _, ok := paths["pods|/status/containerStatuses/0/image"]; ok {
		t.Fatalf("status container image path must be skipped")
	}
}

func TestBuildBulkModifierSnapshotFromResources_ReplaceExactValueOnlyForbiddenPathIsZeroMatch(t *testing.T) {
	t.Parallel()

	spec := &dapisv1.DisasterInstanceSpec{
		RestorePolicy: &dapisv1.RestorePolicy{},
	}
	actions, err := normalizeBulkModifierActions([]dapisv1.BulkModifierAction{{
		ID:          "replace-status-image",
		Action:      dapisv1.BulkModifierActionReplaceExactValue,
		SourceValue: "registry.local/app:v1",
		TargetValue: "registry.dr/app:v1",
	}})
	if err != nil {
		t.Fatalf("expected normalization success, got %v", err)
	}

	_, err = buildBulkModifierSnapshotFromResources(spec, actions, []bulkScannedResource{{
		GroupResource: "pods",
		Namespace:     "demo-ns",
		Name:          "only-status",
		Object: map[string]any{
			"status": map[string]any{
				"containerStatuses": []any{
					map[string]any{
						"image": "registry.local/app:v1",
					},
				},
			},
		},
	}})
	if err == nil {
		t.Fatalf("expected zero-match rejection after filtering forbidden status path")
	}
	if !strings.Contains(err.Error(), "matched zero resources") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildBulkModifierSnapshotFromResources_RemoveKeySkipsForbiddenPaths(t *testing.T) {
	t.Parallel()

	spec := &dapisv1.DisasterInstanceSpec{
		RestorePolicy: &dapisv1.RestorePolicy{},
	}
	actions, err := normalizeBulkModifierActions([]dapisv1.BulkModifierAction{{
		ID:     "drop-site-role",
		Action: dapisv1.BulkModifierActionRemoveKey,
		Key:    "site-role",
	}})
	if err != nil {
		t.Fatalf("expected normalization success, got %v", err)
	}

	got, err := buildBulkModifierSnapshotFromResources(spec, actions, []bulkScannedResource{{
		GroupResource: "configmaps",
		Namespace:     "demo-ns",
		Name:          "demo",
		Object: map[string]any{
			"metadata": map[string]any{
				"annotations": map[string]any{
					"site-role": "primary",
				},
				"finalizers": []any{
					map[string]any{
						"site-role": "forbidden-finalizer-child",
					},
				},
				"ownerReferences": []any{
					map[string]any{
						"site-role": "forbidden-owner-child",
					},
				},
			},
			"status": map[string]any{
				"site-role": "forbidden-status-child",
			},
		},
	}})
	if err != nil {
		t.Fatalf("expected snapshot build success, got %v", err)
	}
	if len(got.ModifierRuleSnapshot) != 1 {
		t.Fatalf("expected only one allowed removeKey rule, got %d", len(got.ModifierRuleSnapshot))
	}
	rule := got.ModifierRuleSnapshot[0]
	if rule.VeleroRule == nil || len(rule.VeleroRule.Patches) != 1 {
		t.Fatalf("expected one veleroNative patch, got %#v", rule.VeleroRule)
	}
	if gotPath := rule.VeleroRule.Patches[0].Path; gotPath != "/metadata/annotations/site-role" {
		t.Fatalf("expected only annotation key to be removed, got %s", gotPath)
	}
}

func TestBuildBulkModifierSnapshotFromResources_RejectsZeroMatch(t *testing.T) {
	t.Parallel()

	spec := &dapisv1.DisasterInstanceSpec{
		RestorePolicy: &dapisv1.RestorePolicy{},
	}
	actions, err := normalizeBulkModifierActions([]dapisv1.BulkModifierAction{{
		ID:          "replace-ip",
		Action:      dapisv1.BulkModifierActionReplaceExactValue,
		SourceValue: "10.10.0.12",
		TargetValue: "10.20.0.12",
	}})
	if err != nil {
		t.Fatalf("expected normalization success, got %v", err)
	}

	_, err = buildBulkModifierSnapshotFromResources(spec, actions, []bulkScannedResource{{
		GroupResource: "deployments.apps",
		Namespace:     "demo-ns",
		Name:          "demo",
		Object: map[string]any{
			"metadata": map[string]any{
				"name": "demo",
			},
		},
	}})
	if err == nil {
		t.Fatalf("expected zero-match rejection, got nil")
	}
	if !strings.Contains(err.Error(), "matched zero resources") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveBulkScanScope_ExplicitNamespacesAndIncludedNamespacesDisableNamespaceAll(t *testing.T) {
	t.Parallel()

	spec := &dapisv1.DisasterInstanceSpec{
		Namespaces: []string{"team-a", "team-b"},
		RestorePolicy: &dapisv1.RestorePolicy{
			ResourceSelection: &dapisv1.RestoreResourceSelectionPolicy{
				IncludedNamespaces: []string{"team-a"},
			},
		},
	}

	scope, err := resolveBulkScanScope(spec)
	if err != nil {
		t.Fatalf("expected scope resolution success, got %v", err)
	}
	if scope.namespaceAll {
		t.Fatalf("expected namespaceAll=false when namespaces are intersected")
	}
	if !reflect.DeepEqual(scope.namespaces, []string{"team-a"}) {
		t.Fatalf("expected narrowed namespaces [team-a], got %v", scope.namespaces)
	}
}

func TestBulkModifierScanner_ListScopedResourcesSupportsFullyQualifiedNamespaceScopedResources(t *testing.T) {
	t.Parallel()

	deploymentGVR := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	ingressGVR := schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"}
	configMapGVR := schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}

	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		map[schema.GroupVersionResource]string{
			deploymentGVR: "DeploymentList",
			ingressGVR:    "IngressList",
			configMapGVR:  "ConfigMapList",
		},
		&unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]any{
				"name":      "deploy-hit",
				"namespace": "demo-ns",
				"labels": map[string]any{
					"bulk-scan": "hit",
				},
			},
		}},
		&unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "networking.k8s.io/v1",
			"kind":       "Ingress",
			"metadata": map[string]any{
				"name":      "ing-hit",
				"namespace": "demo-ns",
				"labels": map[string]any{
					"bulk-scan": "hit",
				},
			},
		}},
		&unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":      "cm-ignore",
				"namespace": "demo-ns",
				"labels": map[string]any{
					"bulk-scan": "hit",
				},
			},
		}},
	)

	scanner := &bulkModifierScanner{
		dynamicClient: dyn,
		preferredResources: []*metav1.APIResourceList{
			{
				GroupVersion: "apps/v1",
				APIResources: []metav1.APIResource{{
					Name:       "deployments",
					Namespaced: true,
					Verbs:      metav1.Verbs{"list"},
				}},
			},
			{
				GroupVersion: "networking.k8s.io/v1",
				APIResources: []metav1.APIResource{{
					Name:       "ingresses",
					Namespaced: true,
					Verbs:      metav1.Verbs{"list"},
				}},
			},
			{
				GroupVersion: "v1",
				APIResources: []metav1.APIResource{{
					Name:       "configmaps",
					Namespaced: true,
					Verbs:      metav1.Verbs{"list"},
				}},
			},
		},
	}

	spec := &dapisv1.DisasterInstanceSpec{
		Namespaces: []string{"demo-ns"},
		RestorePolicy: &dapisv1.RestorePolicy{
			ResourceSelection: &dapisv1.RestoreResourceSelectionPolicy{
				IncludedNamespaces:               []string{"demo-ns"},
				IncludedNamespaceScopedResources: []string{"deployments.apps", "ingresses.networking.k8s.io"},
			},
		},
		LabelSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"bulk-scan": "hit",
			},
		},
	}

	resources, err := scanner.listScopedResources(context.Background(), spec)
	if err != nil {
		t.Fatalf("expected scoped scan success, got %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("expected 2 scoped resources, got %d: %#v", len(resources), resources)
	}

	gotGroupResources := []string{resources[0].GroupResource, resources[1].GroupResource}
	if !reflect.DeepEqual(gotGroupResources, []string{"deployments.apps", "ingresses.networking.k8s.io"}) {
		t.Fatalf("unexpected scoped resources: %v", gotGroupResources)
	}
}

func TestBuildBulkModifierSnapshotFromResources_FullyQualifiedNamespaceScopedSelectionExpandsAllMatches(t *testing.T) {
	t.Parallel()

	deploymentGVR := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	ingressGVR := schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"}
	configMapGVR := schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}
	serviceGVR := schema.GroupVersionResource{Version: "v1", Resource: "services"}

	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		map[schema.GroupVersionResource]string{
			deploymentGVR: "DeploymentList",
			ingressGVR:    "IngressList",
			configMapGVR:  "ConfigMapList",
			serviceGVR:    "ServiceList",
		},
		&unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]any{
				"name":      "deploy-hit",
				"namespace": "demo-ns",
				"labels": map[string]any{
					"bulk-scan": "hit",
				},
				"annotations": map[string]any{
					"bulk.demo/ip": "10.30.23.12",
				},
			},
			"spec": map[string]any{
				"template": map[string]any{
					"metadata": map[string]any{
						"annotations": map[string]any{
							"bulk.demo/ip": "10.30.23.12",
						},
					},
					"spec": map[string]any{
						"containers": []any{
							map[string]any{
								"name": "web",
								"env": []any{
									map[string]any{
										"name":  "BACKEND_IP",
										"value": "10.30.23.12",
									},
								},
							},
						},
					},
				},
			},
		}},
		&unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "networking.k8s.io/v1",
			"kind":       "Ingress",
			"metadata": map[string]any{
				"name":      "ing-hit",
				"namespace": "demo-ns",
				"labels": map[string]any{
					"bulk-scan": "hit",
				},
				"annotations": map[string]any{
					"nginx.ingress.kubernetes.io/upstream-vhost": "10.30.23.12",
				},
			},
		}},
		&unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":      "cm-hit",
				"namespace": "demo-ns",
				"labels": map[string]any{
					"bulk-scan": "hit",
				},
				"annotations": map[string]any{
					"bulk.demo/ip": "10.30.23.12",
				},
			},
			"data": map[string]any{
				"DB_HOST": "10.30.23.12",
			},
		}},
		&unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]any{
				"name":      "svc-hit",
				"namespace": "demo-ns",
				"labels": map[string]any{
					"bulk-scan": "hit",
				},
				"annotations": map[string]any{
					"bulk.demo/upstream-ip": "10.30.23.12",
				},
			},
		}},
	)

	scanner := &bulkModifierScanner{
		dynamicClient: dyn,
		preferredResources: []*metav1.APIResourceList{
			{
				GroupVersion: "apps/v1",
				APIResources: []metav1.APIResource{{
					Name:       "deployments",
					Namespaced: true,
					Verbs:      metav1.Verbs{"list"},
				}},
			},
			{
				GroupVersion: "networking.k8s.io/v1",
				APIResources: []metav1.APIResource{{
					Name:       "ingresses",
					Namespaced: true,
					Verbs:      metav1.Verbs{"list"},
				}},
			},
			{
				GroupVersion: "v1",
				APIResources: []metav1.APIResource{
					{
						Name:       "configmaps",
						Namespaced: true,
						Verbs:      metav1.Verbs{"list"},
					},
					{
						Name:       "services",
						Namespaced: true,
						Verbs:      metav1.Verbs{"list"},
					},
				},
			},
		},
	}

	spec := &dapisv1.DisasterInstanceSpec{
		Namespaces: []string{"demo-ns"},
		LabelSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"bulk-scan": "hit",
			},
		},
		RestorePolicy: &dapisv1.RestorePolicy{
			ResourceSelection: &dapisv1.RestoreResourceSelectionPolicy{
				IncludedNamespaces: []string{"demo-ns"},
				IncludedNamespaceScopedResources: []string{
					"deployments.apps",
					"configmaps",
					"services",
					"ingresses.networking.k8s.io",
				},
			},
		},
	}
	actions, err := normalizeBulkModifierActions([]dapisv1.BulkModifierAction{{
		ID:          "replace-ip",
		Action:      dapisv1.BulkModifierActionReplaceExactValue,
		SourceValue: "10.30.23.12",
		TargetValue: "10.40.23.22",
	}})
	if err != nil {
		t.Fatalf("expected normalization success, got %v", err)
	}

	resources, err := scanner.listScopedResources(context.Background(), spec)
	if err != nil {
		t.Fatalf("expected scoped scan success, got %v", err)
	}
	got, err := buildBulkModifierSnapshotFromResources(spec, actions, resources)
	if err != nil {
		t.Fatalf("expected snapshot build success, got %v", err)
	}
	if len(got.ModifierRuleSnapshot) != 7 {
		t.Fatalf("expected 7 snapshot rules, got %d", len(got.ModifierRuleSnapshot))
	}

	expected := map[string]struct{}{
		"configmaps|/data/DB_HOST":                                                                      {},
		"configmaps|/metadata/annotations/bulk.demo~1ip":                                                {},
		"deployments.apps|/metadata/annotations/bulk.demo~1ip":                                          {},
		"deployments.apps|/spec/template/metadata/annotations/bulk.demo~1ip":                            {},
		"deployments.apps|/spec/template/spec/containers/0/env/0/value":                                 {},
		"ingresses.networking.k8s.io|/metadata/annotations/nginx.ingress.kubernetes.io~1upstream-vhost": {},
		"services|/metadata/annotations/bulk.demo~1upstream-ip":                                         {},
	}
	for _, rule := range got.ModifierRuleSnapshot {
		if rule.Pair == nil {
			t.Fatalf("expected reversible pair rule, got %#v", rule)
		}
		key := rule.Conditions.GroupResource + "|" + rule.Pair.Path
		if _, ok := expected[key]; !ok {
			t.Fatalf("unexpected generated rule: %s", key)
		}
		delete(expected, key)
	}
	if len(expected) != 0 {
		t.Fatalf("missing generated rules: %v", expected)
	}
}

func TestPrepareRestorePolicyForPersist_UnrelatedUpdateKeepsSnapshotWithoutRebuild(t *testing.T) {
	t.Parallel()

	previous := &dapisv1.DisasterInstance{
		Spec: dapisv1.DisasterInstanceSpec{
			Config:                  "cfg-1",
			OperationTimeoutMinutes: 60,
			RestorePolicy: &dapisv1.RestorePolicy{
				UseUnifiedDirectionResolver: boolPtr(true),
				BulkModifierActions: []dapisv1.BulkModifierAction{{
					ID:      "replace-ip",
					Action:  dapisv1.BulkModifierActionReplaceExactValue,
					Enabled: boolPtr(true),
					ApplyTo: []dapisv1.RestoreModifierApplyTarget{dapisv1.RestoreModifierApplyResourceSync},
				}},
				ModifierRuleSnapshot: []dapisv1.RestoreModifierRule{{
					ID:   "bulk-replace-ip-0001",
					Mode: dapisv1.RestoreModifierModeReversible,
					Conditions: dapisv1.Conditions{
						GroupResource:     "deployments.apps",
						ResourceNameRegex: "^demo$",
						Namespaces:        []string{"demo-ns"},
					},
					Pair: &dapisv1.RestoreModifierPair{
						Path:        "/metadata/annotations/site-role",
						SourceValue: "secondary",
						TargetValue: "primary",
					},
				}},
				ModifierRuleSnapshotHash: "sha256:existing",
			},
		},
	}
	spec := previous.Spec.DeepCopy()
	spec.OperationTimeoutMinutes = 120

	h := newMockHandler()
	h.BuildBulkModifierSnapshotFunc = func(context.Context, *dapisv1.DisasterInstanceSpec, *rest.Config) (*bulkModifierSnapshotBuildResult, error) {
		t.Fatalf("bulk snapshot builder should not run when only unrelated fields change")
		return nil, nil
	}

	if err := h.prepareRestorePolicyForPersist(context.Background(), spec, previous); err != nil {
		t.Fatalf("expected prepare success, got %v", err)
	}
	if spec.RestorePolicy.ModifierRuleSnapshotHash != "sha256:existing" {
		t.Fatalf("expected existing snapshot hash to be preserved, got %s", spec.RestorePolicy.ModifierRuleSnapshotHash)
	}
	if !reflect.DeepEqual(spec.RestorePolicy.ModifierRuleSnapshot, previous.Spec.RestorePolicy.ModifierRuleSnapshot) {
		t.Fatalf("expected existing snapshot to be preserved")
	}
}

func TestPrepareRestorePolicyForPersist_BulkSnapshotForbiddenPathRejected(t *testing.T) {
	t.Parallel()

	spec := &dapisv1.DisasterInstanceSpec{
		Config: "cfg-1",
		RestorePolicy: &dapisv1.RestorePolicy{
			UseUnifiedDirectionResolver: boolPtr(true),
			BulkModifierActions: []dapisv1.BulkModifierAction{{
				ID:          "replace-ip",
				Action:      dapisv1.BulkModifierActionReplaceExactValue,
				SourceValue: "10.10.0.12",
				TargetValue: "10.20.0.12",
			}},
		},
	}

	h := newMockHandler()
	h.BuildBulkModifierSnapshotFunc = func(context.Context, *dapisv1.DisasterInstanceSpec, *rest.Config) (*bulkModifierSnapshotBuildResult, error) {
		return &bulkModifierSnapshotBuildResult{
			Actions: cloneBulkModifierActions(spec.RestorePolicy.BulkModifierActions),
			ModifierRuleSnapshot: []dapisv1.RestoreModifierRule{{
				ID:   "bulk-forbidden-0001",
				Mode: dapisv1.RestoreModifierModeVeleroNative,
				Conditions: dapisv1.Conditions{
					GroupResource: "deployments.apps",
				},
				VeleroRule: &dapisv1.RestoreModifierVeleroRule{
					Patches: []dapisv1.JSONPatch{{
						Operation: "add",
						Path:      "/status/phase",
						Value:     "patched",
					}},
				},
			}},
			ModifierRuleSnapshotHash: "sha256:test",
		}, nil
	}

	err := h.prepareRestorePolicyForPersist(context.Background(), spec, nil)
	if err == nil {
		t.Fatalf("expected forbidden-path rejection, got nil")
	}
	if !strings.Contains(err.Error(), "forbidden") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrepareRestorePolicyForPersist_BulkFinalizerPathRejectedAsForbidden(t *testing.T) {
	t.Parallel()

	spec := &dapisv1.DisasterInstanceSpec{
		Config:     "cfg-1",
		Namespaces: []string{"demo-ns"},
		RestorePolicy: &dapisv1.RestorePolicy{
			UseUnifiedDirectionResolver: boolPtr(true),
			BulkModifierActions: []dapisv1.BulkModifierAction{{
				ID:          "replace-finalizer",
				Action:      dapisv1.BulkModifierActionReplaceExactValue,
				SourceValue: "bulk.test/finalizer",
				TargetValue: "bulk.test/finalizer-new",
			}},
			ResourceSelection: &dapisv1.RestoreResourceSelectionPolicy{
				IncludedNamespaces:               []string{"demo-ns"},
				IncludedNamespaceScopedResources: []string{"configmaps"},
			},
		},
	}

	h := newMockHandler()
	h.BuildBulkModifierSnapshotFunc = func(context.Context, *dapisv1.DisasterInstanceSpec, *rest.Config) (*bulkModifierSnapshotBuildResult, error) {
		return &bulkModifierSnapshotBuildResult{
			Actions: cloneBulkModifierActions(spec.RestorePolicy.BulkModifierActions),
			ModifierRuleSnapshot: []dapisv1.RestoreModifierRule{{
				ID:   "bulk-replace-finalizer-0001",
				Mode: dapisv1.RestoreModifierModeReversible,
				Conditions: dapisv1.Conditions{
					GroupResource:     "configmaps",
					ResourceNameRegex: "^b08-config$",
					Namespaces:        []string{"demo-ns"},
				},
				Pair: &dapisv1.RestoreModifierPair{
					Path:        "/metadata/finalizers/0",
					SourceValue: "bulk.test/finalizer",
					TargetValue: "bulk.test/finalizer-new",
				},
			}},
			ModifierRuleSnapshotHash: "sha256:test",
		}, nil
	}

	err := h.prepareRestorePolicyForPersist(context.Background(), spec, nil)
	if err == nil {
		t.Fatalf("expected forbidden-path rejection, got nil")
	}
	if strings.Contains(err.Error(), "outside instance namespaces") {
		t.Fatalf("expected forbidden-path rejection, got namespace scope error: %v", err)
	}
	if !strings.Contains(err.Error(), "forbidden") {
		t.Fatalf("unexpected error: %v", err)
	}
}
