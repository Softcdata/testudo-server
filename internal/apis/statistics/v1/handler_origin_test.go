package v1

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-operator/pkg/clientset/versioned/fake"
	informers "github.com/softcdata/testudo-operator/pkg/informers/externalversions"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/kube"
	"github.com/softcdata/testudo-server/internal/transport"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func newMockStatisticsHandler(objects ...runtime.Object) *StatisticsHandler {
	fakeClient := fake.NewSimpleClientset(objects...)
	informerFactory := informers.NewSharedInformerFactory(fakeClient, 0)

	_ = informerFactory.Disaster().V1().BackupRestoreStatisticses().Informer()
	_ = informerFactory.Disaster().V1().AppBackups().Informer()
	_ = informerFactory.Disaster().V1().AppRestores().Informer()
	_ = informerFactory.Disaster().V1().DisasterPolicies().Informer()

	stopCh := make(chan struct{})
	informerFactory.Start(stopCh)
	informerFactory.WaitForCacheSync(stopCh)

	kc := &kube.KubeClient{
		DisasterClient:  fakeClient,
		InformerFactory: informerFactory,
	}

	h := server.Default()
	rg := h.Group("/v1")

	return NewStatisticsHandler(kc, rg)
}

func testStatsObject(name, ownerKind, scopeKind, scopeName string, statistics dapisv1.StatisticsData) *dapisv1.BackupRestoreStatistics {
	return &dapisv1.BackupRestoreStatistics{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: common.DisasterSystemNamespace,
			Labels: map[string]string{
				"testudo.softcdata.com/owner-kind": ownerKind,
			},
		},
		Spec: dapisv1.BackupRestoreStatisticsSpec{
			ScopeType: dapisv1.ScopeTypeApp,
			ScopeRef: dapisv1.ScopeReference{
				Kind:      scopeKind,
				Name:      scopeName,
				Namespace: common.DisasterSystemNamespace,
			},
		},
		Status: dapisv1.BackupRestoreStatisticsStatus{
			Statistics: statistics,
		},
	}
}

func TestBackupStatistics_OriginFilter(t *testing.T) {
	controllerTrue := true

	userBackup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "user-backup",
			Namespace: common.DisasterSystemNamespace,
		},
	}
	instanceBackupByLabel := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "instance-backup-label",
			Namespace: common.DisasterSystemNamespace,
			Labels: map[string]string{
				appResourceOriginLabelKey: appResourceOriginDisasterInstance,
			},
		},
	}
	instanceBackupByOwner := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "instance-backup-owner",
			Namespace: common.DisasterSystemNamespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:       "ResourceSync",
					Name:       "rs-demo",
					Controller: &controllerTrue,
				},
			},
		},
	}

	statsUser := testStatsObject("stats-user-backup", "AppBackup", "AppBackup", "user-backup", dapisv1.StatisticsData{
		Total:      2,
		Completed:  1,
		InProgress: 1,
	})
	statsInstanceByLabel := testStatsObject("stats-instance-backup-label", "AppBackup", "AppBackup", "instance-backup-label", dapisv1.StatisticsData{
		Total:     4,
		Completed: 3,
		Failed:    1,
	})
	statsInstanceByOwner := testStatsObject("stats-instance-backup-owner", "AppBackup", "AppBackup", "instance-backup-owner", dapisv1.StatisticsData{
		Total:    3,
		Failed:   2,
		Canceled: 1,
	})

	h := newMockStatisticsHandler(
		userBackup,
		instanceBackupByLabel,
		instanceBackupByOwner,
		statsUser,
		statsInstanceByLabel,
		statsInstanceByOwner,
	)

	type statsResponse struct {
		Code int           `json:"code"`
		Data StatisticsDTO `json:"data"`
	}

	tests := []struct {
		name      string
		uri       string
		wantCode  int
		wantStats StatisticsDTO
	}{
		{
			name:     "default user filter",
			uri:      "/backups",
			wantCode: consts.StatusOK,
			wantStats: StatisticsDTO{
				Total:      2,
				InProgress: 1,
				Completed:  1,
			},
		},
		{
			name:     "origin instance",
			uri:      "/backups?origin=instance",
			wantCode: consts.StatusOK,
			wantStats: StatisticsDTO{
				Total:     7,
				Completed: 3,
				Failed:    3,
				Canceled:  1,
			},
		},
		{
			name:     "origin all",
			uri:      "/backups?origin=all",
			wantCode: consts.StatusOK,
			wantStats: StatisticsDTO{
				Total:      9,
				InProgress: 1,
				Completed:  4,
				Failed:     3,
				Canceled:   1,
			},
		},
		{
			name:     "invalid origin",
			uri:      "/backups?origin=oops",
			wantCode: consts.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := app.NewContext(16)
			ctx.Request.SetRequestURI(tc.uri)

			h.GetBackupStatistics(context.Background(), ctx)
			assert.Equal(t, tc.wantCode, ctx.Response.StatusCode())

			if tc.wantCode != consts.StatusOK {
				var errorResp struct {
					Code int `json:"code"`
				}
				err := json.Unmarshal(ctx.Response.Body(), &errorResp)
				assert.NoError(t, err)
				assert.Equal(t, transport.CodeBadRequest, errorResp.Code)
				return
			}

			var resp statsResponse
			err := json.Unmarshal(ctx.Response.Body(), &resp)
			assert.NoError(t, err)
			assert.Equal(t, transport.CodeOK, resp.Code)
			assert.Equal(t, tc.wantStats.Total, resp.Data.Total)
			assert.Equal(t, tc.wantStats.InProgress, resp.Data.InProgress)
			assert.Equal(t, tc.wantStats.Completed, resp.Data.Completed)
			assert.Equal(t, tc.wantStats.Failed, resp.Data.Failed)
			assert.Equal(t, tc.wantStats.Canceled, resp.Data.Canceled)
		})
	}
}

