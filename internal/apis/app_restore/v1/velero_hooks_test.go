package apprestore

import (
	"encoding/json"
	"testing"
	"time"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/stretchr/testify/assert"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func appRestoreTestHooks(name string) *velerov1.RestoreHooks {
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

func TestCreateAppRestoreRequestProjectsHooks(t *testing.T) {
	hooks := appRestoreTestHooks("restore-hook")

	req := CreateAppRestoreRequest{
		Name:         "restore-a",
		BackupSource: "backup-a",
		Cluster:      "cluster-b",
		BackupName:   "velero-backup-a",
		Hooks:        hooks,
	}
	spec := req.ToCRD()

	if assert.Len(t, spec.Template.Hooks.Resources, 1) {
		assert.Equal(t, "restore-hook", spec.Template.Hooks.Resources[0].Name)
	}
	dto := ConvertSpecToDTO(spec)
	if assert.NotNil(t, dto.Hooks) && assert.Len(t, dto.Hooks.Resources, 1) {
		assert.Equal(t, "restore-hook", dto.Hooks.Resources[0].Name)
	}
}

func TestUpdateAppRestoreRequestReplacesAndClearsHooks(t *testing.T) {
	spec := dapisv1.AppRestoreSpec{
		Template: velerov1.RestoreSpec{
			Hooks: *appRestoreTestHooks("old-hook"),
		},
	}

	var replaceReq UpdateAppRestoreRequest
	assert.NoError(t, json.Unmarshal([]byte(`{
		"name":"restore-a",
		"hooks":{
			"resources":[{
				"name":"new-hook",
				"includedResources":["pods"],
				"postHooks":[{"exec":{"command":["sh","-c","echo new"]}}]
			}]
		}
	}`), &replaceReq))
	replaceReq.MergeToCRD(&spec)
	if assert.Len(t, spec.Template.Hooks.Resources, 1) {
		assert.Equal(t, "new-hook", spec.Template.Hooks.Resources[0].Name)
	}

	var clearReq UpdateAppRestoreRequest
	assert.NoError(t, json.Unmarshal([]byte(`{"name":"restore-a","hooks":null}`), &clearReq))
	clearReq.MergeToCRD(&spec)
	assert.Empty(t, spec.Template.Hooks.Resources)
}
