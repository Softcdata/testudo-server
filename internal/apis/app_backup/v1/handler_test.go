package appbackup

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route/param"
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-operator/pkg/clientset/versioned/fake"
	informers "github.com/softcdata/testudo-operator/pkg/informers/externalversions"
	"github.com/softcdata/testudo-operator/pkg/metadata"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/kube"
	"github.com/softcdata/testudo-server/internal/storage"
	transport "github.com/softcdata/testudo-server/internal/transport"
	"github.com/stretchr/testify/assert"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

type MockStorage struct {
	MockGetDownloadURL func(ctx context.Context, endpoint, accessKey, secretKey, bucket, region string, addressingStyle dapisv1.StorageRepositoryAddressingStyle, caBundle []byte, objectKey string, expiry time.Duration) (string, error)
	MockListObjects    func(ctx context.Context, endpoint, accessKey, secretKey, bucket, region string, addressingStyle dapisv1.StorageRepositoryAddressingStyle, caBundle []byte, prefixes []string) ([]storage.ObjectInfo, error)
	MockGetObject      func(ctx context.Context, endpoint, accessKey, secretKey, bucket, region string, addressingStyle dapisv1.StorageRepositoryAddressingStyle, caBundle []byte, objectKey, rangeHeader string) (*storage.ObjectStream, error)
}

func (m *MockStorage) ListObjects(ctx context.Context, endpoint, accessKey, secretKey, bucket, region string, addressingStyle dapisv1.StorageRepositoryAddressingStyle, caBundle []byte, prefixes []string) ([]storage.ObjectInfo, error) {
	if m.MockListObjects != nil {
		return m.MockListObjects(ctx, endpoint, accessKey, secretKey, bucket, region, addressingStyle, caBundle, prefixes)
	}
	return nil, nil
}

func (m *MockStorage) GetDownloadURL(ctx context.Context, endpoint, accessKey, secretKey, bucket, region string, addressingStyle dapisv1.StorageRepositoryAddressingStyle, caBundle []byte, objectKey string, expiry time.Duration) (string, error) {
	if m.MockGetDownloadURL != nil {
		return m.MockGetDownloadURL(ctx, endpoint, accessKey, secretKey, bucket, region, addressingStyle, caBundle, objectKey, expiry)
	}
	return "http://mock-download-url", nil
}

func (m *MockStorage) GetObject(ctx context.Context, endpoint, accessKey, secretKey, bucket, region string, addressingStyle dapisv1.StorageRepositoryAddressingStyle, caBundle []byte, objectKey, rangeHeader string) (*storage.ObjectStream, error) {
	if m.MockGetObject != nil {
		return m.MockGetObject(ctx, endpoint, accessKey, secretKey, bucket, region, addressingStyle, caBundle, objectKey, rangeHeader)
	}
	return &storage.ObjectStream{
		Body:          io.NopCloser(strings.NewReader("mock")),
		ContentLength: int64(len("mock")),
		Size:          int64(len("mock")),
		ContentType:   "application/octet-stream",
	}, nil
}

func newMockHandler(objects ...runtime.Object) (*AppBackupHandler, *MockStorage) {
	fakeClient := fake.NewSimpleClientset(objects...)
	informerFactory := informers.NewSharedInformerFactory(fakeClient, 0)

	// Pre-initialize informer so it starts syncing
	_ = informerFactory.Disaster().V1().AppBackups().Informer()

	stopCh := make(chan struct{})
	informerFactory.Start(stopCh)
	informerFactory.WaitForCacheSync(stopCh)

	kc := &kube.KubeClient{
		K8sClient:       k8sfake.NewSimpleClientset(),
		DisasterClient:  fakeClient,
		InformerFactory: informerFactory,
	}

	h := server.Default()
	rg := h.Group("/v1")

	mockStorage := &MockStorage{}
	return NewAppBackupHandler(kc, rg, mockStorage), mockStorage
}

func TestNewAppBackupHandler(t *testing.T) {
	handler, _ := newMockHandler()
	assert.NotNil(t, handler)
}

func TestAppBackups_List(t *testing.T) {
	backup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-backup",
			Namespace: common.DisasterSystemNamespace,
		},
	}
	h, _ := newMockHandler(backup)

	ctx := app.NewContext(16)
	h.appBackups(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	type ListResponse struct {
		Code int `json:"code"`
		Data struct {
			Items []AppBackupDTO `json:"items"`
		} `json:"data"`
	}

	var resp ListResponse
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp.Data.Items, 1)
	assert.Equal(t, "test-backup", resp.Data.Items[0].Name)
	// Verify History is empty/nil in list view logic (though DTO definition has it omitempty, if it's empty slice or nil)
	assert.Empty(t, resp.Data.Items[0].Status.History)
}

