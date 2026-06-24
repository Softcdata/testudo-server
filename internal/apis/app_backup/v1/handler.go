package appbackup

import (
	"archive/tar"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	goerrors "errors"
	"fmt"
	"io"
	"mime"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	listers "github.com/softcdata/testudo-operator/pkg/listers/disaster/v1"
	metadata "github.com/softcdata/testudo-operator/pkg/metadata"
	"github.com/softcdata/testudo-server/configs"
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
	backupDownloadProxyExpiry         = time.Hour
	backupDownloadDefaultType         = "resource"
	backupDownloadTokenVersion        = 1
	backupDownloadTokenPurpose        = "app-backup-download"
	httpTimeFormat                    = "Mon, 02 Jan 2006 15:04:05 GMT"
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

	appBackup, repo, normalizedType, err := h.prepareBackupDownload(c, name, backupName, downloadType)
	if err != nil {
		writeBackupDownloadError(ctx, err)
		return
	}

	expiry := time.Now().Add(backupDownloadProxyExpiry)
	token, err := h.signBackupDownloadToken(appBackup, repo, backupName, normalizedType, expiry, backupDownloadUser(ctx))
	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	resp := BackupDownloadResponse{
		DownloadURL: h.buildBackupDownloadStreamURL(name, backupName, token),
		ExpiresAt:   common.NewLocalTime(matev1.NewTime(expiry)),
		Mode:        "proxy",
		Type:        normalizedType,
		FileName:    backupDownloadFileName(backupName, normalizedType),
	}
	transport.WriteSuccess(ctx, consts.StatusOK, resp, nil)
}

func (h *AppBackupHandler) downloadBackupStream(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")
	backupName := ctx.Param("backupName")
	token := strings.TrimSpace(ctx.Query("downloadToken"))

	if name == "" || backupName == "" || token == "" {
		transport.WriteError(ctx, transport.CodeBadRequest, "downloadToken, name and backupName are required", nil)
		return
	}

	claims, err := h.verifyBackupDownloadToken(token)
	if err != nil {
		transport.WriteError(ctx, transport.CodeForbidden, err.Error(), nil)
		return
	}
	if claims.Name != name || claims.BackupName != backupName {
		transport.WriteError(ctx, transport.CodeForbidden, "download token does not match request path", nil)
		return
	}

	appBackup, repo, normalizedType, err := h.prepareBackupDownload(c, name, backupName, claims.Type)
	if err != nil {
		writeBackupDownloadError(ctx, err)
		return
	}
	if claims.AppBackupUID != string(appBackup.UID) || claims.StorageRepositoryUID != string(repo.UID) || claims.Type != normalizedType {
		transport.WriteError(ctx, transport.CodeForbidden, "download token does not match current resource state", nil)
		return
	}

	caBundle, err := h.loadStorageCABundle(c, repo)
	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, "Failed to load storage CA bundle: "+err.Error(), nil)
		return
	}

	switch normalizedType {
	case "resource":
		if err := h.streamBackupResource(c, ctx, appBackup, repo, caBundle, backupName); err != nil {
			writeBackupDownloadError(ctx, err)
			return
		}
	case "data", "all":
		if err := h.streamBackupArchive(c, ctx, appBackup, repo, caBundle, backupName, normalizedType); err != nil {
			writeBackupDownloadError(ctx, err)
			return
		}
	default:
		transport.WriteError(ctx, transport.CodeBadRequest, fmt.Sprintf("unsupported download type %q", normalizedType), nil)
	}
}

type backupDownloadError struct {
	code int
	msg  string
}

func (e *backupDownloadError) Error() string {
	return e.msg
}

func newBackupDownloadError(code int, msg string) error {
	return &backupDownloadError{code: code, msg: msg}
}

func writeBackupDownloadError(ctx *app.RequestContext, err error) {
	var downloadErr *backupDownloadError
	if goerrors.As(err, &downloadErr) {
		transport.WriteError(ctx, downloadErr.code, downloadErr.msg, nil)
		return
	}
	transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
}

