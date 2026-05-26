package drill

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route/param"
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-operator/pkg/clientset/versioned/fake"
	informers "github.com/softcdata/testudo-operator/pkg/informers/externalversions"
	instancev1 "github.com/softcdata/testudo-server/internal/apis/disaster_instance/v1"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/kube"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
)

const testNamespace = "disaster-system"

func newMockHandler(objects ...runtime.Object) *DrillHandler {
	fakeClient := fake.NewSimpleClientset(objects...)
	informerFactory := informers.NewSharedInformerFactory(fakeClient, 0)

	kc := &kube.KubeClient{
		DisasterClient:  fakeClient,
		InformerFactory: informerFactory,
	}

	h := server.Default()
	rg := h.Group("/v1")

	// 匹配 NewDrillHandler 签名: func(kc, rg, mw...)
	return NewDrillHandler(kc, rg)
}

func boolPtr(v bool) *bool {
	return &v
}

// TestListDrills_Empty 测试空列表
func TestListDrills_Empty(t *testing.T) {
	h := newMockHandler()

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/drills")

	h.listDrills(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
}

// TestListDrills_WithData 测试有数据的列表
func TestListDrills_WithData(t *testing.T) {
	drill1 := &dapisv1.DisasterDrill{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "drill-001",
			Namespace: testNamespace,
		},
		Spec: dapisv1.DisasterDrillSpec{
			InstanceName: "my-app-dr",
		},
		Status: dapisv1.DisasterDrillStatus{
			State: dapisv1.DrillStateReady,
		},
	}
	drill2 := &dapisv1.DisasterDrill{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "drill-002",
			Namespace: testNamespace,
		},
		Spec: dapisv1.DisasterDrillSpec{
			InstanceName: "other-app-dr",
		},
		Status: dapisv1.DisasterDrillStatus{
			State: dapisv1.DrillStatePending,
		},
	}
	h := newMockHandler(drill1, drill2)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/drills")

	h.listDrills(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp map[string]interface{}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)

	dataMap := resp["data"].(map[string]interface{})
	items := dataMap["items"].([]interface{})
	assert.Len(t, items, 2)
	for _, it := range items {
		drill := it.(map[string]interface{})
		status, ok := drill["status"].(map[string]interface{})
		assert.True(t, ok)
		assert.Contains(t, status, "state")
		_, hasTopLevelState := drill["state"]
		assert.False(t, hasTopLevelState)
	}
}

// TestListDrills_FilterByInstanceName 测试按 instanceName 过滤
func TestListDrills_FilterByInstanceName(t *testing.T) {
	drill1 := &dapisv1.DisasterDrill{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "drill-001",
			Namespace: testNamespace,
		},
		Spec: dapisv1.DisasterDrillSpec{
			InstanceName: "my-app-dr",
		},
	}
	drill2 := &dapisv1.DisasterDrill{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "drill-002",
			Namespace: testNamespace,
		},
		Spec: dapisv1.DisasterDrillSpec{
			InstanceName: "other-app-dr",
		},
	}
	h := newMockHandler(drill1, drill2)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/drills?instanceName=my-app-dr")

	h.listDrills(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp map[string]interface{}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)

	dataMap := resp["data"].(map[string]interface{})
	items := dataMap["items"].([]interface{})
	assert.Len(t, items, 1) // 只返回 drill-001
}

// TestListDrills_FilterByGroupName 测试按 groupName 过滤
func TestListDrills_FilterByGroupName(t *testing.T) {
	drill1 := &dapisv1.DisasterDrill{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "drill-group-1",
			Namespace: testNamespace,
		},
		Spec: dapisv1.DisasterDrillSpec{
			GroupName: "my-group",
		},
	}
	drill2 := &dapisv1.DisasterDrill{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "drill-group-2",
			Namespace: testNamespace,
		},
		Spec: dapisv1.DisasterDrillSpec{
			GroupName: "other-group",
		},
	}
	h := newMockHandler(drill1, drill2)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/drills?groupName=my-group")

	h.listDrills(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp map[string]interface{}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)

	dataMap := resp["data"].(map[string]interface{})
	items := dataMap["items"].([]interface{})
	assert.Len(t, items, 1)

	firstDrill := items[0].(map[string]interface{})
	assert.Equal(t, "drill-group-1", firstDrill["name"])
}