func TestAppBackups_ListOriginFilter(t *testing.T) {
	controllerTrue := true
	userBackup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "user-backup",
			Namespace: common.DisasterSystemNamespace,
		},
	}
	instanceBackup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "instance-backup",
			Namespace: common.DisasterSystemNamespace,
			Labels: map[string]string{
				appResourceOriginLabelKey: appResourceOriginDisasterInstance,
			},
		},
	}
	instanceBackupNoLabel := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "instance-backup-no-label",
			Namespace: common.DisasterSystemNamespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:       "ResourceSync",
					Name:       "rs-1",
					Controller: &controllerTrue,
				},
			},
		},
	}
	h, _ := newMockHandler(userBackup, instanceBackup, instanceBackupNoLabel)

	type ListResponse struct {
		Code int `json:"code"`
		Data struct {
			Items []AppBackupDTO `json:"items"`
		} `json:"data"`
	}

	testCases := []struct {
		name      string
		uri       string
		wantNames []string
	}{
		{
			name:      "default user filter",
			uri:       "/appbackups",
			wantNames: []string{"user-backup"},
		},
		{
			name:      "origin all",
			uri:       "/appbackups?origin=all",
			wantNames: []string{"user-backup", "instance-backup", "instance-backup-no-label"},
		},
		{
			name:      "origin instance",
			uri:       "/appbackups?origin=instance",
			wantNames: []string{"instance-backup", "instance-backup-no-label"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := app.NewContext(16)
			ctx.Request.SetRequestURI(tc.uri)

			h.appBackups(context.Background(), ctx)
			assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

			var resp ListResponse
			err := json.Unmarshal(ctx.Response.Body(), &resp)
			assert.NoError(t, err)

			gotNames := make([]string, 0, len(resp.Data.Items))
			for _, item := range resp.Data.Items {
				gotNames = append(gotNames, item.Name)
			}
			assert.ElementsMatch(t, tc.wantNames, gotNames)
		})
	}
}

func TestAppBackup_Get(t *testing.T) {
	backup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-backup",
			Namespace: common.DisasterSystemNamespace,
		},
	}
	h, _ := newMockHandler(backup)

	ctx := app.NewContext(16)
	ctx.Params = param.Params{
		{Key: "name", Value: "test-backup"},
	}

	h.appBackup(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	type GetResponse struct {
		Code int          `json:"code"`
		Data AppBackupDTO `json:"data"`
	}
	var resp GetResponse
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "test-backup", resp.Data.Name)
}

func TestCreateAppBackup(t *testing.T) {
	cluster := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-cluster",
		},
		Status: dapisv1.ClusterStatus{
			Status: "Ready",
		},
	}
	repo := &dapisv1.StorageRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-storage",
			Namespace: common.DisasterSystemNamespace,
		},
		Status: dapisv1.StorageRepositoryStatus{
			Status: "Available",
		},
	}
	h, _ := newMockHandler(cluster, repo)

	ctx := app.NewContext(16)
	req := CreateAppBackupRequest{
		Name:            "new-backup",
		Cluster:         "test-cluster",
		Schedule:        "@daily",
		StorageLocation: "test-storage",
	}
	body, _ := json.Marshal(req)
	t.Logf("Request Body: %s", string(body))
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createAppBackup(context.Background(), ctx)

	if ctx.Response.StatusCode() != consts.StatusCreated {
		t.Logf("Create failed: %s", string(ctx.Response.Body()))
	}
	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())

	var resp struct {
		Code int          `json:"code"`
		Data AppBackupDTO `json:"data"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	if resp.Data.Name == "" {
		t.Logf("Response body: %s", string(ctx.Response.Body()))
	}
	assert.Equal(t, "new-backup", resp.Data.Name)
}

func TestUpdateAppBackup(t *testing.T) {
	backup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-backup",
			Namespace: common.DisasterSystemNamespace,
		},
	}
	h, _ := newMockHandler(backup)

	ctx := app.NewContext(16)
	req := UpdateAppBackupRequest{
		Name:    "test-backup",
		Cluster: "updated-cluster",
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.updateAppBackup(context.Background(), ctx)

	if ctx.Response.StatusCode() != consts.StatusOK {
		t.Logf("Update failed: %s", string(ctx.Response.Body()))
	}
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp struct {
		Code int          `json:"code"`
		Data AppBackupDTO `json:"data"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "updated-cluster", resp.Data.Spec.Cluster)
}

func TestDeleteAppBackup(t *testing.T) {
	backup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-backup",
			Namespace: common.DisasterSystemNamespace,
		},
	}
	h, _ := newMockHandler(backup)

	ctx := app.NewContext(16)
	ctx.Params = param.Params{
		{Key: "name", Value: "test-backup"},
	}

	h.deleteAppBackup(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	// Verify deletion
	actions := h.KubeClient.DisasterClient.(*fake.Clientset).Actions()
	foundDelete := false
	for _, action := range actions {
		if action.GetVerb() == "delete" && action.GetResource().Resource == "appbackups" {
			foundDelete = true
			break
		}
	}
	assert.True(t, foundDelete)
}

