package verifier

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	metadata "github.com/softcdata/testudo-operator/pkg/metadata"
	"github.com/softcdata/testudo-server/internal/common"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultRestorePreflightWaitSeconds = 20
	maxRestorePreflightWaitSeconds     = 60

	annotationEnsureStorageSourceCluster = "testudo.softcdata.com/ensure-storage-source-cluster"
)

// RestorePreflightResult is the structured output for app restore preflight validation.
type RestorePreflightResult struct {
	Valid             bool   `json:"valid"`
	RequiredBSL       string `json:"requiredBSL"`
	SourceCluster     string `json:"sourceCluster"`
	TargetCluster     string `json:"targetCluster"`
	StorageRepository string `json:"storageRepository"`
	Phase             string `json:"phase"`
	Reason            string `json:"reason"`
}

// RestorePreflightVerifier defines interface for cross-cluster app restore preflight verification.
type RestorePreflightVerifier interface {
	VerifyRestorePreflight(ctx context.Context, targetCli client.Client, mgmtCli client.Client, appBackup *dapisv1.AppBackup, targetCluster string, waitSeconds int) (*RestorePreflightResult, error)
}

type DefaultRestorePreflightVerifier struct{}

// NewRestorePreflightVerifier creates a new restore preflight verifier.
func NewRestorePreflightVerifier() RestorePreflightVerifier {
	return &DefaultRestorePreflightVerifier{}
}

// NormalizeRestorePreflightWaitSeconds applies default and max boundaries for preflight polling.
func NormalizeRestorePreflightWaitSeconds(waitSeconds int) int {
	if waitSeconds <= 0 {
		return defaultRestorePreflightWaitSeconds
	}
	if waitSeconds > maxRestorePreflightWaitSeconds {
		return maxRestorePreflightWaitSeconds
	}
	return waitSeconds
}

// VerifyRestorePreflight validates required BSL for cross-cluster restore.
func (v *DefaultRestorePreflightVerifier) VerifyRestorePreflight(ctx context.Context, targetCli client.Client, mgmtCli client.Client, appBackup *dapisv1.AppBackup, targetCluster string, waitSeconds int) (*RestorePreflightResult, error) {
	result := &RestorePreflightResult{
		TargetCluster: targetCluster,
	}

	if appBackup == nil {
		result.Reason = "app backup is required"
		return result, nil
	}
	if targetCli == nil {
		result.Reason = "target cluster client is required"
		return result, errors.New(result.Reason)
	}
	if mgmtCli == nil {
		result.Reason = "management cluster client is required"
		return result, errors.New(result.Reason)
	}

	sourceCluster := strings.TrimSpace(appBackup.Spec.Cluster)
	storageRepository := strings.TrimSpace(appBackup.Spec.Template.StorageLocation)
	result.SourceCluster = sourceCluster
	result.StorageRepository = storageRepository

	if sourceCluster == "" || storageRepository == "" {
		result.Reason = fmt.Sprintf("AppBackup %s is missing required fields", appBackup.Name)
		return result, nil
	}

	requiredBSL := fmt.Sprintf("%s-%s", storageRepository, sourceCluster)
	result.RequiredBSL = requiredBSL

	bslPhase, bslExists, err := getBSLPhase(ctx, targetCli, requiredBSL)
	if err != nil {
		result.Reason = fmt.Sprintf("failed to get BackupStorageLocation %s: %v", requiredBSL, err)
		return result, err
	}
	if bslExists {
		result.Phase = string(bslPhase)
		if bslPhase == velerov1.BackupStorageLocationPhaseAvailable {
			result.Valid = true
			result.Reason = "required BSL is available"
			return result, nil
		}
		if bslPhase == velerov1.BackupStorageLocationPhaseUnavailable {
			result.Reason = fmt.Sprintf("required BSL %s phase is %s", requiredBSL, bslPhase)
			return result, nil
		}
	} else if err := patchEnsureStorageSignal(ctx, mgmtCli, targetCluster, storageRepository, sourceCluster); err != nil {
		result.Reason = fmt.Sprintf("failed to patch ensure-storage signal: %v", err)
		return result, err
	}

	maxChecks := NormalizeRestorePreflightWaitSeconds(waitSeconds)
	for i := 0; i < maxChecks; i++ {
		time.Sleep(1 * time.Second)
		bslPhase, bslExists, err = getBSLPhase(ctx, targetCli, requiredBSL)
		if err != nil {
			result.Reason = fmt.Sprintf("failed to get BackupStorageLocation %s: %v", requiredBSL, err)
			return result, err
		}
		if !bslExists {
			continue
		}

		result.Phase = string(bslPhase)
		if bslPhase == velerov1.BackupStorageLocationPhaseAvailable {
			result.Valid = true
			result.Reason = "required BSL is available"
			return result, nil
		}
		if bslPhase == velerov1.BackupStorageLocationPhaseUnavailable {
			result.Reason = fmt.Sprintf("required BSL %s phase is %s", requiredBSL, bslPhase)
			return result, nil
		}
	}

	if result.Phase == "" {
		result.Phase = "NotFound"
	}
	result.Reason = fmt.Sprintf("timeout waiting for required BSL %s to become Available", requiredBSL)
	return result, nil
}

func getBSLPhase(ctx context.Context, targetCli client.Client, bslName string) (velerov1.BackupStorageLocationPhase, bool, error) {
	bsl := &velerov1.BackupStorageLocation{}
	err := targetCli.Get(ctx, types.NamespacedName{Name: bslName, Namespace: common.VeleroNamespace}, bsl)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return "", false, nil
		}
		return "", false, err
	}
	return bsl.Status.Phase, true, nil
}

func patchEnsureStorageSignal(ctx context.Context, mgmtCli client.Client, targetCluster, storageRepository, sourceCluster string) error {
	cluster := &dapisv1.Cluster{}
	if err := mgmtCli.Get(ctx, types.NamespacedName{Name: targetCluster}, cluster); err != nil {
		return err
	}

	patch := client.MergeFrom(cluster.DeepCopy())
	if cluster.Annotations == nil {
		cluster.Annotations = make(map[string]string)
	}
	cluster.Annotations[metadata.AnnotationEnsureStorage] = storageRepository
	cluster.Annotations[annotationEnsureStorageSourceCluster] = sourceCluster

	return mgmtCli.Patch(ctx, cluster, patch)
}
