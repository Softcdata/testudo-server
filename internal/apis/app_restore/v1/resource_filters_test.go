package apprestore

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/service/verifier"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func restoreResourceMapper() meta.RESTMapper {
	mapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{
		{Group: "apps", Version: "v1"},
		{Group: "", Version: "v1"},
	})
	mapper.AddSpecific(
		schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"},
		schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"},
		schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployment"},
		meta.RESTScopeNamespace,
	)
	mapper.AddSpecific(
		schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap"},
		schema.GroupVersionResource{Version: "v1", Resource: "configmaps"},
		schema.GroupVersionResource{Version: "v1", Resource: "configmap"},
		meta.RESTScopeNamespace,
	)
	return mapper
}

func restoreResourceClient(t *testing.T) client.Client {
	t.Helper()
	return fake.NewClientBuilder().WithRESTMapper(restoreResourceMapper()).Build()
}

func validRestorePreflight() *mockRestorePreflightVerifier {
	return &mockRestorePreflightVerifier{result: &verifier.RestorePreflightResult{
		Valid:             true,
		RequiredBSL:       "repo-a-source-a",
		SourceCluster:     "source-a",
		TargetCluster:     "cluster-b",
		StorageRepository: "repo-a",
		Phase:             "Available",
		Reason:            "required BSL is available",
	}}
}

