package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route/param"
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-operator/pkg/clientset/versioned/fake"
	informers "github.com/softcdata/testudo-operator/pkg/informers/externalversions"
	platformlicense "github.com/softcdata/testudo-operator/pkg/license"
	metadata "github.com/softcdata/testudo-operator/pkg/metadata"
	"github.com/softcdata/testudo-operator/pkg/tools"
	platformlicenseapi "github.com/softcdata/testudo-server/internal/apis/platform_license/v1"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/kube"
	"github.com/softcdata/testudo-server/internal/transport"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type refreshNamespacesResponse struct {
	Code int                          `json:"code"`
	Data RefreshNamespacesAcceptedDTO `json:"data"`
}

func newMockHandler(objects ...runtime.Object) *ClusterHandler {
	fakeClient := fake.NewSimpleClientset(objects...)
	informerFactory := informers.NewSharedInformerFactory(fakeClient, 0)

	kc := &kube.KubeClient{
		K8sClient:       k8sfake.NewSimpleClientset(),
		DisasterClient:  fakeClient,
		InformerFactory: informerFactory,
	}

	h := server.Default()
	rg := h.Group("/v1")

	handler := NewClusterHandler(kc, rg)
	handler.LicenseGateEnabled = false
	informerFactory.Disaster().V1().Clusters().Informer()
	informerFactory.Disaster().V1().DisasterConfigs().Informer()
	informerFactory.Disaster().V1().DisasterInstances().Informer()
	informerFactory.Start(context.Background().Done())
	informerFactory.WaitForCacheSync(context.Background().Done())
	return handler
}

type fakeClusterLicenseService struct {
	check platformlicenseapi.ClusterCreateCheck
	err   error
}

func (f *fakeClusterLicenseService) CheckClusterCreate(_ context.Context) (platformlicenseapi.ClusterCreateCheck, error) {
	return f.check, f.err
}

type fakeSubResourceClient struct{}

func (f *fakeSubResourceClient) Get(context.Context, client.Object, client.Object, ...client.SubResourceGetOption) error {
	return nil
}
func (f *fakeSubResourceClient) Create(context.Context, client.Object, client.Object, ...client.SubResourceCreateOption) error {
	return nil
}
func (f *fakeSubResourceClient) Update(context.Context, client.Object, ...client.SubResourceUpdateOption) error {
	return nil
}
func (f *fakeSubResourceClient) Patch(context.Context, client.Object, client.Patch, ...client.SubResourcePatchOption) error {
	return nil
}

type fakeTargetClusterClient struct {
	storageClasses []storagev1.StorageClass
	ingressClasses []networkingv1.IngressClass
	listErr        error
}

func (f *fakeTargetClusterClient) Get(context.Context, client.ObjectKey, client.Object, ...client.GetOption) error {
	return fmt.Errorf("not implemented")
}

func (f *fakeTargetClusterClient) List(_ context.Context, list client.ObjectList, _ ...client.ListOption) error {
	if f.listErr != nil {
		return f.listErr
	}
	switch typed := list.(type) {
	case *storagev1.StorageClassList:
		typed.Items = append([]storagev1.StorageClass(nil), f.storageClasses...)
		return nil
	case *networkingv1.IngressClassList:
		typed.Items = append([]networkingv1.IngressClass(nil), f.ingressClasses...)
		return nil
	default:
		return fmt.Errorf("unsupported list type %T", list)
	}
}

func (f *fakeTargetClusterClient) Create(context.Context, client.Object, ...client.CreateOption) error {
	return fmt.Errorf("not implemented")
}
func (f *fakeTargetClusterClient) Delete(context.Context, client.Object, ...client.DeleteOption) error {
	return fmt.Errorf("not implemented")
}
func (f *fakeTargetClusterClient) Update(context.Context, client.Object, ...client.UpdateOption) error {
	return fmt.Errorf("not implemented")
}
func (f *fakeTargetClusterClient) Patch(context.Context, client.Object, client.Patch, ...client.PatchOption) error {
	return fmt.Errorf("not implemented")
}
func (f *fakeTargetClusterClient) DeleteAllOf(context.Context, client.Object, ...client.DeleteAllOfOption) error {
	return fmt.Errorf("not implemented")
}
func (f *fakeTargetClusterClient) Status() client.SubResourceWriter {
	return &fakeSubResourceClient{}
}
func (f *fakeTargetClusterClient) SubResource(string) client.SubResourceClient {
	return &fakeSubResourceClient{}
}
func (f *fakeTargetClusterClient) Scheme() *runtime.Scheme {
	return runtime.NewScheme()
}
func (f *fakeTargetClusterClient) RESTMapper() meta.RESTMapper {
	return nil
}
func (f *fakeTargetClusterClient) GroupVersionKindFor(runtime.Object) (schema.GroupVersionKind, error) {
	return schema.GroupVersionKind{}, nil
}
func (f *fakeTargetClusterClient) IsObjectNamespaced(runtime.Object) (bool, error) {
	return false, nil
}

type restoreClassesResponse struct {
	Code int                 `json:"code"`
	Data RestoreClassListDTO `json:"data"`
}

