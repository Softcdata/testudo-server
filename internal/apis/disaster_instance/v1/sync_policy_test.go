package instance

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
	"k8s.io/apimachinery/pkg/types"
)

func TestConvertToDisasterInstanceDTO_PopulatesEffectiveSyncPoliciesAndSources(t *testing.T) {
	testCases := []struct {
		name                       string
		instanceDataSyncPolicy     string
		instanceResourceSyncPolicy string
		configDataSyncPolicy       string
		configResourceSyncPolicy   string
		expectedDataSyncPolicy     string
		expectedResourceSyncPolicy string
		expectedDataSyncSource     string
		expectedResourceSyncSource string
	}{
		{
			name:                       "inherits config when override is empty",
			configDataSyncPolicy:       "policy-shared",
			configResourceSyncPolicy:   "policy-shared",
			expectedDataSyncPolicy:     "policy-shared",
			expectedResourceSyncPolicy: "policy-shared",
			expectedDataSyncSource:     syncPolicySourceConfig,
			expectedResourceSyncSource: syncPolicySourceConfig,
		},
		{
			name:                       "overrides data policy only",
			instanceDataSyncPolicy:     "policy-data-instance",
			configDataSyncPolicy:       "policy-data-config",
			configResourceSyncPolicy:   "policy-resource-config",
			expectedDataSyncPolicy:     "policy-data-instance",
			expectedResourceSyncPolicy: "policy-resource-config",
			expectedDataSyncSource:     syncPolicySourceInstance,
			expectedResourceSyncSource: syncPolicySourceConfig,
		},
		{
			name:                       "overrides resource policy only",
			instanceResourceSyncPolicy: "policy-resource-instance",
			configDataSyncPolicy:       "policy-data-config",
			configResourceSyncPolicy:   "policy-resource-config",
			expectedDataSyncPolicy:     "policy-data-config",
			expectedResourceSyncPolicy: "policy-resource-instance",
			expectedDataSyncSource:     syncPolicySourceConfig,
			expectedResourceSyncSource: syncPolicySourceInstance,
		},
		{
			name:                       "overrides both policies with same unified value",
			instanceDataSyncPolicy:     "policy-shared-instance",
			instanceResourceSyncPolicy: "policy-shared-instance",
			configDataSyncPolicy:       "policy-data-config",
			configResourceSyncPolicy:   "policy-resource-config",
			expectedDataSyncPolicy:     "policy-shared-instance",
			expectedResourceSyncPolicy: "policy-shared-instance",
			expectedDataSyncSource:     syncPolicySourceInstance,
			expectedResourceSyncSource: syncPolicySourceInstance,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			instance := &dapisv1.DisasterInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "inst-sync-policy",
					Namespace: "disaster-system",
					UID:       types.UID("uid-inst-sync-policy"),
				},
				Spec: dapisv1.DisasterInstanceSpec{
					Config:             "cfg-sync-policy",
					DataSyncPolicy:     tc.instanceDataSyncPolicy,
					ResourceSyncPolicy: tc.instanceResourceSyncPolicy,
				},
			}
			config := &dapisv1.DisasterConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "cfg-sync-policy"},
				Spec: dapisv1.DisasterConfigSpec{
					DataSyncPolicy:     tc.configDataSyncPolicy,
					ResourceSyncPolicy: tc.configResourceSyncPolicy,
				},
			}

			dto := ConvertToDisasterInstanceDTO(instance, config, nil)

			assert.Equal(t, tc.instanceDataSyncPolicy, dto.Spec.DataSyncPolicy)
			assert.Equal(t, tc.instanceResourceSyncPolicy, dto.Spec.ResourceSyncPolicy)
			assert.Equal(t, tc.expectedDataSyncPolicy, dto.Spec.EffectiveDataSyncPolicy)
			assert.Equal(t, tc.expectedResourceSyncPolicy, dto.Spec.EffectiveResourceSyncPolicy)
			assert.Equal(t, tc.expectedDataSyncSource, dto.Spec.DataSyncPolicySource)
			assert.Equal(t, tc.expectedResourceSyncSource, dto.Spec.ResourceSyncPolicySource)

			body, err := json.Marshal(dto.Spec)
			assert.NoError(t, err)
			assert.NotContains(t, string(body), `"syncPolicy"`)
		})
	}
}

func TestCreateInstance_WithDualPolicyFieldsWritesToCRD(t *testing.T) {
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg-sync-policy"},
	}
	h := newMockHandler(cfg)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances")
	ctx.Request.SetBody([]byte(`{
		"name":"inst-sync-policy",
		"namespace":"disaster-system",
		"config":"cfg-sync-policy",
		"dataSyncPolicy":"policy-data",
		"resourceSyncPolicy":"policy-resource"
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createInstance(context.Background(), ctx)

	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())
	created, err := h.DisasterClient.DisasterV1().DisasterInstances("disaster-system").Get(context.Background(), "inst-sync-policy", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "policy-data", created.Spec.DataSyncPolicy)
	assert.Equal(t, "policy-resource", created.Spec.ResourceSyncPolicy)
}

func TestCreateInstance_WithSyncPolicyReturnsBadRequest(t *testing.T) {
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg-sync-policy-conflict"},
	}
	h := newMockHandler(cfg)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances")
	ctx.Request.SetBody([]byte(`{
		"name":"inst-sync-policy-conflict",
		"namespace":"disaster-system",
		"config":"cfg-sync-policy-conflict",
		"syncPolicy":"policy-a"
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createInstance(context.Background(), ctx)

	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	assert.Contains(t, string(ctx.Response.Body()), "syncPolicy is not supported")
}

func TestUpdateInstance_WithDualPolicyFieldsUpdatesCRD(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-update-sync-policy",
			Namespace: ns,
			UID:       types.UID("uid-update-sync-policy"),
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Config:             "cfg-sync-policy-update",
			DataSyncPolicy:     "policy-old-data",
			ResourceSyncPolicy: "policy-old-resource",
		},
	}
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg-sync-policy-update"},
	}
	h := newMockHandler(inst, cfg)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-update-sync-policy?namespace=disaster-system")
	ctx.Params = param.Params{{Key: "name", Value: "inst-update-sync-policy"}}
	ctx.Request.SetBody([]byte(`{"dataSyncPolicy":"policy-new-data","resourceSyncPolicy":"policy-new-resource"}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.updateInstance(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	updated, err := h.DisasterClient.DisasterV1().DisasterInstances(ns).Get(context.Background(), "inst-update-sync-policy", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "policy-new-data", updated.Spec.DataSyncPolicy)
	assert.Equal(t, "policy-new-resource", updated.Spec.ResourceSyncPolicy)
}

func TestUpdateInstance_WithSyncPolicyReturnsBadRequest(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-update-sync-policy-conflict",
			Namespace: ns,
			UID:       types.UID("uid-update-sync-policy-conflict"),
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Config:             "cfg-sync-policy-update-conflict",
			DataSyncPolicy:     "policy-old-data",
			ResourceSyncPolicy: "policy-old-resource",
		},
	}
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg-sync-policy-update-conflict"},
	}
	h := newMockHandler(inst, cfg)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-update-sync-policy-conflict?namespace=disaster-system")
	ctx.Params = param.Params{{Key: "name", Value: "inst-update-sync-policy-conflict"}}
	ctx.Request.SetBody([]byte(`{"syncPolicy":"policy-a"}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.updateInstance(context.Background(), ctx)

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
