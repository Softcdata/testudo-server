package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route/param"
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-operator/pkg/clientset/versioned/fake"
	informers "github.com/softcdata/testudo-operator/pkg/informers/externalversions"
	instanceapi "github.com/softcdata/testudo-server/internal/apis/disaster_instance/v1"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/kube"
	"github.com/softcdata/testudo-server/internal/transport"
	"github.com/stretchr/testify/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// inst 快速构造 InstanceSummaryDTO（只关心 FsmState）
func inst(fsmState string) InstanceSummaryDTO {
	return InstanceSummaryDTO{Name: "inst-" + fsmState, FsmState: fsmState}
}

func TestComputeGroupFsmState(t *testing.T) {
	tests := []struct {
		name        string
		instances   []InstanceSummaryDTO
		wantState   string
		wantOpsHas  []string // availableOperations 至少包含这些
		wantOpsNone bool     // availableOperations 应为空
	}{
		// Unknown
		{
			name:        "空组 → Unknown",
			instances:   []InstanceSummaryDTO{},
			wantState:   "Unknown",
			wantOpsNone: true,
		},
		// Initializing
		{
			name:        "有 Pending 实例 → Initializing",
			instances:   []InstanceSummaryDTO{inst("Pending"), inst("Protected")},
			wantState:   "Initializing",
			wantOpsNone: true,
		},
		{
			name:        "有 Initializing 实例 → Initializing",
			instances:   []InstanceSummaryDTO{inst("Initializing"), inst("Protected")},
			wantState:   "Initializing",
			wantOpsNone: true,
		},
		// Protected
		{
			name:       "全部 Protected → Protected，含完整操作集",
			instances:  []InstanceSummaryDTO{inst("Protected"), inst("Protected"), inst("Protected")},
			wantState:  "Protected",
			wantOpsHas: []string{"failover", "pause", "synconce", "syncdata", "syncresource"},
		},
		// PartialProtected
		{
			name:       "混合 Protected+Paused → PartialProtected",
			instances:  []InstanceSummaryDTO{inst("Protected"), inst("Paused")},
			wantState:  "PartialProtected",
			wantOpsHas: []string{"failover", "pause", "synconce"},
		},
		// Paused
		{
			name:       "全部 Paused → Paused",
			instances:  []InstanceSummaryDTO{inst("Paused"), inst("Paused")},
			wantState:  "Paused",
			wantOpsHas: []string{"resume"},
		},
		// Active
		{
			name:       "全部 Active → Active",
			instances:  []InstanceSummaryDTO{inst("Active"), inst("Active")},
			wantState:  "Active",
			wantOpsHas: []string{"reprotect"},
		},
		// FailingOver — 最高优先级，即使有其他状态也覆盖
		{
			name:        "有 FailingOver 实例（含 Protected） → FailingOver，操作为空",
			instances:   []InstanceSummaryDTO{inst("FailingOver"), inst("Protected"), inst("Failed")},
			wantState:   "FailingOver",
			wantOpsNone: true,
		},
		// FailingBack
		{
			name:        "有 FailingBack 实例 → FailingBack，操作为空",
			instances:   []InstanceSummaryDTO{inst("FailingBack"), inst("Active")},
			wantState:   "FailingBack",
			wantOpsNone: true,
		},
		// Degraded — Failed 优先于全量状态，但不如 FailingOver
		{
			name:        "有 Failed 实例（无操作进行中） → Degraded，操作为空",
			instances:   []InstanceSummaryDTO{inst("Failed"), inst("Protected")},
			wantState:   "Degraded",
			wantOpsNone: true,
		},
		{
			name:        "有 ConfigError 实例（无操作进行中） → Degraded，操作为空",
			instances:   []InstanceSummaryDTO{inst("ConfigError"), inst("Protected")},
			wantState:   "Degraded",
			wantOpsNone: true,
		},
		// FailingOver 优先于 Failed
		{
			name:        "同时有 FailingOver 和 Failed → FailingOver（FailingOver 优先）",
			instances:   []InstanceSummaryDTO{inst("FailingOver"), inst("Failed")},
			wantState:   "FailingOver",
			wantOpsNone: true,
		},
		{
			name:        "同时有 FailingOver 和 ConfigError → FailingOver（FailingOver 优先）",
			instances:   []InstanceSummaryDTO{inst("FailingOver"), inst("ConfigError")},
			wantState:   "FailingOver",
			wantOpsNone: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotState, gotOps := computeGroupFsmState(tc.instances)
			assert.Equal(t, tc.wantState, gotState)
			if tc.wantOpsNone {
				assert.Empty(t, gotOps)
			}
			for _, op := range tc.wantOpsHas {
				assert.Contains(t, gotOps, op)
			}
		})
	}
}

