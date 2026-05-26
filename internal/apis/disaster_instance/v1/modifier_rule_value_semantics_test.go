package instance

import (
	"strings"
	"testing"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
)

func TestEnsureJSONPointerLocatable_AllowsAddMissingFinalMapKey(t *testing.T) {
	t.Parallel()

	err := ensureJSONPointerLocatable(
		map[string]any{
			"metadata": map[string]any{
				"labels": map[string]any{
					"app": "nginx",
				},
			},
		},
		"/metadata/labels/type",
		"add",
	)
	if err != nil {
		t.Fatalf("expected add of missing final map key to pass, got %v", err)
	}
}

func TestEnsureJSONPointerLocatable_RejectsReplaceMissingFinalMapKey(t *testing.T) {
	t.Parallel()

	err := ensureJSONPointerLocatable(
		map[string]any{
			"metadata": map[string]any{
				"labels": map[string]any{
					"app": "nginx",
				},
			},
		},
		"/metadata/labels/type",
		"replace",
	)
	if err == nil {
		t.Fatalf("expected replace of missing final map key to fail")
	}
	if !strings.Contains(err.Error(), `path segment "type" not found`) {
		t.Fatalf("unexpected error detail: %v", err)
	}
}

func TestEnsureJSONPointerLocatable_RejectsAddMissingIntermediateMapKey(t *testing.T) {
	t.Parallel()

	err := ensureJSONPointerLocatable(
		map[string]any{
			"metadata": map[string]any{
				"labels": map[string]any{
					"app": "nginx",
				},
			},
		},
		"/metadata/not-exists/type",
		"add",
	)
	if err == nil {
		t.Fatalf("expected add with missing intermediate map key to fail")
	}
	if !strings.Contains(err.Error(), `path segment "not-exists" not found`) {
		t.Fatalf("unexpected error detail: %v", err)
	}
}

func TestValidateReversiblePairValueCompatibility_AllowsMetadataStringFields(t *testing.T) {
	t.Parallel()

	err := validateReversiblePairValueCompatibility(
		"/metadata/annotations/testudo.softcdata.com~1site-role",
		&dapisv1.RestoreModifierPair{
			SourceValue: "1",
			TargetValue: "2",
		},
		map[string]any{
			"metadata": map[string]any{
				"annotations": map[string]any{
					"testudo.softcdata.com/site-role": "1",
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("expected metadata string field to pass, got %v", err)
	}
}

func TestValidateReversiblePairValueCompatibility_RejectsNumericLiteralForStringField(t *testing.T) {
	t.Parallel()

	err := validateReversiblePairValueCompatibility(
		"/spec/template/spec/containers/0/env/0/value",
		&dapisv1.RestoreModifierPair{
			SourceValue: "1",
			TargetValue: "2",
		},
		map[string]any{
			"spec": map[string]any{
				"template": map[string]any{
					"spec": map[string]any{
						"containers": []any{
							map[string]any{
								"env": []any{
									map[string]any{"value": "cluster-a"},
								},
							},
						},
					},
				},
			},
		},
	)
	if err == nil {
		t.Fatalf("expected type mismatch rejection, got nil")
	}
	if !strings.Contains(err.Error(), "would be applied as number but live field type is string") {
		t.Fatalf("unexpected error detail: %v", err)
	}
}

func TestValidateReversiblePairValueCompatibility_RejectsStringLiteralForNumberField(t *testing.T) {
	t.Parallel()

	err := validateReversiblePairValueCompatibility(
		"/spec/replicas",
		&dapisv1.RestoreModifierPair{
			SourceValue: "1",
			TargetValue: "primary",
		},
		map[string]any{
			"spec": map[string]any{
				"replicas": int64(1),
			},
		},
	)
	if err == nil {
		t.Fatalf("expected number field rejection, got nil")
	}
	if !strings.Contains(err.Error(), "would be applied as string but live field type is number") {
		t.Fatalf("unexpected error detail: %v", err)
	}
}
