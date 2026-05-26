package config

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route/param"
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-operator/pkg/clientset/versioned/fake"
	informers "github.com/softcdata/testudo-operator/pkg/informers/externalversions"
	"github.com/softcdata/testudo-server/internal/kube"
	"github.com/softcdata/testudo-server/internal/transport"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

func newMockHandler(objects ...runtime.Object) *ConfigHandler {
	fakeClient := fake.NewSimpleClientset(objects...)
	informerFactory := informers.NewSharedInformerFactory(fakeClient, 0)

	kc := &kube.KubeClient{
		DisasterClient:  fakeClient,
		InformerFactory: informerFactory,
	}

	h := server.Default()
	rg := h.Group("/v1")

	return NewConfigHandler(kc, rg)
}

func TestConfigNames_IncludesStatus(t *testing.T) {
	configA := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "config-a",
			UID:  types.UID("uid-a"),
		},
		Spec: dapisv1.DisasterConfigSpec{
			SourceCluster: "cluster-a",
			TargetCluster: "cluster-b",
		},
		Status: dapisv1.DisasterConfigStatus{
			Status: dapisv1.DisasterConfigStatusReady,
		},
	}
	configB := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "config-b",
			UID:  types.UID("uid-b"),
		},
		Spec: dapisv1.DisasterConfigSpec{
			SourceCluster: "cluster-b",
			TargetCluster: "cluster-c",
		},
		Status: dapisv1.DisasterConfigStatus{
			Status: dapisv1.DisasterConfigStatusError,
		},
	}

	h := newMockHandler(configA, configB)

	stopCh := make(chan struct{})
	defer close(stopCh)
	h.InformerFactory.Start(stopCh)
	for _, ok := range h.InformerFactory.WaitForCacheSync(stopCh) {
		assert.True(t, ok)
	}

	ctx := app.NewContext(16)
	h.configNames(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp struct {
		Code int                     `json:"code"`
		Data []DisasterConfigNameDTO `json:"data"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)

	gotStatus := make(map[string]dapisv1.StatusType, len(resp.Data))
	gotSource := make(map[string]string, len(resp.Data))
	gotTarget := make(map[string]string, len(resp.Data))
	for _, item := range resp.Data {
		gotStatus[item.Name] = item.Status
		gotSource[item.Name] = item.SourceCluster
		gotTarget[item.Name] = item.TargetCluster
	}

	assert.Equal(t, dapisv1.DisasterConfigStatusReady, gotStatus["config-a"])
	assert.Equal(t, dapisv1.DisasterConfigStatusError, gotStatus["config-b"])
	assert.Equal(t, "cluster-a", gotSource["config-a"])
	assert.Equal(t, "cluster-b", gotSource["config-b"])
	assert.Equal(t, "cluster-b", gotTarget["config-a"])
	assert.Equal(t, "cluster-c", gotTarget["config-b"])
}

func TestListConfigs_KeywordByNameAndSummary(t *testing.T) {
	configReady := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "prod-ready",
			UID:  types.UID("uid-ready"),
		},
		Status: dapisv1.DisasterConfigStatus{
			Status: dapisv1.DisasterConfigStatusReady,
		},
	}
	configError := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "prod-error",
			UID:  types.UID("uid-error"),
		},
		Status: dapisv1.DisasterConfigStatus{
			Status: dapisv1.DisasterConfigStatusError,
		},
	}
	configOther := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dev-notready",
			UID:  types.UID("uid-other"),
		},
		Status: dapisv1.DisasterConfigStatus{
			Status: dapisv1.DisasterConfigStatusNotReady,
		},
	}

	h := newMockHandler(configReady, configError, configOther)

	stopCh := make(chan struct{})
	defer close(stopCh)
	h.InformerFactory.Start(stopCh)
	for _, ok := range h.InformerFactory.WaitForCacheSync(stopCh) {
		assert.True(t, ok)
	}

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/configs?keyword=prod&limit=-1")

	h.configs(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Items []DisasterConfigDTO `json:"items"`
		} `json:"data"`
		Meta struct {
			Summary map[string]int `json:"summary"`
		} `json:"meta"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	assert.Len(t, resp.Data.Items, 2)
	assert.Equal(t, 1, resp.Meta.Summary["healthyCount"])
	assert.Equal(t, 1, resp.Meta.Summary["abnormalCount"])

	gotNames := map[string]bool{}
	for _, item := range resp.Data.Items {
		gotNames[item.Name] = true
	}
	assert.True(t, gotNames["prod-ready"])
	assert.True(t, gotNames["prod-error"])
	assert.False(t, gotNames["dev-notready"])
}

func TestUpdateConfig_UsesPathNameWithoutBodyName(t *testing.T) {
	existing := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "config-a",
			UID:         types.UID("uid-a"),
			Annotations: map[string]string{"testudo.softcdata.com/description": "old desc"},
		},
		Spec: dapisv1.DisasterConfigSpec{
			SourceCluster:      "cluster-a",
			TargetCluster:      "cluster-b",
			StorageRepository:  "repo-1",
			DataSyncType:       "snapshot",
			ResourceSyncPolicy: "resource-policy-old",
			DataSyncPolicy:     "data-policy-old",
		},
	}

	h := newMockHandler(existing)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/configs/config-a")
	ctx.Params = param.Params{
		{Key: "name", Value: "config-a"},
	}
	ctx.Request.SetBody([]byte(`{"description":"new desc","sourceCluster":"cluster-c","resourcesSyncPolicy":""}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.updateConfig(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp struct {
		Code int `json:"code"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)

	updated, err := h.DisasterClient.DisasterV1().DisasterConfigs().Get(context.Background(), "config-a", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "cluster-c", updated.Spec.SourceCluster)
	assert.Equal(t, "cluster-b", updated.Spec.TargetCluster)
	assert.Equal(t, "", updated.Spec.ResourceSyncPolicy)
	assert.Equal(t, "new desc", updated.Annotations["testudo.softcdata.com/description"])
}

