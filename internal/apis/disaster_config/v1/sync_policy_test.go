package config

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route/param"
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-server/internal/transport"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConvertSpecToDTO_PreservesDualPolicyFieldsWithoutSyncPolicy(t *testing.T) {
	dto := ConvertSpecToDTO(dapisv1.DisasterConfigSpec{
		DataSyncPolicy:     "policy-data",
		ResourceSyncPolicy: "policy-resource",
	})

	assert.Equal(t, "policy-data", dto.DataSyncPolicy)
	assert.Equal(t, "policy-resource", dto.ResourceSyncPolicy)
	assert.Equal(t, "policy-resource", dto.ResourcesSyncPolicy)

	body, err := json.Marshal(dto)
	assert.NoError(t, err)
	assert.NotContains(t, string(body), `"syncPolicy"`)
}

func TestCreateConfig_WithDualPolicyFieldsWritesToCRD(t *testing.T) {
	h := newMockHandler()

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/configs")
	ctx.Request.SetBody([]byte(`{
		"name":"config-dual-policy",
		"sourceCluster":"cluster-a",
		"targetCluster":"cluster-b",
		"storageRepository":"repo-1",
		"dataSyncType":"snapshot",
		"dataSyncPolicy":"policy-data",
		"resourcesSyncPolicy":"policy-resource"
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createConfig(context.Background(), ctx)

	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())
	created, err := h.DisasterClient.DisasterV1().DisasterConfigs().Get(context.Background(), "config-dual-policy", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "policy-data", created.Spec.DataSyncPolicy)
	assert.Equal(t, "policy-resource", created.Spec.ResourceSyncPolicy)
}

func TestCreateConfig_WithSyncPolicyReturnsBadRequest(t *testing.T) {
	h := newMockHandler()

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/configs")
	ctx.Request.SetBody([]byte(`{
		"name":"config-sync-policy",
		"sourceCluster":"cluster-a",
		"targetCluster":"cluster-b",
		"storageRepository":"repo-1",
		"dataSyncType":"snapshot",
		"syncPolicy":"policy-shared"
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createConfig(context.Background(), ctx)

	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	assert.Contains(t, string(ctx.Response.Body()), "syncPolicy is not supported")
}

func TestUpdateConfig_WithDualPolicyFieldsUpdatesCRD(t *testing.T) {
	existing := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "config-update-dual-policy"},
		Spec: dapisv1.DisasterConfigSpec{
			SourceCluster:      "cluster-a",
			TargetCluster:      "cluster-b",
			StorageRepository:  "repo-1",
			DataSyncType:       "snapshot",
			DataSyncPolicy:     "policy-old-data",
			ResourceSyncPolicy: "policy-old-resource",
		},
	}
	h := newMockHandler(existing)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/configs/config-update-dual-policy")
	ctx.Params = param.Params{{Key: "name", Value: "config-update-dual-policy"}}
	ctx.Request.SetBody([]byte(`{"dataSyncPolicy":"policy-new-data","resourcesSyncPolicy":"policy-new-resource"}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.updateConfig(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	updated, err := h.DisasterClient.DisasterV1().DisasterConfigs().Get(context.Background(), "config-update-dual-policy", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "policy-new-data", updated.Spec.DataSyncPolicy)
	assert.Equal(t, "policy-new-resource", updated.Spec.ResourceSyncPolicy)
}

func TestUpdateConfig_WithSyncPolicyReturnsBadRequest(t *testing.T) {
	existing := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "config-update-sync-policy"},
		Spec: dapisv1.DisasterConfigSpec{
			SourceCluster:      "cluster-a",
			TargetCluster:      "cluster-b",
			StorageRepository:  "repo-1",
			DataSyncType:       "snapshot",
			DataSyncPolicy:     "policy-old-data",
			ResourceSyncPolicy: "policy-old-resource",
		},
	}
	h := newMockHandler(existing)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/configs/config-update-sync-policy")
	ctx.Params = param.Params{{Key: "name", Value: "config-update-sync-policy"}}
	ctx.Request.SetBody([]byte(`{"syncPolicy":"policy-new"}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.updateConfig(context.Background(), ctx)

	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeBadRequest, resp.Code)
	assert.Contains(t, resp.Message, "syncPolicy is not supported")
}
