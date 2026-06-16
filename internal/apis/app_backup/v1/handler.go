package appbackup

import (
	"context"
	"fmt"
	"time"

	"strings"
	"sync"

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
	"github.com/softcdata/testudo-server/internal/storage"
	transport "github.com/softcdata/testudo-server/internal/transport"
	watchutils "github.com/softcdata/testudo-server/internal/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	matev1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/util/retry"
)

type AppBackupHandler struct {
	*kube.KubeClient
	Rg                      *route.RouterGroup
	Mw                      []app.HandlerFunc
	AppBackupLister         listers.AppBackupLister
	Storage                 storage.StorageService
	getRemoteClient         remoteClientGetter
	fetchBackupResourceList backupResourceListFetcher
	includesCache           *veleroBackupIncludesCache
}

type veleroBackupIncludesCache struct {
	mu      sync.RWMutex
	entries map[string]cachedVeleroBackupIncludes
}

type cachedVeleroBackupIncludes struct {
	dto             VeleroBackupIncludesDTO
	resourceVersion string
	expiresAt       time.Time
}

const (
	appResourceOriginLabelKey         = "testudo.softcdata.com/app-resource-origin"
	appResourceOriginUser             = "user"
	appResourceOriginDisasterInstance = "disaster-instance"
	backupIncludesCacheTTL            = 5 * time.Minute
)

func NewAppBackupHandler(kc *kube.KubeClient, rg *route.RouterGroup, storage storage.StorageService, mw ...app.HandlerFunc) *AppBackupHandler {
	return &AppBackupHandler{
		KubeClient:              kc,
		Rg:                      rg,
		Mw:                      mw,
		AppBackupLister:         kc.InformerFactory.Disaster().V1().AppBackups().Lister(),
		Storage:                 storage,
		getRemoteClient:         defaultRemoteClientGetter(kc),
		fetchBackupResourceList: fetchBackupResourceListFromDownloadRequest,
		includesCache: &veleroBackupIncludesCache{
			entries: make(map[string]cachedVeleroBackupIncludes),
		},
	}
}

