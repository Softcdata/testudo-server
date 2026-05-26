package apprestore

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route/param"
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/service/verifier"
	"github.com/stretchr/testify/assert"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestResolveStorageClassMapping(t *testing.T) {
	newField := map[string]string{"standard": "gold"}
	oldField := map[string]string{"standard": "gold"}
	conflictOldField := map[string]string{"standard": "silver"}

	resolved, err := resolveStorageClassMapping(newField, nil)
	assert.NoError(t, err)
	assert.Equal(t, newField, resolved)

	resolved, err = resolveStorageClassMapping(nil, oldField)
	assert.NoError(t, err)
	assert.Equal(t, oldField, resolved)

	resolved, err = resolveStorageClassMapping(newField, oldField)
	assert.NoError(t, err)
	assert.Equal(t, newField, resolved)

	resolved, err = resolveStorageClassMapping(newField, conflictOldField)
	assert.Nil(t, resolved)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "storageClassMapping and scMapping conflict")
}

func TestCreateAppRestore_UsesStorageClassMapping(t *testing.T) {
	cluster := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster-b",
		},
		Status: dapisv1.ClusterStatus{
			Status: "Ready",
		},
	}
	appBackup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "backup-a",
			Namespace: common.DisasterSystemNamespace,
		},
		Spec: dapisv1.AppBackupSpec{
			Cluster: "source-a",
			Template: velerov1.BackupSpec{
				StorageLocation: "repo-a",
			},
		},
	}
	h := newMockRestoreHandler(cluster, appBackup)
	h.RestorePreflightVerifier = &mockRestorePreflightVerifier{
		result: &verifier.RestorePreflightResult{
			Valid:             true,
			RequiredBSL:       "repo-a-source-a",
			SourceCluster:     "source-a",
			TargetCluster:     "cluster-b",
			StorageRepository: "repo-a",
			Phase:             "Available",
			Reason:            "required BSL is available",
		},
	}
	h.GetClusterClient = func(ctx context.Context, clusterName string) (ctrclient.Client, error) {
		return nil, nil
	}

	ctx := app.NewContext(16)
	req := CreateAppRestoreRequest{
		Name:                "restore-storage-map",
		BackupSource:        "backup-a",
		Cluster:             "cluster-b",
		BackupName:          "backup-object-001",
		StorageClassMapping: map[string]string{"standard": "gold"},
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	ctx.Request.SetRequestURI("/apprestores")

	h.createAppRestore(context.Background(), ctx)
	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())

	item, err := h.DisasterClient.DisasterV1().AppRestores(common.DisasterSystemNamespace).Get(context.Background(), "restore-storage-map", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.True(t, hasStorageClassMappingRule(item.Spec.ResourceModifierRules, "standard", "gold"))
}

func TestCreateAppRestore_UsesSCMappingAlias(t *testing.T) {
	cluster := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster-b",
		},
		Status: dapisv1.ClusterStatus{
			Status: "Ready",
		},
	}
	appBackup := &dapisv1.AppBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "backup-a",
			Namespace: common.DisasterSystemNamespace,
		},
		Spec: dapisv1.AppBackupSpec{
			Cluster: "source-a",
			Template: velerov1.BackupSpec{
				StorageLocation: "repo-a",
			},
		},
	}
	h := newMockRestoreHandler(cluster, appBackup)
	h.RestorePreflightVerifier = &mockRestorePreflightVerifier{
		result: &verifier.RestorePreflightResult{
			Valid:             true,
			RequiredBSL:       "repo-a-source-a",
			SourceCluster:     "source-a",
			TargetCluster:     "cluster-b",
			StorageRepository: "repo-a",
			Phase:             "Available",
			Reason:            "required BSL is available",
		},
	}
	h.GetClusterClient = func(ctx context.Context, clusterName string) (ctrclient.Client, error) {
		return nil, nil
	}

	ctx := app.NewContext(16)
	req := CreateAppRestoreRequest{
		Name:         "restore-sc-alias",
		BackupSource: "backup-a",
		Cluster:      "cluster-b",
		BackupName:   "backup-object-001",
		SCMapping:    map[string]string{"standard": "silver"},
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	ctx.Request.SetRequestURI("/apprestores")

	h.createAppRestore(context.Background(), ctx)
	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())

	item, err := h.DisasterClient.DisasterV1().AppRestores(common.DisasterSystemNamespace).Get(context.Background(), "restore-sc-alias", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.True(t, hasStorageClassMappingRule(item.Spec.ResourceModifierRules, "standard", "silver"))
}

