package appbackup

import (
	"fmt"
	"strings"
)

const AppBackupResourceFilterInvalid = "ResourceFilterInvalid"

type appBackupResourceFilters struct {
	includedResources       []string
	excludedResources       []string
	includeClusterResources *bool
	includedNamespaceScoped []string
	excludedNamespaceScoped []string
	includedClusterScoped   []string
	excludedClusterScoped   []string
}

func hasScopedAppBackupResourceFilters(
	includedNamespaceScoped []string,
	excludedNamespaceScoped []string,
	includedClusterScoped []string,
	excludedClusterScoped []string,
) bool {
	return len(normalizeResourceFilterList(includedNamespaceScoped)) > 0 ||
		len(normalizeResourceFilterList(excludedNamespaceScoped)) > 0 ||
		len(normalizeResourceFilterList(includedClusterScoped)) > 0 ||
		len(normalizeResourceFilterList(excludedClusterScoped)) > 0
}

func validateCreateAppBackupResourceFilters(req *CreateAppBackupRequest) error {
	if req == nil {
		return nil
	}
	req.normalizeScopedResourceFilters()
	return validateAppBackupResourceFilters(appBackupResourceFilters{
		includedResources:       req.IncludedResources,
		excludedResources:       req.ExcludedResources,
		includeClusterResources: req.IncludeClusterResources,
		includedNamespaceScoped: req.IncludedNamespaceScopedResources,
		excludedNamespaceScoped: req.ExcludedNamespaceScopedResources,
		includedClusterScoped:   req.IncludedClusterScopedResources,
		excludedClusterScoped:   req.ExcludedClusterScopedResources,
	})
}

func validateUpdateAppBackupResourceFilters(req *UpdateAppBackupRequest) error {
	if req == nil {
		return nil
	}
	req.normalizeScopedResourceFilters()
	return validateAppBackupResourceFilters(appBackupResourceFilters{
		includedResources:       req.IncludedResources,
		excludedResources:       req.ExcludedResources,
		includeClusterResources: req.IncludeClusterResources,
		includedNamespaceScoped: req.IncludedNamespaceScopedResources,
		excludedNamespaceScoped: req.ExcludedNamespaceScopedResources,
		includedClusterScoped:   req.IncludedClusterScopedResources,
		excludedClusterScoped:   req.ExcludedClusterScopedResources,
	})
}

func validateAppBackupResourceFilters(filters appBackupResourceFilters) error {
	includedResources := normalizeResourceFilterList(filters.includedResources)
	excludedResources := normalizeResourceFilterList(filters.excludedResources)
	includedNamespaceScoped := normalizeResourceFilterList(filters.includedNamespaceScoped)
	excludedNamespaceScoped := normalizeResourceFilterList(filters.excludedNamespaceScoped)
	includedClusterScoped := normalizeResourceFilterList(filters.includedClusterScoped)
	excludedClusterScoped := normalizeResourceFilterList(filters.excludedClusterScoped)

	hasOld := len(includedResources) > 0 || len(excludedResources) > 0 || filters.includeClusterResources != nil
	hasScoped := len(includedNamespaceScoped) > 0 || len(excludedNamespaceScoped) > 0 || len(includedClusterScoped) > 0 || len(excludedClusterScoped) > 0
	if hasOld && hasScoped {
		return fmt.Errorf(
			"%s: old resource fields (includedResources/excludedResources/includeClusterResources) cannot be mixed with scoped resource fields",
			AppBackupResourceFilterInvalid,
		)
	}

	if hasScoped {
		if err := validateResourceFilterPair(
			"includedNamespaceScopedResources",
			"excludedNamespaceScopedResources",
			includedNamespaceScoped,
			excludedNamespaceScoped,
		); err != nil {
			return err
		}
		return validateResourceFilterPair(
			"includedClusterScopedResources",
			"excludedClusterScopedResources",
			includedClusterScoped,
			excludedClusterScoped,
		)
	}

	return validateResourceFilterPair(
		"includedResources",
		"excludedResources",
		includedResources,
		excludedResources,
	)
}

func validateResourceFilterPair(includeField string, excludeField string, include []string, exclude []string) error {
	if len(include) == 0 || len(exclude) == 0 {
		return nil
	}

	includeSet := make(map[string]struct{}, len(include))
	for _, item := range include {
		includeSet[item] = struct{}{}
	}
	excludeSet := make(map[string]struct{}, len(exclude))
	for _, item := range exclude {
		excludeSet[item] = struct{}{}
	}

	if _, hasWildcard := includeSet["*"]; hasWildcard {
		return fmt.Errorf(
			"%s: %s contains '*' and cannot be combined with %s",
			AppBackupResourceFilterInvalid,
			includeField,
			excludeField,
		)
	}
	if _, hasWildcard := excludeSet["*"]; hasWildcard {
		return fmt.Errorf(
			"%s: %s contains '*' and cannot be combined with %s",
			AppBackupResourceFilterInvalid,
			excludeField,
			includeField,
		)
	}

	for _, item := range include {
		if _, conflict := excludeSet[item]; conflict {
			return fmt.Errorf(
				"%s: %s and %s conflict on resource %q",
				AppBackupResourceFilterInvalid,
				includeField,
				excludeField,
				item,
			)
		}
	}
	return nil
}

func normalizeResourceFilterList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		v := strings.TrimSpace(value)
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
