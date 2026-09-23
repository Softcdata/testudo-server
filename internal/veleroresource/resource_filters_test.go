package veleroresource

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func testRESTMapper() *meta.DefaultRESTMapper {
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

func TestNormalizeResourceFilters_ConvertsGVKAndPreservesVeleroValues(t *testing.T) {
	got, err := NormalizeResourceFilters("includedResources", []string{
		"apps/v1/Deployment",
		"v1/ConfigMap",
		"deployments.apps",
		"*",
	}, testRESTMapper())

	require.NoError(t, err)
	require.Equal(t, []string{"deployments.apps", "configmaps", "deployments.apps", "*"}, got)
}

func TestNormalizeResourceFilters_ReportsUnresolvableGVK(t *testing.T) {
	_, err := NormalizeResourceFilters("excludedResources", []string{"example.io/v1/Widget"}, testRESTMapper())

	require.Error(t, err)
	require.Contains(t, err.Error(), "excludedResources")
	require.Contains(t, err.Error(), "example.io/v1/Widget")
}

func TestNormalizeResourceFilters_RequiresMapperForGVK(t *testing.T) {
	_, err := NormalizeResourceFilters("includedResources", []string{"apps/v1/Deployment"}, nil)

	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "RESTMapper"))
}

func TestNormalizeResourceFilters_DoesNotRequireMapperForGroupResource(t *testing.T) {
	got, err := NormalizeResourceFilters("includedResources", []string{"deployments.apps", "*"}, nil)

	require.NoError(t, err)
	require.Equal(t, []string{"deployments.apps", "*"}, got)
}
