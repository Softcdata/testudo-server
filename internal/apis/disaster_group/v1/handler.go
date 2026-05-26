package v1

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	metadata "github.com/softcdata/testudo-operator/pkg/metadata"
	instanceapi "github.com/softcdata/testudo-server/internal/apis/disaster_instance/v1"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/kube"
	"github.com/softcdata/testudo-server/internal/transport"
	watchutils "github.com/softcdata/testudo-server/internal/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	matev1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/util/retry"
)

const skipScaleDownSourceAnnotation = "testudo.softcdata.com/skip-scale-down-source"

type GroupHandler struct {
	*kube.KubeClient
	Rg *route.RouterGroup
	Mw []app.HandlerFunc
}

func NewGroupHandler(kc *kube.KubeClient, rg *route.RouterGroup, mw ...app.HandlerFunc) *GroupHandler {
	return &GroupHandler{
		KubeClient: kc,
		Rg:         rg,
		Mw:         mw,
	}
}

// 1. Create DisasterGroup
func (h *GroupHandler) createGroup(c context.Context, ctx *app.RequestContext) {
	var req CreateDisasterGroupRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	annotations := make(map[string]string)
	if req.Description != "" {
		annotations["testudo.softcdata.com/description"] = req.Description
	}

	body := dapisv1.DisasterGroup{
		ObjectMeta: matev1.ObjectMeta{
			Name:        req.Name,
			Namespace:   common.DisasterSystemNamespace,
			Annotations: annotations,
		},
		Spec: req.ToCRD(),
	}

	transport.SetTraceAnnotation(&body.ObjectMeta, ctx, metadata.AnnotationTraceID)

	rc, err := h.DisasterClient.DisasterV1().DisasterGroups(common.DisasterSystemNamespace).Create(c, &body, matev1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			transport.WriteError(ctx, transport.CodeConflict, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	dto := ConvertToDisasterGroupDTO(rc)
	transport.WriteSuccess(ctx, consts.StatusCreated, dto, nil)
}

// 2. List DisasterGroups
func (h *GroupHandler) listGroups(c context.Context, ctx *app.RequestContext) {
	qParams := transport.ParseOptions(c, ctx)
	keyword := strings.ToLower(string(ctx.Query("keyword")))
	statusFilter := strings.ToLower(string(ctx.Query("status")))

	list, err := h.DisasterClient.DisasterV1().DisasterGroups(common.DisasterSystemNamespace).List(c, matev1.ListOptions{})
	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	// 预加载所有实例用于筛选
	allInstances := h.preloadInstances(c)
	allConfigs := h.preloadConfigs(c)

	// Filter, enrich & convert to DTOs
	dtos := make([]DisasterGroupDTO, 0, len(list.Items))
	for i := range list.Items {
		item := &list.Items[i]
		dto := h.buildGroupDTO(c, item, allInstances, allConfigs)

		// 关键字搜索
		if keyword != "" && !h.matchKeyword(dto, keyword) {
			continue
		}

		// 状态筛选
		if statusFilter != "" && statusFilter != "all" && !h.matchStatus(dto.Instances, statusFilter) {
			continue
		}

		dtos = append(dtos, dto)
	}

	// 按创建时间倒序排序 (最新创建的在前)
	sort.Slice(dtos, func(i, j int) bool {
		return dtos[i].CreationTimestamp.Time.Time.After(dtos[j].CreationTimestamp.Time.Time)
	})

	summary := summarizeDisasterGroupList(dtos)

	// Pagination
	pagedDtos, total := transport.Paginate(dtos, qParams)

	// Build Response
	requestPath := string(ctx.URI().Path())
	data, meta := transport.BuildCollectionResponse(
		requestPath,
		"disasterGroup",
		pagedDtos,
		qParams,
		total,
		nil,
		nil,
	)
	meta.Summary = summary
	transport.WriteSuccess(ctx, consts.StatusOK, data, meta)
}

func summarizeDisasterGroupList(groups []DisasterGroupDTO) map[string]int {
	instanceCount := 0
	abnormalCount := 0
	for _, group := range groups {
		instanceCount += len(group.Instances)
		if isAbnormalDisasterGroup(group) {
			abnormalCount++
		}
	}
	return map[string]int{
		"instanceCount": instanceCount,
		"abnormalCount": abnormalCount,
	}
}

func isAbnormalDisasterGroup(group DisasterGroupDTO) bool {
	if strings.TrimSpace(group.Status.Reason) != "" {
		return true
	}
	for _, cond := range group.Status.Conditions {
		if cond.Type == "Error" && cond.Status == matev1.ConditionTrue {
			return true
		}
	}
	if group.Status.FsmState == "Degraded" {
		return true
	}
	for _, inst := range group.Instances {
		switch inst.FsmState {
		case dapisv1.FsmStateFailed, "ConfigError", "NotFound":
			return true
		}
		if strings.TrimSpace(inst.Reason) != "" {
			return true
		}
	}
	return false
}

// 3. Get DisasterGroup
func (h *GroupHandler) getGroup(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")

	item, err := h.DisasterClient.DisasterV1().DisasterGroups(common.DisasterSystemNamespace).Get(c, name, matev1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	dto := h.buildGroupDTO(c, item, nil, nil)

	transport.WriteSuccess(ctx, consts.StatusOK, dto, nil)
}

// 4. Update DisasterGroup
func (h *GroupHandler) updateGroup(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")
	var req UpdateDisasterGroupRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	var result *dapisv1.DisasterGroup
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		existing, err := h.DisasterClient.DisasterV1().DisasterGroups(common.DisasterSystemNamespace).Get(c, name, matev1.GetOptions{})
		if err != nil {
			return err
		}

		if req.Description != nil {
			if existing.Annotations == nil {
				existing.Annotations = make(map[string]string)
			}
			existing.Annotations["testudo.softcdata.com/description"] = *req.Description
		}
		if req.Levels != nil {
			existing.Spec.Levels = req.Levels
		}
		if req.Policy != nil {
			existing.Spec.Policy = *req.Policy
		}

		transport.SetTraceAnnotation(&existing.ObjectMeta, ctx, metadata.AnnotationTraceID)
		result, err = h.DisasterClient.DisasterV1().DisasterGroups(common.DisasterSystemNamespace).Update(c, existing, matev1.UpdateOptions{})
		return err
	})

	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	dto := ConvertToDisasterGroupDTO(result)
	transport.WriteSuccess(ctx, consts.StatusOK, dto, nil)
}

// 5. Delete DisasterGroup
func (h *GroupHandler) deleteGroup(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")

	err := h.DisasterClient.DisasterV1().DisasterGroups(common.DisasterSystemNamespace).Delete(c, name, matev1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteSuccess(ctx, consts.StatusOK, utils.H{"name": name}, nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}
	transport.WriteSuccess(ctx, consts.StatusOK, utils.H{"name": name}, nil)
}

// 6. Execute Group Action (Similar to InstanceAction)
func (h *GroupHandler) executeAction(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")

	// Create request DTO similar to instance action
	var req instanceapi.ExecuteActionRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	// Validate Group exists
	_, err := h.DisasterClient.DisasterV1().DisasterGroups(common.DisasterSystemNamespace).Get(c, name, matev1.GetOptions{})
	if err != nil {
		transport.WriteError(ctx, transport.CodeNotFound, fmt.Sprintf("Group %s not found", name), nil)
		return
	}

	// Extract config values safely (支持首字母大小写兼容)
	force, _ := req.Config["force"].(bool)
	if !force {
		force, _ = req.Config["Force"].(bool)
	}

	skipFinalSync, _ := req.Config["skipFinalSync"].(bool)
	if !skipFinalSync {
		skipFinalSync, _ = req.Config["SkipFinalSync"].(bool)
	}

	skipScaleDownSource := false
	if req.Operation == "failover" {
		skipScaleDownSource, _ = req.Config["skipScaleDownSource"].(bool)
		if !skipScaleDownSource {
			skipScaleDownSource, _ = req.Config["SkipScaleDownSource"].(bool)
		}
	}

	// Map frontend action names to CRD enum types (same as instance handler)
	var opType dapisv1.OperationType
	switch req.Operation {
	case "sync-data":
		opType = dapisv1.OperationTypeSyncData
	case "sync-resource":
		opType = dapisv1.OperationTypeSyncResource
	default:
		opType = dapisv1.OperationType(req.Operation)
	}

	// Create DisasterOperation
	opName := fmt.Sprintf("%s-%s-%d", req.Operation, name, time.Now().Unix())

	// 获取 DisasterGroup 以读取 RetryPolicy
	group, err := h.DisasterClient.DisasterV1().DisasterGroups(common.DisasterSystemNamespace).Get(c, name, matev1.GetOptions{})
	var retryPolicy *dapisv1.RetryPolicy
	if err == nil {
		retryPolicy = group.Spec.Policy.RetryPolicy
	}

	op := &dapisv1.DisasterOperation{
		ObjectMeta: matev1.ObjectMeta{
			Name:      opName,
			Namespace: common.DisasterSystemNamespace,
			Labels: map[string]string{
				"testudo.softcdata.com/group":     name,
				"testudo.softcdata.com/operation": string(opType),
			},
		},
		Spec: dapisv1.DisasterOperationSpec{
			GroupName:     name, // Set GroupName instead of InstanceName
			OperationType: opType,
			Force:         force,
			SkipFinalSync: skipFinalSync,
			RetryPolicy:   retryPolicy,
		},
	}
	if skipScaleDownSource {
		if op.Annotations == nil {
			op.Annotations = make(map[string]string)
		}
		op.Annotations[skipScaleDownSourceAnnotation] = "true"
	}
	setSkipScaleDownSourceCompat(&op.Spec, skipScaleDownSource)

	// Extract timeout if present (JSON numbers are float64 unmarshaled into interface{})
	if tm, ok := req.Config["timeoutMinutes"].(float64); ok {
		op.Spec.TimeoutMinutes = int32(tm)
	}
	skipPodReadyCheck, skipProvided := parseBoolConfig(req.Config, "skipPodReadyCheck", "SkipPodReadyCheck")
	waitUntilReady, waitProvided := parseBoolConfig(req.Config, "waitUntilReady", "WaitUntilReady")
	switch {
	case skipProvided:
		op.Spec.SkipPodReadyCheck = boolPtr(skipPodReadyCheck)
		op.Spec.WaitUntilReady = !skipPodReadyCheck
	case waitProvided:
		op.Spec.SkipPodReadyCheck = boolPtr(!waitUntilReady)
		op.Spec.WaitUntilReady = waitUntilReady
	}

	transport.SetTraceAnnotation(&op.ObjectMeta, ctx, metadata.AnnotationTraceID)

	_, err = h.DisasterClient.DisasterV1().DisasterOperations(common.DisasterSystemNamespace).Create(c, op, matev1.CreateOptions{})
	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	transport.WriteSuccess(ctx, consts.StatusAccepted, utils.H{
		"operationName": opName,
		"message":       fmt.Sprintf("Group Operation %s started", req.Operation),
	}, nil)
}

// setSkipScaleDownSourceCompat 兼容不同版本的 disaster-operator 依赖。
// 当 Spec 存在 SkipScaleDownSource 字段时通过反射写入；旧版本则安全跳过。
func setSkipScaleDownSourceCompat(spec interface{}, value bool) {
	v := reflect.ValueOf(spec)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return
	}
	elem := v.Elem()
	if !elem.IsValid() || elem.Kind() != reflect.Struct {
		return
	}
	f := elem.FieldByName("SkipScaleDownSource")
	if f.IsValid() && f.CanSet() && f.Kind() == reflect.Bool {
		f.SetBool(value)
	}
}

func parseBoolConfig(config map[string]interface{}, keys ...string) (bool, bool) {
	for _, key := range keys {
		if value, ok := config[key].(bool); ok {
			return value, true
		}
	}
	return false, false
}

func boolPtr(v bool) *bool {
	out := v
	return &out
}

// 7. Get Group History
func (h *GroupHandler) getHistory(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")

	// List DisasterOperations with label selector
	labelSelector := fmt.Sprintf("testudo.softcdata.com/group=%s", name)
	list, err := h.DisasterClient.DisasterV1().DisasterOperations(common.DisasterSystemNamespace).List(c, matev1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	history := make([]instanceapi.HistoryDTO, 0, len(list.Items))
	for _, op := range list.Items {
		opCopy := op.DeepCopy()
		history = append(history, instanceapi.ConvertToHistoryDTO(opCopy))
	}

	// Sort by Time Descending
	sort.Slice(history, func(i, j int) bool {
		return history[i].Time.Time.Time.After(history[j].Time.Time.Time)
	})

	transport.WriteSuccess(ctx, consts.StatusOK, history, nil)
}

// ============== Helper Methods ==============

// preloadInstances 预加载所有 DisasterInstance 用于批量查询
func (h *GroupHandler) preloadInstances(c context.Context) map[string]*dapisv1.DisasterInstance {
	result := make(map[string]*dapisv1.DisasterInstance)

	list, err := h.DisasterClient.DisasterV1().DisasterInstances(common.DisasterSystemNamespace).List(c, matev1.ListOptions{})
	if err != nil {
		return result
	}

	for i := range list.Items {
		inst := &list.Items[i]
		result[inst.Name] = inst
	}

	return result
}

// preloadConfigs 预加载所有 DisasterConfig 用于批量查询
func (h *GroupHandler) preloadConfigs(c context.Context) map[string]*dapisv1.DisasterConfig {
	result := make(map[string]*dapisv1.DisasterConfig)

	list, err := h.DisasterClient.DisasterV1().DisasterConfigs().List(c, matev1.ListOptions{})
	if err != nil {
		return result
	}

	for i := range list.Items {
		cfg := &list.Items[i]
		result[cfg.Name] = cfg
	}

	return result
}

// collectInstanceSummaries 从 levels 中收集实例摘要（直接查询，用于详情接口）
// computeGroupFsmState 根据组内实例的 FsmState 推导组聚合状态及可用操作。
// 采用优先级投票算法：操作进行中 > 错误 > 全量匹配 > 初始化中 > 混合 > 空组。
func computeGroupFsmState(instances []InstanceSummaryDTO) (fsmState string, availableOps []string) {
	total := len(instances)
	if total == 0 {
		return "Unknown", []string{}
	}

	stats := make(map[string]int, total)
	for _, inst := range instances {
		stats[inst.FsmState]++
	}

	// 优先级 1：操作进行中（防止重入）
	if stats["FailingOver"] > 0 {
		return "FailingOver", []string{}
	}
	if stats["FailingBack"] > 0 {
		return "FailingBack", []string{}
	}

	// 优先级 2：有实例失败
	if stats["Failed"] > 0 {
		return "Degraded", []string{}
	}
	if stats["ConfigError"] > 0 {
		return "Degraded", []string{}
	}

	// 优先级 3：全量匹配
	if stats["Active"] == total {
		return "Active", []string{"reprotect"}
	}
	if stats["Paused"] == total {
		return "Paused", []string{"resume"}
	}
	if stats["Protected"] == total {
		return "Protected", []string{"failover", "pause", "synconce", "syncdata", "syncresource"}
	}

	// 优先级 4：有实例正在初始化
	if stats["Pending"]+stats["Initializing"] > 0 {
		return "Initializing", []string{}
	}

	// 优先级 5：其他混合状态（部分保护）
	return "PartialProtected", []string{"failover", "pause", "synconce"}
}

func (h *GroupHandler) collectInstanceSummaries(c context.Context, levels [][]string) []InstanceSummaryDTO {
	var result []InstanceSummaryDTO

	for levelIdx, instanceNames := range levels {
		for _, instName := range instanceNames {
			inst, err := h.DisasterClient.DisasterV1().DisasterInstances(common.DisasterSystemNamespace).Get(c, instName, matev1.GetOptions{})
			if err != nil {
				// 实例不存在或获取失败，添加占位条目
				result = append(result, InstanceSummaryDTO{
					Name:     instName,
					FsmState: "NotFound",
					Level:    levelIdx + 1,
				})
				continue
			}

			storageRepo := ""
			if inst.Spec.Config != "" {
				cfg, err := h.DisasterClient.DisasterV1().DisasterConfigs().Get(c, inst.Spec.Config, matev1.GetOptions{})
				if err == nil {
					storageRepo = cfg.Spec.StorageRepository
				}
			}

			result = append(result, InstanceSummaryDTO{
				Name:              inst.Name,
				Namespaces:        inst.Spec.Namespaces,
				FsmState:          inst.Status.FsmState,
				Reason:            inst.Status.Reason,
				Message:           inst.Status.Message,
				PrimaryCluster:    inst.Status.PrimaryCluster,
				SecondaryCluster:  inst.Status.SecondaryCluster,
				StorageRepository: storageRepo,
				Level:             levelIdx + 1,
			})
		}
	}

	return result
}

// collectInstanceSummariesWithCache 从 levels 中收集实例摘要（使用缓存，用于列表接口）
func (h *GroupHandler) collectInstanceSummariesWithCache(levels [][]string, cache map[string]*dapisv1.DisasterInstance, configCache map[string]*dapisv1.DisasterConfig) []InstanceSummaryDTO {
	var result []InstanceSummaryDTO

	for levelIdx, instanceNames := range levels {
		for _, instName := range instanceNames {
			if inst, ok := cache[instName]; ok {
				storageRepo := ""
				if cfg, okCfg := configCache[inst.Spec.Config]; okCfg {
					storageRepo = cfg.Spec.StorageRepository
				}

				result = append(result, InstanceSummaryDTO{
					Name:              inst.Name,
					Namespaces:        inst.Spec.Namespaces,
					FsmState:          inst.Status.FsmState,
					Reason:            inst.Status.Reason,
					Message:           inst.Status.Message,
					PrimaryCluster:    inst.Status.PrimaryCluster,
					SecondaryCluster:  inst.Status.SecondaryCluster,
					StorageRepository: storageRepo,
					Level:             levelIdx + 1,
				})
			} else {
				result = append(result, InstanceSummaryDTO{
					Name:     instName,
					FsmState: "NotFound",
					Level:    levelIdx + 1,
				})
			}
		}
	}

	return result
}

// matchKeyword 检查组是否匹配关键字
func (h *GroupHandler) matchKeyword(dto DisasterGroupDTO, keyword string) bool {
	if keyword == "" {
		return true
	}

	// 匹配名称
	if strings.Contains(strings.ToLower(dto.Name), keyword) {
		return true
	}

	// 匹配描述
	if strings.Contains(strings.ToLower(dto.Description), keyword) {
		return true
	}

	// 匹配实例名称或保护的命名空间
	for _, inst := range dto.Instances {
		if strings.Contains(strings.ToLower(inst.Name), keyword) {
			return true
		}
		for _, ns := range inst.Namespaces {
			if strings.Contains(strings.ToLower(ns), keyword) {
				return true
			}
		}
	}

	return false
}

// matchStatus 检查组内实例是否匹配状态筛选条件
func (h *GroupHandler) matchStatus(instances []InstanceSummaryDTO, statusFilter string) bool {
	if statusFilter == "" || statusFilter == "all" {
		return true
	}

	for _, inst := range instances {
		switch statusFilter {
		case "running":
			if inst.FsmState == dapisv1.FsmStateProtected || inst.FsmState == dapisv1.FsmStateInitializing {
				return true
			}
		case "paused":
			if inst.FsmState == dapisv1.FsmStatePaused {
				return true
			}
		case "error":
			if inst.FsmState == dapisv1.FsmStateFailed || inst.FsmState == "ConfigError" || inst.FsmState == "NotFound" || inst.FsmState == dapisv1.FsmStateFailingOver || inst.FsmState == dapisv1.FsmStateFailingBack {
				return true
			}
		}
	}

	return false
}

// instancePicker 返回轻量级实例列表，专供前端"选择容灾实例"场景使用。
//
// 查询参数：
//   - keyword: 模糊搜索（不区分大小写，匹配 name / namespace / description）
//   - status:  精确过滤 FsmState（与 keyword 为 AND 关系）
//   - page / limit: 标准分页
func (h *GroupHandler) instancePicker(c context.Context, ctx *app.RequestContext) {
	qParams := transport.ParseOptions(c, ctx)
	keyword := strings.ToLower(string(ctx.Query("keyword")))
	statusFilter := string(ctx.Query("status"))

	// 列举所有命名空间的 DisasterInstance
	list, err := h.DisasterClient.DisasterV1().DisasterInstances("").List(c, matev1.ListOptions{})
	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	dtos := make([]InstancePickerItemDTO, 0, len(list.Items))
	for i := range list.Items {
		inst := &list.Items[i]
		desc := inst.Annotations["testudo.softcdata.com/description"]
		fsmState := inst.Status.FsmState
		reason := getStructStringField(&inst.Status, "Reason")
		message := getStructStringField(&inst.Status, "Message")

		var cfg *dapisv1.DisasterConfig
		var cfgErr error
		if inst.Spec.Config != "" {
			cfg, cfgErr = h.DisasterClient.DisasterV1().DisasterConfigs().Get(c, inst.Spec.Config, matev1.GetOptions{})
		}
		displayState, reason, message := deriveGroupMemberStatus(fsmState, reason, message, inst.Spec.Config, cfg, cfgErr)

		// status 过滤：优先匹配对前端展示的状态，同时兼容历史按 fsmState 过滤。
		if statusFilter != "" && statusFilter != "all" && displayState != statusFilter && fsmState != statusFilter {
			continue
		}

		// keyword 模糊过滤（name / namespaces / description，Contains + 不区分大小写）
		if keyword != "" && !instancePickerMatchKeyword(inst.Name, strings.Join(inst.Spec.Namespaces, " "), desc, keyword) {
			continue
		}

		dtos = append(dtos, InstancePickerItemDTO{
			Name:        inst.Name,
			Namespaces:  inst.Spec.Namespaces,
			Description: desc,
			Status: GroupMemberStatusDTO{
				State:   displayState,
				Reason:  reason,
				Message: message,
			},
			FsmState: fsmState,
		})
	}

	// 分页
	pagedDtos, total := transport.Paginate(dtos, qParams)

	// 标准响应
	requestPath := string(ctx.URI().Path())
	data, meta := transport.BuildCollectionResponse(
		requestPath,
		"instancePickerItem",
		pagedDtos,
		qParams,
		total,
		nil,
		nil,
	)
	transport.WriteSuccess(ctx, consts.StatusOK, data, meta)
}

// instancePickerMatchKeyword 判断实例是否命中关键词（Contains，不区分大小写）
// 匹配范围：name / namespace / description（annotation 业务描述文字）
func instancePickerMatchKeyword(name, namespace, description, keyword string) bool {
	if strings.Contains(strings.ToLower(name), keyword) {
		return true
	}
	if strings.Contains(strings.ToLower(namespace), keyword) {
		return true
	}
	if strings.Contains(strings.ToLower(description), keyword) {
		return true
	}
	return false
}

// listGroupInstances 返回指定容灾组已选实例的摘要列表。
//
// 读取 DisasterGroup.spec.levels 中所有实例名称，查询对应的 DisasterInstance，
// 返回 GroupMemberInstanceDTO 数组，支持 keyword 过滤和 status 过滤。
func (h *GroupHandler) listGroupInstances(c context.Context, ctx *app.RequestContext) {
	groupName := ctx.Param("name")
	qParams := transport.ParseOptions(c, ctx)
	keyword := strings.ToLower(string(ctx.Query("keyword")))
	statusFilter := string(ctx.Query("status"))

	// 获取 DisasterGroup（搜索所有命名空间）
	groupList, err := h.DisasterClient.DisasterV1().DisasterGroups("").List(c, matev1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", groupName),
	})
	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}
	if len(groupList.Items) == 0 {
		transport.WriteError(ctx, transport.CodeNotFound, fmt.Sprintf("DisasterGroup %s not found", groupName), nil)
		return
	}
	group := &groupList.Items[0]

	// 展平 spec.levels，收集去重的实例名称
	seen := make(map[string]struct{})
	instanceNames := make([]string, 0)
	for _, level := range group.Spec.Levels {
		for _, instName := range level {
			if _, ok := seen[instName]; !ok {
				seen[instName] = struct{}{}
				instanceNames = append(instanceNames, instName)
			}
		}
	}

	// 查询各实例并构建 DTO
	dtos := make([]GroupMemberInstanceDTO, 0, len(instanceNames))
	for _, instName := range instanceNames {
		// 尝试在同命名空间找，找不到则搜全局
		inst, err := h.DisasterClient.DisasterV1().DisasterInstances(group.Namespace).Get(c, instName, matev1.GetOptions{})

		desc := ""
		namespaces := []string{}
		fsmState := "NotFound"
		displayState := "NotFound"
		reason := ""
		message := ""

		if err == nil {
			desc = inst.Annotations["testudo.softcdata.com/description"]
			namespaces = inst.Spec.Namespaces
			fsmState = inst.Status.FsmState
			reason = getStructStringField(&inst.Status, "Reason")
			message = getStructStringField(&inst.Status, "Message")

			var cfg *dapisv1.DisasterConfig
			var cfgErr error
			if inst.Spec.Config != "" {
				cfg, cfgErr = h.DisasterClient.DisasterV1().DisasterConfigs().Get(c, inst.Spec.Config, matev1.GetOptions{})
			}
			displayState, reason, message = deriveGroupMemberStatus(fsmState, reason, message, inst.Spec.Config, cfg, cfgErr)
		} else {
			reason = "InstanceNotFound"
			message = fmt.Sprintf("DisasterInstance %s not found", instName)
		}

		// status 过滤
		if statusFilter != "" && statusFilter != "all" && displayState != statusFilter {
			continue
		}

		// keyword 过滤（name / namespaces / description）
		if keyword != "" && !instancePickerMatchKeyword(instName, strings.Join(namespaces, " "), desc, keyword) {
			continue
		}

		dtos = append(dtos, GroupMemberInstanceDTO{
			Name:        instName,
			Description: desc,
			Namespaces:  namespaces,
			Status: GroupMemberStatusDTO{
				State:   displayState,
				Reason:  reason,
				Message: message,
			},
			FsmState: fsmState,
		})
	}

	// 分页
	pagedDtos, total := transport.Paginate(dtos, qParams)

	// 标准响应
	requestPath := string(ctx.URI().Path())
	data, meta := transport.BuildCollectionResponse(
		requestPath,
		"groupMemberInstance",
		pagedDtos,
		qParams,
		total,
		nil,
		nil,
	)
	transport.WriteSuccess(ctx, consts.StatusOK, data, meta)
}