type clusterConflictResponse struct {
	Code    int                         `json:"code"`
	Message string                      `json:"message"`
	Meta    ClusterEndpointConflictMeta `json:"meta"`
}

type protectedNamespacesResponse struct {
	Code int `json:"code"`
	Data struct {
		Items []ClusterProtectedNamespaceDTO `json:"items"`
	} `json:"data"`
	Meta struct {
		Type         string `json:"type"`
		ResourceType string `json:"resourceType"`
		Pagination   struct {
			Limit   int   `json:"limit"`
			Total   int64 `json:"total"`
			Partial bool  `json:"partial"`
		} `json:"pagination"`
		Sort struct {
			Name  string `json:"name"`
			Order string `json:"order"`
		} `json:"sort"`
	} `json:"meta"`
}

func TestPatchCluster(t *testing.T) {
	clusterName := "test-cluster"
	cluster := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
		},
		Spec: dapisv1.ClusterSpec{
			Token:    "old-token",
			Endpoint: "old-endpoint",
		},
	}
	h := newMockHandler(cluster)

	ctx := app.NewContext(16)
	ctx.Params = param.Params{
		{Key: "name", Value: clusterName},
	}

	newToken := "new-token"
	newTag := "prod"
	req := PatchDisasterClusterRequest{
		Token: &newToken,
		Tag:   &newTag,
	}

	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.patchCluster(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	// Verify Update
	updatedCluster, err := h.DisasterClient.DisasterV1().Clusters().Get(context.Background(), clusterName, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "new-token", updatedCluster.Spec.Token)
	assert.Equal(t, "prod", updatedCluster.Labels[ClusterTagLabel])
	assert.Equal(t, "old-endpoint", updatedCluster.Spec.Endpoint) // Should be unchanged
}

func TestCreateCluster_WithImageSources(t *testing.T) {
	h := newMockHandler()

	ctx := app.NewContext(16)
	req := CreateDisasterClusterRequest{
		Name:  "cluster-with-image-sources",
		Token: "token-1",
		ImageSources: []ImageSourceDTO{
			{Name: "prod-main", Registry: "harbor.prod.local"},
			{Name: "dr-main", Registry: "harbor.dr.local"},
		},
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createCluster(context.Background(), ctx)

	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())
	created, err := h.DisasterClient.DisasterV1().Clusters().Get(context.Background(), req.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, []dapisv1.ImageSource{
		{Name: "prod-main", Registry: "harbor.prod.local"},
		{Name: "dr-main", Registry: "harbor.dr.local"},
	}, created.Spec.ImageSources)
}

