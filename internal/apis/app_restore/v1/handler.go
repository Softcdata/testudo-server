package apprestore

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
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
	"github.com/softcdata/testudo-server/internal/resourcemodifier"
	"github.com/softcdata/testudo-server/internal/service/verifier"
	"github.com/softcdata/testudo-server/internal/transport"
	watchutils "github.com/softcdata/testudo-server/internal/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	matev1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/util/retry"
	ctrclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type AppRestoreHandler struct {
	*kube.KubeClient
	Rg                       *route.RouterGroup
	Mw                       []app.HandlerFunc
	AppRestoreLister         listers.AppRestoreLister
	RestorePreflightVerifier verifier.RestorePreflightVerifier
	GetClusterClient         func(ctx context.Context, clusterName string) (ctrclient.Client, error)
}

const (
	appResourceOriginLabelKey         = "testudo.softcdata.com/app-resource-origin"
	appResourceOriginUser             = "user"
	appResourceOriginDisasterInstance = "disaster-instance"
)

func NewAppRestoreHandler(kc *kube.KubeClient, rg *route.RouterGroup, mw ...app.HandlerFunc) *AppRestoreHandler {
	return &AppRestoreHandler{
		KubeClient:               kc,
		Rg:                       rg,
		Mw:                       mw,
		AppRestoreLister:         kc.InformerFactory.Disaster().V1().AppRestores().Lister(),
		RestorePreflightVerifier: verifier.NewRestorePreflightVerifier(),
		GetClusterClient: func(ctx context.Context, clusterName string) (ctrclient.Client, error) {
			return kc.GetKubeClient(ctx, kc.RuntimeClient(), kc.Scheme(), clusterName)
		},
	}
}

