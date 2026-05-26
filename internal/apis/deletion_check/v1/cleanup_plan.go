package deletioncheck

import (
	"context"
	"fmt"
	"sort"
	"strings"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-server/internal/common"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// remoteLister is the minimal surface we need for remote cleanup resolution.
// Keeping this narrow avoids pulling in heavy fake client implementations in vendored builds.
type remoteLister interface {
	List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error
}

type remoteClientGetter func(ctx context.Context, clusterName string) (remoteLister, error)

const (
	// Cleanup labels (defined by openspec add-deletion-cleanup-plan).
	labelCleanupOwnerToken = "testudo.softcdata.com/cleanup-owner-token"
	labelCleanupRelation   = "testudo.softcdata.com/cleanup-relation"
	labelCleanupStrategy   = "testudo.softcdata.com/cleanup-strategy"
	labelCleanupManagedBy  = "testudo.softcdata.com/cleanup-managed-by"

	cleanupManagedByValueOperator = "disaster-operator"

	// Expected relation-codes (v1).
	relationFinalizerVeleroSchedule            = "finalizer.veleroSchedule"
	relationFinalizerVeleroBackup              = "finalizer.veleroBackup"
	relationFinalizerVeleroRestore             = "finalizer.veleroRestore"
	relationFinalizerResourceModifierConfigMap = "finalizer.resourceModifierConfigMap"

	// Expected strategies (v1).
	cleanupStrategyDelete        = "delete"
	cleanupStrategyDeleteRequest = "delete_request"
	cleanupStrategyOwnerRef      = "owner_reference"
)

func (h *DeletionCheckHandler) buildCleanupPlan(ctx context.Context, targetKind string, namespace string, targetObj metav1.Object) CleanupPlan {
	plan := CleanupPlan{
		HasCleanup:       false,
		FinalizerCleanup: make([]CleanupRef, 0),
		CascadeCleanup:   make([]CleanupRef, 0),
	}
	if targetObj == nil {
		return plan
	}

	switch targetKind {
	case kindAppBackup:
		if ab, ok := targetObj.(*dapisv1.AppBackup); ok {
			plan.FinalizerCleanup = append(plan.FinalizerCleanup, h.planAppBackupFinalizerCleanup(ctx, ab)...)
			plan.CascadeCleanup = append(plan.CascadeCleanup, h.planOwnerRefCascadeCleanup(ctx, namespace, string(ab.UID), kindBackupRestoreStatistics)...)
		}
	case kindAppRestore:
		if ar, ok := targetObj.(*dapisv1.AppRestore); ok {
			plan.FinalizerCleanup = append(plan.FinalizerCleanup, h.planAppRestoreFinalizerCleanup(ctx, ar)...)
		}
	case kindDisasterInstance:
		if inst, ok := targetObj.(*dapisv1.DisasterInstance); ok {
			plan.CascadeCleanup = append(plan.CascadeCleanup, h.planOwnerRefCascadeCleanup(ctx, namespace, string(inst.UID), kindDataSync)...)
			plan.CascadeCleanup = append(plan.CascadeCleanup, h.planOwnerRefCascadeCleanup(ctx, namespace, string(inst.UID), kindResourceSync)...)
			plan.CascadeCleanup = append(plan.CascadeCleanup, h.planOperationCascadeCleanupBySpec(ctx, namespace, inst.Name, "")...)
		}
	case kindDisasterGroup:
		if grp, ok := targetObj.(*dapisv1.DisasterGroup); ok {
			plan.CascadeCleanup = append(plan.CascadeCleanup, h.planOperationCascadeCleanupBySpec(ctx, namespace, "", grp.Name)...)
		}
	case kindDisasterDrill:
		if drill, ok := targetObj.(*dapisv1.DisasterDrill); ok {
			plan.CascadeCleanup = append(plan.CascadeCleanup, h.planOwnerRefCascadeCleanup(ctx, namespace, string(drill.UID), kindDisasterOperation)...)
		}
	}

	plan = dedupCleanupPlan(plan)
	plan.HasCleanup = len(plan.FinalizerCleanup) > 0 || len(plan.CascadeCleanup) > 0
	return plan
}

func dedupCleanupPlan(plan CleanupPlan) CleanupPlan {
	plan.FinalizerCleanup = dedupCleanupRefs(plan.FinalizerCleanup)
	plan.CascadeCleanup = dedupCleanupRefs(plan.CascadeCleanup)
	return plan
}

func dedupCleanupRefs(in []CleanupRef) []CleanupRef {
	if len(in) == 0 {
		return in
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]CleanupRef, 0, len(in))
	for _, ref := range in {
		// For resolved refs, (kind/ns/name/uid/cluster/relation) is stable.
		// For unresolved refs, include selector+reason to avoid dropping distinct entries.
		key := strings.Join([]string{
			ref.Kind,
			ref.Namespace,
			ref.Name,
			ref.UID,
			ref.Cluster,
			ref.RelationCode,
			ref.Strategy,
			fmt.Sprintf("%t", ref.Resolved),
			fmt.Sprintf("%v", ref.Selector),
			ref.UnresolvedReason,
		}, "|")
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, ref)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		if out[i].Cluster != out[j].Cluster {
			return out[i].Cluster < out[j].Cluster
		}
		if out[i].Namespace != out[j].Namespace {
			return out[i].Namespace < out[j].Namespace
		}
		return out[i].Name < out[j].Name
	})
	return out
}