func TestCreateCluster_WithVeleroInstallCredentialsCreatesManagedSecret(t *testing.T) {
	h := newMockHandler()

	ctx := app.NewContext(16)
	req := CreateDisasterClusterRequest{
		Name:  "cluster-with-velero-registry",
		Token: "token-1",
		VeleroInstall: &VeleroInstallWriteDTO{
			ImageRegistry: " harbor.customer.local/disaster/ ",
			Username:      "registry-user",
			Password:      "registry-password",
		},
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createCluster(context.Background(), ctx)

	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())
	created, err := h.DisasterClient.DisasterV1().Clusters().Get(context.Background(), req.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	if assert.NotNil(t, created.Spec.VeleroInstall) {
		assert.Equal(t, "harbor.customer.local/disaster", created.Spec.VeleroInstall.ImageRegistry)
		if assert.NotNil(t, created.Spec.VeleroInstall.RegistryCredentialSecretRef) {
			assert.Equal(t, managedVeleroRegistrySecretName(req.Name), created.Spec.VeleroInstall.RegistryCredentialSecretRef.Name)
		}
	}

	secret, err := h.K8sClient.CoreV1().Secrets(common.DisasterSystemNamespace).Get(context.Background(), managedVeleroRegistrySecretName(req.Name), metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, corev1.SecretTypeDockerConfigJson, secret.Type)
	assert.Contains(t, string(secret.Data[corev1.DockerConfigJsonKey]), "registry-user")
	assert.NotContains(t, string(ctx.Response.Body()), "registry-password")
}

func TestCreateCluster_RejectsWhenLicenseLimitExceeded(t *testing.T) {
	h := newMockHandler(
		&dapisv1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "existing-a"}},
		&dapisv1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "existing-b"}},
	)
	licenseService := &fakeClusterLicenseService{
		check: platformlicenseapi.ClusterCreateCheck{
			Allowed: false,
			Message: "cluster quota exceeded",
			Meta: platformlicenseapi.LicenseErrorMeta{
				Reason:          platformlicense.ReasonLicenseLimitExceeded,
				State:           string(platformlicense.StateFree),
				MaxClusters:     2,
				CurrentClusters: 2,
			},
		},
	}
	h.LicenseGateEnabled = true
	h.LicenseService = licenseService

	ctx := app.NewContext(16)
	req := CreateDisasterClusterRequest{
		Name:  "third-cluster",
		Token: "token-1",
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createCluster(context.Background(), ctx)

	assert.Equal(t, consts.StatusForbidden, ctx.Response.StatusCode())

	var resp struct {
		Code int                                 `json:"code"`
		Meta platformlicenseapi.LicenseErrorMeta `json:"meta"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeForbidden, resp.Code)
	assert.Equal(t, platformlicense.ReasonLicenseLimitExceeded, resp.Meta.Reason)
	assert.Equal(t, 2, resp.Meta.CurrentClusters)

	_, err = h.DisasterClient.DisasterV1().Clusters().Get(context.Background(), req.Name, metav1.GetOptions{})
	assert.True(t, apierrors.IsNotFound(err))
}

func TestCreateCluster_InvalidClusterSpecReturnsBadRequest(t *testing.T) {
	h := newMockHandler()
	fakeClient := h.DisasterClient.(*fake.Clientset)
	fakeClient.PrependReactor("create", "clusters", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewInvalid(
			schema.GroupKind{Group: "cluster.testudo.softcdata.com", Kind: "Cluster"},
			"token-https-test",
			field.ErrorList{
				field.Invalid(field.NewPath("spec", "kubeConfig"), "", `spec.kubeConfig in body must be of type byte: ""`),
			},
		)
	})

	ctx := app.NewContext(16)
	req := map[string]string{
		"name":       "token-https-test",
		"token":      "token-1",
		"endpoint":   "https://192.0.2.170:6443",
		"kubeConfig": "",
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createCluster(context.Background(), ctx)

	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	assert.Contains(t, string(ctx.Response.Body()), "spec.kubeConfig")
}

func TestCreateCluster_RejectsDuplicateEndpoint(t *testing.T) {
	h := newMockHandler(&dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "existing-cluster"},
		Spec: dapisv1.ClusterSpec{
			Endpoint: " HTTPS://api.demo.local:443/ ",
		},
	})

	ctx := app.NewContext(16)
	req := CreateDisasterClusterRequest{
		Name:     "new-cluster",
		Endpoint: "https://API.demo.local",
		Token:    "token-1",
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createCluster(context.Background(), ctx)

	assert.Equal(t, consts.StatusConflict, ctx.Response.StatusCode())

	var resp clusterConflictResponse
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeConflict, resp.Code)
	assert.Contains(t, resp.Message, "existing-cluster")
	assert.Equal(t, "clusterEndpoint", resp.Meta.ConflictType)
	assert.Equal(t, "existing-cluster", resp.Meta.ConflictCluster)
	assert.Equal(t, "https://api.demo.local", resp.Meta.ConflictEndpoint)
}

func TestCreateCluster_RejectsDuplicateEndpointDerivedFromKubeConfig(t *testing.T) {
	existingKubeConfig, err := tools.GenerateKubeConfigFromToken("https://api.demo.local:443/", "token-1")
	assert.NoError(t, err)
	newKubeConfig, err := tools.GenerateKubeConfigFromToken("HTTPS://API.demo.local/", "token-2")
	assert.NoError(t, err)

	h := newMockHandler(&dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "existing-cluster"},
		Spec: dapisv1.ClusterSpec{
			KubeConfig: existingKubeConfig,
		},
	})

	ctx := app.NewContext(16)
	req := CreateDisasterClusterRequest{
		Name:       "new-cluster",
		KubeConfig: newKubeConfig,
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createCluster(context.Background(), ctx)

	assert.Equal(t, consts.StatusConflict, ctx.Response.StatusCode())

	var resp clusterConflictResponse
	err = json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeConflict, resp.Code)
	assert.Equal(t, "existing-cluster", resp.Meta.ConflictCluster)
	assert.Equal(t, "https://api.demo.local", resp.Meta.ConflictEndpoint)
}

func TestProtectedNamespaces_Success(t *testing.T) {
	h := newMockHandler(
		&dapisv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster-a"},
		},
		&dapisv1.DisasterConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "cfg-a"},
			Spec: dapisv1.DisasterConfigSpec{
				SourceCluster: "cluster-a",
			},
		},
		&dapisv1.DisasterConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "cfg-a-2"},
			Spec: dapisv1.DisasterConfigSpec{
				SourceCluster: "cluster-a",
			},
		},
		&dapisv1.DisasterConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "cfg-b"},
			Spec: dapisv1.DisasterConfigSpec{
				SourceCluster: "cluster-b",
			},
		},
		&dapisv1.DisasterInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "inst-a", Namespace: "disaster-system"},
			Spec: dapisv1.DisasterInstanceSpec{
				Config:     "cfg-a",
				Namespaces: []string{"app-b", " app-a "},
			},
		},
		&dapisv1.DisasterInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "inst-b", Namespace: "disaster-system"},
			Spec: dapisv1.DisasterInstanceSpec{
				Config:     "cfg-a-2",
				Namespaces: []string{"app-a"},
			},
		},
		&dapisv1.DisasterInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "inst-c", Namespace: "disaster-system"},
			Spec: dapisv1.DisasterInstanceSpec{
				Config:     "cfg-b",
				Namespaces: []string{"app-c"},
			},
		},
	)

	ctx := app.NewContext(16)
	ctx.Params = param.Params{{Key: "name", Value: "cluster-a"}}
	ctx.Request.SetRequestURI("/clusters/cluster-a/protected-namespaces")

	h.protectedNamespaces(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp protectedNamespacesResponse
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	assert.Equal(t, "collection", resp.Meta.Type)
	assert.Equal(t, "clusterProtectedNamespace", resp.Meta.ResourceType)
	assert.EqualValues(t, 2, resp.Meta.Pagination.Total)
	assert.Equal(t, 2, resp.Meta.Pagination.Limit)
	assert.Equal(t, "namespace", resp.Meta.Sort.Name)
	assert.Equal(t, "asc", resp.Meta.Sort.Order)
	if assert.Len(t, resp.Data.Items, 2) {
		assert.Equal(t, "app-a", resp.Data.Items[0].Namespace)
		assert.Equal(t, "cluster-a", resp.Data.Items[0].Cluster)
		if assert.Len(t, resp.Data.Items[0].Owners, 2) {
			assert.Equal(t, "inst-a", resp.Data.Items[0].Owners[0].InstanceName)
			assert.Equal(t, "cfg-a", resp.Data.Items[0].Owners[0].ConfigName)
			assert.Equal(t, "inst-b", resp.Data.Items[0].Owners[1].InstanceName)
			assert.Equal(t, "cfg-a-2", resp.Data.Items[0].Owners[1].ConfigName)
		}
		assert.Equal(t, "app-b", resp.Data.Items[1].Namespace)
		if assert.Len(t, resp.Data.Items[1].Owners, 1) {
			assert.Equal(t, "inst-a", resp.Data.Items[1].Owners[0].InstanceName)
		}
	}
}

func TestProtectedNamespaces_ClusterNotFound(t *testing.T) {
	h := newMockHandler()

	ctx := app.NewContext(16)
	ctx.Params = param.Params{{Key: "name", Value: "missing"}}

	h.protectedNamespaces(context.Background(), ctx)

	assert.Equal(t, consts.StatusNotFound, ctx.Response.StatusCode())
	assert.Contains(t, string(ctx.Response.Body()), "missing")
}

func TestPatchCluster_WithImageSources(t *testing.T) {
	clusterName := "test-cluster"
	cluster := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
		},
		Spec: dapisv1.ClusterSpec{
			Token: "old-token",
		},
	}
	h := newMockHandler(cluster)

	ctx := app.NewContext(16)
	ctx.Params = param.Params{
		{Key: "name", Value: clusterName},
	}

	imageSources := []ImageSourceDTO{
		{Name: " source-main ", Registry: " harbor.src.local "},
		{Name: "dr-main", Registry: "harbor.dr.local"},
	}
	req := PatchDisasterClusterRequest{
		ImageSources: &imageSources,
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.patchCluster(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	updatedCluster, err := h.DisasterClient.DisasterV1().Clusters().Get(context.Background(), clusterName, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, []dapisv1.ImageSource{
		{Name: "source-main", Registry: "harbor.src.local"},
		{Name: "dr-main", Registry: "harbor.dr.local"},
	}, updatedCluster.Spec.ImageSources)
}

func TestPatchCluster_WithVeleroInstallImageRegistryKeepsCredentialRef(t *testing.T) {
	clusterName := "cluster-with-velero-registry"
	clusterObj := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
		},
		Spec: dapisv1.ClusterSpec{
			Token: "old-token",
			VeleroInstall: &dapisv1.VeleroInstallSpec{
				ImageRegistry: "harbor.old.local/disaster",
				RegistryCredentialSecretRef: &corev1.LocalObjectReference{
					Name: managedVeleroRegistrySecretName(clusterName),
				},
			},
		},
	}
	h := newMockHandler(clusterObj)
	_, err := h.K8sClient.CoreV1().Secrets(common.DisasterSystemNamespace).Create(context.Background(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      managedVeleroRegistrySecretName(clusterName),
			Namespace: common.DisasterSystemNamespace,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: []byte(`{"auths":{"harbor.old.local":{"auth":"fake-auth"}}}`),
		},
	}, metav1.CreateOptions{})
	assert.NoError(t, err)

	ctx := app.NewContext(16)
	ctx.Params = param.Params{{Key: "name", Value: clusterName}}
	newRegistry := "harbor.new.local/disaster"
	req := PatchDisasterClusterRequest{
		VeleroInstall: &PatchVeleroInstallWriteDTO{
			ImageRegistry: &newRegistry,
		},
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.patchCluster(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	updatedCluster, err := h.DisasterClient.DisasterV1().Clusters().Get(context.Background(), clusterName, metav1.GetOptions{})
	assert.NoError(t, err)
	if assert.NotNil(t, updatedCluster.Spec.VeleroInstall) {
		assert.Equal(t, "harbor.new.local/disaster", updatedCluster.Spec.VeleroInstall.ImageRegistry)
		if assert.NotNil(t, updatedCluster.Spec.VeleroInstall.RegistryCredentialSecretRef) {
			assert.Equal(t, managedVeleroRegistrySecretName(clusterName), updatedCluster.Spec.VeleroInstall.RegistryCredentialSecretRef.Name)
		}
	}
	_, err = h.K8sClient.CoreV1().Secrets(common.DisasterSystemNamespace).Get(context.Background(), managedVeleroRegistrySecretName(clusterName), metav1.GetOptions{})
	assert.NoError(t, err)
}

func TestPatchCluster_WithVeleroInstallCredentialRotationUpdatesManagedSecret(t *testing.T) {
	clusterName := "cluster-rotate-velero-registry"
	clusterObj := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
		},
		Spec: dapisv1.ClusterSpec{
			Token: "old-token",
			VeleroInstall: &dapisv1.VeleroInstallSpec{
				ImageRegistry: "harbor.customer.local/disaster",
				RegistryCredentialSecretRef: &corev1.LocalObjectReference{
					Name: managedVeleroRegistrySecretName(clusterName),
				},
			},
		},
	}
	h := newMockHandler(clusterObj)
	_, err := h.K8sClient.CoreV1().Secrets(common.DisasterSystemNamespace).Create(context.Background(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      managedVeleroRegistrySecretName(clusterName),
			Namespace: common.DisasterSystemNamespace,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: []byte(`{"auths":{"harbor.customer.local":{"username":"old-user","password":"old-pass","auth":"fake-auth"}}}`),
		},
	}, metav1.CreateOptions{})
	assert.NoError(t, err)

	ctx := app.NewContext(16)
	ctx.Params = param.Params{{Key: "name", Value: clusterName}}
	newUsername := "new-user"
	newPassword := "new-pass"
	req := PatchDisasterClusterRequest{
		VeleroInstall: &PatchVeleroInstallWriteDTO{
			Username: &newUsername,
			Password: &newPassword,
		},
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.patchCluster(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	secret, err := h.K8sClient.CoreV1().Secrets(common.DisasterSystemNamespace).Get(context.Background(), managedVeleroRegistrySecretName(clusterName), metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Contains(t, string(secret.Data[corev1.DockerConfigJsonKey]), "new-user")

	updatedCluster, err := h.DisasterClient.DisasterV1().Clusters().Get(context.Background(), clusterName, metav1.GetOptions{})
	assert.NoError(t, err)
	if assert.NotNil(t, updatedCluster.Spec.VeleroInstall) {
		assert.Equal(t, "harbor.customer.local/disaster", updatedCluster.Spec.VeleroInstall.ImageRegistry)
		if assert.NotNil(t, updatedCluster.Spec.VeleroInstall.RegistryCredentialSecretRef) {
			assert.Equal(t, managedVeleroRegistrySecretName(clusterName), updatedCluster.Spec.VeleroInstall.RegistryCredentialSecretRef.Name)
		}
	}
}

func TestPatchCluster_WithVeleroInstallRemoveCredentialDeletesManagedSecret(t *testing.T) {
	clusterName := "cluster-remove-velero-registry"
	clusterObj := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
		},
		Spec: dapisv1.ClusterSpec{
			Token: "old-token",
			VeleroInstall: &dapisv1.VeleroInstallSpec{
				ImageRegistry: "harbor.customer.local/disaster",
				RegistryCredentialSecretRef: &corev1.LocalObjectReference{
					Name: managedVeleroRegistrySecretName(clusterName),
				},
			},
		},
	}
	h := newMockHandler(clusterObj)
	_, err := h.K8sClient.CoreV1().Secrets(common.DisasterSystemNamespace).Create(context.Background(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      managedVeleroRegistrySecretName(clusterName),
			Namespace: common.DisasterSystemNamespace,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: []byte(`{"auths":{"harbor.customer.local":{"auth":"fake-auth"}}}`),
		},
	}, metav1.CreateOptions{})
	assert.NoError(t, err)

	ctx := app.NewContext(16)
	ctx.Params = param.Params{{Key: "name", Value: clusterName}}
	removeCredential := true
	req := PatchDisasterClusterRequest{
		VeleroInstall: &PatchVeleroInstallWriteDTO{
			RemoveCredential: &removeCredential,
		},
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.patchCluster(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	updatedCluster, err := h.DisasterClient.DisasterV1().Clusters().Get(context.Background(), clusterName, metav1.GetOptions{})
	assert.NoError(t, err)
	if assert.NotNil(t, updatedCluster.Spec.VeleroInstall) {
		assert.Nil(t, updatedCluster.Spec.VeleroInstall.RegistryCredentialSecretRef)
		assert.Equal(t, "harbor.customer.local/disaster", updatedCluster.Spec.VeleroInstall.ImageRegistry)
	}
	_, err = h.K8sClient.CoreV1().Secrets(common.DisasterSystemNamespace).Get(context.Background(), managedVeleroRegistrySecretName(clusterName), metav1.GetOptions{})
	assert.True(t, apierrors.IsNotFound(err))
}

func TestPatchCluster_WithDuplicateImageSourceName_ShouldFail(t *testing.T) {
	clusterName := "test-cluster"
	cluster := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
		},
	}
	h := newMockHandler(cluster)

	ctx := app.NewContext(16)
	ctx.Params = param.Params{
		{Key: "name", Value: clusterName},
	}

	imageSources := []ImageSourceDTO{
		{Name: "main", Registry: "harbor.src.local"},
		{Name: "main", Registry: "harbor.dr.local"},
	}
	req := PatchDisasterClusterRequest{
		ImageSources: &imageSources,
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.patchCluster(context.Background(), ctx)

	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	updatedCluster, err := h.DisasterClient.DisasterV1().Clusters().Get(context.Background(), clusterName, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Empty(t, updatedCluster.Spec.ImageSources)
}

func TestPatchCluster_DoNotUpdateEndpoint(t *testing.T) {
	// This test simulates sending JSON with "endpoint" field, but since we bind to PatchDisasterClusterRequest
	// which lacks "endpoint" field, it should be ignored.
	clusterName := "test-cluster"
	cluster := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
		},
		Spec: dapisv1.ClusterSpec{
			Endpoint: "original-endpoint",
		},
	}
	h := newMockHandler(cluster)

	ctx := app.NewContext(16)
	ctx.Params = param.Params{
		{Key: "name", Value: clusterName},
	}

	// Construct JSON manually with extra field
	jsonBody := `{"endpoint": "hacked-endpoint", "token": "new-token"}`
	ctx.Request.SetBody([]byte(jsonBody))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.patchCluster(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	// Verify Update
	updatedCluster, err := h.DisasterClient.DisasterV1().Clusters().Get(context.Background(), clusterName, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "new-token", updatedCluster.Spec.Token)
	assert.Equal(t, "original-endpoint", updatedCluster.Spec.Endpoint) // MUST be unchanged
}

func TestRefreshNamespacesAction_Success(t *testing.T) {
	clusterName := "refresh-cluster"
	clusterObj := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
		},
		Status: dapisv1.ClusterStatus{
			WorkloadNamespaceCount: 2,
			WorkloadTotalCount:     5,
		},
	}
	h := newMockHandler(clusterObj)

	ctx := app.NewContext(16)
	ctx.Params = param.Params{{Key: "name", Value: clusterName}}
	body, _ := json.Marshal(RefreshNamespacesRequest{Type: string(metadata.ClusterStatsRefreshTypeWorkloadNamespaceStats)})
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.refreshNamespaces(context.Background(), ctx)

	assert.Equal(t, consts.StatusAccepted, ctx.Response.StatusCode())

	updatedCluster, err := h.DisasterClient.DisasterV1().Clusters().Get(context.Background(), clusterName, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, string(metadata.ClusterStatsRefreshTypeWorkloadNamespaceStats), updatedCluster.Annotations[metadata.AnnotationRefreshClusterStats])
	assert.NotContains(t, updatedCluster.Annotations, metadata.AnnotationUser)

	var resp refreshNamespacesResponse
	err = json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	assert.Equal(t, clusterName, resp.Data.Cluster.Name)
	assert.Equal(t, string(metadata.ClusterStatsRefreshTypeWorkloadNamespaceStats), resp.Data.Type)
}

func TestRefreshNamespacesAction_InvalidType(t *testing.T) {
	clusterName := "refresh-cluster-invalid"
	clusterObj := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
		},
	}
	h := newMockHandler(clusterObj)

	ctx := app.NewContext(16)
	ctx.Params = param.Params{{Key: "name", Value: clusterName}}
	ctx.Request.SetBody([]byte(`{"type":"unknown"}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.refreshNamespaces(context.Background(), ctx)

	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	updatedCluster, err := h.DisasterClient.DisasterV1().Clusters().Get(context.Background(), clusterName, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.NotContains(t, updatedCluster.Annotations, metadata.AnnotationRefreshClusterStats)
}

func TestRefreshNamespacesAction_RetryOnConflictWithoutUserAnnotation(t *testing.T) {
	clusterName := "refresh-cluster-retry"
	clusterObj := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
			Annotations: map[string]string{
				"existing": "keep",
			},
		},
	}
	h := newMockHandler(clusterObj)
	fakeClient := h.DisasterClient.(*fake.Clientset)

	updateCalls := 0
	fakeClient.PrependReactor("update", "clusters", func(action k8stesting.Action) (bool, runtime.Object, error) {
		updateCalls++
		if updateCalls == 1 {
			return true, nil, apierrors.NewConflict(schema.GroupResource{Group: dapisv1.GroupVersion.Group, Resource: "clusters"}, clusterName, fmt.Errorf("conflict"))
		}
		return false, nil, nil
	})

	ctx := app.NewContext(16)
	ctx.Params = param.Params{{Key: "name", Value: clusterName}}
	ctx.Set("userName", "tester")
	ctx.Set("trace_id", "trace-refresh")
	body, _ := json.Marshal(RefreshNamespacesRequest{Type: string(metadata.ClusterStatsRefreshTypeAll)})
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.refreshNamespaces(context.Background(), ctx)

	assert.Equal(t, consts.StatusAccepted, ctx.Response.StatusCode())
	assert.Equal(t, 2, updateCalls)

	updatedCluster, err := h.DisasterClient.DisasterV1().Clusters().Get(context.Background(), clusterName, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "keep", updatedCluster.Annotations["existing"])
	assert.Equal(t, string(metadata.ClusterStatsRefreshTypeAll), updatedCluster.Annotations[metadata.AnnotationRefreshClusterStats])
	assert.NotContains(t, updatedCluster.Annotations, metadata.AnnotationUser)
	assert.NotContains(t, updatedCluster.Annotations, metadata.AnnotationTraceID)
}

