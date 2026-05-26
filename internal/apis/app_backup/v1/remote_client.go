package appbackup

import (
	"context"
	"fmt"
	"strings"

	"github.com/softcdata/testudo-server/internal/kube"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type remoteClientGetter func(ctx context.Context, clusterName string) (ctrlclient.Reader, error)

func defaultRemoteClientGetter(kc *kube.KubeClient) remoteClientGetter {
	return func(ctx context.Context, clusterName string) (ctrlclient.Reader, error) {
		if kc == nil || kc.ClusterClient == nil {
			return nil, fmt.Errorf("cluster client is not initialized")
		}
		clusterName = strings.TrimSpace(clusterName)
		if clusterName == "" {
			return nil, fmt.Errorf("cluster is required")
		}
		return kc.GetKubeClient(ctx, kc.RuntimeClient(), kc.Scheme(), clusterName)
	}
}
