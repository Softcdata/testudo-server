package drill

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	metadata "github.com/softcdata/testudo-operator/pkg/metadata"
	instancev1 "github.com/softcdata/testudo-server/internal/apis/disaster_instance/v1"
	velerohooks "github.com/softcdata/testudo-server/internal/apis/velero_hooks"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/kube"
	"github.com/softcdata/testudo-server/internal/transport"
	watchutils "github.com/softcdata/testudo-server/internal/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
)

// DrillHandler 处理容灾演练 API
type DrillHandler struct {
	*kube.KubeClient
	Rg *route.RouterGroup
	Mw []app.HandlerFunc

	GetClusterRESTConfigFunc      func(ctx context.Context, clusterName string) (*rest.Config, error)
	BuildBulkModifierSnapshotFunc func(ctx context.Context, spec *dapisv1.DisasterInstanceSpec, restConfig *rest.Config) (*instancev1.BulkModifierSnapshotBuildResult, error)
}

// NewDrillHandler 创建 DrillHandler
func NewDrillHandler(kc *kube.KubeClient, rg *route.RouterGroup, mw ...app.HandlerFunc) *DrillHandler {
	return &DrillHandler{
		KubeClient: kc,
		Rg:         rg,
		Mw:         mw,
	}
}

// 8. Restart Drill - 重跑演练
func (h *DrillHandler) restartDrill(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")

	// 命名空间解析
	ns := ctx.QueryArgs().Peek("namespace")
	namespace := string(ns)
	if namespace == "" {
		var err error
		namespace, err = h.findNamespace(c, name)
		if err != nil {
			if errors.IsNotFound(err) {
				transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
				return
			}
			transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
			return
		}
	}

	var result *dapisv1.DisasterDrill
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// 获取当前状态
		existing, err := h.DisasterClient.DisasterV1().DisasterDrills(namespace).Get(c, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		// 验证状态必须是 Completed 或 Failed
		if existing.Status.State != dapisv1.DrillStateCompleted && existing.Status.State != dapisv1.DrillStateFailed {
			return fmt.Errorf("drill must be in Completed or Failed state to restart, current: %s", existing.Status.State)
		}

		// 设置 Restart Annotation
		if existing.Annotations == nil {
			existing.Annotations = make(map[string]string)
		}
		existing.Annotations[metadata.AnnotationRestartTimestamp] = time.Now().Format(time.RFC3339)

		// 注入 TraceID
		transport.SetTraceAnnotation(&existing.ObjectMeta, ctx, metadata.AnnotationTraceID)

		result, err = h.DisasterClient.DisasterV1().DisasterDrills(namespace).Update(c, existing, metav1.UpdateOptions{})
		return err
	})

	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		// 状态校验失败 (非 conflict 错误)
		if result == nil {
			transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	dto := ConvertToDisasterDrillDTO(result)
	transport.WriteSuccess(ctx, consts.StatusOK, dto, nil)
}

// findNamespace 根据名称查找 Drill 所在的命名空间
func (h *DrillHandler) findNamespace(c context.Context, name string) (string, error) {
	list, err := h.DisasterClient.DisasterV1().DisasterDrills("").List(c, metav1.ListOptions{
		FieldSelector: "metadata.name=" + name,
	})
	if err != nil {
		return "", err
	}
	if len(list.Items) == 0 {
		return "", errors.NewNotFound(dapisv1.Resource("disasterdrill"), name)
	}
	return list.Items[0].Namespace, nil
}

