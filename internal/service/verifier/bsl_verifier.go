package verifier

import (
	"context"
	"fmt"
	"time"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	metadata "github.com/softcdata/testudo-operator/pkg/metadata"
	"github.com/softcdata/testudo-server/internal/common"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BSLVerifier defines interface for BSL verification
type BSLVerifier interface {
	VerifyBSL(ctx context.Context, cli client.Client, mgmtCli client.Client, storageName, clusterName string) (bool, string, error)
}

type DefaultBSLVerifier struct{}

// NewBSLVerifier creates a new verifier
func NewBSLVerifier() BSLVerifier {
	return &DefaultBSLVerifier{}
}

// VerifyBSL checks if BSL is available on remote cluster
// BSL Name convention: {storage_name}-{cluster_name}
func (v *DefaultBSLVerifier) VerifyBSL(ctx context.Context, cli client.Client, mgmtCli client.Client, storageName, clusterName string) (bool, string, error) {
	bslName := fmt.Sprintf("%s-%s", storageName, clusterName)
	bsl := &velerov1.BackupStorageLocation{}

	// Use global Velero Namespace convention
	veleroNamespace := common.VeleroNamespace

	checkBSL := func() (bool, string, error) {
		err := cli.Get(ctx, types.NamespacedName{Name: bslName, Namespace: veleroNamespace}, bsl)
		if err != nil {
			return false, "", err
		}
		if bsl.Status.Phase == velerov1.BackupStorageLocationPhaseAvailable {
			return true, "Connection is available", nil
		}
		return false, fmt.Sprintf("BackupStorageLocation phase is %s", bsl.Status.Phase), nil
	}

	// 1. First check
	if valid, msg, err := checkBSL(); err == nil {
		if valid {
			return true, msg, nil
		}
		// If BSL exists but not available, we return status directly (maybe auth failed)
		return false, msg, nil
	} else if !apierrors.IsNotFound(err) {
		return false, fmt.Sprintf("Failed to get BackupStorageLocation: %v", err), err
	}

	// 2. BSL Not Found - Signal Operator
	cluster := &dapisv1.Cluster{}
	if err := mgmtCli.Get(ctx, types.NamespacedName{Name: clusterName}, cluster); err != nil {
		return false, fmt.Sprintf("Failed to get Cluster %s: %v", clusterName, err), err
	}

	// Patch Annotation
	patch := client.MergeFrom(cluster.DeepCopy())
	if cluster.Annotations == nil {
		cluster.Annotations = make(map[string]string)
	}
	cluster.Annotations[metadata.AnnotationEnsureStorage] = storageName
	if err := mgmtCli.Patch(ctx, cluster, patch); err != nil {
		return false, fmt.Sprintf("Failed to patch ensure-storage annotation: %v", err), err
	}

	// 3. Poll for BSL creation and readiness
	// Wait up to 10 seconds
	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)
		if valid, msg, err := checkBSL(); err == nil {
			if valid {
				return true, msg, nil
			}
			// If Phase is Unavailable, keep waiting? Or return failure?
			// Velero might take time to validate. Unavailable might be transient initially?
			// Usually valid failed auth returns Unavailable immediately.
			// But creating... might be "New".
			if bsl.Status.Phase != "" {
				return false, msg, nil
			}
			// Phase empty -> Waiting
		}
	}

	return false, "Timeout waiting for BackupStorageLocation to be ready", nil
}