func TestDownloadBackup(t *testing.T) {
	backup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-backup",
			Namespace: common.DisasterSystemNamespace,
		},
		Spec: dapisv1.AppBackupSpec{
			Cluster: "test-cluster",
			Template: velerov1.BackupSpec{
				StorageLocation: "test-storage",
			},
		},
		Status: dapisv1.AppBackupStatus{
			History: []dapisv1.BackupRecord{
				{Name: "velero-backup-1"},
			},
		},
	}
	h, _ := newMockHandler(backup)

	// Mock Storage Repository fetching logic is missing in newMockHandler because it only mocks DisasterClient.
	// But downloadBackup needs to fetch StorageRepository (Disaster resource).
	// So we need to add StorageRepository to newMockHandler objects or fake client.
	// Wait, StorageRepository is also a disaster resource, so adding it to objects works.
	storageRepo := &dapisv1.StorageRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-storage",
			Namespace: common.DisasterSystemNamespace,
		},
		Spec: dapisv1.StorageRepositorySpec{
			Endpoint:  "http://minio",
			AccessKey: "minio",
			SecretKey: "minio123",
			Bucket:    "backups",
			Region:    "us-east-1",
		},
	}
	// Re-init with both objects
	h, _ = newMockHandler(backup, storageRepo)

	ctx := app.NewContext(16)
	ctx.Params = param.Params{
		{Key: "name", Value: "test-backup"},
		{Key: "backupName", Value: "velero-backup-1"},
	}

	h.downloadBackup(context.Background(), ctx)

	t.Logf("Response Body: %s", string(ctx.Response.Body()))

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	var resp struct {
		Data BackupDownloadResponse `json:"data"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Contains(t, resp.Data.DownloadURL, "/appbackups/test-backup/backups/velero-backup-1/download/stream?downloadToken=")
	assert.NotContains(t, resp.Data.DownloadURL, "http://minio")
	assert.Equal(t, "proxy", resp.Data.Mode)
	assert.Equal(t, "resource", resp.Data.Type)
	assert.Equal(t, "velero-backup-1.tar.gz", resp.Data.FileName)
}

func TestDownloadBackupStream_Resource(t *testing.T) {
	backup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-backup",
			Namespace: common.DisasterSystemNamespace,
			UID:       "appbackup-uid",
		},
		Spec: dapisv1.AppBackupSpec{
			Cluster: "test-cluster",
			Template: velerov1.BackupSpec{
				StorageLocation: "test-storage",
			},
		},
		Status: dapisv1.AppBackupStatus{
			History: []dapisv1.BackupRecord{
				{Name: "velero-backup-1"},
			},
		},
	}
	storageRepo := &dapisv1.StorageRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-storage",
			Namespace: common.DisasterSystemNamespace,
			UID:       "repo-uid",
		},
		Spec: dapisv1.StorageRepositorySpec{
			Endpoint:  "http://minio",
			AccessKey: "minio",
			SecretKey: "minio123",
			Bucket:    "backups",
			Region:    "us-east-1",
		},
	}
	h, mockStorage := newMockHandler(backup, storageRepo)

	token, err := h.signBackupDownloadToken(backup, storageRepo, "velero-backup-1", "resource", time.Now().Add(time.Hour), "tester")
	assert.NoError(t, err)

	ctx := app.NewContext(16)
	ctx.Params = param.Params{
		{Key: "name", Value: "test-backup"},
		{Key: "backupName", Value: "velero-backup-1"},
	}
	ctx.Request.URI().QueryArgs().Set("downloadToken", token)

	calledGetObject := false
	mockStorage.MockGetObject = func(ctx context.Context, endpoint, accessKey, secretKey, bucket, region string, addressingStyle dapisv1.StorageRepositoryAddressingStyle, caBundle []byte, objectKey, rangeHeader string) (*storage.ObjectStream, error) {
		calledGetObject = true
		assert.Equal(t, "test-cluster/backups/velero-backup-1/velero-backup-1.tar.gz", objectKey)
		return &storage.ObjectStream{
			Body:          io.NopCloser(strings.NewReader("backup-data")),
			ContentLength: int64(len("backup-data")),
			Size:          int64(len("backup-data")),
			ContentType:   "application/gzip",
		}, nil
	}

	h.downloadBackupStream(context.Background(), ctx)

	assert.True(t, calledGetObject)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	assert.Equal(t, "application/gzip", string(ctx.Response.Header.ContentType()))
	assert.Contains(t, string(ctx.Response.Header.Peek("Content-Disposition")), "velero-backup-1.tar.gz")
	body, err := io.ReadAll(ctx.Response.BodyStream())
	assert.NoError(t, err)
	assert.Equal(t, "backup-data", string(body))
}

func TestDownloadBackup_HistoryNotFound(t *testing.T) {
	backup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-backup",
			Namespace: common.DisasterSystemNamespace,
		},
		Spec: dapisv1.AppBackupSpec{
			Cluster: "test-cluster",
			Template: velerov1.BackupSpec{
				StorageLocation: "test-storage",
			},
		},
		Status: dapisv1.AppBackupStatus{
			History: []dapisv1.BackupRecord{
				{Name: "other-backup"},
			},
		},
	}
	storageRepo := &dapisv1.StorageRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-storage",
			Namespace: common.DisasterSystemNamespace,
		},
		Spec: dapisv1.StorageRepositorySpec{
			Endpoint: "http://minio",
			Bucket:   "backups",
		},
	}
	h, mockStorage := newMockHandler(backup, storageRepo)

	calledGetObject := false
	mockStorage.MockGetObject = func(ctx context.Context, endpoint, accessKey, secretKey, bucket, region string, addressingStyle dapisv1.StorageRepositoryAddressingStyle, caBundle []byte, objectKey, rangeHeader string) (*storage.ObjectStream, error) {
		calledGetObject = true
		return nil, nil
	}

	ctx := app.NewContext(16)
	ctx.Params = param.Params{
		{Key: "name", Value: "test-backup"},
		{Key: "backupName", Value: "missing-backup"},
	}

	h.downloadBackup(context.Background(), ctx)

	assert.Equal(t, consts.StatusNotFound, ctx.Response.StatusCode())
	assert.False(t, calledGetObject)
}

func TestDownloadBackupStream_InvalidOrExpiredToken(t *testing.T) {
	backup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-backup",
			Namespace: common.DisasterSystemNamespace,
			UID:       "appbackup-uid",
		},
		Spec: dapisv1.AppBackupSpec{
			Cluster: "test-cluster",
			Template: velerov1.BackupSpec{
				StorageLocation: "test-storage",
			},
		},
		Status: dapisv1.AppBackupStatus{
			History: []dapisv1.BackupRecord{
				{Name: "velero-backup-1"},
			},
		},
	}
	storageRepo := &dapisv1.StorageRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-storage",
			Namespace: common.DisasterSystemNamespace,
			UID:       "repo-uid",
		},
		Spec: dapisv1.StorageRepositorySpec{
			Endpoint: "http://minio",
			Bucket:   "backups",
		},
	}
	h, mockStorage := newMockHandler(backup, storageRepo)
	expiredToken, err := h.signBackupDownloadToken(backup, storageRepo, "velero-backup-1", "resource", time.Now().Add(-time.Hour), "tester")
	assert.NoError(t, err)

	tests := []struct {
		name  string
		token string
	}{
		{name: "tampered", token: expiredToken + "tampered"},
		{name: "expired", token: expiredToken},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calledGetObject := false
			mockStorage.MockGetObject = func(ctx context.Context, endpoint, accessKey, secretKey, bucket, region string, addressingStyle dapisv1.StorageRepositoryAddressingStyle, caBundle []byte, objectKey, rangeHeader string) (*storage.ObjectStream, error) {
				calledGetObject = true
				return nil, nil
			}

			ctx := app.NewContext(16)
			ctx.Params = param.Params{
				{Key: "name", Value: "test-backup"},
				{Key: "backupName", Value: "velero-backup-1"},
			}
			ctx.Request.URI().QueryArgs().Set("downloadToken", tt.token)

			h.downloadBackupStream(context.Background(), ctx)

			assert.Equal(t, consts.StatusForbidden, ctx.Response.StatusCode())
			assert.False(t, calledGetObject)
		})
	}
}

func TestDownloadBackupStream_ObjectStoreFailure(t *testing.T) {
	backup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-backup",
			Namespace: common.DisasterSystemNamespace,
			UID:       "appbackup-uid",
		},
		Spec: dapisv1.AppBackupSpec{
			Cluster: "test-cluster",
			Template: velerov1.BackupSpec{
				StorageLocation: "test-storage",
			},
		},
		Status: dapisv1.AppBackupStatus{
			History: []dapisv1.BackupRecord{
				{Name: "velero-backup-1"},
			},
		},
	}
	storageRepo := &dapisv1.StorageRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-storage",
			Namespace: common.DisasterSystemNamespace,
			UID:       "repo-uid",
		},
		Spec: dapisv1.StorageRepositorySpec{
			Endpoint: "http://minio",
			Bucket:   "backups",
		},
	}
	h, mockStorage := newMockHandler(backup, storageRepo)
	token, err := h.signBackupDownloadToken(backup, storageRepo, "velero-backup-1", "resource", time.Now().Add(time.Hour), "tester")
	assert.NoError(t, err)

	mockStorage.MockGetObject = func(ctx context.Context, endpoint, accessKey, secretKey, bucket, region string, addressingStyle dapisv1.StorageRepositoryAddressingStyle, caBundle []byte, objectKey, rangeHeader string) (*storage.ObjectStream, error) {
		return nil, errors.New("object storage unavailable")
	}

	ctx := app.NewContext(16)
	ctx.Params = param.Params{
		{Key: "name", Value: "test-backup"},
		{Key: "backupName", Value: "velero-backup-1"},
	}
	ctx.Request.URI().QueryArgs().Set("downloadToken", token)

	h.downloadBackupStream(context.Background(), ctx)

	assert.Equal(t, consts.StatusBadGateway, ctx.Response.StatusCode())
	var resp struct {
		Code int `json:"code"`
	}
	err = json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeUpstreamError, resp.Code)
}

func TestExecuteAction_Delete_WithTarget(t *testing.T) {
	backup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-backup",
			Namespace: common.DisasterSystemNamespace,
		},
	}
	h, _ := newMockHandler(backup)

	ctx := app.NewContext(16)
	ctx.Params = param.Params{
		{Key: "name", Value: "test-backup"},
		{Key: "type", Value: "delete"},
	}

	req := AppBackupActionRequest{
		TargetBackup: "velero-backup-1",
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.executeAction(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	// Verify Update happened
	updatedBackup, err := h.DisasterClient.DisasterV1().AppBackups(common.DisasterSystemNamespace).Get(context.Background(), "test-backup", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, updatedBackup.Spec.Action)
	assert.Equal(t, "Delete", updatedBackup.Spec.Action.Type)
	assert.Equal(t, "velero-backup-1", updatedBackup.Spec.Action.TargetBackup)
}

func TestExecuteAction_PauseRecordsManualPauseIntent(t *testing.T) {
	backup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-backup",
			Namespace: common.DisasterSystemNamespace,
		},
	}
	h, _ := newMockHandler(backup)

	ctx := app.NewContext(16)
	ctx.Params = param.Params{
		{Key: "name", Value: "test-backup"},
		{Key: "type", Value: "pause"},
	}

	h.executeAction(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	updatedBackup, err := h.DisasterClient.DisasterV1().AppBackups(common.DisasterSystemNamespace).Get(context.Background(), "test-backup", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.True(t, updatedBackup.Spec.Paused)
	assert.Equal(t, "true", updatedBackup.Annotations[metadata.AnnotationAppBackupManualPaused])
}

func TestExecuteAction_ResumeRecordsManualResumeIntent(t *testing.T) {
	backup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-backup",
			Namespace: common.DisasterSystemNamespace,
			Annotations: map[string]string{
				metadata.AnnotationAppBackupManualPaused: "true",
			},
		},
		Spec: dapisv1.AppBackupSpec{
			Paused: true,
		},
	}
	h, _ := newMockHandler(backup)

	ctx := app.NewContext(16)
	ctx.Params = param.Params{
		{Key: "name", Value: "test-backup"},
		{Key: "type", Value: "resume"},
	}

	h.executeAction(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	updatedBackup, err := h.DisasterClient.DisasterV1().AppBackups(common.DisasterSystemNamespace).Get(context.Background(), "test-backup", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.False(t, updatedBackup.Spec.Paused)
	assert.Equal(t, "false", updatedBackup.Annotations[metadata.AnnotationAppBackupManualPaused])
}

func TestWatchAppBackups(t *testing.T) {
	// Basic test to ensure method exists and runs without panic
	h, _ := newMockHandler()
	// ctx := app.NewContext(16)

	// Since StreamWatch blocks, we run it in a goroutine or just skip execution in unit test
	// For now, we just assert the handler is not nil
	assert.NotNil(t, h)
}

func TestDownloadBackup_Data(t *testing.T) {
	backup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-backup",
			Namespace: common.DisasterSystemNamespace,
		},
		Spec: dapisv1.AppBackupSpec{
			Cluster: "test-cluster",
			Template: velerov1.BackupSpec{
				StorageLocation:    "test-storage",
				IncludedNamespaces: []string{"ns1", "ns2"},
			},
		},
		Status: dapisv1.AppBackupStatus{
			History: []dapisv1.BackupRecord{
				{Name: "velero-backup-1"},
			},
		},
	}
	storageRepo := &dapisv1.StorageRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-storage",
			Namespace: common.DisasterSystemNamespace,
		},
		Spec: dapisv1.StorageRepositorySpec{
			Endpoint:  "http://minio",
			AccessKey: "minio",
			SecretKey: "minio123",
			Bucket:    "backups",
			Region:    "us-east-1",
		},
	}

	h, mockStorage := newMockHandler(backup, storageRepo)

	ctx := app.NewContext(16)
	ctx.Params = param.Params{
		{Key: "name", Value: "test-backup"},
		{Key: "backupName", Value: "velero-backup-1"},
	}
	ctx.Request.URI().QueryArgs().Set("type", "data")

	calledList := false
	mockStorage.MockListObjects = func(ctx context.Context, endpoint, accessKey, secretKey, bucket, region string, addressingStyle dapisv1.StorageRepositoryAddressingStyle, caBundle []byte, prefixes []string) ([]storage.ObjectInfo, error) {
		calledList = true
		return []storage.ObjectInfo{
			{Key: "test-cluster/kopia/ns1/data.bin", Size: 1024},
			{Key: "test-cluster/restic/ns2/snapshot.dat", Size: 2048},
		}, nil
	}

	h.downloadBackup(context.Background(), ctx)

	assert.False(t, calledList)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp struct {
		Data BackupDownloadResponse `json:"data"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Empty(t, resp.Data.Files)
	assert.Contains(t, resp.Data.DownloadURL, "/appbackups/test-backup/backups/velero-backup-1/download/stream?downloadToken=")
	assert.NotContains(t, resp.Data.DownloadURL, "http://minio")
	assert.Equal(t, "proxy", resp.Data.Mode)
	assert.Equal(t, "data", resp.Data.Type)
	assert.Equal(t, "velero-backup-1-data.tar", resp.Data.FileName)
}