func newMockGroupHandler(objects ...runtime.Object) *GroupHandler {
	fakeClient := fake.NewSimpleClientset(objects...)
	informerFactory := informers.NewSharedInformerFactory(fakeClient, 0)

	kc := &kube.KubeClient{
		DisasterClient:  fakeClient,
		InformerFactory: informerFactory,
	}

	h := server.Default()
	rg := h.Group("/v1")
	return NewGroupHandler(kc, rg)
}

func TestListGroups_KeywordByNameAndInstanceSummary(t *testing.T) {
	group := &dapisv1.DisasterGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "prod-group",
			Namespace:         "disaster-system",
			UID:               types.UID("uid-prod-group"),
			CreationTimestamp: metav1.NewTime(time.Date(2026, 4, 15, 11, 0, 0, 0, time.UTC)),
		},
		Spec: dapisv1.DisasterGroupSpec{
			Levels: [][]string{{"inst-a"}, {"inst-missing"}},
		},
	}
	otherGroup := &dapisv1.DisasterGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "dev-group",
			Namespace:         "disaster-system",
			UID:               types.UID("uid-dev-group"),
			CreationTimestamp: metav1.NewTime(time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC)),
		},
		Spec: dapisv1.DisasterGroupSpec{
			Levels: [][]string{{"inst-b"}},
		},
	}
	instA := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-a",
			Namespace: "disaster-system",
		},
		Status: dapisv1.DisasterInstanceStatus{
			FsmState: dapisv1.FsmStateProtected,
		},
	}
	instB := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-b",
			Namespace: "disaster-system",
		},
		Status: dapisv1.DisasterInstanceStatus{
			FsmState: dapisv1.FsmStateProtected,
		},
	}

	h := newMockGroupHandler(group, otherGroup, instA, instB)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/groups?keyword=prod&page=1&limit=-1")
	ctx.Request.URI().SetQueryString("keyword=prod&page=1&limit=-1")

	h.listGroups(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Items []DisasterGroupDTO `json:"items"`
		} `json:"data"`
		Meta struct {
			Summary map[string]int `json:"summary"`
		} `json:"meta"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	assert.Len(t, resp.Data.Items, 1)
	assert.Equal(t, "prod-group", resp.Data.Items[0].Name)
	assert.Equal(t, 2, resp.Meta.Summary["instanceCount"])
	assert.Equal(t, 1, resp.Meta.Summary["abnormalCount"])
}

func TestSummarizeDisasterGroupList_AbnormalCount(t *testing.T) {
	summary := summarizeDisasterGroupList([]DisasterGroupDTO{
		{
			Name: "healthy",
			Status: DisasterGroupStatusDTO{
				FsmState: "Protected",
			},
			Instances: []InstanceSummaryDTO{{Name: "inst-a", FsmState: dapisv1.FsmStateProtected}},
		},
		{
			Name: "failed-member",
			Status: DisasterGroupStatusDTO{
				FsmState: "Degraded",
			},
			Instances: []InstanceSummaryDTO{{Name: "inst-b", FsmState: dapisv1.FsmStateFailed}},
		},
		{
			Name: "missing-member",
			Status: DisasterGroupStatusDTO{
				FsmState: "PartialProtected",
			},
			Instances: []InstanceSummaryDTO{{Name: "inst-c", FsmState: "NotFound"}},
		},
		{
			Name: "running-failover",
			Status: DisasterGroupStatusDTO{
				FsmState: "FailingOver",
			},
			Instances: []InstanceSummaryDTO{{Name: "inst-d", FsmState: dapisv1.FsmStateFailingOver}},
		},
		{
			Name: "reason-only",
			Status: DisasterGroupStatusDTO{
				FsmState: "Protected",
				Reason:   "InstanceFailed",
			},
			Instances: []InstanceSummaryDTO{{Name: "inst-e", FsmState: dapisv1.FsmStateProtected}},
		},
		{
			Name: "condition-error",
			Status: DisasterGroupStatusDTO{
				FsmState: "Protected",
				Conditions: []common.LocalCondition{
					{Type: "Error", Status: metav1.ConditionTrue, Reason: "InstanceFailed"},
				},
			},
			Instances: []InstanceSummaryDTO{{Name: "inst-f", FsmState: dapisv1.FsmStateProtected}},
		},
	})

	assert.Equal(t, 6, summary["instanceCount"])
	assert.Equal(t, 4, summary["abnormalCount"])
}

func TestMatchStatusErrorIncludesMissingInstance(t *testing.T) {
	h := &GroupHandler{}
	assert.True(t, h.matchStatus([]InstanceSummaryDTO{{Name: "inst-missing", FsmState: "NotFound"}}, "error"))
}

func TestBuildGroupDTOUsesConsistentAggregation(t *testing.T) {
	group := &dapisv1.DisasterGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "g-status",
			Namespace: "disaster-system",
		},
		Spec: dapisv1.DisasterGroupSpec{
			Levels: [][]string{{"inst-a"}},
		},
		Status: dapisv1.DisasterGroupStatus{
			TotalInstances: 1,
			ReadyInstances: 1,
		},
	}

	instance := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-a",
			Namespace: "disaster-system",
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Config:     "cfg-a",
			Namespaces: []string{"app-ns"},
		},
		Status: dapisv1.DisasterInstanceStatus{
			FsmState:         dapisv1.FsmStateProtected,
			PrimaryCluster:   "cluster-a",
			SecondaryCluster: "cluster-b",
		},
	}

	config := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cfg-a",
		},
		Spec: dapisv1.DisasterConfigSpec{
			StorageRepository: "repo-a",
		},
	}

	h := newMockGroupHandler(instance, config)
	liveDTO := h.buildGroupDTO(context.Background(), group, nil, nil)

	instanceCache := map[string]*dapisv1.DisasterInstance{
		instance.Name: instance,
	}
	configCache := map[string]*dapisv1.DisasterConfig{
		config.Name: config,
	}
	cachedDTO := h.buildGroupDTO(context.Background(), group, instanceCache, configCache)

	assert.Equal(t, "g-status", liveDTO.Name)
	assert.Equal(t, "1 Levels, 1 Instances", liveDTO.Status.Summary)
	assert.Equal(t, "Protected", liveDTO.Status.FsmState)
	assert.Contains(t, liveDTO.Status.AvailableOperations, "failover")
	if assert.Len(t, liveDTO.Instances, 1) {
		assert.Equal(t, "inst-a", liveDTO.Instances[0].Name)
		assert.Equal(t, "repo-a", liveDTO.Instances[0].StorageRepository)
	}

	assert.Equal(t, liveDTO.Status.Summary, cachedDTO.Status.Summary)
	assert.Equal(t, liveDTO.Status.FsmState, cachedDTO.Status.FsmState)
	assert.Equal(t, liveDTO.Status.AvailableOperations, cachedDTO.Status.AvailableOperations)
	if assert.Len(t, cachedDTO.Instances, 1) {
		assert.Equal(t, "inst-a", cachedDTO.Instances[0].Name)
		assert.Equal(t, "repo-a", cachedDTO.Instances[0].StorageRepository)
	}
}

func TestGroupExecuteAction_FailoverPassesSkipScaleDownSource(t *testing.T) {
	group := &dapisv1.DisasterGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "g1",
			Namespace: "disaster-system",
			UID:       types.UID("group-uid"),
		},
		Spec: dapisv1.DisasterGroupSpec{
			Levels: [][]string{{"inst-a"}},
		},
	}

	h := newMockGroupHandler(group)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/groups/g1/actions")
	ctx.Params = param.Params{
		{Key: "name", Value: "g1"},
	}
	ctx.Request.SetBody([]byte(`{"operation":"failover","config":{"SkipScaleDownSource":true,"waitUntilReady":true}}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.executeAction(context.Background(), ctx)

	assert.Equal(t, consts.StatusAccepted, ctx.Response.StatusCode())

	ops, err := h.DisasterClient.DisasterV1().DisasterOperations("disaster-system").List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Len(t, ops.Items, 1)

	op := ops.Items[0]
	assert.Equal(t, dapisv1.OperationTypeFailover, op.Spec.OperationType)
	assert.Equal(t, "g1", op.Spec.GroupName)
	assert.Equal(t, "true", op.Annotations[skipScaleDownSourceAnnotation])
	if value, ok := getSkipScaleDownSourceCompat(op.Spec); ok {
		assert.True(t, value)
	}
	assert.True(t, op.Spec.WaitUntilReady)
	if assert.NotNil(t, op.Spec.SkipPodReadyCheck) {
		assert.False(t, *op.Spec.SkipPodReadyCheck)
	}
}