const (
	kindBackupRestoreStatistics = "BackupRestoreStatistics"
)

func (h *DeletionCheckHandler) planOwnerRefCascadeCleanup(ctx context.Context, namespace string, ownerUID string, kind string) []CleanupRef {
	ownerUID = strings.TrimSpace(ownerUID)
	if ownerUID == "" {
		return nil
	}
	switch kind {
	case kindDataSync:
		list, err := h.DisasterClient.DisasterV1().DataSyncs(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return []CleanupRef{{
				Kind:             kindDataSync,
				Namespace:        namespace,
				RelationCode:     "ownerReference.dataSync",
				Strategy:         cleanupStrategyOwnerRef,
				Resolved:         false,
				UnresolvedReason: err.Error(),
			}}
		}
		refs := make([]CleanupRef, 0)
		for i := range list.Items {
			item := &list.Items[i]
			if !hasOwnerUID(item.GetOwnerReferences(), ownerUID) {
				continue
			}
			refs = append(refs, CleanupRef{
				Kind:         kindDataSync,
				Name:         item.Name,
				Namespace:    item.Namespace,
				UID:          string(item.UID),
				RelationCode: "ownerReference.dataSync",
				Strategy:     cleanupStrategyOwnerRef,
				Resolved:     true,
			})
		}
		return refs
	case kindResourceSync:
		list, err := h.DisasterClient.DisasterV1().ResourceSyncs(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return []CleanupRef{{
				Kind:             kindResourceSync,
				Namespace:        namespace,
				RelationCode:     "ownerReference.resourceSync",
				Strategy:         cleanupStrategyOwnerRef,
				Resolved:         false,
				UnresolvedReason: err.Error(),
			}}
		}
		refs := make([]CleanupRef, 0)
		for i := range list.Items {
			item := &list.Items[i]
			if !hasOwnerUID(item.GetOwnerReferences(), ownerUID) {
				continue
			}
			refs = append(refs, CleanupRef{
				Kind:         kindResourceSync,
				Name:         item.Name,
				Namespace:    item.Namespace,
				UID:          string(item.UID),
				RelationCode: "ownerReference.resourceSync",
				Strategy:     cleanupStrategyOwnerRef,
				Resolved:     true,
			})
		}
		return refs
	case kindDisasterOperation:
		list, err := h.DisasterClient.DisasterV1().DisasterOperations(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return []CleanupRef{{
				Kind:             kindDisasterOperation,
				Namespace:        namespace,
				RelationCode:     "ownerReference.disasterOperation",
				Strategy:         cleanupStrategyOwnerRef,
				Resolved:         false,
				UnresolvedReason: err.Error(),
			}}
		}
		refs := make([]CleanupRef, 0)
		for i := range list.Items {
			item := &list.Items[i]
			if !hasOwnerUID(item.GetOwnerReferences(), ownerUID) {
				continue
			}
			refs = append(refs, CleanupRef{
				Kind:         kindDisasterOperation,
				Name:         item.Name,
				Namespace:    item.Namespace,
				UID:          string(item.UID),
				RelationCode: "ownerReference.disasterOperation",
				Strategy:     cleanupStrategyOwnerRef,
				Resolved:     true,
			})
		}
		return refs
	case kindBackupRestoreStatistics:
		list, err := h.DisasterClient.DisasterV1().BackupRestoreStatisticses(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return []CleanupRef{{
				Kind:             kindBackupRestoreStatistics,
				Namespace:        namespace,
				RelationCode:     "ownerReference.backupRestoreStatistics",
				Strategy:         cleanupStrategyOwnerRef,
				Resolved:         false,
				UnresolvedReason: err.Error(),
			}}
		}
		refs := make([]CleanupRef, 0)
		for i := range list.Items {
			item := &list.Items[i]
			if !hasOwnerUID(item.GetOwnerReferences(), ownerUID) {
				continue
			}
			refs = append(refs, CleanupRef{
				Kind:         kindBackupRestoreStatistics,
				Name:         item.Name,
				Namespace:    item.Namespace,
				UID:          string(item.UID),
				RelationCode: "ownerReference.backupRestoreStatistics",
				Strategy:     cleanupStrategyOwnerRef,
				Resolved:     true,
			})
		}
		return refs
	default:
		return nil
	}
}

