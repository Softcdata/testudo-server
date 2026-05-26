package backup

import (
	"context"
	"fmt"

	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	listers "github.com/softcdata/testudo-operator/pkg/listers/disaster/v1"
	metadata "github.com/softcdata/testudo-operator/pkg/metadata"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/i18n"
	"github.com/softcdata/testudo-server/internal/kube"
	"github.com/softcdata/testudo-server/internal/transport"
	watchutils "github.com/softcdata/testudo-server/internal/utils"

	"k8s.io/apimachinery/pkg/api/errors"
	matev1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/util/retry"
)

type BackupHandler struct {
	*kube.KubeClient
	Rg                   *route.RouterGroup
	Mw                   []app.HandlerFunc
	DisasterBackupLister listers.DisasterBackupLister
}

func NewBackupHandler(kc *kube.KubeClient, rg *route.RouterGroup, mw ...app.HandlerFunc) *BackupHandler {
	return &BackupHandler{
		KubeClient:           kc,
		Rg:                   rg,
		Mw:                   mw,
		DisasterBackupLister: kc.InformerFactory.Disaster().V1().DisasterBackups().Lister(),
	}
}

func (cluster *BackupHandler) backups(c context.Context, ctx *app.RequestContext) {
	// 1. 解析通用查询参数
	qParams := transport.ParseOptions(c, ctx)

	// 2. 构建 Label Selector
	selector := transport.BuildLabelSelector(qParams)

	// 3. 调用 Lister 获取初步后的数据
	items, err := cluster.DisasterBackupLister.DisasterBackups(common.DisasterSystemNamespace).List(selector)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	// 3.0 全量内存模糊过滤
	filteredItems := make([]*dapisv1.DisasterBackup, 0)
	for _, item := range items {
		match := true
		for k, v := range qParams.Filters {
			actual := item.Labels[k]
			if !transport.MatchFuzzy(actual, v) {
				match = false
				break
			}
		}
		if match {
			filteredItems = append(filteredItems, item)
		}
	}

	// 4. 内存排序逻辑
	sortedItems := transport.Sort(filteredItems, qParams, func(a, b *dapisv1.DisasterBackup, field string) int {
		switch field {
		case "name":
			return strings.Compare(a.Name, b.Name)
		case "creationTimestamp":
			if a.CreationTimestamp.Before(&b.CreationTimestamp) {
				return -1
			}
			if a.CreationTimestamp.After(b.CreationTimestamp.Time) {
				return 1
			}
			return 0
		default:
			return 0
		}
	})

	// 5. 内存分页逻辑
	pagedItems, total := transport.Paginate(sortedItems, qParams)

	// Convert to DTOs
	dtos := make([]DisasterBackupDTO, len(pagedItems))
	for i, item := range pagedItems {
		dtos[i] = ConvertToDisasterBackupDTO(item)
	}

	// 6. 构建标准响应
	requestPath := string(ctx.URI().Path())
	data, meta := transport.BuildCollectionResponse(
		requestPath,
		"disasterBackup",
		dtos,
		qParams,
		total,
		nil,
		func(item DisasterBackupDTO) map[string]string {
			return map[string]string{
				item.Name: fmt.Sprintf("%s/%s", strings.TrimRight(requestPath, "/"), item.Name),
			}
		},
	)

	transport.WriteSuccess(ctx, consts.StatusOK, data, meta)
}

func (cluster *BackupHandler) backup(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")
	if name == "" {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationNameRequired, nil, nil)
		return
	}
	item, err := cluster.DisasterClient.DisasterV1().DisasterBackups(common.DisasterSystemNamespace).Get(c, name, matev1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}
	dto := ConvertToDisasterBackupDTO(item)
	transport.WriteSuccess(ctx, consts.StatusOK, dto, nil)
}