func TestDownloadBackupStream_DataTar(t *testing.T) {
	backup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-backup",
			Namespace: common.DisasterSystemNamespace,
			UID:       "appbackup-uid",
		},
		Spec: dapisv1.AppBackupSpec{
			Cluster: "test-cluster",
			Template: velerov1.BackupSpec{
				StorageLocation:    "test-storage",
				IncludedNamespaces: []string{"ns1"},
			},
		},
		Status: dapisv1.AppBackupStatus{
			History: []dapisv1.BackupRecord{
				{Name: "velero-backup-1"},
			},
		},
	}
	storageRepo := &dapisv1.StorageRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-storage",
			Namespace: common.DisasterSystemNamespace,
			UID:       "repo-uid",
		},
		Spec: dapisv1.StorageRepositorySpec{
			Endpoint:  "http://minio",
			AccessKey: "minio",
			SecretKey: "minio123",
			Bucket:    "backups",
			Region:    "us-east-1",
		},
	}
	h, mockStorage := newMockHandler(backup, storageRepo)
	token, err := h.signBackupDownloadToken(backup, storageRepo, "velero-backup-1", "data", time.Now().Add(time.Hour), "tester")
	assert.NoError(t, err)

	ctx := app.NewContext(16)
	ctx.Params = param.Params{
		{Key: "name", Value: "test-backup"},
		{Key: "backupName", Value: "velero-backup-1"},
	}
	ctx.Request.URI().QueryArgs().Set("downloadToken", token)

	mockStorage.MockListObjects = func(ctx context.Context, endpoint, accessKey, secretKey, bucket, region string, addressingStyle dapisv1.StorageRepositoryAddressingStyle, caBundle []byte, prefixes []string) ([]storage.ObjectInfo, error) {
		assert.Contains(t, prefixes, "test-cluster/kopia/ns1/")
		assert.Contains(t, prefixes, "test-cluster/restic/ns1/")
		assert.NotContains(t, prefixes, "test-cluster/backups/velero-backup-1/")
		return []storage.ObjectInfo{
			{Key: "test-cluster/kopia/ns1/data.bin", Size: int64(len("payload"))},
		}, nil
	}
	mockStorage.MockGetObject = func(ctx context.Context, endpoint, accessKey, secretKey, bucket, region string, addressingStyle dapisv1.StorageRepositoryAddressingStyle, caBundle []byte, objectKey, rangeHeader string) (*storage.ObjectStream, error) {
		assert.Equal(t, "test-cluster/kopia/ns1/data.bin", objectKey)
		return &storage.ObjectStream{
			Body:          io.NopCloser(strings.NewReader("payload")),
			ContentLength: int64(len("payload")),
			Size:          int64(len("payload")),
		}, nil
	}

	h.downloadBackupStream(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	assert.Equal(t, "application/x-tar", string(ctx.Response.Header.ContentType()))
	body, err := io.ReadAll(ctx.Response.BodyStream())
	assert.NoError(t, err)
	assert.Contains(t, string(body), "kopia/ns1/data.bin")
	assert.Contains(t, string(body), "payload")
}

