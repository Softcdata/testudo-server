package resourcemodifier

import (
	"strings"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
)

func CleanVolume() dapisv1.ResourceModifierRule {
	return dapisv1.ResourceModifierRule{
		Conditions: dapisv1.Conditions{
			GroupResource: "persistentvolumeclaims",
		},
		Patches: []dapisv1.JSONPatch{
			{
				// Use add with an empty value instead of remove. RFC6902 remove fails when
				// Velero restores a PVC that no longer has spec.volumeName in the backup.
				Operation: "add",
				Path:      "/spec/volumeName",
				Value:     "",
			},
		},
	}
}
func SCMapping(mapping map[string]string) []dapisv1.ResourceModifierRule {
	var rules []dapisv1.ResourceModifierRule
	for oldSC, newSC := range mapping {
		rules = append(rules, dapisv1.ResourceModifierRule{
			Conditions: dapisv1.Conditions{
				GroupResource: "persistentvolumeclaims",
			},
			Patches: []dapisv1.JSONPatch{
				{
					Operation: "test",
					Path:      "/spec/storageClassName",
					Value:     oldSC,
				},
				{
					Operation: "replace",
					Path:      "/spec/storageClassName",
					Value:     newSC,
				},
			},
		})
	}
	return rules
}

// IngressClassMapping generates rules to map Ingress class names during restore
// Example: {"nginx": "traefik"} will replace ingressClassName from "nginx" to "traefik"
func IngressClassMapping(mapping map[string]string) []dapisv1.ResourceModifierRule {
	var rules []dapisv1.ResourceModifierRule
	for oldClass, newClass := range mapping {
		rules = append(rules, dapisv1.ResourceModifierRule{
			Conditions: dapisv1.Conditions{
				GroupResource: "ingresses.networking.k8s.io",
			},
			Patches: []dapisv1.JSONPatch{
				{
					Operation: "test",
					Path:      "/spec/ingressClassName",
					Value:     oldClass,
				},
				{
					Operation: "replace",
					Path:      "/spec/ingressClassName",
					Value:     newClass,
				},
			},
		})
	}
	return rules
}

// ScaleToZero generates rules to scale deployments and statefulsets to 0 replicas
func ScaleToZero(names []string) []dapisv1.ResourceModifierRule {
	if len(names) == 0 {
		return nil
	}

	// Create regex: ^(name1|name2)$
	// TODO: Escape strings to avoid regex injection if needed
	regex := "^(" + strings.Join(names, "|") + ")$"
	if len(names) == 1 && names[0] == "*" {
		regex = ".*"
	}

	patch := dapisv1.JSONPatch{
		Operation: "replace",
		Path:      "/spec/replicas",
		Value:     "0",
	}

	return []dapisv1.ResourceModifierRule{
		{
			Conditions: dapisv1.Conditions{
				GroupResource:     "deployments.apps",
				ResourceNameRegex: regex,
			},
			Patches: []dapisv1.JSONPatch{patch},
		},
		{
			Conditions: dapisv1.Conditions{
				GroupResource:     "statefulsets.apps",
				ResourceNameRegex: regex,
			},
			Patches: []dapisv1.JSONPatch{patch},
		},
	}
}

// StandbyReplacement generates rules to replace image with busybox and command with sleep infinity
func StandbyReplacement(names []string) []dapisv1.ResourceModifierRule {
	if len(names) == 0 {
		return nil
	}

	regex := "^(" + strings.Join(names, "|") + ")$"
	if len(names) == 1 && names[0] == "*" {
		regex = ".*"
	}

	patches := []dapisv1.JSONPatch{
		{
			Operation: "replace",
			Path:      "/spec/template/spec/containers/0/image",
			Value:     "busybox:latest",
		},
		{
			Operation: "add",
			Path:      "/spec/template/spec/containers/0/command",
			Value:     "[\"/bin/sh\", \"-c\", \"sleep infinity\"]",
		},
		{
			Operation: "remove",
			Path:      "/spec/template/spec/containers/0/livenessProbe",
		},
		{
			Operation: "remove",
			Path:      "/spec/template/spec/containers/0/readinessProbe",
		},
		{
			Operation: "remove",
			Path:      "/spec/template/spec/containers/0/startupProbe",
		},
	}

	return []dapisv1.ResourceModifierRule{
		{
			Conditions: dapisv1.Conditions{
				GroupResource:     "deployments.apps",
				ResourceNameRegex: regex,
			},
			Patches: patches,
		},
		{
			Conditions: dapisv1.Conditions{
				GroupResource:     "statefulsets.apps",
				ResourceNameRegex: regex,
			},
			Patches: patches,
		},
	}
}

// TrafficlessRestore generates rules to sterilize pods for Scheme A (造壳填肉)
// This modifies pods directly (not workload templates) to:
// 1. Add trafficless=true label for identification
// 2. Replace image with busybox for lightweight data transfer
// 3. Inject sleep command to keep the pod running
func TrafficlessRestore(image string) dapisv1.ResourceModifierRule {
	if image == "" {
		image = "busybox:latest"
	}

	patches := []dapisv1.JSONPatch{
		{
			Operation: "add",
			Path:      "/metadata/labels/trafficless",
			Value:     "true",
		},
		{
			Operation: "replace",
			Path:      "/spec/containers/0/image",
			Value:     image,
		},
		{
			Operation: "add",
			Path:      "/spec/containers/0/command",
			Value:     "[\"sleep\", \"3600\"]",
		},
	}

	return dapisv1.ResourceModifierRule{
		Conditions: dapisv1.Conditions{
			GroupResource: "pods",
		},
		Patches: patches,
	}
}
