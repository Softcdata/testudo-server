package businessdefaultconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func newMockHandler(configMap *corev1.ConfigMap) (*Handler, *memoryConfigMapStore) {
	store := newMemoryConfigMapStore()
	if configMap != nil {
		if _, err := store.Create(context.Background(), configMap); err != nil {
			panic(err)
		}
	}
	return &Handler{store: store}, store
}

func buildConfigMap(t *testing.T, values map[string]interface{}) *corev1.ConfigMap {
	t.Helper()
	doc := configDocument{
		SchemaVersion: configSchemaVersion,
		Values:        values,
		UpdatedAt:     "2026-07-03T12:00:00Z",
		UpdatedBy:     "admin",
	}
	raw, err := json.Marshal(doc)
	require.NoError(t, err)
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: common.DisasterSystemNamespace,
		},
		Data: map[string]string{
			configDataKey: string(raw),
		},
	}
}

func TestGetSnapshotReturnsGroupedDefaultsAndDescriptions(t *testing.T) {
	h, _ := newMockHandler(nil)
	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/api/v1/business-default-config")

	h.getSnapshot(context.Background(), ctx)

	require.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	var resp struct {
		Code int         `json:"code"`
		Data SnapshotDTO `json:"data"`
	}
	require.NoError(t, json.Unmarshal(ctx.Response.Body(), &resp))
	require.Equal(t, transport.CodeOK, resp.Code)
	require.Len(t, resp.Data.Groups, len(groupDefinitions))

	field := findSnapshotField(t, resp.Data, "backupRuntime.pollInterval")
	assert.Equal(t, "备份状态轮询间隔", field.Name)
	assert.NotEmpty(t, field.Description)
	assert.Equal(t, "10s", field.Value)
	assert.Equal(t, "10s", field.DefaultValue)
	assert.Equal(t, FieldValueTypeDuration, field.DataType)
	assert.Equal(t, "1s", field.Min)
	assert.Equal(t, "5m", field.Max)
}

func TestListFieldsSupportsKeywordFilterPaginationAndLinks(t *testing.T) {
	h, _ := newMockHandler(buildConfigMap(t, map[string]interface{}{
		"restoreRuntime.retryBackoff": "30s",
	}))
	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/api/v1/business-default-config/fields?keyword=timeout&limit=-1")

	h.listFields(context.Background(), ctx)

	require.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Items []FieldDTO `json:"items"`
		} `json:"data"`
		Meta struct {
			Links      map[string]string `json:"links"`
			Pagination struct {
				Limit int   `json:"limit"`
				Total int64 `json:"total"`
			} `json:"pagination"`
		} `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(ctx.Response.Body(), &resp))
	require.Equal(t, transport.CodeOK, resp.Code)
	assert.Greater(t, resp.Meta.Pagination.Total, int64(0))
	assert.Equal(t, -1, resp.Meta.Pagination.Limit)
	assert.Contains(t, resp.Meta.Links["self"], "keyword=timeout")
	for _, item := range resp.Data.Items {
		searchText := strings.ToLower(item.Key + " " + item.Name + " " + item.Description + " " + item.GroupKey + " " + item.GroupName)
		assert.Contains(t, searchText, "timeout")
	}
}

func TestListFieldsSupportsGroupAndEditableFilters(t *testing.T) {
	h, _ := newMockHandler(nil)
	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/api/v1/business-default-config/fields?groupKey=backupRuntime&editable=true&limit=-1")

	h.listFields(context.Background(), ctx)

	require.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	var resp struct {
		Data struct {
			Items []FieldDTO `json:"items"`
		} `json:"data"`
		Meta struct {
			Pagination struct {
				Total int64 `json:"total"`
			} `json:"pagination"`
		} `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(ctx.Response.Body(), &resp))
	assert.Equal(t, int64(3), resp.Meta.Pagination.Total)
	require.Len(t, resp.Data.Items, 3)
	for _, item := range resp.Data.Items {
		assert.Equal(t, "backupRuntime", item.GroupKey)
		assert.True(t, item.Editable)
	}
}