func TestGetProtectedNamespaces_ByInstance(t *testing.T) {
	instance := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-app-dr",
			Namespace: testNamespace,
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Namespaces: []string{"app-a", " app-b ", "app-a"},
		},
	}
	h := newMockHandler(instance)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/drills/actions/protected-namespaces?instanceName=my-app-dr")

	h.getProtectedNamespaces(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp map[string]interface{}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)

	dataMap := resp["data"].(map[string]interface{})
	assert.Equal(t, "Instance", dataMap["type"])
	assert.Equal(t, "my-app-dr", dataMap["instanceName"])

	namespaces := dataMap["namespaces"].([]interface{})
	assert.Len(t, namespaces, 2)
	assert.Equal(t, "app-a", namespaces[0])
	assert.Equal(t, "app-b", namespaces[1])
}

func TestGetProtectedNamespaces_ByGroup(t *testing.T) {
	group := &dapisv1.DisasterGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-group",
			Namespace: testNamespace,
		},
		Spec: dapisv1.DisasterGroupSpec{
			Levels: [][]string{
				{"inst-a", "inst-b"},
				{"inst-a", "missing-inst"},
			},
		},
	}
	instA := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-a",
			Namespace: testNamespace,
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Namespaces: []string{"ns1", "common"},
		},
	}
	instB := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-b",
			Namespace: testNamespace,
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Namespaces: []string{"ns2", "common"},
		},
	}
	h := newMockHandler(group, instA, instB)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/drills/actions/protected-namespaces?groupName=my-group")

	h.getProtectedNamespaces(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp map[string]interface{}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)

	dataMap := resp["data"].(map[string]interface{})
	assert.Equal(t, "Group", dataMap["type"])
	assert.Equal(t, "my-group", dataMap["groupName"])

	namespaces := dataMap["namespaces"].([]interface{})
	assert.Len(t, namespaces, 3)
	assert.Equal(t, "common", namespaces[0])
	assert.Equal(t, "ns1", namespaces[1])
	assert.Equal(t, "ns2", namespaces[2])

	missingInstances := dataMap["missingInstances"].([]interface{})
	assert.Len(t, missingInstances, 1)
	assert.Equal(t, "missing-inst", missingInstances[0])
}

func TestGetProtectedNamespaces_DefaultNamespaceDoesNotFallbackForInstance(t *testing.T) {
	oldNamespace := common.DisasterSystemNamespace
	common.SetDisasterSystemNamespace(testNamespace)
	defer common.SetDisasterSystemNamespace(oldNamespace)

	instance := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "same-name",
			Namespace: "other-system",
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Namespaces: []string{"app-other"},
		},
	}
	h := newMockHandler(instance)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/drills/actions/protected-namespaces?instanceName=same-name")

	h.getProtectedNamespaces(context.Background(), ctx)

	assert.Equal(t, consts.StatusNotFound, ctx.Response.StatusCode())
}

func TestGetProtectedNamespaces_DefaultNamespaceDoesNotFallbackForGroup(t *testing.T) {
	oldNamespace := common.DisasterSystemNamespace
	common.SetDisasterSystemNamespace(testNamespace)
	defer common.SetDisasterSystemNamespace(oldNamespace)

	group := &dapisv1.DisasterGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "same-group",
			Namespace: "other-system",
		},
	}
	h := newMockHandler(group)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/drills/actions/protected-namespaces?groupName=same-group")

	h.getProtectedNamespaces(context.Background(), ctx)

	assert.Equal(t, consts.StatusNotFound, ctx.Response.StatusCode())
}

func TestGetProtectedNamespaces_InvalidParams(t *testing.T) {
	h := newMockHandler()

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/drills/actions/protected-namespaces?instanceName=i1&groupName=g1")

	h.getProtectedNamespaces(context.Background(), ctx)

	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
}