func (h *AppRestoreHandler) appRestores(c context.Context, ctx *app.RequestContext) {
	// 1. 解析通用查询参数
	qParams := transport.ParseOptions(c, ctx)
	originFilter, err := parseAppResourceOriginFilter(ctx.Query("origin"))
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	// origin 为业务过滤参数，不参与通用标签模糊匹配
	delete(qParams.Filters, "origin")

	// 2. 构建 Label Selector
	selector := transport.BuildLabelSelector(qParams)

	// 3. 调用 Lister 获取初步数据
	items, err := h.AppRestoreLister.AppRestores(common.DisasterSystemNamespace).List(selector)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	// 3.0 全量内存模糊过滤
	filteredItems := make([]*dapisv1.AppRestore, 0)
	for _, item := range items {
		if !matchAppResourceOriginFilter(item.Name, item.Labels, item.OwnerReferences, originFilter) {
			continue
		}

		match := true
		for k, v := range qParams.Filters {
			// Special handling for TargetNamespaces search
			if k == "targetNamespaces" {
				// We search if ANY of the target namespaces in the item matches the query value (fuzzy)
				// Item source: Status.TargetNamespaces or Spec...
				// For simplicity/performance, let's use the same logic as DTO conversion or just Status if populated
				// Let's use the DTO logic helper to get the list? Or just check Status for now as it contains the truth often.
				// But we haven't converted to DTO yet.
				// Logic: Check if Status.TargetNamespaces contains v (substring match)
				// OR if Spec.NamespaceMapping values contain v
				// OR if Spec.IncludedNamespaces contains v (if mapping empty)

				// Re-using DTO logic briefly without full conversion to save perf?
				// Actually, full conversion happens later. Let's do a quick extraction.
				targets := item.Status.TargetNamespaces
				if len(targets) == 0 {
					if len(item.Spec.Template.NamespaceMapping) > 0 {
						for _, val := range item.Spec.Template.NamespaceMapping {
							targets = append(targets, val)
						}
					} else {
						targets = append(targets, item.Spec.Template.IncludedNamespaces...)
					}
				}

				nsMatch := false
				for _, t := range targets {
					if transport.MatchFuzzy(t, v) {
						nsMatch = true
						break
					}
				}
				if !nsMatch {
					match = false
					break
				}
				continue
			}

			// Default Label matching
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

	// 4. 内存排序逻辑
	sortedItems := transport.Sort(filteredItems, qParams, func(a, b *dapisv1.AppRestore, field string) int {
		switch field {
		case "name":
			return strings.Compare(a.Name, b.Name)
		case "creationTimestamp":
			if a.CreationTimestamp.Before(&b.CreationTimestamp) {
				return -1
			}
			if a.CreationTimestamp.After(b.CreationTimestamp.Time) {
				return 1
			}
			return 0
		default:
			return 0
		}
	})

	// 5. 内存分页逻辑
	pagedItems, total := transport.Paginate(sortedItems, qParams)

	// Convert to DTOs
	dtos := make([]AppRestoreDTO, len(pagedItems))
	for i, item := range pagedItems {
		dtos[i] = ConvertToAppRestoreDTO(item)
	}

	// 6. 构建标准响应
	qParams.Filters["origin"] = originFilter
	requestPath := string(ctx.URI().Path())
	data, meta := transport.BuildCollectionResponse(
		requestPath,
		"appRestore",
		dtos,
		qParams,
		total,
		nil,
		func(item AppRestoreDTO) map[string]string {
			return map[string]string{
				item.Name: fmt.Sprintf("%s/%s", strings.TrimRight(requestPath, "/"), item.Name),
			}
		},
	)

	transport.WriteSuccess(ctx, consts.StatusOK, data, meta)
}

func parseAppResourceOriginFilter(origin string) (string, error) {
	switch origin {
	case "", "user":
		return "user", nil
	case "instance":
		return "instance", nil
	case "all":
		return "all", nil
	default:
		return "", fmt.Errorf("invalid origin: %s, must be one of user|instance|all", origin)
	}
}

func matchAppResourceOriginFilter(resourceName string, resourceLabels map[string]string, ownerRefs []matev1.OwnerReference, originFilter string) bool {
	if originFilter == "all" {
		return true
	}

	origin := inferAppResourceOrigin(resourceName, resourceLabels, ownerRefs)

	if originFilter == "instance" {
		return origin == appResourceOriginDisasterInstance
	}
	// 默认 user: 兼容旧数据（未打标）和明确 user 打标
	return origin != appResourceOriginDisasterInstance
}

func inferAppResourceOrigin(resourceName string, resourceLabels map[string]string, ownerRefs []matev1.OwnerReference) string {
	for _, ownerRef := range ownerRefs {
		if ownerRef.Controller != nil && *ownerRef.Controller {
			if ownerRef.Kind == "DataSync" || ownerRef.Kind == "ResourceSync" {
				return appResourceOriginDisasterInstance
			}
		}
	}

	// Drill/Operation 自动创建的 AppRestore 可能在历史数据里被标为 user，
	// 这里基于稳定标签与命名前缀兜底判定为系统来源，避免污染默认用户视图。
	if isDrillManagedRestore(resourceName, resourceLabels) {
		return appResourceOriginDisasterInstance
	}

	if resourceLabels != nil {
		if value := resourceLabels[appResourceOriginLabelKey]; value != "" {
			return value
		}
	}

	return appResourceOriginUser
}

func isDrillManagedRestore(resourceName string, resourceLabels map[string]string) bool {
	if resourceLabels != nil {
		if _, ok := resourceLabels["testudo.softcdata.com/drill"]; ok {
			return true
		}
	}

	return strings.HasPrefix(resourceName, "ddr-") || strings.HasPrefix(resourceName, "drr-")
}

func (h *AppRestoreHandler) appRestore(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")
	if name == "" {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationNameRequired, nil, nil)
		return
	}
	item, err := h.DisasterClient.DisasterV1().AppRestores(common.DisasterSystemNamespace).Get(c, name, matev1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}
	dto := ConvertToAppRestoreDTO(item)
	transport.WriteSuccess(ctx, consts.StatusOK, dto, nil)
}

func (h *AppRestoreHandler) validateRestorePreflight(c context.Context, ctx *app.RequestContext) {
	var req ValidateRestorePreflightRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	appBackup, err := h.DisasterClient.DisasterV1().AppBackups(common.DisasterSystemNamespace).Get(c, req.BackupSource, matev1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteErrorKey(ctx, transport.CodeNotFound, i18n.KeyAppBackupNotFound, map[string]any{"name": req.BackupSource}, nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, fmt.Sprintf("failed to get AppBackup %s: %v", req.BackupSource, err), nil)
		return
	}

	targetCli, err := h.getClusterClient(c, req.TargetCluster)
	if err != nil {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyClusterClientFailed, map[string]any{"cluster": req.TargetCluster}, map[string]string{"details": err.Error()})
		return
	}

	preflightVerifier := h.getRestorePreflightVerifier()
	result, err := preflightVerifier.VerifyRestorePreflight(c, targetCli, h.getRuntimeClient(), appBackup, req.TargetCluster, req.WaitSeconds)
	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), buildRestorePreflightMeta(result))
		return
	}
	if result == nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, "restore preflight verifier returned nil result", nil)
		return
	}

	if !result.Valid {
		transport.WriteSuccess(ctx, consts.StatusOK, false, buildRestorePreflightMeta(result))
		return
	}

	transport.WriteSuccess(ctx, consts.StatusOK, true, buildRestorePreflightMeta(result))
}

