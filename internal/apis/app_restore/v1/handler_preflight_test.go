package apprestore

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/service/verifier"
	"github.com/stretchr/testify/assert"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type mockRestorePreflightVerifier struct {
	result    *verifier.RestorePreflightResult
	err       error
	called    bool
	wait      int
	target    string
	backupRef string
}

func (m *mockRestorePreflightVerifier) VerifyRestorePreflight(ctx context.Context, targetCli ctrclient.Client, mgmtCli ctrclient.Client, appBackup *dapisv1.AppBackup, targetCluster string, waitSeconds int) (*verifier.RestorePreflightResult, error) {
	m.called = true
	m.wait = waitSeconds
	m.target = targetCluster
	if appBackup != nil {
		m.backupRef = appBackup.Name
	}
	return m.result, m.err
}

func TestValidateRestorePreflight(t *testing.T) {
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
	h := newMockRestoreHandler(appBackup)
	mockVerifier := &mockRestorePreflightVerifier{
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
	h.RestorePreflightVerifier = mockVerifier
	h.GetClusterClient = func(ctx context.Context, clusterName string) (ctrclient.Client, error) {
		return nil, nil
	}

	ctx := app.NewContext(16)
	req := ValidateRestorePreflightRequest{
		BackupSource:  "backup-a",
		TargetCluster: "cluster-b",
		WaitSeconds:   12,
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	ctx.Request.SetRequestURI("/apprestores/preflight/validate")

	h.validateRestorePreflight(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
	assert.True(t, mockVerifier.called)
	assert.Equal(t, 12, mockVerifier.wait)
	assert.Equal(t, "cluster-b", mockVerifier.target)
	assert.Equal(t, "backup-a", mockVerifier.backupRef)

	var resp struct {
		Code int               `json:"code"`
		Data bool              `json:"data"`
		Meta map[string]string `json:"meta"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, 0, resp.Code)
	assert.True(t, resp.Data)
	assert.Equal(t, "repo-a-source-a", resp.Meta["required_bsl"])
	assert.Equal(t, "Available", resp.Meta["state"])
}

func TestCreateAppRestoreBlockedByPreflight(t *testing.T) {
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
			Valid:             false,
			RequiredBSL:       "repo-a-source-a",
			SourceCluster:     "source-a",
			TargetCluster:     "cluster-b",
			StorageRepository: "repo-a",
			Phase:             "Unavailable",
			Reason:            "required BSL repo-a-source-a phase is Unavailable",
		},
	}
	h.GetClusterClient = func(ctx context.Context, clusterName string) (ctrclient.Client, error) {
		return nil, nil
	}

	ctx := app.NewContext(16)
	req := CreateAppRestoreRequest{
		Name:         "restore-a",
		BackupSource: "backup-a",
		Cluster:      "cluster-b",
		BackupName:   "backup-object-001",
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	ctx.Request.SetRequestURI("/apprestores")

	h.createAppRestore(context.Background(), ctx)

	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	var resp struct {
		Code    int               `json:"code"`
		Message string            `json:"message"`
		Meta    map[string]string `json:"meta"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, 1000, resp.Code)
	assert.Contains(t, resp.Message, "phase is Unavailable")
	assert.Equal(t, "repo-a-source-a", resp.Meta["required_bsl"])

	_, getErr := h.DisasterClient.DisasterV1().AppRestores(common.DisasterSystemNamespace).Get(context.Background(), "restore-a", metav1.GetOptions{})
	assert.True(t, k8serrors.IsNotFound(getErr))
}
