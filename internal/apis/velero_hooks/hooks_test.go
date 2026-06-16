package velerohooks

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/stretchr/testify/assert"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func testBackupHooks(command ...string) *velerov1.BackupHooks {
	return &velerov1.BackupHooks{
		Resources: []velerov1.BackupResourceHookSpec{
			{
				Name:              "backup-pod-hook",
				IncludedResources: []string{"pods"},
				PreHooks: []velerov1.BackupResourceHook{
					{
						Exec: &velerov1.ExecHook{
							Command: command,
							Timeout: metav1.Duration{Duration: time.Minute},
						},
					},
				},
			},
		},
	}
}

func testRestoreHooks(waitTimeout time.Duration, command ...string) *velerov1.RestoreHooks {
	return &velerov1.RestoreHooks{
		Resources: []velerov1.RestoreResourceHookSpec{
			{
				Name:              "restore-pod-hook",
				IncludedResources: []string{"pods"},
				PostHooks: []velerov1.RestoreResourceHook{
					{
						Exec: &velerov1.ExecRestoreHook{
							Command:     command,
							WaitTimeout: metav1.Duration{Duration: waitTimeout},
						},
					},
				},
			},
		},
	}
}

func TestDisasterVeleroHooksPatchPreservesChildPresence(t *testing.T) {
	target := &dapisv1.DisasterVeleroHooks{
		DataBackup:  testBackupHooks("sh", "-c", "echo backup"),
		DataRestore: testRestoreHooks(time.Minute, "sh", "-c", "echo restore"),
	}

	patch, err := DecodeDisasterVeleroHooksPatch(json.RawMessage(`{"dataRestore":null}`))
	assert.NoError(t, err)
	ApplyDisasterVeleroHooksPatch(&target, patch)

	if assert.NotNil(t, target) {
		assert.NotNil(t, target.DataBackup)
		assert.Nil(t, target.DataRestore)
	}

	patch, err = DecodeDisasterVeleroHooksPatch(json.RawMessage(`{}`))
	assert.NoError(t, err)
	ApplyDisasterVeleroHooksPatch(&target, patch)
	assert.Nil(t, target)
}

func TestDisasterVeleroHooksRequestInternalPresenceFlagsAreNotJSON(t *testing.T) {
	body, err := json.Marshal(DisasterVeleroHooksRequest{
		DataRestore: testRestoreHooks(time.Minute, "sh", "-c", "echo restore"),
	})
	assert.NoError(t, err)
	assert.Contains(t, string(body), "dataRestore")
	assert.NotContains(t, string(body), "DataRestoreSet")
	assert.NotContains(t, string(body), "DataBackupSet")
}

func TestDisasterVeleroHooksRequestToCRDPreservesEmptyObject(t *testing.T) {
	var req DisasterVeleroHooksRequest
	assert.NoError(t, json.Unmarshal([]byte(`{}`), &req))

	crd := req.ToCRD()
	if assert.NotNil(t, crd) {
		assert.Nil(t, crd.DataBackup)
		assert.Nil(t, crd.DataRestore)
	}
}

func TestValidateBackupHooksRejectsSensitiveParameter(t *testing.T) {
	err := ValidateBackupHooks(testBackupHooks("sh", "-c", "curl -H token=plain http://example"), "hooks")

	var validationErr *ValidationError
	assert.True(t, errors.As(err, &validationErr))
	if assert.NotNil(t, validationErr) {
		assert.Equal(t, SensitiveParameterErrorCode, validationErr.Code)
		assert.Equal(t, "hooks.resources[0].pre[0].exec.command[2]", validationErr.FieldPath)
	}
}

func TestValidateRestoreHooksRejectsWaitTimeoutOverLimit(t *testing.T) {
	err := ValidateRestoreHooks(testRestoreHooks(31*time.Minute, "sh", "-c", "echo restore"), "hooks")

	var validationErr *ValidationError
	assert.True(t, errors.As(err, &validationErr))
	if assert.NotNil(t, validationErr) {
		assert.Equal(t, "hooks.resources[0].postHooks[0].exec.waitTimeout", validationErr.FieldPath)
		assert.Contains(t, validationErr.Message, "30m")
	}
}
