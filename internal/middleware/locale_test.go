package middleware

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/softcdata/testudo-server/internal/i18n"
)

func TestResolveRequestLocale(t *testing.T) {
	ctx := app.NewContext(16)
	ctx.Request.Header.Set(i18n.HeaderAcceptLanguage, "fr-FR, en-US;q=0.8")

	got := resolveRequestLocale(ctx)
	if got != i18n.LocaleEnUS {
		t.Fatalf("resolveRequestLocale()=%q, want %q", got, i18n.LocaleEnUS)
	}
}

func TestResolveRequestLocaleWebSocketLang(t *testing.T) {
	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/watch?lang=en-US")
	ctx.Request.Header.Set("Upgrade", "websocket")
	ctx.Request.Header.Set(i18n.HeaderLanguage, "zh-CN")

	got := resolveRequestLocale(ctx)
	if got != i18n.LocaleEnUS {
		t.Fatalf("resolveRequestLocale()=%q, want %q", got, i18n.LocaleEnUS)
	}
}

func TestLocaleMiddlewareSetsHeaders(t *testing.T) {
	ctx := app.NewContext(16)
	ctx.Request.Header.Set(i18n.HeaderLanguage, "en-US")

	LocaleMiddleware()(context.Background(), ctx)

	if got := string(ctx.Response.Header.Peek(i18n.HeaderContentLocale)); got != i18n.LocaleEnUS {
		t.Fatalf("Content-Language=%q, want %q", got, i18n.LocaleEnUS)
	}
	if got := string(ctx.Response.Header.Peek(i18n.HeaderVary)); got != "X-Language, Accept-Language" {
		t.Fatalf("Vary=%q", got)
	}
	if value, ok := ctx.Get(i18n.ContextLocaleKey); !ok || value != i18n.LocaleEnUS {
		t.Fatalf("locale context=%v, ok=%v", value, ok)
	}
}