func (h *AppRestoreHandler) createAppRestore(c context.Context, ctx *app.RequestContext) {
	var req CreateAppRestoreRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	effectiveSCMapping, err := resolveStorageClassMapping(req.StorageClassMapping, req.SCMapping)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	if err := velerohooks.ValidateRestoreHooks(req.Hooks, "hooks"); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), velerohooks.ErrorMeta(err))
		return
	}
	body := dapisv1.AppRestore{
		ObjectMeta: matev1.ObjectMeta{
			Name:      req.Name,
			Namespace: common.DisasterSystemNamespace,
			Annotations: map[string]string{
				AppRestoreDescriptionAnnotation: req.Description,
			},
		},
		Spec: req.ToCRD(),
	}

	// Inject trace_id annotation for operator correlation
	// Inject trace_id annotation for operator correlation
	transport.SetTraceAnnotation(&body.ObjectMeta, ctx, metadata.AnnotationTraceID)
	if user, ok := ctx.Get("userName"); ok {
		if body.Annotations == nil {
			body.Annotations = make(map[string]string)
		}
		body.Annotations["testudo.softcdata.com/user"] = user.(string)
	}

	// 查询appbackup获取源备份信息
	appBackup, err := h.DisasterClient.DisasterV1().AppBackups(common.DisasterSystemNamespace).Get(c, body.Spec.BackupSource, matev1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteErrorKey(ctx, transport.CodeNotFound, i18n.KeyAppBackupNotFound, map[string]any{"name": body.Spec.BackupSource}, nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, fmt.Sprintf("failed to get AppBackup %s: %v", body.Spec.BackupSource, err), nil)
		return
	}
	if appBackup == nil {
		transport.WriteErrorKey(ctx, transport.CodeNotFound, i18n.KeyAppBackupNotFound, map[string]any{"name": body.Spec.BackupSource}, nil)
		return
	}
	if appBackup.Spec.Cluster == "" || appBackup.Spec.Template.StorageLocation == "" {
		transport.WriteError(ctx, transport.CodeBadRequest, fmt.Sprintf("AppBackup %s is missing required fields", body.Spec.BackupSource), nil)
		return
	}
	// Fill SourceCluster
	body.Spec.SourceCluster = appBackup.Spec.Cluster
	body.Spec.StorageRepository = appBackup.Spec.Template.StorageLocation

	// If BackupName is not provided, use the latest one from AppBackup history
	if body.Spec.Template.BackupName == "" {
		if len(appBackup.Status.History) == 0 {
			transport.WriteError(ctx, transport.CodeBadRequest, fmt.Sprintf("AppBackup %s has no backup history", body.Spec.BackupSource), nil)
			return
		}
		// Use the latest backup name
		body.Spec.Template.BackupName = appBackup.Status.History[0].Name
	}

	body.Spec.ResourceModifierRules = make([]dapisv1.ResourceModifierRule, 0)
	if len(effectiveSCMapping) > 0 {
		body.Spec.ResourceModifierRules = append(body.Spec.ResourceModifierRules, resourcemodifier.SCMapping(effectiveSCMapping)...)
	}
	if len(req.IngressClassMapping) > 0 {
		body.Spec.ResourceModifierRules = append(body.Spec.ResourceModifierRules, resourcemodifier.IngressClassMapping(req.IngressClassMapping)...)
	}
	if len(req.ScaleToZeroList) > 0 {
		body.Spec.ResourceModifierRules = append(body.Spec.ResourceModifierRules, resourcemodifier.ScaleToZero(req.ScaleToZeroList)...)
	}
	if len(req.StandbyList) > 0 {
		body.Spec.ResourceModifierRules = append(body.Spec.ResourceModifierRules, resourcemodifier.StandbyReplacement(req.StandbyList)...)
	}
	if req.TrafficlessImage != "" {
		body.Spec.ResourceModifierRules = append(body.Spec.ResourceModifierRules, resourcemodifier.TrafficlessRestore(req.TrafficlessImage))
	}
	if needsCleanVolumeRuleForCreate(req) {
		body.Spec.ResourceModifierRules = ensureCleanVolumeRule(body.Spec.ResourceModifierRules)
	}

	// Validate target Cluster is ready before creating AppRestore
	if err := common.ValidateClusterReady(c, h.KubeClient, body.Spec.Cluster); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	targetCli, err := h.getClusterClient(c, body.Spec.Cluster)
	if err != nil {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyClusterClientFailed, map[string]any{"cluster": body.Spec.Cluster}, map[string]string{"details": err.Error()})
		return
	}

	preflightVerifier := h.getRestorePreflightVerifier()
	preflightResult, err := preflightVerifier.VerifyRestorePreflight(c, targetCli, h.getRuntimeClient(), appBackup, body.Spec.Cluster, 0)
	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), buildRestorePreflightMeta(preflightResult))
		return
	}
	if preflightResult == nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, "restore preflight verifier returned nil result", nil)
		return
	}
	if !preflightResult.Valid {
		transport.WriteError(ctx, transport.CodeBadRequest, preflightResult.Reason, buildRestorePreflightMeta(preflightResult))
		return
	}

	// Validate ExistingResourcePolicy
	if req.ExistingResourcePolicy != "" && req.ExistingResourcePolicy != "none" && req.ExistingResourcePolicy != "update" {
		transport.WriteError(ctx, transport.CodeBadRequest, fmt.Sprintf("invalid existingResourcePolicy: %s, must be 'none' or 'update'", req.ExistingResourcePolicy), nil)
		return
	}

	item, err := h.DisasterClient.DisasterV1().AppRestores(common.DisasterSystemNamespace).Create(c, &body, matev1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			transport.WriteError(ctx, transport.CodeConflict, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}
	dto := ConvertToAppRestoreDTO(item)
	transport.WriteSuccess(ctx, consts.StatusCreated, dto, nil)
}

