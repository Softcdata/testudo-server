package event

import (
	"fmt"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
)

var eventResourceKindAliases = map[string]string{
	"appbackup":           "AppBackup",
	"appbackups":          "AppBackup",
	"apprestore":          "AppRestore",
	"apprestores":         "AppRestore",
	"cluster":             "Cluster",
	"clusters":            "Cluster",
	"storagerepository":   "StorageRepository",
	"storagerepositories": "StorageRepository",
	"disasterpolicy":      "DisasterPolicy",
	"disasterpolicies":    "DisasterPolicy",
	"disasterconfig":      "DisasterConfig",
	"disasterconfigs":     "DisasterConfig",
	"disasterinstance":    "DisasterInstance",
	"disasterinstances":   "DisasterInstance",
	"disastergroup":       "DisasterGroup",
	"disastergroups":      "DisasterGroup",
	"disasterdrill":       "DisasterDrill",
	"disasterdrills":      "DisasterDrill",
	"disasteroperation":   "DisasterOperation",
	"disasteroperations":  "DisasterOperation",
	"datasync":            "DataSync",
	"datasyncs":           "DataSync",
	"resourcesync":        "ResourceSync",
	"resourcesyncs":       "ResourceSync",
	"disasterbackup":      "DisasterBackup",
	"disasterbackups":     "DisasterBackup",
	"disasterjob":         "DisasterJob",
	"disasterjobs":        "DisasterJob",
}

func resolveEventResourceKind(resource string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(resource))
	if normalized == "" {
		return "", fmt.Errorf("resource is required")
	}
	if kind, ok := eventResourceKindAliases[normalized]; ok {
		return kind, nil
	}
	supported := make([]string, 0, len(eventResourceKindAliases))
	for alias := range eventResourceKindAliases {
		supported = append(supported, alias)
	}
	sort.Strings(supported)
	return "", fmt.Errorf("unsupported resource: %s (supported: %s)", resource, strings.Join(supported, ","))
}

func filterEventsByKind(items []corev1.Event, expectedKind string) []corev1.Event {
	if expectedKind == "" {
		return items
	}
	filtered := make([]corev1.Event, 0, len(items))
	for _, item := range items {
		if item.InvolvedObject.Kind == expectedKind {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func buildWatchKindFilter(expectedKind string) func(watch.Event) bool {
	if expectedKind == "" {
		return nil
	}
	return func(evt watch.Event) bool {
		e, ok := evt.Object.(*corev1.Event)
		if !ok {
			return false
		}
		return e.InvolvedObject.Kind == expectedKind
	}
}