// 0. Get Protected Namespaces - 查询实例/容灾组保护的命名空间
func (h *DrillHandler) getProtectedNamespaces(c context.Context, ctx *app.RequestContext) {
	instanceName := strings.TrimSpace(string(ctx.Query("instanceName")))
	groupName := strings.TrimSpace(string(ctx.Query("groupName")))
	namespace := strings.TrimSpace(string(ctx.Query("namespace")))
	if namespace == "" {
		namespace = common.DisasterSystemNamespace
	}

	// 二选一参数校验
	if (instanceName == "" && groupName == "") || (instanceName != "" && groupName != "") {
		transport.WriteError(ctx, transport.CodeBadRequest, "exactly one of instanceName or groupName is required", nil)
		return
	}

	// 实例维度查询
	if instanceName != "" {
		inst, err := h.DisasterClient.DisasterV1().DisasterInstances(namespace).Get(c, instanceName, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
				return
			}
			transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
			return
		}

		transport.WriteSuccess(ctx, consts.StatusOK, ProtectedNamespacesDTO{
			Type:         "Instance",
			InstanceName: inst.Name,
			Namespaces:   uniqueSortedStrings(inst.Spec.Namespaces),
		}, nil)
		return
	}

	// 容灾组维度查询（遍历组内实例并聚合）
	group, err := h.DisasterClient.DisasterV1().DisasterGroups(namespace).Get(c, groupName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	instanceNames := flattenAndDedupeGroupInstances(group.Spec.Levels)
	aggregatedNamespaces := make([]string, 0)
	missingInstances := make([]string, 0)

	for _, instName := range instanceNames {
		inst, getErr := h.DisasterClient.DisasterV1().DisasterInstances(group.Namespace).Get(c, instName, metav1.GetOptions{})
		if getErr != nil {
			if errors.IsNotFound(getErr) {
				missingInstances = append(missingInstances, instName)
				continue
			}
			transport.WriteError(ctx, transport.CodeInternalServerError, getErr.Error(), nil)
			return
		}
		aggregatedNamespaces = append(aggregatedNamespaces, inst.Spec.Namespaces...)
	}

	transport.WriteSuccess(ctx, consts.StatusOK, ProtectedNamespacesDTO{
		Type:             "Group",
		GroupName:        group.Name,
		Namespaces:       uniqueSortedStrings(aggregatedNamespaces),
		MissingInstances: uniqueSortedStrings(missingInstances),
	}, nil)
}

func (h *DrillHandler) getDisasterInstanceByName(c context.Context, namespace, name string) (*dapisv1.DisasterInstance, error) {
	// 优先使用指定命名空间查询
	if namespace != "" {
		inst, err := h.DisasterClient.DisasterV1().DisasterInstances(namespace).Get(c, name, metav1.GetOptions{})
		if err == nil {
			return inst, nil
		}
		if !errors.IsNotFound(err) {
			return nil, err
		}
	}

	// 回退全局按名称查询
	list, err := h.DisasterClient.DisasterV1().DisasterInstances("").List(c, metav1.ListOptions{
		FieldSelector: "metadata.name=" + name,
	})
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		if list.Items[i].Name == name {
			return &list.Items[i], nil
		}
	}
	return nil, errors.NewNotFound(dapisv1.Resource("disasterinstance"), name)
}

func (h *DrillHandler) getDisasterGroupByName(c context.Context, namespace, name string) (*dapisv1.DisasterGroup, error) {
	// 优先使用指定命名空间查询
	if namespace != "" {
		group, err := h.DisasterClient.DisasterV1().DisasterGroups(namespace).Get(c, name, metav1.GetOptions{})
		if err == nil {
			return group, nil
		}
		if !errors.IsNotFound(err) {
			return nil, err
		}
	}

	// 回退全局按名称查询
	list, err := h.DisasterClient.DisasterV1().DisasterGroups("").List(c, metav1.ListOptions{
		FieldSelector: "metadata.name=" + name,
	})
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		if list.Items[i].Name == name {
			return &list.Items[i], nil
		}
	}
	return nil, errors.NewNotFound(dapisv1.Resource("disastergroup"), name)
}

func (h *DrillHandler) resolveSourceClusterByConfig(c context.Context, configName string) (string, error) {
	configName = strings.TrimSpace(configName)
	if configName == "" {
		return "", fmt.Errorf("DrillRestorePolicyInvalid: instance.spec.config is empty")
	}
	cfg, err := h.DisasterClient.DisasterV1().DisasterConfigs().Get(c, configName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return "", fmt.Errorf("DrillRestorePolicyInvalid: config %s not found", configName)
		}
		return "", err
	}
	sourceCluster := strings.TrimSpace(cfg.Spec.SourceCluster)
	if sourceCluster == "" {
		return "", fmt.Errorf("DrillRestorePolicyInvalid: config %s has empty spec.sourceCluster", configName)
	}
	return sourceCluster, nil
}