func (h *AppRestoreHandler) getRestorePreflightVerifier() verifier.RestorePreflightVerifier {
	if h.RestorePreflightVerifier != nil {
		return h.RestorePreflightVerifier
	}
	return verifier.NewRestorePreflightVerifier()
}

func (h *AppRestoreHandler) getRuntimeClient() ctrclient.Client {
	if h.KubeClient == nil || h.KubeClient.ClusterClient == nil {
		return nil
	}
	return h.KubeClient.RuntimeClient()
}

func (h *AppRestoreHandler) getClusterClient(ctx context.Context, clusterName string) (ctrclient.Client, error) {
	if h.GetClusterClient != nil {
		return h.GetClusterClient(ctx, clusterName)
	}
	return h.KubeClient.GetKubeClient(ctx, h.KubeClient.RuntimeClient(), h.KubeClient.Scheme(), clusterName)
}

func buildRestorePreflightMeta(result *verifier.RestorePreflightResult) map[string]string {
	if result == nil {
		return nil
	}
	return map[string]string{
		"required_bsl":       result.RequiredBSL,
		"source_cluster":     result.SourceCluster,
		"target_cluster":     result.TargetCluster,
		"storage_repository": result.StorageRepository,
		"state":              result.Phase,
		"error":              result.Reason,
	}
}

