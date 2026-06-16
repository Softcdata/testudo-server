package instance

import (
	"context"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	hlog "github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	listers "github.com/softcdata/testudo-operator/pkg/listers/disaster/v1"
	metadata "github.com/softcdata/testudo-operator/pkg/metadata"
	velerohooks "github.com/softcdata/testudo-server/internal/apis/velero_hooks"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/i18n"
	"github.com/softcdata/testudo-server/internal/kube"
	"github.com/softcdata/testudo-server/internal/transport"
	watchutils "github.com/softcdata/testudo-server/internal/utils"
	networkingv1 "k8s.io/api/networking/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	matev1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"
	ctrclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type InstanceHandler struct {
	*kube.KubeClient
	Rg *route.RouterGroup
	Mw []app.HandlerFunc

	DisasterConfigLister          listers.DisasterConfigLister
	InstanceLister                listers.DisasterInstanceLister
	GetClusterClient              func(ctx context.Context, clusterName string) (ctrclient.Client, error)
	GetClusterRESTConfigFunc      func(ctx context.Context, clusterName string) (*rest.Config, error)
	BuildBulkModifierSnapshotFunc func(ctx context.Context, spec *dapisv1.DisasterInstanceSpec, restConfig *rest.Config) (*bulkModifierSnapshotBuildResult, error)
}

const (
	syncHistorySourceSyncRecord = "syncRecord"
	syncHistorySourceOperation  = "operation"
	syncHistorySourceAll        = "all"

	syncHistoryTypeAll          = "all"
	syncHistoryTypeDataSync     = "dataSync"
	syncHistoryTypeResourceSync = "resourceSync"
	syncHistoryTypeSyncOnce     = "syncOnce"

	syncHistoryStatusAll       = "all"
	syncHistoryStatusPending   = "Pending"
	syncHistoryStatusRunning   = "Running"
	syncHistoryStatusCompleted = "Completed"
	syncHistoryStatusFailed    = "Failed"
	syncHistoryStatusUnknown   = "Unknown"
)

type syncHistoryItemWithSort struct {
	SyncHistoryItemDTO
	creationTimestamp *matev1.Time
}

func NewInstanceHandler(kc *kube.KubeClient, rg *route.RouterGroup, mw ...app.HandlerFunc) *InstanceHandler {
	return &InstanceHandler{
		KubeClient:           kc,
		Rg:                   rg,
		Mw:                   mw,
		DisasterConfigLister: kc.InformerFactory.Disaster().V1().DisasterConfigs().Lister(),
		InstanceLister:       kc.InformerFactory.Disaster().V1().DisasterInstances().Lister(),
	}
}

// Helper to find namespace for an instance name
func (h *InstanceHandler) findNamespace(c context.Context, name string) (string, error) {
	// Optimization: First check the configured management namespace.
	_, err := h.DisasterClient.DisasterV1().DisasterInstances(common.DisasterSystemNamespace).Get(c, name, matev1.GetOptions{})
	if err == nil {
		return common.DisasterSystemNamespace, nil
	}

	// Optimistically look in all namespaces
	list, err := h.DisasterClient.DisasterV1().DisasterInstances("").List(c, matev1.ListOptions{
		FieldSelector: "metadata.name=" + name,
	})
	if err != nil {
		return "", err
	}
	if len(list.Items) == 0 {
		return "", errors.NewNotFound(dapisv1.Resource("disasterinstance"), name)
	}
	// Return the namespace of the first match (assuming uniqueness or taking first)
	return list.Items[0].Namespace, nil
}

func isLaterOperation(candidate, current *dapisv1.DisasterOperation) bool {
	if candidate == nil {
		return false
	}
	if current == nil {
		return true
	}
	if candidate.CreationTimestamp.After(current.CreationTimestamp.Time) {
		return true
	}
	if current.CreationTimestamp.After(candidate.CreationTimestamp.Time) {
		return false
	}
	return candidate.Name > current.Name
}

func (h *InstanceHandler) getLatestFailoverOperation(c context.Context, namespace, instanceName string) (*dapisv1.DisasterOperation, error) {
	if namespace == "" || instanceName == "" {
		return nil, nil
	}

	list, err := h.DisasterClient.DisasterV1().DisasterOperations(namespace).List(c, matev1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var latest *dapisv1.DisasterOperation
	for i := range list.Items {
		op := &list.Items[i]
		if op.Spec.OperationType != dapisv1.OperationTypeFailover || op.Spec.InstanceName != instanceName {
			continue
		}
		if isLaterOperation(op, latest) {
			latest = op.DeepCopy()
		}
	}

	return latest, nil
}

func (h *InstanceHandler) enrichListAutoCancel(c context.Context, dtos []DisasterInstanceDTO) {
	type nsBucket struct {
		indexesByInstance map[string][]int
	}

	buckets := make(map[string]*nsBucket)
	for i := range dtos {
		dto := &dtos[i]
		if dto.Namespace == "" || dto.Name == "" {
			continue
		}
		bucket := buckets[dto.Namespace]
		if bucket == nil {
			bucket = &nsBucket{indexesByInstance: make(map[string][]int)}
			buckets[dto.Namespace] = bucket
		}
		bucket.indexesByInstance[dto.Name] = append(bucket.indexesByInstance[dto.Name], i)
	}

	for namespace, bucket := range buckets {
		list, err := h.DisasterClient.DisasterV1().DisasterOperations(namespace).List(c, matev1.ListOptions{})
		if err != nil {
			continue
		}

		latestByInstance := make(map[string]*dapisv1.DisasterOperation)
		for i := range list.Items {
			op := &list.Items[i]
			if op.Spec.OperationType != dapisv1.OperationTypeFailover {
				continue
			}
			if _, ok := bucket.indexesByInstance[op.Spec.InstanceName]; !ok {
				continue
			}
			if isLaterOperation(op, latestByInstance[op.Spec.InstanceName]) {
				latestByInstance[op.Spec.InstanceName] = op.DeepCopy()
			}
		}

		for instanceName, indexes := range bucket.indexesByInstance {
			summary := ConvertToAutoCancelSummary(latestByInstance[instanceName])
			for _, idx := range indexes {
				dtos[idx].AutoCancel = summary
			}
		}
	}
}

// 1. List Instances
func (h *InstanceHandler) listInstances(c context.Context, ctx *app.RequestContext) {
	qParams := transport.ParseOptions(c, ctx)

	protectedNamespace := strings.TrimSpace(qParams.Filters["namespace"])
	delete(qParams.Filters, "namespace")
	if protectedNamespace == "*" {
		protectedNamespace = ""
	}

	qParams.Sort = strings.TrimSpace(qParams.Sort)
	qParams.Order = strings.ToLower(strings.TrimSpace(qParams.Order))

	// Default sort to keep list deterministic for UI consumers
	if qParams.Sort == "" {
		qParams.Sort = "creationTimestamp"
		qParams.Order = "desc"
	} else if qParams.Order == "" {
		qParams.Order = "asc"
	}

	// List CRs
	list, err := h.DisasterClient.DisasterV1().DisasterInstances(common.DisasterSystemNamespace).List(c, matev1.ListOptions{})
	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	// Filter & Sort (Memory)
	// Note: For simplicity, we filter the raw list first. In production, use Informer/Lister.
	items := make([]*dapisv1.DisasterInstance, 0, len(list.Items))
	for i := range list.Items {
		items = append(items, &list.Items[i])
	}

	// 内存模糊过滤（labels）
	filteredItems := make([]*dapisv1.DisasterInstance, 0, len(items))
	for _, item := range items {
		if !instanceProtectsNamespace(item, protectedNamespace) {
			continue
		}
		match := true
		for k, v := range qParams.Filters {
			actual := item.Labels[k]
			if !transport.MatchFuzzy(actual, v) {
				match = false
				break
			}
		}
		if match {
			filteredItems = append(filteredItems, item)
		}
	}

	// 内存关键字过滤（name / namespace / labels values）
	if qParams.Keyword != "" {
		var matched []*dapisv1.DisasterInstance
		keyword := qParams.Keyword
		for _, item := range filteredItems {
			if strings.Contains(item.Name, keyword) || strings.Contains(item.Namespace, keyword) {
				matched = append(matched, item)
				continue
			}
			for _, v := range item.Labels {
				if strings.Contains(v, keyword) {
					matched = append(matched, item)
					break
				}
			}
		}
		filteredItems = matched
	}

	// 内存排序逻辑（稳定排序）
	sortedItems := transport.Sort(filteredItems, qParams, func(a, b *dapisv1.DisasterInstance, field string) int {
		switch field {
		case "name":
			if nameCmp := strings.Compare(a.Name, b.Name); nameCmp != 0 {
				return nameCmp
			}
			if nsCmp := strings.Compare(a.Namespace, b.Namespace); nsCmp != 0 {
				return nsCmp
			}
			if a.CreationTimestamp.Before(&b.CreationTimestamp) {
				return -1
			}
			if a.CreationTimestamp.After(b.CreationTimestamp.Time) {
				return 1
			}
			return 0
		case "namespace":
			if nsCmp := strings.Compare(a.Namespace, b.Namespace); nsCmp != 0 {
				return nsCmp
			}
			return strings.Compare(a.Name, b.Name)
		case "creationTimestamp":
			if a.CreationTimestamp.Before(&b.CreationTimestamp) {
				return -1
			}
			if a.CreationTimestamp.After(b.CreationTimestamp.Time) {
				return 1
			}
			if nsCmp := strings.Compare(a.Namespace, b.Namespace); nsCmp != 0 {
				return nsCmp
			}
			return strings.Compare(a.Name, b.Name)
		default:
			return 0
		}
	})

	summary := summarizeDisasterInstanceList(sortedItems)

	// Aggregate Configs
	// Optimization: Fetch all unique configs in one pass per name, or cache them.
	// For now, we fetch per item (N+1), optimizable later.

	// Pagination
	pagedItems, total := transport.Paginate(sortedItems, qParams)

	dtos := make([]DisasterInstanceDTO, 0, len(pagedItems))
	for _, item := range pagedItems {
		var config *dapisv1.DisasterConfig
		if item.Spec.Config != "" {
			// Ignore error if config not found, just return partial data
			cfg, _ := h.DisasterClient.DisasterV1().DisasterConfigs().Get(c, item.Spec.Config, matev1.GetOptions{})
			config = cfg
		}
		dtos = append(dtos, ConvertToDisasterInstanceDTO(item, config, nil))
	}

	// Enrich with Sync Status
	h.enrichListSyncStatus(c, dtos)
	h.enrichListAutoCancel(c, dtos)

	// Build Response
	requestPath := string(ctx.URI().Path())
	data, meta := transport.BuildCollectionResponse(
		requestPath,
		"disasterInstance",
		dtos,
		qParams,
		total,
		nil,
		nil,
	)
	meta.Summary = summary
	transport.WriteSuccess(ctx, consts.StatusOK, data, meta)
}

func summarizeDisasterInstanceList(items []*dapisv1.DisasterInstance) map[string]int {
	protectedCount := 0
	for _, item := range items {
		if item.Status.FsmState == dapisv1.FsmStateProtected {
			protectedCount++
		}
	}
	return map[string]int{
		"protectedCount": protectedCount,
	}
}

func instanceProtectsNamespace(item *dapisv1.DisasterInstance, namespace string) bool {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return true
	}
	if item == nil {
		return false
	}
	for _, protectedNamespace := range item.Spec.Namespaces {
		if transport.MatchFuzzy(strings.TrimSpace(protectedNamespace), namespace) {
			return true
		}
	}
	return false
}

// Helper to enrich list with sync status
func (h *InstanceHandler) enrichListSyncStatus(c context.Context, dtos []DisasterInstanceDTO) {
	for i := range dtos {
		dto := &dtos[i]
		namespace := dto.Namespace
		if namespace == "" {
			continue
		}

		// Fetch DataSync
		if dto.Status.DataSyncName != "" {
			ds, err := h.DisasterClient.DisasterV1().DataSyncs(namespace).Get(c, dto.Status.DataSyncName, matev1.GetOptions{})
			if err == nil {
				reason, message := resolveCurrentSyncError(string(ds.Status.State), ds.Status.Reason, ds.Status.Message, ds.Status.Conditions)
				dto.DataSyncStatus = &SyncSummaryDTO{
					State:        string(ds.Status.State),
					Reason:       reason,
					Message:      message,
					LastSyncTime: common.NewLocalTimePtr(ds.Status.LastSyncTime),
					Paused:       ds.Spec.Paused,
				}
			}
		}

		// Fetch ResourceSync
		if dto.Status.ResourceSyncName != "" {
			rs, err := h.DisasterClient.DisasterV1().ResourceSyncs(namespace).Get(c, dto.Status.ResourceSyncName, matev1.GetOptions{})
			if err == nil {
				reason, message := resolveCurrentSyncError(string(rs.Status.State), rs.Status.Reason, rs.Status.Message, rs.Status.Conditions)
				dto.ResourceSyncStatus = &SyncSummaryDTO{
					State:        string(rs.Status.State),
					Reason:       reason,
					Message:      message,
					LastSyncTime: common.NewLocalTimePtr(rs.Status.LastSyncTime),
					Paused:       rs.Spec.Paused,
				}
			}
		}
	}
}

// 2. Create Instance
func (h *InstanceHandler) createInstance(c context.Context, ctx *app.RequestContext) {
	if err := rejectUnsupportedSyncPolicyField(ctx.Request.Body()); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	var req CreateDisasterInstanceRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	spec, err := req.ToCRD()
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	if err := velerohooks.ValidateDisasterVeleroHooks(spec.VeleroHooks, "veleroHooks"); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), velerohooks.ErrorMeta(err))
		return
	}

	// Validate Config exists
	config, err := h.DisasterClient.DisasterV1().DisasterConfigs().Get(c, req.Config, matev1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeBadRequest, fmt.Sprintf("Config %s not found", req.Config), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}
	if err := h.validateProtectedNamespaces(config.Spec.SourceCluster, spec.Namespaces, "", ""); err != nil {
		if conflictErr, ok := err.(*protectedNamespaceConflictError); ok {
			transport.WriteError(ctx, transport.CodeConflict, conflictErr.Error(), conflictErr.meta)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	if err := h.prepareRestorePolicyForPersist(c, &spec, nil); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	// Use provided namespace or default
	ns := req.Namespace
	if ns == "" {
		ns = common.DisasterSystemNamespace
	}

	body := dapisv1.DisasterInstance{
		ObjectMeta: matev1.ObjectMeta{
			Name:      req.Name,
			Namespace: ns,
			Labels: map[string]string{
				"testudo.softcdata.com/config": req.Config,
			},
			Annotations: map[string]string{
				"testudo.softcdata.com/description": req.Description,
			},
		},
		Spec: spec,
	}
	// Default values
	if body.Spec.PodRestoreMethod == "" {
		body.Spec.PodRestoreMethod = "replica"
	}

	transport.SetTraceAnnotation(&body.ObjectMeta, ctx, metadata.AnnotationTraceID)

	rc, err := h.DisasterClient.DisasterV1().DisasterInstances(ns).Create(c, &body, matev1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			transport.WriteError(ctx, transport.CodeConflict, err.Error(), nil)
			return
		}
		if errors.IsInvalid(err) || errors.IsBadRequest(err) {
			transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
			return
		}
		if errors.IsForbidden(err) {
			transport.WriteError(ctx, transport.CodeForbidden, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	// Return DTO
	dto := ConvertToDisasterInstanceDTO(rc, config, nil)
	transport.WriteSuccess(ctx, consts.StatusCreated, dto, nil)
}

// 3. Get Instance
func (h *InstanceHandler) getInstance(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")

	// Determine Namespace
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

	item, err := h.DisasterClient.DisasterV1().DisasterInstances(namespace).Get(c, name, matev1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	var config *dapisv1.DisasterConfig
	var storage *dapisv1.StorageRepository
	if item.Spec.Config != "" {
		config, _ = h.DisasterClient.DisasterV1().DisasterConfigs().Get(c, item.Spec.Config, matev1.GetOptions{})
		if config != nil && config.Spec.StorageRepository != "" {
			storage, _ = h.DisasterClient.DisasterV1().StorageRepositories(common.DisasterSystemNamespace).Get(c, config.Spec.StorageRepository, matev1.GetOptions{})
		}
	}
	dto := ConvertToDisasterInstanceDTO(item, config, storage)
	if latestOp, err := h.getLatestFailoverOperation(c, namespace, name); err == nil {
		dto.AutoCancel = ConvertToAutoCancelSummary(latestOp)
	}
	transport.WriteSuccess(ctx, consts.StatusOK, dto, nil)
}

// 3.1 Get Instance Group Membership
func (h *InstanceHandler) getInstanceGroups(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")

	// Namespace resolution
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

	// Ensure instance exists in resolved namespace.
	if _, err := h.DisasterClient.DisasterV1().DisasterInstances(namespace).Get(c, name, matev1.GetOptions{}); err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	groups, err := h.findContainingGroups(c, namespace, name)
	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	transport.WriteSuccess(ctx, consts.StatusOK, InstanceGroupsDTO{
		Instance:  name,
		Namespace: namespace,
		InGroup:   len(groups) > 0,
		Groups:    groups,
	}, nil)
}

// 3.2 Validate Target
// 供前端在触发实例操作前主动校验目标可执行性。
func (h *InstanceHandler) validateTarget(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")
	operation := strings.ToLower(strings.TrimSpace(string(ctx.Query("operation"))))

	// Namespace resolution
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

	instance, err := h.DisasterClient.DisasterV1().DisasterInstances(namespace).Get(c, name, matev1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	groups, err := h.findContainingGroups(c, namespace, name)
	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	available, valid, reason, message := validateInstanceOperationAllowed(instance, operation)

	transport.WriteSuccess(ctx, consts.StatusOK, ValidateTargetDTO{
		TargetName:          name,
		Namespace:           namespace,
		Operation:           operation,
		Valid:               valid,
		Reason:              reason,
		Message:             message,
		FsmState:            instance.Status.FsmState,
		AvailableOperations: available,
		InGroup:             len(groups) > 0,
		Groups:              groups,
	}, nil)
}

func (h *InstanceHandler) validateRestoreClasses(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")

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

	req := ValidateRestoreClassesRequest{}
	if len(ctx.Request.Body()) > 0 {
		if err := ctx.BindJSON(&req); err != nil {
			transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
			return
		}
	}

	instance, err := h.DisasterClient.DisasterV1().DisasterInstances(namespace).Get(c, name, matev1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	targetCluster, err := h.resolveRestoreClassValidationTarget(c, instance, strings.TrimSpace(req.TargetCluster))
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	storagePolicy := req.StorageClassMapping
	ingressPolicy := req.IngressClassMapping
	if storagePolicy == nil && ingressPolicy == nil {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationClassMappingRequired, nil, nil)
		return
	}

	targetCli, err := h.getClusterClient(c, targetCluster)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, fmt.Sprintf("failed to get client for cluster %s: %v", targetCluster, err), nil)
		return
	}

	result := ValidateRestoreClassesDTO{
		InstanceName:  name,
		Namespace:     namespace,
		TargetCluster: targetCluster,
		Valid:         true,
	}

	if err := applyStorageClassCheck(c, targetCli, storagePolicy, &result); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	if err := applyIngressClassCheck(c, targetCli, ingressPolicy, &result); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	transport.WriteSuccess(ctx, consts.StatusOK, result, nil)
}

func (h *InstanceHandler) resolveRestoreClassValidationTarget(c context.Context, instance *dapisv1.DisasterInstance, requestedTarget string) (string, error) {
	if requestedTarget != "" {
		return requestedTarget, nil
	}

	if instance != nil && strings.TrimSpace(instance.Status.SecondaryCluster) != "" {
		return strings.TrimSpace(instance.Status.SecondaryCluster), nil
	}

	if instance == nil || strings.TrimSpace(instance.Spec.Config) == "" {
		return "", fmt.Errorf("targetCluster is required")
	}

	cfg, err := h.DisasterClient.DisasterV1().DisasterConfigs().Get(c, instance.Spec.Config, matev1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return "", fmt.Errorf("config %s not found", instance.Spec.Config)
		}
		return "", fmt.Errorf("get config %s failed: %w", instance.Spec.Config, err)
	}

	targetCluster := strings.TrimSpace(cfg.Spec.TargetCluster)
	if targetCluster == "" {
		return "", fmt.Errorf("targetCluster is required")
	}
	return targetCluster, nil
}

func (h *InstanceHandler) getClusterClient(ctx context.Context, clusterName string) (ctrclient.Client, error) {
	if h.GetClusterClient != nil {
		return h.GetClusterClient(ctx, clusterName)
	}
	if h.KubeClient == nil || h.KubeClient.ClusterClient == nil {
		return nil, fmt.Errorf("cluster client is not initialized")
	}
	return h.KubeClient.GetKubeClient(ctx, h.KubeClient.RuntimeClient(), h.KubeClient.Scheme(), clusterName)
}

func (h *InstanceHandler) resolveInstanceSourceCluster(c context.Context, configName string) (string, error) {
	configName = strings.TrimSpace(configName)
	if configName == "" {
		return "", fmt.Errorf("instance.spec.config is empty")
	}
	cfg, err := h.DisasterClient.DisasterV1().DisasterConfigs().Get(c, configName, matev1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return "", fmt.Errorf("config %s not found", configName)
		}
		return "", err
	}
	sourceCluster := strings.TrimSpace(cfg.Spec.SourceCluster)
	if sourceCluster == "" {
		return "", fmt.Errorf("config %s has empty spec.sourceCluster", configName)
	}
	return sourceCluster, nil
}

func (h *InstanceHandler) shouldRunLiveModifierRuleValidation() bool {
	return h != nil && h.KubeClient != nil && h.KubeClient.ClusterClient != nil
}

func (h *InstanceHandler) getClusterRESTConfig(ctx context.Context, clusterName string) (*rest.Config, error) {
	if h != nil && h.GetClusterRESTConfigFunc != nil {
		return h.GetClusterRESTConfigFunc(ctx, clusterName)
	}
	clusterName = strings.TrimSpace(clusterName)
	if clusterName == "" {
		return nil, fmt.Errorf("clusterName is required")
	}
	if h.KubeClient == nil || h.KubeClient.ClusterClient == nil {
		return nil, fmt.Errorf("cluster client is not initialized")
	}
	runtimeClient := h.KubeClient.RuntimeClient()
	if runtimeClient == nil {
		return nil, fmt.Errorf("runtime client is not initialized")
	}
	cluster := &dapisv1.Cluster{}
	if err := runtimeClient.Get(ctx, types.NamespacedName{Name: clusterName}, cluster); err != nil {
		return nil, fmt.Errorf("failed to get Cluster %s: %w", clusterName, err)
	}

	if len(cluster.Spec.KubeConfig) > 0 {
		cfg, err := clientcmd.RESTConfigFromKubeConfig(cluster.Spec.KubeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to parse kubeconfig for Cluster %s: %w", clusterName, err)
		}
		cfg.QPS = 100
		cfg.Burst = 200
		return cfg, nil
	}

	if cluster.Spec.Token != "" && cluster.Spec.Endpoint != "" {
		token := cluster.Spec.Token
		if !strings.HasPrefix(token, "eyJ") {
			if decoded, decErr := base64.StdEncoding.DecodeString(token); decErr == nil {
				token = string(decoded)
			}
		}
		return &rest.Config{
			Host:            cluster.Spec.Endpoint,
			BearerToken:     token,
			TLSClientConfig: rest.TLSClientConfig{Insecure: true},
			QPS:             100,
			Burst:           200,
		}, nil
	}

	return nil, fmt.Errorf("Cluster %s has no kubeconfig or token/endpoint", clusterName)
}

func applyStorageClassCheck(ctx context.Context, targetCli ctrclient.Client, p *RestoreClassMappingPolicy, result *ValidateRestoreClassesDTO) error {
	targets, strict, err := collectClassTargets(p)
	if err != nil {
		return err
	}
	result.StorageClassCheck = RestoreClassCheckDTO{
		Enabled:                p != nil,
		StrictTargetValidation: strict,
		CheckedTargets:         targets,
	}
	if len(targets) == 0 {
		return nil
	}

	list := &storagev1.StorageClassList{}
	if err := targetCli.List(ctx, list); err != nil {
		return fmt.Errorf("failed to list storageclasses: %w", err)
	}
	available := make(map[string]struct{}, len(list.Items))
	for _, item := range list.Items {
		available[item.Name] = struct{}{}
	}
	result.StorageClassCheck.MissingTargets = collectMissingTargets(targets, available)
	if len(result.StorageClassCheck.MissingTargets) > 0 && strict && result.Valid {
		result.Valid = false
		result.Code = "StorageClassTargetNotFound"
		result.Message = fmt.Sprintf(
			"missing StorageClass in target cluster %s: %s",
			result.TargetCluster,
			strings.Join(result.StorageClassCheck.MissingTargets, ","),
		)
	}
	return nil
}

func applyIngressClassCheck(ctx context.Context, targetCli ctrclient.Client, p *RestoreClassMappingPolicy, result *ValidateRestoreClassesDTO) error {
	targets, strict, err := collectClassTargets(p)
	if err != nil {
		return err
	}
	result.IngressClassCheck = RestoreClassCheckDTO{
		Enabled:                p != nil,
		StrictTargetValidation: strict,
		CheckedTargets:         targets,
	}
	if len(targets) == 0 {
		return nil
	}

	list := &networkingv1.IngressClassList{}
	if err := targetCli.List(ctx, list); err != nil {
		return fmt.Errorf("failed to list ingressclasses: %w", err)
	}
	available := make(map[string]struct{}, len(list.Items))
	for _, item := range list.Items {
		available[item.Name] = struct{}{}
	}
	result.IngressClassCheck.MissingTargets = collectMissingTargets(targets, available)
	if len(result.IngressClassCheck.MissingTargets) > 0 && strict && result.Valid {
		result.Valid = false
		result.Code = "IngressClassTargetNotFound"
		result.Message = fmt.Sprintf(
			"missing IngressClass in target cluster %s: %s",
			result.TargetCluster,
			strings.Join(result.IngressClassCheck.MissingTargets, ","),
		)
	}
	return nil
}

func collectClassTargets(p *RestoreClassMappingPolicy) ([]string, bool, error) {
	if p == nil {
		return nil, false, nil
	}
	if p.UnmatchedPolicy != "" &&
		p.UnmatchedPolicy != RestoreClassUnmatchedPolicyKeep &&
		p.UnmatchedPolicy != RestoreClassUnmatchedPolicyFail {
		return nil, p.StrictTargetValidation, fmt.Errorf("ClassMappingInvalid: invalid unmatchedPolicy=%s", p.UnmatchedPolicy)
	}
	if p.UnmatchedPolicy == RestoreClassUnmatchedPolicyFail && len(p.Mappings) == 0 {
		return nil, p.StrictTargetValidation, fmt.Errorf("ClassMappingInvalid: mappings is required when unmatchedPolicy=Fail")
	}

	seen := make(map[string]string, len(p.Mappings))
	targetSet := make(map[string]struct{}, len(p.Mappings))
	for _, m := range p.Mappings {
		source := strings.TrimSpace(m.SourceClass)
		target := strings.TrimSpace(m.TargetClass)
		if source == "" || target == "" {
			return nil, p.StrictTargetValidation, fmt.Errorf("ClassMappingInvalid: sourceClass and targetClass are required")
		}
		if prev, ok := seen[source]; ok && prev != target {
			return nil, p.StrictTargetValidation, fmt.Errorf("ClassMappingInvalid: duplicate sourceClass=%s maps to multiple targets", source)
		}
		seen[source] = target
		targetSet[target] = struct{}{}
	}
	targets := make([]string, 0, len(targetSet))
	for target := range targetSet {
		targets = append(targets, target)
	}
	sort.Strings(targets)
	return targets, p.StrictTargetValidation, nil
}

func collectMissingTargets(targets []string, available map[string]struct{}) []string {
	missing := make([]string, 0)
	for _, target := range targets {
		if _, ok := available[target]; !ok {
			missing = append(missing, target)
		}
	}
	sort.Strings(missing)
	return missing
}

func (h *InstanceHandler) findContainingGroups(c context.Context, namespace, instanceName string) ([]string, error) {
	groups, err := h.DisasterClient.DisasterV1().DisasterGroups(namespace).List(c, matev1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]string, 0)
	seen := make(map[string]struct{}, len(groups.Items))
	for i := range groups.Items {
		group := &groups.Items[i]
		if !groupContainsInstance(group, instanceName) {
			continue
		}
		if _, ok := seen[group.Name]; ok {
			continue
		}
		seen[group.Name] = struct{}{}
		result = append(result, group.Name)
	}
	sort.Strings(result)
	return result, nil
}