func TestGroupExecuteAction_SkipPodReadyCheckOverridesWaitUntilReady(t *testing.T) {
	group := &dapisv1.DisasterGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "g2",
			Namespace: "disaster-system",
			UID:       types.UID("group-uid-2"),
		},
		Spec: dapisv1.DisasterGroupSpec{
			Levels: [][]string{{"inst-a"}},
		},
	}

	h := newMockGroupHandler(group)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/groups/g2/actions")
	ctx.Params = param.Params{
		{Key: "name", Value: "g2"},
	}
	ctx.Request.SetBody([]byte(`{"operation":"failover","config":{"skipPodReadyCheck":true,"waitUntilReady":true}}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.executeAction(context.Background(), ctx)

	assert.Equal(t, consts.StatusAccepted, ctx.Response.StatusCode())

	ops, err := h.DisasterClient.DisasterV1().DisasterOperations("disaster-system").List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Len(t, ops.Items, 1)

	op := ops.Items[0]
	if assert.NotNil(t, op.Spec.SkipPodReadyCheck) {
		assert.True(t, *op.Spec.SkipPodReadyCheck)
	}
	assert.False(t, op.Spec.WaitUntilReady)
}

func TestGetGroupOperationDetail_ReturnsProjectedDetail(t *testing.T) {
	group := &dapisv1.DisasterGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "g-detail",
			Namespace: "disaster-system",
			UID:       types.UID("group-detail-uid"),
		},
	}
	op := &dapisv1.DisasterOperation{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "failover-g-detail-1",
			Namespace:         "disaster-system",
			UID:               types.UID("group-op-uid"),
			CreationTimestamp: metav1.NewTime(time.Date(2026, 4, 15, 11, 0, 0, 0, time.UTC)),
		},
		Spec: dapisv1.DisasterOperationSpec{
			GroupName:     "g-detail",
			OperationType: dapisv1.OperationTypeFailover,
		},
		Status: dapisv1.DisasterOperationStatus{
			State:       dapisv1.OperationStateRunning,
			CurrentStep: "ScaleUpTarget",
			Steps: []dapisv1.StepStatus{
				{Name: "PreCheck", State: "Completed"},
				{Name: "ScaleUpTarget", State: "Running"},
			},
			GroupStatus: &dapisv1.GroupStatus{
				TotalLevels:       2,
				CurrentLevelIndex: 1,
			},
		},
	}

	h := newMockGroupHandler(group, op)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/groups/g-detail/operations/failover-g-detail-1")
	ctx.Params = param.Params{
		{Key: "name", Value: "g-detail"},
		{Key: "operationName", Value: "failover-g-detail-1"},
	}

	h.getOperationDetail(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp struct {
		Code int                            `json:"code"`
		Data instanceapi.OperationDetailDTO `json:"data"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "DisasterGroup", resp.Data.OwnerKind)
	assert.Equal(t, "g-detail", resp.Data.OwnerName)
	if assert.NotNil(t, resp.Data.GroupStatus) {
		assert.Equal(t, 2, resp.Data.GroupStatus.TotalLevels)
		assert.Equal(t, 1, resp.Data.GroupStatus.CurrentLevelIndex)
	}
}