func TestDownloadBackupStream_DataEmptyObjects(t *testing.T) {
	backup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-backup",
			Namespace: common.DisasterSystemNamespace,
			UID:       "appbackup-uid",
		},
		Spec: dapisv1.AppBackupSpec{
			Cluster: "test-cluster",
			Template: velerov1.BackupSpec{
				StorageLocation:    "test-storage",
				IncludedNamespaces: []string{"ns1"},
			},
		},
		Status: dapisv1.AppBackupStatus{
			History: []dapisv1.BackupRecord{
				{Name: "velero-backup-1"},
			},
		},
	}
	storageRepo := &dapisv1.StorageRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-storage",
			Namespace: common.DisasterSystemNamespace,
			UID:       "repo-uid",
		},
		Spec: dapisv1.StorageRepositorySpec{
			Endpoint: "http://minio",
			Bucket:   "backups",
		},
	}
	h, mockStorage := newMockHandler(backup, storageRepo)
	token, err := h.signBackupDownloadToken(backup, storageRepo, "velero-backup-1", "data", time.Now().Add(time.Hour), "tester")
	assert.NoError(t, err)

	calledGetObject := false
	mockStorage.MockListObjects = func(ctx context.Context, endpoint, accessKey, secretKey, bucket, region string, addressingStyle dapisv1.StorageRepositoryAddressingStyle, caBundle []byte, prefixes []string) ([]storage.ObjectInfo, error) {
		return nil, nil
	}
	mockStorage.MockGetObject = func(ctx context.Context, endpoint, accessKey, secretKey, bucket, region string, addressingStyle dapisv1.StorageRepositoryAddressingStyle, caBundle []byte, objectKey, rangeHeader string) (*storage.ObjectStream, error) {
		calledGetObject = true
		return nil, nil
	}

	ctx := app.NewContext(16)
	ctx.Params = param.Params{
		{Key: "name", Value: "test-backup"},
		{Key: "backupName", Value: "velero-backup-1"},
	}
	ctx.Request.URI().QueryArgs().Set("downloadToken", token)

	h.downloadBackupStream(context.Background(), ctx)

	assert.Equal(t, consts.StatusNotFound, ctx.Response.StatusCode())
	assert.False(t, calledGetObject)
}