func getStructStringField(status any, fieldName string) string {
	if status == nil {
		return ""
	}
	v := reflect.ValueOf(status)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return ""
	}
	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return ""
	}
	field := elem.FieldByName(fieldName)
	if !field.IsValid() || field.Kind() != reflect.String {
		return ""
	}
	return field.String()
}

func deriveGroupMemberStatus(
	fsmState, reason, message, configName string,
	config *dapisv1.DisasterConfig,
	configErr error,
) (state, outReason, outMessage string) {
	state = fsmState
	if state == "" {
		state = "Unknown"
	}

	outReason = reason
	outMessage = message

	// 实例自身已透出错误语义时，组成员接口应直接呈现失败态，避免“看起来正常”。
	if outReason != "" && state != "Failed" && state != "NotFound" {
		state = "Failed"
	}

	if configName == "" {
		return state, outReason, outMessage
	}

	if configErr != nil {
		if errors.IsNotFound(configErr) {
			if outReason == "" {
				outReason = "ConfigNotFound"
			}
			if outMessage == "" {
				outMessage = fmt.Sprintf("DisasterConfig %s not found", configName)
			}
			if state != "NotFound" {
				state = "ConfigError"
			}
		}
		return state, outReason, outMessage
	}

	if config == nil {
		return state, outReason, outMessage
	}

	if config.Status.Status == dapisv1.DisasterConfigStatusError || config.Status.Status == dapisv1.DisasterConfigStatusNotReady {
		if outReason == "" {
			outReason = config.Status.Reason
			if outReason == "" {
				outReason = "ConfigNotReady"
			}
		}
		if outMessage == "" {
			outMessage = config.Status.Message
			if outMessage == "" {
				outMessage = fmt.Sprintf("DisasterConfig %s status=%s", configName, config.Status.Status)
			}
		}
		if state != "NotFound" {
			state = "ConfigError"
		}
	}

	return state, outReason, outMessage
}