func TestGetGroupOperationDetail_ReturnsNotFoundWhenOwnerMismatch(t *testing.T) {
	group := &dapisv1.DisasterGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "g-a",
			Namespace: "disaster-system",
			UID:       types.UID("group-a-uid"),
		},
	}
	op := &dapisv1.DisasterOperation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "failover-g-b-1",
			Namespace: "disaster-system",
			UID:       types.UID("group-op-mismatch"),
		},
		Spec: dapisv1.DisasterOperationSpec{
			GroupName:     "g-b",
			OperationType: dapisv1.OperationTypeFailover,
		},
	}

	h := newMockGroupHandler(group, op)

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/groups/g-a/operations/failover-g-b-1")
	ctx.Params = param.Params{
		{Key: "name", Value: "g-a"},
		{Key: "operationName", Value: "failover-g-b-1"},
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

func TestDeriveGroupMemberStatus(t *testing.T) {
	tests := []struct {
		name        string
		fsmState    string
		reason      string
		message     string
		configName  string
		config      *dapisv1.DisasterConfig
		configErr   error
		wantState   string
		wantReason  string
		wantMessage string
	}{
		{
			name:       "正常实例且配置就绪保持原状态",
			fsmState:   "Protected",
			configName: "cfg-a",
			config: &dapisv1.DisasterConfig{
				Status: dapisv1.DisasterConfigStatus{Status: dapisv1.DisasterConfigStatusReady},
			},
			wantState:   "Protected",
			wantReason:  "",
			wantMessage: "",
		},
		{
			name:       "实例自身 reason 非空时呈现失败态",
			fsmState:   "Protected",
			reason:     "DataSyncFailed",
			message:    "periodic sync failed",
			configName: "cfg-a",
			config: &dapisv1.DisasterConfig{
				Status: dapisv1.DisasterConfigStatus{Status: dapisv1.DisasterConfigStatusReady},
			},
			wantState:   "Failed",
			wantReason:  "DataSyncFailed",
			wantMessage: "periodic sync failed",
		},
		{
			name:       "配置错误时即使实例 Protected 也应输出 ConfigError",
			fsmState:   "Protected",
			configName: "cfg-a",
			config: &dapisv1.DisasterConfig{
				Status: dapisv1.DisasterConfigStatus{
					Status:  dapisv1.DisasterConfigStatusError,
					Reason:  "SourceClusterNotFound",
					Message: "source cluster not found",
				},
			},
			wantState:   "ConfigError",
			wantReason:  "SourceClusterNotFound",
			wantMessage: "source cluster not found",
		},
		{
			name:       "配置缺失时给出 ConfigNotFound",
			fsmState:   "Protected",
			configName: "cfg-missing",
			configErr: apierrors.NewNotFound(
				schema.GroupResource{Group: "testudo.softcdata.com", Resource: "disasterconfigs"},
				"cfg-missing",
			),
			wantState:   "ConfigError",
			wantReason:  "ConfigNotFound",
			wantMessage: "DisasterConfig cfg-missing not found",
		},
		{
			name:       "实例不存在保持 NotFound",
			fsmState:   "NotFound",
			reason:     "InstanceNotFound",
			message:    "DisasterInstance inst-a not found",
			configName: "cfg-a",
			config: &dapisv1.DisasterConfig{
				Status: dapisv1.DisasterConfigStatus{Status: dapisv1.DisasterConfigStatusError},
			},
			wantState:   "NotFound",
			wantReason:  "InstanceNotFound",
			wantMessage: "DisasterInstance inst-a not found",
		},
		{
			name:       "配置错误但无详细信息时使用兜底文案",
			fsmState:   "Protected",
			configName: "cfg-a",
			config: &dapisv1.DisasterConfig{
				Status: dapisv1.DisasterConfigStatus{Status: dapisv1.DisasterConfigStatusNotReady},
			},
			wantState:   "ConfigError",
			wantReason:  "ConfigNotReady",
			wantMessage: "DisasterConfig cfg-a status=NotReady",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotState, gotReason, gotMessage := deriveGroupMemberStatus(
				tc.fsmState,
				tc.reason,
				tc.message,
				tc.configName,
				tc.config,
				tc.configErr,
			)

			assert.Equal(t, tc.wantState, gotState)
			assert.Equal(t, tc.wantReason, gotReason)
			assert.Equal(t, tc.wantMessage, gotMessage, fmt.Sprintf("message mismatch for case %s", tc.name))
		})
	}
}