func (h *AppRestoreHandler) updateAppRestore(c context.Context, ctx *app.RequestContext) {
	var req UpdateAppRestoreRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	// 路径参数是对象定位主键：优先采用 URL 中的 :name，兼容历史 body.name。
	pathName := strings.TrimSpace(ctx.Param("name"))
	bodyName := strings.TrimSpace(req.Name)
	switch {
	case pathName == "" && bodyName == "":
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationNameRequired, nil, nil)
		return
	case pathName != "" && bodyName != "" && pathName != bodyName:
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationNameURLBodyMismatch, nil, nil)
		return
	case pathName != "":
		req.Name = pathName
	default:
		req.Name = bodyName
	}

	effectiveSCMapping, err := resolveStorageClassMapping(req.StorageClassMapping, req.SCMapping)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	if err := velerohooks.ValidateRestoreHooks(req.Hooks, "hooks"); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), velerohooks.ErrorMeta(err))
		return
	}

	// Validate ExistingResourcePolicy
	if req.ExistingResourcePolicy != "" && req.ExistingResourcePolicy != "none" && req.ExistingResourcePolicy != "update" {
		transport.WriteError(ctx, transport.CodeBadRequest, fmt.Sprintf("invalid existingResourcePolicy: %s, must be 'none' or 'update'", req.ExistingResourcePolicy), nil)
		return
	}

	var result *dapisv1.AppRestore
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Get existing object
		existing, err := h.DisasterClient.DisasterV1().AppRestores(common.DisasterSystemNamespace).Get(c, req.Name, matev1.GetOptions{})
		if err != nil {
			return err
		}

		// Update Spec
		req.MergeToCRD(&existing.Spec)

		if len(effectiveSCMapping) > 0 {
			existing.Spec.ResourceModifierRules = append(existing.Spec.ResourceModifierRules, resourcemodifier.SCMapping(effectiveSCMapping)...)
		}
		if len(req.IngressClassMapping) > 0 {
			existing.Spec.ResourceModifierRules = append(existing.Spec.ResourceModifierRules, resourcemodifier.IngressClassMapping(req.IngressClassMapping)...)
		}
		if len(req.ScaleToZeroList) > 0 {
			existing.Spec.ResourceModifierRules = append(existing.Spec.ResourceModifierRules, resourcemodifier.ScaleToZero(req.ScaleToZeroList)...)
		}
		if len(req.StandbyList) > 0 {
			existing.Spec.ResourceModifierRules = append(existing.Spec.ResourceModifierRules, resourcemodifier.StandbyReplacement(req.StandbyList)...)
		}
		if needsCleanVolumeRuleForUpdate(req) {
			existing.Spec.ResourceModifierRules = ensureCleanVolumeRule(existing.Spec.ResourceModifierRules)
		}

		// Update trace-id annotation for this write operation
		// Update trace-id annotation for this write operation
		transport.SetTraceAnnotation(&existing.ObjectMeta, ctx, metadata.AnnotationTraceID)
		if user, ok := ctx.Get("userName"); ok {
			if existing.Annotations == nil {
				existing.Annotations = make(map[string]string)
			}
			existing.Annotations["testudo.softcdata.com/user"] = user.(string)
		}

		// Update Description if provided
		if req.Description != nil {
			if existing.Annotations == nil {
				existing.Annotations = make(map[string]string)
			}
			if *req.Description == "" {
				delete(existing.Annotations, AppRestoreDescriptionAnnotation)
			} else {
				existing.Annotations[AppRestoreDescriptionAnnotation] = *req.Description
			}
		}

		result, err = h.DisasterClient.DisasterV1().AppRestores(common.DisasterSystemNamespace).Update(c, existing, matev1.UpdateOptions{})
		return err
	})

	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		if errors.IsConflict(err) {
			transport.WriteError(ctx, transport.CodeConflict, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}
	dto := ConvertToAppRestoreDTO(result)
	transport.WriteSuccess(ctx, consts.StatusOK, dto, nil)
}

