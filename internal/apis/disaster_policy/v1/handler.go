package policy

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
	"github.com/softcdata/testudo-operator/pkg/metadata"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/i18n"
	"github.com/softcdata/testudo-server/internal/kube"
	"github.com/softcdata/testudo-server/internal/transport"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

type PolicyHandler struct {
	*kube.KubeClient
	Rg                   *route.RouterGroup
	Mw                   []app.HandlerFunc
	DisasterPolicyLister listers.DisasterPolicyLister
}

func NewPolicyHandler(kc *kube.KubeClient, rg *route.RouterGroup, mw ...app.HandlerFunc) *PolicyHandler {
	return &PolicyHandler{
		KubeClient:           kc,
		Rg:                   rg,
		Mw:                   mw,
		DisasterPolicyLister: kc.InformerFactory.Disaster().V1().DisasterPolicies().Lister(),
	}
}

func (h *PolicyHandler) policies(c context.Context, ctx *app.RequestContext) {
	// 1. 解析通用查询参数
	qParams := transport.ParseOptions(c, ctx)

	// 2. 构建 Label Selector
	selector := transport.BuildLabelSelector(qParams)

	// 3. 调用 Lister 获取初步数据
	items, err := h.DisasterPolicyLister.List(selector)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	// 3.0 全量内存模糊过滤
	filteredItems := make([]*dapisv1.DisasterPolicy, 0)
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
	sortedItems := transport.Sort(filteredItems, qParams, compareDisasterPolicies)

	// 5. 内存分页逻辑
	pagedItems, total := transport.Paginate(sortedItems, qParams)

	// Convert to DTOs
	dtos := make([]DisasterPolicyDTO, len(pagedItems))
	for i, item := range pagedItems {
		dtos[i] = ConvertToDisasterPolicyDTO(item)
	}

	// 6. 构建标准响应
	requestPath := string(ctx.URI().Path())
	data, meta := transport.BuildCollectionResponse(
		requestPath,
		"disasterPolicy",
		dtos,
		qParams,
		total,
		nil,
		func(item DisasterPolicyDTO) map[string]string {
			return map[string]string{
				item.Name: fmt.Sprintf("%s/%s", strings.TrimRight(requestPath, "/"), item.Name),
			}
		},
	)

	transport.WriteSuccess(ctx, consts.StatusOK, data, meta)
}

func compareDisasterPolicies(a, b *dapisv1.DisasterPolicy, field string) int {
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
		// Informer listers do not guarantee input order. Use name as a deterministic
		// tie-breaker so state-only updates do not make rows jump in the table.
		return strings.Compare(a.Name, b.Name)
	default:
		return strings.Compare(a.Name, b.Name)
	}
}

func (h *PolicyHandler) policyNames(c context.Context, ctx *app.RequestContext) {
	qParams := transport.ParseOptions(c, ctx)
	selector := transport.BuildLabelSelector(qParams)

	items, err := h.DisasterPolicyLister.List(selector)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	// Parse enabled query parameter for state filtering
	enabledParam := string(ctx.Query("enabled"))
	// Remove 'enabled' from Filters to prevent it from being used in Label matching
	delete(qParams.Filters, "enabled")

	// Parse type query parameter for type filtering
	typeParam := string(ctx.Query("type"))
	// Remove 'type' from Filters to prevent it from being used in Label matching
	delete(qParams.Filters, "type")

	// 内存模糊过滤
	matchedItems := make([]*dapisv1.DisasterPolicy, 0)
	for _, item := range items {
		match := true

		// Filter by enabled state if parameter is provided
		if enabledParam != "" {
			if enabledParam == "true" && item.Spec.State != dapisv1.PolicyStateEnabled {
				continue
			}
			if enabledParam == "false" && item.Spec.State != dapisv1.PolicyStateDisabled {
				continue
			}
		}

		// Filter by type if parameter is provided. Server exposes SyncPolicy as
		// the unified sync-policy type while the operator still stores concrete
		// DataSync / ResourceSync values.
		if typeParam != "" && !matchesExternalPolicyTypeFilter(item.Spec.Type, typeParam) {
			continue
		}

		// Exclude policies in Deleting phase (being deleted but blocked by dependencies)
		if item.Status.Phase == dapisv1.PolicyPhaseDeleting {
			continue
		}

		for k, v := range qParams.Filters {
			actual := item.Labels[k]
			if !transport.MatchFuzzy(actual, v) {
				match = false
				break
			}
		}
		if match {
			matchedItems = append(matchedItems, item)
		}
	}
	items = matchedItems

	dtos := make([]DisasterPolicyNameDTO, len(items))
	for i, item := range items {
		dtos[i] = DisasterPolicyNameDTO{
			Name:     item.Name,
			ID:       string(item.UID),
			Type:     externalPolicyType(item.Spec.Type),
			Schedule: item.Spec.Schedule,
			TTL:      item.Spec.TTL,
		}
	}

	transport.WriteSuccess(ctx, consts.StatusOK, dtos, nil)
}

