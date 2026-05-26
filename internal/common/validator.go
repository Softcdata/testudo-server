package common

import (
	"context"
	"fmt"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-server/internal/kube"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ValidateClusterReady validates that a Cluster resource exists and is in Ready status.
// Returns an error with a descriptive message if validation fails.
func ValidateClusterReady(ctx context.Context, client *kube.KubeClient, clusterName string) error {
	if clusterName == "" {
		return fmt.Errorf("cluster name is required")
	}

	cluster, err := client.DisasterClient.DisasterV1().Clusters().Get(ctx, clusterName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("Cluster not found: %s", clusterName)
		}
		return fmt.Errorf("failed to get Cluster %s: %w", clusterName, err)
	}

	if cluster.Status.Status != "Ready" {
		return fmt.Errorf("Cluster %s is not ready (current status: %s)", clusterName, cluster.Status.Status)
	}

	return nil
}

// ValidateStorageRepositoryAvailable validates that a StorageRepository resource exists and is in Available status.
// Returns an error with a descriptive message if validation fails.
func ValidateStorageRepositoryAvailable(ctx context.Context, client *kube.KubeClient, repoName string) error {
	if repoName == "" {
		return fmt.Errorf("storage repository name is required")
	}

	repo, err := client.DisasterClient.DisasterV1().StorageRepositories(DisasterSystemNamespace).Get(ctx, repoName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("StorageRepository not found: %s", repoName)
		}
		return fmt.Errorf("failed to get StorageRepository %s: %w", repoName, err)
	}

	if repo.Status.Status != dapisv1.StorageRepositoryStatusAvailable {
		return fmt.Errorf("StorageRepository %s is not available (current status: %s)", repoName, repo.Status.Status)
	}

	return nil
}
