package systemsettings

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route/param"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/transport"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type memoryConfigMapStore struct {
	mu        sync.Mutex
	configMap map[string]*corev1.ConfigMap
	version   int64
}

func newMemoryConfigMapStore() *memoryConfigMapStore {
	return &memoryConfigMapStore{
		configMap: make(map[string]*corev1.ConfigMap),
	}
}

func (m *memoryConfigMapStore) nextVersion() string {
	m.version++
	return strconv.FormatInt(m.version, 10)
}

func (m *memoryConfigMapStore) Get(_ context.Context, name string) (*corev1.ConfigMap, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cm, exists := m.configMap[name]
	if !exists {
		return nil, k8serrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, name)
	}
	return cm.DeepCopy(), nil
}

func (m *memoryConfigMapStore) Create(_ context.Context, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	name := configMap.Name
	if _, exists := m.configMap[name]; exists {
		return nil, k8serrors.NewAlreadyExists(schema.GroupResource{Resource: "configmaps"}, name)
	}

	cm := configMap.DeepCopy()
	cm.ResourceVersion = m.nextVersion()
	m.configMap[name] = cm
	return cm.DeepCopy(), nil
}

func (m *memoryConfigMapStore) Update(_ context.Context, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	name := configMap.Name
	if _, exists := m.configMap[name]; !exists {
		return nil, k8serrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, name)
	}

	cm := configMap.DeepCopy()
	cm.ResourceVersion = m.nextVersion()
	m.configMap[name] = cm
	return cm.DeepCopy(), nil
}

func newMockHandler(configMap *corev1.ConfigMap) (*SystemSettingsHandler, *memoryConfigMapStore) {
	store := newMemoryConfigMapStore()
	if configMap != nil {
		_, err := store.Create(context.Background(), configMap)
		if err != nil {
			panic(err)
		}
	}

	return &SystemSettingsHandler{
		store: store,
	}, store
}

func buildConfigMap(t *testing.T, items map[string]SystemSettingItem) *corev1.ConfigMap {
	t.Helper()
	doc := settingsDocument{
		SchemaVersion: settingsSchemaVersion,
		Items:         items,
		UpdatedAt:     "2026-03-11T12:00:00Z",
		UpdatedBy:     "admin",
	}
	raw, err := json.Marshal(doc)
	assert.NoError(t, err)

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      settingsConfigMapName,
			Namespace: common.DisasterSystemNamespace,
		},
		Data: map[string]string{
			settingsDataKey: string(raw),
		},
	}
}

func TestListSettingsSortedByConfigKey(t *testing.T) {
	cm := buildConfigMap(t, map[string]SystemSettingItem{
		"example.b": {
			Name:      "B",
			ConfigKey: "example.b",
			Value:     "2",
			Remark:    "",
		},
		"example.a": {
			Name:      "A",
			ConfigKey: "example.a",
			Value:     "1",
			Remark:    "",
		},
	})

	h, _ := newMockHandler(cm)
	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/api/v1/system-settings")

	h.listSettings(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Items []SystemSettingItem `json:"items"`
		} `json:"data"`
		Meta struct {
			Pagination struct {
				Limit int   `json:"limit"`
				Total int64 `json:"total"`
			} `json:"pagination"`
		} `json:"meta"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	assert.Len(t, resp.Data.Items, 2)
	assert.Equal(t, "example.a", resp.Data.Items[0].ConfigKey)
	assert.Equal(t, "example.b", resp.Data.Items[1].ConfigKey)
	assert.Equal(t, 10, resp.Meta.Pagination.Limit)
	assert.Equal(t, int64(2), resp.Meta.Pagination.Total)
}

func TestListSettingsPagination(t *testing.T) {
	cm := buildConfigMap(t, map[string]SystemSettingItem{
		"example.a": {Name: "A", ConfigKey: "example.a", Value: "1"},
		"example.b": {Name: "B", ConfigKey: "example.b", Value: "2"},
		"example.c": {Name: "C", ConfigKey: "example.c", Value: "3"},
	})

	h, _ := newMockHandler(cm)
	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/api/v1/system-settings?page=2&limit=1")

	h.listSettings(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Items []SystemSettingItem `json:"items"`
		} `json:"data"`
		Meta struct {
			Pagination struct {
				Limit int   `json:"limit"`
				Total int64 `json:"total"`
			} `json:"pagination"`
		} `json:"meta"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	assert.Len(t, resp.Data.Items, 1)
	assert.Equal(t, "example.b", resp.Data.Items[0].ConfigKey)
	assert.Equal(t, 1, resp.Meta.Pagination.Limit)
	assert.Equal(t, int64(3), resp.Meta.Pagination.Total)
}

