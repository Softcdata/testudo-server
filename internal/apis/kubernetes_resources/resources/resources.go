package resources

import (
	"github.com/softcdata/testudo-server/internal/apis/kubernetes_resources/resources/namespaces"
	"github.com/softcdata/testudo-server/internal/kube"
	"k8s.io/apimachinery/pkg/runtime"
)

type Resources interface {
	List(namespace string) (runtime.Object, error)
	Get(namespace, name string) (runtime.Object, error)
}

func NewResourcesHandler(kc *kube.KubeClient) map[string]Resources {
	resources := make(map[string]Resources)
	resources["namespaces"] = namespaces.NewNamespaceHandler(kc.ClusterClient.GetClient())
	return resources
}