func (h *AppBackupHandler) prepareBackupDownload(c context.Context, name, backupName, rawDownloadType string) (*dapisv1.AppBackup, *dapisv1.StorageRepository, string, error) {
	downloadType, err := normalizeBackupDownloadType(rawDownloadType)
	if err != nil {
		return nil, nil, "", newBackupDownloadError(transport.CodeBadRequest, err.Error())
	}

	appBackup, err := h.DisasterClient.DisasterV1().AppBackups(common.DisasterSystemNamespace).Get(c, name, matev1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil, "", newBackupDownloadError(transport.CodeNotFound, err.Error())
		}
		return nil, nil, "", err
	}

	found := false
	for _, rec := range appBackup.Status.History {
		if rec.Name == backupName {
			found = true
			break
		}
	}
	if !found {
		return nil, nil, "", newBackupDownloadError(transport.CodeNotFound, i18n.T(i18n.DefaultLocale, i18n.KeyAppBackupRecordMiss, nil))
	}
	if strings.TrimSpace(appBackup.Spec.Cluster) == "" {
		return nil, nil, "", newBackupDownloadError(transport.CodeBadRequest, "AppBackup cluster is required")
	}

	storageLocation := appBackup.Spec.Template.StorageLocation
	if storageLocation == "" {
		return nil, nil, "", newBackupDownloadError(transport.CodeBadRequest, i18n.T(i18n.DefaultLocale, i18n.KeyAppBackupStorageMiss, nil))
	}

	repo, err := h.DisasterClient.DisasterV1().StorageRepositories(common.DisasterSystemNamespace).Get(c, storageLocation, matev1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil, "", newBackupDownloadError(transport.CodeNotFound, fmt.Sprintf("StorageRepository %s not found", storageLocation))
		}
		return nil, nil, "", err
	}

	return appBackup, repo, downloadType, nil
}

func normalizeBackupDownloadType(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", backupDownloadDefaultType:
		return "resource", nil
	case "data":
		return "data", nil
	case "all":
		return "all", nil
	default:
		return "", fmt.Errorf("unsupported download type %q", value)
	}
}

type backupDownloadTokenClaims struct {
	Version              int    `json:"v"`
	Purpose              string `json:"purpose"`
	Name                 string `json:"name"`
	BackupName           string `json:"backupName"`
	Type                 string `json:"type"`
	AppBackupUID         string `json:"appBackupUID"`
	StorageRepositoryUID string `json:"storageRepositoryUID"`
	User                 string `json:"user,omitempty"`
	IssuedAt             int64  `json:"iat"`
	ExpiresAt            int64  `json:"exp"`
}

func (h *AppBackupHandler) signBackupDownloadToken(appBackup *dapisv1.AppBackup, repo *dapisv1.StorageRepository, backupName, downloadType string, expiresAt time.Time, user string) (string, error) {
	claims := backupDownloadTokenClaims{
		Version:              backupDownloadTokenVersion,
		Purpose:              backupDownloadTokenPurpose,
		Name:                 appBackup.Name,
		BackupName:           backupName,
		Type:                 downloadType,
		AppBackupUID:         string(appBackup.UID),
		StorageRepositoryUID: string(repo.UID),
		User:                 user,
		IssuedAt:             time.Now().Unix(),
		ExpiresAt:            expiresAt.Unix(),
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := signBackupDownloadTokenPayload([]byte(encodedPayload), backupDownloadSigningSecret())
	return encodedPayload + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func (h *AppBackupHandler) verifyBackupDownloadToken(token string) (*backupDownloadTokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid download token")
	}
	expectedSig := signBackupDownloadTokenPayload([]byte(parts[0]), backupDownloadSigningSecret())
	gotSig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(gotSig, expectedSig) {
		return nil, fmt.Errorf("invalid download token signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid download token payload")
	}
	var claims backupDownloadTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("invalid download token claims")
	}
	if claims.Version != backupDownloadTokenVersion || claims.Purpose != backupDownloadTokenPurpose {
		return nil, fmt.Errorf("invalid download token purpose")
	}
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, fmt.Errorf("download token expired")
	}
	if _, err := normalizeBackupDownloadType(claims.Type); err != nil {
		return nil, err
	}
	return &claims, nil
}

