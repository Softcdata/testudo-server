package verifier

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	metadata "github.com/softcdata/testudo-operator/pkg/metadata"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/stretchr/testify/assert"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type memoryClient struct {
	mu sync.RWMutex

	scheme *runtime.Scheme

	clusters map[string]*dapisv1.Cluster
	bsls     map[string]*velerov1.BackupStorageLocation
}

func newMemoryClient(scheme *runtime.Scheme, objs ...ctrclient.Object) *memoryClient {
	c := &memoryClient{
		scheme:   scheme,
		clusters: make(map[string]*dapisv1.Cluster),
		bsls:     make(map[string]*velerov1.BackupStorageLocation),
	}
	for _, obj := range objs {
		_ = c.Create(context.Background(), obj)
	}
	return c
}

func bslKey(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

func (m *memoryClient) Get(_ context.Context, key ctrclient.ObjectKey, obj ctrclient.Object, _ ...ctrclient.GetOption) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	switch out := obj.(type) {
	case *dapisv1.Cluster:
		stored, ok := m.clusters[key.Name]
		if !ok {
			return apierrors.NewNotFound(schema.GroupResource{Group: dapisv1.GroupVersion.Group, Resource: "clusters"}, key.Name)
		}
		*out = *stored.DeepCopy()
		return nil
	case *velerov1.BackupStorageLocation:
		stored, ok := m.bsls[bslKey(key.Namespace, key.Name)]
		if !ok {
			return apierrors.NewNotFound(schema.GroupResource{Group: "velero.io", Resource: "backupstoragelocations"}, key.Name)
		}
		*out = *stored.DeepCopy()
		return nil
	default:
		return fmt.Errorf("unsupported object type for Get: %T", obj)
	}
}

func (m *memoryClient) List(_ context.Context, _ ctrclient.ObjectList, _ ...ctrclient.ListOption) error {
	return nil
}

func (m *memoryClient) Create(_ context.Context, obj ctrclient.Object, _ ...ctrclient.CreateOption) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch in := obj.(type) {
	case *dapisv1.Cluster:
		m.clusters[in.Name] = in.DeepCopy()
		return nil
	case *velerov1.BackupStorageLocation:
		m.bsls[bslKey(in.Namespace, in.Name)] = in.DeepCopy()
		return nil
	default:
		return fmt.Errorf("unsupported object type for Create: %T", obj)
	}
}

func (m *memoryClient) Delete(_ context.Context, _ ctrclient.Object, _ ...ctrclient.DeleteOption) error {
	return nil
}

func (m *memoryClient) Update(_ context.Context, obj ctrclient.Object, _ ...ctrclient.UpdateOption) error {
	return m.Create(context.Background(), obj)
}

func (m *memoryClient) Patch(_ context.Context, obj ctrclient.Object, _ ctrclient.Patch, _ ...ctrclient.PatchOption) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch in := obj.(type) {
	case *dapisv1.Cluster:
		m.clusters[in.Name] = in.DeepCopy()
		return nil
	default:
		return fmt.Errorf("unsupported object type for Patch: %T", obj)
	}
}

func (m *memoryClient) DeleteAllOf(_ context.Context, _ ctrclient.Object, _ ...ctrclient.DeleteAllOfOption) error {
	return nil
}

func (m *memoryClient) Status() ctrclient.SubResourceWriter {
	return &noopSubResourceClient{}
}

func (m *memoryClient) SubResource(_ string) ctrclient.SubResourceClient {
	return &noopSubResourceClient{}
}

func (m *memoryClient) Scheme() *runtime.Scheme {
	return m.scheme
}

func (m *memoryClient) RESTMapper() meta.RESTMapper {
	return nil
}

func (m *memoryClient) GroupVersionKindFor(_ runtime.Object) (schema.GroupVersionKind, error) {
	return schema.GroupVersionKind{}, nil
}

func (m *memoryClient) IsObjectNamespaced(_ runtime.Object) (bool, error) {
	return true, nil
}

type noopSubResourceClient struct{}

func (n *noopSubResourceClient) Get(_ context.Context, _ ctrclient.Object, _ ctrclient.Object, _ ...ctrclient.SubResourceGetOption) error {
	return nil
}

func (n *noopSubResourceClient) Create(_ context.Context, _ ctrclient.Object, _ ctrclient.Object, _ ...ctrclient.SubResourceCreateOption) error {
	return nil
}

func (n *noopSubResourceClient) Update(_ context.Context, _ ctrclient.Object, _ ...ctrclient.SubResourceUpdateOption) error {
	return nil
}

func (n *noopSubResourceClient) Patch(_ context.Context, _ ctrclient.Object, _ ctrclient.Patch, _ ...ctrclient.SubResourcePatchOption) error {
	return nil
}

func buildRestorePreflightScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	assert.NoError(t, dapisv1.AddToScheme(scheme))
	assert.NoError(t, velerov1.AddToScheme(scheme))
	return scheme
}

func buildAppBackup(name, sourceCluster, storageRepository string) *dapisv1.AppBackup {
	return &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: common.DisasterSystemNamespace,
		},
		Spec: dapisv1.AppBackupSpec{
			Cluster: sourceCluster,
			Template: velerov1.BackupSpec{
				StorageLocation: storageRepository,
			},
		},
	}
}