func needsCleanVolumeRuleForCreate(req CreateAppRestoreRequest) bool {
	return req.CleanVolumes || isRestorePVsEnabled(req.RestorePVs)
}

func needsCleanVolumeRuleForUpdate(req UpdateAppRestoreRequest) bool {
	return (req.CleanVolumes != nil && *req.CleanVolumes) || isRestorePVsEnabled(req.RestorePVs)
}

func isRestorePVsEnabled(v *bool) bool {
	return v != nil && *v
}

func ensureCleanVolumeRule(rules []dapisv1.ResourceModifierRule) []dapisv1.ResourceModifierRule {
	normalized := make([]dapisv1.ResourceModifierRule, 0, len(rules)+1)
	for i := range rules {
		rule := rules[i]
		if rule.Conditions.GroupResource != "persistentvolumeclaims" {
			normalized = append(normalized, rule)
			continue
		}
		patches := make([]dapisv1.JSONPatch, 0, len(rule.Patches))
		for _, patch := range rule.Patches {
			if isCleanVolumePatch(patch) {
				continue
			}
			patches = append(patches, patch)
		}
		if len(patches) == 0 {
			continue
		}
		rule.Patches = patches
		normalized = append(normalized, rule)
	}
	return append(normalized, resourcemodifier.CleanVolume())
}

func hasCleanVolumeRule(rules []dapisv1.ResourceModifierRule) bool {
	for i := range rules {
		rule := rules[i]
		if rule.Conditions.GroupResource != "persistentvolumeclaims" {
			continue
		}
		for _, patch := range rule.Patches {
			if isIdempotentCleanVolumePatch(patch) {
				return true
			}
		}
	}
	return false
}

func isCleanVolumePatch(patch dapisv1.JSONPatch) bool {
	return patch.Path == "/spec/volumeName" &&
		(patch.Operation == "remove" || isIdempotentCleanVolumePatch(patch))
}

func isIdempotentCleanVolumePatch(patch dapisv1.JSONPatch) bool {
	return patch.Operation == "add" && patch.Path == "/spec/volumeName" && patch.Value == ""
}

func resolveStorageClassMapping(storageClassMapping, scMapping map[string]string) (map[string]string, error) {
	if storageClassMapping != nil && scMapping != nil && !reflect.DeepEqual(storageClassMapping, scMapping) {
		return nil, fmt.Errorf("storageClassMapping and scMapping conflict: values are inconsistent")
	}
	if storageClassMapping != nil {
		return storageClassMapping, nil
	}
	return scMapping, nil
}

