package v1

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/transport"
	"github.com/stretchr/testify/assert"
	"github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type taskProgressResponse struct {
	Code int                  `json:"code"`
	Data TaskProgressTrendDTO `json:"data"`
}

func taskProgressTestTime(t *testing.T, dayOffset int) *metav1.Time {
	t.Helper()
	location, err := time.LoadLocation(taskProgressDefaultTimezone)
	assert.NoError(t, err)
	now := time.Now().In(location)
	value := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, location).AddDate(0, 0, dayOffset)
	if !value.Before(now) {
		value = value.AddDate(0, 0, -1)
	}
	out := metav1.NewTime(value)
	return &out
}

func taskProgressDate(t *testing.T, value *metav1.Time) string {
	t.Helper()
	location, err := time.LoadLocation(taskProgressDefaultTimezone)
	assert.NoError(t, err)
	return value.In(location).Format("2006-01-02")
}

func taskProgressBucketByDate(data TaskProgressTrendDTO) map[string]TaskProgressBucketDTO {
	out := make(map[string]TaskProgressBucketDTO, len(data.Buckets))
	for _, bucket := range data.Buckets {
		out[bucket.Date] = bucket
	}
	return out
}

func taskProgressSourceByScope(data TaskProgressTrendDTO) map[string]TaskProgressSourceDTO {
	out := make(map[string]TaskProgressSourceDTO, len(data.Sources))
	for _, source := range data.Sources {
		out[source.Scope] = source
	}
	return out
}

