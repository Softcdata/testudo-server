package kubernetesresources

import (
	"fmt"

	v1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
)

func (k *KubernetesResourcesHandler) Register() {
	path := fmt.Sprintf("resources.%s", v1.GroupVersion.String())
	g := k.Rg.Group(path).Use(k.Mw...)
	g.GET("/:resource", k.getResources)
}
