package systemsettings

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/i18n"
	"github.com/softcdata/testudo-server/internal/kube"
	"github.com/softcdata/testudo-server/internal/transport"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

const (
	settingsConfigMapName = "disaster-platform-settings"
	settingsDataKey       = "settings"
	settingsSchemaVersion = 1

	maxNameLength       = 64
	maxRemarkLength     = 1024
	maxValueBytes       = 350 * 1024
	maxAssetRawBytes    = 256 * 1024
	maxAssetValueBytes  = 350 * 1024
	maxSettingsJSONSize = 900 * 1024
)

var (
	configKeyRegex = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)

	errSettingNotFound  = errors.New("setting not found")
	errSettingExists    = errors.New("setting already exists")
	errSettingsTooLarge = errors.New("settings payload exceeds limit")
	errAssetTooLarge    = errors.New("asset file exceeds limit")
	errInvalidDataURL   = errors.New("invalid base64 data url")
)

type SystemSettingsHandler struct {
	*kube.KubeClient
	Rg *route.RouterGroup
	Mw []app.HandlerFunc

	store configMapStore
}

func NewSystemSettingsHandler(kc *kube.KubeClient, rg *route.RouterGroup, mw ...app.HandlerFunc) *SystemSettingsHandler {
	var store configMapStore
	if kc != nil && kc.K8sClient != nil {
		store = &kubeConfigMapStore{client: kc.K8sClient}
	}
	return &SystemSettingsHandler{
		KubeClient: kc,
		Rg:         rg,
		Mw:         mw,
		store:      store,
	}
}

func (h *SystemSettingsHandler) listSettings(c context.Context, ctx *app.RequestContext) {
	qParams := transport.ParseOptions(c, ctx)
	doc, err := h.getSettingsDocument(c)
	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	keysFilter := parseKeys(string(ctx.Query("keys")))
	keyword := strings.TrimSpace(string(ctx.Query("q")))
	if keyword == "" {
		keyword = strings.TrimSpace(qParams.Keyword)
	}
	items := collectItems(doc.Items, keysFilter, keyword)
	pagedItems, total := transport.Paginate(items, qParams)

	requestPath := string(ctx.URI().Path())
	data, meta := transport.BuildCollectionResponse(
		requestPath,
		"systemSetting",
		pagedItems,
		qParams,
		total,
		nil,
		func(item SystemSettingItem) map[string]string {
			return map[string]string{
				item.ConfigKey: fmt.Sprintf("%s/%s", strings.TrimRight(requestPath, "/"), item.ConfigKey),
			}
		},
	)

	transport.WriteSuccess(ctx, consts.StatusOK, data, meta)
}

func (h *SystemSettingsHandler) listPublicSettings(c context.Context, ctx *app.RequestContext) {
	qParams := transport.ParseOptions(c, ctx)
	keysFilter := parseKeys(string(ctx.Query("keys")))
	if len(keysFilter) == 0 {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationKeysRequired, nil, nil)
		return
	}

	doc, err := h.getSettingsDocument(c)
	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	items := collectItems(doc.Items, keysFilter, "")
	pagedItems, total := transport.Paginate(items, qParams)

	requestPath := string(ctx.URI().Path())
	data, meta := transport.BuildCollectionResponse(
		requestPath,
		"systemSetting",
		pagedItems,
		qParams,
		total,
		nil,
		func(item SystemSettingItem) map[string]string {
			return map[string]string{
				item.ConfigKey: fmt.Sprintf("%s/%s", strings.TrimRight(requestPath, "/"), item.ConfigKey),
			}
		},
	)

	transport.WriteSuccess(ctx, consts.StatusOK, data, meta)
}

func (h *SystemSettingsHandler) createSetting(c context.Context, ctx *app.RequestContext) {
	if !requireSystemAdmin(ctx) {
		return
	}

	var req CreateSystemSettingRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	item, err := normalizeCreateRequest(req)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	_, err = h.mutateSettingsDocument(c, actorFromContext(ctx), func(doc *settingsDocument) error {
		if _, exists := doc.Items[item.ConfigKey]; exists {
			return errSettingExists
		}
		doc.Items[item.ConfigKey] = item
		return nil
	})
	if err != nil {
		writeSettingsError(ctx, err)
		return
	}

	transport.WriteSuccess(ctx, consts.StatusCreated, item, nil)
}