func TestGetProtectedNamespaces_MissingParams(t *testing.T) {
	h := newMockHandler()

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/drills/actions/protected-namespaces")

	h.getProtectedNamespaces(context.Background(), ctx)

	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
}

// TestCreateDrill_Instance_Success 测试创建实例演练
func TestCreateDrill_Instance_Success(t *testing.T) {
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
		InstanceName:  "my-app-dr",
		Name:          "test-drill-inst",
		TargetCluster: "cluster-b",
		Namespace:     testNamespace,
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createDrill(context.Background(), ctx)

	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())

	// 验证 CR 已创建
	created, err := h.DisasterClient.DisasterV1().DisasterDrills(testNamespace).Get(
		context.Background(), "test-drill-inst", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "my-app-dr", created.Spec.InstanceName)
	assert.Equal(t, "", created.Spec.GroupName)
	// 验证 labels
	assert.Equal(t, "my-app-dr", created.Labels["testudo.softcdata.com/instance"])
}

// TestCreateDrill_Group_Success 测试创建容灾组演练
func TestCreateDrill_Group_Success(t *testing.T) {
	group := &dapisv1.DisasterGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-group",
			Namespace: testNamespace,
		},
	}
	h := newMockHandler(group)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/drills")

	req := CreateDrillRequest{
		GroupName:     "my-group",
		Name:          "test-drill-group",
		TargetCluster: "cluster-b",
		Namespace:     testNamespace,
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createDrill(context.Background(), ctx)

	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())

	// 验证 CR 已创建
	created, err := h.DisasterClient.DisasterV1().DisasterDrills(testNamespace).Get(
		context.Background(), "test-drill-group", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "my-group", created.Spec.GroupName)
	assert.Equal(t, "", created.Spec.InstanceName)
	// 验证 labels
	assert.Equal(t, "my-group", created.Labels["testudo.softcdata.com/group"])
}

func TestCreateDrill_Instance_RestorePolicyTextSuccess(t *testing.T) {
	instance := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-app-dr",
			Namespace: testNamespace,
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Namespaces: []string{"demo-ns"},
		},
	}
	h := newMockHandler(instance)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/drills")

	req := CreateDrillRequest{
		InstanceName: "my-app-dr",
		Name:         "test-drill-restore-policy",
		RestorePolicy: &instancev1.RestorePolicyRequest{
			UseUnifiedDirectionResolver: boolPtr(true),
			ModifierRulesText:           "[{\"id\":\"drill-rule\",\"mode\":\"reversible\",\"applyTo\":[\"drill\"],\"conditions\":{\"groupResource\":\"deployments.apps\",\"namespaces\":[\"demo-ns\"]},\"pair\":{\"path\":\"/metadata/annotations/drill-test\",\"sourceValue\":\"src\",\"targetValue\":\"dst\"}}]",
		},
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createDrill(context.Background(), ctx)

	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())

	created, err := h.DisasterClient.DisasterV1().DisasterDrills(testNamespace).Get(
		context.Background(), "test-drill-restore-policy", metav1.GetOptions{},
	)
	assert.NoError(t, err)
	if assert.NotNil(t, created.Spec.RestorePolicy) {
		assert.Len(t, created.Spec.RestorePolicy.ModifierRules, 1)
		assert.True(t, created.Spec.RestorePolicy.UseUnifiedDirectionResolver != nil && *created.Spec.RestorePolicy.UseUnifiedDirectionResolver)
	}

	var resp map[string]any
	err = json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	data := resp["data"].(map[string]any)
	restorePolicy := data["restorePolicy"].(map[string]any)
	assert.NotEmpty(t, restorePolicy["modifierRulesText"])
}

