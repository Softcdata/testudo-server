package storage

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route/param"
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-operator/pkg/clientset/versioned/fake"
	informers "github.com/softcdata/testudo-operator/pkg/informers/externalversions"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/kube"
	"github.com/softcdata/testudo-server/internal/transport"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func newMockHandler(objects ...runtime.Object) *StorageHandler {
	fakeClient := fake.NewSimpleClientset(objects...)
	informerFactory := informers.NewSharedInformerFactory(fakeClient, 0)

	kc := &kube.KubeClient{
		K8sClient:       k8sfake.NewSimpleClientset(),
		DisasterClient:  fakeClient,
		InformerFactory: informerFactory,
	}

	h := server.Default()
	rg := h.Group("/v1")

	return NewStorageHandler(kc, rg)
}

func TestPatchStorage(t *testing.T) {
	storageName := "test-storage"
	storage := &dapisv1.StorageRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      storageName,
			Namespace: common.DisasterSystemNamespace,
		},
		Spec: dapisv1.StorageRepositorySpec{
			AccessKey: "old-ak",
			SecretKey: "old-sk",
			Endpoint:  "old-endpoint",
			Bucket:    "old-bucket",
		},
	}
	h := newMockHandler(storage)

	ctx := app.NewContext(16)
	ctx.Params = param.Params{
		{Key: "name", Value: storageName},
	}

	newAk := "new-ak"
	newSk := "new-sk"
	req := PatchStorageRepositoryRequest{
		AccessKey: &newAk,
		SecretKey: &newSk,
	}

	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.patchStorage(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	// Verify Update
	updatedStorage, err := h.DisasterClient.DisasterV1().StorageRepositories(common.DisasterSystemNamespace).Get(context.Background(), storageName, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "new-ak", updatedStorage.Spec.AccessKey)
	assert.Equal(t, "new-sk", updatedStorage.Spec.SecretKey)
	assert.Equal(t, "old-endpoint", updatedStorage.Spec.Endpoint) // Should be unchanged
	assert.Equal(t, "old-bucket", updatedStorage.Spec.Bucket)     // Should be unchanged
}

func TestPatchStorage_DoNotChangeEndpoint(t *testing.T) {
	storageName := "test-storage"
	storage := &dapisv1.StorageRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      storageName,
			Namespace: common.DisasterSystemNamespace,
		},
		Spec: dapisv1.StorageRepositorySpec{
			Endpoint: "original-endpoint",
		},
	}
	h := newMockHandler(storage)

	ctx := app.NewContext(16)
	ctx.Params = param.Params{
		{Key: "name", Value: storageName},
	}

	// Using manual JSON to attempt injecting "endpoint"
	jsonBody := `{"endpoint": "hacked-endpoint", "accessKey": "new-ak"}`
	ctx.Request.SetBody([]byte(jsonBody))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.patchStorage(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	// Verify Update
	updatedStorage, err := h.DisasterClient.DisasterV1().StorageRepositories(common.DisasterSystemNamespace).Get(context.Background(), storageName, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "new-ak", updatedStorage.Spec.AccessKey)
	assert.Equal(t, "original-endpoint", updatedStorage.Spec.Endpoint) // MUST be unchanged
}

func TestCreateStorage_WithCABundleCreatesManagedSecretAndDefaultsAddressingStyle(t *testing.T) {
	h := newMockHandler()

	ctx := app.NewContext(16)
	jsonBody := `{"name":"repo-a","storageType":"s3","bucket":"bucket-a","region":"us-east-1","endpoint":"https://s3.example.com","accessKey":"ak","secretKey":"sk","caBundle":"-----BEGIN CERTIFICATE-----\nTEST\n-----END CERTIFICATE-----"}`
	ctx.Request.SetBody([]byte(jsonBody))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createStorage(context.Background(), ctx)

	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())

	created, err := h.DisasterClient.DisasterV1().StorageRepositories(common.DisasterSystemNamespace).Get(context.Background(), "repo-a", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, dapisv1.StorageRepositoryAddressingStylePathStyle, created.Spec.GetAddressingStyle())
	if assert.NotNil(t, created.Spec.CASecretRef) {
		assert.Equal(t, managedCASecretName("repo-a"), created.Spec.CASecretRef.Name)
	}

	secret, err := h.K8sClient.CoreV1().Secrets(common.DisasterSystemNamespace).Get(context.Background(), managedCASecretName("repo-a"), metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "-----BEGIN CERTIFICATE-----\nTEST\n-----END CERTIFICATE-----", string(secret.Data[dapisv1.StorageRepositoryCASecretKey]))
}

