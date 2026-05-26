package policy

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
	"github.com/softcdata/testudo-operator/pkg/metadata"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/kube"
	"github.com/softcdata/testudo-server/internal/transport"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

func newMockHandler(objects ...runtime.Object) *PolicyHandler {
	fakeClient := fake.NewSimpleClientset(objects...)
	informerFactory := informers.NewSharedInformerFactory(fakeClient, 0)

	kc := &kube.KubeClient{
		DisasterClient:  fakeClient,
		InformerFactory: informerFactory,
	}

	h := server.Default()
	rg := h.Group("/v1")

	return NewPolicyHandler(kc, rg)
}

func TestConvertSpecToDTO_MapsSyncPolicyTypesToUnifiedExternalType(t *testing.T) {
	dto := ConvertSpecToDTO(dapisv1.DisasterPolicySpec{
		Type:     dapisv1.PolicyTypeResourceSync,
		Schedule: "*/5 * * * *",
		State:    dapisv1.PolicyStateEnabled,
	})

	assert.Equal(t, ExternalPolicyTypeSyncPolicy, dto.Type)

	body, err := json.Marshal(dto)
	assert.NoError(t, err)
	assert.NotContains(t, string(body), legacyExternalPolicyTypeData)
	assert.NotContains(t, string(body), legacyExternalPolicyTypeResource)
}

func TestConvertToDisasterPolicyDTO_NormalizesSyncPolicyLabelToUnifiedType(t *testing.T) {
	item := &dapisv1.DisasterPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "policy-sync-label",
			Namespace: common.DisasterSystemNamespace,
			UID:       types.UID("policy-sync-label-uid"),
			Labels: map[string]string{
				metadata.LabelDisasterPolicyType:  string(dapisv1.PolicyTypeResourceSync),
				metadata.LabelDisasterPolicyName:  "policy-sync-label",
				metadata.LabelDisasterPolicyState: string(dapisv1.PolicyStateEnabled),
			},
		},
		Spec: dapisv1.DisasterPolicySpec{
			Type:     dapisv1.PolicyTypeResourceSync,
			Schedule: "*/5 * * * *",
			State:    dapisv1.PolicyStateEnabled,
		},
	}

	dto := ConvertToDisasterPolicyDTO(item)

	assert.Equal(t, ExternalPolicyTypeSyncPolicy, dto.Spec.Type)
	assert.Equal(t, ExternalPolicyTypeSyncPolicy, dto.Labels[metadata.LabelDisasterPolicyType])
	assert.Equal(t, string(dapisv1.PolicyTypeResourceSync), item.Labels[metadata.LabelDisasterPolicyType])
}