type trackingReadCloser struct {
	reader *strings.Reader
	closed chan struct{}
}

func (r *trackingReadCloser) Read(p []byte) (int, error) {
	return r.reader.Read(p)
}

func (r *trackingReadCloser) Close() error {
	select {
	case <-r.closed:
	default:
		close(r.closed)
	}
	return nil
}

func TestDownloadBackupStream_DataClientCloseClosesObject(t *testing.T) {
	backup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-backup",
			Namespace: common.DisasterSystemNamespace,
			UID:       "appbackup-uid",
		},
		Spec: dapisv1.AppBackupSpec{
			Cluster: "test-cluster",
			Template: velerov1.BackupSpec{
				StorageLocation:    "test-storage",
				IncludedNamespaces: []string{"ns1"},
			},
		},
		Status: dapisv1.AppBackupStatus{
			History: []dapisv1.BackupRecord{
				{Name: "velero-backup-1"},
			},
		},
	}
	storageRepo := &dapisv1.StorageRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-storage",
			Namespace: common.DisasterSystemNamespace,
			UID:       "repo-uid",
		},
		Spec: dapisv1.StorageRepositorySpec{
			Endpoint: "http://minio",
			Bucket:   "backups",
		},
	}
	h, mockStorage := newMockHandler(backup, storageRepo)
	token, err := h.signBackupDownloadToken(backup, storageRepo, "velero-backup-1", "data", time.Now().Add(time.Hour), "tester")
	assert.NoError(t, err)

	bodyClosed := make(chan struct{})
	mockStorage.MockListObjects = func(ctx context.Context, endpoint, accessKey, secretKey, bucket, region string, addressingStyle dapisv1.StorageRepositoryAddressingStyle, caBundle []byte, prefixes []string) ([]storage.ObjectInfo, error) {
		return []storage.ObjectInfo{
			{Key: "test-cluster/kopia/ns1/data.bin", Size: int64(len("payload"))},
		}, nil
	}
	mockStorage.MockGetObject = func(ctx context.Context, endpoint, accessKey, secretKey, bucket, region string, addressingStyle dapisv1.StorageRepositoryAddressingStyle, caBundle []byte, objectKey, rangeHeader string) (*storage.ObjectStream, error) {
		return &storage.ObjectStream{
			Body:          &trackingReadCloser{reader: strings.NewReader("payload"), closed: bodyClosed},
			ContentLength: int64(len("payload")),
			Size:          int64(len("payload")),
		}, nil
	}

	ctx := app.NewContext(16)
	ctx.Params = param.Params{
		{Key: "name", Value: "test-backup"},
		{Key: "backupName", Value: "velero-backup-1"},
	}
	ctx.Request.URI().QueryArgs().Set("downloadToken", token)

	h.downloadBackupStream(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	assert.True(t, ctx.Response.IsBodyStream())
	assert.NoError(t, ctx.Response.CloseBodyStream())
	select {
	case <-bodyClosed:
	case <-time.After(time.Second):
		t.Fatal("expected object body to be closed after client closes stream")
	}
}