func TestCreateDrill_Instance_BulkModifierActionsTextSuccess(t *testing.T) {
	instance := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-app-dr",
			Namespace: testNamespace,
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Namespaces: []string{"demo-ns"},
		},
	}
	h := newMockHandler(instance)
	h.BuildBulkModifierSnapshotFunc = func(
		_ context.Context,
		spec *dapisv1.DisasterInstanceSpec,
		_ *rest.Config,
	) (*instancev1.BulkModifierSnapshotBuildResult, error) {
		return &instancev1.BulkModifierSnapshotBuildResult{
			Actions: spec.RestorePolicy.BulkModifierActions,
			ModifierRuleSnapshot: []dapisv1.RestoreModifierRule{{
				ID:   "bulk-generated",
				Mode: dapisv1.RestoreModifierModeVeleroNative,
				ApplyTo: []dapisv1.RestoreModifierApplyTarget{
					dapisv1.RestoreModifierApplyDrill,
				},
				Conditions: dapisv1.Conditions{
					GroupResource: "deployments.apps",
					Namespaces:    []string{"demo-ns"},
				},
				VeleroRule: &dapisv1.RestoreModifierVeleroRule{
					Patches: []dapisv1.JSONPatch{{
						Operation: "add",
						Path:      "/metadata/annotations/from-bulk",
						Value:     "true",
					}},
				},
			}},
			ModifierRuleSnapshotHash: "sha256:drill-bulk",
		}, nil
	}

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/drills")

	req := CreateDrillRequest{
		InstanceName: "my-app-dr",
		Name:         "test-drill-bulk-policy",
		RestorePolicy: &instancev1.RestorePolicyRequest{
			UseUnifiedDirectionResolver: boolPtr(true),
			BulkModifierActionsText:     "{\"id\":\"replace-ip\",\"action\":\"replaceExactValue\",\"sourceValue\":\"10.10.0.12\",\"targetValue\":\"10.20.0.12\"}",
		},
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createDrill(context.Background(), ctx)

	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())

	created, err := h.DisasterClient.DisasterV1().DisasterDrills(testNamespace).Get(
		context.Background(), "test-drill-bulk-policy", metav1.GetOptions{},
	)
	assert.NoError(t, err)
	if assert.NotNil(t, created.Spec.RestorePolicy) {
		assert.Len(t, created.Spec.RestorePolicy.BulkModifierActions, 1)
		assert.Len(t, created.Spec.RestorePolicy.ModifierRuleSnapshot, 1)
		assert.Equal(t, "sha256:drill-bulk", created.Spec.RestorePolicy.ModifierRuleSnapshotHash)
	}
}

func TestCreateDrill_RestorePolicyConflict(t *testing.T) {
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
		Name:         "test-drill-conflict",
		RestorePolicy: &instancev1.RestorePolicyRequest{
			UseUnifiedDirectionResolver: boolPtr(true),
			ModifierRules: []dapisv1.RestoreModifierRule{{
				ID:   "rule-a",
				Mode: dapisv1.RestoreModifierModeReversible,
				Pair: &dapisv1.RestoreModifierPair{
					Path:        "/metadata/annotations/a",
					SourceValue: "x",
					TargetValue: "y",
				},
			}},
			ModifierRulesText: "[{\"id\":\"rule-b\",\"mode\":\"reversible\",\"pair\":{\"path\":\"/metadata/annotations/a\",\"sourceValue\":\"x\",\"targetValue\":\"z\"}}]",
		},
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createDrill(context.Background(), ctx)

	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
}

func TestCreateDrill_GroupRestorePolicyRejectsMixedSourceClusters(t *testing.T) {
	group := &dapisv1.DisasterGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-group",
			Namespace: testNamespace,
		},
		Spec: dapisv1.DisasterGroupSpec{
			Levels: [][]string{{"inst-a", "inst-b"}},
		},
	}
	instA := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-a",
			Namespace: testNamespace,
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Config:     "cfg-a",
			Namespaces: []string{"ns-a"},
		},
	}
	instB := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-b",
			Namespace: testNamespace,
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Config:     "cfg-b",
			Namespaces: []string{"ns-b"},
		},
	}
	cfgA := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg-a"},
		Spec:       dapisv1.DisasterConfigSpec{SourceCluster: "cluster-a"},
	}
	cfgB := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg-b"},
		Spec:       dapisv1.DisasterConfigSpec{SourceCluster: "cluster-b"},
	}
	h := newMockHandler(group, instA, instB, cfgA, cfgB)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/drills")

	req := CreateDrillRequest{
		GroupName: "my-group",
		Name:      "test-drill-group-policy-mixed-source",
		RestorePolicy: &instancev1.RestorePolicyRequest{
			UseUnifiedDirectionResolver: boolPtr(true),
			ModifierRulesText:           "[{\"id\":\"drill-rule\",\"mode\":\"reversible\",\"applyTo\":[\"drill\"],\"conditions\":{\"groupResource\":\"deployments.apps\",\"namespaces\":[\"ns-a\"]},\"pair\":{\"path\":\"/metadata/annotations/drill-only\",\"sourceValue\":\"from-a\",\"targetValue\":\"to-b\"}}]",
		},
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createDrill(context.Background(), ctx)

	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
}

