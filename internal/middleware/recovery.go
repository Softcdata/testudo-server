package middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	hlog "github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/softcdata/testudo-server/internal/i18n"
	"github.com/softcdata/testudo-server/internal/transport"
)

func Recovery() app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		defer func() {
			if r := recover(); r != nil {
				hlog.Errorf("panic recovered: %v", r)
				transport.WriteErrorKey(ctx, transport.CodeInternalServerError, i18n.KeyCommonInternalError, nil, nil)
			}
		}()
		ctx.Next(c)
	}
}