func (h *SystemSettingsHandler) updateSetting(c context.Context, ctx *app.RequestContext) {
	if !requireSystemAdmin(ctx) {
		return
	}

	configKey := strings.TrimSpace(ctx.Param("config_key"))
	if err := validateConfigKey(configKey); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	var req UpdateSystemSettingRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	if req.Name == nil && req.Value == nil && req.Remark == nil {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationAtLeastOneField, nil, nil)
		return
	}

	var normalizedName *string
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if err := validateName(name); err != nil {
			transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
			return
		}
		normalizedName = &name
	}

	var normalizedRemark *string
	if req.Remark != nil {
		remark := strings.TrimSpace(*req.Remark)
		if err := validateRemark(remark); err != nil {
			transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
			return
		}
		normalizedRemark = &remark
	}

	var normalizedValue *string
	if req.Value != nil {
		if err := validateValue(*req.Value); err != nil {
			transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
			return
		}
		value := *req.Value
		normalizedValue = &value
	}

	var updated SystemSettingItem
	_, err := h.mutateSettingsDocument(c, actorFromContext(ctx), func(doc *settingsDocument) error {
		item, exists := doc.Items[configKey]
		if !exists {
			return errSettingNotFound
		}

		if item.ConfigKey == "" {
			item.ConfigKey = configKey
		}

		if normalizedName != nil {
			item.Name = *normalizedName
		}

		if normalizedValue != nil {
			item.Value = *normalizedValue
		}

		if normalizedRemark != nil {
			item.Remark = *normalizedRemark
		}

		doc.Items[configKey] = item
		updated = item
		return nil
	})
	if err != nil {
		writeSettingsError(ctx, err)
		return
	}

	transport.WriteSuccess(ctx, consts.StatusOK, updated, nil)
}

func (h *SystemSettingsHandler) deleteSetting(c context.Context, ctx *app.RequestContext) {
	if !requireSystemAdmin(ctx) {
		return
	}

	configKey := strings.TrimSpace(ctx.Param("config_key"))
	if err := validateConfigKey(configKey); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	_, err := h.mutateSettingsDocument(c, actorFromContext(ctx), func(doc *settingsDocument) error {
		if _, exists := doc.Items[configKey]; !exists {
			return errSettingNotFound
		}
		delete(doc.Items, configKey)
		return nil
	})
	if err != nil {
		writeSettingsError(ctx, err)
		return
	}

	transport.WriteSuccess(ctx, consts.StatusOK, map[string]string{"config_key": configKey}, nil)
}

func (h *SystemSettingsHandler) uploadAsset(c context.Context, ctx *app.RequestContext) {
	if !requireSystemAdmin(ctx) {
		return
	}

	configKey := strings.TrimSpace(ctx.Param("config_key"))
	if err := validateConfigKey(configKey); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationFileRequired, nil, nil)
		return
	}
	if fileHeader.Size > maxAssetRawBytes {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationAssetFileExceedsLimit, nil, nil)
		return
	}

	assetFile, err := fileHeader.Open()
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	defer assetFile.Close()

	raw, err := io.ReadAll(io.LimitReader(assetFile, maxAssetRawBytes+1))
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	if len(raw) > maxAssetRawBytes {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationAssetFileExceedsLimit, nil, nil)
		return
	}

	contentType, err := resolveContentType(fileHeader)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	value := encodeDataURL(contentType, raw)
	if len(value) > maxAssetValueBytes {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationAssetValueExceedsLimit, nil, nil)
		return
	}

	var updated SystemSettingItem
	_, err = h.mutateSettingsDocument(c, actorFromContext(ctx), func(doc *settingsDocument) error {
		item, exists := doc.Items[configKey]
		if !exists {
			item = SystemSettingItem{
				Name:      configKey,
				ConfigKey: configKey,
				Remark:    "",
			}
		}
		if item.ConfigKey == "" {
			item.ConfigKey = configKey
		}
		if strings.TrimSpace(item.Name) == "" {
			item.Name = configKey
		}
		item.Value = value
		doc.Items[configKey] = item
		updated = item
		return nil
	})
	if err != nil {
		writeSettingsError(ctx, err)
		return
	}

	transport.WriteSuccess(ctx, consts.StatusOK, updated, nil)
}

func (h *SystemSettingsHandler) getAsset(c context.Context, ctx *app.RequestContext) {
	configKey := strings.TrimSpace(ctx.Param("config_key"))
	if err := validateConfigKey(configKey); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	doc, err := h.getSettingsDocument(c)
	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	item, exists := doc.Items[configKey]
	if !exists {
		transport.WriteError(ctx, transport.CodeNotFound, errSettingNotFound.Error(), nil)
		return
	}

	contentType, decoded, err := decodeDataURL(item.Value)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	ctx.Data(consts.StatusOK, contentType, decoded)
}