func (h *AppBackupHandler) appBackups(c context.Context, ctx *app.RequestContext) {
	// 1. 解析通用查询参数
	qParams := transport.ParseOptions(c, ctx)
	originFilter, err := parseAppResourceOriginFilter(ctx.Query("origin"))
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	// origin 为业务过滤参数，不参与通用标签模糊匹配
	delete(qParams.Filters, "origin")

	// 2. 构建 Label Selector (K8s 级别初步筛选)
	selector := transport.BuildLabelSelector(qParams)

	// 3. 调用 Lister 获取初步筛选后的数据
	items, err := h.AppBackupLister.AppBackups(common.DisasterSystemNamespace).List(selector)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	// 3.1 全量内存模糊过滤 (实现“一律模糊搜索”)
	filteredItems := make([]*dapisv1.AppBackup, 0)
	for _, item := range items {
		if !matchAppResourceOriginFilter(item.Labels, item.OwnerReferences, originFilter) {
			continue
		}

		match := true
		for k, v := range qParams.Filters {
			// 获取资源对应的标签值进行包含匹配
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
	sortedItems := transport.Sort(filteredItems, qParams, func(a, b *dapisv1.AppBackup, field string) int {
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
	dtos := make([]AppBackupDTO, len(pagedItems))
	for i, item := range pagedItems {
		dtos[i] = ConvertToAppBackupDTO(item)
	}

	// 6. 构建标准响应
	qParams.Filters["origin"] = originFilter
	requestPath := string(ctx.URI().Path())
	data, meta := transport.BuildCollectionResponse(
		requestPath,
		"appBackup",
		dtos,
		qParams,
		total,
		nil,
		func(item AppBackupDTO) map[string]string {
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

func matchAppResourceOriginFilter(resourceLabels map[string]string, ownerRefs []matev1.OwnerReference, originFilter string) bool {
	if originFilter == "all" {
		return true
	}

	origin := inferAppResourceOrigin(resourceLabels, ownerRefs)

	if originFilter == "instance" {
		return origin == appResourceOriginDisasterInstance
	}
	// 默认 user: 兼容旧数据（未打标）和明确 user 打标
	return origin != appResourceOriginDisasterInstance
}

func inferAppResourceOrigin(resourceLabels map[string]string, ownerRefs []matev1.OwnerReference) string {
	if resourceLabels != nil {
		if value := resourceLabels[appResourceOriginLabelKey]; value != "" {
			return value
		}
	}

	for _, ownerRef := range ownerRefs {
		if ownerRef.Controller != nil && *ownerRef.Controller {
			if ownerRef.Kind == "DataSync" || ownerRef.Kind == "ResourceSync" {
				return appResourceOriginDisasterInstance
			}
		}
	}

	return appResourceOriginUser
}

// getAppBackupClusters returns a unique list of clusters that are used in AppBackups
func (h *AppBackupHandler) getAppBackupClusters(c context.Context, ctx *app.RequestContext) {
	// List all AppBackups without filtering
	items, err := h.AppBackupLister.AppBackups(common.DisasterSystemNamespace).List(labels.NewSelector())
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	// Use a map to collect unique clusters
	uniqueClusters := make(map[string]struct{})
	for _, item := range items {
		if item.Spec.Cluster != "" {
			uniqueClusters[item.Spec.Cluster] = struct{}{}
		}
	}

	// Convert map to list of simple DTOs
	type ClusterNameDTO struct {
		Name string `json:"name"`
	}
	dtos := make([]ClusterNameDTO, 0, len(uniqueClusters))
	for name := range uniqueClusters {
		dtos = append(dtos, ClusterNameDTO{Name: name})
	}

	transport.WriteSuccess(ctx, consts.StatusOK, dtos, nil)
}

func (h *AppBackupHandler) appBackup(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")
	if name == "" {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationNameRequired, nil, nil)
		return
	}
	item, err := h.DisasterClient.DisasterV1().AppBackups(common.DisasterSystemNamespace).Get(c, name, matev1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}
	dto := ConvertToAppBackupDTO(item)
	// Manually populate history for detail view
	dto.Status.History = ConvertBackupRecordsToDTO(item.Status.History)
	transport.WriteSuccess(ctx, consts.StatusOK, dto, nil)
}

func (h *AppBackupHandler) getBackupHistory(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")
	if name == "" {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationNameRequired, nil, nil)
		return
	}
	item, err := h.DisasterClient.DisasterV1().AppBackups(common.DisasterSystemNamespace).Get(c, name, matev1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	// Filter by status if parameter is provided
	statusFilter := string(ctx.Query("status"))
	if statusFilter != "" {
		var filtered []BackupRecordDTO
		for _, rec := range item.Status.History {
			if rec.Phase == statusFilter {
				filtered = append(filtered, ConvertBackupRecordToDTO(rec))
			}
		}
		transport.WriteSuccess(ctx, consts.StatusOK, filtered, nil)
		return
	}

	// Return just the history array
	transport.WriteSuccess(ctx, consts.StatusOK, ConvertBackupRecordsToDTO(item.Status.History), nil)
}

func (h *AppBackupHandler) createAppBackup(c context.Context, ctx *app.RequestContext) {
	var req CreateAppBackupRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	if err := validateCreateAppBackupResourceFilters(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	if err := velerohooks.ValidateBackupHooks(req.Hooks, "hooks"); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), velerohooks.ErrorMeta(err))
		return
	}

	// Validate dependencies before creating AppBackup
	// 1. Validate Cluster is ready
	if err := common.ValidateClusterReady(c, h.KubeClient, req.Cluster); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	// 2. Validate StorageRepository is available
	if err := common.ValidateStorageRepositoryAvailable(c, h.KubeClient, req.StorageLocation); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	annotations := make(map[string]string)
	if req.Description != "" {
		annotations[AppBackupDescriptionAnnotation] = req.Description
	}
	body := dapisv1.AppBackup{
		ObjectMeta: matev1.ObjectMeta{
			Name:        req.Name,
			Namespace:   common.DisasterSystemNamespace,
			Annotations: annotations,
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

	item, err := h.DisasterClient.DisasterV1().AppBackups(common.DisasterSystemNamespace).Create(c, &body, matev1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			transport.WriteError(ctx, transport.CodeConflict, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}
	dto := ConvertToAppBackupDTO(item)
	transport.WriteSuccess(ctx, consts.StatusCreated, dto, nil)
}

func (h *AppBackupHandler) updateAppBackup(c context.Context, ctx *app.RequestContext) {
	var req UpdateAppBackupRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	if err := validateUpdateAppBackupResourceFilters(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	if err := velerohooks.ValidateBackupHooks(req.Hooks, "hooks"); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), velerohooks.ErrorMeta(err))
		return
	}

	var result *dapisv1.AppBackup
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Get existing object
		existing, err := h.DisasterClient.DisasterV1().AppBackups(common.DisasterSystemNamespace).Get(c, req.Name, matev1.GetOptions{})
		if err != nil {
			return err
		}

		if req.Description != nil {
			if existing.Annotations == nil {
				existing.Annotations = make(map[string]string)
			}
			if *req.Description == "" {
				delete(existing.Annotations, AppBackupDescriptionAnnotation)
			} else {
				existing.Annotations[AppBackupDescriptionAnnotation] = *req.Description
			}
		}
		// Update Spec
		req.MergeToCRD(&existing.Spec)

		// Update trace-id annotation for this write operation
		// Update trace-id annotation for this write operation
		transport.SetTraceAnnotation(&existing.ObjectMeta, ctx, metadata.AnnotationTraceID)
		if user, ok := ctx.Get("userName"); ok {
			if existing.Annotations == nil {
				existing.Annotations = make(map[string]string)
			}
			existing.Annotations["testudo.softcdata.com/user"] = user.(string)
		}

		result, err = h.DisasterClient.DisasterV1().AppBackups(common.DisasterSystemNamespace).Update(c, existing, matev1.UpdateOptions{})
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
	dto := ConvertToAppBackupDTO(result)
	transport.WriteSuccess(ctx, consts.StatusOK, dto, nil)
}

func (h *AppBackupHandler) deleteAppBackup(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")
	// Optionally annotate with current trace-id before deletion for correlation
	// Best-effort: fetch, set annotation, update, then delete
	// If get/update fails, proceed to delete to avoid blocking
	existing, _ := h.DisasterClient.DisasterV1().AppBackups(common.DisasterSystemNamespace).Get(c, name, matev1.GetOptions{})
	if existing != nil {
		transport.SetTraceAnnotation(&existing.ObjectMeta, ctx, metadata.AnnotationTraceID)
		if user, ok := ctx.Get("userName"); ok {
			if existing.Annotations == nil {
				existing.Annotations = make(map[string]string)
			}
			existing.Annotations["testudo.softcdata.com/user"] = user.(string)
		}
		_, _ = h.DisasterClient.DisasterV1().AppBackups(common.DisasterSystemNamespace).Update(c, existing, matev1.UpdateOptions{})
	}
	err := h.DisasterClient.DisasterV1().AppBackups(common.DisasterSystemNamespace).Delete(c, name, matev1.DeleteOptions{})
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

func (h *AppBackupHandler) watchAppBackups(c context.Context, ctx *app.RequestContext) {
	watcherFunc := func(ctx context.Context) (watch.Interface, error) {
		return h.DisasterClient.DisasterV1().AppBackups(common.DisasterSystemNamespace).Watch(ctx, matev1.ListOptions{})
	}
	watchutils.StreamWatch(c, ctx, watcherFunc, func(obj interface{}) interface{} {
		if item, ok := obj.(*dapisv1.AppBackup); ok {
			return ConvertToAppBackupDTO(item)
		}
		return nil
	})
}

func (h *AppBackupHandler) watchAppBackup(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")
	if name == "" {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationNameRequired, nil, nil)
		return
	}

	watcherFunc := func(ctx context.Context) (watch.Interface, error) {
		return h.DisasterClient.DisasterV1().AppBackups(common.DisasterSystemNamespace).Watch(ctx, matev1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", name),
		})
	}
	watchutils.StreamWatch(c, ctx, watcherFunc, func(obj interface{}) interface{} {
		if item, ok := obj.(*dapisv1.AppBackup); ok {
			return ConvertToAppBackupDTO(item)
		}
		return nil
	})
}

func (h *AppBackupHandler) downloadBackup(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")
	backupName := ctx.Param("backupName")
	downloadType := ctx.Query("type")

	if name == "" || backupName == "" {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationNameBackupNameRequired, nil, nil)
		return
	}

	appBackup, err := h.DisasterClient.DisasterV1().AppBackups(common.DisasterSystemNamespace).Get(c, name, matev1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	// Verify backup exists in history
	found := false
	for _, rec := range appBackup.Status.History {
		if rec.Name == backupName {
			found = true
			break
		}
	}
	if !found {
		transport.WriteErrorKey(ctx, transport.CodeNotFound, i18n.KeyAppBackupRecordMiss, nil, nil)
		return
	}

	// Get StorageRepository
	storageLocation := appBackup.Spec.Template.StorageLocation
	if storageLocation == "" {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyAppBackupStorageMiss, nil, nil)
		return
	}

	repo, err := h.DisasterClient.DisasterV1().StorageRepositories(common.DisasterSystemNamespace).Get(c, storageLocation, matev1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, fmt.Sprintf("StorageRepository %s not found", storageLocation), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	// Object Key Structure: <prefix>/backups/<backup-name>/<backup-name>.tar.gz
	// The prefix is generally the Cluster name (see disaster-operator logic).

	expiry := 1 * time.Hour
	caBundle, err := h.loadStorageCABundle(c, repo)
	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, "Failed to load storage CA bundle: "+err.Error(), nil)
		return
	}

	if downloadType == "data" || downloadType == "all" {
		prefixes := make([]string, 0)
		cluster := appBackup.Spec.Cluster

		if downloadType == "all" {
			prefixes = append(prefixes, fmt.Sprintf("%s/backups/%s/", cluster, backupName))
		}

		for _, ns := range appBackup.Spec.Template.IncludedNamespaces {
			prefixes = append(prefixes, fmt.Sprintf("%s/kopia/%s/", cluster, ns))
			prefixes = append(prefixes, fmt.Sprintf("%s/restic/%s/", cluster, ns))
		}

		// List all objects under the prefixes
		objects, err := h.Storage.ListObjects(c, repo.Spec.Endpoint, repo.Spec.AccessKey, repo.Spec.SecretKey, repo.Spec.Bucket, repo.Spec.Region, repo.Spec.GetAddressingStyle(), caBundle, prefixes)
		if err != nil {
			transport.WriteError(ctx, transport.CodeInternalServerError, "Failed to list backup objects: "+err.Error(), nil)
			return
		}

		if len(objects) == 0 {
			transport.WriteError(ctx, transport.CodeNotFound, "No backup data files found for the specified prefixes", nil)
			return
		}

		// Generate presigned URLs for each object
		files := make([]BackupFileDownload, 0, len(objects))
		for _, obj := range objects {
			url, err := h.Storage.GetDownloadURL(c, repo.Spec.Endpoint, repo.Spec.AccessKey, repo.Spec.SecretKey, repo.Spec.Bucket, repo.Spec.Region, repo.Spec.GetAddressingStyle(), caBundle, obj.Key, expiry)
			if err != nil {
				transport.WriteError(ctx, transport.CodeInternalServerError, fmt.Sprintf("Failed to generate download URL for %s: %s", obj.Key, err.Error()), nil)
				return
			}
			files = append(files, BackupFileDownload{
				Key:         obj.Key,
				DownloadURL: url,
				Size:        obj.Size,
			})
		}

		resp := BackupDownloadResponse{
			Files:     files,
			ExpiresAt: common.NewLocalTime(matev1.NewTime(time.Now().Add(expiry))),
		}
		transport.WriteSuccess(ctx, consts.StatusOK, resp, nil)
		return
	}

	objectKey := fmt.Sprintf("%s/backups/%s/%s.tar.gz", appBackup.Spec.Cluster, backupName, backupName)

	url, err := h.Storage.GetDownloadURL(c, repo.Spec.Endpoint, repo.Spec.AccessKey, repo.Spec.SecretKey, repo.Spec.Bucket, repo.Spec.Region, repo.Spec.GetAddressingStyle(), caBundle, objectKey, expiry)
	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, "Failed to generate download URL: "+err.Error(), nil)
		return
	}

	resp := BackupDownloadResponse{
		DownloadURL: url,
		ExpiresAt:   common.NewLocalTime(matev1.NewTime(time.Now().Add(expiry))),
	}
	transport.WriteSuccess(ctx, consts.StatusOK, resp, nil)
}

