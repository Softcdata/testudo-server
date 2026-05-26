package appbackup

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route/param"
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/transport"
	"github.com/stretchr/testify/assert"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type staticReader struct {
	backups map[string]*velerov1.Backup
	bsls    map[string]*velerov1.BackupStorageLocation
}

func (r *staticReader) Get(ctx context.Context, key ctrlclient.ObjectKey, obj ctrlclient.Object, opts ...ctrlclient.GetOption) error {
	switch dst := obj.(type) {
	case *velerov1.Backup:
		item, ok := r.backups[key.Namespace+"/"+key.Name]
		if !ok {
			return apierrors.NewNotFound(schema.GroupResource{Group: "velero.io", Resource: "backups"}, key.Name)
		}
		*dst = *item.DeepCopy()
		return nil
	case *velerov1.BackupStorageLocation:
		item, ok := r.bsls[key.Namespace+"/"+key.Name]
		if !ok {
			return apierrors.NewNotFound(schema.GroupResource{Group: "velero.io", Resource: "backupstoragelocations"}, key.Name)
		}
		*dst = *item.DeepCopy()
		return nil
	default:
		return fmt.Errorf("unexpected object type: %T", obj)
	}
}

func (r *staticReader) List(ctx context.Context, list ctrlclient.ObjectList, opts ...ctrlclient.ListOption) error {
	return nil
}

func TestGetVeleroBackupIncludes_Success(t *testing.T) {
	backup := &velerov1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "velero-backup-1",
			Namespace: common.VeleroNamespace,
		},
		Spec: velerov1.BackupSpec{
			IncludedNamespaces: []string{"ns1", "ns2"},
			IncludedResources:  []string{"pods", "services"},
		},
	}
	remote := &staticReader{
		backups: map[string]*velerov1.Backup{
			common.VeleroNamespace + "/velero-backup-1": backup,
		},
	}

	h, _ := newMockHandler()
	h.getRemoteClient = func(ctx context.Context, clusterName string) (ctrlclient.Reader, error) {
		assert.Equal(t, "cluster-a", clusterName)
		return remote, nil
	}

	ctx := app.NewContext(16)
	ctx.Params = param.Params{
		{Key: "backupName", Value: "velero-backup-1"},
	}
	ctx.Request.URI().SetQueryString("cluster=cluster-a")

	h.getVeleroBackupIncludes(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp struct {
		Code int                     `json:"code"`
		Data VeleroBackupIncludesDTO `json:"data"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	assert.Equal(t, []string{"ns1", "ns2"}, resp.Data.IncludedNamespaces)
	assert.Equal(t, []string{"pods", "services"}, resp.Data.IncludedResources)
}

func TestGetVeleroBackupIncludes_ClusterRequired(t *testing.T) {
	h, _ := newMockHandler()

	ctx := app.NewContext(16)
	ctx.Params = param.Params{
		{Key: "backupName", Value: "velero-backup-1"},
	}

	h.getVeleroBackupIncludes(context.Background(), ctx)

	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())

	var resp struct {
		Code       int    `json:"code"`
		MessageKey string `json:"message_key"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeBadRequest, resp.Code)
	assert.Equal(t, "validation.cluster_required", resp.MessageKey)
}