// TestCreateDrill_MissingAll 测试缺少必填字段
func TestCreateDrill_MissingAll(t *testing.T) {
	h := newMockHandler()

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/drills")

	req := CreateDrillRequest{
		Name: "test-drill",
		// 缺少 InstanceName 和 GroupName
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createDrill(context.Background(), ctx)

	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
}

// TestCreateDrill_Mutex 测试互斥字段同时存在
func TestCreateDrill_Mutex(t *testing.T) {
	h := newMockHandler()

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/drills")

	req := CreateDrillRequest{
		InstanceName: "inst1",
		GroupName:    "grp1",
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createDrill(context.Background(), ctx)

	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
}

// TestCreateDrill_GroupNotFound 测试容灾组不存在
func TestCreateDrill_GroupNotFound(t *testing.T) {
	h := newMockHandler()

	ctx := app.NewContext(16)
	req := CreateDrillRequest{
		GroupName: "not-exist-group",
		Namespace: testNamespace,
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)

	h.createDrill(context.Background(), ctx)

	assert.Equal(t, consts.StatusNotFound, ctx.Response.StatusCode())
}

// TestCleanupDrill_Success 测试成功清理演练资源
func TestCleanupDrill_Success(t *testing.T) {
	drill := &dapisv1.DisasterDrill{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "drill-to-clean",
			Namespace: testNamespace,
		},
		Spec: dapisv1.DisasterDrillSpec{
			InstanceName: "test-inst",
		},
		Status: dapisv1.DisasterDrillStatus{
			State: dapisv1.DrillStateCompleted,
		},
	}
	h := newMockHandler(drill)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/drills/drill-to-clean/cleanup")
	ctx.Params = param.Params{param.Param{Key: "name", Value: "drill-to-clean"}}

	h.cleanupDrill(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	updated, err := h.DisasterClient.DisasterV1().DisasterDrills(testNamespace).Get(context.Background(), "drill-to-clean", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.True(t, updated.Spec.CleanUp)
}

// TestCleanupDrill_InvalidState 测试状态不符时的清理操作
func TestCleanupDrill_InvalidState(t *testing.T) {
	drill := &dapisv1.DisasterDrill{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "drill-invalid-state",
			Namespace: testNamespace,
		},
		Status: dapisv1.DisasterDrillStatus{
			State: dapisv1.DrillStateReady, // Not Completed
		},
	}
	h := newMockHandler(drill)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/drills/drill-invalid-state/cleanup")
	ctx.Params = param.Params{param.Param{Key: "name", Value: "drill-invalid-state"}}

	h.cleanupDrill(context.Background(), ctx)

	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
}

// TestCleanupDrill_AlreadyTriggered 测试重复触发清理
func TestCleanupDrill_AlreadyTriggered(t *testing.T) {
	drill := &dapisv1.DisasterDrill{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "drill-already-clean",
			Namespace: testNamespace,
		},
		Spec: dapisv1.DisasterDrillSpec{
			CleanUp: true,
		},
		Status: dapisv1.DisasterDrillStatus{
			State: dapisv1.DrillStateCompleted,
		},
	}
	h := newMockHandler(drill)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/drills/drill-already-clean/cleanup")
	ctx.Params = param.Params{param.Param{Key: "name", Value: "drill-already-clean"}}

	h.cleanupDrill(context.Background(), ctx)

	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
}
