package transport

import (
	"errors"
	"runtime/debug"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/softcdata/testudo-server/internal/i18n"
)

// Business codes
const (
	CodeOK                  = 0
	CodeBadRequest          = 1000
	CodeUnauthorized        = 2001
	CodeForbidden           = 2003
	CodeNotFound            = 3004
	CodeConflict            = 3009
	CodeUpstreamError       = 4000
	CodeInternalServerError = 5000
)

// Envelope is the unified response wrapper.
type Envelope struct {
	Code       int         `json:"code"`
	Message    string      `json:"message,omitempty"`
	MessageKey string      `json:"message_key,omitempty"`
	Data       interface{} `json:"data,omitempty"`
	Meta       interface{} `json:"meta,omitempty"`
	TraceID    string      `json:"trace_id,omitempty"`
}

func Success(ctx *app.RequestContext, data interface{}, meta interface{}) *Envelope {
	return &Envelope{
		Code:    CodeOK,
		Message: "OK",
		Data:    data,
		Meta:    meta,
		TraceID: getTraceID(ctx),
	}
}

func Error(ctx *app.RequestContext, bizCode int, message string, meta interface{}) *Envelope {
	return &Envelope{
		Code:    bizCode,
		Message: message,
		Data:    nil,
		Meta:    meta,
		TraceID: getTraceID(ctx),
	}
}

func ErrorKey(ctx *app.RequestContext, bizCode int, key string, args map[string]any, meta interface{}) *Envelope {
	return &Envelope{
		Code:       bizCode,
		Message:    i18n.T(getLocale(ctx), key, args),
		MessageKey: key,
		Data:       nil,
		Meta:       meta,
		TraceID:    getTraceID(ctx),
	}
}

func MapHTTPStatus(bizCode int) int {
	switch bizCode {
	case CodeOK:
		return 200
	case CodeBadRequest:
		return 400
	case CodeUnauthorized:
		return 401
	case CodeForbidden:
		return 403
	case CodeNotFound:
		return 404
	case CodeConflict:
		return 409
	case CodeUpstreamError:
		return 502
	case CodeInternalServerError:
		return 500
	default:
		return 500
	}
}

func getTraceID(ctx *app.RequestContext) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Get("trace_id"); ok {
		if s, ok2 := v.(string); ok2 {
			return s
		}
	}
	return ""
}

func getLocale(ctx *app.RequestContext) string {
	if ctx == nil {
		return i18n.DefaultLocale
	}
	if v, ok := ctx.Get(i18n.ContextLocaleKey); ok {
		if s, ok2 := v.(string); ok2 {
			if locale, ok3 := i18n.NormalizeLocale(s); ok3 {
				return locale
			}
		}
	}
	return i18n.DefaultLocale
}

// Write helpers
func Write(ctx *app.RequestContext, httpStatus int, env *Envelope) {
	if ctx == nil || env == nil {
		return
	}
	setLanguageHeaders(ctx)
	ctx.JSON(httpStatus, env)
}

func WriteSuccess(ctx *app.RequestContext, httpStatus int, data interface{}, meta interface{}) {
	env := Success(ctx, data, meta)
	Write(ctx, httpStatus, env)
}

func WriteError(ctx *app.RequestContext, bizCode int, message string, meta interface{}) {
	if bizCode == CodeInternalServerError {
		hlog.Errorf("Internal Server Error: %s\nStack:\n%s", message, debug.Stack())
	}
	env := Error(ctx, bizCode, message, meta)
	Write(ctx, MapHTTPStatus(env.Code), env)
}

func WriteErrorKey(ctx *app.RequestContext, bizCode int, key string, args map[string]any, meta interface{}) {
	if bizCode == CodeInternalServerError {
		hlog.Errorf("Internal Server Error: %s\nStack:\n%s", i18n.T(i18n.DefaultLocale, key, args), debug.Stack())
	}
	env := ErrorKey(ctx, bizCode, key, args, meta)
	Write(ctx, MapHTTPStatus(env.Code), env)
}

func WriteErrorFrom(ctx *app.RequestContext, bizCode int, err error, meta interface{}) {
	if err == nil {
		WriteErrorKey(ctx, bizCode, i18n.KeyCommonInternalError, nil, meta)
		return
	}
	var localized *i18n.LocalizedError
	if errors.As(err, &localized) {
		WriteErrorKey(ctx, bizCode, localized.Key, localized.Args, meta)
		return
	}
	WriteError(ctx, bizCode, err.Error(), meta)
}

func setLanguageHeaders(ctx *app.RequestContext) {
	locale := getLocale(ctx)
	ctx.Response.Header.Set(i18n.HeaderContentLocale, locale)
	appendResponseVary(ctx, i18n.HeaderLanguage, i18n.HeaderAcceptLanguage)
}

func appendResponseVary(ctx *app.RequestContext, values ...string) {
	seen := map[string]struct{}{}
	ordered := make([]string, 0, len(values))
	current := string(ctx.Response.Header.Peek(i18n.HeaderVary))
	for _, value := range strings.Split(current, ",") {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		lower := strings.ToLower(value)
		if _, ok := seen[lower]; ok {
			continue
		}
		seen[lower] = struct{}{}
		ordered = append(ordered, value)
	}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		lower := strings.ToLower(value)
		if _, ok := seen[lower]; ok {
			continue
		}
		seen[lower] = struct{}{}
		ordered = append(ordered, value)
	}
	ctx.Response.Header.Set(i18n.HeaderVary, strings.Join(ordered, ", "))
}