func (h *DrillHandler) buildPreparedDrillRestorePolicy(
	c context.Context,
	namespace string,
	instance *dapisv1.DisasterInstance,
	group *dapisv1.DisasterGroup,
	req *CreateDrillRequest,
) (*dapisv1.RestorePolicy, error) {
	if req == nil || req.RestorePolicy == nil {
		return nil, nil
	}

	policy, err := req.RestorePolicy.ToCRD()
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return nil, nil
	}

	var spec *dapisv1.DisasterInstanceSpec
	if instance != nil {
		spec = instance.Spec.DeepCopy()
		spec.RestorePolicy = policy.DeepCopy()
	} else {
		if group == nil {
			return nil, fmt.Errorf("DrillRestorePolicyInvalid: missing drill target")
		}
		instanceNames := flattenAndDedupeGroupInstances(group.Spec.Levels)
		if len(instanceNames) == 0 {
			return nil, fmt.Errorf("DrillRestorePolicyInvalid: disaster group %s has no instances", group.Name)
		}

		namespaces := make([]string, 0)
		sourceClusters := make(map[string]struct{})
		var configName string
		for _, instName := range instanceNames {
			inst, err := h.getDisasterInstanceByName(c, namespace, instName)
			if err != nil {
				return nil, fmt.Errorf("DrillRestorePolicyInvalid: get group instance %s failed: %w", instName, err)
			}
			if configName == "" {
				configName = strings.TrimSpace(inst.Spec.Config)
			}
			sourceCluster, err := h.resolveSourceClusterByConfig(c, inst.Spec.Config)
			if err != nil {
				return nil, err
			}
			sourceClusters[sourceCluster] = struct{}{}
			namespaces = append(namespaces, inst.Spec.Namespaces...)
		}

		if len(sourceClusters) > 1 {
			return nil, fmt.Errorf(
				"DrillRestorePolicyInvalid: group drill restorePolicy requires all member instances to share the same source cluster",
			)
		}

		spec = &dapisv1.DisasterInstanceSpec{
			Config:        configName,
			Namespaces:    uniqueSortedStrings(namespaces),
			RestorePolicy: policy.DeepCopy(),
		}
	}

	tmpHandler := &instancev1.InstanceHandler{KubeClient: h.KubeClient}
	tmpHandler.GetClusterRESTConfigFunc = h.GetClusterRESTConfigFunc
	if h.BuildBulkModifierSnapshotFunc != nil {
		tmpHandler.BuildBulkModifierSnapshotFunc = func(
			ctx context.Context,
			spec *dapisv1.DisasterInstanceSpec,
			restConfig *rest.Config,
		) (*instancev1.BulkModifierSnapshotBuildResult, error) {
			return h.BuildBulkModifierSnapshotFunc(ctx, spec, restConfig)
		}
	}
	if err := tmpHandler.PrepareRestorePolicyForPersist(c, spec, nil); err != nil {
		return nil, err
	}
	if spec.RestorePolicy == nil {
		return nil, nil
	}
	return spec.RestorePolicy.DeepCopy(), nil
}

func flattenAndDedupeGroupInstances(levels [][]string) []string {
	result := make([]string, 0)
	seen := make(map[string]struct{})

	for _, level := range levels {
		for _, name := range level {
			n := strings.TrimSpace(name)
			if n == "" {
				continue
			}
			if _, ok := seen[n]; ok {
				continue
			}
			seen[n] = struct{}{}
			result = append(result, n)
		}
	}
	return result
}

func uniqueSortedStrings(values []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(values))

	for _, v := range values {
		item := strings.TrimSpace(v)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	sort.Strings(result)
	return result
}

