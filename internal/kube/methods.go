package kube

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetKubeClient returns a client for the target cluster
func (k *KubeClient) GetKubeClient(ctx context.Context, c client.Client, scheme *runtime.Scheme, clusterName string) (client.Client, error) {
	// 1. Fetch the Cluster CR
	cluster := &dapisv1.Cluster{}
	if err := c.Get(ctx, types.NamespacedName{Name: clusterName}, cluster); err != nil {
		return nil, fmt.Errorf("failed to get Cluster %s: %w", clusterName, err)
	}

	var clientConfig *rest.Config
	var err error

	// 2. Determine authentication method
	if len(cluster.Spec.KubeConfig) > 0 {
		// Use kubeconfig bytes
		clientConfig, err = clientcmd.RESTConfigFromKubeConfig(cluster.Spec.KubeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to parse kubeconfig for Cluster %s: %w", clusterName, err)
		}
	} else if cluster.Spec.Token != "" && cluster.Spec.Endpoint != "" {
		// Use Token + Endpoint
		// Decode token if it's Base64 encoded (JWT starts with "eyJ")
		token := cluster.Spec.Token
		if !strings.HasPrefix(token, "eyJ") {
			if decoded, decErr := base64.StdEncoding.DecodeString(token); decErr == nil {
				token = string(decoded)
			}
		}
		clientConfig = &rest.Config{
			Host:            cluster.Spec.Endpoint,
			BearerToken:     token,
			TLSClientConfig: rest.TLSClientConfig{Insecure: true},
		}
	} else {
		return nil, fmt.Errorf("Cluster %s has no kubeconfig or token/endpoint", clusterName)
	}

	// 3. Create client with the scheme
	cli, err := client.New(clientConfig, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("failed to create client for Cluster %s: %w", clusterName, err)
	}

	return cli, nil
}

func (k *KubeClient) RuntimeClient() client.Client {
	return k.ClusterClient.GetClient()
}

func (k *KubeClient) RuntimeReader() client.Reader {
	if k == nil || k.ClusterClient == nil {
		return nil
	}
	return k.ClusterClient.GetAPIReader()
}

func (k *KubeClient) Scheme() *runtime.Scheme {
	return k.ClusterClient.GetScheme()
}

func (k *KubeClient) LicenseCABundle() []byte {
	if k == nil || k.Config == nil {
		return nil
	}
	if len(k.Config.TLSClientConfig.CAData) > 0 {
		return append([]byte(nil), k.Config.TLSClientConfig.CAData...)
	}
	if strings.TrimSpace(k.Config.TLSClientConfig.CAFile) != "" {
		if bundle, err := os.ReadFile(strings.TrimSpace(k.Config.TLSClientConfig.CAFile)); err == nil && len(bundle) > 0 {
			return bundle
		}
	}
	return nil
}