func normalizeOperation(operation string) string {
	op := strings.ToLower(strings.TrimSpace(operation))
	switch op {
	case "sync-data":
		return "syncdata"
	case "sync-resource":
		return "syncresource"
	default:
		return op
	}
}

func containsOperation(available []string, operation string) bool {
	if len(available) == 0 {
		return false
	}
	want := normalizeOperation(operation)
	for _, op := range available {
		if normalizeOperation(op) == want {
			return true
		}
	}
	return false
}

func appendOperation(available []string, operation string) []string {
	for _, op := range available {
		if normalizeOperation(op) == normalizeOperation(operation) {
			return available
		}
	}
	return append(available, operation)
}

func isRecoverableSyncFailureReason(reason string) bool {
	switch strings.TrimSpace(reason) {
	case "DataSyncFailed",
		"ResourceSyncFailed",
		"InitializationFailed",
		"BackupFailed",
		"BuildRestoreSpecFailed",
		"RestoreFailed",
		"DependencyFailed",
		"StorageUnavailable":
		return true
	default:
		return false
	}
}

func effectiveAvailableOperations(instance *dapisv1.DisasterInstance) []string {
	if instance == nil {
		return nil
	}

	available := append([]string(nil), instance.Status.AvailableOperations...)
	if instance.Status.FsmState == dapisv1.FsmStateFailingOver {
		return appendOperation(available, "cancel")
	}
	if instance.Status.FsmState != dapisv1.FsmStateFailed || !isRecoverableSyncFailureReason(instance.Status.Reason) {
		return available
	}

	switch strings.TrimSpace(instance.Status.Reason) {
	case "DataSyncFailed":
		return appendOperation(available, "syncdata")
	case "ResourceSyncFailed":
		return appendOperation(available, "syncresource")
	default:
		available = appendOperation(available, "syncdata")
		return appendOperation(available, "syncresource")
	}
}

