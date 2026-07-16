package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route/param"
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-operator/pkg/clientset/versioned/fake"
	informers "github.com/softcdata/testudo-operator/pkg/informers/externalversions"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/kube"
	"github.com/softcdata/testudo-server/internal/transport"
	"github.com/stretchr/testify/assert"
	networkingv1 "k8s.io/api/networking/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newMockHandler(objects ...runtime.Object) *InstanceHandler {
	fakeClient := fake.NewSimpleClientset(objects...)
	informerFactory := informers.NewSharedInformerFactory(fakeClient, 0)

	kc := &kube.KubeClient{
		DisasterClient:  fakeClient,
		InformerFactory: informerFactory,
	}

	h := server.Default()
	rg := h.Group("/v1")

	handler := NewInstanceHandler(kc, rg)
	informerFactory.Disaster().V1().DisasterConfigs().Informer()
	informerFactory.Disaster().V1().DisasterInstances().Informer()
	informerFactory.Start(context.Background().Done())
	informerFactory.WaitForCacheSync(context.Background().Done())
	return handler
}

type listInstancesResponse struct {
	Code int `json:"code"`
	Data struct {
		Items []DisasterInstanceDTO `json:"items"`
	} `json:"data"`
	Meta struct {
		Summary map[string]int `json:"summary"`
	} `json:"meta"`
}

type getInstanceResponse struct {
	Code int                 `json:"code"`
	Data DisasterInstanceDTO `json:"data"`
}

type getInstanceGroupsResponse struct {
	Code int               `json:"code"`
	Data InstanceGroupsDTO `json:"data"`
}

type validateTargetResponse struct {
	Code int               `json:"code"`
	Data ValidateTargetDTO `json:"data"`
}

type validateRestoreClassesResponse struct {
	Code int                       `json:"code"`
	Data ValidateRestoreClassesDTO `json:"data"`
}

type syncHistoryResponse struct {
	Code int `json:"code"`
	Data struct {
		Items []SyncHistoryItemDTO `json:"items"`
	} `json:"data"`
	Meta struct {
		Summary map[string]int `json:"summary"`
	} `json:"meta"`
}

type getOperationDetailResponse struct {
	Code int                `json:"code"`
	Data OperationDetailDTO `json:"data"`
}

type protectedNamespaceConflictResponse struct {
	Code    int                            `json:"code"`
	Message string                         `json:"message"`
	Meta    ProtectedNamespaceConflictMeta `json:"meta"`
}

type fakeSubResourceClient struct{}

func useDisasterSystemNamespace(t *testing.T, namespace string) {
	t.Helper()
	oldNamespace := common.DisasterSystemNamespace
	common.SetDisasterSystemNamespace(namespace)
	t.Cleanup(func() {
		common.SetDisasterSystemNamespace(oldNamespace)
	})
}

func (f *fakeSubResourceClient) Get(context.Context, client.Object, client.Object, ...client.SubResourceGetOption) error {
	return nil
}
func (f *fakeSubResourceClient) Create(context.Context, client.Object, client.Object, ...client.SubResourceCreateOption) error {
	return nil
}
func (f *fakeSubResourceClient) Update(context.Context, client.Object, ...client.SubResourceUpdateOption) error {
	return nil
}
func (f *fakeSubResourceClient) Patch(context.Context, client.Object, client.Patch, ...client.SubResourcePatchOption) error {
	return nil
}

type fakeTargetClusterClient struct {
	storageClasses []string
	ingressClasses []string
}

func (f *fakeTargetClusterClient) Get(context.Context, client.ObjectKey, client.Object, ...client.GetOption) error {
	return fmt.Errorf("not implemented")
}

func (f *fakeTargetClusterClient) List(_ context.Context, list client.ObjectList, _ ...client.ListOption) error {
	switch typed := list.(type) {
	case *storagev1.StorageClassList:
		typed.Items = make([]storagev1.StorageClass, 0, len(f.storageClasses))
		for _, name := range f.storageClasses {
			typed.Items = append(typed.Items, storagev1.StorageClass{ObjectMeta: metav1.ObjectMeta{Name: name}})
		}
		return nil
	case *networkingv1.IngressClassList:
		typed.Items = make([]networkingv1.IngressClass, 0, len(f.ingressClasses))
		for _, name := range f.ingressClasses {
			typed.Items = append(typed.Items, networkingv1.IngressClass{ObjectMeta: metav1.ObjectMeta{Name: name}})
		}
		return nil
	default:
		return fmt.Errorf("unsupported list type %T", list)
	}
}

func (f *fakeTargetClusterClient) Create(context.Context, client.Object, ...client.CreateOption) error {
	return fmt.Errorf("not implemented")
}
func (f *fakeTargetClusterClient) Delete(context.Context, client.Object, ...client.DeleteOption) error {
	return fmt.Errorf("not implemented")
}
func (f *fakeTargetClusterClient) Update(context.Context, client.Object, ...client.UpdateOption) error {
	return fmt.Errorf("not implemented")
}
func (f *fakeTargetClusterClient) Patch(context.Context, client.Object, client.Patch, ...client.PatchOption) error {
	return fmt.Errorf("not implemented")
}
func (f *fakeTargetClusterClient) DeleteAllOf(context.Context, client.Object, ...client.DeleteAllOfOption) error {
	return fmt.Errorf("not implemented")
}
func (f *fakeTargetClusterClient) Status() client.SubResourceWriter {
	return &fakeSubResourceClient{}
}
func (f *fakeTargetClusterClient) SubResource(string) client.SubResourceClient {
	return &fakeSubResourceClient{}
}
func (f *fakeTargetClusterClient) Scheme() *runtime.Scheme {
	return runtime.NewScheme()
}
func (f *fakeTargetClusterClient) RESTMapper() meta.RESTMapper {
	return nil
}
func (f *fakeTargetClusterClient) GroupVersionKindFor(runtime.Object) (schema.GroupVersionKind, error) {
	return schema.GroupVersionKind{}, nil
}
func (f *fakeTargetClusterClient) IsObjectNamespaced(runtime.Object) (bool, error) {
	return false, nil
}

func newFakeRemoteClient(storageClasses []string, ingressClasses []string) client.Client {
	return &fakeTargetClusterClient{
		storageClasses: append([]string{}, storageClasses...),
		ingressClasses: append([]string{}, ingressClasses...),
	}
}

func TestDetermineCurrentState_ConfigErrorMappedToFailed(t *testing.T) {
	item := &dapisv1.DisasterInstance{
		Status: dapisv1.DisasterInstanceStatus{
			FsmState: "ConfigError",
		},
	}

	assert.Equal(t, string(dapisv1.CurrentStateFailed), determineCurrentState(item))
}

func TestConvertToDisasterInstanceDTOPreservesConfigErrorFsmState(t *testing.T) {
	item := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-a",
			Namespace: "disaster-system",
			UID:       types.UID("uid-inst-a"),
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Config: "cfg-a",
		},
		Status: dapisv1.DisasterInstanceStatus{
			FsmState: "ConfigError",
			Reason:   "ConfigNotReady",
			Message:  "config is not ready",
		},
	}

	dto := ConvertToDisasterInstanceDTO(item, nil, nil)
	assert.Equal(t, "ConfigError", dto.Status.FsmState)
	assert.Equal(t, string(dapisv1.CurrentStateFailed), dto.CurrentState)
}

func TestConvertToDisasterInstanceDTO_ExposesRoleDriftConditionSummary(t *testing.T) {
	transitionTime := metav1.NewTime(time.Date(2026, 4, 28, 15, 42, 30, 0, time.UTC))
	item := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-role-drift",
			Namespace: "disaster-system",
			UID:       types.UID("uid-role-drift"),
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Config: "cfg-role-drift",
		},
		Status: dapisv1.DisasterInstanceStatus{
			FsmState: "Failed",
			Reason:   "RoleDriftDetected",
			Message:  "both clusters are scaled to zero",
			Conditions: []metav1.Condition{
				{
					Type:               "RoleDrift",
					Status:             metav1.ConditionTrue,
					Reason:             "BothStandby",
					Message:            "both clusters are scaled to zero",
					LastTransitionTime: transitionTime,
					ObservedGeneration: 7,
				},
			},
		},
	}

	dto := ConvertToDisasterInstanceDTO(item, nil, nil)
	if assert.Len(t, dto.Conditions, 1) {
		assert.Equal(t, "RoleDrift", dto.Conditions[0].Type)
		assert.Equal(t, "True", dto.Conditions[0].Status)
		assert.Equal(t, "BothStandby", dto.Conditions[0].Reason)
		assert.Equal(t, "both clusters are scaled to zero", dto.Conditions[0].Message)
		assert.Equal(t, int64(7), dto.Conditions[0].ObservedGeneration)
	}
	if assert.NotNil(t, dto.ConditionSummary) && assert.NotNil(t, dto.ConditionSummary.RoleDrift) {
		assert.Equal(t, "True", dto.ConditionSummary.RoleDrift.Status)
		assert.Equal(t, "BothStandby", dto.ConditionSummary.RoleDrift.Reason)
		assert.Equal(t, "both clusters are scaled to zero", dto.ConditionSummary.RoleDrift.Message)
		assert.True(t, transitionTime.Time.Equal(dto.ConditionSummary.RoleDrift.LastTransitionTime.Time.Time))
	}
}

func TestConvertToDisasterInstanceDTO_EchoesModifierRulesText(t *testing.T) {
	item := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-echo",
			Namespace: "disaster-system",
			UID:       types.UID("uid-inst-echo"),
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Config: "cfg-echo",
			RestorePolicy: &dapisv1.RestorePolicy{
				ModifierRules: []dapisv1.RestoreModifierRule{
					{
						ID:   "rule-echo",
						Mode: dapisv1.RestoreModifierModeReversible,
						Conditions: dapisv1.Conditions{
							GroupResource: "deployments.apps",
						},
						Pair: &dapisv1.RestoreModifierPair{
							Path:        "/metadata/annotations/patched-by",
							SourceValue: "from-a",
							TargetValue: "from-b",
						},
					},
				},
			},
		},
	}

	dto := ConvertToDisasterInstanceDTO(item, nil, nil)
	if assert.NotNil(t, dto.Spec.RestorePolicy) {
		assert.NotEmpty(t, dto.Spec.RestorePolicy.ModifierRulesText)

		var parsed []dapisv1.RestoreModifierRule
		err := json.Unmarshal([]byte(dto.Spec.RestorePolicy.ModifierRulesText), &parsed)
		assert.NoError(t, err)
		if assert.Len(t, parsed, 1) {
			assert.Equal(t, "rule-echo", parsed[0].ID)
			assert.Equal(t, dapisv1.RestoreModifierModeReversible, parsed[0].Mode)
		}
	}
}

func TestConvertToDisasterInstanceDTO_EchoesBulkModifierActionsText(t *testing.T) {
	item := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-bulk-echo",
			Namespace: "disaster-system",
			UID:       types.UID("uid-inst-bulk-echo"),
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Config: "cfg-bulk-echo",
			RestorePolicy: &dapisv1.RestorePolicy{
				BulkModifierActions: []dapisv1.BulkModifierAction{
					{
						ID:          "replace-db-host",
						Action:      dapisv1.BulkModifierActionReplaceExactValue,
						SourceValue: "10.10.0.12",
						TargetValue: "10.20.0.12",
					},
				},
			},
		},
	}

	dto := ConvertToDisasterInstanceDTO(item, nil, nil)
	if assert.NotNil(t, dto.Spec.RestorePolicy) {
		assert.NotEmpty(t, dto.Spec.RestorePolicy.BulkModifierActionsText)

		var parsed []dapisv1.BulkModifierAction
		err := json.Unmarshal([]byte(dto.Spec.RestorePolicy.BulkModifierActionsText), &parsed)
		assert.NoError(t, err)
		if assert.Len(t, parsed, 1) {
			assert.Equal(t, "replace-db-host", parsed[0].ID)
			assert.Equal(t, dapisv1.BulkModifierActionReplaceExactValue, parsed[0].Action)
			assert.Equal(t, "10.10.0.12", parsed[0].SourceValue)
			assert.Equal(t, "10.20.0.12", parsed[0].TargetValue)
		}
	}
}

func TestGetInstance_EchoesModifierRulesText(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-get-echo",
			Namespace: ns,
			UID:       types.UID("uid-inst-get-echo"),
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Config: "cfg-1",
			RestorePolicy: &dapisv1.RestorePolicy{
				ModifierRules: []dapisv1.RestoreModifierRule{
					{
						ID:   "rule-get-echo",
						Mode: dapisv1.RestoreModifierModeVeleroNative,
						Conditions: dapisv1.Conditions{
							GroupResource: "statefulsets.apps",
						},
						VeleroRule: &dapisv1.RestoreModifierVeleroRule{
							Patches: []dapisv1.JSONPatch{
								{
									Operation: "add",
									Path:      "/metadata/annotations/patched-by",
									Value:     "server",
								},
							},
						},
					},
				},
			},
		},
	}

	h := newMockHandler(inst)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-get-echo?namespace=disaster-system")
	ctx.Params = param.Params{{Key: "name", Value: "inst-get-echo"}}

	h.getInstance(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	var resp getInstanceResponse
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	if assert.NotNil(t, resp.Data.Spec.RestorePolicy) {
		assert.NotEmpty(t, resp.Data.Spec.RestorePolicy.ModifierRulesText)
		var parsed []dapisv1.RestoreModifierRule
		err = json.Unmarshal([]byte(resp.Data.Spec.RestorePolicy.ModifierRulesText), &parsed)
		assert.NoError(t, err)
		if assert.Len(t, parsed, 1) {
			assert.Equal(t, "rule-get-echo", parsed[0].ID)
			assert.Equal(t, dapisv1.RestoreModifierModeVeleroNative, parsed[0].Mode)
		}
	}
}

func TestGetInstance_EchoesBulkModifierActionsText(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-get-bulk-echo",
			Namespace: ns,
			UID:       types.UID("uid-inst-get-bulk-echo"),
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Config: "cfg-1",
			RestorePolicy: &dapisv1.RestorePolicy{
				BulkModifierActions: []dapisv1.BulkModifierAction{
					{
						ID:          "replace-db-host",
						Action:      dapisv1.BulkModifierActionReplaceExactValue,
						SourceValue: "10.10.0.12",
						TargetValue: "10.20.0.12",
					},
				},
			},
		},
	}

	h := newMockHandler(inst)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-get-bulk-echo?namespace=disaster-system")
	ctx.Params = param.Params{{Key: "name", Value: "inst-get-bulk-echo"}}

	h.getInstance(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	var resp getInstanceResponse
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	if assert.NotNil(t, resp.Data.Spec.RestorePolicy) {
		assert.NotEmpty(t, resp.Data.Spec.RestorePolicy.BulkModifierActionsText)
		var parsed []dapisv1.BulkModifierAction
		err = json.Unmarshal([]byte(resp.Data.Spec.RestorePolicy.BulkModifierActionsText), &parsed)
		assert.NoError(t, err)
		if assert.Len(t, parsed, 1) {
			assert.Equal(t, "replace-db-host", parsed[0].ID)
			assert.Equal(t, dapisv1.BulkModifierActionReplaceExactValue, parsed[0].Action)
		}
	}
}

func TestGetInstance_EchoesUseUnifiedDirectionResolver(t *testing.T) {
	ns := "disaster-system"
	enabled := true
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-get-unified",
			Namespace: ns,
			UID:       types.UID("uid-inst-get-unified"),
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Config: "cfg-1",
			RestorePolicy: &dapisv1.RestorePolicy{
				UseUnifiedDirectionResolver: &enabled,
			},
		},
	}

	h := newMockHandler(inst)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-get-unified?namespace=disaster-system")
	ctx.Params = param.Params{{Key: "name", Value: "inst-get-unified"}}

	h.getInstance(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	var resp getInstanceResponse
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	if assert.NotNil(t, resp.Data.Spec.RestorePolicy) && assert.NotNil(t, resp.Data.Spec.RestorePolicy.UseUnifiedDirectionResolver) {
		assert.True(t, *resp.Data.Spec.RestorePolicy.UseUnifiedDirectionResolver)
	}
}

