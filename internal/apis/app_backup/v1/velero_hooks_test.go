package appbackup

import (
	"encoding/json"
	"testing"
	"time"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/stretchr/testify/assert"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func appBackupTestHooks(name string) *velerov1.BackupHooks {
	return &velerov1.BackupHooks{
		Resources: []velerov1.BackupResourceHookSpec{
			{
				Name:              name,
				IncludedResources: []string{"pods"},
				PreHooks: []velerov1.BackupResourceHook{
					{
						Exec: &velerov1.ExecHook{
							Command: []string{"sh", "-c", "echo backup"},
							Timeout: metav1.Duration{Duration: time.Minute},
						},
					},
				},
			},
		},
	}
}

func TestCreateAppBackupRequestProjectsHooks(t *testing.T) {
	hooks := appBackupTestHooks("backup-hook")

	req := CreateAppBackupRequest{
		Name:            "backup-a",
		Cluster:         "cluster-a",
		Schedule:        "@daily",
		StorageLocation: "repo-a",
		Hooks:           hooks,
	}
	spec := req.ToCRD()

	if assert.Len(t, spec.Template.Hooks.Resources, 1) {
		assert.Equal(t, "backup-hook", spec.Template.Hooks.Resources[0].Name)
	}
	dto := ConvertSpecToDTO(spec)
	if assert.NotNil(t, dto.Hooks) && assert.Len(t, dto.Hooks.Resources, 1) {
		assert.Equal(t, "backup-hook", dto.Hooks.Resources[0].Name)
	}
}

func TestUpdateAppBackupRequestReplacesAndClearsHooks(t *testing.T) {
	spec := dapisv1.AppBackupSpec{
		Template: velerov1.BackupSpec{
			Hooks: *appBackupTestHooks("old-hook"),
		},
	}

	var replaceReq UpdateAppBackupRequest
	assert.NoError(t, json.Unmarshal([]byte(`{
		"name":"backup-a",
		"hooks":{
			"resources":[{
				"name":"new-hook",
				"includedResources":["pods"],
				"pre":[{"exec":{"command":["sh","-c","echo new"]}}]
			}]
		}
	}`), &replaceReq))
	replaceReq.MergeToCRD(&spec)
	if assert.Len(t, spec.Template.Hooks.Resources, 1) {
		assert.Equal(t, "new-hook", spec.Template.Hooks.Resources[0].Name)
	}

	var clearReq UpdateAppBackupRequest
	assert.NoError(t, json.Unmarshal([]byte(`{"name":"backup-a","hooks":{}}`), &clearReq))
	clearReq.MergeToCRD(&spec)
	assert.Empty(t, spec.Template.Hooks.Resources)
}
