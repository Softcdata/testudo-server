package drill

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	velerohooks "github.com/softcdata/testudo-server/internal/apis/velero_hooks"
	"github.com/stretchr/testify/assert"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func drillRestoreHooks(name string) *velerov1.RestoreHooks {
	return &velerov1.RestoreHooks{
		Resources: []velerov1.RestoreResourceHookSpec{
			{
				Name:              name,
				IncludedResources: []string{"pods"},
				PostHooks: []velerov1.RestoreResourceHook{
					{
						Exec: &velerov1.ExecRestoreHook{
							Command:     []string{"sh", "-c", "echo drill"},
							WaitTimeout: metav1.Duration{Duration: time.Minute},
						},
					},
				},
			},
		},
	}
}

func TestCreateDrillProjectsDataRestoreHooks(t *testing.T) {
	instance := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-app-dr",
			Namespace: testNamespace,
		},
	}
	h := newMockHandler(instance)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/drills")
	req := CreateDrillRequest{
		InstanceName: "my-app-dr",
		Name:         "test-drill-hooks",
		Namespace:    testNamespace,
		VeleroHooks: &velerohooks.DisasterVeleroHooksRequest{
			DataRestore: drillRestoreHooks("drill-restore-hook"),
		},
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createDrill(context.Background(), ctx)

	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())
	created, err := h.DisasterClient.DisasterV1().DisasterDrills(testNamespace).Get(
		context.Background(), "test-drill-hooks", metav1.GetOptions{},
	)
	assert.NoError(t, err)
	if assert.NotNil(t, created.Spec.VeleroHooks) && assert.NotNil(t, created.Spec.VeleroHooks.DataRestore) {
		assert.Equal(t, "drill-restore-hook", created.Spec.VeleroHooks.DataRestore.Resources[0].Name)
	}

	var resp struct {
		Data DisasterDrillDTO `json:"data"`
	}
	assert.NoError(t, json.Unmarshal(ctx.Response.Body(), &resp))
	if assert.NotNil(t, resp.Data.VeleroHooks) && assert.NotNil(t, resp.Data.VeleroHooks.DataRestore) {
		assert.Equal(t, "drill-restore-hook", resp.Data.VeleroHooks.DataRestore.Resources[0].Name)
	}
}

func TestCreateDrillRejectsDataBackupHooks(t *testing.T) {
	h := newMockHandler()

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/drills")
	ctx.Request.SetBody([]byte(`{
		"instanceName":"my-app-dr",
		"name":"test-drill-backup-hooks",
		"veleroHooks":{"dataBackup":{}}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createDrill(context.Background(), ctx)

	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	assert.Contains(t, string(ctx.Response.Body()), "veleroHooks.dataBackup")
}

func TestCreateDrillPreservesEmptyVeleroHooksOverride(t *testing.T) {
	instance := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-app-dr",
			Namespace: testNamespace,
		},
	}
	h := newMockHandler(instance)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/drills")
	ctx.Request.SetBody([]byte(`{
		"instanceName":"my-app-dr",
		"name":"test-drill-empty-hooks",
		"namespace":"default",
		"veleroHooks":{}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createDrill(context.Background(), ctx)

	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())
	created, err := h.DisasterClient.DisasterV1().DisasterDrills(testNamespace).Get(
		context.Background(), "test-drill-empty-hooks", metav1.GetOptions{},
	)
	assert.NoError(t, err)
	if assert.NotNil(t, created.Spec.VeleroHooks) {
		assert.Nil(t, created.Spec.VeleroHooks.DataBackup)
		assert.Nil(t, created.Spec.VeleroHooks.DataRestore)
	}

	var resp struct {
		Data DisasterDrillDTO `json:"data"`
	}
	assert.NoError(t, json.Unmarshal(ctx.Response.Body(), &resp))
	if assert.NotNil(t, resp.Data.VeleroHooks) {
		assert.Nil(t, resp.Data.VeleroHooks.DataBackup)
		assert.Nil(t, resp.Data.VeleroHooks.DataRestore)
	}
}