func (h *GroupHandler) buildGroupDTO(
	c context.Context,
	item *dapisv1.DisasterGroup,
	instanceCache map[string]*dapisv1.DisasterInstance,
	configCache map[string]*dapisv1.DisasterConfig,
) DisasterGroupDTO {
	dto := ConvertToDisasterGroupDTO(item)
	dto.Status.Summary = fmt.Sprintf("%d Levels, %d Instances", len(item.Spec.Levels), item.Status.TotalInstances)

	if instanceCache != nil && configCache != nil {
		dto.Instances = h.collectInstanceSummariesWithCache(item.Spec.Levels, instanceCache, configCache)
	} else {
		dto.Instances = h.collectInstanceSummaries(c, item.Spec.Levels)
	}

	dto.Status.FsmState, dto.Status.AvailableOperations = computeGroupFsmState(dto.Instances)
	return dto
}

// watchGroupStatuses 监听容灾组状态的实时变化
func (h *GroupHandler) watchGroupStatuses(c context.Context, ctx *app.RequestContext) {
	watcherFunc := func(watchCtx context.Context) (watch.Interface, error) {
		listOpts := matev1.ListOptions{}
		// List+Watch 模式：先 List 获取当前 ResourceVersion，
		// 再从该版本开始 Watch，避免历史资源在建立连接时全量 ADDED 洪泛
		existing, err := h.DisasterClient.DisasterV1().DisasterGroups(common.DisasterSystemNamespace).List(watchCtx, listOpts)
		if err != nil {
			return nil, err
		}
		listOpts.ResourceVersion = existing.ResourceVersion
		return h.DisasterClient.DisasterV1().DisasterGroups(common.DisasterSystemNamespace).Watch(watchCtx, listOpts)
	}

	converter := func(obj interface{}) interface{} {
		if group, ok := obj.(*dapisv1.DisasterGroup); ok {
			return h.buildGroupDTO(c, group, nil, nil)
		}
		return nil
	}

	watchutils.StreamWatch(c, ctx, watcherFunc, converter)
}

