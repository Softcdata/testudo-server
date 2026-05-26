package cluster

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-operator/pkg/tools"
)

type clusterEndpointConflictError struct {
	message string
	meta    ClusterEndpointConflictMeta
}

func (e *clusterEndpointConflictError) Error() string {
	if e == nil {
		return ""
	}
	return e.message
}

func newClusterEndpointConflictError(clusterName, endpoint string) error {
	return &clusterEndpointConflictError{
		message: fmt.Sprintf("cluster endpoint %s already exists in cluster %s", endpoint, clusterName),
		meta: ClusterEndpointConflictMeta{
			ConflictType:     "clusterEndpoint",
			ConflictCluster:  clusterName,
			ConflictEndpoint: endpoint,
		},
	}
}

func resolveCreateClusterEffectiveEndpoint(req CreateDisasterClusterRequest) (string, error) {
	if endpoint := strings.TrimSpace(req.Endpoint); endpoint != "" {
		return normalizeClusterEndpoint(endpoint)
	}
	if len(req.KubeConfig) == 0 {
		return "", nil
	}

	restConfig, err := tools.GetRestConfig(req.KubeConfig)
	if err != nil {
		return "", err
	}
	return normalizeClusterEndpoint(restConfig.Host)
}

func findClusterEndpointConflict(items []*dapisv1.Cluster, clusterName, endpoint string) error {
	clusterName = strings.TrimSpace(clusterName)
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil
	}

	for _, item := range items {
		if item == nil || item.Name == clusterName {
			continue
		}
		existingEndpoint, ok := resolveClusterEffectiveEndpoint(item)
		if !ok {
			continue
		}
		if existingEndpoint == endpoint {
			return newClusterEndpointConflictError(item.Name, endpoint)
		}
	}
	return nil
}

func resolveClusterEffectiveEndpoint(item *dapisv1.Cluster) (string, bool) {
	if item == nil {
		return "", false
	}

	if endpoint, err := normalizeClusterEndpoint(item.Spec.Endpoint); err == nil && endpoint != "" {
		return endpoint, true
	}
	if endpoint, err := normalizeClusterEndpoint(item.Status.Endpoint); err == nil && endpoint != "" {
		return endpoint, true
	}
	if len(item.Spec.KubeConfig) == 0 {
		return "", false
	}

	restConfig, err := tools.GetRestConfig(item.Spec.KubeConfig)
	if err != nil {
		return "", false
	}
	endpoint, err := normalizeClusterEndpoint(restConfig.Host)
	if err != nil || endpoint == "" {
		return "", false
	}
	return endpoint, true
}

func normalizeClusterEndpoint(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid endpoint %q: %w", raw, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid endpoint %q", raw)
	}

	parsed.Scheme = strings.ToLower(parsed.Scheme)
	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return "", fmt.Errorf("invalid endpoint %q", raw)
	}

	port := parsed.Port()
	if (parsed.Scheme == "https" && port == "443") || (parsed.Scheme == "http" && port == "80") {
		port = ""
	}

	if port != "" {
		parsed.Host = net.JoinHostPort(host, port)
	} else {
		parsed.Host = host
	}

	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.Path = strings.TrimRight(parsed.Path, "/")

	return parsed.String(), nil
}