func hasOwnerUID(refs []metav1.OwnerReference, uid string) bool {
	for _, r := range refs {
		if string(r.UID) == uid {
			return true
		}
	}
	return false
}

func (h *DeletionCheckHandler) planOperationCascadeCleanupBySpec(ctx context.Context, namespace string, instanceName string, groupName string) []CleanupRef {
	instanceName = strings.TrimSpace(instanceName)
	groupName = strings.TrimSpace(groupName)
	if instanceName == "" && groupName == "" {
		return nil
	}

	list, err := h.DisasterClient.DisasterV1().DisasterOperations(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		ref := CleanupRef{
			Kind:             kindDisasterOperation,
			Namespace:        namespace,
			RelationCode:     "spec.instanceName",
			Strategy:         cleanupStrategyDelete,
			Resolved:         false,
			UnresolvedReason: err.Error(),
		}
		if groupName != "" {
			ref.RelationCode = "spec.groupName"
		}
		return []CleanupRef{ref}
	}

	out := make([]CleanupRef, 0)
	for i := range list.Items {
		op := &list.Items[i]
		if instanceName != "" && op.Spec.InstanceName != instanceName {
			continue
		}
		if groupName != "" && op.Spec.GroupName != groupName {
			continue
		}
		relationCode := "spec.instanceName"
		if groupName != "" {
			relationCode = "spec.groupName"
		}
		out = append(out, CleanupRef{
			Kind:         kindDisasterOperation,
			Name:         op.Name,
			Namespace:    op.Namespace,
			UID:          string(op.UID),
			RelationCode: relationCode,
			Strategy:     cleanupStrategyDelete,
			Resolved:     true,
		})
	}
	return out
}