func TestListFrontendSpecFieldsSupportsKeywordSearch(t *testing.T) {
	h, _ := newMockHandler(nil)
	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/api/v1/business-default-config/frontend-fields?keyword=timeout&limit=-1")

	h.listFrontendSpecFields(context.Background(), ctx)

	require.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Items []FrontendSpecFieldDTO `json:"items"`
		} `json:"data"`
		Meta struct {
			Links      map[string]string `json:"links"`
			Pagination struct {
				Limit int   `json:"limit"`
				Total int64 `json:"total"`
			} `json:"pagination"`
		} `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(ctx.Response.Body(), &resp))
	require.Equal(t, transport.CodeOK, resp.Code)
	assert.Equal(t, -1, resp.Meta.Pagination.Limit)
	assert.Contains(t, resp.Meta.Links["self"], "keyword=timeout")

	keys := frontendSpecFieldKeys(resp.Data.Items)
	assert.Contains(t, keys, "backup.timeout")
	assert.Contains(t, keys, "restore.timeout")
	assert.Contains(t, keys, "operation.timeoutMinutes")
	assert.Contains(t, keys, "instance.operationTimeoutMinutes")
	for _, item := range resp.Data.Items {
		searchText := strings.ToLower(item.Key + " " + fmt.Sprint(item.Value))
		assert.Contains(t, searchText, "timeout")
	}
}

func TestListFrontendSpecFieldsReturnsFieldMapWithHierarchyAndAPIUsages(t *testing.T) {
	h, _ := newMockHandler(nil)
	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/api/v1/business-default-config/frontend-fields?keyword=config.timeoutMinutes&limit=-1")

	h.listFrontendSpecFields(context.Background(), ctx)

	require.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	var resp struct {
		Data struct {
			Items    []FrontendSpecFieldDTO          `json:"items"`
			FieldMap map[string]FrontendSpecFieldDTO `json:"fieldMap"`
		} `json:"data"`
		Meta struct {
			Pagination struct {
				Total int64 `json:"total"`
			} `json:"pagination"`
		} `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(ctx.Response.Body(), &resp))
	require.Equal(t, int64(1), resp.Meta.Pagination.Total)
	require.Len(t, resp.Data.Items, 1)

	item := resp.Data.Items[0]
	assert.Equal(t, "operation.timeoutMinutes", item.Key)
	assert.Equal(t, float64(60), item.Value)
	assert.Equal(t, "config.timeoutMinutes", item.RequestPath)
	assert.Equal(t, []string{"operation", "timeoutMinutes"}, item.KeySegments)
	assert.Equal(t, []string{"config", "timeoutMinutes"}, item.RequestPathSegments)
	assert.Equal(t, []string{"spec", "timeoutMinutes"}, item.SpecPathSegments)
	require.NotEmpty(t, item.APIUsages)
	assert.Contains(t, apiUsagePaths(item.APIUsages), "/apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/actions")
	assert.Contains(t, apiUsagePaths(item.APIUsages), "/apis/disastergroups.testudo.softcdata.com/v1/groups/:name/actions")

	field, ok := resp.Data.FieldMap["operation.timeoutMinutes"]
	require.True(t, ok)
	assert.Equal(t, item.Key, field.Key)
	assert.Equal(t, item.RequestPathSegments, field.RequestPathSegments)
	assert.Equal(t, item.APIUsages, field.APIUsages)
}