func TestToDisasterConfigDTO_DoesNotPopulateDefaultCronsWithoutPolicy(t *testing.T) {
	item := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "config-default",
			UID:  types.UID("uid-default"),
		},
		Spec: dapisv1.DisasterConfigSpec{
			SourceCluster:     "cluster-a",
			TargetCluster:     "cluster-b",
			StorageRepository: "repo-a",
		},
	}

	h := newMockHandler()
	dto := h.toDisasterConfigDTO(context.Background(), item)

	assert.Empty(t, dto.Spec.DataSyncCron)
	assert.Empty(t, dto.Spec.ResourceSyncCron)
}

func TestConvertToDisasterConfigDTO_FormatsCreationTimestampInLocalTime(t *testing.T) {
	oldLocal := time.Local
	loc, err := time.LoadLocation("Asia/Shanghai")
	assert.NoError(t, err)
	time.Local = loc
	defer func() {
		time.Local = oldLocal
	}()

	item := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "config-time",
			UID:               types.UID("uid-time"),
			CreationTimestamp: metav1.NewTime(time.Date(2026, 5, 15, 3, 8, 19, 0, time.UTC)),
		},
	}

	dto := ConvertToDisasterConfigDTO(item)

	data, err := json.Marshal(dto)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"creation_timestamp":"2026-05-15T11:08:19+08:00"`)
}

func TestCreateConfig_WithImageRewrite(t *testing.T) {
	srcCluster := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-a"},
		Spec: dapisv1.ClusterSpec{
			ImageSources: []dapisv1.ImageSource{
				{Name: "prod-main", Registry: "harbor.prod.local"},
			},
		},
	}
	dstCluster := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-b"},
		Spec: dapisv1.ClusterSpec{
			ImageSources: []dapisv1.ImageSource{
				{Name: "dr-main", Registry: "harbor.dr.local"},
			},
		},
	}

	h := newMockHandler(srcCluster, dstCluster)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/configs")
	ctx.Request.SetBody([]byte(`{
		"name":"config-with-image-rewrite",
		"sourceCluster":"cluster-a",
		"targetCluster":"cluster-b",
		"storageRepository":"repo-1",
		"dataSyncType":"fsb",
		"imageRewrite":{
			"enabled":true,
			"applyTo":["resourceSync","drill"],
			"mappings":[
				{"sourceImageSource":"prod-main","targetImageSource":"dr-main"}
			]
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createConfig(context.Background(), ctx)

	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())
	created, err := h.DisasterClient.DisasterV1().DisasterConfigs().Get(context.Background(), "config-with-image-rewrite", metav1.GetOptions{})
	assert.NoError(t, err)
	if assert.NotNil(t, created.Spec.ImageRewrite) {
		assert.Equal(t, dapisv1.ImageRewriteUnmatchedPolicyFail, created.Spec.ImageRewrite.UnmatchedPolicy)
		assert.Equal(t, []dapisv1.ImageRewriteApplyTarget{
			dapisv1.ImageRewriteApplyResourceSync,
			dapisv1.ImageRewriteApplyDrill,
		}, created.Spec.ImageRewrite.ApplyTo)
		assert.Equal(t, []dapisv1.ImageSourceMapping{
			{SourceImageSource: "prod-main", TargetImageSource: "dr-main"},
		}, created.Spec.ImageRewrite.Mappings)
	}
}