func (h *PolicyHandler) policy(c context.Context, ctx *app.RequestContext) {
	item, err := h.DisasterClient.DisasterV1().DisasterPolicies(common.DisasterSystemNamespace).Get(c, ctx.Param("name"), v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}
	dto := ConvertToDisasterPolicyDTO(item)
	transport.WriteSuccess(ctx, consts.StatusOK, dto, nil)
}

func (h *PolicyHandler) createPolicy(c context.Context, ctx *app.RequestContext) {
	var req CreateDisasterPolicyRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	spec, err := req.ToCRD()
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	policy := dapisv1.DisasterPolicy{
		ObjectMeta: v1.ObjectMeta{
			Name:      req.Name,
			Namespace: common.DisasterSystemNamespace,
		},
		Spec: spec,
	}

	// Inject trace_id and user annotation for operator correlation
	transport.SetTraceAnnotation(&policy.ObjectMeta, ctx, metadata.AnnotationTraceID)
	if user, ok := ctx.Get("userName"); ok {
		if policy.Annotations == nil {
			policy.Annotations = make(map[string]string)
		}
		policy.Annotations["testudo.softcdata.com/user"] = user.(string)
	}

	createdPolicy, err := h.DisasterClient.DisasterV1().DisasterPolicies(common.DisasterSystemNamespace).Create(c, &policy, v1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			transport.WriteError(ctx, transport.CodeConflict, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}
	dto := ConvertToDisasterPolicyDTO(createdPolicy)
	transport.WriteSuccess(ctx, consts.StatusCreated, dto, nil)
}

func (h *PolicyHandler) updatePolicy(c context.Context, ctx *app.RequestContext) {
	var req UpdateDisasterPolicyRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	// Ensure the name in the URL matches the name in the body
	if req.Name != ctx.Param("name") {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationNameURLBodyMismatch, nil, nil)
		return
	}

	var updatedPolicy *dapisv1.DisasterPolicy
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest, err := h.DisasterClient.DisasterV1().DisasterPolicies(common.DisasterSystemNamespace).Get(c, req.Name, v1.GetOptions{})
		if err != nil {
			return err
		}
		if latest.Spec.Type != dapisv1.PolicyTypeAutoBackup {
			// AutoBackup policies are allowed to change while referenced; AppBackup reconciles
			// the updated schedule/ttl/state into the Velero Schedule.
			appbackup, err := h.DisasterClient.DisasterV1().AppBackups(common.DisasterSystemNamespace).List(c, v1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s", metadata.LabelDisasterPolicyUID, string(latest.UID)),
			})
			if err != nil {
				return err
			}
			if len(appbackup.Items) > 0 {
				return fmt.Errorf("cannot update policy %s because it is referenced by existing AppBackups", req.Name)
			}
		}

		// Update Spec
		if err := req.MergeToCRD(&latest.Spec); err != nil {
			return err
		}

		// Update trace-id and user annotation for this write operation
		transport.SetTraceAnnotation(&latest.ObjectMeta, ctx, metadata.AnnotationTraceID)
		if user, ok := ctx.Get("userName"); ok {
			if latest.Annotations == nil {
				latest.Annotations = make(map[string]string)
			}
			latest.Annotations["testudo.softcdata.com/user"] = user.(string)
		}

		updatedPolicy, err = h.DisasterClient.DisasterV1().DisasterPolicies(common.DisasterSystemNamespace).Update(c, latest, v1.UpdateOptions{})
		return err
	})

	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		if isPolicyValidationError(err) {
			transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
			return
		}
		if errors.IsConflict(err) || strings.Contains(err.Error(), "referenced by existing AppBackups") {
			transport.WriteError(ctx, transport.CodeConflict, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}
	dto := ConvertToDisasterPolicyDTO(updatedPolicy)
	transport.WriteSuccess(ctx, consts.StatusOK, dto, nil)
}

func (h *PolicyHandler) deletePolicy(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")
	latest, err := h.DisasterClient.DisasterV1().DisasterPolicies(common.DisasterSystemNamespace).Get(c, name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	// Best-effort annotation before delete
	if latest != nil {
		transport.SetTraceAnnotation(&latest.ObjectMeta, ctx, metadata.AnnotationTraceID)
		if user, ok := ctx.Get("userName"); ok {
			if latest.Annotations == nil {
				latest.Annotations = make(map[string]string)
			}
			latest.Annotations["testudo.softcdata.com/user"] = user.(string)
		}
		_, _ = h.DisasterClient.DisasterV1().DisasterPolicies(common.DisasterSystemNamespace).Update(c, latest, v1.UpdateOptions{})
	}

	err = h.DisasterClient.DisasterV1().DisasterPolicies(common.DisasterSystemNamespace).Delete(c, name, v1.DeleteOptions{})
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