func TestCreateAppBackup_WithScopedClusterFields(t *testing.T) {
	cluster := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster-scoped"},
		Status:     dapisv1.ClusterStatus{Status: "Ready"},
	}
	repo := &dapisv1.StorageRepository{
		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-scoped", Namespace: common.DisasterSystemNamespace},
		Status:     dapisv1.StorageRepositoryStatus{Status: "Available"},
	}
	h, _ := newMockHandler(cluster, repo)

	ctx := app.NewContext(16)
	includeClusterResources := false
	req := CreateAppBackupRequest{
		Name:                             "scoped-backup",
		Cluster:                          "test-cluster-scoped",
		Schedule:                         "@daily",
		StorageLocation:                  "test-storage-scoped",
		IncludeClusterResources:          &includeClusterResources,
		IncludedNamespaceScopedResources: []string{"deployments.apps"},
		ExcludedNamespaceScopedResources: []string{"secrets"},
		IncludedClusterScopedResources:   []string{"nodes"},
		ExcludedClusterScopedResources:   []string{"clusterrolebindings"},
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createAppBackup(context.Background(), ctx)
	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())

	var resp struct {
		Code int          `json:"code"`
		Data AppBackupDTO `json:"data"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, []string{"nodes"}, resp.Data.Spec.IncludedClusterScopedResources)
	assert.Equal(t, []string{"clusterrolebindings"}, resp.Data.Spec.ExcludedClusterScopedResources)
	assert.Nil(t, resp.Data.Spec.IncludeClusterResources)

	stored, err := h.DisasterClient.DisasterV1().AppBackups(common.DisasterSystemNamespace).Get(context.Background(), "scoped-backup", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, []string{"nodes"}, stored.Spec.Template.IncludedClusterScopedResources)
	assert.Equal(t, []string{"clusterrolebindings"}, stored.Spec.Template.ExcludedClusterScopedResources)
	assert.Nil(t, stored.Spec.Template.IncludeClusterResources)
}

func TestUpdateAppBackup_WithScopedClusterFields(t *testing.T) {
	includeClusterResources := true
	backup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{Name: "test-backup-scoped-update", Namespace: common.DisasterSystemNamespace},
		Spec: dapisv1.AppBackupSpec{
			Template: velerov1.BackupSpec{
				IncludeClusterResources: &includeClusterResources,
			},
		},
	}
	h, _ := newMockHandler(backup)

	ctx := app.NewContext(16)
	updateIncludeClusterResources := false
	req := UpdateAppBackupRequest{
		Name:                           "test-backup-scoped-update",
		IncludeClusterResources:        &updateIncludeClusterResources,
		IncludedClusterScopedResources: []string{"nodes", "clusterroles"},
		ExcludedClusterScopedResources: []string{"clusterrolebindings"},
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.updateAppBackup(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp struct {
		Code int          `json:"code"`
		Data AppBackupDTO `json:"data"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, []string{"nodes", "clusterroles"}, resp.Data.Spec.IncludedClusterScopedResources)
	assert.Equal(t, []string{"clusterrolebindings"}, resp.Data.Spec.ExcludedClusterScopedResources)
	assert.Nil(t, resp.Data.Spec.IncludeClusterResources)

	stored, err := h.DisasterClient.DisasterV1().AppBackups(common.DisasterSystemNamespace).Get(context.Background(), "test-backup-scoped-update", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Nil(t, stored.Spec.Template.IncludeClusterResources)
}