func (h *AppBackupHandler) executeAction(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")
	actionType := strings.ToLower(ctx.Param("type"))

	if name == "" {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationNameRequired, nil, nil)
		return
	}

	// Parse optional body for parameters like targetBackup
	var req AppBackupActionRequest
	if string(ctx.Request.Body()) != "" {
		if err := ctx.BindJSON(&req); err != nil {
			// Log warning but proceed? Or fail? Best to allow empty body if bind fails due to empty or malformed if not critical,
			// but explicit body is better. Let's assume strict JSON if body present.
			transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationInvalidRequestBody, map[string]any{"error": err}, nil)
			return
		}
	}

	// 1. Handle Backup/Retry/Cancel/Delete Actions
	if actionType == "backup" || actionType == "retry" || actionType == "cancel" || actionType == "delete" {
		var normalizedType string
		switch actionType {
		case "backup":
			normalizedType = "Backup"
		case "retry":
			normalizedType = "Retry"
		case "cancel":
			normalizedType = "Cancel"
		case "delete":
			normalizedType = "Delete"
		}

		var result *dapisv1.AppBackup
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			existing, err := h.DisasterClient.DisasterV1().AppBackups(common.DisasterSystemNamespace).Get(c, name, matev1.GetOptions{})
			if err != nil {
				return err
			}

			existing.Spec.Action = &dapisv1.BackupAction{
				Type:         normalizedType,
				TargetBackup: req.TargetBackup,
				RequestAt:    matev1.Now(),
			}

			transport.SetTraceAnnotation(&existing.ObjectMeta, ctx, metadata.AnnotationTraceID)
			if user, ok := ctx.Get("userName"); ok {
				if existing.Annotations == nil {
					existing.Annotations = make(map[string]string)
				}
				existing.Annotations["testudo.softcdata.com/user"] = user.(string)
			}
			result, err = h.DisasterClient.DisasterV1().AppBackups(common.DisasterSystemNamespace).Update(c, existing, matev1.UpdateOptions{})
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

		data := map[string]interface{}{
			"type":          result.Spec.Action.Type,
			"target_backup": result.Spec.Action.TargetBackup,
			"request_at":    result.Spec.Action.RequestAt,
		}
		transport.WriteSuccess(ctx, consts.StatusOK, data, nil)
		return
	}

	// 2. Handle Pause/Resume Actions
	if actionType == "pause" || actionType == "resume" {
		targetPausedState := (actionType == "pause")

		var result *dapisv1.AppBackup
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			existing, err := h.DisasterClient.DisasterV1().AppBackups(common.DisasterSystemNamespace).Get(c, name, matev1.GetOptions{})
			if err != nil {
				return err
			}

			existing.Spec.Paused = targetPausedState
			if existing.Annotations == nil {
				existing.Annotations = make(map[string]string)
			}
			existing.Annotations[metadata.AnnotationAppBackupManualPaused] = fmt.Sprintf("%t", targetPausedState)

			transport.SetTraceAnnotation(&existing.ObjectMeta, ctx, metadata.AnnotationTraceID)
			if user, ok := ctx.Get("userName"); ok {
				if existing.Annotations == nil {
					existing.Annotations = make(map[string]string)
				}
				existing.Annotations["testudo.softcdata.com/user"] = user.(string)
			}
			result, err = h.DisasterClient.DisasterV1().AppBackups(common.DisasterSystemNamespace).Update(c, existing, matev1.UpdateOptions{})
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

		data := map[string]interface{}{
			"paused": result.Spec.Paused,
		}
		transport.WriteSuccess(ctx, consts.StatusOK, data, nil)
		return
	}

	transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationUnsupportedAction, map[string]any{"type": actionType}, nil)
}

func (h *AppBackupHandler) loadStorageCABundle(ctx context.Context, repo *dapisv1.StorageRepository) ([]byte, error) {
	if repo.Spec.CASecretRef == nil || repo.Spec.CASecretRef.Name == "" {
		return nil, nil
	}

	secret, err := h.K8sClient.CoreV1().Secrets(common.DisasterSystemNamespace).Get(ctx, repo.Spec.CASecretRef.Name, matev1.GetOptions{})
	if err != nil {
		return nil, err
	}

	caBundle, ok := secret.Data[dapisv1.StorageRepositoryCASecretKey]
	if !ok || len(caBundle) == 0 {
		return nil, fmt.Errorf("secret %s does not contain %s", repo.Spec.CASecretRef.Name, dapisv1.StorageRepositoryCASecretKey)
	}
	return caBundle, nil
}