func (h *DeletionCheckHandler) planAppBackupFinalizerCleanup(ctx context.Context, ab *dapisv1.AppBackup) []CleanupRef {
	if ab == nil {
		return nil
	}

	clusterName := strings.TrimSpace(ab.Spec.Cluster)
	ownerUID := string(ab.UID)
	ownerToken := buildDependencyToken(ownerUID)

	expected := []CleanupRef{
		{
			Kind:         "Schedule",
			Cluster:      clusterName,
			RelationCode: relationFinalizerVeleroSchedule,
			Strategy:     cleanupStrategyDelete,
			Resolved:     false,
			Selector:     cleanupSelector(ownerToken, relationFinalizerVeleroSchedule),
		},
		{
			Kind:         "Backup",
			Cluster:      clusterName,
			RelationCode: relationFinalizerVeleroBackup,
			Strategy:     cleanupStrategyDeleteRequest,
			Resolved:     false,
			Selector:     cleanupSelector(ownerToken, relationFinalizerVeleroBackup),
		},
	}

	if clusterName == "" || h.getRemoteClient == nil {
		for i := range expected {
			expected[i].UnresolvedReason = "missing remote cluster client or spec.cluster is empty"
		}
		return expected
	}

	remote, err := h.getRemoteClient(ctx, clusterName)
	if err != nil {
		for i := range expected {
			expected[i].UnresolvedReason = err.Error()
		}
		return expected
	}

	out := make([]CleanupRef, 0)
	out = append(out, resolveVeleroSchedules(ctx, remote, ownerToken, ownerUID, clusterName)...)
	out = append(out, resolveVeleroBackups(ctx, remote, ownerToken, ownerUID, clusterName)...)

	// If we couldn't find anything, keep the expected skeleton so the UI still knows the impact surface.
	if len(out) == 0 {
		for i := range expected {
			expected[i].UnresolvedReason = "no matched objects found (labels missing or resources not created yet)"
		}
		return expected
	}
	return out
}

func (h *DeletionCheckHandler) planAppRestoreFinalizerCleanup(ctx context.Context, ar *dapisv1.AppRestore) []CleanupRef {
	if ar == nil {
		return nil
	}

	clusterName := strings.TrimSpace(ar.Spec.Cluster)
	ownerUID := string(ar.UID)
	ownerToken := buildDependencyToken(ownerUID)

	expected := []CleanupRef{
		{
			Kind:         "Restore",
			Cluster:      clusterName,
			RelationCode: relationFinalizerVeleroRestore,
			Strategy:     cleanupStrategyDelete,
			Resolved:     false,
			Selector:     cleanupSelector(ownerToken, relationFinalizerVeleroRestore),
		},
		{
			Kind:         "ConfigMap",
			Cluster:      clusterName,
			RelationCode: relationFinalizerResourceModifierConfigMap,
			Strategy:     cleanupStrategyDelete,
			Resolved:     false,
			Selector:     cleanupSelector(ownerToken, relationFinalizerResourceModifierConfigMap),
		},
	}

	if clusterName == "" || h.getRemoteClient == nil {
		for i := range expected {
			expected[i].UnresolvedReason = "missing remote cluster client or spec.cluster is empty"
		}
		return expected
	}

	remote, err := h.getRemoteClient(ctx, clusterName)
	if err != nil {
		for i := range expected {
			expected[i].UnresolvedReason = err.Error()
		}
		return expected
	}

	out := make([]CleanupRef, 0)
	out = append(out, resolveVeleroRestores(ctx, remote, ownerToken, ownerUID, clusterName)...)
	out = append(out, resolveResourceModifierConfigMaps(ctx, remote, ownerToken, ownerUID, clusterName)...)

	if len(out) == 0 {
		for i := range expected {
			expected[i].UnresolvedReason = "no matched objects found (labels missing or resources not created yet)"
		}
		return expected
	}
	return out
}

func cleanupSelector(ownerToken string, relation string) map[string]string {
	selector := map[string]string{
		labelCleanupManagedBy: cleanupManagedByValueOperator,
	}
	if ownerToken != "" {
		selector[labelCleanupOwnerToken] = ownerToken
	}
	if relation != "" {
		selector[labelCleanupRelation] = relation
	}
	return selector
}