func TestListFrontendSpecFieldsIgnoresUnsupportedFilters(t *testing.T) {
	h, _ := newMockHandler(nil)
	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/api/v1/business-default-config/frontend-fields?resourceKind=DisasterOperation&serverSupported=true&operation=failover&q=timeout&limit=-1")

	h.listFrontendSpecFields(context.Background(), ctx)

	require.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	var resp struct {
		Data struct {
			Items []FrontendSpecFieldDTO `json:"items"`
		} `json:"data"`
		Meta struct {
			Filters map[string]interface{} `json:"filters"`
		} `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(ctx.Response.Body(), &resp))

	keys := frontendSpecFieldKeys(resp.Data.Items)
	assert.Contains(t, keys, "backup.timeout")
	assert.Contains(t, keys, "restore.timeout")
	assert.Contains(t, keys, "operation.timeoutMinutes")
	assert.Contains(t, keys, "instance.operationTimeoutMinutes")
	assert.Equal(t, map[string]interface{}{"q": "timeout"}, resp.Meta.Filters)
}

func TestPatchConfigMergesAndPersistsValues(t *testing.T) {
	h, store := newMockHandler(nil)
	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/api/v1/business-default-config")
	ctx.Set("userName", "admin")
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	ctx.Request.SetBody([]byte(`{
		"backupRuntime": {"pollInterval": "30s"},
		"operationRuntime.defaultTimeoutMinutes": 120
	}`))

	h.patchConfig(context.Background(), ctx)

	require.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	var resp struct {
		Code int         `json:"code"`
		Data SnapshotDTO `json:"data"`
	}
	require.NoError(t, json.Unmarshal(ctx.Response.Body(), &resp))
	require.Equal(t, transport.CodeOK, resp.Code)
	assert.Equal(t, "30s", findSnapshotField(t, resp.Data, "backupRuntime.pollInterval").Value)
	assert.Equal(t, float64(120), findSnapshotField(t, resp.Data, "operationRuntime.defaultTimeoutMinutes").Value)
	assert.NotEmpty(t, resp.Data.UpdatedAt)
	assert.Equal(t, "admin", resp.Data.UpdatedBy)

	cm, err := store.Get(context.Background(), configMapName)
	require.NoError(t, err)
	doc, err := decodeConfigDocument(cm)
	require.NoError(t, err)
	assert.Equal(t, "30s", doc.Values["backupRuntime.pollInterval"])
	assert.Equal(t, 120, doc.Values["operationRuntime.defaultTimeoutMinutes"])
}

func TestPatchConfigRejectsInvalidDuration(t *testing.T) {
	h, _ := newMockHandler(nil)
	ctx := patchContext(`{"restoreRuntime": {"retryBackoff": "0s"}}`)

	h.patchConfig(context.Background(), ctx)

	require.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	var resp struct {
		Code int            `json:"code"`
		Meta FieldErrorMeta `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(ctx.Response.Body(), &resp))
	assert.Equal(t, transport.CodeBadRequest, resp.Code)
	assert.Equal(t, "restoreRuntime.retryBackoff", resp.Meta.Field)
	assert.Equal(t, "1s", resp.Meta.Min)
}

func TestPatchConfigRejectsReadOnlyField(t *testing.T) {
	h, _ := newMockHandler(nil)
	ctx := patchContext(`{"clusterRuntime": {"veleroZombieLockThreshold": "15m"}}`)

	h.patchConfig(context.Background(), ctx)

	require.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	var resp struct {
		Meta FieldErrorMeta `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(ctx.Response.Body(), &resp))
	assert.Equal(t, "clusterRuntime.veleroZombieLockThreshold", resp.Meta.Field)
	assert.Equal(t, "field is read-only", resp.Meta.Reason)
}

func TestPatchConfigRejectsCrossFieldInvalidValues(t *testing.T) {
	h, _ := newMockHandler(nil)
	ctx := patchContext(`{
		"instanceRuntime": {
			"transitionWatchdogTimeout": "30s",
			"minTransitionWatchdogTimeout": "1m"
		}
	}`)

	h.patchConfig(context.Background(), ctx)

	require.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	var resp struct {
		Meta FieldErrorMeta `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(ctx.Response.Body(), &resp))
	assert.Equal(t, "instanceRuntime.transitionWatchdogTimeout", resp.Meta.Field)
	assert.Contains(t, resp.Meta.Reason, "greater than or equal")
}

func patchContext(body string) *app.RequestContext {
	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/api/v1/business-default-config")
	ctx.Set("userName", "admin")
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	ctx.Request.SetBody([]byte(body))
	return ctx
}

func findSnapshotField(t *testing.T, snapshot SnapshotDTO, key string) FieldDTO {
	t.Helper()
	for _, group := range snapshot.Groups {
		for _, field := range group.Fields {
			if field.Key == key {
				return field
			}
		}
	}
	t.Fatalf("field %s not found", key)
	return FieldDTO{}
}

func frontendSpecFieldKeys(items []FrontendSpecFieldDTO) []string {
	keys := make([]string, 0, len(items))
	for _, item := range items {
		keys = append(keys, item.Key)
	}
	return keys
}

func apiUsagePaths(usages []FrontendSpecAPIUsageDTO) []string {
	paths := make([]string, 0, len(usages))
	for _, usage := range usages {
		paths = append(paths, usage.Path)
	}
	return paths
}
