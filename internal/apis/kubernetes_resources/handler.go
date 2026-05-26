package kubernetesresources

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/softcdata/testudo-server/internal/apis/kubernetes_resources/resources"
	"github.com/softcdata/testudo-server/internal/kube"
	"github.com/softcdata/testudo-server/internal/transport"
)

type KubernetesResourcesHandler struct {
	resources map[string]resources.Resources
	Rg        *route.RouterGroup
	Mw        []app.HandlerFunc
}

func NewKubernetesResourcesHandler(kc *kube.KubeClient, rg *route.RouterGroup, mw ...app.HandlerFunc) *KubernetesResourcesHandler {
	return &KubernetesResourcesHandler{
		resources: resources.NewResourcesHandler(kc),
		Rg:        rg,
		Mw:        mw,
	}
}

func (k *KubernetesResourcesHandler) getResources(c context.Context, ctx *app.RequestContext) {
	_ = ctx.Param("resource")
	resource, err := k.resources[ctx.Param("resource")].List(ctx.Query("namespace"))
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	ctx.JSON(consts.StatusOK, resource)
}