func TestGetVeleroBackupIncludes_NotFound(t *testing.T) {
	remote := &staticReader{backups: map[string]*velerov1.Backup{}}

	h, _ := newMockHandler()
	h.getRemoteClient = func(ctx context.Context, clusterName string) (ctrlclient.Reader, error) {
		assert.Equal(t, "cluster-a", clusterName)
		return remote, nil
	}

	ctx := app.NewContext(16)
	ctx.Params = param.Params{
		{Key: "backupName", Value: "velero-backup-1"},
	}
	ctx.Request.URI().SetQueryString("cluster=cluster-a")

	h.getVeleroBackupIncludes(context.Background(), ctx)

	assert.Equal(t, consts.StatusNotFound, ctx.Response.StatusCode())

	var resp struct {
		Code int `json:"code"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeNotFound, resp.Code)
}

func TestGetVeleroBackupIncludes_UsesActualResourceListWhenAvailable(t *testing.T) {
	backup := &velerov1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "velero-backup-actual",
			Namespace: common.VeleroNamespace,
		},
		Spec: velerov1.BackupSpec{
			IncludedNamespaces: []string{"spec-ns"},
			IncludedResources:  []string{"configmaps"},
		},
	}
	remote := &staticReader{
		backups: map[string]*velerov1.Backup{
			common.VeleroNamespace + "/velero-backup-actual": backup,
		},
	}

	h, _ := newMockHandler()
	h.getRemoteClient = func(ctx context.Context, clusterName string) (ctrlclient.Reader, error) {
		assert.Equal(t, "cluster-a", clusterName)
		return remote, nil
	}
	h.fetchBackupResourceList = func(ctx context.Context, remote ctrlclient.Reader, backup *velerov1.Backup, httpClient *http.Client) (map[string][]string, error) {
		return map[string][]string{
			"pods":             {"ns-b/pod-2", "ns-a/pod-1"},
			"deployments.apps": {"ns-a/deploy-1"},
			"nodes":            {"node-1"},
		}, nil
	}

	ctx := app.NewContext(16)
	ctx.Params = param.Params{
		{Key: "backupName", Value: "velero-backup-actual"},
	}
	ctx.Request.URI().SetQueryString("cluster=cluster-a")

	h.getVeleroBackupIncludes(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp struct {
		Code int                     `json:"code"`
		Data VeleroBackupIncludesDTO `json:"data"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	assert.Equal(t, []string{"ns-a", "ns-b"}, resp.Data.IncludedNamespaces)
	assert.Equal(t, []string{"deployments.apps", "nodes", "pods"}, resp.Data.IncludedResources)
}

func TestGetVeleroBackupIncludes_UsesStorageRepositoryCAForHTTPSDownload(t *testing.T) {
	repo := &dapisv1.StorageRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "storage-minio",
			Namespace: common.DisasterSystemNamespace,
		},
		Spec: dapisv1.StorageRepositorySpec{
			CASecretRef: &corev1.LocalObjectReference{Name: "storage-ca"},
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "storage-ca",
			Namespace: common.DisasterSystemNamespace,
		},
		Data: map[string][]byte{
			dapisv1.StorageRepositoryCASecretKey: []byte(testStorageCAPEM),
		},
	}
	backup := &velerov1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "velero-backup-https",
			Namespace: common.VeleroNamespace,
		},
		Spec: velerov1.BackupSpec{
			StorageLocation:    "storage-minio-cluster-a",
			IncludedNamespaces: []string{"spec-ns"},
			IncludedResources:  []string{"configmaps"},
		},
	}
	remote := &staticReader{
		backups: map[string]*velerov1.Backup{
			common.VeleroNamespace + "/velero-backup-https": backup,
		},
	}

	h, _ := newMockHandler(repo)
	h.K8sClient = k8sfake.NewSimpleClientset(secret)
	h.getRemoteClient = func(ctx context.Context, clusterName string) (ctrlclient.Reader, error) {
		assert.Equal(t, "cluster-a", clusterName)
		return remote, nil
	}

	fetchCalled := false
	h.fetchBackupResourceList = func(ctx context.Context, remote ctrlclient.Reader, backup *velerov1.Backup, httpClient *http.Client) (map[string][]string, error) {
		fetchCalled = true
		if httpClient == nil {
			t.Fatal("expected custom http client")
		}
		if httpClient.Timeout != downloadBodyTimeout {
			t.Fatalf("unexpected timeout: got %v want %v", httpClient.Timeout, downloadBodyTimeout)
		}
		transport, ok := httpClient.Transport.(*http.Transport)
		if !ok {
			t.Fatalf("unexpected transport type: %T", httpClient.Transport)
		}
		if transport.TLSClientConfig == nil || transport.TLSClientConfig.RootCAs == nil {
			t.Fatal("expected CA-aware TLS config on backup resource http client")
		}
		return nil, fmt.Errorf("force fallback")
	}

	ctx := app.NewContext(16)
	ctx.Params = param.Params{
		{Key: "backupName", Value: "velero-backup-https"},
	}
	ctx.Request.URI().SetQueryString("cluster=cluster-a")

	h.getVeleroBackupIncludes(context.Background(), ctx)

	assert.True(t, fetchCalled)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp struct {
		Code int                     `json:"code"`
		Data VeleroBackupIncludesDTO `json:"data"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	assert.Equal(t, []string{"spec-ns"}, resp.Data.IncludedNamespaces)
	assert.Equal(t, []string{"configmaps"}, resp.Data.IncludedResources)
}

func TestGetVeleroBackupIncludes_DirectStorageFetchAndCache(t *testing.T) {
	resourceListBody := `{"deployments.apps":["ns-a/deploy-1"],"pods":["ns-a/pod-1","ns-b/pod-2"]}`
	downloadServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(resourceListBody))
	}))
	defer downloadServer.Close()

	certDER := downloadServer.TLS.Certificates[0].Certificate[0]
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	repo := &dapisv1.StorageRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "storage-minio",
			Namespace: common.DisasterSystemNamespace,
		},
		Spec: dapisv1.StorageRepositorySpec{
			Endpoint:    "https://minio.example.local",
			Bucket:      "velero",
			AccessKey:   "ak",
			SecretKey:   "sk",
			CASecretRef: &corev1.LocalObjectReference{Name: "storage-ca"},
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "storage-ca",
			Namespace: common.DisasterSystemNamespace,
		},
		Data: map[string][]byte{
			dapisv1.StorageRepositoryCASecretKey: certPEM,
		},
	}
	backup := &velerov1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "velero-backup-direct",
			Namespace:       common.VeleroNamespace,
			ResourceVersion: "rv-1",
		},
		Spec: velerov1.BackupSpec{
			StorageLocation: "storage-minio-cluster-a",
		},
	}
	remote := &staticReader{
		backups: map[string]*velerov1.Backup{
			common.VeleroNamespace + "/velero-backup-direct": backup,
		},
	}

	h, mockStorage := newMockHandler(repo)
	h.K8sClient = k8sfake.NewSimpleClientset(secret)
	h.getRemoteClient = func(ctx context.Context, clusterName string) (ctrlclient.Reader, error) {
		assert.Equal(t, "cluster-a", clusterName)
		return remote, nil
	}

	getURLCalls := 0
	mockStorage.MockGetDownloadURL = func(ctx context.Context, endpoint, accessKey, secretKey, bucket, region string, addressingStyle dapisv1.StorageRepositoryAddressingStyle, caBundle []byte, objectKey string, expiry time.Duration) (string, error) {
		getURLCalls++
		assert.Equal(t, "cluster-a/backups/velero-backup-direct/velero-backup-direct-resource-list.json.gz", objectKey)
		assert.NotEmpty(t, caBundle)
		return downloadServer.URL, nil
	}
	h.fetchBackupResourceList = func(ctx context.Context, remote ctrlclient.Reader, backup *velerov1.Backup, httpClient *http.Client) (map[string][]string, error) {
		t.Fatal("expected direct storage fetch to avoid DownloadRequest fallback")
		return nil, nil
	}

	call := func() VeleroBackupIncludesDTO {
		ctx := app.NewContext(16)
		ctx.Params = param.Params{
			{Key: "backupName", Value: "velero-backup-direct"},
		}
		ctx.Request.URI().SetQueryString("cluster=cluster-a")

		h.getVeleroBackupIncludes(context.Background(), ctx)
		assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

		var resp struct {
			Code int                     `json:"code"`
			Data VeleroBackupIncludesDTO `json:"data"`
		}
		err := json.Unmarshal(ctx.Response.Body(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, transport.CodeOK, resp.Code)
		return resp.Data
	}

	first := call()
	second := call()

	assert.Equal(t, 1, getURLCalls)
	assert.Equal(t, []string{"ns-a", "ns-b"}, first.IncludedNamespaces)
	assert.Equal(t, []string{"deployments.apps", "pods"}, first.IncludedResources)
	assert.Equal(t, first, second)
}

const testStorageCAPEM = `-----BEGIN CERTIFICATE-----
MIIDOzCCAiOgAwIBAgIUVN1uV3M+8Rh8OCBZVPS+OmqyeZgwDQYJKoZIhvcNAQEL
BQAwITELMAkGA1UEBhMCQ04xEjAQBgNVBAMMCWNhdHRsZS1jYTAeFw0yNTEyMTkw
MzIwMjdaFw0zNTEyMTcwMzIwMjdaMCwxCzAJBgNVBAYTAkNOMR0wGwYDVQQDDBRn
aXQyMTEuaTJkb2NrZXIuc2l0ZTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoC
ggEBAL5a3wqgJ7bkrghjXS/nco3NGhHiZO4XipLEOHVonH4PSUuhyml/C5LE/5vb
ODYHkkhgGzhDbBTq0ZsCaNDnEQ9hae3Zgc42uGtK2owi8rbU61nkZcCdpavFYsnd
Dl1Jx7HxVIO1KE6poYrhPrSBZd9qAzKW0jlGQ2hIRHDJtOg8C18PmujSekIPN1CM
W9WApLScFkLWiSFLNiLwx44Wsp2lcCQKV0bHlfJyUdLVlWbvu1GgbVJxKWOtvSYB
m/d+s6CldJ3J8vkL49uOOGc3qqdLlveQnKVTz3ZM2a1wGhfH7wlCK5vnXfv+wBjh
oJHvzyuU8mLW7gzQTu30iB3L4scCAwEAAaNgMF4wCQYDVR0TBAIwADALBgNVHQ8E
BAMCBeAwHQYDVR0lBBYwFAYIKwYBBQUHAwIGCCsGAQUFBwMBMCUGA1UdEQQeMByC
FGdpdDIxMS5pMmRvY2tlci5zaXRlhwTIyMjTMA0GCSqGSIb3DQEBCwUAA4IBAQAw
5OtRA48ATnGqsA34NrKg/xPKp9CU4rjI5N3802GoFSTNMhjcxezzXywZiNIWj0q9
bQPCamMjFHe12nU6n+ISCdTAIlvTIpqKHQ0i8J1TfK8o0PjZJ5QqX/gOgw81bpXM
X9d1Kbb02RYjnSDFQ7nyeOXS9lr5DdsMGXc1TAAMdbsikQRSpxNXfgBue/x1BXQs
fPEmjwafxeZyM32rBovIoukCRjlIikSoymQF9/RlcaPzt5QW0Z+p8rnPhwxPElf4
SnoHiwQcXopa6uf2+VwDDiL44aD4/4oTqdKAo6W6jQFsW/RpDTIQoJYXEnwFa+G2
zmF2rXZyuoHTS8sb/1dP
-----END CERTIFICATE-----`