func validateInstanceOperationAllowed(instance *dapisv1.DisasterInstance, operation string) ([]string, bool, string, string) {
	available := effectiveAvailableOperations(instance)
	operation = normalizeOperation(operation)
	if operation == "" || containsOperation(available, operation) {
		return available, true, "", ""
	}
	return available, false, "OperationNotAllowed", fmt.Sprintf("operation %s is not allowed in current state %s", operation, instance.Status.FsmState)
}

func groupContainsInstance(group *dapisv1.DisasterGroup, instanceName string) bool {
	for _, level := range group.Spec.Levels {
		for _, name := range level {
			if name == instanceName {
				return true
			}
		}
	}
	return false
}

// 4. Update Instance
func (h *InstanceHandler) updateInstance(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")
	if err := rejectUnsupportedSyncPolicyField(ctx.Request.Body()); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	var req UpdateDisasterInstanceRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	restorePolicy, err := req.ResolveRestorePolicy()
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	// Namespace resolution
	ns := ctx.QueryArgs().Peek("namespace")
	namespace := string(ns)
	if namespace == "" {
		var err error
		namespace, err = h.findNamespace(c, name)
		if err != nil {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
	}
	var result *dapisv1.DisasterInstance
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		existing, err := h.DisasterClient.DisasterV1().DisasterInstances(namespace).Get(c, name, matev1.GetOptions{})
		if err != nil {
			return err
		}
		previous := existing.DeepCopy()

		if req.Namespaces != nil {
			existing.Spec.Namespaces = req.Namespaces
		}
		if req.LabelSelector != nil {
			existing.Spec.LabelSelector = req.LabelSelector
		}
		if req.PodRestoreMethod != nil {
			existing.Spec.PodRestoreMethod = *req.PodRestoreMethod
		}
		req.applyPolicyFields(&existing.Spec)
		if req.SkipPodReadyCheck != nil {
			existing.Spec.SkipPodReadyCheck = req.SkipPodReadyCheck
		}
		req.ApplyVeleroHooksPatch(&existing.Spec)
		if err := velerohooks.ValidateDisasterVeleroHooks(existing.Spec.VeleroHooks, "veleroHooks"); err != nil {
			return err
		}
		sourceCluster, err := h.resolveProtectedNamespaceSourceCluster(existing.Spec.Config)
		if err != nil {
			return err
		}
		if err := h.validateProtectedNamespaces(sourceCluster, existing.Spec.Namespaces, existing.Namespace, existing.Name); err != nil {
			return err
		}
		if restorePolicy != nil {
			existing.Spec.RestorePolicy = mergeRestorePolicyForUpdate(existing.Spec.RestorePolicy, req.RestorePolicy, restorePolicy)
		}
		if existing.Spec.RestorePolicy != nil && (restorePolicy != nil || req.Namespaces != nil || req.LabelSelector != nil) {
			if err := h.prepareRestorePolicyForPersist(c, &existing.Spec, previous); err != nil {
				return err
			}
		}
		if req.Description != nil {
			if existing.Annotations == nil {
				existing.Annotations = make(map[string]string)
			}
			existing.Annotations["testudo.softcdata.com/description"] = *req.Description
		}

		transport.SetTraceAnnotation(&existing.ObjectMeta, ctx, metadata.AnnotationTraceID)
		result, err = h.DisasterClient.DisasterV1().DisasterInstances(namespace).Update(c, existing, matev1.UpdateOptions{})
		return err
	})

	if err != nil {
		if conflictErr, ok := err.(*protectedNamespaceConflictError); ok {
			transport.WriteError(ctx, transport.CodeConflict, conflictErr.Error(), conflictErr.meta)
			return
		}
		if validationErr, ok := err.(*protectedNamespaceValidationError); ok {
			transport.WriteError(ctx, transport.CodeBadRequest, validationErr.Error(), nil)
			return
		}
		if isModifierRuleValidationError(err) {
			transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
			return
		}
		if _, ok := err.(*velerohooks.ValidationError); ok {
			transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), velerohooks.ErrorMeta(err))
			return
		}
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		if errors.IsInvalid(err) || errors.IsBadRequest(err) {
			transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
			return
		}
		if errors.IsForbidden(err) {
			transport.WriteError(ctx, transport.CodeForbidden, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	// Fetch config for DTO
	var config *dapisv1.DisasterConfig
	if result.Spec.Config != "" {
		config, _ = h.DisasterClient.DisasterV1().DisasterConfigs().Get(c, result.Spec.Config, matev1.GetOptions{})
	}
	transport.WriteSuccess(ctx, consts.StatusOK, ConvertToDisasterInstanceDTO(result, config, nil), nil)
}

// 5. Delete Instance
func (h *InstanceHandler) deleteInstance(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")

	// Namespace resolution
	ns := ctx.QueryArgs().Peek("namespace")
	namespace := string(ns)
	if namespace == "" {
		var err error
		namespace, err = h.findNamespace(c, name)
		if err != nil {
			// If not found, maybe already deleted
			transport.WriteSuccess(ctx, consts.StatusOK, utils.H{"name": name}, nil)
			return
		}
	}

	err := h.DisasterClient.DisasterV1().DisasterInstances(namespace).Delete(c, name, matev1.DeleteOptions{})
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

// 6. Get Sync Status
func (h *InstanceHandler) getSyncStatus(c context.Context, ctx *app.RequestContext) {
	start := time.Now()
	name := ctx.Param("name")
	hlog.Infof("Start getSyncStatus for %s", name)
	defer func() {
		hlog.Infof("End getSyncStatus for %s, total duration: %v", name, time.Since(start))
	}()

	// Namespace resolution for Instance
	ns := ctx.QueryArgs().Peek("namespace")
	namespace := string(ns)
	if namespace == "" {
		var err error
		findStart := time.Now()
		namespace, err = h.findNamespace(c, name)
		hlog.Infof("findNamespace took %v", time.Since(findStart))
		if err != nil {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
	}

	instStart := time.Now()
	instance, err := h.DisasterClient.DisasterV1().DisasterInstances(namespace).Get(c, name, matev1.GetOptions{})
	hlog.Infof("getInstance took %v", time.Since(instStart))
	if err != nil {
		transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
		return
	}

	status := SyncStatusDTO{}

	// Helper to enrich status from AppRestore (Priority) or AppBackup
	// Helper to enrich status from AppRestore (Priority) or AppBackup
	// Helper to enrich status from AppRestore (Priority) or AppBackup
	// Helper to enrich status from History (DataSync/ResourceSync Status.History)
	enrichStatus := func(dto *SubResourceStatusDTO, history []dapisv1.SyncHistoryRecord, lastBackupName string, parentUID types.UID) {
		enrichStart := time.Now()
		defer func() {
			hlog.Infof("enrichStatus for %s took %v", dto.Name, time.Since(enrichStart))
		}()

		dto.LastSyncStatus = convertLastSyncStatus(latestSyncHistoryRecord(history))

		// 1. Find matching record in History
		var targetRecord *dapisv1.SyncHistoryRecord
		if lastBackupName != "" {
			for _, rec := range history {
				// Assuming History is sorted or we just find the match
				if rec.BackupName == lastBackupName {
					// Need to make a copy or take address of loop var? Range var is by value.
					r := rec
					targetRecord = &r
					break
				}
			}
		} else if len(history) > 0 {
			targetRecord = &history[0] // Assume first is latest?
		}

		if targetRecord != nil {
			dto.ResourceCount = targetRecord.BackupResourceCount
			dto.BackupResourceCount = targetRecord.BackupResourceCount
			dto.RestoreResourceCount = targetRecord.RestoreResourceCount
			dto.Duration = targetRecord.Duration

			// Note: SyncHistoryRecord currently doesn't have ErrorCount,
			// so we rely on Status or logs for details.
			// DTO FailureCount will be 0 unless sourced elsewhere.
		}

		// 2. Fetch Historical Stats from BackupRestoreStatistics CRD
		// Now using Parent (RS/DS) specific stats directly
		labelSelector := fmt.Sprintf("disaster.io/scope-uid=%s", parentUID)
		statsList, err := h.DisasterClient.DisasterV1().BackupRestoreStatisticses(namespace).List(c, matev1.ListOptions{
			LabelSelector:   labelSelector,
			ResourceVersion: "0",
		})

		if err == nil && len(statsList.Items) > 0 {
			for _, stat := range statsList.Items {
				// Double check owner match (although UID match is strong enough)
				if stat.Spec.ScopeRef.UID == parentUID {
					dto.SyncSuccessCount += int(stat.Status.Statistics.Completed)
					dto.SyncFailureCount += int(stat.Status.Statistics.Failed)
				}
			}
		}

		// failureCount 对齐为失败同步次数；若统计尚未落盘但当前处于 Failed，至少回显 1。
		dto.FailureCount = dto.SyncFailureCount
		if dto.FailureCount == 0 && strings.EqualFold(strings.TrimSpace(dto.Status), "Failed") {
			dto.FailureCount = 1
		}
	}

	// DataSync (Usually in same namespace as Instance)
	if instance.Status.DataSyncName != "" {
		ds, err := h.DisasterClient.DisasterV1().DataSyncs(namespace).Get(c, instance.Status.DataSyncName, matev1.GetOptions{})
		if err == nil {
			reason, message := resolveCurrentSyncError(string(ds.Status.State), ds.Status.Reason, ds.Status.Message, ds.Status.Conditions)
			dto := &SubResourceStatusDTO{
				Name:           ds.Name,
				Status:         string(ds.Status.State),
				Reason:         reason,
				Message:        message,
				Paused:         ds.Spec.Paused,
				LastBackupName: ds.Status.LastBackupName,
				LastTime:       common.NewLocalTimePtr(ds.Status.LastSyncTime),
			}
			enrichStatus(dto, ds.Status.History, ds.Status.LastBackupName, ds.UID)
			status.DataSync = dto
		}
	}

	// ResourceSync (Same NS)
	if instance.Status.ResourceSyncName != "" {
		rs, err := h.DisasterClient.DisasterV1().ResourceSyncs(namespace).Get(c, instance.Status.ResourceSyncName, matev1.GetOptions{})
		if err == nil {
			reason, message := resolveCurrentSyncError(string(rs.Status.State), rs.Status.Reason, rs.Status.Message, rs.Status.Conditions)
			dto := &SubResourceStatusDTO{
				Name:           rs.Name,
				Status:         string(rs.Status.State),
				Reason:         reason,
				Message:        message,
				Paused:         rs.Spec.Paused,
				LastBackupName: rs.Status.LastBackupName,
				LastTime:       common.NewLocalTimePtr(rs.Status.LastSyncTime),
			}
			enrichStatus(dto, rs.Status.History, rs.Status.LastBackupName, rs.UID)
			status.ResourceSync = dto
		}
	}

	transport.WriteSuccess(ctx, consts.StatusOK, status, nil)
}

func resolveCurrentSyncError(state, reason, message string, conditions []matev1.Condition) (string, string) {
	if !strings.EqualFold(strings.TrimSpace(state), "Failed") {
		return "", ""
	}

	trimmedReason := strings.TrimSpace(reason)
	trimmedMessage := strings.TrimSpace(message)
	if trimmedReason != "" || trimmedMessage != "" {
		return trimmedReason, trimmedMessage
	}

	var latest *matev1.Condition
	for i := range conditions {
		cond := conditions[i]
		if cond.Status != matev1.ConditionTrue {
			continue
		}
		if strings.TrimSpace(cond.Reason) == "" && strings.TrimSpace(cond.Message) == "" {
			continue
		}
		if latest == nil || cond.LastTransitionTime.Time.After(latest.LastTransitionTime.Time) {
			c := cond
			latest = &c
		}
	}
	if latest != nil {
		trimmedReason = strings.TrimSpace(latest.Reason)
		trimmedMessage = strings.TrimSpace(latest.Message)
	}
	if trimmedReason == "" {
		trimmedReason = "SyncFailed"
	}
	if trimmedMessage == "" {
		trimmedMessage = "sync sub-resource is in Failed state"
	}
	return trimmedReason, trimmedMessage
}

func latestSyncHistoryRecord(history []dapisv1.SyncHistoryRecord) *dapisv1.SyncHistoryRecord {
	if len(history) == 0 {
		return nil
	}
	selected := -1
	for i := range history {
		if history[i].CompletionTime == nil {
			continue
		}
		if selected == -1 || history[i].CompletionTime.Time.After(history[selected].CompletionTime.Time) {
			selected = i
		}
	}
	if selected != -1 {
		record := history[selected]
		return &record
	}
	for i := range history {
		if history[i].StartTime == nil {
			continue
		}
		if selected == -1 || history[i].StartTime.Time.After(history[selected].StartTime.Time) {
			selected = i
		}
	}
	if selected != -1 {
		record := history[selected]
		return &record
	}
	record := history[len(history)-1]
	return &record
}

func convertLastSyncStatus(record *dapisv1.SyncHistoryRecord) *LastSyncStatusDTO {
	if record == nil {
		return nil
	}
	status := strings.TrimSpace(record.Status)
	if status == "" {
		status = syncHistoryStatusUnknown
	}
	return &LastSyncStatusDTO{
		Status:               status,
		StartTime:            common.NewLocalTimePtr(record.StartTime),
		CompletionTime:       common.NewLocalTimePtr(record.CompletionTime),
		Duration:             record.Duration,
		BackupName:           record.BackupName,
		RestoreName:          record.RestoreName,
		BackupResourceCount:  record.BackupResourceCount,
		RestoreResourceCount: record.RestoreResourceCount,
		BackupHookStatus:     convertSyncHistoryHookStatus(record.BackupHookStatus),
		RestoreHookStatus:    convertSyncHistoryHookStatus(record.RestoreHookStatus),
	}
}

func convertSyncHistoryHookStatus(status *dapisv1.SyncHistoryHookStatus) *SyncHistoryHookStatusDTO {
	if status == nil {
		return nil
	}
	return &SyncHistoryHookStatusDTO{
		HooksAttempted: status.HooksAttempted,
		HooksFailed:    status.HooksFailed,
	}
}

func normalizeSyncHistoryParam(raw, defaultValue string, allowed map[string]string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		value = defaultValue
	}
	normalized, ok := allowed[strings.ToLower(value)]
	return normalized, ok
}

func syncHistorySourceValues() map[string]string {
	return map[string]string{
		"syncrecord": syncHistorySourceSyncRecord,
		"operation":  syncHistorySourceOperation,
		"all":        syncHistorySourceAll,
	}
}

func syncHistoryTypeValues() map[string]string {
	return map[string]string{
		"all":          syncHistoryTypeAll,
		"datasync":     syncHistoryTypeDataSync,
		"resourcesync": syncHistoryTypeResourceSync,
		"synconce":     syncHistoryTypeSyncOnce,
	}
}

func syncHistoryStatusValues() map[string]string {
	return map[string]string{
		"all":       syncHistoryStatusAll,
		"pending":   syncHistoryStatusPending,
		"running":   syncHistoryStatusRunning,
		"completed": syncHistoryStatusCompleted,
		"failed":    syncHistoryStatusFailed,
		"unknown":   syncHistoryStatusUnknown,
	}
}

func syncHistoryState(state string) string {
	trimmed := strings.TrimSpace(state)
	if trimmed == "" {
		return syncHistoryStatusUnknown
	}
	return trimmed
}

func appendSyncRecordHistory(items *[]syncHistoryItemWithSort, syncType, subResourceName string, history []dapisv1.SyncHistoryRecord) {
	for i := range history {
		record := history[i]
		state := syncHistoryState(record.Status)
		*items = append(*items, syncHistoryItemWithSort{
			SyncHistoryItemDTO: SyncHistoryItemDTO{
				ID:                   fmt.Sprintf("%s:%s:%s:%d", syncHistorySourceSyncRecord, syncType, subResourceName, i),
				SyncType:             syncType,
				Source:               syncHistorySourceSyncRecord,
				Status:               HistoryStatusDTO{State: state},
				StartTime:            common.NewLocalTimePtr(record.StartTime),
				CompletionTime:       common.NewLocalTimePtr(record.CompletionTime),
				Duration:             record.Duration,
				BackupName:           record.BackupName,
				RestoreName:          record.RestoreName,
				BackupResourceCount:  record.BackupResourceCount,
				RestoreResourceCount: record.RestoreResourceCount,
				BackupHookStatus:     convertSyncHistoryHookStatus(record.BackupHookStatus),
				RestoreHookStatus:    convertSyncHistoryHookStatus(record.RestoreHookStatus),
				SubResourceName:      subResourceName,
				HasOperationDetail:   false,
			},
		})
	}
}

func syncHistoryTypeFromOperation(operationType dapisv1.OperationType) (string, bool) {
	switch operationType {
	case dapisv1.OperationTypeSyncData:
		return syncHistoryTypeDataSync, true
	case dapisv1.OperationTypeSyncResource:
		return syncHistoryTypeResourceSync, true
	case dapisv1.OperationTypeSyncOnce:
		return syncHistoryTypeSyncOnce, true
	default:
		return "", false
	}
}

func syncHistoryItemFromOperation(op *dapisv1.DisasterOperation) (syncHistoryItemWithSort, bool) {
	if op == nil {
		return syncHistoryItemWithSort{}, false
	}
	syncType, ok := syncHistoryTypeFromOperation(op.Spec.OperationType)
	if !ok {
		return syncHistoryItemWithSort{}, false
	}
	creationTimestamp := op.CreationTimestamp
	return syncHistoryItemWithSort{
		SyncHistoryItemDTO: SyncHistoryItemDTO{
			ID:                 fmt.Sprintf("%s:%s", syncHistorySourceOperation, op.Name),
			SyncType:           syncType,
			Source:             syncHistorySourceOperation,
			Status:             HistoryStatusDTO{State: syncHistoryState(string(op.Status.State)), Reason: op.Status.Reason, Message: op.Status.Message},
			StartTime:          common.NewLocalTimePtr(op.Status.StartTime),
			CompletionTime:     common.NewLocalTimePtr(op.Status.CompletionTime),
			OperationName:      op.Name,
			OperationUID:       string(op.UID),
			OperationType:      string(op.Spec.OperationType),
			HasOperationDetail: true,
		},
		creationTimestamp: &creationTimestamp,
	}, true
}

func syncHistoryMatches(item SyncHistoryItemDTO, syncType, status string) bool {
	if syncType != syncHistoryTypeAll && item.SyncType != syncType {
		return false
	}
	if status != syncHistoryStatusAll && item.Status.State != status {
		return false
	}
	return true
}

func sortSyncHistoryItems(items []syncHistoryItemWithSort) {
	sort.SliceStable(items, func(i, j int) bool {
		a := items[i]
		b := items[j]
		if a.CompletionTime != nil || b.CompletionTime != nil {
			if a.CompletionTime == nil {
				return false
			}
			if b.CompletionTime == nil {
				return true
			}
			if !a.CompletionTime.Time.Time.Equal(b.CompletionTime.Time.Time) {
				return a.CompletionTime.Time.Time.After(b.CompletionTime.Time.Time)
			}
		}
		if a.StartTime != nil || b.StartTime != nil {
			if a.StartTime == nil {
				return false
			}
			if b.StartTime == nil {
				return true
			}
			if !a.StartTime.Time.Time.Equal(b.StartTime.Time.Time) {
				return a.StartTime.Time.Time.After(b.StartTime.Time.Time)
			}
		}
		if a.creationTimestamp != nil || b.creationTimestamp != nil {
			if a.creationTimestamp == nil {
				return false
			}
			if b.creationTimestamp == nil {
				return true
			}
			if !a.creationTimestamp.Time.Equal(b.creationTimestamp.Time) {
				return a.creationTimestamp.Time.After(b.creationTimestamp.Time)
			}
		}
		return a.ID > b.ID
	})
}

func summarizeSyncHistoryItems(items []SyncHistoryItemDTO) map[string]int {
	summary := map[string]int{
		"totalCount":        len(items),
		"dataSyncCount":     0,
		"resourceSyncCount": 0,
		"completedCount":    0,
		"failedCount":       0,
	}
	for _, item := range items {
		switch item.SyncType {
		case syncHistoryTypeDataSync:
			summary["dataSyncCount"]++
		case syncHistoryTypeResourceSync:
			summary["resourceSyncCount"]++
		}
		switch item.Status.State {
		case syncHistoryStatusCompleted:
			summary["completedCount"]++
		case syncHistoryStatusFailed:
			summary["failedCount"]++
		}
	}
	return summary
}

func (h *InstanceHandler) getSyncHistory(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")
	qParams := transport.ParseOptions(c, ctx)
	source, ok := normalizeSyncHistoryParam(string(ctx.Query("source")), syncHistorySourceSyncRecord, syncHistorySourceValues())
	if !ok {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationInvalidQueryValue, map[string]any{"name": "source", "allowed": "syncRecord, operation, all"}, nil)
		return
	}
	syncType, ok := normalizeSyncHistoryParam(string(ctx.Query("syncType")), syncHistoryTypeAll, syncHistoryTypeValues())
	if !ok {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationInvalidQueryValue, map[string]any{"name": "syncType", "allowed": "all, dataSync, resourceSync, syncOnce"}, nil)
		return
	}
	statusFilter, ok := normalizeSyncHistoryParam(string(ctx.Query("status")), syncHistoryStatusAll, syncHistoryStatusValues())
	if !ok {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationInvalidQueryValue, map[string]any{"name": "status", "allowed": "all, Pending, Running, Completed, Failed, Unknown"}, nil)
		return
	}

	namespace := string(ctx.QueryArgs().Peek("namespace"))
	if namespace == "" {
		var err error
		namespace, err = h.findNamespace(c, name)
		if err != nil {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
	}

	instance, err := h.DisasterClient.DisasterV1().DisasterInstances(namespace).Get(c, name, matev1.GetOptions{})
	if err != nil {
		transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
		return
	}

	rawItems := make([]syncHistoryItemWithSort, 0)
	if source == syncHistorySourceSyncRecord || source == syncHistorySourceAll {
		if instance.Status.DataSyncName != "" {
			if ds, err := h.DisasterClient.DisasterV1().DataSyncs(namespace).Get(c, instance.Status.DataSyncName, matev1.GetOptions{}); err == nil {
				appendSyncRecordHistory(&rawItems, syncHistoryTypeDataSync, ds.Name, ds.Status.History)
			}
		}
		if instance.Status.ResourceSyncName != "" {
			if rs, err := h.DisasterClient.DisasterV1().ResourceSyncs(namespace).Get(c, instance.Status.ResourceSyncName, matev1.GetOptions{}); err == nil {
				appendSyncRecordHistory(&rawItems, syncHistoryTypeResourceSync, rs.Name, rs.Status.History)
			}
		}
	}

	if source == syncHistorySourceOperation || source == syncHistorySourceAll {
		labelSelector := fmt.Sprintf("testudo.softcdata.com/instance=%s", name)
		list, err := h.DisasterClient.DisasterV1().DisasterOperations(namespace).List(c, matev1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
			return
		}
		for i := range list.Items {
			item, ok := syncHistoryItemFromOperation(&list.Items[i])
			if ok {
				rawItems = append(rawItems, item)
			}
		}
	}

	filtered := make([]syncHistoryItemWithSort, 0, len(rawItems))
	for _, item := range rawItems {
		if syncHistoryMatches(item.SyncHistoryItemDTO, syncType, statusFilter) {
			filtered = append(filtered, item)
		}
	}
	sortSyncHistoryItems(filtered)

	dtos := make([]SyncHistoryItemDTO, 0, len(filtered))
	for _, item := range filtered {
		dtos = append(dtos, item.SyncHistoryItemDTO)
	}
	summary := summarizeSyncHistoryItems(dtos)
	pagedDtos, total := transport.Paginate(dtos, qParams)

	requestPath := string(ctx.URI().Path())
	data, meta := transport.BuildCollectionResponse(
		requestPath,
		"syncHistory",
		pagedDtos,
		qParams,
		total,
		nil,
		nil,
	)
	meta.Summary = summary
	transport.WriteSuccess(ctx, consts.StatusOK, data, meta)
}

