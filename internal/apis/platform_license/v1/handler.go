package platformlicenseapi

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	platformlicense "github.com/softcdata/testudo-operator/pkg/license"
	"github.com/softcdata/testudo-server/internal/i18n"
	"github.com/softcdata/testudo-server/internal/kube"
	"github.com/softcdata/testudo-server/internal/transport"
)

type Handler struct {
	*kube.KubeClient
	Rg      *route.RouterGroup
	Mw      []app.HandlerFunc
	Service *Service
}

func NewHandler(kc *kube.KubeClient, rg *route.RouterGroup, namespace, caPath string, mw ...app.HandlerFunc) *Handler {
	var service *Service
	if kc != nil && kc.ClusterClient != nil {
		service = NewService(kc.RuntimeClient(), namespace, caPath)
		service.RuntimeReader = kc.RuntimeReader()
		if bundle := kc.LicenseCABundle(); len(bundle) > 0 {
			service.CABundle = bundle
		}
	}
	return &Handler{
		KubeClient: kc,
		Rg:         rg,
		Mw:         mw,
		Service:    service,
	}
}

func (h *Handler) status(c context.Context, ctx *app.RequestContext) {
	if h.Service == nil {
		transport.WriteErrorKey(ctx, transport.CodeInternalServerError, i18n.KeyLicenseServiceNotReady, nil, nil)
		return
	}
	status, err := h.Service.Status(c)
	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}
	transport.WriteSuccess(ctx, consts.StatusOK, status, nil)
}

func (h *Handler) install(c context.Context, ctx *app.RequestContext) {
	if h.Service == nil {
		transport.WriteErrorKey(ctx, transport.CodeInternalServerError, i18n.KeyLicenseServiceNotReady, nil, nil)
		return
	}

	var req InstallRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	raw := []byte(strings.TrimSpace(req.License))
	if len(raw) == 0 {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyLicenseRequired, nil, nil)
		return
	}
	if !json.Valid(raw) {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyLicenseJSONRequired, nil, map[string]string{"reason": platformlicense.ReasonLicenseInvalid})
		return
	}

	status, err := h.Service.Install(c, raw)
	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}
	transport.WriteSuccess(ctx, consts.StatusOK, status, nil)
}