// 1. List Drills - 列出所有演练
func (h *DrillHandler) listDrills(c context.Context, ctx *app.RequestContext) {
	qParams := transport.ParseOptions(c, ctx)

	// 解析过滤参数
	ns := qParams.Filters["namespace"]
	instanceName := string(ctx.Query("instanceName"))
	groupName := string(ctx.Query("groupName"))
	stateFilter := string(ctx.Query("state"))
	typeFilter := string(ctx.Query("type")) // Instance, Group
	keyword := string(ctx.Query("keyword")) // Name fuzzy search

	// 列出 CRs
	list, err := h.DisasterClient.DisasterV1().DisasterDrills(ns).List(c, metav1.ListOptions{})
	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	// 内存过滤
	dtos := make([]DisasterDrillDTO, 0, len(list.Items))
	for i := range list.Items {
		item := &list.Items[i]

		// 1. Keyword Search (Name)
		if keyword != "" && !strings.Contains(item.Name, keyword) {
			continue
		}

		// 2. Object Type Filter
		if typeFilter != "" {
			if typeFilter == "Instance" && item.Spec.InstanceName == "" {
				continue
			}
			if typeFilter == "Group" && item.Spec.GroupName == "" {
				continue
			}
		}

		// 3. Existing Exact Filters
		if instanceName != "" && item.Spec.InstanceName != instanceName {
			continue
		}
		if groupName != "" && item.Spec.GroupName != groupName {
			continue
		}

		// 4. State Filter (Mapped)
		if stateFilter != "" {
			current := string(item.Status.State)
			// Map specific UI states to CRD states
			if stateFilter == "NotStarted" {
				if current != string(dapisv1.DrillStatePending) && current != string(dapisv1.DrillStateReady) {
					continue
				}
			} else if stateFilter == "InProgress" { // Alias for Executing if needed, or just use Executing
				if current != string(dapisv1.DrillStateExecuting) {
					continue
				}
			} else {
				// Exact match
				if current != stateFilter {
					continue
				}
			}
		}

		dtos = append(dtos, ConvertToDisasterDrillDTO(item))
	}

	// 按创建时间倒序排序
	sort.Slice(dtos, func(i, j int) bool {
		return dtos[i].CreationTimestamp.Time.Time.After(dtos[j].CreationTimestamp.Time.Time)
	})

	// 分页
	pagedDtos, total := transport.Paginate(dtos, qParams)

	// 构建响应
	requestPath := string(ctx.URI().Path())
	data, meta := transport.BuildCollectionResponse(
		requestPath,
		"disasterDrill",
		pagedDtos,
		qParams,
		total,
		nil,
		nil,
	)
	transport.WriteSuccess(ctx, consts.StatusOK, data, meta)
}