func TestCreateInstance_ResponseEchoesUseUnifiedDirectionResolver(t *testing.T) {
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg-1"},
	}
	h := newMockHandler(cfg)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances")
	ctx.Request.SetBody([]byte(`{
		"name":"inst-create-unified",
		"namespace":"disaster-system",
		"config":"cfg-1",
		"restorePolicy":{
			"useUnifiedDirectionResolver":true
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createInstance(context.Background(), ctx)

	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())
	var resp getInstanceResponse
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	if assert.NotNil(t, resp.Data.Spec.RestorePolicy) && assert.NotNil(t, resp.Data.Spec.RestorePolicy.UseUnifiedDirectionResolver) {
		assert.True(t, *resp.Data.Spec.RestorePolicy.UseUnifiedDirectionResolver)
	}
}

func TestListInstances_DefaultSortByCreationTimestampDesc(t *testing.T) {
	ns := "ns-a"
	useDisasterSystemNamespace(t, ns)

	instA := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "inst-a",
			Namespace:         ns,
			UID:               types.UID("uid-a"),
			CreationTimestamp: metav1.NewTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
		},
	}
	instB := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "inst-b",
			Namespace:         ns,
			UID:               types.UID("uid-b"),
			CreationTimestamp: metav1.NewTime(time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)),
		},
	}
	instC := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "inst-c",
			Namespace:         ns,
			UID:               types.UID("uid-c"),
			CreationTimestamp: metav1.NewTime(time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)),
		},
	}

	h := newMockHandler(instA, instB, instC)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances")

	h.listInstances(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp listInstancesResponse
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	assert.Len(t, resp.Data.Items, 3)

	gotNames := []string{
		resp.Data.Items[0].Name,
		resp.Data.Items[1].Name,
		resp.Data.Items[2].Name,
	}
	assert.Equal(t, []string{"inst-b", "inst-c", "inst-a"}, gotNames)
}

func TestListInstances_SortByNameAsc(t *testing.T) {
	ns := "ns-a"
	useDisasterSystemNamespace(t, ns)

	instB := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "b",
			Namespace:         ns,
			UID:               types.UID("uid-b"),
			CreationTimestamp: metav1.NewTime(time.Date(2024, 2, 2, 0, 0, 0, 0, time.UTC)),
		},
	}
	instA := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "a",
			Namespace:         ns,
			UID:               types.UID("uid-a"),
			CreationTimestamp: metav1.NewTime(time.Date(2024, 2, 3, 0, 0, 0, 0, time.UTC)),
		},
	}
	instC := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "c",
			Namespace:         ns,
			UID:               types.UID("uid-c"),
			CreationTimestamp: metav1.NewTime(time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)),
		},
	}

	h := newMockHandler(instB, instA, instC)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances?sort=name&order=asc&limit=-1")

	h.listInstances(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp listInstancesResponse
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	assert.Len(t, resp.Data.Items, 3)

	gotNames := []string{
		resp.Data.Items[0].Name,
		resp.Data.Items[1].Name,
		resp.Data.Items[2].Name,
	}
	assert.Equal(t, []string{"a", "b", "c"}, gotNames)
}

func TestListInstances_FilterByNamespaceAndKeyword(t *testing.T) {
	useDisasterSystemNamespace(t, "ns-a")

	inst1 := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-1",
			Namespace: "ns-a",
			UID:       types.UID("uid-1"),
			Labels: map[string]string{
				"app": "mysql-prod",
			},
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Namespaces: []string{"ns-a"},
		},
	}
	inst2 := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-2",
			Namespace: "ns-a",
			UID:       types.UID("uid-2"),
			Labels: map[string]string{
				"app": "mysql-dev",
			},
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Namespaces: []string{"ns-a"},
		},
	}
	inst3 := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-3",
			Namespace: "ns-b",
			UID:       types.UID("uid-3"),
			Labels: map[string]string{
				"app": "mysql-prod",
			},
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Namespaces: []string{"ns-b"},
		},
	}

	h := newMockHandler(inst1, inst2, inst3)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances?namespace=ns-a&keyword=prod&limit=-1")

	h.listInstances(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp listInstancesResponse
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	assert.Len(t, resp.Data.Items, 1)
	assert.Equal(t, "inst-1", resp.Data.Items[0].Name)
	assert.Equal(t, "ns-a", resp.Data.Items[0].Namespace)
}

func TestListInstances_FilterByProtectedNamespaceNotCRNamespace(t *testing.T) {
	nonMatchingProtectedNamespace := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "non-matching-protected-namespace",
			Namespace: "disaster-system",
			UID:       types.UID("uid-non-matching-protected-namespace"),
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Namespaces: []string{"app-b"},
		},
	}
	protectedNamespaceMatch := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "protected-namespace-match",
			Namespace: "disaster-system",
			UID:       types.UID("uid-protected-namespace-match"),
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Namespaces: []string{"app-a"},
		},
	}

	h := newMockHandler(nonMatchingProtectedNamespace, protectedNamespaceMatch)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances?namespace=app-a&limit=-1")

	h.listInstances(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp listInstancesResponse
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	assert.Len(t, resp.Data.Items, 1)
	assert.Equal(t, "protected-namespace-match", resp.Data.Items[0].Name)
	assert.Equal(t, "disaster-system", resp.Data.Items[0].Namespace)
	assert.Equal(t, []string{"app-a"}, resp.Data.Items[0].Spec.Namespaces)
}

func TestListInstances_FilterByProtectedNamespaceFuzzy(t *testing.T) {
	nonMatchingProtectedNamespace := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "non-matching-protected-namespace",
			Namespace: "disaster-system",
			UID:       types.UID("uid-non-matching-protected-namespace"),
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Namespaces: []string{"prod-nginx-1"},
		},
	}
	protectedNamespaceMatch := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "protected-namespace-match",
			Namespace: "disaster-system",
			UID:       types.UID("uid-protected-namespace-match"),
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Namespaces: []string{"test-nginx-1"},
		},
	}

	h := newMockHandler(nonMatchingProtectedNamespace, protectedNamespaceMatch)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances?namespace=test-nginx&limit=-1")

	h.listInstances(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp listInstancesResponse
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	assert.Len(t, resp.Data.Items, 1)
	assert.Equal(t, "protected-namespace-match", resp.Data.Items[0].Name)
	assert.Equal(t, []string{"test-nginx-1"}, resp.Data.Items[0].Spec.Namespaces)
}

func TestListInstances_KeywordByNameAndProtectedSummary(t *testing.T) {
	ns := "ns-a"
	useDisasterSystemNamespace(t, ns)
	protected := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prod-protected",
			Namespace: ns,
			UID:       types.UID("uid-protected"),
		},
		Status: dapisv1.DisasterInstanceStatus{
			FsmState: dapisv1.FsmStateProtected,
		},
	}
	initializing := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prod-initializing",
			Namespace: ns,
			UID:       types.UID("uid-initializing"),
		},
		Status: dapisv1.DisasterInstanceStatus{
			FsmState: dapisv1.FsmStateInitializing,
		},
	}
	other := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dev-protected",
			Namespace: ns,
			UID:       types.UID("uid-other"),
		},
		Status: dapisv1.DisasterInstanceStatus{
			FsmState: dapisv1.FsmStateProtected,
		},
	}

	h := newMockHandler(protected, initializing, other)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances?keyword=prod&limit=-1")

	h.listInstances(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp listInstancesResponse
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	assert.Len(t, resp.Data.Items, 2)
	assert.Equal(t, 1, resp.Meta.Summary["protectedCount"])

	gotNames := map[string]bool{}
	for _, item := range resp.Data.Items {
		gotNames[item.Name] = true
	}
	assert.True(t, gotNames["prod-protected"])
	assert.True(t, gotNames["prod-initializing"])
	assert.False(t, gotNames["dev-protected"])
}

func TestUpdateInstance_UpdatesDescriptionAnnotation(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "inst-1",
			Namespace:   ns,
			UID:         types.UID("uid-1"),
			Annotations: map[string]string{"testudo.softcdata.com/description": "old desc"},
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Config: "cfg-1",
		},
	}
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cfg-1",
			UID:  types.UID("cfg-uid-1"),
		},
	}

	h := newMockHandler(inst, cfg)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-1?namespace=disaster-system")
	ctx.Params = param.Params{
		{Key: "name", Value: "inst-1"},
	}
	ctx.Request.SetBody([]byte(`{"description":"new desc"}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.updateInstance(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp struct {
		Code int `json:"code"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)

	updated, err := h.DisasterClient.DisasterV1().DisasterInstances(ns).Get(context.Background(), "inst-1", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "new desc", updated.Annotations["testudo.softcdata.com/description"])
}

func TestCreateInstance_Success(t *testing.T) {
	ns := "disaster-system"
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg-1"},
	}

	h := newMockHandler(cfg)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances")
	ctx.Request.SetBody([]byte(`{
		"name":"inst-1",
		"namespace":"disaster-system",
		"config":"cfg-1"
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createInstance(context.Background(), ctx)

	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())
	created, err := h.DisasterClient.DisasterV1().DisasterInstances(ns).Get(context.Background(), "inst-1", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "cfg-1", created.Spec.Config)
}

func TestCreateInstance_WithRestorePolicyAndSkipPodReadyCheck(t *testing.T) {
	ns := "disaster-system"
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg-1"},
	}
	h := newMockHandler(cfg)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances")
	ctx.Request.SetBody([]byte(`{
		"name":"inst-restore-policy",
		"namespace":"disaster-system",
		"config":"cfg-1",
		"skipPodReadyCheck":true,
		"restorePolicy":{
			"execution":{"existingResourcePolicy":"update"},
			"storageClassMapping":{
				"mappings":[{"sourceClass":"standard","targetClass":"gold"}],
				"unmatchedPolicy":"Keep"
			}
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())

	created, err := h.DisasterClient.DisasterV1().DisasterInstances(ns).Get(context.Background(), "inst-restore-policy", metav1.GetOptions{})
	assert.NoError(t, err)
	if assert.NotNil(t, created.Spec.SkipPodReadyCheck) {
		assert.True(t, *created.Spec.SkipPodReadyCheck)
	}
	if assert.NotNil(t, created.Spec.RestorePolicy) {
		if assert.NotNil(t, created.Spec.RestorePolicy.Execution) {
			assert.Equal(t, "update", created.Spec.RestorePolicy.Execution.ExistingResourcePolicy)
		}
		if assert.NotNil(t, created.Spec.RestorePolicy.StorageClassMapping) {
			assert.Len(t, created.Spec.RestorePolicy.StorageClassMapping.Mappings, 1)
			assert.Equal(t, "standard", created.Spec.RestorePolicy.StorageClassMapping.Mappings[0].SourceClass)
			assert.Equal(t, "gold", created.Spec.RestorePolicy.StorageClassMapping.Mappings[0].TargetClass)
		}
	}
}

func TestCreateInstance_WithOperationTimeoutMinutes(t *testing.T) {
	ns := "disaster-system"
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg-1"},
	}
	h := newMockHandler(cfg)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances")
	ctx.Request.SetBody([]byte(`{
		"name":"inst-timeout",
		"namespace":"disaster-system",
		"config":"cfg-1",
		"operationTimeoutMinutes":240
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())

	created, err := h.DisasterClient.DisasterV1().DisasterInstances(ns).Get(context.Background(), "inst-timeout", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, int32(240), created.Spec.OperationTimeoutMinutes)

	var resp getInstanceResponse
	err = json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, int32(240), resp.Data.Spec.OperationTimeoutMinutes)
}

func TestCreateInstance_WithRestorePolicyModifierRules(t *testing.T) {
	ns := "disaster-system"
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg-1"},
	}
	h := newMockHandler(cfg)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances")
	ctx.Request.SetBody([]byte(`{
		"name":"inst-modifier-rules",
		"namespace":"disaster-system",
		"config":"cfg-1",
		"restorePolicy":{
			"useUnifiedDirectionResolver":true,
			"modifierRules":[
				{
					"id":"rule-1",
					"mode":"reversible",
					"applyTo":["dataSync","resourceSync"],
					"priority":10,
					"conditions":{
						"groupResource":"deployments.apps",
						"namespaces":["demo-ns"],
						"resourceNameRegex":"^web-.*"
					},
					"pair":{
						"path":"/metadata/annotations/patched-by",
						"sourceValue":"dr-source",
						"targetValue":"dr-target"
					}
				}
			]
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())

	created, err := h.DisasterClient.DisasterV1().DisasterInstances(ns).Get(context.Background(), "inst-modifier-rules", metav1.GetOptions{})
	assert.NoError(t, err)
	if assert.NotNil(t, created.Spec.RestorePolicy) {
		if assert.NotNil(t, created.Spec.RestorePolicy.UseUnifiedDirectionResolver) {
			assert.True(t, *created.Spec.RestorePolicy.UseUnifiedDirectionResolver)
		}
		if assert.Len(t, created.Spec.RestorePolicy.ModifierRules, 1) {
			rule := created.Spec.RestorePolicy.ModifierRules[0]
			assert.Equal(t, "rule-1", rule.ID)
			assert.Equal(t, dapisv1.RestoreModifierModeReversible, rule.Mode)
			assert.Equal(t, int32(10), rule.Priority)
			if assert.NotNil(t, rule.Pair) {
				assert.Equal(t, "/metadata/annotations/patched-by", rule.Pair.Path)
				assert.Equal(t, "dr-source", rule.Pair.SourceValue)
				assert.Equal(t, "dr-target", rule.Pair.TargetValue)
			}
		}
	}
}

func TestCreateInstance_WithRestorePolicyModifierRulesText(t *testing.T) {
	ns := "disaster-system"
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg-1"},
	}
	h := newMockHandler(cfg)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances")
	ctx.Request.SetBody([]byte(`{
		"name":"inst-modifier-rules-text",
		"namespace":"disaster-system",
		"config":"cfg-1",
		"restorePolicy":{
			"useUnifiedDirectionResolver":true,
			"modifierRulesText":"[{\"id\":\"rule-text\",\"mode\":\"reversible\",\"applyTo\":[\"dataSync\"],\"conditions\":{\"groupResource\":\"deployments.apps\",\"namespaces\":[\"demo-ns\"]},\"pair\":{\"path\":\"/metadata/annotations/patched-by\",\"sourceValue\":\"dr-source\",\"targetValue\":\"dr-target\"}}]"
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())

	created, err := h.DisasterClient.DisasterV1().DisasterInstances(ns).Get(context.Background(), "inst-modifier-rules-text", metav1.GetOptions{})
	assert.NoError(t, err)
	if assert.NotNil(t, created.Spec.RestorePolicy) {
		if assert.NotNil(t, created.Spec.RestorePolicy.UseUnifiedDirectionResolver) {
			assert.True(t, *created.Spec.RestorePolicy.UseUnifiedDirectionResolver)
		}
		if assert.Len(t, created.Spec.RestorePolicy.ModifierRules, 1) {
			assert.Equal(t, "rule-text", created.Spec.RestorePolicy.ModifierRules[0].ID)
			assert.Equal(t, dapisv1.RestoreModifierModeReversible, created.Spec.RestorePolicy.ModifierRules[0].Mode)
		}
	}
}

func TestCreateInstance_WithBulkModifierActionsTextBuildsSnapshot(t *testing.T) {
	ns := "disaster-system"
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg-1"},
	}
	h := newMockHandler(cfg)
	h.BuildBulkModifierSnapshotFunc = func(_ context.Context, spec *dapisv1.DisasterInstanceSpec, _ *rest.Config) (*bulkModifierSnapshotBuildResult, error) {
		if spec == nil || spec.RestorePolicy == nil {
			t.Fatalf("expected restorePolicy for bulk snapshot build")
		}
		if assert.Len(t, spec.RestorePolicy.BulkModifierActions, 1) {
			action := spec.RestorePolicy.BulkModifierActions[0]
			if assert.NotNil(t, action.Enabled) {
				assert.True(t, *action.Enabled)
			}
			assert.Equal(t, []dapisv1.RestoreModifierApplyTarget{dapisv1.RestoreModifierApplyResourceSync}, action.ApplyTo)
			assert.Equal(t, dapisv1.RestoreModifierDirectionPolicyAuto, action.DirectionPolicy)
			assert.Equal(t, "replace-ip-text", action.ID)
		}
		return &bulkModifierSnapshotBuildResult{
			Actions: cloneBulkModifierActions(spec.RestorePolicy.BulkModifierActions),
			ModifierRuleSnapshot: []dapisv1.RestoreModifierRule{{
				ID:   "bulk-text-snapshot-0001",
				Mode: dapisv1.RestoreModifierModeVeleroNative,
				Conditions: dapisv1.Conditions{
					GroupResource: "configmaps",
				},
				VeleroRule: &dapisv1.RestoreModifierVeleroRule{
					Patches: []dapisv1.JSONPatch{{
						Operation: "add",
						Path:      "/metadata/annotations/from-bulk-text",
						Value:     "snapshot",
					}},
				},
			}},
			ModifierRuleSnapshotHash: "sha256:test-bulk-text",
		}, nil
	}

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances")
	ctx.Request.SetBody([]byte(`{
		"name":"inst-bulk-actions-text",
		"namespace":"disaster-system",
		"config":"cfg-1",
		"restorePolicy":{
			"useUnifiedDirectionResolver":true,
			"bulkModifierActionsText":"{\"id\":\"replace-ip-text\",\"action\":\"replaceExactValue\",\"sourceValue\":\"10.10.0.12\",\"targetValue\":\"10.20.0.12\"}"
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())

	created, err := h.DisasterClient.DisasterV1().DisasterInstances(ns).Get(context.Background(), "inst-bulk-actions-text", metav1.GetOptions{})
	assert.NoError(t, err)
	if assert.NotNil(t, created.Spec.RestorePolicy) {
		if assert.Len(t, created.Spec.RestorePolicy.BulkModifierActions, 1) {
			action := created.Spec.RestorePolicy.BulkModifierActions[0]
			assert.Equal(t, "replace-ip-text", action.ID)
			assert.Equal(t, dapisv1.BulkModifierActionReplaceExactValue, action.Action)
		}
		if assert.Len(t, created.Spec.RestorePolicy.ModifierRuleSnapshot, 1) {
			assert.Equal(t, "bulk-text-snapshot-0001", created.Spec.RestorePolicy.ModifierRuleSnapshot[0].ID)
		}
		assert.Equal(t, "sha256:test-bulk-text", created.Spec.RestorePolicy.ModifierRuleSnapshotHash)
	}
}

func TestCreateInstance_WithBulkModifierActionsBuildsSnapshot(t *testing.T) {
	ns := "disaster-system"
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg-1"},
	}
	h := newMockHandler(cfg)
	h.BuildBulkModifierSnapshotFunc = func(_ context.Context, spec *dapisv1.DisasterInstanceSpec, _ *rest.Config) (*bulkModifierSnapshotBuildResult, error) {
		if spec == nil || spec.RestorePolicy == nil {
			t.Fatalf("expected restorePolicy for bulk snapshot build")
		}
		if assert.Len(t, spec.RestorePolicy.BulkModifierActions, 1) {
			action := spec.RestorePolicy.BulkModifierActions[0]
			if assert.NotNil(t, action.Enabled) {
				assert.True(t, *action.Enabled)
			}
			assert.Equal(t, []dapisv1.RestoreModifierApplyTarget{dapisv1.RestoreModifierApplyResourceSync}, action.ApplyTo)
		}
		return &bulkModifierSnapshotBuildResult{
			Actions: cloneBulkModifierActions(spec.RestorePolicy.BulkModifierActions),
			ModifierRuleSnapshot: []dapisv1.RestoreModifierRule{{
				ID:   "bulk-snapshot-0001",
				Mode: dapisv1.RestoreModifierModeVeleroNative,
				Conditions: dapisv1.Conditions{
					GroupResource: "deployments.apps",
				},
				VeleroRule: &dapisv1.RestoreModifierVeleroRule{
					Patches: []dapisv1.JSONPatch{{
						Operation: "add",
						Path:      "/metadata/annotations/from-bulk",
						Value:     "snapshot",
					}},
				},
			}},
			ModifierRuleSnapshotHash: "sha256:test-bulk",
		}, nil
	}

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances")
	ctx.Request.SetBody([]byte(`{
		"name":"inst-bulk-actions",
		"namespace":"disaster-system",
		"config":"cfg-1",
		"restorePolicy":{
			"useUnifiedDirectionResolver":true,
			"bulkModifierActions":[
				{
					"id":"replace-ip",
					"action":"replaceExactValue",
					"sourceValue":"10.10.0.12",
					"targetValue":"10.20.0.12"
				}
			]
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())

	created, err := h.DisasterClient.DisasterV1().DisasterInstances(ns).Get(context.Background(), "inst-bulk-actions", metav1.GetOptions{})
	assert.NoError(t, err)
	if assert.NotNil(t, created.Spec.RestorePolicy) {
		if assert.Len(t, created.Spec.RestorePolicy.BulkModifierActions, 1) {
			action := created.Spec.RestorePolicy.BulkModifierActions[0]
			assert.Equal(t, dapisv1.BulkModifierActionReplaceExactValue, action.Action)
			if assert.NotNil(t, action.Enabled) {
				assert.True(t, *action.Enabled)
			}
			assert.Equal(t, []dapisv1.RestoreModifierApplyTarget{dapisv1.RestoreModifierApplyResourceSync}, action.ApplyTo)
		}
		if assert.Len(t, created.Spec.RestorePolicy.ModifierRuleSnapshot, 1) {
			assert.Equal(t, "bulk-snapshot-0001", created.Spec.RestorePolicy.ModifierRuleSnapshot[0].ID)
		}
		assert.Equal(t, "sha256:test-bulk", created.Spec.RestorePolicy.ModifierRuleSnapshotHash)
	}
}

func TestCreateInstance_WithRewriteImageBulkModifierPersistsRuntimeIntentWithoutSnapshot(t *testing.T) {
	ns := "disaster-system"
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg-1"},
	}
	h := newMockHandler(cfg)
	h.BuildBulkModifierSnapshotFunc = func(context.Context, *dapisv1.DisasterInstanceSpec, *rest.Config) (*bulkModifierSnapshotBuildResult, error) {
		t.Fatalf("rewriteImage should not run static bulk snapshot builder")
		return nil, nil
	}

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances")
	ctx.Request.SetBody([]byte(`{
		"name":"inst-rewrite-image-runtime",
		"namespace":"disaster-system",
		"config":"cfg-1",
		"namespaces":["demo-ns"],
		"restorePolicy":{
			"useUnifiedDirectionResolver":true,
			"bulkModifierActions":[
				{
					"id":"rewrite-primary-registry",
					"action":"rewriteImage",
					"enabled":true,
					"applyTo":["resourceSync","drill"],
					"directionPolicy":"Auto",
					"imageRewrite":{
						"sourcePrefix":"10.11.11.1:5000/",
						"targetPrefix":"registry-test.xxx.xxx.com:30088/dr_images/10_11_11_1_5000/",
						"unmatchedPolicy":"Keep",
						"digestPolicy":"Preserve"
					}
				}
			]
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode(), string(ctx.Response.Body()))

	created, err := h.DisasterClient.DisasterV1().DisasterInstances(ns).Get(context.Background(), "inst-rewrite-image-runtime", metav1.GetOptions{})
	assert.NoError(t, err)
	if assert.NotNil(t, created.Spec.RestorePolicy) {
		if assert.Len(t, created.Spec.RestorePolicy.BulkModifierActions, 1) {
			action := created.Spec.RestorePolicy.BulkModifierActions[0]
			assert.Equal(t, dapisv1.BulkModifierActionRewriteImage, action.Action)
			assert.Equal(t, []dapisv1.RestoreModifierApplyTarget{dapisv1.RestoreModifierApplyResourceSync, dapisv1.RestoreModifierApplyDrill}, action.ApplyTo)
			if assert.NotNil(t, action.ImageRewrite) {
				assert.Equal(t, "10.11.11.1:5000/", action.ImageRewrite.SourcePrefix)
				assert.Equal(t, "registry-test.xxx.xxx.com:30088/dr_images/10_11_11_1_5000/", action.ImageRewrite.TargetPrefix)
				assert.Equal(t, dapisv1.ImageRewriteUnmatchedPolicyKeep, action.ImageRewrite.UnmatchedPolicy)
				assert.Equal(t, dapisv1.ImageRewriteDigestPolicyPreserve, action.ImageRewrite.DigestPolicy)
			}
		}
		assert.Empty(t, created.Spec.RestorePolicy.ModifierRuleSnapshot)
		assert.Empty(t, created.Spec.RestorePolicy.ModifierRuleSnapshotHash)
	}
}

func TestCreateInstance_WithRewriteImageMissingPrefixReturnsBadRequest(t *testing.T) {
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg-1"},
	}
	h := newMockHandler(cfg)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances")
	ctx.Request.SetBody([]byte(`{
		"name":"inst-rewrite-image-invalid",
		"namespace":"disaster-system",
		"config":"cfg-1",
		"restorePolicy":{
			"bulkModifierActions":[
				{
					"id":"rewrite-primary-registry",
					"action":"rewriteImage",
					"imageRewrite":{
						"targetPrefix":"registry-test.xxx.xxx.com:30088/dr_images/10_11_11_1_5000/"
					}
				}
			]
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	assert.Contains(t, string(ctx.Response.Body()), "ModifierRuleRejected")
	assert.Contains(t, string(ctx.Response.Body()), "imageRewrite.sourcePrefix is required")
}

func TestCreateInstance_BulkModifierImageReplacementSkipsForbiddenPodStatusPath(t *testing.T) {
	ns := "disaster-system"
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg-1"},
	}
	h := newMockHandler(cfg)
	h.BuildBulkModifierSnapshotFunc = func(_ context.Context, spec *dapisv1.DisasterInstanceSpec, _ *rest.Config) (*bulkModifierSnapshotBuildResult, error) {
		return bulkImageReplacementSnapshotBuildResultForTest(t, spec), nil
	}

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances")
	ctx.Request.SetBody([]byte(`{
		"name":"inst-bulk-image-skip-status",
		"namespace":"disaster-system",
		"config":"cfg-1",
		"restorePolicy":{
			"useUnifiedDirectionResolver":true,
			"bulkModifierActions":[
				{
					"id":"replace-bkcmdb-synchronizer-image",
					"action":"replaceExactValue",
					"sourceValue":"10.11.11.1:5000/blueking/bcs-bkcmdb-synchronizer:v1.30.0",
					"targetValue":"registry-test.xxx.xxx.com:30088/dr_images/10_11_11_1_5000/blueking/bcs-bkcmdb-synchronizer:v1.30.0"
				}
			]
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode(), string(ctx.Response.Body()))
	assert.NotContains(t, string(ctx.Response.Body()), "/status/containerStatuses/0/image")

	created, err := h.DisasterClient.DisasterV1().DisasterInstances(ns).Get(context.Background(), "inst-bulk-image-skip-status", metav1.GetOptions{})
	assert.NoError(t, err)
	assertBulkImageReplacementSnapshotSkipsStatusPath(t, created.Spec.RestorePolicy)
}

func TestCreateInstance_WithDisabledBulkModifierActionsSkipsSnapshotBuild(t *testing.T) {
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg-1"},
	}
	h := newMockHandler(cfg)
	h.BuildBulkModifierSnapshotFunc = func(context.Context, *dapisv1.DisasterInstanceSpec, *rest.Config) (*bulkModifierSnapshotBuildResult, error) {
		t.Fatalf("bulk snapshot builder should not run when all actions are disabled")
		return nil, nil
	}

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances")
	ctx.Request.SetBody([]byte(`{
		"name":"inst-bulk-disabled",
		"namespace":"disaster-system",
		"config":"cfg-1",
		"restorePolicy":{
			"useUnifiedDirectionResolver":true,
			"bulkModifierActions":[
				{
					"id":"replace-ip",
					"action":"replaceExactValue",
					"sourceValue":"10.10.0.12",
					"targetValue":"10.20.0.12",
					"enabled":false
				}
			]
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())
	assert.NotContains(t, string(ctx.Response.Body()), "ModifierRuleRejected")
}

func TestCreateInstance_WithBulkModifierActionsAndTextEquivalentAccepted(t *testing.T) {
	ns := "disaster-system"
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg-1"},
	}
	h := newMockHandler(cfg)
	h.BuildBulkModifierSnapshotFunc = func(_ context.Context, spec *dapisv1.DisasterInstanceSpec, _ *rest.Config) (*bulkModifierSnapshotBuildResult, error) {
		return &bulkModifierSnapshotBuildResult{
			Actions: cloneBulkModifierActions(spec.RestorePolicy.BulkModifierActions),
		}, nil
	}

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances")
	ctx.Request.SetBody([]byte(`{
		"name":"inst-bulk-actions-equal",
		"namespace":"disaster-system",
		"config":"cfg-1",
		"restorePolicy":{
			"useUnifiedDirectionResolver":true,
			"bulkModifierActions":[
				{
					"id":"replace-ip",
					"action":"replaceExactValue",
					"enabled":true,
					"applyTo":["resourceSync"],
					"directionPolicy":"Auto",
					"sourceValue":"10.10.0.12",
					"targetValue":"10.20.0.12"
				}
			],
			"bulkModifierActionsText":"{\"id\":\"replace-ip\",\"action\":\"replaceExactValue\",\"sourceValue\":\"10.10.0.12\",\"targetValue\":\"10.20.0.12\"}"
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())

	created, err := h.DisasterClient.DisasterV1().DisasterInstances(ns).Get(context.Background(), "inst-bulk-actions-equal", metav1.GetOptions{})
	assert.NoError(t, err)
	if assert.NotNil(t, created.Spec.RestorePolicy) {
		assert.Len(t, created.Spec.RestorePolicy.BulkModifierActions, 1)
	}
}

func TestCreateInstance_WithBulkModifierActionsAndTextConflictReturnsBadRequest(t *testing.T) {
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg-1"},
	}
	h := newMockHandler(cfg)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances")
	ctx.Request.SetBody([]byte(`{
		"name":"inst-bulk-actions-conflict",
		"namespace":"disaster-system",
		"config":"cfg-1",
		"restorePolicy":{
			"useUnifiedDirectionResolver":true,
			"bulkModifierActions":[
				{
					"id":"replace-ip",
					"action":"replaceExactValue",
					"sourceValue":"10.10.0.12",
					"targetValue":"10.20.0.12"
				}
			],
			"bulkModifierActionsText":"{\"id\":\"replace-ip\",\"action\":\"replaceExactValue\",\"sourceValue\":\"10.10.0.12\",\"targetValue\":\"10.30.0.12\"}"
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())

	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeBadRequest, resp.Code)
	assert.Contains(t, resp.Message, "BulkModifierActionsInputConflict")
}

func TestCreateInstance_AllowsProtectedNamespaceConflict(t *testing.T) {
	h := newMockHandler(
		&dapisv1.DisasterConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "cfg-old"},
			Spec: dapisv1.DisasterConfigSpec{
				SourceCluster: "cluster-a",
			},
		},
		&dapisv1.DisasterConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "cfg-new"},
			Spec: dapisv1.DisasterConfigSpec{
				SourceCluster: "cluster-a",
			},
		},
		&dapisv1.DisasterInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "inst-old", Namespace: "disaster-system"},
			Spec: dapisv1.DisasterInstanceSpec{
				Config:     "cfg-old",
				Namespaces: []string{"app-a"},
			},
		},
	)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances")
	ctx.Request.SetBody([]byte(`{
		"name":"inst-new",
		"namespace":"disaster-system",
		"config":"cfg-new",
		"namespaces":["app-a","app-b"]
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createInstance(context.Background(), ctx)

	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())

	created, err := h.DisasterClient.DisasterV1().DisasterInstances("disaster-system").Get(context.Background(), "inst-new", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, []string{"app-a", "app-b"}, created.Spec.Namespaces)
}

func TestCreateInstance_AllowsSameNamespaceAcrossDifferentSourceClusters(t *testing.T) {
	h := newMockHandler(
		&dapisv1.DisasterConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "cfg-a"},
			Spec: dapisv1.DisasterConfigSpec{
				SourceCluster: "cluster-a",
			},
		},
		&dapisv1.DisasterConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "cfg-b"},
			Spec: dapisv1.DisasterConfigSpec{
				SourceCluster: "cluster-b",
			},
		},
		&dapisv1.DisasterInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "inst-old", Namespace: "disaster-system"},
			Spec: dapisv1.DisasterInstanceSpec{
				Config:     "cfg-a",
				Namespaces: []string{"app-a"},
			},
		},
	)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances")
	ctx.Request.SetBody([]byte(`{
		"name":"inst-new",
		"namespace":"disaster-system",
		"config":"cfg-b",
		"namespaces":["app-a"]
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())
}

func TestCreateInstance_WithModifierRulesAndTextConflict(t *testing.T) {
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg-1"},
	}
	h := newMockHandler(cfg)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances")
	ctx.Request.SetBody([]byte(`{
		"name":"inst-modifier-rules-conflict",
		"namespace":"disaster-system",
		"config":"cfg-1",
		"restorePolicy":{
				"modifierRules":[
					{
						"id":"rule-object",
						"mode":"reversible",
						"conditions":{"groupResource":"deployments.apps"},
						"pair":{"path":"/metadata/annotations/a","sourceValue":"x","targetValue":"y"}
					}
				],
				"modifierRulesText":"[{\"id\":\"rule-text\",\"mode\":\"reversible\",\"conditions\":{\"groupResource\":\"deployments.apps\"},\"pair\":{\"path\":\"/metadata/annotations/b\",\"sourceValue\":\"x\",\"targetValue\":\"y\"}}]"
			}
		}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())

	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeBadRequest, resp.Code)
	assert.Contains(t, resp.Message, "ModifierRulesInputConflict")
}

func TestCreateInstance_WithLegacyTransformModifierRulesTextReturnsBadRequest(t *testing.T) {
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg-1"},
	}
	h := newMockHandler(cfg)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances")
	ctx.Request.SetBody([]byte(`{
		"name":"inst-modifier-rules-legacy-transform",
		"namespace":"disaster-system",
		"config":"cfg-1",
		"restorePolicy":{
			"useUnifiedDirectionResolver":true,
			"modifierRulesText":"[{\"id\":\"legacy-rule\",\"mode\":\"reversible\",\"conditions\":{\"groupResource\":\"deployments.apps\"},\"transform\":{\"type\":\"pair\",\"path\":\"/metadata/annotations/legacy\",\"forwardValue\":\"x\",\"reverseValue\":\"y\"}}]"
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	assert.Contains(t, string(ctx.Response.Body()), "ModifierRuleRejected")
	assert.Contains(t, string(ctx.Response.Body()), "pair canonical form")
}

func TestCreateInstance_WithVeleroNativeModifierRuleMissingVeleroRuleReturnsBadRequest(t *testing.T) {
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg-1"},
	}
	h := newMockHandler(cfg)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances")
	ctx.Request.SetBody([]byte(`{
		"name":"inst-modifier-missing-velero",
		"namespace":"disaster-system",
		"config":"cfg-1",
		"namespaces":["demo-ns"],
		"restorePolicy":{
			"useUnifiedDirectionResolver":true,
			"modifierRules":[
				{
					"id":"rule-missing-velero",
					"mode":"veleroNative",
					"conditions":{
						"groupResource":"deployments.apps",
						"namespaces":["demo-ns"]
					}
				}
			]
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	assert.Contains(t, string(ctx.Response.Body()), "ModifierRuleRejected")
	assert.Contains(t, string(ctx.Response.Body()), "missing veleroRule")
}

func TestCreateInstance_WithModifierRuleNamespaceOutsideInstanceScopeReturnsBadRequest(t *testing.T) {
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg-1"},
	}
	h := newMockHandler(cfg)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances")
	ctx.Request.SetBody([]byte(`{
		"name":"inst-modifier-ns-outside",
		"namespace":"disaster-system",
		"config":"cfg-1",
		"namespaces":["demo-ns"],
		"restorePolicy":{
			"useUnifiedDirectionResolver":true,
			"modifierRules":[
				{
					"id":"rule-ns-outside",
					"mode":"reversible",
					"conditions":{
						"groupResource":"deployments.apps",
						"namespaces":["not-exist-ns"]
					},
					"pair":{
						"path":"/metadata/annotations/patched-by",
						"sourceValue":"x",
						"targetValue":"y"
					}
				}
			]
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	assert.Contains(t, string(ctx.Response.Body()), "ModifierRuleRejected")
	assert.Contains(t, string(ctx.Response.Body()), "outside instance namespaces")
}

func TestCreateInstance_WithModifierRulesExceedingLimitReturnsBadRequest(t *testing.T) {
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg-1"},
	}
	h := newMockHandler(cfg)

	rules := make([]map[string]any, 0, maxModifierRulesPerInstance+1)
	for i := 0; i < maxModifierRulesPerInstance+1; i++ {
		rules = append(rules, map[string]any{
			"id":   fmt.Sprintf("rule-%03d", i),
			"mode": "reversible",
			"conditions": map[string]any{
				"groupResource": "deployments.apps",
				"namespaces":    []string{"demo-ns"},
			},
			"pair": map[string]any{
				"path":        "/metadata/annotations/patched-by",
				"sourceValue": "x",
				"targetValue": "y",
			},
		})
	}
	body := map[string]any{
		"name":       "inst-modifier-over-limit",
		"namespace":  "disaster-system",
		"config":     "cfg-1",
		"namespaces": []string{"demo-ns"},
		"restorePolicy": map[string]any{
			"useUnifiedDirectionResolver": true,
			"modifierRules":               rules,
		},
	}
	raw, err := json.Marshal(body)
	assert.NoError(t, err)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances")
	ctx.Request.SetBody(raw)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	assert.Contains(t, string(ctx.Response.Body()), "ModifierRuleRejected")
	assert.Contains(t, string(ctx.Response.Body()), "exceeds limit")
}

func TestCreateInstance_WithForbiddenModifierPathReturnsBadRequest(t *testing.T) {
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg-1"},
	}
	h := newMockHandler(cfg)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances")
	ctx.Request.SetBody([]byte(`{
		"name":"inst-modifier-forbidden-path",
		"namespace":"disaster-system",
		"config":"cfg-1",
		"namespaces":["demo-ns"],
		"restorePolicy":{
			"useUnifiedDirectionResolver":true,
			"modifierRules":[
				{
					"id":"rule-forbidden-path",
					"mode":"reversible",
					"conditions":{
						"groupResource":"deployments.apps",
						"namespaces":["demo-ns"]
					},
					"pair":{
						"path":"/status/phase",
						"sourceValue":"x",
						"targetValue":"y"
					}
				}
			]
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	assert.Contains(t, string(ctx.Response.Body()), "ModifierRuleRejected")
	assert.Contains(t, string(ctx.Response.Body()), "forbidden")
}

func TestUpdateInstance_WithModifierRuleNamespaceOutsideScopeReturnsBadRequest(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-update-modifier-ns-outside",
			Namespace: ns,
			UID:       types.UID("uid-update-modifier-ns-outside"),
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Config:     "cfg-1",
			Namespaces: []string{"demo-ns"},
		},
	}
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg-1"},
	}
	h := newMockHandler(inst, cfg)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-update-modifier-ns-outside?namespace=disaster-system")
	ctx.Params = param.Params{{Key: "name", Value: "inst-update-modifier-ns-outside"}}
	ctx.Request.SetBody([]byte(`{
		"restorePolicy":{
			"useUnifiedDirectionResolver":true,
			"modifierRules":[
				{
					"id":"rule-update-ns-outside",
					"mode":"reversible",
					"conditions":{
						"groupResource":"deployments.apps",
						"namespaces":["another-ns"]
					},
					"pair":{
						"path":"/metadata/annotations/patched-by",
						"sourceValue":"x",
						"targetValue":"y"
					}
				}
			]
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.updateInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	assert.Contains(t, string(ctx.Response.Body()), "ModifierRuleRejected")
	assert.Contains(t, string(ctx.Response.Body()), "outside instance namespaces")
}

func TestUpdateInstance_UpdatesRestorePolicyAndSkipPodReadyCheck(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-update-restore-policy",
			Namespace: ns,
			UID:       types.UID("uid-update-restore-policy"),
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Config: "cfg-1",
		},
	}
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cfg-1",
		},
	}
	h := newMockHandler(inst, cfg)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-update-restore-policy?namespace=disaster-system")
	ctx.Params = param.Params{{Key: "name", Value: "inst-update-restore-policy"}}
	ctx.Request.SetBody([]byte(`{
		"skipPodReadyCheck":false,
		"restorePolicy":{
			"ingressClassMapping":{
				"mappings":[{"sourceClass":"nginx","targetClass":"traefik"}],
				"unmatchedPolicy":"Fail"
			}
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.updateInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	updated, err := h.DisasterClient.DisasterV1().DisasterInstances(ns).Get(context.Background(), "inst-update-restore-policy", metav1.GetOptions{})
	assert.NoError(t, err)
	if assert.NotNil(t, updated.Spec.SkipPodReadyCheck) {
		assert.False(t, *updated.Spec.SkipPodReadyCheck)
	}
	if assert.NotNil(t, updated.Spec.RestorePolicy) && assert.NotNil(t, updated.Spec.RestorePolicy.IngressClassMapping) {
		assert.Len(t, updated.Spec.RestorePolicy.IngressClassMapping.Mappings, 1)
		assert.Equal(t, "nginx", updated.Spec.RestorePolicy.IngressClassMapping.Mappings[0].SourceClass)
		assert.Equal(t, "traefik", updated.Spec.RestorePolicy.IngressClassMapping.Mappings[0].TargetClass)
	}
}

func TestUpdateInstance_UpdatesOperationTimeoutMinutes(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-update-timeout",
			Namespace: ns,
			UID:       types.UID("uid-update-timeout"),
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Config:                  "cfg-1",
			OperationTimeoutMinutes: 60,
		},
	}
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cfg-1",
		},
	}
	h := newMockHandler(inst, cfg)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-update-timeout?namespace=disaster-system")
	ctx.Params = param.Params{{Key: "name", Value: "inst-update-timeout"}}
	ctx.Request.SetBody([]byte(`{"operationTimeoutMinutes":300}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.updateInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	updated, err := h.DisasterClient.DisasterV1().DisasterInstances(ns).Get(context.Background(), "inst-update-timeout", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, int32(300), updated.Spec.OperationTimeoutMinutes)

	var resp getInstanceResponse
	err = json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, int32(300), resp.Data.Spec.OperationTimeoutMinutes)
}

func TestUpdateInstance_UpdatesRestorePolicyModifierRulesText(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-update-modifier-rules-text",
			Namespace: ns,
			UID:       types.UID("uid-update-modifier-rules-text"),
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Config: "cfg-1",
		},
	}
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cfg-1",
		},
	}
	h := newMockHandler(inst, cfg)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-update-modifier-rules-text?namespace=disaster-system")
	ctx.Params = param.Params{{Key: "name", Value: "inst-update-modifier-rules-text"}}
	ctx.Request.SetBody([]byte(`{
		"restorePolicy":{
			"useUnifiedDirectionResolver":true,
			"modifierRulesText":"[{\"id\":\"rule-update-text\",\"mode\":\"veleroNative\",\"conditions\":{\"groupResource\":\"statefulsets.apps\",\"namespaces\":[\"demo-ns\"]},\"veleroRule\":{\"patches\":[{\"operation\":\"add\",\"path\":\"/metadata/annotations/patched-by\",\"value\":\"server\"}]}}]"
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.updateInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	updated, err := h.DisasterClient.DisasterV1().DisasterInstances(ns).Get(context.Background(), "inst-update-modifier-rules-text", metav1.GetOptions{})
	assert.NoError(t, err)
	if assert.NotNil(t, updated.Spec.RestorePolicy) {
		if assert.NotNil(t, updated.Spec.RestorePolicy.UseUnifiedDirectionResolver) {
			assert.True(t, *updated.Spec.RestorePolicy.UseUnifiedDirectionResolver)
		}
		if assert.Len(t, updated.Spec.RestorePolicy.ModifierRules, 1) {
			rule := updated.Spec.RestorePolicy.ModifierRules[0]
			assert.Equal(t, "rule-update-text", rule.ID)
			assert.Equal(t, dapisv1.RestoreModifierModeVeleroNative, rule.Mode)
		}
	}
}

func TestUpdateInstance_UpdatesRestorePolicyBulkModifierActionsText(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-update-bulk-actions-text",
			Namespace: ns,
			UID:       types.UID("uid-update-bulk-actions-text"),
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Config: "cfg-1",
		},
	}
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cfg-1",
		},
	}
	h := newMockHandler(inst, cfg)
	h.BuildBulkModifierSnapshotFunc = func(_ context.Context, spec *dapisv1.DisasterInstanceSpec, _ *rest.Config) (*bulkModifierSnapshotBuildResult, error) {
		return &bulkModifierSnapshotBuildResult{
			Actions: cloneBulkModifierActions(spec.RestorePolicy.BulkModifierActions),
		}, nil
	}

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-update-bulk-actions-text?namespace=disaster-system")
	ctx.Params = param.Params{{Key: "name", Value: "inst-update-bulk-actions-text"}}
	ctx.Request.SetBody([]byte(`{
		"restorePolicy":{
			"useUnifiedDirectionResolver":true,
			"bulkModifierActionsText":"{\"id\":\"replace-ip-update-text\",\"action\":\"replaceExactValue\",\"sourceValue\":\"10.10.0.12\",\"targetValue\":\"10.20.0.12\"}"
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.updateInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	updated, err := h.DisasterClient.DisasterV1().DisasterInstances(ns).Get(context.Background(), "inst-update-bulk-actions-text", metav1.GetOptions{})
	assert.NoError(t, err)
	if assert.NotNil(t, updated.Spec.RestorePolicy) {
		if assert.NotNil(t, updated.Spec.RestorePolicy.UseUnifiedDirectionResolver) {
			assert.True(t, *updated.Spec.RestorePolicy.UseUnifiedDirectionResolver)
		}
		if assert.Len(t, updated.Spec.RestorePolicy.BulkModifierActions, 1) {
			action := updated.Spec.RestorePolicy.BulkModifierActions[0]
			assert.Equal(t, "replace-ip-update-text", action.ID)
			assert.Equal(t, dapisv1.BulkModifierActionReplaceExactValue, action.Action)
		}
	}
}

func TestUpdateInstance_BulkModifierImageReplacementSkipsForbiddenPodStatusPath(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-update-bulk-image-skip-status",
			Namespace: ns,
			UID:       types.UID("uid-update-bulk-image-skip-status"),
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Config: "cfg-1",
		},
	}
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cfg-1",
		},
	}
	h := newMockHandler(inst, cfg)
	h.BuildBulkModifierSnapshotFunc = func(_ context.Context, spec *dapisv1.DisasterInstanceSpec, _ *rest.Config) (*bulkModifierSnapshotBuildResult, error) {
		return bulkImageReplacementSnapshotBuildResultForTest(t, spec), nil
	}

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-update-bulk-image-skip-status?namespace=disaster-system")
	ctx.Params = param.Params{{Key: "name", Value: "inst-update-bulk-image-skip-status"}}
	ctx.Request.SetBody([]byte(`{
		"restorePolicy":{
			"useUnifiedDirectionResolver":true,
			"bulkModifierActions":[
				{
					"id":"replace-bkcmdb-synchronizer-image",
					"action":"replaceExactValue",
					"sourceValue":"10.11.11.1:5000/blueking/bcs-bkcmdb-synchronizer:v1.30.0",
					"targetValue":"registry-test.xxx.xxx.com:30088/dr_images/10_11_11_1_5000/blueking/bcs-bkcmdb-synchronizer:v1.30.0"
				}
			]
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.updateInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode(), string(ctx.Response.Body()))
	assert.NotContains(t, string(ctx.Response.Body()), "/status/containerStatuses/0/image")

	updated, err := h.DisasterClient.DisasterV1().DisasterInstances(ns).Get(context.Background(), "inst-update-bulk-image-skip-status", metav1.GetOptions{})
	assert.NoError(t, err)
	assertBulkImageReplacementSnapshotSkipsStatusPath(t, updated.Spec.RestorePolicy)
}

func TestUpdateInstance_WithInvalidModifierRulesTextReturnsBadRequest(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-update-invalid-modifier-text",
			Namespace: ns,
			UID:       types.UID("uid-update-invalid-modifier-text"),
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Config: "cfg-1",
		},
	}
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cfg-1",
		},
	}
	h := newMockHandler(inst, cfg)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-update-invalid-modifier-text?namespace=disaster-system")
	ctx.Params = param.Params{{Key: "name", Value: "inst-update-invalid-modifier-text"}}
	ctx.Request.SetBody([]byte(`{
		"restorePolicy":{
			"modifierRulesText":"{not-json-array}"
		}
	}`))
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
	assert.Contains(t, resp.Message, "ModifierRulesTextInvalid")
}

func TestUpdateInstance_WithInvalidBulkModifierActionsTextReturnsBadRequest(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-update-invalid-bulk-text",
			Namespace: ns,
			UID:       types.UID("uid-update-invalid-bulk-text"),
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Config: "cfg-1",
		},
	}
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cfg-1",
		},
	}
	h := newMockHandler(inst, cfg)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-update-invalid-bulk-text?namespace=disaster-system")
	ctx.Params = param.Params{{Key: "name", Value: "inst-update-invalid-bulk-text"}}
	ctx.Request.SetBody([]byte(`{
		"restorePolicy":{
			"bulkModifierActionsText":"not-json"
		}
	}`))
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
	assert.Contains(t, resp.Message, "BulkModifierActionsTextInvalid")
}

func TestUpdateInstance_UpdatesRestorePolicyModifierRules(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-update-modifier-rules",
			Namespace: ns,
			UID:       types.UID("uid-update-modifier-rules"),
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Config: "cfg-1",
		},
	}
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cfg-1",
		},
	}
	h := newMockHandler(inst, cfg)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-update-modifier-rules?namespace=disaster-system")
	ctx.Params = param.Params{{Key: "name", Value: "inst-update-modifier-rules"}}
	ctx.Request.SetBody([]byte(`{
		"restorePolicy":{
			"useUnifiedDirectionResolver":true,
			"modifierRules":[
				{
					"id":"rule-update",
					"mode":"veleroNative",
					"conditions":{
						"groupResource":"statefulsets.apps",
						"namespaces":["demo-ns"]
					},
					"veleroRule":{
						"patches":[
							{
								"operation":"add",
								"path":"/metadata/annotations/patched-by",
								"value":"server"
							}
						]
					}
				}
			]
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.updateInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	updated, err := h.DisasterClient.DisasterV1().DisasterInstances(ns).Get(context.Background(), "inst-update-modifier-rules", metav1.GetOptions{})
	assert.NoError(t, err)
	if assert.NotNil(t, updated.Spec.RestorePolicy) {
		if assert.NotNil(t, updated.Spec.RestorePolicy.UseUnifiedDirectionResolver) {
			assert.True(t, *updated.Spec.RestorePolicy.UseUnifiedDirectionResolver)
		}
		if assert.Len(t, updated.Spec.RestorePolicy.ModifierRules, 1) {
			rule := updated.Spec.RestorePolicy.ModifierRules[0]
			assert.Equal(t, "rule-update", rule.ID)
			assert.Equal(t, dapisv1.RestoreModifierModeVeleroNative, rule.Mode)
			if assert.NotNil(t, rule.VeleroRule) && assert.Len(t, rule.VeleroRule.Patches, 1) {
				assert.Equal(t, "add", rule.VeleroRule.Patches[0].Operation)
				assert.Equal(t, "/metadata/annotations/patched-by", rule.VeleroRule.Patches[0].Path)
				assert.Equal(t, "server", rule.VeleroRule.Patches[0].Value)
			}
		}
	}
}

func TestUpdateInstance_ClearsBulkSnapshotWhenBulkActionsAreCleared(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-update-clear-bulk",
			Namespace: ns,
			UID:       types.UID("uid-update-clear-bulk"),
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Config: "cfg-1",
			RestorePolicy: &dapisv1.RestorePolicy{
				UseUnifiedDirectionResolver: boolPtr(true),
				BulkModifierActions: []dapisv1.BulkModifierAction{{
					ID:      "replace-ip",
					Action:  dapisv1.BulkModifierActionReplaceExactValue,
					Enabled: boolPtr(true),
					ApplyTo: []dapisv1.RestoreModifierApplyTarget{dapisv1.RestoreModifierApplyResourceSync},
				}},
				ModifierRuleSnapshot: []dapisv1.RestoreModifierRule{{
					ID:   "bulk-snapshot-0001",
					Mode: dapisv1.RestoreModifierModeVeleroNative,
					Conditions: dapisv1.Conditions{
						GroupResource: "deployments.apps",
					},
					VeleroRule: &dapisv1.RestoreModifierVeleroRule{
						Patches: []dapisv1.JSONPatch{{
							Operation: "add",
							Path:      "/metadata/annotations/from-bulk",
							Value:     "snapshot",
						}},
					},
				}},
				ModifierRuleSnapshotHash: "sha256:old",
			},
		},
	}
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg-1"},
	}
	h := newMockHandler(inst, cfg)
	h.BuildBulkModifierSnapshotFunc = func(context.Context, *dapisv1.DisasterInstanceSpec, *rest.Config) (*bulkModifierSnapshotBuildResult, error) {
		t.Fatalf("bulk snapshot builder should not run when bulk actions are cleared")
		return nil, nil
	}

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-update-clear-bulk?namespace=disaster-system")
	ctx.Params = param.Params{{Key: "name", Value: "inst-update-clear-bulk"}}
	ctx.Request.SetBody([]byte(`{
		"restorePolicy":{
			"useUnifiedDirectionResolver":true,
			"bulkModifierActions":[]
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.updateInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	updated, err := h.DisasterClient.DisasterV1().DisasterInstances(ns).Get(context.Background(), "inst-update-clear-bulk", metav1.GetOptions{})
	assert.NoError(t, err)
	if assert.NotNil(t, updated.Spec.RestorePolicy) {
		assert.Empty(t, updated.Spec.RestorePolicy.ModifierRuleSnapshot)
		assert.Empty(t, updated.Spec.RestorePolicy.ModifierRuleSnapshotHash)
	}
}

func TestUpdateInstance_RestorePolicyPartialModifierSurfaceUpdatePreservesUntouchedInput(t *testing.T) {
	t.Run("only bulk text update keeps manual rules", func(t *testing.T) {
		updated := runModifierSurfacePartialUpdate(t, `{
			"restorePolicy":{
				"bulkModifierActionsText":"{\"id\":\"replace-ip-new\",\"action\":\"replaceExactValue\",\"sourceValue\":\"10.10.0.12\",\"targetValue\":\"10.30.0.12\"}"
			}
		}`)
		assertModifierSurfaceState(t, updated.Spec.RestorePolicy, modifierSurfaceExpect{
			manualIDs:     []string{"manual-old"},
			bulkIDs:       []string{"replace-ip-new"},
			snapshotPaths: []string{"/metadata/annotations/from-bulk-snapshot", "/metadata/annotations/manual-old"},
			snapshotHash:  "sha256:1-actions-1-manual",
			manualTextID:  "manual-old",
			bulkTextID:    "replace-ip-new",
		})
	})

	t.Run("only manual text update keeps bulk actions", func(t *testing.T) {
		updated := runModifierSurfacePartialUpdate(t, `{
			"restorePolicy":{
				"modifierRulesText":"[{\"id\":\"manual-new\",\"mode\":\"veleroNative\",\"conditions\":{\"groupResource\":\"deployments.apps\"},\"veleroRule\":{\"patches\":[{\"operation\":\"add\",\"path\":\"/metadata/annotations/manual-new\",\"value\":\"manual\"}]}}]"
			}
		}`)
		assertModifierSurfaceState(t, updated.Spec.RestorePolicy, modifierSurfaceExpect{
			manualIDs:     []string{"manual-new"},
			bulkIDs:       []string{"replace-ip-old"},
			snapshotPaths: []string{"/metadata/annotations/from-bulk-snapshot", "/metadata/annotations/manual-new"},
			snapshotHash:  "sha256:1-actions-1-manual",
			manualTextID:  "manual-new",
			bulkTextID:    "replace-ip-old",
		})
	})

	t.Run("clear bulk keeps manual rules and clears snapshot", func(t *testing.T) {
		updated := runModifierSurfacePartialUpdate(t, `{
			"restorePolicy":{
				"bulkModifierActions":[]
			}
		}`)
		assertModifierSurfaceState(t, updated.Spec.RestorePolicy, modifierSurfaceExpect{
			manualIDs:    []string{"manual-old"},
			manualTextID: "manual-old",
		})
	})

	t.Run("clear manual keeps bulk actions and rebuilds bulk-only snapshot", func(t *testing.T) {
		updated := runModifierSurfacePartialUpdate(t, `{
			"restorePolicy":{
				"modifierRules":[]
			}
		}`)
		assertModifierSurfaceState(t, updated.Spec.RestorePolicy, modifierSurfaceExpect{
			bulkIDs:       []string{"replace-ip-old"},
			snapshotPaths: []string{"/metadata/annotations/from-bulk-snapshot"},
			snapshotHash:  "sha256:1-actions-0-manual",
			bulkTextID:    "replace-ip-old",
		})
	})
}

type modifierSurfaceExpect struct {
	manualIDs     []string
	bulkIDs       []string
	snapshotPaths []string
	snapshotHash  string
	manualTextID  string
	bulkTextID    string
}

func runModifierSurfacePartialUpdate(t *testing.T, body string) *dapisv1.DisasterInstance {
	t.Helper()

	ns := "disaster-system"
	name := "inst-update-modifier-surface"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			UID:       types.UID("uid-update-modifier-surface"),
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Config: "cfg-1",
			RestorePolicy: &dapisv1.RestorePolicy{
				UseUnifiedDirectionResolver: boolPtr(true),
				BulkModifierActions: []dapisv1.BulkModifierAction{{
					ID:          "replace-ip-old",
					Action:      dapisv1.BulkModifierActionReplaceExactValue,
					Enabled:     boolPtr(true),
					SourceValue: "10.10.0.12",
					TargetValue: "10.20.0.12",
					ApplyTo:     []dapisv1.RestoreModifierApplyTarget{dapisv1.RestoreModifierApplyResourceSync},
				}},
				ModifierRules: []dapisv1.RestoreModifierRule{{
					ID:   "manual-old",
					Mode: dapisv1.RestoreModifierModeVeleroNative,
					Conditions: dapisv1.Conditions{
						GroupResource: "deployments.apps",
					},
					VeleroRule: &dapisv1.RestoreModifierVeleroRule{
						Patches: []dapisv1.JSONPatch{{
							Operation: "add",
							Path:      "/metadata/annotations/manual-old",
							Value:     "manual",
						}},
					},
				}},
				ModifierRuleSnapshot: []dapisv1.RestoreModifierRule{
					{
						ID:   "bulk-snapshot-old",
						Mode: dapisv1.RestoreModifierModeVeleroNative,
						Conditions: dapisv1.Conditions{
							GroupResource: "deployments.apps",
						},
						VeleroRule: &dapisv1.RestoreModifierVeleroRule{
							Patches: []dapisv1.JSONPatch{{
								Operation: "add",
								Path:      "/metadata/annotations/from-bulk-snapshot",
								Value:     "bulk",
							}},
						},
					},
					{
						ID:   "manual-old",
						Mode: dapisv1.RestoreModifierModeVeleroNative,
						Conditions: dapisv1.Conditions{
							GroupResource: "deployments.apps",
						},
						VeleroRule: &dapisv1.RestoreModifierVeleroRule{
							Patches: []dapisv1.JSONPatch{{
								Operation: "add",
								Path:      "/metadata/annotations/manual-old",
								Value:     "manual",
							}},
						},
					},
				},
				ModifierRuleSnapshotHash: "sha256:old",
			},
		},
	}
	cfg := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg-1"},
	}
	h := newMockHandler(inst, cfg)
	h.BuildBulkModifierSnapshotFunc = func(_ context.Context, spec *dapisv1.DisasterInstanceSpec, _ *rest.Config) (*bulkModifierSnapshotBuildResult, error) {
		snapshot := []dapisv1.RestoreModifierRule{{
			ID:   "bulk-snapshot-rebuilt",
			Mode: dapisv1.RestoreModifierModeVeleroNative,
			Conditions: dapisv1.Conditions{
				GroupResource: "deployments.apps",
			},
			VeleroRule: &dapisv1.RestoreModifierVeleroRule{
				Patches: []dapisv1.JSONPatch{{
					Operation: "add",
					Path:      "/metadata/annotations/from-bulk-snapshot",
					Value:     "bulk",
				}},
			},
		}}
		snapshot = append(snapshot, cloneRestoreModifierRules(spec.RestorePolicy.ModifierRules)...)
		return &bulkModifierSnapshotBuildResult{
			Actions:                  cloneBulkModifierActions(spec.RestorePolicy.BulkModifierActions),
			ModifierRuleSnapshot:     snapshot,
			ModifierRuleSnapshotHash: fmt.Sprintf("sha256:%d-actions-%d-manual", len(spec.RestorePolicy.BulkModifierActions), len(spec.RestorePolicy.ModifierRules)),
		}, nil
	}

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/" + name + "?namespace=" + ns)
	ctx.Params = param.Params{{Key: "name", Value: name}}
	ctx.Request.SetBody([]byte(body))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.updateInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode(), string(ctx.Response.Body()))

	updated, err := h.DisasterClient.DisasterV1().DisasterInstances(ns).Get(context.Background(), name, metav1.GetOptions{})
	assert.NoError(t, err)
	return updated
}

func assertModifierSurfaceState(t *testing.T, policy *dapisv1.RestorePolicy, want modifierSurfaceExpect) {
	t.Helper()
	if !assert.NotNil(t, policy) {
		return
	}

	if len(want.manualIDs) == 0 {
		assert.Empty(t, restoreModifierRuleIDs(policy.ModifierRules))
	} else {
		assert.Equal(t, want.manualIDs, restoreModifierRuleIDs(policy.ModifierRules))
	}
	if len(want.bulkIDs) == 0 {
		assert.Empty(t, bulkModifierActionIDs(policy.BulkModifierActions))
	} else {
		assert.Equal(t, want.bulkIDs, bulkModifierActionIDs(policy.BulkModifierActions))
	}
	assert.Equal(t, want.snapshotHash, policy.ModifierRuleSnapshotHash)
	for _, path := range want.snapshotPaths {
		assert.True(t, restoreModifierSnapshotHasPatchPath(policy.ModifierRuleSnapshot, path), "expected snapshot patch path %s", path)
	}
	if len(want.snapshotPaths) == 0 {
		assert.Empty(t, policy.ModifierRuleSnapshot)
	} else {
		assert.Len(t, policy.ModifierRuleSnapshot, len(want.snapshotPaths))
	}

	dto := convertRestorePolicyDTO(policy)
	if want.manualTextID == "" {
		assert.Empty(t, dto.ModifierRulesText)
	} else {
		assert.Contains(t, dto.ModifierRulesText, want.manualTextID)
	}
	if want.bulkTextID == "" {
		assert.Empty(t, dto.BulkModifierActionsText)
	} else {
		assert.Contains(t, dto.BulkModifierActionsText, want.bulkTextID)
	}
}

func restoreModifierRuleIDs(rules []dapisv1.RestoreModifierRule) []string {
	out := make([]string, 0, len(rules))
	for _, rule := range rules {
		out = append(out, rule.ID)
	}
	return out
}

func bulkModifierActionIDs(actions []dapisv1.BulkModifierAction) []string {
	out := make([]string, 0, len(actions))
	for _, action := range actions {
		out = append(out, action.ID)
	}
	return out
}

func restoreModifierSnapshotHasPatchPath(rules []dapisv1.RestoreModifierRule, path string) bool {
	for _, rule := range rules {
		if rule.VeleroRule == nil {
			continue
		}
		for _, patch := range rule.VeleroRule.Patches {
			if patch.Path == path {
				return true
			}
		}
	}
	return false
}

func bulkImageReplacementSnapshotBuildResultForTest(t *testing.T, spec *dapisv1.DisasterInstanceSpec) *bulkModifierSnapshotBuildResult {
	t.Helper()

	if spec == nil || spec.RestorePolicy == nil {
		t.Fatalf("expected restorePolicy for bulk image replacement snapshot")
	}
	const sourceImage = "10.11.11.1:5000/blueking/bcs-bkcmdb-synchronizer:v1.30.0"
	result, err := buildBulkModifierSnapshotFromResources(spec, spec.RestorePolicy.BulkModifierActions, []bulkScannedResource{
		{
			GroupResource: "deployments.apps",
			Namespace:     "demo-ns",
			Name:          "bkcmdb",
			Object: map[string]any{
				"spec": map[string]any{
					"template": map[string]any{
						"spec": map[string]any{
							"containers": []any{
								map[string]any{
									"name":  "synchronizer",
									"image": sourceImage,
								},
							},
						},
					},
				},
			},
		},
		{
			GroupResource: "pods",
			Namespace:     "demo-ns",
			Name:          "bkcmdb-pod",
			Object: map[string]any{
				"spec": map[string]any{
					"containers": []any{
						map[string]any{
							"name":  "synchronizer",
							"image": sourceImage,
						},
					},
				},
				"status": map[string]any{
					"containerStatuses": []any{
						map[string]any{
							"name":  "synchronizer",
							"image": sourceImage,
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("expected bulk image replacement snapshot build success, got %v", err)
	}
	return result
}

func assertBulkImageReplacementSnapshotSkipsStatusPath(t *testing.T, policy *dapisv1.RestorePolicy) {
	t.Helper()

	if !assert.NotNil(t, policy) {
		return
	}
	assert.Len(t, policy.ModifierRuleSnapshot, 2)

	paths := make(map[string]struct{}, len(policy.ModifierRuleSnapshot))
	for _, rule := range policy.ModifierRuleSnapshot {
		if !assert.NotNil(t, rule.Pair) {
			continue
		}
		assert.NotContains(t, rule.Pair.Path, "/status/")
		paths[rule.Conditions.GroupResource+"|"+rule.Pair.Path] = struct{}{}
	}
	assert.Contains(t, paths, "deployments.apps|/spec/template/spec/containers/0/image")
	assert.Contains(t, paths, "pods|/spec/containers/0/image")
	assert.NotContains(t, paths, "pods|/status/containerStatuses/0/image")
	assert.NotEmpty(t, policy.ModifierRuleSnapshotHash)
}

func TestUpdateInstance_AllowsOwnProtectedNamespaces(t *testing.T) {
	h := newMockHandler(
		&dapisv1.DisasterConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "cfg-a"},
			Spec: dapisv1.DisasterConfigSpec{
				SourceCluster: "cluster-a",
			},
		},
		&dapisv1.DisasterInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "inst-a", Namespace: "disaster-system"},
			Spec: dapisv1.DisasterInstanceSpec{
				Config:     "cfg-a",
				Namespaces: []string{"app-a"},
			},
		},
	)

	ctx := app.NewContext(16)
	ctx.Params = param.Params{{Key: "name", Value: "inst-a"}}
	ctx.Request.SetRequestURI("/instances/inst-a?namespace=disaster-system")
	ctx.Request.SetBody([]byte(`{"namespaces":["app-a"]}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.updateInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
}

func TestUpdateInstance_AllowsOtherProtectedNamespaces(t *testing.T) {
	h := newMockHandler(
		&dapisv1.DisasterConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "cfg-a"},
			Spec: dapisv1.DisasterConfigSpec{
				SourceCluster: "cluster-a",
			},
		},
		&dapisv1.DisasterInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "inst-a", Namespace: "disaster-system"},
			Spec: dapisv1.DisasterInstanceSpec{
				Config:     "cfg-a",
				Namespaces: []string{"app-a"},
			},
		},
		&dapisv1.DisasterInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "inst-b", Namespace: "disaster-system"},
			Spec: dapisv1.DisasterInstanceSpec{
				Config:     "cfg-a",
				Namespaces: []string{"app-b"},
			},
		},
	)

	ctx := app.NewContext(16)
	ctx.Params = param.Params{{Key: "name", Value: "inst-a"}}
	ctx.Request.SetRequestURI("/instances/inst-a?namespace=disaster-system")
	ctx.Request.SetBody([]byte(`{"namespaces":["app-b"]}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.updateInstance(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	updated, err := h.DisasterClient.DisasterV1().DisasterInstances("disaster-system").Get(context.Background(), "inst-a", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, []string{"app-b"}, updated.Spec.Namespaces)
}

func TestGetSyncStatus_OnlyFailedSubResourceReturnsCurrentError(t *testing.T) {
	ns := "disaster-system"
	now := metav1.NewTime(time.Now().UTC())

	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-sync-1",
			Namespace: ns,
			UID:       types.UID("uid-sync-1"),
		},
		Status: dapisv1.DisasterInstanceStatus{
			DataSyncName:     "dr-ds-inst-sync-1",
			ResourceSyncName: "dr-rs-inst-sync-1",
		},
	}
	ds := &dapisv1.DataSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dr-ds-inst-sync-1",
			Namespace: ns,
		},
		Status: dapisv1.DataSyncStatus{
			State:        dapisv1.DataSyncStateFailed,
			Reason:       "BackupFailed",
			Message:      "Velero Backup bak-001 failed",
			LastSyncTime: &now,
		},
	}
	rs := &dapisv1.ResourceSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dr-rs-inst-sync-1",
			Namespace: ns,
		},
		Status: dapisv1.ResourceSyncStatus{
			State:        dapisv1.ResourceSyncStateReady,
			Reason:       "ShouldNotLeak",
			Message:      "stale error should be hidden",
			LastSyncTime: &now,
		},
	}

	h := newMockHandler(inst, ds, rs)
	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-sync-1/sync-status?namespace=disaster-system")
	ctx.Request.URI().SetQueryString("namespace=disaster-system")
	ctx.Params = param.Params{{Key: "name", Value: "inst-sync-1"}}

	h.getSyncStatus(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp struct {
		Code int           `json:"code"`
		Data SyncStatusDTO `json:"data"`
	}
	assert.NoError(t, json.Unmarshal(ctx.Response.Body(), &resp))
	assert.Equal(t, transport.CodeOK, resp.Code)
	if assert.NotNil(t, resp.Data.DataSync) {
		assert.Equal(t, "Failed", resp.Data.DataSync.Status)
		assert.Equal(t, "BackupFailed", resp.Data.DataSync.Reason)
		assert.Equal(t, "Velero Backup bak-001 failed", resp.Data.DataSync.Message)
	}
	if assert.NotNil(t, resp.Data.ResourceSync) {
		assert.Equal(t, "Ready", resp.Data.ResourceSync.Status)
		assert.Equal(t, "", resp.Data.ResourceSync.Reason)
		assert.Equal(t, "", resp.Data.ResourceSync.Message)
	}

	// reason/message 必须始终返回（即使为空）以便前端稳定判定。
	var raw map[string]any
	assert.NoError(t, json.Unmarshal(ctx.Response.Body(), &raw))
	data, ok := raw["data"].(map[string]any)
	assert.True(t, ok)
	rsObj, ok := data["resourceSync"].(map[string]any)
	assert.True(t, ok)
	_, hasReason := rsObj["reason"]
	_, hasMessage := rsObj["message"]
	assert.True(t, hasReason)
	assert.True(t, hasMessage)
}

func TestGetSyncStatus_FailedSubResourceFallsBackToConditionError(t *testing.T) {
	ns := "disaster-system"
	now := metav1.NewTime(time.Now().UTC())

	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-sync-2",
			Namespace: ns,
			UID:       types.UID("uid-sync-2"),
		},
		Status: dapisv1.DisasterInstanceStatus{
			DataSyncName: "dr-ds-inst-sync-2",
		},
	}
	ds := &dapisv1.DataSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dr-ds-inst-sync-2",
			Namespace: ns,
		},
		Status: dapisv1.DataSyncStatus{
			State:        dapisv1.DataSyncStateFailed,
			LastSyncTime: &now,
			Conditions: []metav1.Condition{
				{
					Type:               "BackupFailed",
					Status:             metav1.ConditionTrue,
					Reason:             "BackupFailed",
					Message:            "backup job timeout",
					LastTransitionTime: now,
				},
			},
		},
	}

	h := newMockHandler(inst, ds)
	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-sync-2/sync-status?namespace=disaster-system")
	ctx.Request.URI().SetQueryString("namespace=disaster-system")
	ctx.Params = param.Params{{Key: "name", Value: "inst-sync-2"}}

	h.getSyncStatus(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp struct {
		Code int           `json:"code"`
		Data SyncStatusDTO `json:"data"`
	}
	assert.NoError(t, json.Unmarshal(ctx.Response.Body(), &resp))
	assert.Equal(t, transport.CodeOK, resp.Code)
	if assert.NotNil(t, resp.Data.DataSync) {
		assert.Equal(t, "Failed", resp.Data.DataSync.Status)
		assert.Equal(t, "BackupFailed", resp.Data.DataSync.Reason)
		assert.Equal(t, "backup job timeout", resp.Data.DataSync.Message)
	}
}

func TestGetSyncStatus_FailureCountFollowsStatistics(t *testing.T) {
	ns := "disaster-system"
	now := metav1.NewTime(time.Now().UTC())

	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-sync-3",
			Namespace: ns,
			UID:       types.UID("uid-sync-3"),
		},
		Status: dapisv1.DisasterInstanceStatus{
			DataSyncName: "dr-ds-inst-sync-3",
		},
	}
	ds := &dapisv1.DataSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dr-ds-inst-sync-3",
			Namespace: ns,
			UID:       types.UID("uid-ds-sync-3"),
		},
		Status: dapisv1.DataSyncStatus{
			State:        dapisv1.DataSyncStateFailed,
			Reason:       "BuildRestoreSpecFailed",
			Message:      "build restore failed",
			LastSyncTime: &now,
		},
	}
	stats := &dapisv1.BackupRestoreStatistics{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ds-dr-ds-inst-sync-3-stats",
			Namespace: ns,
			Labels: map[string]string{
				"disaster.io/scope-uid": string(ds.UID),
			},
		},
		Spec: dapisv1.BackupRestoreStatisticsSpec{
			ScopeType: dapisv1.ScopeTypeApp,
			ScopeRef: dapisv1.ScopeReference{
				Kind:      "DataSync",
				Name:      ds.Name,
				Namespace: ns,
				UID:       ds.UID,
			},
		},
		Status: dapisv1.BackupRestoreStatisticsStatus{
			Statistics: dapisv1.StatisticsData{
				Completed: 20,
				Failed:    2,
			},
			LastUpdateTime: &now,
		},
	}

	h := newMockHandler(inst, ds, stats)
	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-sync-3/sync-status?namespace=disaster-system")
	ctx.Request.URI().SetQueryString("namespace=disaster-system")
	ctx.Params = param.Params{{Key: "name", Value: "inst-sync-3"}}

	h.getSyncStatus(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	var resp struct {
		Code int           `json:"code"`
		Data SyncStatusDTO `json:"data"`
	}
	assert.NoError(t, json.Unmarshal(ctx.Response.Body(), &resp))
	assert.Equal(t, transport.CodeOK, resp.Code)
	if assert.NotNil(t, resp.Data.DataSync) {
		assert.Equal(t, 20, resp.Data.DataSync.SyncSuccessCount)
		assert.Equal(t, 2, resp.Data.DataSync.SyncFailureCount)
		assert.Equal(t, 2, resp.Data.DataSync.FailureCount)
	}
}

func TestGetSyncStatus_ReturnsLastSyncStatus(t *testing.T) {
	ns := "disaster-system"
	startOld := metav1.NewTime(time.Date(2026, 5, 14, 8, 0, 0, 0, time.UTC))
	endOld := metav1.NewTime(time.Date(2026, 5, 14, 8, 5, 0, 0, time.UTC))
	startNew := metav1.NewTime(time.Date(2026, 5, 14, 9, 0, 0, 0, time.UTC))
	endNew := metav1.NewTime(time.Date(2026, 5, 14, 9, 3, 0, 0, time.UTC))

	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{Name: "inst-sync-last", Namespace: ns},
		Status: dapisv1.DisasterInstanceStatus{
			DataSyncName:     "dr-ds-inst-sync-last",
			ResourceSyncName: "dr-rs-inst-sync-last",
		},
	}
	ds := &dapisv1.DataSync{
		ObjectMeta: metav1.ObjectMeta{Name: "dr-ds-inst-sync-last", Namespace: ns},
		Status: dapisv1.DataSyncStatus{
			State: dapisv1.DataSyncStateReady,
			History: []dapisv1.SyncHistoryRecord{
				{
					StartTime:           &startOld,
					CompletionTime:      &endOld,
					Duration:            "5m",
					BackupName:          "backup-old",
					BackupResourceCount: 1,
					Status:              "Completed",
				},
				{
					StartTime:            &startNew,
					CompletionTime:       &endNew,
					Duration:             "3m",
					BackupName:           "backup-new",
					RestoreName:          "restore-new",
					BackupResourceCount:  10,
					RestoreResourceCount: 8,
					Status:               "Failed",
				},
			},
		},
	}
	rs := &dapisv1.ResourceSync{
		ObjectMeta: metav1.ObjectMeta{Name: "dr-rs-inst-sync-last", Namespace: ns},
		Status: dapisv1.ResourceSyncStatus{
			State: dapisv1.ResourceSyncStateReady,
		},
	}

	h := newMockHandler(inst, ds, rs)
	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-sync-last/sync-status?namespace=disaster-system")
	ctx.Request.URI().SetQueryString("namespace=disaster-system")
	ctx.Params = param.Params{{Key: "name", Value: "inst-sync-last"}}

	h.getSyncStatus(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	var resp struct {
		Code int           `json:"code"`
		Data SyncStatusDTO `json:"data"`
	}
	assert.NoError(t, json.Unmarshal(ctx.Response.Body(), &resp))
	if assert.NotNil(t, resp.Data.DataSync) && assert.NotNil(t, resp.Data.DataSync.LastSyncStatus) {
		assert.Equal(t, "Ready", resp.Data.DataSync.Status)
		assert.Equal(t, "Failed", resp.Data.DataSync.LastSyncStatus.Status)
		assert.Equal(t, "backup-new", resp.Data.DataSync.LastSyncStatus.BackupName)
		assert.Equal(t, "restore-new", resp.Data.DataSync.LastSyncStatus.RestoreName)
		assert.Equal(t, 10, resp.Data.DataSync.LastSyncStatus.BackupResourceCount)
		assert.Equal(t, 8, resp.Data.DataSync.LastSyncStatus.RestoreResourceCount)
	}
	if assert.NotNil(t, resp.Data.ResourceSync) {
		assert.Nil(t, resp.Data.ResourceSync.LastSyncStatus)
	}
}

func TestGetSyncStatus_ReturnsResourceSyncLastSyncStatus(t *testing.T) {
	ns := "disaster-system"
	start := metav1.NewTime(time.Date(2026, 5, 14, 9, 40, 0, 0, time.UTC))
	end := metav1.NewTime(time.Date(2026, 5, 14, 9, 42, 0, 0, time.UTC))

	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{Name: "inst-rs-last", Namespace: ns},
		Status: dapisv1.DisasterInstanceStatus{
			ResourceSyncName: "dr-rs-inst-rs-last",
		},
	}
	rs := &dapisv1.ResourceSync{
		ObjectMeta: metav1.ObjectMeta{Name: "dr-rs-inst-rs-last", Namespace: ns},
		Status: dapisv1.ResourceSyncStatus{
			State: dapisv1.ResourceSyncStateFailed,
			History: []dapisv1.SyncHistoryRecord{
				{
					StartTime:            &start,
					CompletionTime:       &end,
					Duration:             "2m",
					BackupName:           "backup-rs",
					RestoreName:          "restore-rs",
					BackupResourceCount:  7,
					RestoreResourceCount: 6,
					Status:               "Completed",
				},
			},
		},
	}

	h := newMockHandler(inst, rs)
	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-rs-last/sync-status?namespace=disaster-system")
	ctx.Request.URI().SetQueryString("namespace=disaster-system")
	ctx.Params = param.Params{{Key: "name", Value: "inst-rs-last"}}

	h.getSyncStatus(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	var resp struct {
		Code int           `json:"code"`
		Data SyncStatusDTO `json:"data"`
	}
	assert.NoError(t, json.Unmarshal(ctx.Response.Body(), &resp))
	if assert.NotNil(t, resp.Data.ResourceSync) && assert.NotNil(t, resp.Data.ResourceSync.LastSyncStatus) {
		assert.Equal(t, "Failed", resp.Data.ResourceSync.Status)
		assert.Equal(t, "Completed", resp.Data.ResourceSync.LastSyncStatus.Status)
		assert.Equal(t, "backup-rs", resp.Data.ResourceSync.LastSyncStatus.BackupName)
		assert.Equal(t, "restore-rs", resp.Data.ResourceSync.LastSyncStatus.RestoreName)
		assert.Equal(t, 7, resp.Data.ResourceSync.LastSyncStatus.BackupResourceCount)
		assert.Equal(t, 6, resp.Data.ResourceSync.LastSyncStatus.RestoreResourceCount)
	}
}

func TestGetSyncHistory_DefaultReadsSyncRecords(t *testing.T) {
	ns := "disaster-system"
	startA := metav1.NewTime(time.Date(2026, 5, 14, 8, 0, 0, 0, time.UTC))
	endA := metav1.NewTime(time.Date(2026, 5, 14, 8, 5, 0, 0, time.UTC))
	startB := metav1.NewTime(time.Date(2026, 5, 14, 9, 0, 0, 0, time.UTC))
	endB := metav1.NewTime(time.Date(2026, 5, 14, 9, 2, 0, 0, time.UTC))

	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{Name: "inst-sync-history", Namespace: ns},
		Status: dapisv1.DisasterInstanceStatus{
			DataSyncName:     "dr-ds-inst-sync-history",
			ResourceSyncName: "dr-rs-inst-sync-history",
		},
	}
	ds := &dapisv1.DataSync{
		ObjectMeta: metav1.ObjectMeta{Name: "dr-ds-inst-sync-history", Namespace: ns},
		Status: dapisv1.DataSyncStatus{
			History: []dapisv1.SyncHistoryRecord{
				{
					StartTime:           &startA,
					CompletionTime:      &endA,
					BackupName:          "backup-data",
					BackupResourceCount: 3,
					Status:              "Completed",
				},
			},
		},
	}
	rs := &dapisv1.ResourceSync{
		ObjectMeta: metav1.ObjectMeta{Name: "dr-rs-inst-sync-history", Namespace: ns},
		Status: dapisv1.ResourceSyncStatus{
			History: []dapisv1.SyncHistoryRecord{
				{
					StartTime:            &startB,
					CompletionTime:       &endB,
					BackupName:           "backup-resource",
					RestoreName:          "restore-resource",
					BackupResourceCount:  5,
					RestoreResourceCount: 5,
					Status:               "Failed",
				},
			},
		},
	}

	h := newMockHandler(inst, ds, rs)
	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-sync-history/sync-history?namespace=disaster-system&page=1&limit=-1")
	ctx.Request.URI().SetQueryString("namespace=disaster-system&page=1&limit=-1")
	ctx.Params = param.Params{{Key: "name", Value: "inst-sync-history"}}

	h.getSyncHistory(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	var resp syncHistoryResponse
	assert.NoError(t, json.Unmarshal(ctx.Response.Body(), &resp))
	assert.Equal(t, transport.CodeOK, resp.Code)
	assert.Len(t, resp.Data.Items, 2)
	assert.Equal(t, "resourceSync", resp.Data.Items[0].SyncType)
	assert.Equal(t, "syncRecord", resp.Data.Items[0].Source)
	assert.Equal(t, "Failed", resp.Data.Items[0].Status.State)
	assert.Equal(t, "dataSync", resp.Data.Items[1].SyncType)
	assert.Equal(t, 2, resp.Meta.Summary["totalCount"])
	assert.Equal(t, 1, resp.Meta.Summary["dataSyncCount"])
	assert.Equal(t, 1, resp.Meta.Summary["resourceSyncCount"])
	assert.Equal(t, 1, resp.Meta.Summary["completedCount"])
	assert.Equal(t, 1, resp.Meta.Summary["failedCount"])
}

func TestGetSyncHistory_SourceOperationFiltersSyncOperations(t *testing.T) {
	ns := "disaster-system"
	now := metav1.NewTime(time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC))
	later := metav1.NewTime(time.Date(2026, 5, 14, 11, 0, 0, 0, time.UTC))
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{Name: "inst-sync-op", Namespace: ns},
	}
	syncDataOp := &dapisv1.DisasterOperation{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "syncdata-inst-sync-op-1",
			Namespace:         ns,
			UID:               types.UID("op-syncdata"),
			CreationTimestamp: now,
			Labels:            map[string]string{"testudo.softcdata.com/instance": "inst-sync-op"},
		},
		Spec:   dapisv1.DisasterOperationSpec{InstanceName: "inst-sync-op", OperationType: dapisv1.OperationTypeSyncData},
		Status: dapisv1.DisasterOperationStatus{State: dapisv1.OperationStateCompleted, StartTime: &now, CompletionTime: &now},
	}
	syncResourceOp := &dapisv1.DisasterOperation{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "syncresource-inst-sync-op-1",
			Namespace:         ns,
			UID:               types.UID("op-syncresource"),
			CreationTimestamp: later,
			Labels:            map[string]string{"testudo.softcdata.com/instance": "inst-sync-op"},
		},
		Spec:   dapisv1.DisasterOperationSpec{InstanceName: "inst-sync-op", OperationType: dapisv1.OperationTypeSyncResource},
		Status: dapisv1.DisasterOperationStatus{State: dapisv1.OperationStateRunning, StartTime: &later},
	}
	failoverOp := &dapisv1.DisasterOperation{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "failover-inst-sync-op-1",
			Namespace:         ns,
			UID:               types.UID("op-failover"),
			CreationTimestamp: later,
			Labels:            map[string]string{"testudo.softcdata.com/instance": "inst-sync-op"},
		},
		Spec:   dapisv1.DisasterOperationSpec{InstanceName: "inst-sync-op", OperationType: dapisv1.OperationTypeFailover},
		Status: dapisv1.DisasterOperationStatus{State: dapisv1.OperationStateFailed},
	}

	h := newMockHandler(inst, syncDataOp, syncResourceOp, failoverOp)
	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-sync-op/sync-history?namespace=disaster-system&source=operation&page=1&limit=-1")
	ctx.Request.URI().SetQueryString("namespace=disaster-system&source=operation&page=1&limit=-1")
	ctx.Params = param.Params{{Key: "name", Value: "inst-sync-op"}}

	h.getSyncHistory(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	var resp syncHistoryResponse
	assert.NoError(t, json.Unmarshal(ctx.Response.Body(), &resp))
	assert.Len(t, resp.Data.Items, 2)
	assert.Equal(t, "dataSync", resp.Data.Items[0].SyncType)
	assert.Equal(t, "operation", resp.Data.Items[0].Source)
	assert.Equal(t, "syncdata-inst-sync-op-1", resp.Data.Items[0].OperationName)
	assert.Equal(t, "op-syncdata", resp.Data.Items[0].OperationUID)
	assert.True(t, resp.Data.Items[0].HasOperationDetail)
	assert.Equal(t, "resourceSync", resp.Data.Items[1].SyncType)
}

func TestGetSyncHistory_InvalidSourceReturnsBadRequest(t *testing.T) {
	h := newMockHandler()
	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-sync-bad/sync-history?source=bad")
	ctx.Request.URI().SetQueryString("source=bad")
	ctx.Params = param.Params{{Key: "name", Value: "inst-sync-bad"}}

	h.getSyncHistory(context.Background(), ctx)

	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	var resp struct {
		Code int `json:"code"`
	}
	assert.NoError(t, json.Unmarshal(ctx.Response.Body(), &resp))
	assert.Equal(t, transport.CodeBadRequest, resp.Code)
}

func TestGetSyncHistory_InvalidTypeAndStatusReturnBadRequest(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{name: "invalid syncType", query: "syncType=bad"},
		{name: "invalid status", query: "status=bad"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newMockHandler()
			ctx := app.NewContext(16)
			ctx.Request.SetRequestURI("/instances/inst-sync-bad/sync-history?" + tc.query)
			ctx.Request.URI().SetQueryString(tc.query)
			ctx.Params = param.Params{{Key: "name", Value: "inst-sync-bad"}}

			h.getSyncHistory(context.Background(), ctx)

			assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
			var resp struct {
				Code int `json:"code"`
			}
			assert.NoError(t, json.Unmarshal(ctx.Response.Body(), &resp))
			assert.Equal(t, transport.CodeBadRequest, resp.Code)
		})
	}
}

func TestGetSyncHistory_PaginationSummaryUsesFilteredFullSet(t *testing.T) {
	ns := "disaster-system"
	startA := metav1.NewTime(time.Date(2026, 5, 14, 8, 0, 0, 0, time.UTC))
	endA := metav1.NewTime(time.Date(2026, 5, 14, 8, 5, 0, 0, time.UTC))
	startB := metav1.NewTime(time.Date(2026, 5, 14, 9, 0, 0, 0, time.UTC))
	endB := metav1.NewTime(time.Date(2026, 5, 14, 9, 5, 0, 0, time.UTC))
	startC := metav1.NewTime(time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC))
	endC := metav1.NewTime(time.Date(2026, 5, 14, 10, 5, 0, 0, time.UTC))

	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{Name: "inst-sync-page", Namespace: ns},
		Status: dapisv1.DisasterInstanceStatus{
			DataSyncName:     "dr-ds-inst-sync-page",
			ResourceSyncName: "dr-rs-inst-sync-page",
		},
	}
	ds := &dapisv1.DataSync{
		ObjectMeta: metav1.ObjectMeta{Name: "dr-ds-inst-sync-page", Namespace: ns},
		Status: dapisv1.DataSyncStatus{
			History: []dapisv1.SyncHistoryRecord{
				{StartTime: &startA, CompletionTime: &endA, Status: "Failed", BackupName: "data-failed-old"},
				{StartTime: &startC, CompletionTime: &endC, Status: "Failed", BackupName: "data-failed-new"},
			},
		},
	}
	rs := &dapisv1.ResourceSync{
		ObjectMeta: metav1.ObjectMeta{Name: "dr-rs-inst-sync-page", Namespace: ns},
		Status: dapisv1.ResourceSyncStatus{
			History: []dapisv1.SyncHistoryRecord{
				{StartTime: &startB, CompletionTime: &endB, Status: "Completed", BackupName: "resource-completed"},
			},
		},
	}

	h := newMockHandler(inst, ds, rs)
	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-sync-page/sync-history?namespace=disaster-system&status=Failed&page=1&limit=1")
	ctx.Request.URI().SetQueryString("namespace=disaster-system&status=Failed&page=1&limit=1")
	ctx.Params = param.Params{{Key: "name", Value: "inst-sync-page"}}

	h.getSyncHistory(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	var resp syncHistoryResponse
	assert.NoError(t, json.Unmarshal(ctx.Response.Body(), &resp))
	assert.Len(t, resp.Data.Items, 1)
	assert.Equal(t, "data-failed-new", resp.Data.Items[0].BackupName)
	assert.Equal(t, 2, resp.Meta.Summary["totalCount"])
	assert.Equal(t, 2, resp.Meta.Summary["dataSyncCount"])
	assert.Equal(t, 0, resp.Meta.Summary["resourceSyncCount"])
	assert.Equal(t, 0, resp.Meta.Summary["completedCount"])
	assert.Equal(t, 2, resp.Meta.Summary["failedCount"])
}

func TestListInstances_SyncSummaryOnlyShowsCurrentSyncError(t *testing.T) {
	ns := "disaster-system"
	now := metav1.NewTime(time.Now().UTC())

	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "inst-list-sync-1",
			Namespace:         ns,
			UID:               types.UID("uid-list-sync-1"),
			CreationTimestamp: now,
		},
		Status: dapisv1.DisasterInstanceStatus{
			DataSyncName:     "dr-ds-inst-list-sync-1",
			ResourceSyncName: "dr-rs-inst-list-sync-1",
		},
	}
	ds := &dapisv1.DataSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dr-ds-inst-list-sync-1",
			Namespace: ns,
		},
		Status: dapisv1.DataSyncStatus{
			State:        dapisv1.DataSyncStateFailed,
			Reason:       "BackupFailed",
			Message:      "Velero Backup bak-001 failed",
			LastSyncTime: &now,
		},
	}
	rs := &dapisv1.ResourceSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dr-rs-inst-list-sync-1",
			Namespace: ns,
		},
		Status: dapisv1.ResourceSyncStatus{
			State:        dapisv1.ResourceSyncStateReady,
			Reason:       "ShouldNotLeak",
			Message:      "stale error should be hidden",
			LastSyncTime: &now,
		},
	}

	h := newMockHandler(inst, ds, rs)
	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances")

	h.listInstances(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp listInstancesResponse
	assert.NoError(t, json.Unmarshal(ctx.Response.Body(), &resp))
	assert.Equal(t, transport.CodeOK, resp.Code)
	if assert.Len(t, resp.Data.Items, 1) {
		item := resp.Data.Items[0]
		if assert.NotNil(t, item.DataSyncStatus) {
			assert.Equal(t, "Failed", item.DataSyncStatus.State)
			assert.Equal(t, "BackupFailed", item.DataSyncStatus.Reason)
			assert.Equal(t, "Velero Backup bak-001 failed", item.DataSyncStatus.Message)
		}
		if assert.NotNil(t, item.ResourceSyncStatus) {
			assert.Equal(t, "Ready", item.ResourceSyncStatus.State)
			assert.Equal(t, "", item.ResourceSyncStatus.Reason)
			assert.Equal(t, "", item.ResourceSyncStatus.Message)
		}
	}

	// reason/message 必须始终返回（即使为空），以便 watch/list 前端统一判定。
	var raw map[string]any
	assert.NoError(t, json.Unmarshal(ctx.Response.Body(), &raw))
	data, ok := raw["data"].(map[string]any)
	assert.True(t, ok)
	items, ok := data["items"].([]any)
	assert.True(t, ok)
	assert.NotEmpty(t, items)
	itemObj, ok := items[0].(map[string]any)
	assert.True(t, ok)
	rsObj, ok := itemObj["resourceSyncStatus"].(map[string]any)
	assert.True(t, ok)
	_, hasReason := rsObj["reason"]
	_, hasMessage := rsObj["message"]
	assert.True(t, hasReason)
	assert.True(t, hasMessage)
}

func TestGetInstanceGroups_InGroup(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-1",
			Namespace: ns,
			UID:       types.UID("uid-inst-1"),
		},
	}
	groupA := &dapisv1.DisasterGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "group-a",
			Namespace: ns,
			UID:       types.UID("uid-group-a"),
		},
		Spec: dapisv1.DisasterGroupSpec{
			Levels: [][]string{{"inst-1"}},
		},
	}
	groupB := &dapisv1.DisasterGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "group-b",
			Namespace: ns,
			UID:       types.UID("uid-group-b"),
		},
		Spec: dapisv1.DisasterGroupSpec{
			Levels: [][]string{{"inst-x"}, {"inst-1"}},
		},
	}
	groupC := &dapisv1.DisasterGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "group-c",
			Namespace: ns,
			UID:       types.UID("uid-group-c"),
		},
		Spec: dapisv1.DisasterGroupSpec{
			Levels: [][]string{{"inst-y"}},
		},
	}

	h := newMockHandler(inst, groupA, groupB, groupC)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-1/groups?namespace=disaster-system")
	ctx.Params = param.Params{
		{Key: "name", Value: "inst-1"},
	}

	h.getInstanceGroups(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp getInstanceGroupsResponse
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	assert.Equal(t, "inst-1", resp.Data.Instance)
	assert.Equal(t, ns, resp.Data.Namespace)
	assert.True(t, resp.Data.InGroup)
	assert.Equal(t, []string{"group-a", "group-b"}, resp.Data.Groups)
}

func TestGetInstanceGroups_NotInGroup(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-2",
			Namespace: ns,
			UID:       types.UID("uid-inst-2"),
		},
	}
	group := &dapisv1.DisasterGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "group-a",
			Namespace: ns,
			UID:       types.UID("uid-group-a"),
		},
		Spec: dapisv1.DisasterGroupSpec{
			Levels: [][]string{{"inst-1"}},
		},
	}

	h := newMockHandler(inst, group)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-2/groups?namespace=disaster-system")
	ctx.Params = param.Params{
		{Key: "name", Value: "inst-2"},
	}

	h.getInstanceGroups(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp getInstanceGroupsResponse
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	assert.Equal(t, "inst-2", resp.Data.Instance)
	assert.Equal(t, ns, resp.Data.Namespace)
	assert.False(t, resp.Data.InGroup)
	assert.Empty(t, resp.Data.Groups)
}

func TestValidateTarget_InGroupAndOperationAllowed(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-1",
			Namespace: ns,
			UID:       types.UID("uid-inst-1"),
		},
		Status: dapisv1.DisasterInstanceStatus{
			FsmState:            dapisv1.FsmStateActive,
			AvailableOperations: []string{"undo", "reprotect"},
		},
	}
	group := &dapisv1.DisasterGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "group-a",
			Namespace: ns,
			UID:       types.UID("uid-group-a"),
		},
		Spec: dapisv1.DisasterGroupSpec{
			Levels: [][]string{{"inst-1"}},
		},
	}

	h := newMockHandler(inst, group)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-1/validate-target?namespace=disaster-system&operation=undo")
	ctx.Params = param.Params{
		{Key: "name", Value: "inst-1"},
	}

	h.validateTarget(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	var resp validateTargetResponse
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	assert.Equal(t, "inst-1", resp.Data.TargetName)
	assert.Equal(t, ns, resp.Data.Namespace)
	assert.Equal(t, "undo", resp.Data.Operation)
	assert.True(t, resp.Data.Valid)
	assert.Equal(t, dapisv1.FsmStateActive, resp.Data.FsmState)
	assert.True(t, resp.Data.InGroup)
	assert.Equal(t, []string{"group-a"}, resp.Data.Groups)
}

func TestValidateTarget_OperationNotAllowed(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-2",
			Namespace: ns,
			UID:       types.UID("uid-inst-2"),
		},
		Status: dapisv1.DisasterInstanceStatus{
			FsmState:            dapisv1.FsmStateProtected,
			AvailableOperations: []string{"failover", "pause"},
		},
	}

	h := newMockHandler(inst)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-2/validate-target?namespace=disaster-system&operation=undo")
	ctx.Params = param.Params{
		{Key: "name", Value: "inst-2"},
	}

	h.validateTarget(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	var resp validateTargetResponse
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	assert.Equal(t, "inst-2", resp.Data.TargetName)
	assert.Equal(t, "undo", resp.Data.Operation)
	assert.False(t, resp.Data.Valid)
	assert.Equal(t, "OperationNotAllowed", resp.Data.Reason)
	assert.Contains(t, resp.Data.Message, "operation undo is not allowed")
	assert.False(t, resp.Data.InGroup)
	assert.Empty(t, resp.Data.Groups)
}

func TestValidateTarget_AllowsSyncRetryForRecoverableFailedInstance(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-sync-failed",
			Namespace: ns,
			UID:       types.UID("uid-inst-sync-failed"),
		},
		Status: dapisv1.DisasterInstanceStatus{
			FsmState:            dapisv1.FsmStateFailed,
			Reason:              "DataSyncFailed",
			AvailableOperations: []string{"reset"},
		},
	}

	h := newMockHandler(inst)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-sync-failed/validate-target?namespace=disaster-system&operation=sync-data")
	ctx.Params = param.Params{
		{Key: "name", Value: "inst-sync-failed"},
	}

	h.validateTarget(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	var resp validateTargetResponse
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	assert.True(t, resp.Data.Valid)
	assert.Equal(t, "sync-data", resp.Data.Operation)
	assert.Equal(t, dapisv1.FsmStateFailed, resp.Data.FsmState)
	assert.Contains(t, resp.Data.AvailableOperations, "reset")
	assert.Contains(t, resp.Data.AvailableOperations, "syncdata")
}

func TestValidateTarget_AllowsCancelWhileFailingOver(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-failing-over",
			Namespace: ns,
			UID:       types.UID("uid-inst-failing-over"),
		},
		Status: dapisv1.DisasterInstanceStatus{
			FsmState:            dapisv1.FsmStateFailingOver,
			AvailableOperations: []string{},
		},
	}

	h := newMockHandler(inst)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-failing-over/validate-target?namespace=disaster-system&operation=cancel")
	ctx.Params = param.Params{
		{Key: "name", Value: "inst-failing-over"},
	}

	h.validateTarget(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	var resp validateTargetResponse
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	assert.True(t, resp.Data.Valid)
	assert.Equal(t, "cancel", resp.Data.Operation)
	assert.Equal(t, dapisv1.FsmStateFailingOver, resp.Data.FsmState)
	assert.Contains(t, resp.Data.AvailableOperations, "cancel")
}

func TestValidateRestoreClasses_StrictStorageMissingReturnsInvalid(t *testing.T) {
	ns := "disaster-system"
	instance := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-1",
			Namespace: ns,
			UID:       types.UID("uid-inst-1"),
		},
		Status: dapisv1.DisasterInstanceStatus{
			SecondaryCluster: "cluster-b",
		},
	}

	h := newMockHandler(instance)
	h.GetClusterClient = func(ctx context.Context, clusterName string) (client.Client, error) {
		assert.Equal(t, "cluster-b", clusterName)
		return newFakeRemoteClient(nil, nil), nil
	}

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-1/restore-classes/validate?namespace=disaster-system")
	ctx.Request.SetBody([]byte(`{
		"storageClassMapping":{
			"strictTargetValidation":true,
			"mappings":[{"sourceClass":"standard","targetClass":"gold"}]
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	ctx.Params = param.Params{{Key: "name", Value: "inst-1"}}

	h.validateRestoreClasses(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	var resp validateRestoreClassesResponse
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	assert.False(t, resp.Data.Valid)
	assert.Equal(t, "StorageClassTargetNotFound", resp.Data.Code)
	assert.Equal(t, []string{"gold"}, resp.Data.StorageClassCheck.MissingTargets)
}

func TestValidateRestoreClasses_NonStrictIngressMissingReturnsValid(t *testing.T) {
	ns := "disaster-system"
	instance := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-2",
			Namespace: ns,
			UID:       types.UID("uid-inst-2"),
		},
	}

	h := newMockHandler(instance)
	h.GetClusterClient = func(ctx context.Context, clusterName string) (client.Client, error) {
		assert.Equal(t, "cluster-x", clusterName)
		return newFakeRemoteClient(nil, nil), nil
	}

	reqBody := `{
		"targetCluster":"cluster-x",
		"ingressClassMapping":{
			"strictTargetValidation":false,
			"mappings":[{"sourceClass":"nginx","targetClass":"traefik"}]
		}
	}`
	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-2/restore-classes/validate?namespace=disaster-system")
	ctx.Request.SetBody([]byte(reqBody))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	ctx.Params = param.Params{{Key: "name", Value: "inst-2"}}

	h.validateRestoreClasses(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	var resp validateRestoreClassesResponse
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	assert.True(t, resp.Data.Valid)
	assert.Equal(t, []string{"traefik"}, resp.Data.IngressClassCheck.MissingTargets)
	assert.Equal(t, "", resp.Data.Code)
}

func TestValidateRestoreClasses_EmptyMappingsRejected(t *testing.T) {
	ns := "disaster-system"
	instance := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-empty",
			Namespace: ns,
			UID:       types.UID("uid-inst-empty"),
		},
		Status: dapisv1.DisasterInstanceStatus{
			SecondaryCluster: "cluster-e",
		},
	}

	h := newMockHandler(instance)
	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-empty/restore-classes/validate?namespace=disaster-system")
	ctx.Request.SetBody([]byte(`{}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	ctx.Params = param.Params{{Key: "name", Value: "inst-empty"}}

	h.validateRestoreClasses(context.Background(), ctx)

	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	var resp struct {
		Code       int    `json:"code"`
		MessageKey string `json:"message_key"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeBadRequest, resp.Code)
	assert.Equal(t, "validation.class_mapping_required", resp.MessageKey)
}

func TestValidateRestoreClasses_UsesConfigTargetClusterFallback(t *testing.T) {
	ns := "disaster-system"
	instance := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-3",
			Namespace: ns,
			UID:       types.UID("uid-inst-3"),
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Config: "cfg-a",
		},
	}
	config := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cfg-a",
		},
		Spec: dapisv1.DisasterConfigSpec{
			TargetCluster: "cluster-c",
		},
	}

	h := newMockHandler(instance, config)
	h.GetClusterClient = func(ctx context.Context, clusterName string) (client.Client, error) {
		assert.Equal(t, "cluster-c", clusterName)
		return newFakeRemoteClient([]string{"gold"}, nil), nil
	}

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-3/restore-classes/validate?namespace=disaster-system")
	ctx.Request.SetBody([]byte(`{
		"storageClassMapping":{
			"strictTargetValidation":true,
			"mappings":[{"sourceClass":"standard","targetClass":"gold"}]
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	ctx.Params = param.Params{{Key: "name", Value: "inst-3"}}

	h.validateRestoreClasses(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	var resp validateRestoreClassesResponse
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	assert.True(t, resp.Data.Valid)
	assert.Equal(t, "cluster-c", resp.Data.TargetCluster)
	assert.Equal(t, []string{"gold"}, resp.Data.StorageClassCheck.CheckedTargets)
	assert.Empty(t, resp.Data.StorageClassCheck.MissingTargets)
}

func TestValidateRestoreClasses_RequestMappingValidatedDirectly(t *testing.T) {
	ns := "disaster-system"
	instance := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-4",
			Namespace: ns,
			UID:       types.UID("uid-inst-4"),
		},
		Status: dapisv1.DisasterInstanceStatus{
			SecondaryCluster: "cluster-d",
		},
	}

	h := newMockHandler(instance)
	h.GetClusterClient = func(ctx context.Context, clusterName string) (client.Client, error) {
		assert.Equal(t, "cluster-d", clusterName)
		return newFakeRemoteClient([]string{"gold"}, nil), nil
	}

	reqBody := `{
		"storageClassMapping":{
			"strictTargetValidation":true,
			"mappings":[{"sourceClass":"standard","targetClass":"gold"}]
		}
	}`
	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-4/restore-classes/validate?namespace=disaster-system")
	ctx.Request.SetBody([]byte(reqBody))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	ctx.Params = param.Params{{Key: "name", Value: "inst-4"}}

	h.validateRestoreClasses(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	var resp validateRestoreClassesResponse
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	assert.True(t, resp.Data.Valid)
	assert.Equal(t, []string{"gold"}, resp.Data.StorageClassCheck.CheckedTargets)
}

func TestExecuteAction_FailoverPassesSkipScaleDownSource(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-1",
			Namespace: ns,
			UID:       types.UID("uid-1"),
		},
		Status: dapisv1.DisasterInstanceStatus{
			AvailableOperations: []string{"failover"},
		},
	}

	h := newMockHandler(inst)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-1/actions?namespace=disaster-system")
	ctx.Params = param.Params{
		{Key: "name", Value: "inst-1"},
	}
	ctx.Request.SetBody([]byte(`{"operation":"failover","config":{"skipScaleDownSource":true,"force":true,"skipFinalSync":true,"timeoutMinutes":15,"waitUntilReady":true}}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.executeAction(context.Background(), ctx)

	assert.Equal(t, consts.StatusAccepted, ctx.Response.StatusCode())

	ops, err := h.DisasterClient.DisasterV1().DisasterOperations(ns).List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Len(t, ops.Items, 1)

	op := ops.Items[0]
	assert.Equal(t, dapisv1.OperationTypeFailover, op.Spec.OperationType)
	assert.Equal(t, "inst-1", op.Spec.InstanceName)
	assert.Equal(t, "true", op.Annotations[skipScaleDownSourceAnnotation])
	if value, ok := getSkipScaleDownSourceCompat(op.Spec); ok {
		assert.True(t, value)
	}
	assert.True(t, op.Spec.Force)
	assert.True(t, op.Spec.SkipFinalSync)
	assert.Equal(t, int32(15), op.Spec.TimeoutMinutes)
	assert.True(t, op.Spec.WaitUntilReady)
	if assert.NotNil(t, op.Spec.SkipPodReadyCheck) {
		assert.False(t, *op.Spec.SkipPodReadyCheck)
	}
}

func TestExecuteAction_SkipPodReadyCheckOverridesWaitUntilReady(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-override",
			Namespace: ns,
			UID:       types.UID("uid-override"),
		},
		Status: dapisv1.DisasterInstanceStatus{
			AvailableOperations: []string{"failover"},
		},
	}

	h := newMockHandler(inst)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-override/actions?namespace=disaster-system")
	ctx.Params = param.Params{
		{Key: "name", Value: "inst-override"},
	}
	ctx.Request.SetBody([]byte(`{"operation":"failover","config":{"skipPodReadyCheck":true,"waitUntilReady":true}}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.executeAction(context.Background(), ctx)
	assert.Equal(t, consts.StatusAccepted, ctx.Response.StatusCode())

	ops, err := h.DisasterClient.DisasterV1().DisasterOperations(ns).List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Len(t, ops.Items, 1)

	op := ops.Items[0]
	if assert.NotNil(t, op.Spec.SkipPodReadyCheck) {
		assert.True(t, *op.Spec.SkipPodReadyCheck)
	}
	assert.False(t, op.Spec.WaitUntilReady)
}

func TestExecuteAction_NonFailoverIgnoresSkipScaleDownSource(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-1",
			Namespace: ns,
			UID:       types.UID("uid-1"),
		},
		Status: dapisv1.DisasterInstanceStatus{
			AvailableOperations: []string{"reprotect"},
		},
	}

	h := newMockHandler(inst)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-1/actions?namespace=disaster-system")
	ctx.Params = param.Params{
		{Key: "name", Value: "inst-1"},
	}
	ctx.Request.SetBody([]byte(`{"operation":"reprotect","config":{"skipScaleDownSource":true}}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.executeAction(context.Background(), ctx)

	assert.Equal(t, consts.StatusAccepted, ctx.Response.StatusCode())

	ops, err := h.DisasterClient.DisasterV1().DisasterOperations(ns).List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Len(t, ops.Items, 1)

	op := ops.Items[0]
	assert.Equal(t, dapisv1.OperationTypeReprotect, op.Spec.OperationType)
	assert.Empty(t, op.Annotations[skipScaleDownSourceAnnotation])
	if value, ok := getSkipScaleDownSourceCompat(op.Spec); ok {
		assert.False(t, value)
	}
}

func TestExecuteAction_InstanceInGroupAllowed(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-1",
			Namespace: ns,
			UID:       types.UID("uid-1"),
		},
		Status: dapisv1.DisasterInstanceStatus{
			AvailableOperations: []string{"undo"},
		},
	}
	group := &dapisv1.DisasterGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "group-1",
			Namespace: ns,
			UID:       types.UID("group-uid-1"),
		},
		Spec: dapisv1.DisasterGroupSpec{
			Levels: [][]string{
				{"inst-1"},
			},
		},
	}

	h := newMockHandler(inst, group)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-1/actions?namespace=disaster-system")
	ctx.Params = param.Params{
		{Key: "name", Value: "inst-1"},
	}
	ctx.Request.SetBody([]byte(`{"operation":"undo","config":{}}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.executeAction(context.Background(), ctx)

	assert.Equal(t, 202, ctx.Response.StatusCode())

	ops, err := h.DisasterClient.DisasterV1().DisasterOperations(ns).List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Len(t, ops.Items, 1)
	assert.Equal(t, "inst-1", ops.Items[0].Spec.InstanceName)
	assert.Equal(t, dapisv1.OperationTypeUndo, ops.Items[0].Spec.OperationType)
}

func TestExecuteAction_AllowsSyncRetryForRecoverableFailedInstance(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-sync-failed",
			Namespace: ns,
			UID:       types.UID("uid-inst-sync-failed"),
		},
		Status: dapisv1.DisasterInstanceStatus{
			FsmState:            dapisv1.FsmStateFailed,
			Reason:              "DataSyncFailed",
			AvailableOperations: []string{"reset"},
			DataSyncName:        "dr-ds-inst-sync-failed",
		},
	}

	h := newMockHandler(inst)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-sync-failed/actions?namespace=disaster-system")
	ctx.Params = param.Params{
		{Key: "name", Value: "inst-sync-failed"},
	}
	ctx.Request.SetBody([]byte(`{"operation":"sync-data","config":{"timeoutMinutes":3}}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.executeAction(context.Background(), ctx)

	assert.Equal(t, 202, ctx.Response.StatusCode())

	ops, err := h.DisasterClient.DisasterV1().DisasterOperations(ns).List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Len(t, ops.Items, 1)
	assert.Equal(t, "inst-sync-failed", ops.Items[0].Spec.InstanceName)
	assert.Equal(t, dapisv1.OperationTypeSyncData, ops.Items[0].Spec.OperationType)
}

func TestExecuteAction_AllowsCancelWhileFailingOver(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-failing-over",
			Namespace: ns,
			UID:       types.UID("uid-inst-failing-over"),
		},
		Status: dapisv1.DisasterInstanceStatus{
			FsmState:            dapisv1.FsmStateFailingOver,
			AvailableOperations: []string{},
		},
	}

	h := newMockHandler(inst)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-failing-over/actions?namespace=disaster-system")
	ctx.Params = param.Params{
		{Key: "name", Value: "inst-failing-over"},
	}
	ctx.Request.SetBody([]byte(`{"operation":"cancel","config":{"timeoutMinutes":3}}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.executeAction(context.Background(), ctx)

	assert.Equal(t, 202, ctx.Response.StatusCode())

	ops, err := h.DisasterClient.DisasterV1().DisasterOperations(ns).List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Len(t, ops.Items, 1)
	assert.Equal(t, "inst-failing-over", ops.Items[0].Spec.InstanceName)
	assert.Equal(t, dapisv1.OperationTypeCancel, ops.Items[0].Spec.OperationType)
	assert.Equal(t, int32(3), ops.Items[0].Spec.TimeoutMinutes)
}

func TestExecuteAction_OperationNotAllowedDoesNotCreateOperation(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-role-drift",
			Namespace: ns,
			UID:       types.UID("uid-role-drift"),
		},
		Status: dapisv1.DisasterInstanceStatus{
			FsmState:            "Failed",
			Reason:              "RoleDriftDetected",
			Message:             "both clusters are scaled to zero",
			AvailableOperations: []string{},
		},
	}

	h := newMockHandler(inst)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-role-drift/actions?namespace=disaster-system")
	ctx.Params = param.Params{
		{Key: "name", Value: "inst-role-drift"},
	}
	ctx.Request.SetBody([]byte(`{"operation":"failover","config":{"force":true,"skipFinalSync":true}}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.executeAction(context.Background(), ctx)

	assert.Equal(t, consts.StatusConflict, ctx.Response.StatusCode())
	assert.Contains(t, string(ctx.Response.Body()), "OperationNotAllowed")
	assert.Contains(t, string(ctx.Response.Body()), "operation failover is not allowed")

	ops, err := h.DisasterClient.DisasterV1().DisasterOperations(ns).List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Empty(t, ops.Items)
}

func TestConvertToHistoryDTO_IncludesOperationIdentity(t *testing.T) {
	op := &dapisv1.DisasterOperation{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "failover-inst-a-1",
			Namespace:         "disaster-system",
			UID:               types.UID("op-uid-1"),
			CreationTimestamp: metav1.NewTime(time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC)),
		},
		Spec: dapisv1.DisasterOperationSpec{
			InstanceName:  "inst-a",
			OperationType: dapisv1.OperationTypeFailover,
		},
		Status: dapisv1.DisasterOperationStatus{
			State:   dapisv1.OperationStateFailed,
			Reason:  "StepFailed",
			Message: "step failed",
		},
	}

	dto := ConvertToHistoryDTO(op)
	assert.Equal(t, "failover-inst-a-1", dto.OperationName)
	assert.Equal(t, "op-uid-1", dto.OperationUID)
	assert.True(t, dto.HasDetail)
}

func TestGetOperationDetail_ReturnsProjectedDetail(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-op",
			Namespace: ns,
			UID:       types.UID("inst-op-uid"),
		},
	}
	start := metav1.NewTime(time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC))
	op := &dapisv1.DisasterOperation{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "failover-inst-op-1",
			Namespace:         ns,
			UID:               types.UID("op-detail-uid"),
			CreationTimestamp: start,
		},
		Spec: dapisv1.DisasterOperationSpec{
			InstanceName:  "inst-op",
			OperationType: dapisv1.OperationTypeFailover,
		},
		Status: dapisv1.DisasterOperationStatus{
			State:       dapisv1.OperationStateRunning,
			Reason:      "Running",
			CurrentStep: "ScaleUpTarget",
			Message:     "step running",
			Steps: []dapisv1.StepStatus{
				{Name: "PreCheck", State: "Completed"},
				{Name: "ScaleUpTarget", State: "Running", StartTime: &start},
			},
			RoleStatus: &dapisv1.RoleStatus{
				PrimaryCluster:   "cluster-b",
				SecondaryCluster: "cluster-a",
			},
			StartTime:             &start,
			AutoCancelTriggered:   true,
			AutoCancelStatus:      dapisv1.OperationAutoCancelStatusRunning,
			AutoCancelTriggerStep: "ScaleDownSource",
		},
	}

	h := newMockHandler(inst, op)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-op/operations/failover-inst-op-1?namespace=disaster-system")
	ctx.Params = param.Params{
		{Key: "name", Value: "inst-op"},
		{Key: "operationName", Value: "failover-inst-op-1"},
	}

	h.getOperationDetail(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp getOperationDetailResponse
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	assert.Equal(t, "failover-inst-op-1", resp.Data.Name)
	assert.Equal(t, "op-detail-uid", resp.Data.UID)
	assert.Equal(t, "DisasterInstance", resp.Data.OwnerKind)
	assert.Equal(t, "inst-op", resp.Data.OwnerName)
	assert.Equal(t, "ScaleUpTarget", resp.Data.CurrentStep)
	if assert.Len(t, resp.Data.Steps, 2) {
		assert.Equal(t, "PreCheck", resp.Data.Steps[0].Name)
		assert.Equal(t, "ScaleUpTarget", resp.Data.Steps[1].Name)
	}
	if assert.NotNil(t, resp.Data.RoleStatus) {
		assert.Equal(t, "cluster-b", resp.Data.RoleStatus.PrimaryCluster)
	}
	if assert.NotNil(t, resp.Data.AutoCancel) {
		assert.True(t, resp.Data.AutoCancel.Triggered)
		assert.Equal(t, string(dapisv1.OperationAutoCancelStatusRunning), resp.Data.AutoCancel.Status)
		assert.Equal(t, "ScaleDownSource", resp.Data.AutoCancel.TriggerStep)
	}
	assert.True(t, resp.Data.CreationTimestamp.Time.UTC().Equal(start.Time.UTC()))
	assert.Nil(t, resp.Data.CompletionTime)
}

func TestGetOperationDetail_ReturnsNotFoundWhenOwnerMismatch(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-a",
			Namespace: ns,
			UID:       types.UID("inst-a-uid"),
		},
	}
	op := &dapisv1.DisasterOperation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "failover-inst-b-1",
			Namespace: ns,
			UID:       types.UID("op-owner-mismatch"),
		},
		Spec: dapisv1.DisasterOperationSpec{
			InstanceName:  "inst-b",
			OperationType: dapisv1.OperationTypeFailover,
		},
	}

	h := newMockHandler(inst, op)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-a/operations/failover-inst-b-1?namespace=disaster-system")
	ctx.Params = param.Params{
		{Key: "name", Value: "inst-a"},
		{Key: "operationName", Value: "failover-inst-b-1"},
	}

	h.getOperationDetail(context.Background(), ctx)

	assert.Equal(t, consts.StatusNotFound, ctx.Response.StatusCode())
}