func (h *AppRestoreHandler) deleteAppRestore(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")
	// Optionally annotate with current trace-id before deletion for correlation
	// Best-effort: fetch, set annotation, update, then delete
	// If get/update fails, proceed to delete to avoid blocking
	existing, _ := h.DisasterClient.DisasterV1().AppRestores(common.DisasterSystemNamespace).Get(c, name, matev1.GetOptions{})
	if existing != nil {
		transport.SetTraceAnnotation(&existing.ObjectMeta, ctx, metadata.AnnotationTraceID)
		if user, ok := ctx.Get("userName"); ok {
			if existing.Annotations == nil {
				existing.Annotations = make(map[string]string)
			}
			existing.Annotations["testudo.softcdata.com/user"] = user.(string)
		}
		_, _ = h.DisasterClient.DisasterV1().AppRestores(common.DisasterSystemNamespace).Update(c, existing, matev1.UpdateOptions{})
	}
	err := h.DisasterClient.DisasterV1().AppRestores(common.DisasterSystemNamespace).Delete(c, name, matev1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}
	transport.WriteSuccess(ctx, consts.StatusOK, utils.H{"name": name}, nil)
}

func (h *AppRestoreHandler) watchAppRestores(c context.Context, ctx *app.RequestContext) {
	watcherFunc := func(ctx context.Context) (watch.Interface, error) {
		return h.DisasterClient.DisasterV1().AppRestores(common.DisasterSystemNamespace).Watch(ctx, matev1.ListOptions{})
	}
	watchutils.StreamWatch(c, ctx, watcherFunc, func(obj interface{}) interface{} {
		if item, ok := obj.(*dapisv1.AppRestore); ok {
			return ConvertToAppRestoreDTO(item)
		}
		return nil
	})
}

func (h *AppRestoreHandler) watchAppRestore(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")
	if name == "" {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationNameRequired, nil, nil)
		return
	}

	watcherFunc := func(ctx context.Context) (watch.Interface, error) {
		return h.DisasterClient.DisasterV1().AppRestores(common.DisasterSystemNamespace).Watch(ctx, matev1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", name),
		})
	}
	watchutils.StreamWatch(c, ctx, watcherFunc, func(obj interface{}) interface{} {
		if item, ok := obj.(*dapisv1.AppRestore); ok {
			return ConvertToAppRestoreDTO(item)
		}
		return nil
	})
}

func (h *AppRestoreHandler) executeAction(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")
	actionType := ctx.Param("type")

	if name == "" {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationNameRequired, nil, nil)
		return
	}

	// Normalize and validate action type
	var normalizedType string
	switch strings.ToLower(actionType) {
	case "cancel":
		normalizedType = "cancel"
	case "retry":
		normalizedType = "retry"
	default:
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationUnsupportedAction, map[string]any{"type": actionType}, nil)
		return
	}

	var result *dapisv1.AppRestore
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Get existing object
		existing, err := h.DisasterClient.DisasterV1().AppRestores(common.DisasterSystemNamespace).Get(c, name, matev1.GetOptions{})
		if err != nil {
			return err
		}

		// Update Action
		existing.Spec.Action = &dapisv1.RestoreAction{
			Type:      normalizedType,
			RequestAt: matev1.Now(),
		}

		// Update trace-id annotation
		// Update trace-id annotation
		transport.SetTraceAnnotation(&existing.ObjectMeta, ctx, metadata.AnnotationTraceID)
		if user, ok := ctx.Get("userName"); ok {
			if existing.Annotations == nil {
				existing.Annotations = make(map[string]string)
			}
			existing.Annotations["testudo.softcdata.com/user"] = user.(string)
		}

		result, err = h.DisasterClient.DisasterV1().AppRestores(common.DisasterSystemNamespace).Update(c, existing, matev1.UpdateOptions{})
		return err
	})

	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		if errors.IsConflict(err) {
			transport.WriteError(ctx, transport.CodeConflict, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	// Return success response with action details
	data := map[string]interface{}{
		"type":       result.Spec.Action.Type,
		"request_at": result.Spec.Action.RequestAt,
	}
	transport.WriteSuccess(ctx, consts.StatusOK, data, nil)
}