func TestCreatePolicy_WithSyncPolicyWritesCanonicalCRDType(t *testing.T) {
	h := newMockHandler()

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/policies")
	ctx.Request.SetBody([]byte(`{
		"name":"policy-sync-create",
		"type":"SyncPolicy",
		"schedule":"*/5 * * * *",
		"state":"Enabled"
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createPolicy(context.Background(), ctx)

	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())

	created, err := h.DisasterClient.DisasterV1().DisasterPolicies(common.DisasterSystemNamespace).Get(
		context.Background(),
		"policy-sync-create",
		metav1.GetOptions{},
	)
	assert.NoError(t, err)
	assert.Equal(t, dapisv1.PolicyTypeDataSync, created.Spec.Type)

	var resp struct {
		Code int               `json:"code"`
		Data DisasterPolicyDTO `json:"data"`
	}
	err = json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	assert.Equal(t, ExternalPolicyTypeSyncPolicy, resp.Data.Spec.Type)
}

func TestCreatePolicy_WithAutoBackupTTLWritesAndEchoesTTL(t *testing.T) {
	h := newMockHandler()

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/policies")
	ctx.Request.SetBody([]byte(`{
		"name":"policy-auto-ttl",
		"type":"AutoBackup",
		"schedule":"0 2 * * *",
		"ttl":"720h",
		"state":"Enabled"
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createPolicy(context.Background(), ctx)

	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())

	created, err := h.DisasterClient.DisasterV1().DisasterPolicies(common.DisasterSystemNamespace).Get(
		context.Background(),
		"policy-auto-ttl",
		metav1.GetOptions{},
	)
	assert.NoError(t, err)
	if assert.NotNil(t, created.Spec.TTL) {
		assert.Equal(t, 720*time.Hour, created.Spec.TTL.Duration)
	}

	var resp struct {
		Code int               `json:"code"`
		Data DisasterPolicyDTO `json:"data"`
	}
	err = json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	if assert.NotNil(t, resp.Data.Spec.TTL) {
		assert.Equal(t, 720*time.Hour, resp.Data.Spec.TTL.Duration)
	}
}

func TestPolicyDetail_WithAutoBackupTTLEchoesTTL(t *testing.T) {
	ttl := metav1.Duration{Duration: 720 * time.Hour}
	existing := &dapisv1.DisasterPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "policy-auto-detail-ttl",
			Namespace: common.DisasterSystemNamespace,
			UID:       types.UID("policy-auto-detail-ttl-uid"),
		},
		Spec: dapisv1.DisasterPolicySpec{
			Type:     dapisv1.PolicyTypeAutoBackup,
			Schedule: "0 2 * * *",
			TTL:      &ttl,
			State:    dapisv1.PolicyStateEnabled,
		},
	}
	h := newMockHandler(existing)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/policies/policy-auto-detail-ttl")
	ctx.Params = param.Params{{Key: "name", Value: "policy-auto-detail-ttl"}}

	h.policy(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	var resp struct {
		Code int               `json:"code"`
		Data DisasterPolicyDTO `json:"data"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, ExternalPolicyTypeAutoBackup, resp.Data.Spec.Type)
	if assert.NotNil(t, resp.Data.Spec.TTL) {
		assert.Equal(t, 720*time.Hour, resp.Data.Spec.TTL.Duration)
	}
}

func TestCreatePolicy_RejectsTTLForSyncPolicy(t *testing.T) {
	h := newMockHandler()

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/policies")
	ctx.Request.SetBody([]byte(`{
		"name":"policy-sync-with-ttl",
		"type":"SyncPolicy",
		"schedule":"*/5 * * * *",
		"ttl":"24h",
		"state":"Enabled"
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createPolicy(context.Background(), ctx)

	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	assert.Contains(t, string(ctx.Response.Body()), "ttl")
}

func TestUpdatePolicy_WithSyncPolicyPreservesExistingConcreteSyncType(t *testing.T) {
	existing := &dapisv1.DisasterPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "policy-sync-update",
			Namespace: common.DisasterSystemNamespace,
			UID:       types.UID("policy-sync-update-uid"),
		},
		Spec: dapisv1.DisasterPolicySpec{
			Type:     dapisv1.PolicyTypeResourceSync,
			Schedule: "*/10 * * * *",
			State:    dapisv1.PolicyStateEnabled,
		},
	}
	h := newMockHandler(existing)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/policies/policy-sync-update")
	ctx.Params = param.Params{{Key: "name", Value: "policy-sync-update"}}
	ctx.Request.SetBody([]byte(`{
		"name":"policy-sync-update",
		"type":"SyncPolicy",
		"schedule":"0 * * * *"
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.updatePolicy(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	updated, err := h.DisasterClient.DisasterV1().DisasterPolicies(common.DisasterSystemNamespace).Get(
		context.Background(),
		"policy-sync-update",
		metav1.GetOptions{},
	)
	assert.NoError(t, err)
	assert.Equal(t, dapisv1.PolicyTypeResourceSync, updated.Spec.Type)
	assert.Equal(t, "0 * * * *", updated.Spec.Schedule)

	var resp struct {
		Code int               `json:"code"`
		Data DisasterPolicyDTO `json:"data"`
	}
	err = json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, ExternalPolicyTypeSyncPolicy, resp.Data.Spec.Type)
}

func TestUpdatePolicy_WithAutoBackupTTLSupportsPartialUpdateAndClear(t *testing.T) {
	initialTTL := metav1.Duration{Duration: 24 * time.Hour}
	existing := &dapisv1.DisasterPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "policy-auto-update-ttl",
			Namespace: common.DisasterSystemNamespace,
			UID:       types.UID("policy-auto-update-ttl-uid"),
		},
		Spec: dapisv1.DisasterPolicySpec{
			Type:     dapisv1.PolicyTypeAutoBackup,
			Schedule: "*/10 * * * *",
			TTL:      &initialTTL,
			State:    dapisv1.PolicyStateEnabled,
		},
	}
	h := newMockHandler(existing)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/policies/policy-auto-update-ttl")
	ctx.Params = param.Params{{Key: "name", Value: "policy-auto-update-ttl"}}
	ctx.Request.SetBody([]byte(`{
		"name":"policy-auto-update-ttl",
		"schedule":"0 * * * *"
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.updatePolicy(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	updated, err := h.DisasterClient.DisasterV1().DisasterPolicies(common.DisasterSystemNamespace).Get(
		context.Background(),
		"policy-auto-update-ttl",
		metav1.GetOptions{},
	)
	assert.NoError(t, err)
	if assert.NotNil(t, updated.Spec.TTL) {
		assert.Equal(t, 24*time.Hour, updated.Spec.TTL.Duration)
	}

	clearCtx := app.NewContext(16)
	clearCtx.Request.SetRequestURI("/policies/policy-auto-update-ttl")
	clearCtx.Params = param.Params{{Key: "name", Value: "policy-auto-update-ttl"}}
	clearCtx.Request.SetBody([]byte(`{
		"name":"policy-auto-update-ttl",
		"clearTTL":true
	}`))
	clearCtx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.updatePolicy(context.Background(), clearCtx)

	assert.Equal(t, consts.StatusOK, clearCtx.Response.StatusCode())
	cleared, err := h.DisasterClient.DisasterV1().DisasterPolicies(common.DisasterSystemNamespace).Get(
		context.Background(),
		"policy-auto-update-ttl",
		metav1.GetOptions{},
	)
	assert.NoError(t, err)
	assert.Nil(t, cleared.Spec.TTL)
}

func TestUpdatePolicy_WithReferencedAutoBackupPolicyAllowsScheduleChange(t *testing.T) {
	existing := &dapisv1.DisasterPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "policy-auto-referenced",
			Namespace: common.DisasterSystemNamespace,
			UID:       types.UID("policy-auto-referenced-uid"),
		},
		Spec: dapisv1.DisasterPolicySpec{
			Type:     dapisv1.PolicyTypeAutoBackup,
			Schedule: "*/10 * * * *",
			State:    dapisv1.PolicyStateEnabled,
		},
	}
	appBackup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "referencing-appbackup",
			Namespace: common.DisasterSystemNamespace,
			Labels: map[string]string{
				metadata.LabelDisasterPolicyUID: string(existing.UID),
			},
		},
		Spec: dapisv1.AppBackupSpec{
			DisasterPolicy: existing.Name,
		},
	}
	h := newMockHandler(existing, appBackup)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/policies/policy-auto-referenced")
	ctx.Params = param.Params{{Key: "name", Value: "policy-auto-referenced"}}
	ctx.Request.SetBody([]byte(`{
		"name":"policy-auto-referenced",
		"schedule":"0 2 * * *"
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.updatePolicy(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	updated, err := h.DisasterClient.DisasterV1().DisasterPolicies(common.DisasterSystemNamespace).Get(
		context.Background(),
		"policy-auto-referenced",
		metav1.GetOptions{},
	)
	assert.NoError(t, err)
	assert.Equal(t, "0 2 * * *", updated.Spec.Schedule)
}

func TestUpdatePolicy_WithReferencedSyncPolicyStillConflicts(t *testing.T) {
	existing := &dapisv1.DisasterPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "policy-sync-referenced",
			Namespace: common.DisasterSystemNamespace,
			UID:       types.UID("policy-sync-referenced-uid"),
		},
		Spec: dapisv1.DisasterPolicySpec{
			Type:     dapisv1.PolicyTypeDataSync,
			Schedule: "*/10 * * * *",
			State:    dapisv1.PolicyStateEnabled,
		},
	}
	appBackup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "referencing-sync-appbackup",
			Namespace: common.DisasterSystemNamespace,
			Labels: map[string]string{
				metadata.LabelDisasterPolicyUID: string(existing.UID),
			},
		},
	}
	h := newMockHandler(existing, appBackup)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/policies/policy-sync-referenced")
	ctx.Params = param.Params{{Key: "name", Value: "policy-sync-referenced"}}
	ctx.Request.SetBody([]byte(`{
		"name":"policy-sync-referenced",
		"schedule":"0 2 * * *"
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.updatePolicy(context.Background(), ctx)

	assert.Equal(t, consts.StatusConflict, ctx.Response.StatusCode())
	assert.Contains(t, string(ctx.Response.Body()), "referenced by existing AppBackups")
}

func TestPolicyNames_FilterBySyncPolicyIncludesBothConcreteSyncTypes(t *testing.T) {
	dataPolicy := &dapisv1.DisasterPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "policy-ds",
			Namespace: common.DisasterSystemNamespace,
			UID:       types.UID("uid-ds"),
		},
		Spec: dapisv1.DisasterPolicySpec{
			Type:     dapisv1.PolicyTypeDataSync,
			Schedule: "*/5 * * * *",
			State:    dapisv1.PolicyStateEnabled,
		},
	}
	resourcePolicy := &dapisv1.DisasterPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "policy-rs",
			Namespace: common.DisasterSystemNamespace,
			UID:       types.UID("uid-rs"),
		},
		Spec: dapisv1.DisasterPolicySpec{
			Type:     dapisv1.PolicyTypeResourceSync,
			Schedule: "0 * * * *",
			State:    dapisv1.PolicyStateEnabled,
		},
	}
	autoBackupPolicy := &dapisv1.DisasterPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "policy-ab",
			Namespace: common.DisasterSystemNamespace,
			UID:       types.UID("uid-ab"),
		},
		Spec: dapisv1.DisasterPolicySpec{
			Type:     dapisv1.PolicyTypeAutoBackup,
			Schedule: "30 * * * *",
			State:    dapisv1.PolicyStateEnabled,
		},
	}
	deletingSyncPolicy := &dapisv1.DisasterPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "policy-sync-deleting",
			Namespace: common.DisasterSystemNamespace,
			UID:       types.UID("uid-sync-del"),
		},
		Spec: dapisv1.DisasterPolicySpec{
			Type:     dapisv1.PolicyTypeDataSync,
			Schedule: "45 * * * *",
			State:    dapisv1.PolicyStateEnabled,
		},
		Status: dapisv1.DisasterPolicyStatus{
			Phase: dapisv1.PolicyPhaseDeleting,
		},
	}
	h := newMockHandler(dataPolicy, resourcePolicy, autoBackupPolicy, deletingSyncPolicy)

	stopCh := make(chan struct{})
	defer close(stopCh)
	h.InformerFactory.Start(stopCh)
	for _, ok := range h.InformerFactory.WaitForCacheSync(stopCh) {
		assert.True(t, ok)
	}

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/policies/names?type=SyncPolicy")
	ctx.Request.URI().SetQueryString("type=SyncPolicy")

	h.policyNames(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp struct {
		Code int                     `json:"code"`
		Data []DisasterPolicyNameDTO `json:"data"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)

	gotNames := make([]string, 0, len(resp.Data))
	for _, item := range resp.Data {
		gotNames = append(gotNames, item.Name)
	}

	assert.ElementsMatch(t, []string{"policy-ds", "policy-rs"}, gotNames)
}

func TestPolicyNames_AutoBackupIncludesTTLForSelectionEcho(t *testing.T) {
	ttl := metav1.Duration{Duration: 720 * time.Hour}
	autoBackupPolicy := &dapisv1.DisasterPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "policy-ab-ttl",
			Namespace: common.DisasterSystemNamespace,
			UID:       types.UID("uid-ab-ttl"),
		},
		Spec: dapisv1.DisasterPolicySpec{
			Type:     dapisv1.PolicyTypeAutoBackup,
			Schedule: "0 2 * * *",
			TTL:      &ttl,
			State:    dapisv1.PolicyStateEnabled,
		},
	}
	h := newMockHandler(autoBackupPolicy)

	stopCh := make(chan struct{})
	defer close(stopCh)
	h.InformerFactory.Start(stopCh)
	for _, ok := range h.InformerFactory.WaitForCacheSync(stopCh) {
		assert.True(t, ok)
	}

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/policies/names?type=AutoBackup")
	ctx.Request.URI().SetQueryString("type=AutoBackup")

	h.policyNames(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp struct {
		Code int                     `json:"code"`
		Data []DisasterPolicyNameDTO `json:"data"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	if assert.Len(t, resp.Data, 1) {
		assert.Equal(t, "policy-ab-ttl", resp.Data[0].Name)
		assert.Equal(t, ExternalPolicyTypeAutoBackup, resp.Data[0].Type)
		assert.Equal(t, "0 2 * * *", resp.Data[0].Schedule)
		if assert.NotNil(t, resp.Data[0].TTL) {
			assert.Equal(t, 720*time.Hour, resp.Data[0].TTL.Duration)
		}
	}
}

func TestPolicies_ListResponseNormalizesSyncPolicyLabelOnlyInOutput(t *testing.T) {
	syncPolicy := &dapisv1.DisasterPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "policy-sync-list",
			Namespace: common.DisasterSystemNamespace,
			UID:       types.UID("uid-sync-list"),
			Labels: map[string]string{
				metadata.LabelDisasterPolicyType:  string(dapisv1.PolicyTypeDataSync),
				metadata.LabelDisasterPolicyName:  "policy-sync-list",
				metadata.LabelDisasterPolicyState: string(dapisv1.PolicyStateEnabled),
			},
		},
		Spec: dapisv1.DisasterPolicySpec{
			Type:     dapisv1.PolicyTypeDataSync,
			Schedule: "*/5 * * * *",
			State:    dapisv1.PolicyStateEnabled,
		},
	}
	h := newMockHandler(syncPolicy)

	stopCh := make(chan struct{})
	defer close(stopCh)
	h.InformerFactory.Start(stopCh)
	for _, ok := range h.InformerFactory.WaitForCacheSync(stopCh) {
		assert.True(t, ok)
	}

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/policies?sort=creationTimestamp&order=desc")
	ctx.Request.URI().SetQueryString("sort=creationTimestamp&order=desc")

	h.policies(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Items []DisasterPolicyDTO `json:"items"`
		} `json:"data"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	if assert.Len(t, resp.Data.Items, 1) {
		assert.Equal(t, ExternalPolicyTypeSyncPolicy, resp.Data.Items[0].Spec.Type)
		assert.Equal(t, ExternalPolicyTypeSyncPolicy, resp.Data.Items[0].Labels[metadata.LabelDisasterPolicyType])
	}
	assert.Equal(t, string(dapisv1.PolicyTypeDataSync), syncPolicy.Labels[metadata.LabelDisasterPolicyType])
}

func TestPolicies_SortByCreationTimestampUsesStableNameTieBreaker(t *testing.T) {
	createdAt := metav1.NewTime(time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC))
	policyA := &dapisv1.DisasterPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "policy-a",
			Namespace:         common.DisasterSystemNamespace,
			CreationTimestamp: createdAt,
		},
	}
	policyB := &dapisv1.DisasterPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "policy-b",
			Namespace:         common.DisasterSystemNamespace,
			CreationTimestamp: createdAt,
		},
	}

	q := &transport.Options{Sort: "creationTimestamp", Order: "desc"}
	first := transport.Sort([]*dapisv1.DisasterPolicy{policyA, policyB}, q, compareDisasterPolicies)
	second := transport.Sort([]*dapisv1.DisasterPolicy{policyB, policyA}, q, compareDisasterPolicies)

	assert.Equal(t, []string{"policy-b", "policy-a"}, []string{first[0].Name, first[1].Name})
	assert.Equal(t, []string{"policy-b", "policy-a"}, []string{second[0].Name, second[1].Name})
}
