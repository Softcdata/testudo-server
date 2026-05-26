package middleware

import (
	"context"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/softcdata/testudo-server/internal/i18n"
)

func LocaleMiddleware() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		locale := resolveRequestLocale(c)
		c.Set(i18n.ContextLocaleKey, locale)
		c.Response.Header.Set(i18n.HeaderContentLocale, locale)
		appendVary(c, i18n.HeaderLanguage, i18n.HeaderAcceptLanguage)
		c.Next(ctx)
	}
}

func resolveRequestLocale(c *app.RequestContext) string {
	if c == nil {
		return i18n.DefaultLocale
	}
	languageHeader := string(c.GetHeader(i18n.HeaderLanguage))
	acceptLanguageHeader := string(c.GetHeader(i18n.HeaderAcceptLanguage))
	if isWebSocketRequest(c) {
		return i18n.ResolveWebSocketLocale(c.Query("lang"), languageHeader, acceptLanguageHeader)
	}
	return i18n.ResolveLocale(languageHeader, acceptLanguageHeader)
}

func isWebSocketRequest(c *app.RequestContext) bool {
	if c == nil {
		return false
	}
	return strings.EqualFold(string(c.GetHeader("Upgrade")), "websocket")
}

func appendVary(c *app.RequestContext, values ...string) {
	if c == nil {
		return
	}
	seen := map[string]struct{}{}
	ordered := make([]string, 0, len(values))
	current := string(c.Response.Header.Peek(i18n.HeaderVary))
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
	c.Response.Header.Set(i18n.HeaderVary, strings.Join(ordered, ", "))
}
