package businessdefaultconfig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/softcdata/testudo-server/internal/i18n"
	"github.com/softcdata/testudo-server/internal/kube"
	"github.com/softcdata/testudo-server/internal/transport"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

type Handler struct {
	*kube.KubeClient
	Rg *route.RouterGroup
	Mw []app.HandlerFunc

	store configMapStore
}

func NewHandler(kc *kube.KubeClient, rg *route.RouterGroup, mw ...app.HandlerFunc) *Handler {
	var store configMapStore
	if kc != nil && kc.K8sClient != nil {
		store = &kubeConfigMapStore{client: kc.K8sClient}
	}
	return &Handler{
		KubeClient: kc,
		Rg:         rg,
		Mw:         mw,
		store:      store,
	}
}

func (h *Handler) getSnapshot(c context.Context, ctx *app.RequestContext) {
	doc, err := h.getConfigDocument(c)
	if err != nil {
		writeBusinessConfigError(ctx, err)
		return
	}
	transport.WriteSuccess(ctx, consts.StatusOK, groupedSnapshot(doc), nil)
}

func (h *Handler) listFields(c context.Context, ctx *app.RequestContext) {
	qParams := transport.ParseOptions(c, ctx)
	doc, err := h.getConfigDocument(c)
	if err != nil {
		writeBusinessConfigError(ctx, err)
		return
	}

	fields := fieldDTOs(doc)
	filtered, err := filterFields(fields, qParams, string(ctx.Query("q")))
	if err != nil {
		writeBusinessConfigError(ctx, err)
		return
	}

	if qParams.Sort == "" {
		sortFieldsByKey(filtered)
	} else {
		filtered = transport.Sort(filtered, qParams, compareFieldDTO)
	}
	pagedItems, total := transport.Paginate(filtered, qParams)

	requestURL := requestPathWithQuery(ctx)
	data, meta := transport.BuildCollectionResponse(
		requestURL,
		"businessDefaultConfigField",
		pagedItems,
		qParams,
		total,
		nil,
		func(item FieldDTO) map[string]string {
			return map[string]string{
				item.Key: fmt.Sprintf("%s#%s", strings.TrimRight(string(ctx.URI().Path()), "/"), item.Key),
			}
		},
	)

	transport.WriteSuccess(ctx, consts.StatusOK, data, meta)
}

func (h *Handler) listFrontendSpecFields(c context.Context, ctx *app.RequestContext) {
	qParams := transport.ParseOptions(c, ctx)
	fields := frontendSpecFieldDTOs()
	filtered, err := filterFrontendSpecFields(fields, qParams, string(ctx.Query("q")))
	if err != nil {
		writeBusinessConfigError(ctx, err)
		return
	}

	if qParams.Sort == "" {
		sortFrontendSpecFieldsByKey(filtered)
	} else {
		filtered = transport.Sort(filtered, qParams, compareFrontendSpecFieldDTO)
	}
	pagedItems, total := transport.Paginate(filtered, qParams)
	qParams.Filters = frontendSpecSupportedFilters(qParams.Filters)

	requestURL := requestPathWithQuery(ctx)
	collection, meta := transport.BuildCollectionResponse(
		requestURL,
		"businessDefaultConfigFrontendField",
		pagedItems,
		qParams,
		total,
		nil,
		func(item FrontendSpecFieldDTO) map[string]string {
			return map[string]string{
				item.Key: fmt.Sprintf("%s#%s", strings.TrimRight(string(ctx.URI().Path()), "/"), item.Key),
			}
		},
	)

	data := FrontendSpecFieldCollectionDTO{
		Items:    collection.Items,
		FieldMap: frontendSpecFieldMap(collection.Items),
	}
	transport.WriteSuccess(ctx, consts.StatusOK, data, meta)
}

func (h *Handler) patchConfig(c context.Context, ctx *app.RequestContext) {
	if !requireSystemAdmin(ctx) {
		return
	}

	patches, err := parsePatchRequest(ctx.Request.Body())
	if err != nil {
		writeBusinessConfigError(ctx, err)
		return
	}

	doc, err := h.mutateConfigDocument(c, actorFromContext(ctx), func(doc *configDocument) error {
		nextValues := effectiveValues(doc)
		normalizedPatches := make(map[string]interface{}, len(patches))

		for key, rawValue := range patches {
			field, exists := fieldDefinitionByKey(key)
			if !exists {
				return &requestValidationError{
					meta: FieldErrorMeta{
						Field:       key,
						Reason:      "unknown business default config field",
						ActualValue: rawValue,
					},
				}
			}
			if !field.Editable {
				return &requestValidationError{
					meta: FieldErrorMeta{
						Field:       key,
						Reason:      "field is read-only",
						Expected:    string(field.DataType),
						ActualValue: rawValue,
					},
				}
			}
			normalized, err := validateAndNormalizeValue(field, rawValue)
			if err != nil {
				return err
			}
			normalizedPatches[key] = normalized
			nextValues[key] = normalized
		}

		if err := validateCrossFieldValues(nextValues); err != nil {
			return err
		}
		if doc.Values == nil {
			doc.Values = make(map[string]interface{})
		}
		for key, value := range normalizedPatches {
			doc.Values[key] = value
		}
		return nil
	})
	if err != nil {
		writeBusinessConfigError(ctx, err)
		return
	}

	transport.WriteSuccess(ctx, consts.StatusOK, groupedSnapshot(doc), nil)
}

