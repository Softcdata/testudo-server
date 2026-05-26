package event

import (
	"context"
	"fmt"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route"

	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/i18n"
	"github.com/softcdata/testudo-server/internal/kube"
	"github.com/softcdata/testudo-server/internal/transport"
	watchutils "github.com/softcdata/testudo-server/internal/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type EventHandler struct {
	*kube.KubeClient
	Rg *route.RouterGroup
	Mw []app.HandlerFunc
}

const (
	taskOriginLabelKey         = "testudo.softcdata.com/task-origin"
	taskOriginDisasterInstance = "disaster-instance"
)

func NewEventHandler(kc *kube.KubeClient, rg *route.RouterGroup, mw ...app.HandlerFunc) *EventHandler {
	return &EventHandler{
		KubeClient: kc,
		Rg:         rg,
		Mw:         mw,
	}
}

// watchEvents 监听全局事件 (默认 disaster-system 命名空间)
func (h *EventHandler) watchEvents(c context.Context, ctx *app.RequestContext) {
	namespace := ctx.Query("namespace")
	if namespace == "" {
		namespace = common.DisasterSystemNamespace
	}
	labelSelector, err := buildTaskEventLabelSelector(ctx.Query("origin"))
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	// 先获取最新的 ResourceVersion，避免推送历史全量事件
	list, err := h.KubeClient.K8sClient.CoreV1().Events(namespace).List(c, metav1.ListOptions{
		LabelSelector: labelSelector,
		Limit:         1,
	})
	if err != nil {
		transport.WriteErrorKey(ctx, transport.CodeInternalServerError, i18n.KeyWatcherInitFailed, nil, err)
		return
	}
	rv := list.ListMeta.ResourceVersion

	watcherFunc := func(ctx context.Context) (watch.Interface, error) {
		return h.KubeClient.K8sClient.CoreV1().Events(namespace).Watch(ctx, metav1.ListOptions{
			LabelSelector:   labelSelector,
			ResourceVersion: rv,
		})
	}

	watchutils.StreamWatch(c, ctx, watcherFunc, ConvertToTaskEventDTO)
}

// watchResourceEvents 监听指定资源的事件
func (h *EventHandler) watchResourceEvents(c context.Context, ctx *app.RequestContext) {
	resourceType := ctx.Param("resource")
	name := ctx.Param("name")
	namespace := ctx.Query("namespace")
	if namespace == "" {
		namespace = common.DisasterSystemNamespace
	}
	labelSelector, err := buildTaskEventLabelSelector(ctx.Query("origin"))
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	if resourceType == "" || name == "" {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationResourceNameRequired, nil, nil)
		return
	}
	expectedKind, err := resolveEventResourceKind(resourceType)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	fieldSelector := fmt.Sprintf("involvedObject.name=%s", name)

	// 先获取最新的 ResourceVersion
	list, err := h.KubeClient.K8sClient.CoreV1().Events(namespace).List(c, metav1.ListOptions{
		FieldSelector: fieldSelector,
		LabelSelector: labelSelector,
		Limit:         1,
	})
	if err != nil {
		transport.WriteErrorKey(ctx, transport.CodeInternalServerError, i18n.KeyWatcherInitFailed, nil, err)
		return
	}
	rv := list.ListMeta.ResourceVersion

	watcherFunc := func(ctx context.Context) (watch.Interface, error) {
		return h.KubeClient.K8sClient.CoreV1().Events(namespace).Watch(ctx, metav1.ListOptions{
			FieldSelector:   fieldSelector,
			LabelSelector:   labelSelector,
			ResourceVersion: rv,
		})
	}

	watchutils.StreamWatch(c, ctx, watcherFunc, ConvertToTaskEventDTO, watchutils.WithFilter(buildWatchKindFilter(expectedKind)))
}

func buildTaskEventLabelSelector(origin string) (string, error) {
	base := "testudo.softcdata.com/task-event=true"
	switch origin {
	case "", "user":
		// `!=` 会匹配 key 不存在的对象，兼容历史未打标事件
		return base + "," + taskOriginLabelKey + "!=" + taskOriginDisasterInstance, nil
	case "instance":
		return base + "," + taskOriginLabelKey + "=" + taskOriginDisasterInstance, nil
	case "all":
		return base, nil
	default:
		return "", fmt.Errorf("invalid origin: %s, must be one of user|instance|all", origin)
	}
}
