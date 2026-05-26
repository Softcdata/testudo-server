package instance

import (
	"context"
	"fmt"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/transport"
	watchutils "github.com/softcdata/testudo-server/internal/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	matev1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func (h *InstanceHandler) getOperationDetail(c context.Context, ctx *app.RequestContext) {
	instanceName := ctx.Param("name")
	operationName := ctx.Param("operationName")

	namespace := string(ctx.Query("namespace"))
	if namespace == "" {
		var err error
		namespace, err = h.findNamespace(c, instanceName)
		if err != nil {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
	}

	op, err := h.DisasterClient.DisasterV1().DisasterOperations(namespace).Get(c, operationName, matev1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	if op.Spec.InstanceName != instanceName {
		transport.WriteError(ctx, transport.CodeNotFound, fmt.Sprintf("operation %s not found for instance %s", operationName, instanceName), nil)
		return
	}

	transport.WriteSuccess(ctx, consts.StatusOK, ConvertToOperationDetailDTO(op), nil)
}

func (h *InstanceHandler) watchOperation(c context.Context, ctx *app.RequestContext) {
	operationName := ctx.Param("operationName")

	namespace := string(ctx.Query("namespace"))
	if namespace == "" {
		var err error
		namespace, err = h.findOperationNamespace(c, operationName)
		if err != nil {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
	}

	watcherFunc := func(watchCtx context.Context) (watch.Interface, error) {
		return h.DisasterClient.DisasterV1().DisasterOperations(namespace).Watch(watchCtx, matev1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", operationName),
		})
	}

	converter := func(obj interface{}) interface{} {
		op, ok := obj.(*dapisv1.DisasterOperation)
		if !ok {
			return nil
		}
		return ConvertToOperationDetailDTO(op)
	}

	watchutils.StreamWatch(c, ctx, watcherFunc, converter)
}

func (h *InstanceHandler) findOperationNamespace(c context.Context, name string) (string, error) {
	if name == "" {
		return "", apierrors.NewNotFound(dapisv1.Resource("disasteroperation"), name)
	}

	if _, err := h.DisasterClient.DisasterV1().DisasterOperations(common.DisasterSystemNamespace).Get(c, name, matev1.GetOptions{}); err == nil {
		return common.DisasterSystemNamespace, nil
	}

	list, err := h.DisasterClient.DisasterV1().DisasterOperations("").List(c, matev1.ListOptions{})
	if err != nil {
		return "", err
	}
	for i := range list.Items {
		if list.Items[i].Name == name {
			return list.Items[i].Namespace, nil
		}
	}

	return "", apierrors.NewNotFound(dapisv1.Resource("disasteroperation"), name)
}