func TestPatchStorage_UpdatesAddressingStyleAndClearsManagedCA(t *testing.T) {
	storageName := "test-storage"
	storage := &dapisv1.StorageRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      storageName,
			Namespace: common.DisasterSystemNamespace,
		},
		Spec: dapisv1.StorageRepositorySpec{
			Endpoint:        "https://s3.example.com",
			AddressingStyle: dapisv1.StorageRepositoryAddressingStylePathStyle,
			CASecretRef:     &corev1.LocalObjectReference{Name: managedCASecretName(storageName)},
		},
	}
	h := newMockHandler(storage)
	_, err := h.K8sClient.CoreV1().Secrets(common.DisasterSystemNamespace).Create(context.Background(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      managedCASecretName(storageName),
			Namespace: common.DisasterSystemNamespace,
		},
		Data: map[string][]byte{
			dapisv1.StorageRepositoryCASecretKey: []byte("ca-data"),
		},
	}, metav1.CreateOptions{})
	assert.NoError(t, err)

	ctx := app.NewContext(16)
	ctx.Params = param.Params{
		{Key: "name", Value: storageName},
	}
	jsonBody := `{"addressingStyle":"VirtualHostedStyle","clearCa":true}`
	ctx.Request.SetBody([]byte(jsonBody))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.patchStorage(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	updatedStorage, err := h.DisasterClient.DisasterV1().StorageRepositories(common.DisasterSystemNamespace).Get(context.Background(), storageName, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, dapisv1.StorageRepositoryAddressingStyleVirtualHostedStyle, updatedStorage.Spec.AddressingStyle)
	assert.Nil(t, updatedStorage.Spec.CASecretRef)

	_, err = h.K8sClient.CoreV1().Secrets(common.DisasterSystemNamespace).Get(context.Background(), managedCASecretName(storageName), metav1.GetOptions{})
	assert.Error(t, err)
}

func TestStorageNames_IncludesStatus(t *testing.T) {
	storageA := &dapisv1.StorageRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "storage-a",
			Namespace: common.DisasterSystemNamespace,
			UID:       types.UID("uid-a"),
		},
		Status: dapisv1.StorageRepositoryStatus{
			Status: dapisv1.StorageRepositoryStatusAvailable,
		},
	}
	storageB := &dapisv1.StorageRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "storage-b",
			Namespace: common.DisasterSystemNamespace,
			UID:       types.UID("uid-b"),
		},
		Status: dapisv1.StorageRepositoryStatus{
			Status: dapisv1.StorageRepositoryStatusUnavailable,
		},
	}

	h := newMockHandler(storageA, storageB)

	stopCh := make(chan struct{})
	defer close(stopCh)
	h.InformerFactory.Start(stopCh)
	for _, ok := range h.InformerFactory.WaitForCacheSync(stopCh) {
		assert.True(t, ok)
	}

	ctx := app.NewContext(16)
	h.storageNames(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp struct {
		Code int                      `json:"code"`
		Data []DisasterStorageNameDTO `json:"data"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)

	gotStatus := make(map[string]dapisv1.StatusType, len(resp.Data))
	for _, item := range resp.Data {
		gotStatus[item.Name] = item.Status
	}

	assert.Equal(t, dapisv1.StorageRepositoryStatusAvailable, gotStatus["storage-a"])
	assert.Equal(t, dapisv1.StorageRepositoryStatusUnavailable, gotStatus["storage-b"])
}