func normalizeCreateRequest(req CreateSystemSettingRequest) (SystemSettingItem, error) {
	configKey := strings.TrimSpace(req.ConfigKey)
	if err := validateConfigKey(configKey); err != nil {
		return SystemSettingItem{}, err
	}

	name := strings.TrimSpace(req.Name)
	if err := validateName(name); err != nil {
		return SystemSettingItem{}, err
	}

	remark := strings.TrimSpace(req.Remark)
	if err := validateRemark(remark); err != nil {
		return SystemSettingItem{}, err
	}

	if err := validateValue(req.Value); err != nil {
		return SystemSettingItem{}, err
	}

	return SystemSettingItem{
		Name:      name,
		ConfigKey: configKey,
		Value:     req.Value,
		Remark:    remark,
	}, nil
}

func validateConfigKey(configKey string) error {
	if !configKeyRegex.MatchString(configKey) {
		return fmt.Errorf("invalid config_key: %q", configKey)
	}
	return nil
}

func validateName(name string) error {
	if name == "" {
		return errors.New("name is required")
	}
	if len(name) > maxNameLength {
		return fmt.Errorf("name exceeds max length %d", maxNameLength)
	}
	return nil
}

func validateRemark(remark string) error {
	if len(remark) > maxRemarkLength {
		return fmt.Errorf("remark exceeds max length %d", maxRemarkLength)
	}
	return nil
}

func validateValue(value string) error {
	if len(value) > maxValueBytes {
		return fmt.Errorf("value exceeds max length %d", maxValueBytes)
	}
	return nil
}

func parseKeys(raw string) map[string]struct{} {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	keys := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		key := strings.TrimSpace(part)
		if key == "" {
			continue
		}
		keys[key] = struct{}{}
	}
	if len(keys) == 0 {
		return nil
	}
	return keys
}

func collectItems(items map[string]SystemSettingItem, keysFilter map[string]struct{}, keyword string) []SystemSettingItem {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	out := make([]SystemSettingItem, 0, len(items))
	for key, item := range items {
		if _, ok := keysFilter[key]; len(keysFilter) > 0 && !ok {
			continue
		}
		if item.ConfigKey == "" {
			item.ConfigKey = key
		}
		if strings.TrimSpace(item.Name) == "" {
			item.Name = item.ConfigKey
		}
		if keyword != "" {
			if !strings.Contains(strings.ToLower(item.Name), keyword) &&
				!strings.Contains(strings.ToLower(item.ConfigKey), keyword) {
				continue
			}
		}
		out = append(out, item)
	}

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ConfigKey < out[j].ConfigKey
	})
	return out
}

func actorFromContext(ctx *app.RequestContext) string {
	if ctx == nil {
		return "system"
	}
	if value, ok := ctx.Get("userName"); ok {
		if userName, ok := value.(string); ok {
			userName = strings.TrimSpace(userName)
			if userName != "" {
				return userName
			}
		}
	}
	return "system"
}

func requireSystemAdmin(ctx *app.RequestContext) bool {
	if ctx == nil {
		return false
	}
	value, ok := ctx.Get("userName")
	if !ok {
		// In dev mode JWT may be disabled, keep backward compatibility.
		return true
	}
	userName, _ := value.(string)
	userName = strings.TrimSpace(userName)
	if userName == "" {
		return true
	}
	if strings.EqualFold(userName, "admin") {
		return true
	}
	transport.WriteErrorKey(ctx, transport.CodeForbidden, i18n.KeyOnlyAdminCanModify, nil, nil)
	return false
}

func writeSettingsError(ctx *app.RequestContext, err error) {
	switch {
	case errors.Is(err, errSettingExists):
		transport.WriteError(ctx, transport.CodeConflict, err.Error(), nil)
	case errors.Is(err, errSettingNotFound):
		transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
	case errors.Is(err, errAssetTooLarge):
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationAssetFileExceedsLimit, nil, nil)
	case errors.Is(err, errSettingsTooLarge):
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
	case k8serrors.IsConflict(err):
		transport.WriteError(ctx, transport.CodeConflict, err.Error(), nil)
	default:
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
	}
}

func (h *SystemSettingsHandler) getSettingsDocument(c context.Context) (*settingsDocument, error) {
	if h.store == nil {
		return nil, errors.New("configmap store is not initialized")
	}

	cm, err := h.store.Get(c, settingsConfigMapName)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return defaultSettingsDocument(), nil
		}
		return nil, err
	}
	return decodeSettingsDocument(cm)
}