func TestNormalizeRestorePreflightWaitSeconds(t *testing.T) {
	assert.Equal(t, 20, NormalizeRestorePreflightWaitSeconds(0))
	assert.Equal(t, 20, NormalizeRestorePreflightWaitSeconds(-1))
	assert.Equal(t, 5, NormalizeRestorePreflightWaitSeconds(5))
	assert.Equal(t, 60, NormalizeRestorePreflightWaitSeconds(120))
}

func TestVerifyRestorePreflight_Available(t *testing.T) {
	scheme := buildRestorePreflightScheme(t)
	appBackup := buildAppBackup("backup-a", "source-a", "repo-a")
	targetCluster := "cluster-b"
	requiredBSL := "repo-a-source-a"

	targetCli := newMemoryClient(scheme, &velerov1.BackupStorageLocation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      requiredBSL,
			Namespace: common.VeleroNamespace,
		},
		Status: velerov1.BackupStorageLocationStatus{
			Phase: velerov1.BackupStorageLocationPhaseAvailable,
		},
	})
	mgmtCli := newMemoryClient(scheme)

	v := NewRestorePreflightVerifier()
	result, err := v.VerifyRestorePreflight(context.Background(), targetCli, mgmtCli, appBackup, targetCluster, 1)
	assert.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Equal(t, requiredBSL, result.RequiredBSL)
	assert.Equal(t, "source-a", result.SourceCluster)
	assert.Equal(t, "repo-a", result.StorageRepository)
	assert.Equal(t, targetCluster, result.TargetCluster)
	assert.Equal(t, string(velerov1.BackupStorageLocationPhaseAvailable), result.Phase)
}

func TestVerifyRestorePreflight_Unavailable(t *testing.T) {
	scheme := buildRestorePreflightScheme(t)
	appBackup := buildAppBackup("backup-a", "source-a", "repo-a")
	targetCluster := "cluster-b"
	requiredBSL := "repo-a-source-a"

	targetCli := newMemoryClient(scheme, &velerov1.BackupStorageLocation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      requiredBSL,
			Namespace: common.VeleroNamespace,
		},
		Status: velerov1.BackupStorageLocationStatus{
			Phase: velerov1.BackupStorageLocationPhaseUnavailable,
		},
	})
	mgmtCli := newMemoryClient(scheme)

	v := NewRestorePreflightVerifier()
	result, err := v.VerifyRestorePreflight(context.Background(), targetCli, mgmtCli, appBackup, targetCluster, 1)
	assert.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Equal(t, string(velerov1.BackupStorageLocationPhaseUnavailable), result.Phase)
	assert.Contains(t, result.Reason, "phase is Unavailable")
}

func TestVerifyRestorePreflight_NotFoundSignalThenReady(t *testing.T) {
	scheme := buildRestorePreflightScheme(t)
	appBackup := buildAppBackup("backup-a", "source-a", "repo-a")
	targetCluster := "cluster-b"
	requiredBSL := "repo-a-source-a"

	targetCli := newMemoryClient(scheme)
	mgmtCli := newMemoryClient(scheme, &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: targetCluster,
		},
	})

	go func(cli *memoryClient) {
		time.Sleep(200 * time.Millisecond)
		_ = cli.Create(context.Background(), &velerov1.BackupStorageLocation{
			ObjectMeta: metav1.ObjectMeta{
				Name:      requiredBSL,
				Namespace: common.VeleroNamespace,
			},
			Status: velerov1.BackupStorageLocationStatus{
				Phase: velerov1.BackupStorageLocationPhaseAvailable,
			},
		})
	}(targetCli)

	v := NewRestorePreflightVerifier()
	result, err := v.VerifyRestorePreflight(context.Background(), targetCli, mgmtCli, appBackup, targetCluster, 2)
	assert.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Equal(t, requiredBSL, result.RequiredBSL)

	updatedCluster := &dapisv1.Cluster{}
	assert.NoError(t, mgmtCli.Get(context.Background(), types.NamespacedName{Name: targetCluster}, updatedCluster))
	assert.Equal(t, "repo-a", updatedCluster.Annotations[metadata.AnnotationEnsureStorage])
	assert.Equal(t, "source-a", updatedCluster.Annotations[annotationEnsureStorageSourceCluster])
}

func TestVerifyRestorePreflight_Timeout(t *testing.T) {
	scheme := buildRestorePreflightScheme(t)
	appBackup := buildAppBackup("backup-a", "source-a", "repo-a")
	targetCluster := "cluster-b"

	targetCli := newMemoryClient(scheme)
	mgmtCli := newMemoryClient(scheme, &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: targetCluster,
		},
	})

	v := NewRestorePreflightVerifier()
	result, err := v.VerifyRestorePreflight(context.Background(), targetCli, mgmtCli, appBackup, targetCluster, 1)
	assert.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Equal(t, "NotFound", result.Phase)
	assert.Contains(t, result.Reason, "timeout waiting for required BSL")
}
