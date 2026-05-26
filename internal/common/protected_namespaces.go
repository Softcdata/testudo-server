package common

import (
	"sort"
	"strings"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	listers "github.com/softcdata/testudo-operator/pkg/listers/disaster/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type ProtectedNamespaceOwner struct {
	Namespace         string `json:"namespace"`
	InstanceName      string `json:"instanceName"`
	InstanceNamespace string `json:"instanceNamespace"`
	ConfigName        string `json:"configName"`
}

type ProtectedNamespaceRecord struct {
	Cluster   string                    `json:"cluster"`
	Namespace string                    `json:"namespace"`
	Owners    []ProtectedNamespaceOwner `json:"owners,omitempty"`
}

type ProtectedNamespaceIndex struct {
	byCluster map[string]map[string][]ProtectedNamespaceOwner
}

func BuildProtectedNamespaceIndex(
	configLister listers.DisasterConfigLister,
	instanceLister listers.DisasterInstanceLister,
) (*ProtectedNamespaceIndex, error) {
	configs, err := configLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	instances, err := instanceLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	configToSourceCluster := make(map[string]string, len(configs))
	for _, cfg := range configs {
		if cfg == nil {
			continue
		}
		sourceCluster := strings.TrimSpace(cfg.Spec.SourceCluster)
		if sourceCluster == "" {
			continue
		}
		configToSourceCluster[cfg.Name] = sourceCluster
	}

	index := &ProtectedNamespaceIndex{
		byCluster: make(map[string]map[string][]ProtectedNamespaceOwner),
	}
	for _, instance := range instances {
		if instance == nil {
			continue
		}
		configName := strings.TrimSpace(instance.Spec.Config)
		sourceCluster := configToSourceCluster[configName]
		if sourceCluster == "" {
			continue
		}

		normalizedNamespaces := NormalizeNamespaces(instance.Spec.Namespaces)
		if len(normalizedNamespaces) == 0 {
			continue
		}

		namespaceOwners := index.byCluster[sourceCluster]
		if namespaceOwners == nil {
			namespaceOwners = make(map[string][]ProtectedNamespaceOwner)
			index.byCluster[sourceCluster] = namespaceOwners
		}

		for _, namespace := range normalizedNamespaces {
			namespaceOwners[namespace] = append(namespaceOwners[namespace], ProtectedNamespaceOwner{
				Namespace:         namespace,
				InstanceName:      instance.Name,
				InstanceNamespace: instance.Namespace,
				ConfigName:        configName,
			})
		}
	}

	for clusterName, namespaceOwners := range index.byCluster {
		for namespace, owners := range namespaceOwners {
			namespaceOwners[namespace] = sortProtectedNamespaceOwners(owners)
		}
		index.byCluster[clusterName] = namespaceOwners
	}

	return index, nil
}

func NormalizeNamespaces(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		namespace := strings.TrimSpace(item)
		if namespace == "" {
			continue
		}
		if _, ok := seen[namespace]; ok {
			continue
		}
		seen[namespace] = struct{}{}
		out = append(out, namespace)
	}
	sort.Strings(out)
	return out
}

func (i *ProtectedNamespaceIndex) Records(clusterName string) []ProtectedNamespaceRecord {
	if i == nil {
		return nil
	}
	clusterName = strings.TrimSpace(clusterName)
	namespaceOwners := i.byCluster[clusterName]
	if len(namespaceOwners) == 0 {
		return nil
	}

	namespaces := make([]string, 0, len(namespaceOwners))
	for namespace := range namespaceOwners {
		namespaces = append(namespaces, namespace)
	}
	sort.Strings(namespaces)

	records := make([]ProtectedNamespaceRecord, 0, len(namespaces))
	for _, namespace := range namespaces {
		records = append(records, ProtectedNamespaceRecord{
			Cluster:   clusterName,
			Namespace: namespace,
			Owners:    append([]ProtectedNamespaceOwner(nil), namespaceOwners[namespace]...),
		})
	}
	return records
}

func (i *ProtectedNamespaceIndex) Conflicts(
	clusterName string,
	namespaces []string,
	excludeNamespace string,
	excludeName string,
) ([]string, []ProtectedNamespaceOwner) {
	if i == nil {
		return nil, nil
	}
	clusterName = strings.TrimSpace(clusterName)
	normalizedNamespaces := NormalizeNamespaces(namespaces)
	if clusterName == "" || len(normalizedNamespaces) == 0 {
		return nil, nil
	}

	namespaceOwners := i.byCluster[clusterName]
	if len(namespaceOwners) == 0 {
		return nil, nil
	}

	conflictSet := make(map[string]struct{}, len(normalizedNamespaces))
	owners := make([]ProtectedNamespaceOwner, 0)
	for _, namespace := range normalizedNamespaces {
		currentOwners := namespaceOwners[namespace]
		for _, owner := range currentOwners {
			if owner.InstanceNamespace == excludeNamespace && owner.InstanceName == excludeName {
				continue
			}
			conflictSet[namespace] = struct{}{}
			owners = append(owners, owner)
		}
	}

	if len(conflictSet) == 0 {
		return nil, nil
	}

	conflictNamespaces := make([]string, 0, len(conflictSet))
	for namespace := range conflictSet {
		conflictNamespaces = append(conflictNamespaces, namespace)
	}
	sort.Strings(conflictNamespaces)

	return conflictNamespaces, sortProtectedNamespaceOwners(owners)
}

func sortProtectedNamespaceOwners(owners []ProtectedNamespaceOwner) []ProtectedNamespaceOwner {
	if len(owners) == 0 {
		return nil
	}
	out := append([]ProtectedNamespaceOwner(nil), owners...)
	sort.SliceStable(out, func(i, j int) bool {
		if namespaceCmp := strings.Compare(out[i].Namespace, out[j].Namespace); namespaceCmp != 0 {
			return namespaceCmp < 0
		}
		if instanceNamespaceCmp := strings.Compare(out[i].InstanceNamespace, out[j].InstanceNamespace); instanceNamespaceCmp != 0 {
			return instanceNamespaceCmp < 0
		}
		if instanceNameCmp := strings.Compare(out[i].InstanceName, out[j].InstanceName); instanceNameCmp != 0 {
			return instanceNameCmp < 0
		}
		return strings.Compare(out[i].ConfigName, out[j].ConfigName) < 0
	})
	return out
}

func ProtectedNamespacesFromRecords(records []ProtectedNamespaceRecord) []string {
	if len(records) == 0 {
		return nil
	}
	out := make([]string, 0, len(records))
	for _, record := range records {
		out = append(out, record.Namespace)
	}
	sort.Strings(out)
	return out
}

func ProtectedNamespacesFromInstances(items []*dapisv1.DisasterInstance) []string {
	namespaces := make([]string, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		namespaces = append(namespaces, item.Spec.Namespaces...)
	}
	return NormalizeNamespaces(namespaces)
}