func TestCreateAppRestore_NormalizesIncludedAndExcludedGVK(t *testing.T) {
	h := newMockRestoreHandler(
		&dapisv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster-b"},
			Status:     dapisv1.ClusterStatus{Status: "Ready"},
		},
		&dapisv1.AppBackup{
			ObjectMeta: metav1.ObjectMeta{Name: "backup-a", Namespace: common.DisasterSystemNamespace},
			Spec: dapisv1.AppBackupSpec{
				Cluster:  "source-a",
				Template: velerov1.BackupSpec{StorageLocation: "repo-a"},
			},
		},
	)
	h.RestorePreflightVerifier = validRestorePreflight()
	remoteClient := restoreResourceClient(t)
	h.GetClusterClient = func(ctx context.Context, clusterName string) (client.Client, error) {
		return remoteClient, nil
	}

	ctx := app.NewContext(16)
	ctx.Request.SetBody([]byte(`{
		"name":"restore-gvk-create",
		"backupSource":"backup-a",
		"cluster":"cluster-b",
		"backupName":"backup-object-001",
		"includedResources":["apps/v1/Deployment"],
		"excludedResources":["v1/ConfigMap"],
		"existingResourcePolicy":"update"
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	ctx.Request.SetRequestURI("/apprestores")

	h.createAppRestore(context.Background(), ctx)
	require.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())

	item, err := h.DisasterClient.DisasterV1().AppRestores(common.DisasterSystemNamespace).Get(context.Background(), "restore-gvk-create", metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, []string{"deployments.apps"}, item.Spec.Template.IncludedResources)
	require.Equal(t, []string{"configmaps"}, item.Spec.Template.ExcludedResources)
	require.Equal(t, velerov1.PolicyType("update"), item.Spec.Template.ExistingResourcePolicy)
}

func TestUpdateAppRestore_NormalizesIncludedAndExcludedGVK(t *testing.T) {
	existing := &dapisv1.AppRestore{
		ObjectMeta: metav1.ObjectMeta{Name: "restore-gvk-update", Namespace: common.DisasterSystemNamespace},
		Spec: dapisv1.AppRestoreSpec{
			BackupSource:  "backup-a",
			Cluster:       "cluster-b",
			SourceCluster: "source-a",
			Template:      velerov1.RestoreSpec{BackupName: "backup-object-001"},
		},
	}
	h := newMockRestoreHandler(existing)
	remoteClient := restoreResourceClient(t)
	h.GetClusterClient = func(ctx context.Context, clusterName string) (client.Client, error) {
		return remoteClient, nil
	}

	ctx := app.NewContext(16)
	ctx.Request.SetBody([]byte(`{
		"name":"restore-gvk-update",
		"includedResources":["apps/v1/Deployment"],
		"excludedResources":["v1/ConfigMap"]
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	ctx.Request.SetRequestURI("/apprestores/restore-gvk-update")

	h.updateAppRestore(context.Background(), ctx)
	require.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	item, err := h.DisasterClient.DisasterV1().AppRestores(common.DisasterSystemNamespace).Get(context.Background(), "restore-gvk-update", metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, []string{"deployments.apps"}, item.Spec.Template.IncludedResources)
	require.Equal(t, []string{"configmaps"}, item.Spec.Template.ExcludedResources)
}

func TestUpdateAppRestore_UsesAppBackupSourceClusterWhenRestoreSourceClusterIsEmpty(t *testing.T) {
	existing := &dapisv1.AppRestore{
		ObjectMeta: metav1.ObjectMeta{Name: "restore-gvk-source-cluster", Namespace: common.DisasterSystemNamespace},
		Spec: dapisv1.AppRestoreSpec{
			BackupSource: "backup-a",
			Cluster:      "target-b",
			Template:     velerov1.RestoreSpec{BackupName: "backup-object-001"},
		},
	}
	backup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{Name: "backup-a", Namespace: common.DisasterSystemNamespace},
		Spec: dapisv1.AppBackupSpec{
			Cluster:  "source-a",
			Template: velerov1.BackupSpec{StorageLocation: "repo-a"},
		},
	}
	h := newMockRestoreHandler(existing, backup)
	var mappedCluster string
	h.GetClusterClient = func(ctx context.Context, clusterName string) (client.Client, error) {
		mappedCluster = clusterName
		return restoreResourceClient(t), nil
	}

	ctx := app.NewContext(16)
	ctx.Request.SetBody([]byte(`{
		"name":"restore-gvk-source-cluster",
		"includedResources":["apps/v1/Deployment"]
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	ctx.Request.SetRequestURI("/apprestores/restore-gvk-source-cluster")

	h.updateAppRestore(context.Background(), ctx)
	require.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	require.Equal(t, "source-a", mappedCluster)

	item, err := h.DisasterClient.DisasterV1().AppRestores(common.DisasterSystemNamespace).Get(context.Background(), "restore-gvk-source-cluster", metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, []string{"deployments.apps"}, item.Spec.Template.IncludedResources)
}

func TestUpdateAppRestore_UsesRequestedAppBackupSourceCluster(t *testing.T) {
	existing := &dapisv1.AppRestore{
		ObjectMeta: metav1.ObjectMeta{Name: "restore-gvk-new-source", Namespace: common.DisasterSystemNamespace},
		Spec: dapisv1.AppRestoreSpec{
			BackupSource:  "backup-old",
			Cluster:       "target-b",
			SourceCluster: "source-old",
			Template:      velerov1.RestoreSpec{BackupName: "backup-object-001"},
		},
	}
	backup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{Name: "backup-new", Namespace: common.DisasterSystemNamespace},
		Spec:       dapisv1.AppBackupSpec{Cluster: "source-new"},
	}
	h := newMockRestoreHandler(existing, backup)
	var mappedCluster string
	h.GetClusterClient = func(ctx context.Context, clusterName string) (client.Client, error) {
		mappedCluster = clusterName
		return restoreResourceClient(t), nil
	}

	ctx := app.NewContext(16)
	ctx.Request.SetBody([]byte(`{
		"name":"restore-gvk-new-source",
		"backupSource":"backup-new",
		"includedResources":["apps/v1/Deployment"]
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	ctx.Request.SetRequestURI("/apprestores/restore-gvk-new-source")

	h.updateAppRestore(context.Background(), ctx)
	require.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	require.Equal(t, "source-new", mappedCluster)

	item, err := h.DisasterClient.DisasterV1().AppRestores(common.DisasterSystemNamespace).Get(context.Background(), "restore-gvk-new-source", metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "backup-new", item.Spec.BackupSource)
	require.Equal(t, []string{"deployments.apps"}, item.Spec.Template.IncludedResources)
}

func TestUpdateAppRestore_DoesNotUseTargetClusterWhenSourceClusterCannotBeResolved(t *testing.T) {
	existing := &dapisv1.AppRestore{
		ObjectMeta: metav1.ObjectMeta{Name: "restore-gvk-missing-source", Namespace: common.DisasterSystemNamespace},
		Spec: dapisv1.AppRestoreSpec{
			BackupSource: "missing-backup",
			Cluster:      "target-b",
			Template:     velerov1.RestoreSpec{BackupName: "backup-object-001"},
		},
	}
	h := newMockRestoreHandler(existing)
	clusterClientCalls := 0
	h.GetClusterClient = func(ctx context.Context, clusterName string) (client.Client, error) {
		clusterClientCalls++
		t.Fatalf("unexpected RESTMapper lookup against cluster %q", clusterName)
		return nil, nil
	}

	ctx := app.NewContext(16)
	ctx.Request.SetBody([]byte(`{
		"name":"restore-gvk-missing-source",
		"includedResources":["apps/v1/Deployment"]
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	ctx.Request.SetRequestURI("/apprestores/restore-gvk-missing-source")

	h.updateAppRestore(context.Background(), ctx)
	require.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	require.Contains(t, string(ctx.Response.Body()), "missing-backup")
	require.Equal(t, 0, clusterClientCalls)

	item, err := h.DisasterClient.DisasterV1().AppRestores(common.DisasterSystemNamespace).Get(context.Background(), "restore-gvk-missing-source", metav1.GetOptions{})
	require.NoError(t, err)
	require.Empty(t, item.Spec.Template.IncludedResources)
}

func TestCreateAppRestore_RejectsUnresolvableGVK(t *testing.T) {
	h := newMockRestoreHandler(&dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{Name: "backup-a", Namespace: common.DisasterSystemNamespace},
		Spec: dapisv1.AppBackupSpec{
			Cluster:  "source-a",
			Template: velerov1.BackupSpec{StorageLocation: "repo-a"},
		},
	})
	h.GetClusterClient = func(ctx context.Context, clusterName string) (client.Client, error) {
		return restoreResourceClient(t), nil
	}

	ctx := app.NewContext(16)
	ctx.Request.SetBody([]byte(`{
		"name":"restore-gvk-invalid",
		"backupSource":"backup-a",
		"cluster":"cluster-b",
		"backupName":"backup-object-001",
		"includedResources":["example.io/v1/Widget"]
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	ctx.Request.SetRequestURI("/apprestores")

	h.createAppRestore(context.Background(), ctx)
	require.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	require.Contains(t, string(ctx.Response.Body()), "includedResources")
	require.Contains(t, string(ctx.Response.Body()), "example.io/v1/Widget")

	_, err := h.DisasterClient.DisasterV1().AppRestores(common.DisasterSystemNamespace).Get(context.Background(), "restore-gvk-invalid", metav1.GetOptions{})
	require.Error(t, err)
}

func TestUpdateAppRestore_RejectsUnresolvableGVK(t *testing.T) {
	existing := &dapisv1.AppRestore{
		ObjectMeta: metav1.ObjectMeta{Name: "restore-gvk-invalid-update", Namespace: common.DisasterSystemNamespace},
		Spec: dapisv1.AppRestoreSpec{
			BackupSource:  "backup-a",
			Cluster:       "cluster-b",
			SourceCluster: "source-a",
			Template:      velerov1.RestoreSpec{BackupName: "backup-object-001"},
		},
	}
	h := newMockRestoreHandler(existing)
	h.GetClusterClient = func(ctx context.Context, clusterName string) (client.Client, error) {
		return restoreResourceClient(t), nil
	}

	ctx := app.NewContext(16)
	ctx.Request.SetBody([]byte(`{
		"name":"restore-gvk-invalid-update",
		"includedResources":["example.io/v1/Widget"]
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	ctx.Request.SetRequestURI("/apprestores/restore-gvk-invalid-update")

	h.updateAppRestore(context.Background(), ctx)
	require.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	require.Contains(t, string(ctx.Response.Body()), "includedResources")
	require.Contains(t, string(ctx.Response.Body()), "example.io/v1/Widget")

	item, err := h.DisasterClient.DisasterV1().AppRestores(common.DisasterSystemNamespace).Get(context.Background(), "restore-gvk-invalid-update", metav1.GetOptions{})
	require.NoError(t, err)
	require.Empty(t, item.Spec.Template.IncludedResources)
}