func resolveVeleroSchedules(ctx context.Context, cli remoteLister, ownerToken string, legacyOwnerUID string, clusterName string) []CleanupRef {
	list := &velerov1.ScheduleList{}
	if err := cli.List(ctx, list, client.InNamespace(common.VeleroNamespace)); err != nil {
		return []CleanupRef{{
			Kind:             "Schedule",
			Cluster:          clusterName,
			RelationCode:     relationFinalizerVeleroSchedule,
			Strategy:         cleanupStrategyDelete,
			Resolved:         false,
			Selector:         cleanupSelector(ownerToken, relationFinalizerVeleroSchedule),
			UnresolvedReason: err.Error(),
		}}
	}

	matched := make([]*velerov1.Schedule, 0)
	for i := range list.Items {
		s := &list.Items[i]
		if matchesCleanupLabels(s.Labels, ownerToken, relationFinalizerVeleroSchedule) || matchesLegacyUIDLabel(s.Labels, "testudo.softcdata.com/app-backup-uid", legacyOwnerUID) {
			matched = append(matched, s)
		}
	}

	if len(matched) == 0 {
		return nil
	}

	out := make([]CleanupRef, 0, len(matched))
	for _, s := range matched {
		out = append(out, CleanupRef{
			Kind:         "Schedule",
			Name:         s.Name,
			Namespace:    s.Namespace,
			UID:          string(s.UID),
			Cluster:      clusterName,
			RelationCode: relationFinalizerVeleroSchedule,
			Strategy:     cleanupStrategyDelete,
			Resolved:     true,
		})
	}
	return out
}

func resolveVeleroBackups(ctx context.Context, cli remoteLister, ownerToken string, legacyOwnerUID string, clusterName string) []CleanupRef {
	list := &velerov1.BackupList{}
	if err := cli.List(ctx, list, client.InNamespace(common.VeleroNamespace)); err != nil {
		return []CleanupRef{{
			Kind:             "Backup",
			Cluster:          clusterName,
			RelationCode:     relationFinalizerVeleroBackup,
			Strategy:         cleanupStrategyDeleteRequest,
			Resolved:         false,
			Selector:         cleanupSelector(ownerToken, relationFinalizerVeleroBackup),
			UnresolvedReason: err.Error(),
		}}
	}

	matched := make([]*velerov1.Backup, 0)
	for i := range list.Items {
		b := &list.Items[i]
		if matchesCleanupLabels(b.Labels, ownerToken, relationFinalizerVeleroBackup) || matchesLegacyUIDLabel(b.Labels, "testudo.softcdata.com/app-backup-uid", legacyOwnerUID) {
			matched = append(matched, b)
		}
	}

	if len(matched) == 0 {
		return nil
	}

	const maxItems = 50
	if len(matched) > maxItems {
		return []CleanupRef{{
			Kind:             "Backup",
			Cluster:          clusterName,
			RelationCode:     relationFinalizerVeleroBackup,
			Strategy:         cleanupStrategyDeleteRequest,
			Resolved:         false,
			Selector:         cleanupSelector(ownerToken, relationFinalizerVeleroBackup),
			UnresolvedReason: fmt.Sprintf("matched too many backups: %d > %d", len(matched), maxItems),
		}}
	}

	out := make([]CleanupRef, 0, len(matched))
	for _, b := range matched {
		out = append(out, CleanupRef{
			Kind:         "Backup",
			Name:         b.Name,
			Namespace:    b.Namespace,
			UID:          string(b.UID),
			Cluster:      clusterName,
			RelationCode: relationFinalizerVeleroBackup,
			Strategy:     cleanupStrategyDeleteRequest,
			Resolved:     true,
		})
	}
	return out
}