// watchGroupStatus 监听单个容灾组状态的实时变化
func (h *GroupHandler) watchGroupStatus(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")

	watcherFunc := func(watchCtx context.Context) (watch.Interface, error) {
		return h.DisasterClient.DisasterV1().DisasterGroups(common.DisasterSystemNamespace).Watch(watchCtx, matev1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", name),
		})
	}

	converter := func(obj interface{}) interface{} {
		if group, ok := obj.(*dapisv1.DisasterGroup); ok {
			return h.buildGroupDTO(c, group, nil, nil)
		}
		return nil
	}

	watchutils.StreamWatch(c, ctx, watcherFunc, converter)
}

// watchGroupOperations 监听容灾组操作的实时状态变化
//
// 查询参数：
//   - groupName: 可选，按容灾组名过滤（使用 LabelSelector），留空则监听所有操作
func (h *GroupHandler) watchGroupOperations(c context.Context, ctx *app.RequestContext) {
	groupName := string(ctx.Query("groupName"))

	watcherFunc := func(watchCtx context.Context) (watch.Interface, error) {
		listOpts := matev1.ListOptions{}
		if groupName != "" {
			listOpts.LabelSelector = fmt.Sprintf("testudo.softcdata.com/group=%s", groupName)
		}
		// List+Watch 模式：先 List 获取当前 ResourceVersion，
		// 再从该版本开始 Watch，跳过将历史已存在的资源全量推送为 ADDED 的问题
		existing, err := h.DisasterClient.DisasterV1().DisasterOperations(common.DisasterSystemNamespace).List(watchCtx, listOpts)
		if err != nil {
			return nil, err
		}
		listOpts.ResourceVersion = existing.ResourceVersion
		return h.DisasterClient.DisasterV1().DisasterOperations(common.DisasterSystemNamespace).Watch(watchCtx, listOpts)
	}

	converter := func(obj interface{}) interface{} {
		if op, ok := obj.(*dapisv1.DisasterOperation); ok {
			return ConvertToDisasterOperationDTO(op)
		}
		return nil
	}

	watchutils.StreamWatch(c, ctx, watcherFunc, converter)
}