func TestCreateAppRestore_StorageClassMappingConflictReturnsBadRequest(t *testing.T) {
	h := newMockRestoreHandler()

	ctx := app.NewContext(16)
	ctx.Request.SetBody([]byte(`{
		"name":"restore-conflict",
		"backupSource":"backup-a",
		"cluster":"cluster-b",
		"storageClassMapping":{"standard":"gold"},
		"scMapping":{"standard":"silver"}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	ctx.Request.SetRequestURI("/apprestores")

	h.createAppRestore(context.Background(), ctx)
	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	assert.Contains(t, string(ctx.Response.Body()), "storageClassMapping and scMapping conflict")
}

func TestUpdateAppRestore_UsesStorageClassMapping(t *testing.T) {
	existing := &dapisv1.AppRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "restore-update-storage-map",
			Namespace: common.DisasterSystemNamespace,
		},
		Spec: dapisv1.AppRestoreSpec{
			BackupSource: "backup-a",
			Cluster:      "cluster-b",
			Template: velerov1.RestoreSpec{
				BackupName: "backup-object-001",
			},
		},
	}
	h := newMockRestoreHandler(existing)

	ctx := app.NewContext(16)
	ctx.Request.SetBody([]byte(`{
		"name":"restore-update-storage-map",
		"storageClassMapping":{"standard":"gold"}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	ctx.Request.SetRequestURI("/apprestores/restore-update-storage-map")

	h.updateAppRestore(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	updated, err := h.DisasterClient.DisasterV1().AppRestores(common.DisasterSystemNamespace).Get(context.Background(), "restore-update-storage-map", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.True(t, hasStorageClassMappingRule(updated.Spec.ResourceModifierRules, "standard", "gold"))
}

func TestUpdateAppRestore_UsesSCMappingAlias(t *testing.T) {
	existing := &dapisv1.AppRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "restore-update-sc-alias",
			Namespace: common.DisasterSystemNamespace,
		},
		Spec: dapisv1.AppRestoreSpec{
			BackupSource: "backup-a",
			Cluster:      "cluster-b",
			Template: velerov1.RestoreSpec{
				BackupName: "backup-object-001",
			},
		},
	}
	h := newMockRestoreHandler(existing)

	ctx := app.NewContext(16)
	ctx.Request.SetBody([]byte(`{
		"name":"restore-update-sc-alias",
		"scMapping":{"standard":"silver"}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	ctx.Request.SetRequestURI("/apprestores/restore-update-sc-alias")

	h.updateAppRestore(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	updated, err := h.DisasterClient.DisasterV1().AppRestores(common.DisasterSystemNamespace).Get(context.Background(), "restore-update-sc-alias", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.True(t, hasStorageClassMappingRule(updated.Spec.ResourceModifierRules, "standard", "silver"))
}

func TestUpdateAppRestore_StorageClassMappingConflictReturnsBadRequest(t *testing.T) {
	existing := &dapisv1.AppRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "restore-update-conflict",
			Namespace: common.DisasterSystemNamespace,
		},
		Spec: dapisv1.AppRestoreSpec{
			BackupSource: "backup-a",
			Cluster:      "cluster-b",
			Template: velerov1.RestoreSpec{
				BackupName: "backup-object-001",
			},
		},
	}
	h := newMockRestoreHandler(existing)

	ctx := app.NewContext(16)
	ctx.Request.SetBody([]byte(`{
		"name":"restore-update-conflict",
		"storageClassMapping":{"standard":"gold"},
		"scMapping":{"standard":"silver"}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	ctx.Request.SetRequestURI("/apprestores/restore-update-conflict")

	h.updateAppRestore(context.Background(), ctx)
	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	assert.Contains(t, string(ctx.Response.Body()), "storageClassMapping and scMapping conflict")

	_, err := h.DisasterClient.DisasterV1().AppRestores(common.DisasterSystemNamespace).Get(context.Background(), "restore-update-conflict", metav1.GetOptions{})
	assert.NoError(t, err)
}

func TestUpdateAppRestore_PathNameOnlyBodyWithoutName(t *testing.T) {
	existing := &dapisv1.AppRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "restore-update-path-only",
			Namespace: common.DisasterSystemNamespace,
		},
		Spec: dapisv1.AppRestoreSpec{
			BackupSource: "backup-a",
			Cluster:      "cluster-b",
			Template: velerov1.RestoreSpec{
				BackupName: "backup-object-001",
			},
		},
	}
	h := newMockRestoreHandler(existing)

	ctx := app.NewContext(16)
	ctx.Request.SetBody([]byte(`{
		"storageClassMapping":{"standard":"gold"}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	ctx.Request.SetRequestURI("/apprestores/restore-update-path-only")
	ctx.Params = param.Params{{Key: "name", Value: "restore-update-path-only"}}

	h.updateAppRestore(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	updated, err := h.DisasterClient.DisasterV1().AppRestores(common.DisasterSystemNamespace).Get(context.Background(), "restore-update-path-only", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.True(t, hasStorageClassMappingRule(updated.Spec.ResourceModifierRules, "standard", "gold"))
}

func TestUpdateAppRestore_PathAndBodyNameMismatchReturnsBadRequest(t *testing.T) {
	existing := &dapisv1.AppRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "restore-update-name-a",
			Namespace: common.DisasterSystemNamespace,
		},
		Spec: dapisv1.AppRestoreSpec{
			BackupSource: "backup-a",
			Cluster:      "cluster-b",
			Template: velerov1.RestoreSpec{
				BackupName: "backup-object-001",
			},
		},
	}
	h := newMockRestoreHandler(existing)

	ctx := app.NewContext(16)
	ctx.Request.SetBody([]byte(`{
		"name":"restore-update-name-b",
		"storageClassMapping":{"standard":"gold"}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	ctx.Request.SetRequestURI("/apprestores/restore-update-name-a")
	ctx.Params = param.Params{{Key: "name", Value: "restore-update-name-a"}}

	h.updateAppRestore(context.Background(), ctx)
	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	assert.Contains(t, string(ctx.Response.Body()), `"message_key":"validation.name_url_body_mismatch"`)
}

func TestUpdateAppRestore_MissingPathAndBodyNameReturnsBadRequest(t *testing.T) {
	existing := &dapisv1.AppRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "restore-update-missing-name",
			Namespace: common.DisasterSystemNamespace,
		},
		Spec: dapisv1.AppRestoreSpec{
			BackupSource: "backup-a",
			Cluster:      "cluster-b",
			Template: velerov1.RestoreSpec{
				BackupName: "backup-object-001",
			},
		},
	}
	h := newMockRestoreHandler(existing)

	ctx := app.NewContext(16)
	ctx.Request.SetBody([]byte(`{
		"storageClassMapping":{"standard":"gold"}
	}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	ctx.Request.SetRequestURI("/apprestores")

	h.updateAppRestore(context.Background(), ctx)
	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	assert.Contains(t, string(ctx.Response.Body()), `"message_key":"validation.name_required"`)
}

func hasStorageClassMappingRule(rules []dapisv1.ResourceModifierRule, sourceClass, targetClass string) bool {
	for _, rule := range rules {
		if rule.Conditions.GroupResource != "persistentvolumeclaims" {
			continue
		}
		hasTest := false
		hasReplace := false
		for _, patch := range rule.Patches {
			if patch.Operation == "test" && patch.Path == "/spec/storageClassName" && patch.Value == sourceClass {
				hasTest = true
			}
			if patch.Operation == "replace" && patch.Path == "/spec/storageClassName" && patch.Value == targetClass {
				hasReplace = true
			}
		}
		if hasTest && hasReplace {
			return true
		}
	}
	return false
}
