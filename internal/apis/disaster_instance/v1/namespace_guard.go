package instance

import (
	"fmt"
	"strings"

	"github.com/softcdata/testudo-server/internal/common"
)

type protectedNamespaceConflictError struct {
	message string
	meta    ProtectedNamespaceConflictMeta
}

type protectedNamespaceValidationError struct {
	message string
}

func (e *protectedNamespaceConflictError) Error() string {
	if e == nil {
		return ""
	}
	return e.message
}

func (e *protectedNamespaceValidationError) Error() string {
	if e == nil {
		return ""
	}
	return e.message
}

func newProtectedNamespaceConflictError(sourceCluster string, namespaces []string, owners []common.ProtectedNamespaceOwner) error {
	return &protectedNamespaceConflictError{
		message: fmt.Sprintf(
			"source cluster %s already protects namespaces: %s",
			sourceCluster,
			strings.Join(namespaces, ","),
		),
		meta: ProtectedNamespaceConflictMeta{
			ConflictType:       "protectedNamespaces",
			SourceCluster:      sourceCluster,
			ConflictNamespaces: append([]string(nil), namespaces...),
			Owners:             append([]common.ProtectedNamespaceOwner(nil), owners...),
		},
	}
}

func (h *InstanceHandler) resolveProtectedNamespaceSourceCluster(configName string) (string, error) {
	configName = strings.TrimSpace(configName)
	if configName == "" {
		return "", nil
	}
	cfg, err := h.DisasterConfigLister.Get(configName)
	if err != nil {
		return "", &protectedNamespaceValidationError{
			message: fmt.Sprintf("config %s not found", configName),
		}
	}
	return strings.TrimSpace(cfg.Spec.SourceCluster), nil
}

func (h *InstanceHandler) validateProtectedNamespaces(
	sourceCluster string,
	namespaces []string,
	excludeNamespace string,
	excludeName string,
) error {
	sourceCluster = strings.TrimSpace(sourceCluster)
	if sourceCluster == "" {
		return nil
	}

	index, err := common.BuildProtectedNamespaceIndex(h.DisasterConfigLister, h.InstanceLister)
	if err != nil {
		return err
	}

	conflictNamespaces, owners := index.Conflicts(sourceCluster, namespaces, excludeNamespace, excludeName)
	if len(conflictNamespaces) == 0 {
		return nil
	}
	return newProtectedNamespaceConflictError(sourceCluster, conflictNamespaces, owners)
}