func signBackupDownloadTokenPayload(payload []byte, secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(payload)
	return mac.Sum(nil)
}

func backupDownloadSigningSecret() []byte {
	if configs.Cfg != nil && strings.TrimSpace(configs.Cfg.JWT.Secret) != "" {
		return []byte(configs.Cfg.JWT.Secret)
	}
	return []byte("testudo-download-token-test-secret")
}

func backupDownloadUser(ctx *app.RequestContext) string {
	if user, ok := ctx.Get("userName"); ok {
		if username, ok := user.(string); ok {
			return username
		}
	}
	return ""
}

func (h *AppBackupHandler) buildBackupDownloadStreamURL(name, backupName, token string) string {
	streamPath := fmt.Sprintf(
		"/apis/appbackups.%s/appbackups/%s/backups/%s/download/stream",
		dapisv1.GroupVersion.String(),
		url.PathEscape(name),
		url.PathEscape(backupName),
	)
	return streamPath + "?downloadToken=" + url.QueryEscape(token)
}

func backupDownloadFileName(backupName, downloadType string) string {
	switch downloadType {
	case "data":
		return backupName + "-data.tar"
	case "all":
		return backupName + "-all.tar"
	default:
		return backupName + ".tar.gz"
	}
}

func (h *AppBackupHandler) streamBackupResource(c context.Context, ctx *app.RequestContext, appBackup *dapisv1.AppBackup, repo *dapisv1.StorageRepository, caBundle []byte, backupName string) error {
	objectKey := fmt.Sprintf("%s/backups/%s/%s.tar.gz", appBackup.Spec.Cluster, backupName, backupName)
	object, err := h.Storage.GetObject(c, repo.Spec.Endpoint, repo.Spec.AccessKey, repo.Spec.SecretKey, repo.Spec.Bucket, repo.Spec.Region, repo.Spec.GetAddressingStyle(), caBundle, objectKey, string(ctx.Request.Header.Peek("Range")))
	if err != nil {
		return newBackupDownloadError(transport.CodeUpstreamError, "Failed to read backup object: "+err.Error())
	}

	statusCode := consts.StatusOK
	if object.ContentRange != "" {
		statusCode = consts.StatusPartialContent
		ctx.Response.Header.Set("Content-Range", object.ContentRange)
	}
	if object.AcceptRanges != "" {
		ctx.Response.Header.Set("Accept-Ranges", object.AcceptRanges)
	} else {
		ctx.Response.Header.Set("Accept-Ranges", "bytes")
	}
	if object.ETag != "" {
		ctx.Response.Header.Set("ETag", object.ETag)
	}
	if !object.LastModified.IsZero() {
		ctx.Response.Header.Set("Last-Modified", object.LastModified.UTC().Format(httpTimeFormat))
	}

	contentType := object.ContentType
	if contentType == "" {
		contentType = "application/gzip"
	}
	ctx.Response.Header.SetContentType(contentType)
	ctx.Response.Header.Set("Content-Disposition", contentDisposition(backupDownloadFileName(backupName, "resource")))
	ctx.Response.Header.Set("Cache-Control", "no-store")
	ctx.Response.SetStatusCode(statusCode)
	ctx.SetBodyStream(object.Body, responseBodySize(object.ContentLength))
	return nil
}