func (h *SystemSettingsHandler) mutateSettingsDocument(c context.Context, actor string, mutate func(doc *settingsDocument) error) (*settingsDocument, error) {
	if h.store == nil {
		return nil, errors.New("configmap store is not initialized")
	}

	var result *settingsDocument
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cm, doc, err := h.loadSettingsConfigMap(c)
		if err != nil {
			return err
		}

		if err := mutate(doc); err != nil {
			return err
		}

		doc.SchemaVersion = settingsSchemaVersion
		doc.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		doc.UpdatedBy = actor

		payload, err := json.Marshal(doc)
		if err != nil {
			return err
		}
		if len(payload) > maxSettingsJSONSize {
			return errSettingsTooLarge
		}

		if cm == nil {
			cm = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      settingsConfigMapName,
					Namespace: common.DisasterSystemNamespace,
				},
				Data: map[string]string{
					settingsDataKey: string(payload),
				},
			}
			if _, err := h.store.Create(c, cm); err != nil {
				return err
			}
		} else {
			if cm.Data == nil {
				cm.Data = make(map[string]string)
			}
			cm.Data[settingsDataKey] = string(payload)
			if _, err := h.store.Update(c, cm); err != nil {
				return err
			}
		}

		result = doc
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (h *SystemSettingsHandler) loadSettingsConfigMap(c context.Context) (*corev1.ConfigMap, *settingsDocument, error) {
	cm, err := h.store.Get(c, settingsConfigMapName)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, defaultSettingsDocument(), nil
		}
		return nil, nil, err
	}

	doc, err := decodeSettingsDocument(cm)
	if err != nil {
		return nil, nil, err
	}
	return cm, doc, nil
}

type configMapStore interface {
	Get(ctx context.Context, name string) (*corev1.ConfigMap, error)
	Create(ctx context.Context, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error)
	Update(ctx context.Context, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error)
}

type kubeConfigMapStore struct {
	client kubernetes.Interface
}

func (s *kubeConfigMapStore) Get(ctx context.Context, name string) (*corev1.ConfigMap, error) {
	return s.client.CoreV1().ConfigMaps(common.DisasterSystemNamespace).Get(ctx, name, metav1.GetOptions{})
}

func (s *kubeConfigMapStore) Create(ctx context.Context, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	return s.client.CoreV1().ConfigMaps(common.DisasterSystemNamespace).Create(ctx, configMap, metav1.CreateOptions{})
}

func (s *kubeConfigMapStore) Update(ctx context.Context, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	return s.client.CoreV1().ConfigMaps(common.DisasterSystemNamespace).Update(ctx, configMap, metav1.UpdateOptions{})
}

func defaultSettingsDocument() *settingsDocument {
	return &settingsDocument{
		SchemaVersion: settingsSchemaVersion,
		Items:         make(map[string]SystemSettingItem),
	}
}

func decodeSettingsDocument(cm *corev1.ConfigMap) (*settingsDocument, error) {
	doc := defaultSettingsDocument()
	if cm == nil || cm.Data == nil {
		return doc, nil
	}

	raw := strings.TrimSpace(cm.Data[settingsDataKey])
	if raw == "" {
		return doc, nil
	}

	if err := json.Unmarshal([]byte(raw), doc); err != nil {
		return nil, fmt.Errorf("invalid settings json: %w", err)
	}

	if doc.Items == nil {
		doc.Items = make(map[string]SystemSettingItem)
	}
	if doc.SchemaVersion == 0 {
		doc.SchemaVersion = settingsSchemaVersion
	}
	return doc, nil
}

func resolveContentType(fileHeader *multipart.FileHeader) (string, error) {
	contentType := strings.ToLower(strings.TrimSpace(fileHeader.Header.Get("Content-Type")))
	if semi := strings.Index(contentType, ";"); semi >= 0 {
		contentType = strings.TrimSpace(contentType[:semi])
	}
	if contentType == "image/jpg" {
		contentType = "image/jpeg"
	}

	if contentType == "" || contentType == "application/octet-stream" {
		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		contentType = mime.TypeByExtension(ext)
		contentType = strings.ToLower(strings.TrimSpace(contentType))
		if semi := strings.Index(contentType, ";"); semi >= 0 {
			contentType = strings.TrimSpace(contentType[:semi])
		}
		if contentType == "image/jpg" {
			contentType = "image/jpeg"
		}
	}

	switch contentType {
	case "image/png", "image/jpeg":
		return contentType, nil
	default:
		return "", fmt.Errorf("unsupported content type: %s", contentType)
	}
}

func encodeDataURL(contentType string, payload []byte) string {
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(payload)
}

func decodeDataURL(value string) (string, []byte, error) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "data:") {
		return "", nil, errInvalidDataURL
	}

	parts := strings.SplitN(value, ",", 2)
	if len(parts) != 2 {
		return "", nil, errInvalidDataURL
	}

	header := strings.TrimPrefix(parts[0], "data:")
	if !strings.HasSuffix(header, ";base64") {
		return "", nil, errInvalidDataURL
	}
	contentType := strings.TrimSuffix(header, ";base64")
	if contentType == "" {
		return "", nil, errInvalidDataURL
	}

	decoded, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", nil, errInvalidDataURL
	}
	return contentType, decoded, nil
}