func TestRestoreStatistics_OriginFilter(t *testing.T) {
	controllerTrue := true

	userRestore := &dapisv1.AppRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "user-restore",
			Namespace: common.DisasterSystemNamespace,
		},
	}
	instanceRestoreByOwner := &dapisv1.AppRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rec-ds-instance-001",
			Namespace: common.DisasterSystemNamespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:       "DataSync",
					Name:       "ds-demo",
					Controller: &controllerTrue,
				},
			},
		},
	}
	// 历史兼容场景：Drill 自动创建的恢复被误标为 user
	drillRestoreWrongLabel := &dapisv1.AppRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "drr-drill-demo-123456",
			Namespace: common.DisasterSystemNamespace,
			Labels: map[string]string{
				appResourceOriginLabelKey: appResourceOriginUser,
				"testudo.softcdata.com/drill": "drill-demo",
			},
		},
	}

	statsUser := testStatsObject("stats-user-restore", "AppRestore", "AppRestore", "user-restore", dapisv1.StatisticsData{
		Total:     1,
		Completed: 1,
	})
	statsInstanceByOwner := testStatsObject("stats-instance-restore-owner", "AppRestore", "AppRestore", "rec-ds-instance-001", dapisv1.StatisticsData{
		Total:      3,
		InProgress: 1,
		Completed:  2,
	})
	statsDrillWrongLabel := testStatsObject("stats-drill-restore", "AppRestore", "AppRestore", "drr-drill-demo-123456", dapisv1.StatisticsData{
		Total:  2,
		Failed: 2,
	})

	h := newMockStatisticsHandler(
		userRestore,
		instanceRestoreByOwner,
		drillRestoreWrongLabel,
		statsUser,
		statsInstanceByOwner,
		statsDrillWrongLabel,
	)

	type statsResponse struct {
		Code int           `json:"code"`
		Data StatisticsDTO `json:"data"`
	}

	tests := []struct {
		name      string
		uri       string
		wantStats StatisticsDTO
	}{
		{
			name: "default user filter",
			uri:  "/restores",
			wantStats: StatisticsDTO{
				Total:     1,
				Completed: 1,
			},
		},
		{
			name: "origin instance",
			uri:  "/restores?origin=instance",
			wantStats: StatisticsDTO{
				Total:      5,
				InProgress: 1,
				Completed:  2,
				Failed:     2,
			},
		},
		{
			name: "origin all",
			uri:  "/restores?origin=all",
			wantStats: StatisticsDTO{
				Total:      6,
				InProgress: 1,
				Completed:  3,
				Failed:     2,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := app.NewContext(16)
			ctx.Request.SetRequestURI(tc.uri)

			h.GetRestoreStatistics(context.Background(), ctx)
			assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

			var resp statsResponse
			err := json.Unmarshal(ctx.Response.Body(), &resp)
			assert.NoError(t, err)
			assert.Equal(t, transport.CodeOK, resp.Code)
			assert.Equal(t, tc.wantStats.Total, resp.Data.Total)
			assert.Equal(t, tc.wantStats.InProgress, resp.Data.InProgress)
			assert.Equal(t, tc.wantStats.Completed, resp.Data.Completed)
			assert.Equal(t, tc.wantStats.Failed, resp.Data.Failed)
		})
	}
}