func TestCreateConfig_WithInvalidImageRewrite_ShouldFail(t *testing.T) {
	srcCluster := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-a"},
		Spec: dapisv1.ClusterSpec{
			ImageSources: []dapisv1.ImageSource{
				{Name: "prod-main", Registry: "harbor.prod.local"},
			},
		},
	}
	dstCluster := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-b"},
		Spec: dapisv1.ClusterSpec{
			ImageSources: []dapisv1.ImageSource{
				{Name: "dr-main", Registry: "harbor.dr.local"},
			},
		},
	}

	h := newMockHandler(srcCluster, dstCluster)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/configs")
	ctx.Request.SetBody([]byte(`{
		"name":"config-invalid-image-rewrite",
		"sourceCluster":"cluster-a",
		"targetCluster":"cluster-b",
		"storageRepository":"repo-1",
		"dataSyncType":"fsb",
		"imageRewrite":{
			"enabled":true,
			"mappings":[
				{"sourceImageSource":"prod-main","targetImageSource":"dr-main"},
				{"sourceImageSource":"prod-main","targetImageSource":"dr-main"}
			]
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createConfig(context.Background(), ctx)

	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
}

func TestCreateConfig_WithReverseImageRewriteMapping_ShouldPass(t *testing.T) {
	srcCluster := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-a"},
		Spec: dapisv1.ClusterSpec{
			ImageSources: []dapisv1.ImageSource{
				{Name: "prod-main", Registry: "harbor.prod.local"},
			},
		},
	}
	dstCluster := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-b"},
		Spec: dapisv1.ClusterSpec{
			ImageSources: []dapisv1.ImageSource{
				{Name: "dr-main", Registry: "harbor.dr.local"},
			},
		},
	}

	h := newMockHandler(srcCluster, dstCluster)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/configs")
	ctx.Request.SetBody([]byte(`{
		"name":"config-reverse-image-rewrite",
		"sourceCluster":"cluster-a",
		"targetCluster":"cluster-b",
		"storageRepository":"repo-1",
		"dataSyncType":"fsb",
		"imageRewrite":{
			"enabled":true,
			"mappings":[
				{"sourceImageSource":"dr-main","targetImageSource":"prod-main"}
			]
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createConfig(context.Background(), ctx)
	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())
}

func TestUpdateConfig_WithInvalidImageRewrite_ShouldReturnBadRequest(t *testing.T) {
	srcCluster := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-a"},
		Spec: dapisv1.ClusterSpec{
			ImageSources: []dapisv1.ImageSource{
				{Name: "prod-main", Registry: "harbor.prod.local"},
			},
		},
	}
	dstCluster := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-b"},
		Spec: dapisv1.ClusterSpec{
			ImageSources: []dapisv1.ImageSource{
				{Name: "dr-main", Registry: "harbor.dr.local"},
			},
		},
	}
	existing := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "config-update-invalid"},
		Spec: dapisv1.DisasterConfigSpec{
			SourceCluster: "cluster-a",
			TargetCluster: "cluster-b",
		},
	}

	h := newMockHandler(srcCluster, dstCluster, existing)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/configs/config-update-invalid")
	ctx.Params = param.Params{{Key: "name", Value: "config-update-invalid"}}
	ctx.Request.SetBody([]byte(`{
		"imageRewrite":{
			"enabled":true,
			"mappings":[
				{"sourceImageSource":"not-exist","targetImageSource":"dr-main"}
			]
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.updateConfig(context.Background(), ctx)
	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
}

func TestCreateConfig_WithLegacyImageSources_ShouldPersistImageRewrite(t *testing.T) {
	srcCluster := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-a"},
		Spec: dapisv1.ClusterSpec{
			ImageSources: []dapisv1.ImageSource{
				{Name: "prod-main", Registry: "harbor.prod.local"},
			},
		},
	}
	dstCluster := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-b"},
		Spec: dapisv1.ClusterSpec{
			ImageSources: []dapisv1.ImageSource{
				{Name: "dr-main", Registry: "harbor.dr.local"},
			},
		},
	}

	h := newMockHandler(srcCluster, dstCluster)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/configs")
	ctx.Request.SetBody([]byte(`{
		"name":"config-legacy-image-sources",
		"sourceCluster":"cluster-a",
		"targetCluster":"cluster-b",
		"storageRepository":"repo-1",
		"dataSyncType":"fsb",
		"imageSources":{
			"prod-main":"dr-main"
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createConfig(context.Background(), ctx)
	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())

	created, err := h.DisasterClient.DisasterV1().DisasterConfigs().Get(context.Background(), "config-legacy-image-sources", metav1.GetOptions{})
	assert.NoError(t, err)
	if assert.NotNil(t, created.Spec.ImageRewrite) {
		assert.True(t, created.Spec.ImageRewrite.Enabled)
		assert.Equal(t, []dapisv1.ImageRewriteApplyTarget{
			dapisv1.ImageRewriteApplyResourceSync,
			dapisv1.ImageRewriteApplyDrill,
		}, created.Spec.ImageRewrite.ApplyTo)
		assert.Equal(t, []dapisv1.ImageSourceMapping{
			{SourceImageSource: "prod-main", TargetImageSource: "dr-main"},
		}, created.Spec.ImageRewrite.Mappings)
	}
}

func TestUpdateConfig_WithLegacyImageSources_ShouldPersistImageRewrite(t *testing.T) {
	srcCluster := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-a"},
		Spec: dapisv1.ClusterSpec{
			ImageSources: []dapisv1.ImageSource{
				{Name: "prod-main", Registry: "harbor.prod.local"},
			},
		},
	}
	dstCluster := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-b"},
		Spec: dapisv1.ClusterSpec{
			ImageSources: []dapisv1.ImageSource{
				{Name: "dr-main", Registry: "harbor.dr.local"},
			},
		},
	}
	existing := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "config-update-legacy-image-sources"},
		Spec: dapisv1.DisasterConfigSpec{
			SourceCluster: "cluster-a",
			TargetCluster: "cluster-b",
		},
	}

	h := newMockHandler(srcCluster, dstCluster, existing)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/configs/config-update-legacy-image-sources")
	ctx.Params = param.Params{{Key: "name", Value: "config-update-legacy-image-sources"}}
	ctx.Request.SetBody([]byte(`{
		"imageSources":{
			"prod-main":"dr-main"
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.updateConfig(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	updated, err := h.DisasterClient.DisasterV1().DisasterConfigs().Get(context.Background(), "config-update-legacy-image-sources", metav1.GetOptions{})
	assert.NoError(t, err)
	if assert.NotNil(t, updated.Spec.ImageRewrite) {
		assert.True(t, updated.Spec.ImageRewrite.Enabled)
		assert.Equal(t, []dapisv1.ImageSourceMapping{
			{SourceImageSource: "prod-main", TargetImageSource: "dr-main"},
		}, updated.Spec.ImageRewrite.Mappings)
	}
}