func TestCreateAppBackup_RejectsMixedOldAndScopedResourceFilters(t *testing.T) {
	cluster := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster-mixed"},
		Status:     dapisv1.ClusterStatus{Status: "Ready"},
	}
	repo := &dapisv1.StorageRepository{
		ObjectMeta: metav1.ObjectMeta{Name: "test-storage-mixed", Namespace: common.DisasterSystemNamespace},
		Status:     dapisv1.StorageRepositoryStatus{Status: "Available"},
	}
	h, _ := newMockHandler(cluster, repo)

	ctx := app.NewContext(16)
	req := CreateAppBackupRequest{
		Name:                             "mixed-backup",
		Cluster:                          "test-cluster-mixed",
		Schedule:                         "@hourly",
		StorageLocation:                  "test-storage-mixed",
		IncludedResources:                []string{"deployments.apps"},
		IncludedNamespaceScopedResources: []string{"deployments.apps"},
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createAppBackup(context.Background(), ctx)
	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	assert.Contains(t, string(ctx.Response.Body()), AppBackupResourceFilterInvalid)
}

func TestUpdateAppBackup_RejectsScopedConflictResourceFilters(t *testing.T) {
	backup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{Name: "test-backup-scoped-conflict", Namespace: common.DisasterSystemNamespace},
	}
	h, _ := newMockHandler(backup)

	ctx := app.NewContext(16)
	req := UpdateAppBackupRequest{
		Name:                           "test-backup-scoped-conflict",
		IncludedClusterScopedResources: []string{"nodes"},
		ExcludedClusterScopedResources: []string{"nodes"},
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.updateAppBackup(context.Background(), ctx)
	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	assert.Contains(t, string(ctx.Response.Body()), AppBackupResourceFilterInvalid)
}