func TestConvertStatusToDTO_IncludesWorkloadFields(t *testing.T) {
	dto := ConvertStatusToDTO(dapisv1.ClusterStatus{
		NamespaceCount:         3,
		ResourceTotalCount:     10,
		NamespaceStats:         map[string]int{"a": 4},
		WorkloadNamespaceCount: 2,
		WorkloadTotalCount:     5,
		WorkloadNamespaceStats: map[string]int{"a": 3, "b": 2},
	})

	assert.Equal(t, 2, dto.WorkloadNamespaceCount)
	assert.Equal(t, 5, dto.WorkloadTotalCount)
	assert.Equal(t, map[string]int{"a": 3, "b": 2}, dto.WorkloadNamespaceStats)
}

func TestClusterNames_ExposeWorkloadStats(t *testing.T) {
	h := newMockHandler(
		&dapisv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster-names-workload",
				UID:  "uid-1",
				Labels: map[string]string{
					ClusterTagLabel: "tag-a",
				},
			},
			Status: dapisv1.ClusterStatus{
				NamespaceCount:         3,
				ResourceTotalCount:     20,
				WorkloadNamespaceCount: 2,
				WorkloadTotalCount:     6,
			},
		},
	)
	h.InformerFactory.Start(context.Background().Done())
	h.InformerFactory.WaitForCacheSync(context.Background().Done())

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/clusters/names")

	h.clusterNames(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	var resp struct {
		Code int                      `json:"code"`
		Data []DisasterClusterNameDTO `json:"data"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	if assert.Len(t, resp.Data, 1) {
		assert.Equal(t, 2, resp.Data[0].WorkloadNamespaceCount)
		assert.Equal(t, 6, resp.Data[0].WorkloadTotalCount)
	}
}

func TestDeleteCluster_SetUninstallVeleroAnnotationTrue(t *testing.T) {
	clusterName := "test-cluster"
	cluster := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
		},
	}
	h := newMockHandler(cluster)
	fakeClient := h.DisasterClient.(*fake.Clientset)
	fakeClient.PrependReactor("delete", "clusters", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, fmt.Errorf("forced delete failure")
	})

	ctx := app.NewContext(16)
	ctx.Params = param.Params{{Key: "name", Value: clusterName}}
	ctx.Request.URI().SetQueryString("uninstallVelero=true")

	h.deleteCluster(context.Background(), ctx)

	assert.Equal(t, consts.StatusInternalServerError, ctx.Response.StatusCode())
	updatedCluster, err := h.DisasterClient.DisasterV1().Clusters().Get(context.Background(), clusterName, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "true", updatedCluster.Annotations[metadata.AnnotationUninstallVelero])
}

func TestDeleteCluster_SetUninstallVeleroAnnotationFalse(t *testing.T) {
	clusterName := "test-cluster"
	cluster := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
			Annotations: map[string]string{
				metadata.AnnotationUninstallVelero: "true",
			},
		},
	}
	h := newMockHandler(cluster)
	fakeClient := h.DisasterClient.(*fake.Clientset)
	fakeClient.PrependReactor("delete", "clusters", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, fmt.Errorf("forced delete failure")
	})

	ctx := app.NewContext(16)
	ctx.Params = param.Params{{Key: "name", Value: clusterName}}
	ctx.Request.URI().SetQueryString("uninstallVelero=false")

	h.deleteCluster(context.Background(), ctx)

	assert.Equal(t, consts.StatusInternalServerError, ctx.Response.StatusCode())
	updatedCluster, err := h.DisasterClient.DisasterV1().Clusters().Get(context.Background(), clusterName, metav1.GetOptions{})
	assert.NoError(t, err)
	_, exists := updatedCluster.Annotations[metadata.AnnotationUninstallVelero]
	assert.False(t, exists)
}

func TestDeleteCluster_InvalidUninstallVeleroQuery(t *testing.T) {
	clusterName := "test-cluster"
	cluster := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
		},
	}
	h := newMockHandler(cluster)

	ctx := app.NewContext(16)
	ctx.Params = param.Params{{Key: "name", Value: clusterName}}
	ctx.Request.URI().SetQueryString("uninstallVelero=not-bool")

	h.deleteCluster(context.Background(), ctx)

	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	_, err := h.DisasterClient.DisasterV1().Clusters().Get(context.Background(), clusterName, metav1.GetOptions{})
	assert.NoError(t, err)
}

func TestDeleteCluster_UpdateAnnotationFailed_ShouldNotDeleteCluster(t *testing.T) {
	clusterName := "test-cluster"
	cluster := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
		},
	}
	h := newMockHandler(cluster)
	fakeClient := h.DisasterClient.(*fake.Clientset)

	deleteCalled := false
	fakeClient.PrependReactor("update", "clusters", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, fmt.Errorf("forced update failure")
	})
	fakeClient.PrependReactor("delete", "clusters", func(action k8stesting.Action) (bool, runtime.Object, error) {
		deleteCalled = true
		return false, nil, nil
	})

	ctx := app.NewContext(16)
	ctx.Params = param.Params{{Key: "name", Value: clusterName}}
	ctx.Request.URI().SetQueryString("uninstallVelero=true")

	h.deleteCluster(context.Background(), ctx)

	assert.Equal(t, consts.StatusInternalServerError, ctx.Response.StatusCode())
	assert.False(t, deleteCalled, "cluster should not be deleted when uninstallVelero annotation update fails")
}

func TestListRestoreClasses_Success(t *testing.T) {
	clusterName := "cluster-a"
	clusterObj := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: clusterName},
	}
	h := newMockHandler(clusterObj)
	h.GetClusterClient = func(ctx context.Context, name string) (client.Client, error) {
		assert.Equal(t, clusterName, name)
		return &fakeTargetClusterClient{
			storageClasses: []storagev1.StorageClass{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "gold",
						Annotations: map[string]string{
							StorageClassDefaultAnnotation: "true",
						},
					},
				},
				{ObjectMeta: metav1.ObjectMeta{Name: "bronze"}},
			},
			ingressClasses: []networkingv1.IngressClass{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "nginx",
						Annotations: map[string]string{
							IngressClassDefaultAnnotation: "true",
						},
					},
				},
				{ObjectMeta: metav1.ObjectMeta{Name: "traefik"}},
			},
		}, nil
	}

	ctx := app.NewContext(16)
	ctx.Params = param.Params{{Key: "name", Value: clusterName}}
	ctx.Request.SetRequestURI("/clusters/cluster-a/restore-classes")

	h.listRestoreClasses(context.Background(), ctx)

	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	var resp restoreClassesResponse
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeOK, resp.Code)
	assert.Equal(t, clusterName, resp.Data.TargetCluster)

	assert.Equal(t, []RestoreClassItemDTO{
		{Name: "bronze", IsDefault: false},
		{Name: "gold", IsDefault: true},
	}, resp.Data.StorageClasses)
	assert.Equal(t, []RestoreClassItemDTO{
		{Name: "nginx", IsDefault: true},
		{Name: "traefik", IsDefault: false},
	}, resp.Data.IngressClasses)
}

func TestListRestoreClasses_ClusterNotFound(t *testing.T) {
	clusterName := "missing-cluster"
	h := newMockHandler()
	h.GetClusterClient = func(ctx context.Context, name string) (client.Client, error) {
		return nil, apierrors.NewNotFound(schema.GroupResource{Group: dapisv1.GroupVersion.Group, Resource: "clusters"}, name)
	}

	ctx := app.NewContext(16)
	ctx.Params = param.Params{{Key: "name", Value: clusterName}}
	ctx.Request.SetRequestURI("/clusters/missing-cluster/restore-classes")

	h.listRestoreClasses(context.Background(), ctx)

	assert.Equal(t, consts.StatusNotFound, ctx.Response.StatusCode())
	var resp struct {
		Code int `json:"code"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeNotFound, resp.Code)
}