func (cluster *BackupHandler) createBackup(c context.Context, ctx *app.RequestContext) {
	var req CreateDisasterBackupRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	body := dapisv1.DisasterBackup{
		ObjectMeta: matev1.ObjectMeta{
			Name:      req.Name,
			Namespace: common.DisasterSystemNamespace,
		},
		Spec: req.ToCRD(),
	}

	// Inject trace_id annotation for operator correlation
	transport.SetTraceAnnotation(&body.ObjectMeta, ctx, metadata.AnnotationTraceID)

	rc, err := cluster.DisasterClient.DisasterV1().DisasterBackups(common.DisasterSystemNamespace).Create(c, &body, matev1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			transport.WriteError(ctx, transport.CodeConflict, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}
	dto := ConvertToDisasterBackupDTO(rc)
	transport.WriteSuccess(ctx, consts.StatusCreated, dto, nil)
}

func (cluster *BackupHandler) updateBackup(c context.Context, ctx *app.RequestContext) {
	var req UpdateDisasterBackupRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	var result *dapisv1.DisasterBackup
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		existing, err := cluster.DisasterClient.DisasterV1().DisasterBackups(common.DisasterSystemNamespace).Get(c, req.Name, matev1.GetOptions{})
		if err != nil {
			return err
		}

		// Update Spec
		req.MergeToCRD(&existing.Spec)

		// Update trace-id annotation for this write operation
		transport.SetTraceAnnotation(&existing.ObjectMeta, ctx, metadata.AnnotationTraceID)

		result, err = cluster.DisasterClient.DisasterV1().DisasterBackups(common.DisasterSystemNamespace).Update(c, existing, matev1.UpdateOptions{})
		return err
	})

	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		if errors.IsConflict(err) {
			transport.WriteError(ctx, transport.CodeConflict, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}
	dto := ConvertToDisasterBackupDTO(result)
	transport.WriteSuccess(ctx, consts.StatusOK, dto, nil)
}

func (cluster *BackupHandler) deleteBackup(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")

	// Best-effort annotate before delete for correlation
	existing, _ := cluster.DisasterClient.DisasterV1().DisasterBackups(common.DisasterSystemNamespace).Get(c, name, matev1.GetOptions{})
	if existing != nil {
		transport.SetTraceAnnotation(&existing.ObjectMeta, ctx, metadata.AnnotationTraceID)
		_, _ = cluster.DisasterClient.DisasterV1().DisasterBackups(common.DisasterSystemNamespace).Update(c, existing, matev1.UpdateOptions{})
	}
	err := cluster.DisasterClient.DisasterV1().DisasterBackups(common.DisasterSystemNamespace).Delete(c, name, matev1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}
	transport.WriteSuccess(ctx, consts.StatusOK, utils.H{"name": name}, nil)
}

// watchBackups 监听所有 Backup 资源变化
func (cluster *BackupHandler) watchBackups(c context.Context, ctx *app.RequestContext) {
	watcherFunc := func(ctx context.Context) (watch.Interface, error) {
		return cluster.DisasterClient.DisasterV1().DisasterBackups(common.DisasterSystemNamespace).Watch(ctx, matev1.ListOptions{})
	}
	watchutils.StreamWatch(c, ctx, watcherFunc, func(obj interface{}) interface{} {
		if item, ok := obj.(*dapisv1.DisasterBackup); ok {
			return ConvertToDisasterBackupDTO(item)
		}
		return nil
	})
}

// watchBackup 监听指定的 Backup 资源变化
func (cluster *BackupHandler) watchBackup(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")
	if name == "" {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationNameRequired, nil, nil)
		return
	}

	watcherFunc := func(ctx context.Context) (watch.Interface, error) {
		return cluster.DisasterClient.DisasterV1().DisasterBackups(common.DisasterSystemNamespace).Watch(ctx, matev1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", name),
		})
	}
	watchutils.StreamWatch(c, ctx, watcherFunc, func(obj interface{}) interface{} {
		if item, ok := obj.(*dapisv1.DisasterBackup); ok {
			return ConvertToDisasterBackupDTO(item)
		}
		return nil
	})
}