func getSkipScaleDownSourceCompat(spec interface{}) (bool, bool) {
	v := reflect.ValueOf(spec)
	if !v.IsValid() {
		return false, false
	}
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return false, false
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return false, false
	}
	f := v.FieldByName("SkipScaleDownSource")
	if !f.IsValid() || f.Kind() != reflect.Bool {
		return false, false
	}
	return f.Bool(), true
}

func TestCreateInstance_RestoreResourceSelection_ClearsIncludeClusterResourcesWhenScopedFiltersPresent(t *testing.T) {
	ns := "disaster-system"
	cfg := &dapisv1.DisasterConfig{ObjectMeta: metav1.ObjectMeta{Name: "cfg-priority-create"}}
	h := newMockHandler(cfg)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances")
	ctx.Request.SetBody([]byte(`{
		"name":"inst-priority-create",
		"namespace":"disaster-system",
		"config":"cfg-priority-create",
		"restorePolicy":{
			"resourceSelection":{
				"includeClusterResources":false,
				"includedNamespaceScopedResources":["services"],
				"includedClusterScopedResources":["clusterroles"]
			}
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())

	created, err := h.DisasterClient.DisasterV1().DisasterInstances(ns).Get(context.Background(), "inst-priority-create", metav1.GetOptions{})
	assert.NoError(t, err)
	if assert.NotNil(t, created.Spec.RestorePolicy) && assert.NotNil(t, created.Spec.RestorePolicy.ResourceSelection) {
		assert.Equal(t, []string{"services"}, created.Spec.RestorePolicy.ResourceSelection.IncludedNamespaceScopedResources)
		assert.Equal(t, []string{"clusterroles"}, created.Spec.RestorePolicy.ResourceSelection.IncludedClusterScopedResources)
		assert.Nil(t, created.Spec.RestorePolicy.ResourceSelection.IncludeClusterResources)
	}
}

func TestCreateInstance_RestoreResourceSelection_RejectsScopedConflict(t *testing.T) {
	cfg := &dapisv1.DisasterConfig{ObjectMeta: metav1.ObjectMeta{Name: "cfg-priority-reject-create"}}
	h := newMockHandler(cfg)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances")
	ctx.Request.SetBody([]byte(`{
		"name":"inst-priority-reject-create",
		"namespace":"disaster-system",
		"config":"cfg-priority-reject-create",
		"restorePolicy":{
			"resourceSelection":{
				"includedNamespaceScopedResources":["services"],
				"excludedNamespaceScopedResources":["services"]
			}
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	assert.Contains(t, string(ctx.Response.Body()), "ResourceSelectionInvalid")
}

func TestUpdateInstance_RestoreResourceSelection_ClearsIncludeClusterResourcesWhenScopedFiltersPresent(t *testing.T) {
	ns := "disaster-system"
	includeClusterResources := true
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{Name: "inst-priority-update", Namespace: ns, UID: types.UID("uid-priority-update")},
		Spec: dapisv1.DisasterInstanceSpec{
			Config: "cfg-priority-update",
			RestorePolicy: &dapisv1.RestorePolicy{
				ResourceSelection: &dapisv1.RestoreResourceSelectionPolicy{
					IncludeClusterResources: &includeClusterResources,
				},
			},
		},
	}
	cfg := &dapisv1.DisasterConfig{ObjectMeta: metav1.ObjectMeta{Name: "cfg-priority-update"}}
	h := newMockHandler(inst, cfg)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-priority-update?namespace=disaster-system")
	ctx.Params = param.Params{{Key: "name", Value: "inst-priority-update"}}
	ctx.Request.SetBody([]byte(`{
		"restorePolicy":{
			"resourceSelection":{
				"includeClusterResources":false,
				"includedClusterScopedResources":["nodes"]
			}
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.updateInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	updated, err := h.DisasterClient.DisasterV1().DisasterInstances(ns).Get(context.Background(), "inst-priority-update", metav1.GetOptions{})
	assert.NoError(t, err)
	if assert.NotNil(t, updated.Spec.RestorePolicy) && assert.NotNil(t, updated.Spec.RestorePolicy.ResourceSelection) {
		assert.Equal(t, []string{"nodes"}, updated.Spec.RestorePolicy.ResourceSelection.IncludedClusterScopedResources)
		assert.Nil(t, updated.Spec.RestorePolicy.ResourceSelection.IncludeClusterResources)
	}
}

func TestUpdateInstance_RestoreResourceSelection_RejectsScopedConflict(t *testing.T) {
	ns := "disaster-system"
	inst := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{Name: "inst-priority-update-reject", Namespace: ns, UID: types.UID("uid-priority-update-reject")},
		Spec:       dapisv1.DisasterInstanceSpec{Config: "cfg-priority-update-reject"},
	}
	cfg := &dapisv1.DisasterConfig{ObjectMeta: metav1.ObjectMeta{Name: "cfg-priority-update-reject"}}
	h := newMockHandler(inst, cfg)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/instances/inst-priority-update-reject?namespace=disaster-system")
	ctx.Params = param.Params{{Key: "name", Value: "inst-priority-update-reject"}}
	ctx.Request.SetBody([]byte(`{
		"restorePolicy":{
			"resourceSelection":{
				"includedClusterScopedResources":["nodes"],
				"excludedClusterScopedResources":["nodes"]
			}
		}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.updateInstance(context.Background(), ctx)
	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	assert.Contains(t, string(ctx.Response.Body()), "ResourceSelectionInvalid")
}