func TestInstancePickerIncludesDerivedStatus(t *testing.T) {
	instance := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-a",
			Namespace: "disaster-system",
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Config:     "cfg-a",
			Namespaces: []string{"default"},
		},
		Status: dapisv1.DisasterInstanceStatus{
			FsmState: dapisv1.FsmStateProtected,
		},
	}

	config := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cfg-a",
		},
		Status: dapisv1.DisasterConfigStatus{
			Status:  dapisv1.DisasterConfigStatusError,
			Reason:  "SourceClusterNotFound",
			Message: "source cluster not found",
		},
	}

	h := newMockGroupHandler(instance, config)
	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/groups/instance-picker?page=1&limit=-1")

	h.instancePicker(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var env struct {
		Code int `json:"code"`
		Data struct {
			Items []InstancePickerItemDTO `json:"items"`
		} `json:"data"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &env)
	assert.NoError(t, err)
	assert.Equal(t, 0, env.Code)
	if assert.Len(t, env.Data.Items, 1) {
		item := env.Data.Items[0]
		assert.Equal(t, "inst-a", item.Name)
		assert.Equal(t, dapisv1.FsmStateProtected, item.FsmState)
		assert.Equal(t, "ConfigError", item.Status.State)
		assert.Equal(t, "SourceClusterNotFound", item.Status.Reason)
		assert.Equal(t, "source cluster not found", item.Status.Message)
	}
}

func TestInstancePickerSerializesEmptyReasonAndMessage(t *testing.T) {
	item := InstancePickerItemDTO{
		Name: "inst-a",
		Status: GroupMemberStatusDTO{
			State: "Protected",
		},
		FsmState: "Protected",
	}
	raw, err := json.Marshal(item)
	assert.NoError(t, err)

	var decoded map[string]any
	assert.NoError(t, json.Unmarshal(raw, &decoded))
	status, ok := decoded["status"].(map[string]any)
	assert.True(t, ok)
	_, hasReason := status["reason"]
	_, hasMessage := status["message"]
	assert.True(t, hasReason)
	assert.True(t, hasMessage)
}

func TestInstancePickerStatusFilterMatchesDisplayState(t *testing.T) {
	instance := &dapisv1.DisasterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst-a",
			Namespace: "disaster-system",
		},
		Spec: dapisv1.DisasterInstanceSpec{
			Config: "cfg-a",
		},
		Status: dapisv1.DisasterInstanceStatus{
			FsmState: dapisv1.FsmStateProtected,
		},
	}
	config := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg-a"},
		Status: dapisv1.DisasterConfigStatus{
			Status: dapisv1.DisasterConfigStatusError,
		},
	}

	h := newMockGroupHandler(instance, config)
	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/groups/instance-picker?status=ConfigError&page=1&limit=-1")
	ctx.Request.URI().SetQueryString("status=ConfigError&page=1&limit=-1")

	h.instancePicker(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var env struct {
		Data struct {
			Items []InstancePickerItemDTO `json:"items"`
		} `json:"data"`
		Meta utils.H `json:"meta"`
	}
	assert.NoError(t, json.Unmarshal(ctx.Response.Body(), &env))
	assert.Len(t, env.Data.Items, 1)
}

func TestMatchStatusErrorIncludesConfigError(t *testing.T) {
	h := &GroupHandler{}
	instances := []InstanceSummaryDTO{
		{Name: "inst-a", FsmState: "ConfigError"},
	}
	assert.True(t, h.matchStatus(instances, "error"))
}
