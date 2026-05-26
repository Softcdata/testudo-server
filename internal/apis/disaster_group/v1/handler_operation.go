package v1

import (
	"context"
	"fmt"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	instanceapi "github.com/softcdata/testudo-server/internal/apis/disaster_instance/v1"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/transport"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	matev1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (h *GroupHandler) getOperationDetail(c context.Context, ctx *app.RequestContext) {
	groupName := ctx.Param("name")
	operationName := ctx.Param("operationName")

	op, err := h.DisasterClient.DisasterV1().DisasterOperations(common.DisasterSystemNamespace).Get(c, operationName, matev1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	if op.Spec.GroupName != groupName {
		transport.WriteError(ctx, transport.CodeNotFound, fmt.Sprintf("operation %s not found for group %s", operationName, groupName), nil)
		return
	}

	transport.WriteSuccess(ctx, consts.StatusOK, instanceapi.ConvertToOperationDetailDTO(op), nil)
}
