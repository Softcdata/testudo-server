package instance

import (
	"encoding/json"
	"testing"
	"time"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	velerohooks "github.com/softcdata/testudo-server/internal/apis/velero_hooks"
	"github.com/stretchr/testify/assert"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func instanceBackupHooks(name string) *velerov1.BackupHooks {
	return &velerov1.BackupHooks{
		Resources: []velerov1.BackupResourceHookSpec{
			{
				Name:              name,
				IncludedResources: []string{"pods"},
				PreHooks: []velerov1.BackupResourceHook{
					{Exec: &velerov1.ExecHook{Command: []string{"sh", "-c", "echo backup"}}},
				},
			},
		},
	}
}

func instanceRestoreHooks(name string) *velerov1.RestoreHooks {
	return &velerov1.RestoreHooks{
		Resources: []velerov1.RestoreResourceHookSpec{
			{
				Name:              name,
				IncludedResources: []string{"pods"},
				PostHooks: []velerov1.RestoreResourceHook{
					{
						Exec: &velerov1.ExecRestoreHook{
							Command:     []string{"sh", "-c", "echo restore"},
							WaitTimeout: metav1.Duration{Duration: time.Minute},
						},
					},
				},
			},
		},
	}
}

func TestCreateDisasterInstanceRequestProjectsVeleroHooks(t *testing.T) {
	spec, err := (&CreateDisasterInstanceRequest{
		Config: "cfg-a",
		VeleroHooks: &velerohooks.DisasterVeleroHooksRequest{
			DataBackup:  instanceBackupHooks("backup-hook"),
			DataRestore: instanceRestoreHooks("restore-hook"),
		},
	}).ToCRD()

	assert.NoError(t, err)
	if assert.NotNil(t, spec.VeleroHooks) {
		assert.Equal(t, "backup-hook", spec.VeleroHooks.DataBackup.Resources[0].Name)
		assert.Equal(t, "restore-hook", spec.VeleroHooks.DataRestore.Resources[0].Name)
	}
}

func TestUpdateDisasterInstanceVeleroHooksPatchPresence(t *testing.T) {
	spec := dapisv1.DisasterInstanceSpec{
		VeleroHooks: &dapisv1.DisasterVeleroHooks{
			DataBackup:  instanceBackupHooks("backup-hook"),
			DataRestore: instanceRestoreHooks("restore-hook"),
		},
	}

	var req UpdateDisasterInstanceRequest
	assert.NoError(t, json.Unmarshal([]byte(`{"veleroHooks":{"dataRestore":null}}`), &req))
	req.ApplyVeleroHooksPatch(&spec)
	if assert.NotNil(t, spec.VeleroHooks) {
		assert.NotNil(t, spec.VeleroHooks.DataBackup)
		assert.Nil(t, spec.VeleroHooks.DataRestore)
	}

	var clearReq UpdateDisasterInstanceRequest
	assert.NoError(t, json.Unmarshal([]byte(`{"veleroHooks":{}}`), &clearReq))
	clearReq.ApplyVeleroHooksPatch(&spec)
	assert.Nil(t, spec.VeleroHooks)
}

func TestSyncHistoryHookStatusDTOs(t *testing.T) {
	start := metav1.NewTime(time.Date(2026, 6, 8, 8, 0, 0, 0, time.UTC))
	end := metav1.NewTime(time.Date(2026, 6, 8, 8, 3, 0, 0, time.UTC))
	record := dapisv1.SyncHistoryRecord{
		Status:         "Completed",
		StartTime:      &start,
		CompletionTime: &end,
		BackupName:     "backup-a",
		RestoreName:    "restore-a",
		BackupHookStatus: &dapisv1.SyncHistoryHookStatus{
			HooksAttempted: 2,
			HooksFailed:    1,
		},
		RestoreHookStatus: &dapisv1.SyncHistoryHookStatus{
			HooksAttempted: 3,
		},
	}

	last := convertLastSyncStatus(&record)
	if assert.NotNil(t, last) && assert.NotNil(t, last.BackupHookStatus) && assert.NotNil(t, last.RestoreHookStatus) {
		assert.Equal(t, 2, last.BackupHookStatus.HooksAttempted)
		assert.Equal(t, 1, last.BackupHookStatus.HooksFailed)
		assert.Equal(t, 3, last.RestoreHookStatus.HooksAttempted)
	}

	var items []syncHistoryItemWithSort
	appendSyncRecordHistory(&items, syncHistoryTypeDataSync, "ds-a", []dapisv1.SyncHistoryRecord{record})
	if assert.Len(t, items, 1) && assert.NotNil(t, items[0].BackupHookStatus) && assert.NotNil(t, items[0].RestoreHookStatus) {
		assert.Equal(t, 2, items[0].BackupHookStatus.HooksAttempted)
		assert.Equal(t, 3, items[0].RestoreHookStatus.HooksAttempted)
	}
}