// 2. Get Drill - 获取演练详情
func (h *DrillHandler) getDrill(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")

	// 命名空间解析
	ns := ctx.QueryArgs().Peek("namespace")
	namespace := string(ns)
	if namespace == "" {
		var err error
		namespace, err = h.findNamespace(c, name)
		if err != nil {
			if errors.IsNotFound(err) {
				transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
				return
			}
			transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
			return
		}
	}

	item, err := h.DisasterClient.DisasterV1().DisasterDrills(namespace).Get(c, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	dto := ConvertToDisasterDrillDTO(item)
	transport.WriteSuccess(ctx, consts.StatusOK, dto, nil)
}

// 3. Create Drill - 创建演练
func (h *DrillHandler) createDrill(c context.Context, ctx *app.RequestContext) {
	var req CreateDrillRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	// 验证必填字段 (二选一)
	if req.InstanceName != "" && req.GroupName != "" {
		transport.WriteError(ctx, transport.CodeBadRequest, "cannot specify both instanceName and groupName", nil)
		return
	}
	if req.InstanceName == "" && req.GroupName == "" {
		transport.WriteError(ctx, transport.CodeBadRequest, "instanceName or groupName is required", nil)
		return
	}

	// 校验名称长度
	if req.Name != "" && len(req.Name) > 63 {
		transport.WriteError(ctx, transport.CodeBadRequest, fmt.Sprintf("drill name must be no more than 63 characters, got %d", len(req.Name)), nil)
		return
	}
	if req.VeleroHooks != nil {
		if req.VeleroHooks.DataBackupSet {
			err := &velerohooks.ValidationError{
				FieldPath: "veleroHooks.dataBackup",
				Message:   "drill veleroHooks.dataBackup is not supported",
			}
			transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), velerohooks.ErrorMeta(err))
			return
		}
		if err := velerohooks.ValidateRestoreHooks(req.VeleroHooks.DataRestore, "veleroHooks.dataRestore"); err != nil {
			transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), velerohooks.ErrorMeta(err))
			return
		}
	}

	// 命名空间
	namespace := req.Namespace
	if namespace == "" {
		namespace = common.DisasterSystemNamespace
	}

	var labels map[string]string
	var drillNameBase string
	var instance *dapisv1.DisasterInstance
	var group *dapisv1.DisasterGroup

	if req.InstanceName != "" {
		// --- 实例演练 ---
		drillNameBase = req.InstanceName
		labels = map[string]string{
			"testudo.softcdata.com/instance": req.InstanceName,
		}

		// 验证 DisasterInstance 存在
		var err error
		instance, err = h.DisasterClient.DisasterV1().DisasterInstances(namespace).Get(c, req.InstanceName, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				// 尝试全局查找
				list, listErr := h.DisasterClient.DisasterV1().DisasterInstances("").List(c, metav1.ListOptions{
					FieldSelector: "metadata.name=" + req.InstanceName,
				})
				if listErr != nil || len(list.Items) == 0 {
					transport.WriteError(ctx, transport.CodeNotFound,
						fmt.Sprintf("DisasterInstance %s not found", req.InstanceName), nil)
					return
				}
				instance = &list.Items[0]
				namespace = instance.Namespace
			} else {
				transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
				return
			}
		}
	} else {
		// --- 容灾组演练 ---
		drillNameBase = req.GroupName
		labels = map[string]string{
			"testudo.softcdata.com/group": req.GroupName,
		}

		// 验证 DisasterGroup 存在
		var err error
		group, err = h.DisasterClient.DisasterV1().DisasterGroups(namespace).Get(c, req.GroupName, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				// 尝试全局查找
				list, listErr := h.DisasterClient.DisasterV1().DisasterGroups("").List(c, metav1.ListOptions{
					FieldSelector: "metadata.name=" + req.GroupName,
				})
				if listErr != nil || len(list.Items) == 0 {
					transport.WriteError(ctx, transport.CodeNotFound,
						fmt.Sprintf("DisasterGroup %s not found", req.GroupName), nil)
					return
				}
				group = &list.Items[0]
				namespace = group.Namespace
			} else {
				transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
				return
			}
		}
	}

	restorePolicy, err := h.buildPreparedDrillRestorePolicy(c, namespace, instance, group, &req)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	// 生成演练名称
	drillName := req.Name
	if drillName == "" {
		drillName = GenerateDrillName(drillNameBase)
	}

	// 构建 CRD
	drill := dapisv1.DisasterDrill{
		ObjectMeta: metav1.ObjectMeta{
			Name:      drillName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: req.ToCRD(restorePolicy),
	}

	// 注入 TraceID
	transport.SetTraceAnnotation(&drill.ObjectMeta, ctx, metadata.AnnotationTraceID)

	// 创建
	created, err := h.DisasterClient.DisasterV1().DisasterDrills(namespace).Create(c, &drill, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			transport.WriteError(ctx, transport.CodeConflict, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	dto := ConvertToDisasterDrillDTO(created)
	transport.WriteSuccess(ctx, consts.StatusCreated, dto, nil)
}

// 4. Confirm Drill - 确认执行演练
func (h *DrillHandler) confirmDrill(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")

	// 命名空间解析
	ns := ctx.QueryArgs().Peek("namespace")
	namespace := string(ns)
	if namespace == "" {
		var err error
		namespace, err = h.findNamespace(c, name)
		if err != nil {
			if errors.IsNotFound(err) {
				transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
				return
			}
			transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
			return
		}
	}

	var result *dapisv1.DisasterDrill
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// 获取当前状态
		existing, err := h.DisasterClient.DisasterV1().DisasterDrills(namespace).Get(c, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		// 验证状态必须是 Ready
		if existing.Status.State != dapisv1.DrillStateReady {
			return fmt.Errorf("drill must be in Ready state to confirm, current: %s", existing.Status.State)
		}

		// 已经确认过
		if existing.Spec.Confirmed {
			result = existing
			return nil
		}

		// 设置 confirmed = true
		existing.Spec.Confirmed = true

		// 注入 TraceID
		transport.SetTraceAnnotation(&existing.ObjectMeta, ctx, metadata.AnnotationTraceID)

		result, err = h.DisasterClient.DisasterV1().DisasterDrills(namespace).Update(c, existing, metav1.UpdateOptions{})
		return err
	})

	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		// 状态校验失败
		if result == nil {
			transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	dto := ConvertToDisasterDrillDTO(result)
	transport.WriteSuccess(ctx, consts.StatusOK, dto, nil)
}

// 5. Delete Drill - 删除演练
func (h *DrillHandler) deleteDrill(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")

	// 命名空间解析
	ns := ctx.QueryArgs().Peek("namespace")
	namespace := string(ns)
	if namespace == "" {
		var err error
		namespace, err = h.findNamespace(c, name)
		if err != nil {
			// 如果找不到，可能已经被删除
			transport.WriteSuccess(ctx, consts.StatusOK, utils.H{"name": name}, nil)
			return
		}
	}

	err := h.DisasterClient.DisasterV1().DisasterDrills(namespace).Delete(c, name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteSuccess(ctx, consts.StatusOK, utils.H{"name": name}, nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	transport.WriteSuccess(ctx, consts.StatusOK, utils.H{"name": name, "deleted": true}, nil)
}

// 6. Watch Drills (List)
func (h *DrillHandler) watchDrills(c context.Context, ctx *app.RequestContext) {
	watcherFunc := func(ctx context.Context) (watch.Interface, error) {
		// Parse filters
		// Note: FieldSelector only works for metadata fields client-side usually unless serverside supports it.
		// For simplicity, we watch all and filter later if needed, but here we just stream all changes
		// in all namespaces or specific namespace if provided.
		// Realistically, to support filtering by instanceName/groupName in Watch is hard without filtering in loop.
		// transport.ParseOptions is not available inside here easily without passing params.
		// Let's watch all in all namespaces for now, similar to other handlers.
		return h.DisasterClient.DisasterV1().DisasterDrills("").Watch(ctx, metav1.ListOptions{})
	}

	converter := func(obj interface{}) interface{} {
		if item, ok := obj.(*dapisv1.DisasterDrill); ok {
			return ConvertToDisasterDrillDTO(item)
		}
		return nil
	}

	watchutils.StreamWatch(c, ctx, watcherFunc, converter)
}

// 7. Watch Drill (Single)
func (h *DrillHandler) watchDrill(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")

	watcherFunc := func(ctx context.Context) (watch.Interface, error) {
		// Find Namespace first
		ns, _ := h.findNamespace(c, name) // Ignore error, empty means all

		return h.DisasterClient.DisasterV1().DisasterDrills(ns).Watch(ctx, metav1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", name),
		})
	}

	converter := func(obj interface{}) interface{} {
		if item, ok := obj.(*dapisv1.DisasterDrill); ok {
			return ConvertToDisasterDrillDTO(item)
		}
		return nil
	}

	watchutils.StreamWatch(c, ctx, watcherFunc, converter)
}

// 9. Cleanup Drill - 清理演练资源
func (h *DrillHandler) cleanupDrill(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")

	// 命名空间解析
	ns := ctx.QueryArgs().Peek("namespace")
	namespace := string(ns)
	if namespace == "" {
		var err error
		namespace, err = h.findNamespace(c, name)
		if err != nil {
			if errors.IsNotFound(err) {
				transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
				return
			}
			transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
			return
		}
	}

	var result *dapisv1.DisasterDrill
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// 获取当前状态
		existing, err := h.DisasterClient.DisasterV1().DisasterDrills(namespace).Get(c, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		// 验证状态必须是 Completed
		if existing.Status.State != dapisv1.DrillStateCompleted {
			return fmt.Errorf("drill must be in Completed state to cleanup, current: %s", existing.Status.State)
		}

		// 已有 Cleanup: true，拒绝
		if existing.Spec.CleanUp {
			return fmt.Errorf("drill cleanup is already triggered")
		}

		// 设置 cleanup = true
		existing.Spec.CleanUp = true

		// 注入 TraceID
		transport.SetTraceAnnotation(&existing.ObjectMeta, ctx, metadata.AnnotationTraceID)

		result, err = h.DisasterClient.DisasterV1().DisasterDrills(namespace).Update(c, existing, metav1.UpdateOptions{})
		return err
	})

	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		// 状态校验失败
		if result == nil {
			transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	dto := ConvertToDisasterDrillDTO(result)
	transport.WriteSuccess(ctx, consts.StatusOK, dto, nil)
}