// 7. Get History
func (h *InstanceHandler) getHistory(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")

	// Namespace resolution
	ns := ctx.QueryArgs().Peek("namespace")
	namespace := string(ns)
	if namespace == "" {
		var err error
		namespace, err = h.findNamespace(c, name)
		if err != nil {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
	}

	// List DisasterOperations with label selector
	labelSelector := fmt.Sprintf("testudo.softcdata.com/instance=%s", name)
	list, err := h.DisasterClient.DisasterV1().DisasterOperations(namespace).List(c, matev1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	history := make([]HistoryDTO, 0, len(list.Items))
	for _, op := range list.Items {
		opCopy := op.DeepCopy()
		history = append(history, ConvertToHistoryDTO(opCopy))
	}

	// Sort by Time Descending
	sort.Slice(history, func(i, j int) bool {
		return history[i].Time.Time.Time.After(history[j].Time.Time.Time)
	})

	transport.WriteSuccess(ctx, consts.StatusOK, history, nil)
}

// 8. Watch (Single/All)
func (h *InstanceHandler) watchInstances(c context.Context, ctx *app.RequestContext) {
	watcherFunc := func(ctx context.Context) (watch.Interface, error) {
		w1, err := h.DisasterClient.DisasterV1().DisasterInstances("").Watch(ctx, matev1.ListOptions{})
		if err != nil {
			return nil, err
		}
		w2, err := h.DisasterClient.DisasterV1().DataSyncs("").Watch(ctx, matev1.ListOptions{})
		if err != nil {
			w1.Stop()
			return nil, err
		}
		w3, err := h.DisasterClient.DisasterV1().ResourceSyncs("").Watch(ctx, matev1.ListOptions{})
		if err != nil {
			w1.Stop()
			w2.Stop()
			return nil, err
		}
		return NewMultiWatcher(w1, w2, w3), nil
	}

	converter := func(obj interface{}) interface{} {
		var instance *dapisv1.DisasterInstance

		if item, ok := obj.(*dapisv1.DisasterInstance); ok {
			instance = item
		} else if ds, ok := obj.(*dapisv1.DataSync); ok {
			// Find parent via naming convention: dr-ds-<instanceName>
			if strings.HasPrefix(ds.Name, "dr-ds-") {
				instanceName := strings.TrimPrefix(ds.Name, "dr-ds-")
				inst, err := h.DisasterClient.DisasterV1().DisasterInstances(ds.Namespace).Get(c, instanceName, matev1.GetOptions{})
				if err == nil {
					instance = inst
				}
			}
		} else if rs, ok := obj.(*dapisv1.ResourceSync); ok {
			// Find parent via naming convention: dr-rs-<instanceName>
			if strings.HasPrefix(rs.Name, "dr-rs-") {
				instanceName := strings.TrimPrefix(rs.Name, "dr-rs-")
				inst, err := h.DisasterClient.DisasterV1().DisasterInstances(rs.Namespace).Get(c, instanceName, matev1.GetOptions{})
				if err == nil {
					instance = inst
				}
			}
		}

		if instance != nil {
			dto := ConvertToDisasterInstanceDTO(instance, nil, nil)
			// Enrich with sync status
			dtos := []DisasterInstanceDTO{dto}
			h.enrichListSyncStatus(c, dtos)
			return dtos[0]
		}
		return nil
	}

	watchutils.StreamWatch(c, ctx, watcherFunc, converter)
}

func (h *InstanceHandler) watchInstance(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")

	// Helper to find namespace first? Or assume Watch works with FieldSelector across all namespaces?
	// K8s Watch with FieldSelector name usually requires Namespace to be known or it watches all?
	// But duplicate names in diff namespaces could issue.
	// For now, keep it simple: Watch all instances filtering by name.

	watcherFunc := func(ctx context.Context) (watch.Interface, error) {
		// Only watch the specific instance for single view?
		// User requirement "ws监听资源也需要监听数据同步和资源同步的状态" apply to List primarily.
		// For Single Instance Watch, we should probably also include its syncs?
		// Yes.

		// Find Namespace first to be efficient?
		ns, _ := h.findNamespace(c, name) // Ignore error, empty means all

		w1, err := h.DisasterClient.DisasterV1().DisasterInstances(ns).Watch(ctx, matev1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", name),
		})
		if err != nil {
			return nil, err
		}

		// Also watch syncs for this instance
		// Assuming names are deterministic
		dsName := "dr-ds-" + name
		rsName := "dr-rs-" + name

		w2, err := h.DisasterClient.DisasterV1().DataSyncs(ns).Watch(ctx, matev1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", dsName),
		})
		if err != nil {
			w1.Stop()
			return nil, err
		}

		w3, err := h.DisasterClient.DisasterV1().ResourceSyncs(ns).Watch(ctx, matev1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", rsName),
		})
		if err != nil {
			w1.Stop()
			w2.Stop()
			return nil, err
		}

		return NewMultiWatcher(w1, w2, w3), nil
	}

	// Converter is same as above basically
	converter := func(obj interface{}) interface{} {
		// ... (reuse logic or simplify since we know the instance)
		if item, ok := obj.(*dapisv1.DisasterInstance); ok {
			dto := ConvertToDisasterInstanceDTO(item, nil, nil)
			dtos := []DisasterInstanceDTO{dto}
			h.enrichListSyncStatus(c, dtos)
			return dtos[0]
		}
		// If Sync Object changes, we know it belongs to 'name'.
		// Just fetch the Instance 'name'.
		// Optimization: We know 'name' and 'ns' (mostly).

		// But obj doesn't tell us enough context if we use closures?
		// Actually, if we are in watchInstance(name), we know we want THAT instance.
		// So if ANY of w1, w2, w3 fires, we fetch 'name' and return DTO.

		// However, StreamWatch passes the *Event Object* to converter.
		// If DataSync fires, obj is DataSync.

		// Robust logic:
		ns := ""
		if o, ok := obj.(matev1.Object); ok {
			ns = o.GetNamespace()
		}

		inst, err := h.DisasterClient.DisasterV1().DisasterInstances(ns).Get(c, name, matev1.GetOptions{})
		if err == nil {
			dto := ConvertToDisasterInstanceDTO(inst, nil, nil)
			dtos := []DisasterInstanceDTO{dto}
			h.enrichListSyncStatus(c, dtos)
			return dtos[0]
		}
		return nil
	}

	watchutils.StreamWatch(c, ctx, watcherFunc, converter)
}

// MultiWatcher aggregates multiple watchers into one
type MultiWatcher struct {
	watchers []watch.Interface
	result   chan watch.Event
	stop     chan struct{}
	once     sync.Once
}

func NewMultiWatcher(watchers ...watch.Interface) *MultiWatcher {
	mw := &MultiWatcher{
		watchers: watchers,
		result:   make(chan watch.Event),
		stop:     make(chan struct{}),
	}
	mw.start()
	return mw
}

func (mw *MultiWatcher) start() {
	for _, w := range mw.watchers {
		go func(watcher watch.Interface) {
			for {
				select {
				case event, ok := <-watcher.ResultChan():
					if !ok {
						return
					}
					select {
					case mw.result <- event:
					case <-mw.stop:
						return
					}
				case <-mw.stop:
					return
				}
			}
		}(w)
	}
}

func (mw *MultiWatcher) Stop() {
	mw.once.Do(func() {
		close(mw.stop)
		for _, w := range mw.watchers {
			if w != nil {
				w.Stop()
			}
		}
	})
}

func (mw *MultiWatcher) ResultChan() <-chan watch.Event {
	return mw.result
}
