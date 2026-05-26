package deletioncheck

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/stretchr/testify/assert"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type stubRemoteLister struct {
	schedules  []velerov1.Schedule
	backups    []velerov1.Backup
	restores   []velerov1.Restore
	configMaps []corev1.ConfigMap
}

func (s *stubRemoteLister) List(ctx context.Context, list ctrlclient.ObjectList, opts ...ctrlclient.ListOption) error {
	lo := &ctrlclient.ListOptions{}
	for _, opt := range opts {
		opt.ApplyToList(lo)
	}
	ns := lo.Namespace

	switch l := list.(type) {
	case *velerov1.ScheduleList:
		l.Items = nil
		for _, item := range s.schedules {
			if ns == "" || item.Namespace == ns {
				l.Items = append(l.Items, item)
			}
		}
		return nil
	case *velerov1.BackupList:
		l.Items = nil
		for _, item := range s.backups {
			if ns == "" || item.Namespace == ns {
				l.Items = append(l.Items, item)
			}
		}
		return nil
	case *velerov1.RestoreList:
		l.Items = nil
		for _, item := range s.restores {
			if ns == "" || item.Namespace == ns {
				l.Items = append(l.Items, item)
			}
		}
		return nil
	case *corev1.ConfigMapList:
		l.Items = nil
		for _, item := range s.configMaps {
			if ns == "" || item.Namespace == ns {
				l.Items = append(l.Items, item)
			}
		}
		return nil
	default:
		return errors.New("unsupported list type")
	}
}

