package config

import (
	"context"
	stderrors "errors"
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

type ConfigHandler struct {
	*kube.KubeClient
	Rg                   *route.RouterGroup
	Mw                   []app.HandlerFunc
	DisasterConfigLister listers.DisasterConfigLister
}

type badRequestError struct {
	err error
}

func (e *badRequestError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

func NewConfigHandler(kc *kube.KubeClient, rg *route.RouterGroup, mw ...app.HandlerFunc) *ConfigHandler {
	return &ConfigHandler{
		KubeClient:           kc,
		Rg:                   rg,
		Mw:                   mw,
		DisasterConfigLister: kc.InformerFactory.Disaster().V1().DisasterConfigs().Lister(),
	}
}

// Helper to populate policy crons
func (h *ConfigHandler) populatePolicyCrons(c context.Context, dto *DisasterConfigSpecDTO) {
	namespace := common.DisasterSystemNamespace

	dto.DataSyncCron = ""
	if dto.DataSyncPolicy != "" {
		if policy, err := h.DisasterClient.DisasterV1().DisasterPolicies(namespace).Get(c, dto.DataSyncPolicy, matev1.GetOptions{}); err == nil && policy.Spec.State != dapisv1.PolicyStateDisabled {
			dto.DataSyncCron = policy.Spec.Schedule
		}
	}

	dto.ResourceSyncCron = ""
	resourceSyncPolicy := strings.TrimSpace(dto.ResourceSyncPolicy)
	if resourceSyncPolicy == "" {
		resourceSyncPolicy = strings.TrimSpace(dto.ResourcesSyncPolicy)
	}
	if resourceSyncPolicy != "" {
		if policy, err := h.DisasterClient.DisasterV1().DisasterPolicies(namespace).Get(c, resourceSyncPolicy, matev1.GetOptions{}); err == nil && policy.Spec.State != dapisv1.PolicyStateDisabled {
			dto.ResourceSyncCron = policy.Spec.Schedule
		}
	}
}

func (cluster *ConfigHandler) toDisasterConfigDTO(c context.Context, item *dapisv1.DisasterConfig) DisasterConfigDTO {
	dto := ConvertToDisasterConfigDTO(item)
	cluster.populatePolicyCrons(c, &dto.Spec)
	return dto
}

func (cluster *ConfigHandler) configs(c context.Context, ctx *app.RequestContext) {
	// 1. 解析通用查询参数
	qParams := transport.ParseOptions(c, ctx)

	// 2. 构建 Label Selector
	selector := transport.BuildLabelSelector(qParams)

	// 3. 调用 Lister 获取初步后的数据
	items, err := cluster.DisasterConfigLister.List(selector)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	// 3.0 全量内存模糊过滤
	filteredItems := make([]*dapisv1.DisasterConfig, 0)
	keyword := strings.ToLower(strings.TrimSpace(qParams.Keyword))
	for _, item := range items {
		match := true
		if keyword != "" && !strings.Contains(strings.ToLower(item.Name), keyword) {
			match = false
		}
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
	sortedItems := transport.Sort(filteredItems, qParams, func(a, b *dapisv1.DisasterConfig, field string) int {
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

	summary := summarizeDisasterConfigList(sortedItems)

	// 5. 内存分页逻辑
	pagedItems, total := transport.Paginate(sortedItems, qParams)

	// Convert to DTOs
	dtos := make([]DisasterConfigDTO, len(pagedItems))
	for i, item := range pagedItems {
		dtos[i] = cluster.toDisasterConfigDTO(c, item)
	}

	// 6. 构建标准响应
	requestPath := string(ctx.URI().Path())
	data, meta := transport.BuildCollectionResponse(
		requestPath,
		"disasterConfig",
		dtos,
		qParams,
		total,
		nil,
		func(item DisasterConfigDTO) map[string]string {
			return map[string]string{
				item.Name: fmt.Sprintf("%s/%s", strings.TrimRight(requestPath, "/"), item.Name),
			}
		},
	)
	meta.Summary = summary

	transport.WriteSuccess(ctx, consts.StatusOK, data, meta)
}

func summarizeDisasterConfigList(items []*dapisv1.DisasterConfig) map[string]int {
	healthyCount := 0
	abnormalCount := 0
	for _, item := range items {
		switch item.Status.Status {
		case dapisv1.DisasterConfigStatusReady:
			healthyCount++
		case dapisv1.DisasterConfigStatusNotReady, dapisv1.DisasterConfigStatusError:
			abnormalCount++
		}
	}
	return map[string]int{
		"healthyCount":  healthyCount,
		"abnormalCount": abnormalCount,
	}
}

func (cluster *ConfigHandler) configNames(c context.Context, ctx *app.RequestContext) {
	qParams := transport.ParseOptions(c, ctx)
	selector := transport.BuildLabelSelector(qParams)

	items, err := cluster.DisasterConfigLister.List(selector)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	// 内存模糊过滤
	matchedItems := make([]*dapisv1.DisasterConfig, 0)
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
			matchedItems = append(matchedItems, item)
		}
	}
	items = matchedItems

	dtos := make([]DisasterConfigNameDTO, len(items))
	for i, item := range items {
		dtos[i] = DisasterConfigNameDTO{
			Name:          item.Name,
			ID:            string(item.UID),
			SourceCluster: item.Spec.SourceCluster,
			TargetCluster: item.Spec.TargetCluster,
			Status:        item.Status.Status,
		}
	}

	transport.WriteSuccess(ctx, consts.StatusOK, dtos, nil)
}

func (cluster *ConfigHandler) config(c context.Context, ctx *app.RequestContext) {
	item, err := cluster.DisasterClient.DisasterV1().DisasterConfigs().Get(c, ctx.Param("name"), matev1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}
	dto := cluster.toDisasterConfigDTO(c, item)
	transport.WriteSuccess(ctx, consts.StatusOK, dto, nil)
}

func (cluster *ConfigHandler) createConfig(c context.Context, ctx *app.RequestContext) {
	if err := rejectUnsupportedSyncPolicyField(ctx.Request.Body()); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	var req CreateDisasterConfigRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	spec, err := req.ToCRD()
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	imageRewriteReq := req.EffectiveImageRewrite()
	imageRewrite, err := cluster.validateAndBuildImageRewrite(c, req.SourceCluster, req.TargetCluster, imageRewriteReq)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	annotations := make(map[string]string)
	if req.Description != "" {
		annotations["testudo.softcdata.com/description"] = req.Description
	}

	body := dapisv1.DisasterConfig{
		ObjectMeta: matev1.ObjectMeta{
			Name:        req.Name,
			Annotations: annotations,
		},
		Spec: spec,
	}
	body.Spec.ImageRewrite = imageRewrite

	// Inject trace_id annotation for operator correlation
	transport.SetTraceAnnotation(&body.ObjectMeta, ctx, metadata.AnnotationTraceID)

	rc, err := cluster.DisasterClient.DisasterV1().DisasterConfigs().Create(c, &body, matev1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			transport.WriteError(ctx, transport.CodeConflict, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}
	dto := cluster.toDisasterConfigDTO(c, rc)
	transport.WriteSuccess(ctx, consts.StatusCreated, dto, nil)
}

func (cluster *ConfigHandler) deleteConfig(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")

	// Best-effort annotate before delete for correlation
	existing, _ := cluster.DisasterClient.DisasterV1().DisasterConfigs().Get(c, name, matev1.GetOptions{})
	if existing != nil {
		transport.SetTraceAnnotation(&existing.ObjectMeta, ctx, metadata.AnnotationTraceID)
		_, _ = cluster.DisasterClient.DisasterV1().DisasterConfigs().Update(c, existing, matev1.UpdateOptions{})
	}
	err := cluster.DisasterClient.DisasterV1().DisasterConfigs().Delete(c, name, matev1.DeleteOptions{})
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

func (cluster *ConfigHandler) updateConfig(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")
	if name == "" {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationNamePathRequired, nil, nil)
		return
	}

	if err := rejectUnsupportedSyncPolicyField(ctx.Request.Body()); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	var req UpdateDisasterConfigRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	var result *dapisv1.DisasterConfig
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		existing, err := cluster.DisasterClient.DisasterV1().DisasterConfigs().Get(c, name, matev1.GetOptions{})
		if err != nil {
			return err
		}

		effectiveSource := existing.Spec.SourceCluster
		if req.SourceCluster != "" {
			effectiveSource = req.SourceCluster
		}
		effectiveTarget := existing.Spec.TargetCluster
		if req.TargetCluster != "" {
			effectiveTarget = req.TargetCluster
		}
		effectiveImageRewriteReq := req.EffectiveImageRewrite()
		if effectiveImageRewriteReq != nil {
			imageRewrite, err := cluster.validateAndBuildImageRewrite(c, effectiveSource, effectiveTarget, effectiveImageRewriteReq)
			if err != nil {
				return &badRequestError{err: err}
			}
			if imageRewrite == nil {
				existing.Spec.ImageRewrite = nil
			} else {
				existing.Spec.ImageRewrite = imageRewrite.DeepCopy()
			}
		}

		// Update Spec
		if err := req.MergeToCRD(&existing.Spec); err != nil {
			return &badRequestError{err: err}
		}

		// Update Description
		if req.Description != nil {
			if existing.Annotations == nil {
				existing.Annotations = make(map[string]string)
			}
			existing.Annotations["testudo.softcdata.com/description"] = *req.Description
		}

		// Update trace-id annotation for this write operation
		transport.SetTraceAnnotation(&existing.ObjectMeta, ctx, metadata.AnnotationTraceID)

		result, err = cluster.DisasterClient.DisasterV1().DisasterConfigs().Update(c, existing, matev1.UpdateOptions{})
		return err
	})

	if err != nil {
		var bre *badRequestError
		if stderrors.As(err, &bre) {
			transport.WriteError(ctx, transport.CodeBadRequest, bre.Error(), nil)
			return
		}
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
	dto := cluster.toDisasterConfigDTO(c, result)
	transport.WriteSuccess(ctx, consts.StatusOK, dto, nil)
}

func normalizeImageRewriteConfig(req *ImageRewriteConfigRequest) (*dapisv1.ImageRewriteConfig, error) {
	if req == nil {
		return nil, nil
	}
	normalized := &dapisv1.ImageRewriteConfig{
		Enabled: req.Enabled,
	}

	allowedApplyTo := map[string]dapisv1.ImageRewriteApplyTarget{
		string(dapisv1.ImageRewriteApplyResourceSync): dapisv1.ImageRewriteApplyResourceSync,
		string(dapisv1.ImageRewriteApplyDrill):        dapisv1.ImageRewriteApplyDrill,
	}
	applyToSeen := make(map[string]struct{}, len(req.ApplyTo))
	for i := range req.ApplyTo {
		target := strings.TrimSpace(req.ApplyTo[i])
		if target == "" {
			return nil, fmt.Errorf("imageRewrite.applyTo[%d] is required", i)
		}
		applyTarget, ok := allowedApplyTo[target]
		if !ok {
			return nil, fmt.Errorf("imageRewrite.applyTo[%d] must be one of [resourceSync drill]", i)
		}
		if _, exists := applyToSeen[target]; exists {
			return nil, fmt.Errorf("imageRewrite.applyTo[%d] %q is duplicated", i, target)
		}
		applyToSeen[target] = struct{}{}
		normalized.ApplyTo = append(normalized.ApplyTo, applyTarget)
	}

	unmatchedPolicy := strings.TrimSpace(req.UnmatchedPolicy)
	if unmatchedPolicy == "" {
		unmatchedPolicy = string(dapisv1.ImageRewriteUnmatchedPolicyFail)
	}
	switch unmatchedPolicy {
	case string(dapisv1.ImageRewriteUnmatchedPolicyFail), string(dapisv1.ImageRewriteUnmatchedPolicyKeep):
		normalized.UnmatchedPolicy = dapisv1.ImageRewriteUnmatchedPolicy(unmatchedPolicy)
	default:
		return nil, fmt.Errorf("imageRewrite.unmatchedPolicy must be one of [Fail Keep]")
	}

	sourceAliasSeen := make(map[string]struct{}, len(req.Mappings))
	for i := range req.Mappings {
		sourceAlias := strings.TrimSpace(req.Mappings[i].SourceImageSource)
		targetAlias := strings.TrimSpace(req.Mappings[i].TargetImageSource)
		if sourceAlias == "" {
			return nil, fmt.Errorf("imageRewrite.mappings[%d].sourceImageSource is required", i)
		}
		if targetAlias == "" {
			return nil, fmt.Errorf("imageRewrite.mappings[%d].targetImageSource is required", i)
		}
		if _, exists := sourceAliasSeen[sourceAlias]; exists {
			return nil, fmt.Errorf("imageRewrite.mappings[%d].sourceImageSource %q is duplicated", i, sourceAlias)
		}
		sourceAliasSeen[sourceAlias] = struct{}{}
		normalized.Mappings = append(normalized.Mappings, dapisv1.ImageSourceMapping{
			SourceImageSource: sourceAlias,
			TargetImageSource: targetAlias,
		})
	}
	if normalized.Enabled && len(normalized.Mappings) == 0 {
		return nil, fmt.Errorf("imageRewrite.mappings must contain at least one mapping when imageRewrite.enabled is true")
	}
	return normalized, nil
}

func (cluster *ConfigHandler) validateAndBuildImageRewrite(c context.Context, sourceClusterName string, targetClusterName string, req *ImageRewriteConfigRequest) (*dapisv1.ImageRewriteConfig, error) {
	normalized, err := normalizeImageRewriteConfig(req)
	if err != nil {
		return nil, err
	}
	if normalized == nil {
		return nil, nil
	}
	if len(normalized.Mappings) == 0 {
		return normalized, nil
	}

	sourceClusterName = strings.TrimSpace(sourceClusterName)
	targetClusterName = strings.TrimSpace(targetClusterName)
	if sourceClusterName == "" {
		return nil, fmt.Errorf("sourceCluster is required for imageRewrite validation")
	}
	if targetClusterName == "" {
		return nil, fmt.Errorf("targetCluster is required for imageRewrite validation")
	}

	sourceCluster, err := cluster.DisasterClient.DisasterV1().Clusters().Get(c, sourceClusterName, matev1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("source cluster %q not found", sourceClusterName)
		}
		return nil, err
	}
	targetCluster, err := cluster.DisasterClient.DisasterV1().Clusters().Get(c, targetClusterName, matev1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("target cluster %q not found", targetClusterName)
		}
		return nil, err
	}

	sourceAliases := make(map[string]struct{}, len(sourceCluster.Spec.ImageSources))
	for i := range sourceCluster.Spec.ImageSources {
		name := strings.TrimSpace(sourceCluster.Spec.ImageSources[i].Name)
		if name != "" {
			sourceAliases[name] = struct{}{}
		}
	}
	targetAliases := make(map[string]struct{}, len(targetCluster.Spec.ImageSources))
	for i := range targetCluster.Spec.ImageSources {
		name := strings.TrimSpace(targetCluster.Spec.ImageSources[i].Name)
		if name != "" {
			targetAliases[name] = struct{}{}
		}
	}

	for i := range normalized.Mappings {
		mapping := normalized.Mappings[i]
		_, sourceInSource := sourceAliases[mapping.SourceImageSource]
		_, targetInTarget := targetAliases[mapping.TargetImageSource]
		if sourceInSource && targetInTarget {
			continue
		}

		_, sourceInTarget := targetAliases[mapping.SourceImageSource]
		_, targetInSource := sourceAliases[mapping.TargetImageSource]
		if sourceInTarget && targetInSource {
			continue
		}

		return nil, fmt.Errorf(
			"imageRewrite.mappings[%d] alias pair %q -> %q not found across clusters %q and %q",
			i,
			mapping.SourceImageSource,
			mapping.TargetImageSource,
			sourceClusterName,
			targetClusterName,
		)
	}

	return normalized, nil
}

// watchConfigs 监听所有 Config 资源变化
func (cluster *ConfigHandler) watchConfigs(c context.Context, ctx *app.RequestContext) {
	watcherFunc := func(ctx context.Context) (watch.Interface, error) {
		return cluster.DisasterClient.DisasterV1().DisasterConfigs().Watch(ctx, matev1.ListOptions{})
	}
	watchutils.StreamWatch(c, ctx, watcherFunc, func(obj interface{}) interface{} {
		if item, ok := obj.(*dapisv1.DisasterConfig); ok {
			return cluster.toDisasterConfigDTO(c, item)
		}
		return nil
	})
}

// watchConfig 监听指定的 Config 资源变化
func (cluster *ConfigHandler) watchConfig(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")
	if name == "" {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationNameRequired, nil, nil)
		return
	}

	watcherFunc := func(ctx context.Context) (watch.Interface, error) {
		return cluster.DisasterClient.DisasterV1().DisasterConfigs().Watch(ctx, matev1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", name),
		})
	}
	watchutils.StreamWatch(c, ctx, watcherFunc, func(obj interface{}) interface{} {
		if item, ok := obj.(*dapisv1.DisasterConfig); ok {
			return cluster.toDisasterConfigDTO(c, item)
		}
		return nil
	})
}
