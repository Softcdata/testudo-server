package apprestore

import (
	"context"
	"fmt"
	"strings"

	veleroresource "github.com/softcdata/testudo-server/internal/veleroresource"
	"k8s.io/apimachinery/pkg/api/meta"
)

type resourceFilterValidationError struct {
	err error
}

func (e *resourceFilterValidationError) Error() string {
	return e.err.Error()
}

func (e *resourceFilterValidationError) Unwrap() error {
	return e.err
}

func (h *AppRestoreHandler) normalizeRestoreResourceFilters(ctx context.Context, sourceCluster string, included, excluded []string) ([]string, []string, error) {
	mapper := meta.RESTMapper(nil)
	if veleroresource.NeedsRESTMapper(included, excluded) {
		if strings.TrimSpace(sourceCluster) == "" {
			return nil, nil, fmt.Errorf("source cluster is required to resolve GVK resource filters")
		}
		sourceClient, err := h.getClusterClient(ctx, sourceCluster)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get source cluster client %q for resource filter mapping: %w", sourceCluster, err)
		}
		if sourceClient == nil || sourceClient.RESTMapper() == nil {
			return nil, nil, fmt.Errorf("source cluster client %q does not provide a RESTMapper", sourceCluster)
		}
		mapper = sourceClient.RESTMapper()
	}

	normalizedIncluded, err := veleroresource.NormalizeResourceFilters("includedResources", included, mapper)
	if err != nil {
		return nil, nil, err
	}
	normalizedExcluded, err := veleroresource.NormalizeResourceFilters("excludedResources", excluded, mapper)
	if err != nil {
		return nil, nil, err
	}
	return normalizedIncluded, normalizedExcluded, nil
}
