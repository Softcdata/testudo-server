package veleroresource

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NormalizeResourceFilters converts legacy GVK values to Velero group-resource values.
func NormalizeResourceFilters(field string, values []string, mapper meta.RESTMapper) ([]string, error) {
	if values == nil {
		return nil, nil
	}

	normalized := make([]string, len(values))
	for i, raw := range values {
		value := strings.TrimSpace(raw)
		gvk, isGVK, err := parseGVK(value)
		if err != nil {
			return nil, fmt.Errorf("%s[%d] resource %q: %w", field, i, value, err)
		}
		if !isGVK {
			normalized[i] = value
			continue
		}
		if mapper == nil {
			return nil, fmt.Errorf("%s[%d] resource %q requires source cluster RESTMapper", field, i, value)
		}

		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return nil, fmt.Errorf("%s[%d] resource %q cannot resolve with source cluster RESTMapper: %w", field, i, value, err)
		}
		resource := mapping.Resource.GroupResource().String()
		if resource == "" {
			return nil, fmt.Errorf("%s[%d] resource %q resolved to an empty group-resource", field, i, value)
		}
		normalized[i] = resource
	}

	return normalized, nil
}

// NeedsRESTMapper reports whether any value uses the legacy GVK notation.
func NeedsRESTMapper(values ...[]string) bool {
	for _, list := range values {
		for _, raw := range list {
			if strings.Contains(strings.TrimSpace(raw), "/") {
				return true
			}
		}
	}
	return false
}

func parseGVK(value string) (schema.GroupVersionKind, bool, error) {
	if value == "" || value == "*" || !strings.Contains(value, "/") {
		return schema.GroupVersionKind{}, false, nil
	}

	parts := strings.Split(value, "/")
	switch len(parts) {
	case 2:
		if parts[0] == "" || parts[1] == "" {
			return schema.GroupVersionKind{}, false, fmt.Errorf("invalid GVK format")
		}
		return schema.GroupVersionKind{Version: parts[0], Kind: parts[1]}, true, nil
	case 3:
		if parts[0] == "" || parts[1] == "" || parts[2] == "" {
			return schema.GroupVersionKind{}, false, fmt.Errorf("invalid GVK format")
		}
		return schema.GroupVersionKind{Group: parts[0], Version: parts[1], Kind: parts[2]}, true, nil
	default:
		return schema.GroupVersionKind{}, false, fmt.Errorf("invalid GVK format")
	}
}