// watchGroupOperation 监听指定容灾组操作（或指定容灾组下所有操作）的实时状态变化
//
// 此接口支持动态探测：
// - 如果参数 operationName 是一个已存在的 DisasterGroup 名称，则通过 LabelSelector 监听该组全量操作（使用 List+Watch 模式防 ADDED 洪泛）。
// - 如果参数 operationName 只是单个操作名称，则直接使用 FieldSelector 监听单个操作进度（需立即下发 ADDED 反映当前状态）。
func (h *GroupHandler) watchGroupOperation(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("operationName")

	watcherFunc := func(watchCtx context.Context) (watch.Interface, error) {
		// 动态判断：检查传入的 name 是容灾组名还是单个操作名。
		_, err := h.DisasterClient.DisasterV1().DisasterGroups(common.DisasterSystemNamespace).Get(watchCtx, name, matev1.GetOptions{})
		if err == nil {
			// 存在对应名称的组，进行 List+Watch 模式（监听组内所有操作）
			listOpts := matev1.ListOptions{
				LabelSelector: fmt.Sprintf("testudo.softcdata.com/group=%s", name),
			}
			existing, listErr := h.DisasterClient.DisasterV1().DisasterOperations(common.DisasterSystemNamespace).List(watchCtx, listOpts)
			if listErr != nil {
				return nil, listErr
			}
			listOpts.ResourceVersion = existing.ResourceVersion
			return h.DisasterClient.DisasterV1().DisasterOperations(common.DisasterSystemNamespace).Watch(watchCtx, listOpts)
		}

		// 如果不存在同名组，说明这是一个具体的操作名称，直接 Watch 单个对象
		return h.DisasterClient.DisasterV1().DisasterOperations(common.DisasterSystemNamespace).Watch(watchCtx, matev1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", name),
		})
	}

	converter := func(obj interface{}) interface{} {
		if op, ok := obj.(*dapisv1.DisasterOperation); ok {
			return ConvertToDisasterOperationDTO(op)
		}
		return nil
	}

	watchutils.StreamWatch(c, ctx, watcherFunc, converter)
}