func TestTaskProgressTrendBackup(t *testing.T) {
	controllerTrue := true
	completedAt := taskProgressTestTime(t, -1)
	failedAt := taskProgressTestTime(t, -2)
	canceledAt := taskProgressTestTime(t, -3)
	runningAt := taskProgressTestTime(t, -4)

	userBackup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "user-backup",
			Namespace: common.DisasterSystemNamespace,
		},
		Spec: dapisv1.AppBackupSpec{Cluster: "cluster-a"},
		Status: dapisv1.AppBackupStatus{
			History: []dapisv1.BackupRecord{
				{
					Name:                "user-completed",
					ManagedStatus:       dapisv1.LastBackupStatusCompleted,
					CompletionTimestamp: completedAt,
				},
				{
					Name:                "user-canceled",
					ManagedStatus:       dapisv1.LastBackupStatusCanceled,
					CompletionTimestamp: canceledAt,
				},
			},
		},
	}
	disasterBackup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rs-backup",
			Namespace: common.DisasterSystemNamespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:       "ResourceSync",
					Name:       "rs-demo",
					Controller: &controllerTrue,
				},
			},
		},
		Spec: dapisv1.AppBackupSpec{Cluster: "cluster-a"},
		Status: dapisv1.AppBackupStatus{
			History: []dapisv1.BackupRecord{
				{
					Name:                "disaster-failed",
					Phase:               "PartiallyFailed",
					CompletionTimestamp: failedAt,
				},
				{
					Name:           "disaster-running",
					ManagedStatus:  dapisv1.LastBackupStatusInProgress,
					StartTimestamp: runningAt,
				},
			},
		},
	}
	otherClusterBackup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "other-cluster-backup",
			Namespace: common.DisasterSystemNamespace,
		},
		Spec: dapisv1.AppBackupSpec{Cluster: "cluster-b"},
		Status: dapisv1.AppBackupStatus{
			History: []dapisv1.BackupRecord{
				{
					Name:                "ignored-completed",
					ManagedStatus:       dapisv1.LastBackupStatusCompleted,
					CompletionTimestamp: completedAt,
				},
			},
		},
	}

	h := newMockStatisticsHandler(userBackup, disasterBackup, otherClusterBackup)

	t.Run("all scope returns backup buckets and sources", func(t *testing.T) {
		ctx := app.NewContext(16)
		ctx.Request.SetRequestURI("/tasks/progress?type=backup&scope=all&range=7d&timezone=Asia/Shanghai&cluster=cluster-a")

		h.GetTaskProgressTrend(context.Background(), ctx)
		assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

		var resp taskProgressResponse
		err := json.Unmarshal(ctx.Response.Body(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, transport.CodeOK, resp.Code)
		assert.Equal(t, taskProgressTypeBackup, resp.Data.Type)
		assert.Equal(t, taskProgressScopeAll, resp.Data.Scope)
		assert.Len(t, resp.Data.Buckets, 7)
		assert.Equal(t, int32(4), resp.Data.Summary.Total)
		assert.Equal(t, int32(1), resp.Data.Summary.Completed)
		assert.Equal(t, int32(1), resp.Data.Summary.Failed)
		assert.Equal(t, int32(1), resp.Data.Summary.Canceled)
		assert.Equal(t, int32(1), resp.Data.Summary.InProgress)
		assert.Equal(t, "备份成功", resp.Data.Series[0].Label)
		assert.Equal(t, "备份失败", resp.Data.Series[1].Label)

		buckets := taskProgressBucketByDate(resp.Data)
		assert.Equal(t, int32(1), buckets[taskProgressDate(t, completedAt)].Completed)
		assert.Equal(t, int32(1), buckets[taskProgressDate(t, failedAt)].Failed)
		assert.Equal(t, int32(1), buckets[taskProgressDate(t, canceledAt)].Canceled)
		assert.Equal(t, int32(1), buckets[taskProgressDate(t, runningAt)].InProgress)

		sources := taskProgressSourceByScope(resp.Data)
		assert.Equal(t, int32(2), sources[taskProgressScopeApp].Total)
		assert.Equal(t, int32(1), sources[taskProgressScopeApp].Completed)
		assert.Equal(t, int32(2), sources[taskProgressScopeDisaster].Total)
		assert.Equal(t, int32(1), sources[taskProgressScopeDisaster].Failed)
	})

	t.Run("app scope excludes disaster backups", func(t *testing.T) {
		ctx := app.NewContext(16)
		ctx.Request.SetRequestURI("/tasks/progress?type=backup&scope=app&range=7d&timezone=Asia/Shanghai&cluster=cluster-a")

		h.GetTaskProgressTrend(context.Background(), ctx)
		assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

		var resp taskProgressResponse
		err := json.Unmarshal(ctx.Response.Body(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, int32(2), resp.Data.Summary.Total)
		assert.Equal(t, int32(1), resp.Data.Summary.Completed)
		assert.Equal(t, int32(0), resp.Data.Summary.Failed)
		assert.Len(t, resp.Data.Sources, 1)
		assert.Equal(t, taskProgressScopeApp, resp.Data.Sources[0].Scope)
	})

	t.Run("empty namespace returns zero buckets", func(t *testing.T) {
		ctx := app.NewContext(16)
		ctx.Request.SetRequestURI("/tasks/progress?type=backup&range=7d&timezone=Asia/Shanghai&namespace=empty")

		h.GetTaskProgressTrend(context.Background(), ctx)
		assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

		var resp taskProgressResponse
		err := json.Unmarshal(ctx.Response.Body(), &resp)
		assert.NoError(t, err)
		assert.Len(t, resp.Data.Buckets, 7)
		assert.Equal(t, int32(0), resp.Data.Summary.Total)
		for _, bucket := range resp.Data.Buckets {
			assert.Equal(t, int32(0), bucket.Total)
		}
	})
}

func TestTaskProgressTrendRestore(t *testing.T) {
	controllerTrue := true
	completedAt := taskProgressTestTime(t, -1)
	failedAt := taskProgressTestTime(t, -2)
	canceledAt := taskProgressTestTime(t, -3)
	runningAt := taskProgressTestTime(t, -4)

	userRestore := &dapisv1.AppRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "user-restore",
			Namespace:         common.DisasterSystemNamespace,
			CreationTimestamp: *completedAt,
		},
		Spec: dapisv1.AppRestoreSpec{Cluster: "cluster-a"},
		Status: dapisv1.AppRestoreStatus{
			Status: dapisv1.PhaseSucceeded,
			RestoreStatus: v1.RestoreStatus{
				CompletionTimestamp: completedAt,
			},
		},
	}
	disasterRestore := &dapisv1.AppRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "rec-ds-restore",
			Namespace:         common.DisasterSystemNamespace,
			CreationTimestamp: *failedAt,
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:       "DataSync",
					Name:       "ds-demo",
					Controller: &controllerTrue,
				},
			},
		},
		Spec: dapisv1.AppRestoreSpec{Cluster: "cluster-a"},
		Status: dapisv1.AppRestoreStatus{
			Status: dapisv1.PhaseFailed,
			RestoreStatus: v1.RestoreStatus{
				CompletionTimestamp: failedAt,
			},
		},
	}
	drillRestore := &dapisv1.AppRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "drr-drill-demo",
			Namespace:         common.DisasterSystemNamespace,
			CreationTimestamp: *canceledAt,
			Labels: map[string]string{
				appResourceOriginLabelKey: appResourceOriginUser,
				"testudo.softcdata.com/drill": "drill-demo",
			},
		},
		Spec: dapisv1.AppRestoreSpec{Cluster: "cluster-a"},
		Status: dapisv1.AppRestoreStatus{
			Status: dapisv1.PhaseCancelled,
			RestoreStatus: v1.RestoreStatus{
				CompletionTimestamp: canceledAt,
			},
		},
	}
	runningRestore := &dapisv1.AppRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "running-restore",
			Namespace:         common.DisasterSystemNamespace,
			CreationTimestamp: *runningAt,
		},
		Spec: dapisv1.AppRestoreSpec{Cluster: "cluster-a"},
		Status: dapisv1.AppRestoreStatus{
			Status: dapisv1.PhaseRestoring,
			RestoreStatus: v1.RestoreStatus{
				StartTimestamp: runningAt,
			},
		},
	}
	otherClusterRestore := &dapisv1.AppRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "other-cluster-restore",
			Namespace:         common.DisasterSystemNamespace,
			CreationTimestamp: *completedAt,
		},
		Spec: dapisv1.AppRestoreSpec{Cluster: "cluster-b"},
		Status: dapisv1.AppRestoreStatus{
			Status: dapisv1.PhaseSucceeded,
		},
	}

	h := newMockStatisticsHandler(userRestore, disasterRestore, drillRestore, runningRestore, otherClusterRestore)

	t.Run("all scope returns restore buckets and sources", func(t *testing.T) {
		ctx := app.NewContext(16)
		ctx.Request.SetRequestURI("/tasks/progress?type=restore&scope=all&range=7d&timezone=Asia/Shanghai&cluster=cluster-a")

		h.GetTaskProgressTrend(context.Background(), ctx)
		assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

		var resp taskProgressResponse
		err := json.Unmarshal(ctx.Response.Body(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, transport.CodeOK, resp.Code)
		assert.Equal(t, taskProgressTypeRestore, resp.Data.Type)
		assert.Len(t, resp.Data.Buckets, 7)
		assert.Equal(t, int32(4), resp.Data.Summary.Total)
		assert.Equal(t, int32(1), resp.Data.Summary.Completed)
		assert.Equal(t, int32(1), resp.Data.Summary.Failed)
		assert.Equal(t, int32(1), resp.Data.Summary.Canceled)
		assert.Equal(t, int32(1), resp.Data.Summary.InProgress)
		assert.Equal(t, "恢复成功", resp.Data.Series[0].Label)
		assert.Equal(t, "恢复失败", resp.Data.Series[1].Label)

		buckets := taskProgressBucketByDate(resp.Data)
		assert.Equal(t, int32(1), buckets[taskProgressDate(t, completedAt)].Completed)
		assert.Equal(t, int32(1), buckets[taskProgressDate(t, failedAt)].Failed)
		assert.Equal(t, int32(1), buckets[taskProgressDate(t, canceledAt)].Canceled)
		assert.Equal(t, int32(1), buckets[taskProgressDate(t, runningAt)].InProgress)

		sources := taskProgressSourceByScope(resp.Data)
		assert.Equal(t, int32(2), sources[taskProgressScopeApp].Total)
		assert.Equal(t, int32(1), sources[taskProgressScopeApp].Completed)
		assert.Equal(t, int32(2), sources[taskProgressScopeDisaster].Total)
		assert.Equal(t, int32(1), sources[taskProgressScopeDisaster].Failed)
	})

	t.Run("disaster scope excludes user restores", func(t *testing.T) {
		ctx := app.NewContext(16)
		ctx.Request.SetRequestURI("/tasks/progress?type=restore&scope=disaster&range=7d&timezone=Asia/Shanghai&cluster=cluster-a")

		h.GetTaskProgressTrend(context.Background(), ctx)
		assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

		var resp taskProgressResponse
		err := json.Unmarshal(ctx.Response.Body(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, int32(2), resp.Data.Summary.Total)
		assert.Equal(t, int32(1), resp.Data.Summary.Failed)
		assert.Equal(t, int32(1), resp.Data.Summary.Canceled)
		assert.Len(t, resp.Data.Sources, 1)
		assert.Equal(t, taskProgressScopeDisaster, resp.Data.Sources[0].Scope)
	})
}

func TestTaskProgressTrendInvalidParams(t *testing.T) {
	h := newMockStatisticsHandler()

	tests := []string{
		"/tasks/progress?type=job",
		"/tasks/progress?type=backup&scope=system",
		"/tasks/progress?type=backup&range=365d",
		"/tasks/progress?type=backup&timezone=bad-zone",
	}

	for _, uri := range tests {
		t.Run(uri, func(t *testing.T) {
			ctx := app.NewContext(16)
			ctx.Request.SetRequestURI(uri)

			h.GetTaskProgressTrend(context.Background(), ctx)
			assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())

			var resp struct {
				Code int `json:"code"`
			}
			err := json.Unmarshal(ctx.Response.Body(), &resp)
			assert.NoError(t, err)
			assert.Equal(t, transport.CodeBadRequest, resp.Code)
		})
	}
}
