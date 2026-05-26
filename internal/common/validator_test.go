package common

import (
	"context"
	"testing"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-operator/pkg/clientset/versioned/fake"
	"github.com/softcdata/testudo-server/internal/kube"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestValidateClusterReady(t *testing.T) {
	tests := []struct {
		name        string
		clusterName string
		objects     []runtime.Object
		wantErr     bool
		errContains string
	}{
		{
			name:        "Cluster exists and is ready",
			clusterName: "test-cluster",
			objects: []runtime.Object{
				&dapisv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster",
					},
					Status: dapisv1.ClusterStatus{
						Status: "Ready",
					},
				},
			},
			wantErr: false,
		},
		{
			name:        "Cluster exists but is not ready",
			clusterName: "test-cluster",
			objects: []runtime.Object{
				&dapisv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster",
					},
					Status: dapisv1.ClusterStatus{
						Status: "NotReady",
					},
				},
			},
			wantErr:     true,
			errContains: "is not ready",
		},
		{
			name:        "Cluster does not exist",
			clusterName: "non-existent-cluster",
			objects:     []runtime.Object{},
			wantErr:     true,
			errContains: "not found",
		},
		{
			name:        "Cluster name is empty",
			clusterName: "",
			objects:     []runtime.Object{},
			wantErr:     true,
			errContains: "cluster name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewSimpleClientset(tt.objects...)
			kubeClient := &kube.KubeClient{
				DisasterClient: fakeClient,
			}

			err := ValidateClusterReady(context.TODO(), kubeClient, tt.clusterName)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateStorageRepositoryAvailable(t *testing.T) {
	tests := []struct {
		name        string
		repoName    string
		objects     []runtime.Object
		wantErr     bool
		errContains string
	}{
		{
			name:     "StorageRepository exists and is available",
			repoName: "test-repo",
			objects: []runtime.Object{
				&dapisv1.StorageRepository{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-repo",
						Namespace: DisasterSystemNamespace,
					},
					Status: dapisv1.StorageRepositoryStatus{
						Status: dapisv1.StorageRepositoryStatusAvailable,
					},
				},
			},
			wantErr: false,
		},
		{
			name:     "StorageRepository exists but is not available",
			repoName: "test-repo",
			objects: []runtime.Object{
				&dapisv1.StorageRepository{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-repo",
						Namespace: DisasterSystemNamespace,
					},
					Status: dapisv1.StorageRepositoryStatus{
						Status: "Unavailable",
					},
				},
			},
			wantErr:     true,
			errContains: "is not available",
		},
		{
			name:        "StorageRepository does not exist",
			repoName:    "non-existent-repo",
			objects:     []runtime.Object{},
			wantErr:     true,
			errContains: "not found",
		},
		{
			name:        "StorageRepository name is empty",
			repoName:    "",
			objects:     []runtime.Object{},
			wantErr:     true,
			errContains: "storage repository name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewSimpleClientset(tt.objects...)
			kubeClient := &kube.KubeClient{
				DisasterClient: fakeClient,
			}

			err := ValidateStorageRepositoryAvailable(context.TODO(), kubeClient, tt.repoName)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