func parsePatchRequest(body []byte) (map[string]interface{}, error) {
	var raw map[string]interface{}
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil, &requestValidationError{meta: FieldErrorMeta{Reason: "request body is required"}}
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, &requestValidationError{meta: FieldErrorMeta{Reason: "request body must be a JSON object", ActualValue: string(body)}}
	}
	if len(raw) == 0 {
		return nil, &requestValidationError{meta: FieldErrorMeta{Reason: "at least one field is required"}}
	}

	patches := make(map[string]interface{})
	for key, value := range raw {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, &requestValidationError{meta: FieldErrorMeta{Reason: "field key is required"}}
		}
		if strings.Contains(key, ".") {
			patches[key] = value
			continue
		}

		if _, exists := groupsByKey[key]; !exists {
			patches[key] = value
			continue
		}

		fields, ok := value.(map[string]interface{})
		if !ok {
			return nil, &requestValidationError{
				meta: FieldErrorMeta{
					Field:       key,
					Reason:      "group value must be a JSON object",
					ActualValue: value,
				},
			}
		}
		if len(fields) == 0 {
			return nil, &requestValidationError{
				meta: FieldErrorMeta{
					Field:  key,
					Reason: "group object must contain at least one field",
				},
			}
		}
		for fieldName, fieldValue := range fields {
			fieldName = strings.TrimSpace(fieldName)
			if fieldName == "" {
				return nil, &requestValidationError{
					meta: FieldErrorMeta{
						Field:  key,
						Reason: "field key is required",
					},
				}
			}
			patches[key+"."+fieldName] = fieldValue
		}
	}
	if len(patches) == 0 {
		return nil, &requestValidationError{meta: FieldErrorMeta{Reason: "at least one field is required"}}
	}
	return patches, nil
}

func filterFields(fields []FieldDTO, q *transport.Options, qAlias string) ([]FieldDTO, error) {
	keyword := strings.ToLower(strings.TrimSpace(q.Keyword))
	if keyword == "" {
		keyword = strings.ToLower(strings.TrimSpace(qAlias))
	}
	groupKey := strings.TrimSpace(q.Filters["groupKey"])
	effectMode := strings.TrimSpace(q.Filters["effectMode"])

	var editableFilter *bool
	if rawEditable := strings.TrimSpace(q.Filters["editable"]); rawEditable != "" {
		value, err := parseBoolFilter(rawEditable)
		if err != nil {
			return nil, err
		}
		editableFilter = &value
	}

	out := make([]FieldDTO, 0, len(fields))
	for _, field := range fields {
		if groupKey != "" && field.GroupKey != groupKey {
			continue
		}
		if effectMode != "" && string(field.EffectMode) != effectMode {
			continue
		}
		if editableFilter != nil && field.Editable != *editableFilter {
			continue
		}
		if keyword != "" && !fieldMatchesKeyword(field, keyword) {
			continue
		}
		out = append(out, field)
	}
	return out, nil
}

func parseBoolFilter(raw string) (bool, error) {
	return parseBoolFilterForField("editable", raw)
}

func parseBoolFilterForField(field, raw string) (bool, error) {
	switch strings.ToLower(raw) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, &requestValidationError{
			meta: FieldErrorMeta{
				Field:       field,
				Reason:      field + " filter must be true or false",
				Expected:    "bool",
				ActualValue: raw,
			},
		}
	}
}

func filterFrontendSpecFields(fields []FrontendSpecFieldDTO, q *transport.Options, qAlias string) ([]FrontendSpecFieldDTO, error) {
	keyword := strings.ToLower(strings.TrimSpace(q.Keyword))
	if keyword == "" {
		keyword = strings.ToLower(strings.TrimSpace(qAlias))
	}

	out := make([]FrontendSpecFieldDTO, 0, len(fields))
	for _, field := range fields {
		if keyword != "" && !frontendSpecFieldMatchesKeyword(field, keyword) {
			continue
		}
		out = append(out, field)
	}
	return out, nil
}

func frontendSpecSupportedFilters(filters map[string]string) map[string]string {
	out := make(map[string]string)
	if strings.TrimSpace(filters["q"]) != "" {
		out["q"] = filters["q"]
	}
	return out
}

func fieldMatchesKeyword(field FieldDTO, keyword string) bool {
	candidates := []string{
		field.Key,
		field.Name,
		field.Description,
		fmt.Sprint(field.Value),
		field.GroupKey,
		field.GroupName,
	}
	for _, candidate := range candidates {
		if strings.Contains(strings.ToLower(candidate), keyword) {
			return true
		}
	}
	return false
}

func requestPathWithQuery(ctx *app.RequestContext) string {
	path := string(ctx.URI().Path())
	query := string(ctx.URI().QueryString())
	if strings.TrimSpace(query) == "" {
		return path
	}
	return path + "?" + query
}

func actorFromContext(ctx *app.RequestContext) string {
	if ctx == nil {
		return "system"
	}
	if value, ok := ctx.Get("userName"); ok {
		if userName, ok := value.(string); ok {
			return actorFromContextValue(userName)
		}
	}
	return "system"
}

func requireSystemAdmin(ctx *app.RequestContext) bool {
	if ctx == nil {
		return false
	}
	value, ok := ctx.Get("userName")
	if !ok {
		return true
	}
	userName, _ := value.(string)
	userName = strings.TrimSpace(userName)
	if userName == "" || strings.EqualFold(userName, "admin") {
		return true
	}
	transport.WriteErrorKey(ctx, transport.CodeForbidden, i18n.KeyOnlyAdminCanModify, nil, nil)
	return false
}

func writeBusinessConfigError(ctx *app.RequestContext, err error) {
	var validationErr *requestValidationError
	switch {
	case errors.As(err, &validationErr):
		transport.WriteError(ctx, transport.CodeBadRequest, validationErr.Error(), validationErr.Meta())
	case errors.Is(err, errConfigTooLarge):
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
	case k8serrors.IsConflict(err):
		transport.WriteError(ctx, transport.CodeConflict, err.Error(), nil)
	default:
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
	}
}