func (h *AppBackupHandler) streamBackupArchive(c context.Context, ctx *app.RequestContext, appBackup *dapisv1.AppBackup, repo *dapisv1.StorageRepository, caBundle []byte, backupName, downloadType string) error {
	prefixes := backupArchivePrefixes(appBackup, backupName, downloadType)
	if len(prefixes) == 0 {
		return newBackupDownloadError(transport.CodeNotFound, "No backup data prefixes found")
	}
	objects, err := h.Storage.ListObjects(c, repo.Spec.Endpoint, repo.Spec.AccessKey, repo.Spec.SecretKey, repo.Spec.Bucket, repo.Spec.Region, repo.Spec.GetAddressingStyle(), caBundle, prefixes)
	if err != nil {
		return newBackupDownloadError(transport.CodeUpstreamError, "Failed to list backup objects: "+err.Error())
	}
	if len(objects) == 0 {
		return newBackupDownloadError(transport.CodeNotFound, "No backup data files found for the specified prefixes")
	}

	reader, writer := io.Pipe()
	go func() {
		writerErr := h.writeBackupArchive(c, writer, appBackup, repo, caBundle, objects)
		_ = writer.CloseWithError(writerErr)
	}()

	ctx.Response.Header.SetContentType("application/x-tar")
	ctx.Response.Header.Set("Content-Disposition", contentDisposition(backupDownloadFileName(backupName, downloadType)))
	ctx.Response.Header.Set("Cache-Control", "no-store")
	ctx.Response.SetStatusCode(consts.StatusOK)
	ctx.SetBodyStream(reader, -1)
	return nil
}

func (h *AppBackupHandler) writeBackupArchive(c context.Context, writer io.Writer, appBackup *dapisv1.AppBackup, repo *dapisv1.StorageRepository, caBundle []byte, objects []storage.ObjectInfo) error {
	tw := tar.NewWriter(writer)
	defer func() {
		_ = tw.Close()
	}()
	for _, obj := range objects {
		object, err := h.Storage.GetObject(c, repo.Spec.Endpoint, repo.Spec.AccessKey, repo.Spec.SecretKey, repo.Spec.Bucket, repo.Spec.Region, repo.Spec.GetAddressingStyle(), caBundle, obj.Key, "")
		if err != nil {
			return err
		}
		if object.Body == nil {
			return fmt.Errorf("object %s has empty body", obj.Key)
		}

		size := object.ContentLength
		if size < 0 {
			size = obj.Size
		}
		header := &tar.Header{
			Name:    backupArchiveEntryName(appBackup.Spec.Cluster, obj.Key),
			Mode:    0600,
			Size:    size,
			ModTime: object.LastModified,
		}
		if header.ModTime.IsZero() {
			header.ModTime = time.Now()
		}
		if err := tw.WriteHeader(header); err != nil {
			_ = object.Body.Close()
			return err
		}
		_, copyErr := io.Copy(tw, object.Body)
		closeErr := object.Body.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func backupArchivePrefixes(appBackup *dapisv1.AppBackup, backupName, downloadType string) []string {
	cluster := strings.Trim(appBackup.Spec.Cluster, "/")
	prefixes := make([]string, 0)
	if downloadType == "all" {
		prefixes = append(prefixes, fmt.Sprintf("%s/backups/%s/", cluster, backupName))
	}
	for _, ns := range appBackup.Spec.Template.IncludedNamespaces {
		ns = strings.Trim(ns, "/")
		if ns == "" {
			continue
		}
		prefixes = append(prefixes, fmt.Sprintf("%s/kopia/%s/", cluster, ns))
		prefixes = append(prefixes, fmt.Sprintf("%s/restic/%s/", cluster, ns))
	}
	return prefixes
}

func backupArchiveEntryName(cluster, key string) string {
	name := strings.Trim(strings.TrimPrefix(key, strings.Trim(cluster, "/")+"/"), "/")
	if name == "" {
		return strings.Trim(key, "/")
	}
	return name
}

func contentDisposition(filename string) string {
	return mime.FormatMediaType("attachment", map[string]string{"filename": filename})
}

func responseBodySize(size int64) int {
	if size < 0 || size > int64(int(^uint(0)>>1)) {
		return -1
	}
	return int(size)
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