func TestCreateSettingConflict(t *testing.T) {
	cm := buildConfigMap(t, map[string]SystemSettingItem{
		"example.platformName": {
			Name:      "平台名称",
			ConfigKey: "example.platformName",
			Value:     "容灾平台",
			Remark:    "",
		},
	})

	h, _ := newMockHandler(cm)
	ctx := app.NewContext(16)
	ctx.Set("userName", "admin")

	body := CreateSystemSettingRequest{
		Name:      "平台名称",
		ConfigKey: "example.platformName",
		Value:     "新名称",
		Remark:    "重复 key",
	}
	raw, err := json.Marshal(body)
	assert.NoError(t, err)
	ctx.Request.SetBody(raw)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createSetting(context.Background(), ctx)
	assert.Equal(t, consts.StatusConflict, ctx.Response.StatusCode())
}

func TestUploadAndGetAsset(t *testing.T) {
	cm := buildConfigMap(t, map[string]SystemSettingItem{
		"example.logo": {
			Name:      "Logo",
			ConfigKey: "example.logo",
			Value:     "",
			Remark:    "",
		},
	})
	h, store := newMockHandler(cm)

	assetRaw := []byte{0x89, 0x50, 0x4e, 0x47}

	uploadCtx := app.NewContext(16)
	uploadCtx.Set("userName", "admin")
	uploadCtx.Params = param.Params{
		{Key: "config_key", Value: "example.logo"},
	}
	var uploadBody bytes.Buffer
	writer := multipart.NewWriter(&uploadBody)
	part, err := writer.CreateFormFile("file", "logo.png")
	assert.NoError(t, err)
	_, err = part.Write(assetRaw)
	assert.NoError(t, err)
	err = writer.Close()
	assert.NoError(t, err)
	uploadCtx.Request.SetBody(uploadBody.Bytes())
	uploadCtx.Request.Header.SetContentTypeBytes([]byte(writer.FormDataContentType()))

	h.uploadAsset(context.Background(), uploadCtx)
	assert.Equal(t, consts.StatusOK, uploadCtx.Response.StatusCode())

	cmUpdated, err := store.Get(context.Background(), settingsConfigMapName)
	assert.NoError(t, err)
	doc, err := decodeSettingsDocument(cmUpdated)
	assert.NoError(t, err)
	assert.Contains(t, doc.Items["example.logo"].Value, "data:image/png;base64,")

	downloadCtx := app.NewContext(16)
	downloadCtx.Params = param.Params{
		{Key: "config_key", Value: "example.logo"},
	}

	h.getAsset(context.Background(), downloadCtx)
	assert.Equal(t, consts.StatusOK, downloadCtx.Response.StatusCode())
	assert.Equal(t, assetRaw, downloadCtx.Response.Body())
	assert.True(t, strings.Contains(string(downloadCtx.Response.Header.ContentType()), "image/png"))
}

func TestGetAssetInvalidDataURL(t *testing.T) {
	cm := buildConfigMap(t, map[string]SystemSettingItem{
		"example.logo": {
			Name:      "Logo",
			ConfigKey: "example.logo",
			Value:     "not-a-data-url",
			Remark:    "",
		},
	})
	h, _ := newMockHandler(cm)

	ctx := app.NewContext(16)
	ctx.Params = param.Params{
		{Key: "config_key", Value: "example.logo"},
	}

	h.getAsset(context.Background(), ctx)
	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())

	var resp struct {
		Code int `json:"code"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeBadRequest, resp.Code)
}