func TestListRestoreClasses_GetClusterClientFailed(t *testing.T) {
	h := newMockHandler()
	h.GetClusterClient = func(ctx context.Context, name string) (client.Client, error) {
		return nil, fmt.Errorf("failed to create client for cluster %s: token expired", name)
	}

	ctx := app.NewContext(16)
	ctx.Params = param.Params{{Key: "name", Value: "cluster-b"}}
	ctx.Request.SetRequestURI("/clusters/cluster-b/restore-classes")

	h.listRestoreClasses(context.Background(), ctx)

	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	var resp struct {
		Code int `json:"code"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeBadRequest, resp.Code)
}

func TestListRestoreClasses_ListFailedAndNoMutation(t *testing.T) {
	clusterName := "cluster-c"
	clusterObj := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: clusterName},
	}
	h := newMockHandler(clusterObj)
	h.GetClusterClient = func(ctx context.Context, name string) (client.Client, error) {
		return &fakeTargetClusterClient{
			listErr: fmt.Errorf("target cluster unreachable"),
		}, nil
	}

	ctx := app.NewContext(16)
	ctx.Params = param.Params{{Key: "name", Value: clusterName}}
	ctx.Request.SetRequestURI("/clusters/cluster-c/restore-classes")

	h.listRestoreClasses(context.Background(), ctx)

	assert.Equal(t, consts.StatusInternalServerError, ctx.Response.StatusCode())
	var resp struct {
		Code int `json:"code"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, transport.CodeInternalServerError, resp.Code)

	actions := h.DisasterClient.(*fake.Clientset).Actions()
	for _, action := range actions {
		assert.NotEqual(t, "create", action.GetVerb())
		assert.NotEqual(t, "update", action.GetVerb())
		assert.NotEqual(t, "patch", action.GetVerb())
		assert.NotEqual(t, "delete", action.GetVerb())
	}
}
