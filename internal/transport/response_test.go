package transport

import (
	"encoding/json"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/softcdata/testudo-server/internal/i18n"
)

func TestWriteErrorKeyLocalizesEnvelope(t *testing.T) {
	ctx := app.NewContext(16)
	ctx.Set(i18n.ContextLocaleKey, i18n.LocaleEnUS)

	WriteErrorKey(ctx, CodeBadRequest, i18n.KeyValidationNameRequired, nil, nil)

	if got := ctx.Response.StatusCode(); got != 400 {
		t.Fatalf("status=%d, want 400", got)
	}
	if got := string(ctx.Response.Header.Peek(i18n.HeaderContentLocale)); got != i18n.LocaleEnUS {
		t.Fatalf("Content-Language=%q, want %q", got, i18n.LocaleEnUS)
	}

	var env Envelope
	if err := json.Unmarshal(ctx.Response.Body(), &env); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if env.Code != CodeBadRequest {
		t.Fatalf("code=%d, want %d", env.Code, CodeBadRequest)
	}
	if env.Message != "name is required" {
		t.Fatalf("message=%q", env.Message)
	}
	if env.MessageKey != i18n.KeyValidationNameRequired {
		t.Fatalf("message_key=%q", env.MessageKey)
	}
}