func decodeDeletionCheckResponse(t *testing.T, ctx *app.RequestContext) DeletionCheckResponse {
	t.Helper()
	var resp struct {
		Code int                   `json:"code"`
		Data DeletionCheckResponse `json:"data"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, 0, resp.Code)
	return resp.Data
}

func TestDeletionCheck_AppBackupCleanupPlan_ResolvedRemote(t *testing.T) {
	ns := common.DisasterSystemNamespace

	ab := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ab-1",
			Namespace: ns,
			UID:       types.UID("appbackup-cleanup-uid"),
		},
		Spec: dapisv1.AppBackupSpec{
			Cluster: "cluster-a",
		},
	}

	ownerToken := buildDependencyToken(string(ab.UID))
	remoteSchedule := velerov1.Schedule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-schedule-ab-1",
			Namespace: common.VeleroNamespace,
			UID:       types.UID("velero-schedule-uid"),
			Labels: map[string]string{
				labelCleanupManagedBy:  cleanupManagedByValueOperator,
				labelCleanupOwnerToken: ownerToken,
				labelCleanupRelation:   relationFinalizerVeleroSchedule,
				labelCleanupStrategy:   cleanupStrategyDelete,
				// Legacy labels for compatibility
				"testudo.softcdata.com/app-backup-uid": string(ab.UID),
			},
		},
	}
	remoteBackup := velerov1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bak-ab-1-1",
			Namespace: common.VeleroNamespace,
			UID:       types.UID("velero-backup-uid"),
			Labels: map[string]string{
				labelCleanupManagedBy:  cleanupManagedByValueOperator,
				labelCleanupOwnerToken: ownerToken,
				labelCleanupRelation:   relationFinalizerVeleroBackup,
				labelCleanupStrategy:   cleanupStrategyDeleteRequest,
				// Legacy labels for compatibility
				"testudo.softcdata.com/app-backup-uid": string(ab.UID),
			},
		},
	}
	remote := &stubRemoteLister{
		schedules: []velerov1.Schedule{remoteSchedule},
		backups:   []velerov1.Backup{remoteBackup},
	}

	h := newMockHandler(ab)
	h.getRemoteClient = func(ctx context.Context, clusterName string) (remoteLister, error) {
		assert.Equal(t, "cluster-a", clusterName)
		return remote, nil
	}

	ctx := app.NewContext(16)
	req := DeletionCheckRequest{ResourceKind: "AppBackup", Name: "ab-1", Namespace: ns}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.check(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	data := decodeDeletionCheckResponse(t, ctx)
	assert.True(t, data.CanDelete)
	assert.True(t, data.CleanupPlan.HasCleanup)
	assert.NotEmpty(t, data.CleanupPlan.FinalizerCleanup)

	var foundSchedule, foundBackup bool
	for _, item := range data.CleanupPlan.FinalizerCleanup {
		if item.Kind == "Schedule" {
			foundSchedule = true
			assert.Equal(t, relationFinalizerVeleroSchedule, item.RelationCode)
			assert.Equal(t, "cluster-a", item.Cluster)
			assert.True(t, item.Resolved)
			assert.Equal(t, remoteSchedule.Name, item.Name)
		}
		if item.Kind == "Backup" {
			foundBackup = true
			assert.Equal(t, relationFinalizerVeleroBackup, item.RelationCode)
			assert.Equal(t, cleanupStrategyDeleteRequest, item.Strategy)
			assert.Equal(t, "cluster-a", item.Cluster)
			assert.True(t, item.Resolved)
			assert.Equal(t, remoteBackup.Name, item.Name)
		}
	}
	assert.True(t, foundSchedule)
	assert.True(t, foundBackup)
}

func TestDeletionCheck_AppBackupCleanupPlan_UnresolvedRemote(t *testing.T) {
	ns := common.DisasterSystemNamespace

	ab := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ab-2",
			Namespace: ns,
			UID:       types.UID("appbackup-cleanup-uid-2"),
		},
		Spec: dapisv1.AppBackupSpec{
			Cluster: "cluster-a",
		},
	}

	h := newMockHandler(ab)
	h.getRemoteClient = func(ctx context.Context, clusterName string) (remoteLister, error) {
		return nil, errors.New("remote cluster unreachable")
	}

	ctx := app.NewContext(16)
	req := DeletionCheckRequest{ResourceKind: "AppBackup", Name: "ab-2", Namespace: ns}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.check(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	data := decodeDeletionCheckResponse(t, ctx)
	assert.True(t, data.CleanupPlan.HasCleanup)
	assert.Len(t, data.CleanupPlan.FinalizerCleanup, 2) // schedule + backup skeleton
	for _, item := range data.CleanupPlan.FinalizerCleanup {
		assert.False(t, item.Resolved)
		assert.Equal(t, "cluster-a", item.Cluster)
		assert.NotEmpty(t, item.UnresolvedReason)
		assert.NotEmpty(t, item.Selector)
	}
}

func TestDeletionCheck_DisasterInstance_ChildResourcesNotInUpstream_ButInCleanupPlan(t *testing.T) {
	ns := common.DisasterSystemNamespace

	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "instance-a",
			Namespace: ns,
			UID:       types.UID("instance-cleanup-uid"),
		},
	}
	instToken := buildDependencyToken(string(inst.UID))
	depKey := dependencyToLabelKey(instToken)

	ds := &dapisv1.DataSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ds-1",
			Namespace: ns,
			UID:       types.UID("datasync-uid-1"),
			Labels: map[string]string{
				depKey: "spec.instance",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "DisasterInstance",
					Name: inst.Name,
					UID:  inst.UID,
				},
			},
		},
	}
	rs := &dapisv1.ResourceSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rs-1",
			Namespace: ns,
			UID:       types.UID("resourcesync-uid-1"),
			Labels: map[string]string{
				depKey: "spec.instance",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "DisasterInstance",
					Name: inst.Name,
					UID:  inst.UID,
				},
			},
		},
	}
	op := &dapisv1.DisasterOperation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "op-1",
			Namespace: ns,
			UID:       types.UID("operation-uid-1"),
			Labels: map[string]string{
				depKey: "spec.instanceName",
			},
		},
		Spec: dapisv1.DisasterOperationSpec{
			InstanceName: inst.Name,
		},
	}

	h := newMockHandler(inst, ds, rs, op)

	ctx := app.NewContext(16)
	req := DeletionCheckRequest{ResourceKind: "DisasterInstance", Name: inst.Name, Namespace: ns}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.check(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	data := decodeDeletionCheckResponse(t, ctx)
	assert.True(t, data.CanDelete)
	assert.Empty(t, data.Upstream, "child resources must not block deletion via upstream")
	assert.True(t, data.CleanupPlan.HasCleanup)

	var hasDS, hasRS, hasOp bool
	for _, item := range data.CleanupPlan.CascadeCleanup {
		switch item.Kind {
		case kindDataSync:
			hasDS = true
		case kindResourceSync:
			hasRS = true
		case kindDisasterOperation:
			hasOp = true
		}
	}
	assert.True(t, hasDS)
	assert.True(t, hasRS)
	assert.True(t, hasOp)
}

func TestDeletionCheck_DisasterOperation_UpstreamIncludesDrillOwner(t *testing.T) {
	ns := common.DisasterSystemNamespace

	drill := &dapisv1.DisasterDrill{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "drill-1",
			Namespace: ns,
			UID:       types.UID("drill-uid-1"),
		},
	}
	op := &dapisv1.DisasterOperation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "op-owned",
			Namespace: ns,
			UID:       types.UID("operation-owned-uid-1"),
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "DisasterDrill",
					Name: drill.Name,
					UID:  drill.UID,
				},
			},
		},
	}

	h := newMockHandler(drill, op)

	ctx := app.NewContext(16)
	req := DeletionCheckRequest{ResourceKind: "DisasterOperation", Name: op.Name, Namespace: ns}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.check(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	data := decodeDeletionCheckResponse(t, ctx)
	assert.False(t, data.CanDelete)
	assert.NotEmpty(t, data.Upstream)

	var found bool
	for _, u := range data.Upstream {
		if u.Kind == "DisasterDrill" && u.Name == "drill-1" {
			found = true
		}
	}
	assert.True(t, found)
}

func TestDeletionCheck_DisasterDrill_CascadeIncludesOperation_NotUpstream(t *testing.T) {
	ns := common.DisasterSystemNamespace

	drill := &dapisv1.DisasterDrill{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "drill-2",
			Namespace: ns,
			UID:       types.UID("drill-uid-2"),
		},
	}
	drillToken := buildDependencyToken(string(drill.UID))
	depKey := dependencyToLabelKey(drillToken)

	op := &dapisv1.DisasterOperation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "op-drill-child",
			Namespace: ns,
			UID:       types.UID("operation-drill-child-uid"),
			Labels: map[string]string{
				depKey: "ownerReference.disasterOperation",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "DisasterDrill",
					Name: drill.Name,
					UID:  drill.UID,
				},
			},
		},
	}

	h := newMockHandler(drill, op)

	ctx := app.NewContext(16)
	req := DeletionCheckRequest{ResourceKind: "DisasterDrill", Name: drill.Name, Namespace: ns}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.check(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	data := decodeDeletionCheckResponse(t, ctx)
	assert.True(t, data.CanDelete)
	assert.Empty(t, data.Upstream, "drill child DisasterOperation must not appear in upstream")

	var hasOp bool
	for _, item := range data.CleanupPlan.CascadeCleanup {
		if item.Kind == kindDisasterOperation && item.Name == op.Name {
			hasOp = true
			break
		}
	}
	assert.True(t, hasOp)
}