func resolveVeleroRestores(ctx context.Context, cli remoteLister, ownerToken string, legacyOwnerUID string, clusterName string) []CleanupRef {
	list := &velerov1.RestoreList{}
	if err := cli.List(ctx, list, client.InNamespace(common.VeleroNamespace)); err != nil {
		return []CleanupRef{{
			Kind:             "Restore",
			Cluster:          clusterName,
			RelationCode:     relationFinalizerVeleroRestore,
			Strategy:         cleanupStrategyDelete,
			Resolved:         false,
			Selector:         cleanupSelector(ownerToken, relationFinalizerVeleroRestore),
			UnresolvedReason: err.Error(),
		}}
	}

	matched := make([]*velerov1.Restore, 0)
	for i := range list.Items {
		r := &list.Items[i]
		if matchesCleanupLabels(r.Labels, ownerToken, relationFinalizerVeleroRestore) || matchesLegacyUIDLabel(r.Labels, "testudo.softcdata.com/app-restore-uid", legacyOwnerUID) {
			matched = append(matched, r)
		}
	}

	if len(matched) == 0 {
		return nil
	}

	out := make([]CleanupRef, 0, len(matched))
	for _, r := range matched {
		out = append(out, CleanupRef{
			Kind:         "Restore",
			Name:         r.Name,
			Namespace:    r.Namespace,
			UID:          string(r.UID),
			Cluster:      clusterName,
			RelationCode: relationFinalizerVeleroRestore,
			Strategy:     cleanupStrategyDelete,
			Resolved:     true,
		})
	}
	return out
}

func resolveResourceModifierConfigMaps(ctx context.Context, cli remoteLister, ownerToken string, legacyOwnerUID string, clusterName string) []CleanupRef {
	list := &corev1.ConfigMapList{}
	if err := cli.List(ctx, list, client.InNamespace(common.VeleroNamespace)); err != nil {
		return []CleanupRef{{
			Kind:             "ConfigMap",
			Cluster:          clusterName,
			RelationCode:     relationFinalizerResourceModifierConfigMap,
			Strategy:         cleanupStrategyDelete,
			Resolved:         false,
			Selector:         cleanupSelector(ownerToken, relationFinalizerResourceModifierConfigMap),
			UnresolvedReason: err.Error(),
		}}
	}

	matched := make([]*corev1.ConfigMap, 0)
	for i := range list.Items {
		cm := &list.Items[i]
		if matchesCleanupLabels(cm.Labels, ownerToken, relationFinalizerResourceModifierConfigMap) || matchesLegacyUIDLabel(cm.Labels, "apprestore.testudo.softcdata.com/uid", legacyOwnerUID) {
			matched = append(matched, cm)
		}
	}

	if len(matched) == 0 {
		return nil
	}

	out := make([]CleanupRef, 0, len(matched))
	for _, cm := range matched {
		out = append(out, CleanupRef{
			Kind:         "ConfigMap",
			Name:         cm.Name,
			Namespace:    cm.Namespace,
			UID:          string(cm.UID),
			Cluster:      clusterName,
			RelationCode: relationFinalizerResourceModifierConfigMap,
			Strategy:     cleanupStrategyDelete,
			Resolved:     true,
		})
	}
	return out
}

func matchesCleanupLabels(labels map[string]string, ownerToken string, relation string) bool {
	if len(labels) == 0 {
		return false
	}
	if labels[labelCleanupManagedBy] != cleanupManagedByValueOperator {
		return false
	}
	if ownerToken != "" && labels[labelCleanupOwnerToken] != ownerToken {
		return false
	}
	if relation != "" && labels[labelCleanupRelation] != relation {
		return false
	}
	return true
}

func matchesLegacyUIDLabel(labels map[string]string, key string, uid string) bool {
	if len(labels) == 0 {
		return false
	}
	if strings.TrimSpace(key) == "" || strings.TrimSpace(uid) == "" {
		return false
	}
	return labels[key] == uid
}