func TestAutoBackupExecutionSummary(t *testing.T) {
	now := time.Now()
	autoPolicy := &dapisv1.DisasterPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "auto-policy",
			Namespace: common.DisasterSystemNamespace,
		},
		Spec: dapisv1.DisasterPolicySpec{
			Type: dapisv1.PolicyTypeAutoBackup,
		},
	}
	dataSyncPolicy := &dapisv1.DisasterPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sync-policy",
			Namespace: common.DisasterSystemNamespace,
		},
		Spec: dapisv1.DisasterPolicySpec{
			Type: dapisv1.PolicyTypeDataSync,
		},
	}
	tm := func(t time.Time) *metav1.Time {
		out := metav1.NewTime(t)
		return &out
	}
	autoBackup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "auto-backup",
			Namespace: common.DisasterSystemNamespace,
		},
		Spec: dapisv1.AppBackupSpec{
			DisasterPolicy: "auto-policy",
		},
		Status: dapisv1.AppBackupStatus{
			History: []dapisv1.BackupRecord{
				{
					Name:                "success-recent",
					ManagedStatus:       dapisv1.LastBackupStatusCompleted,
					CompletionTimestamp: tm(now.AddDate(0, 0, -1)),
				},
				{
					Name:                "failed-recent",
					ManagedStatus:       dapisv1.LastBackupStatusFailed,
					CompletionTimestamp: tm(now.AddDate(0, 0, -2)),
				},
				{
					Name:                "failed-older-than-7d",
					ManagedStatus:       dapisv1.LastBackupStatusFailed,
					CompletionTimestamp: tm(now.AddDate(0, 0, -20)),
				},
			},
		},
	}
	syncBackup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sync-backup",
			Namespace: common.DisasterSystemNamespace,
		},
		Spec: dapisv1.AppBackupSpec{
			DisasterPolicy: "sync-policy",
		},
		Status: dapisv1.AppBackupStatus{
			History: []dapisv1.BackupRecord{
				{
					Name:                "sync-failed",
					ManagedStatus:       dapisv1.LastBackupStatusFailed,
					CompletionTimestamp: tm(now.AddDate(0, 0, -1)),
				},
			},
		},
	}
	manualBackup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "manual-backup",
			Namespace: common.DisasterSystemNamespace,
		},
		Status: dapisv1.AppBackupStatus{
			History: []dapisv1.BackupRecord{
				{
					Name:                "manual-success",
					ManagedStatus:       dapisv1.LastBackupStatusCompleted,
					CompletionTimestamp: tm(now.AddDate(0, 0, -1)),
				},
			},
		},
	}

	h := newMockStatisticsHandler(autoPolicy, dataSyncPolicy, autoBackup, syncBackup, manualBackup)

	type summaryResponse struct {
		Code int                           `json:"code"`
		Data AutoBackupExecutionSummaryDTO `json:"data"`
	}

	t.Run("7d only counts recent AutoBackup success and failure", func(t *testing.T) {
		ctx := app.NewContext(16)
		ctx.Request.SetRequestURI("/autobackups/execution-summary?period=7d")

		h.GetAutoBackupExecutionSummary(context.Background(), ctx)
		assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

		var resp summaryResponse
		err := json.Unmarshal(ctx.Response.Body(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, transport.CodeOK, resp.Code)
		assert.Equal(t, "7d", resp.Data.Period)
		assert.Equal(t, "7d", resp.Data.Range)
		assert.Equal(t, int32(2), resp.Data.Total)
		assert.Equal(t, int32(1), resp.Data.Success.Count)
		assert.Equal(t, int32(1), resp.Data.Failed.Count)
		assert.Equal(t, int32(50), resp.Data.Success.Percent)
		assert.Equal(t, int32(50), resp.Data.Failed.Percent)
	})

	t.Run("30d includes older AutoBackup record", func(t *testing.T) {
		ctx := app.NewContext(16)
		ctx.Request.SetRequestURI("/autobackups/execution-summary?period=30d")

		h.GetAutoBackupExecutionSummary(context.Background(), ctx)
		assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

		var resp summaryResponse
		err := json.Unmarshal(ctx.Response.Body(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, int32(3), resp.Data.Total)
		assert.Equal(t, int32(1), resp.Data.Success.Count)
		assert.Equal(t, int32(2), resp.Data.Failed.Count)
		assert.Equal(t, int32(33), resp.Data.Success.Percent)
		assert.Equal(t, int32(67), resp.Data.Failed.Percent)
	})

	t.Run("90d includes the same AutoBackup records in fixture", func(t *testing.T) {
		ctx := app.NewContext(16)
		ctx.Request.SetRequestURI("/autobackups/execution-summary?period=90d")

		h.GetAutoBackupExecutionSummary(context.Background(), ctx)
		assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

		var resp summaryResponse
		err := json.Unmarshal(ctx.Response.Body(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "90d", resp.Data.Period)
		assert.Equal(t, "90d", resp.Data.Range)
		assert.Equal(t, int32(3), resp.Data.Total)
		assert.Equal(t, int32(1), resp.Data.Success.Count)
		assert.Equal(t, int32(2), resp.Data.Failed.Count)
	})

	t.Run("empty namespace returns zero buckets", func(t *testing.T) {
		ctx := app.NewContext(16)
		ctx.Request.SetRequestURI("/autobackups/execution-summary?period=7d&namespace=empty")
		ctx.Request.URI().SetQueryString("period=7d&namespace=empty")

		h.GetAutoBackupExecutionSummary(context.Background(), ctx)
		assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

		var resp summaryResponse
		err := json.Unmarshal(ctx.Response.Body(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, int32(0), resp.Data.Total)
		assert.Equal(t, int32(0), resp.Data.Success.Count)
		assert.Equal(t, int32(0), resp.Data.Failed.Count)
		assert.Equal(t, int32(0), resp.Data.Success.Percent)
		assert.Equal(t, int32(0), resp.Data.Failed.Percent)
	})

	t.Run("invalid period rejected", func(t *testing.T) {
		ctx := app.NewContext(16)
		ctx.Request.SetRequestURI("/autobackups/execution-summary?period=365d")

		h.GetAutoBackupExecutionSummary(context.Background(), ctx)
		assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
		assert.Contains(t, string(ctx.Response.Body()), "invalid period")
	})

	t.Run("range remains backward compatible", func(t *testing.T) {
		ctx := app.NewContext(16)
		ctx.Request.SetRequestURI("/autobackups/execution-summary?range=7d")

		h.GetAutoBackupExecutionSummary(context.Background(), ctx)
		assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

		var resp summaryResponse
		err := json.Unmarshal(ctx.Response.Body(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "7d", resp.Data.Period)
	})
}
