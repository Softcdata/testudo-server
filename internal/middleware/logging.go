package middleware

import (
	"context"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	hlog "github.com/cloudwego/hertz/pkg/common/hlog"
)

func AccessLog() app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		start := time.Now()
		ctx.Next(c)
		latency := time.Since(start)
		hlog.Infof("[%s] %s %s ip=%s status=%d latency=%s", time.Now().Format(time.DateTime), ctx.Method(), string(ctx.Request.URI().Path()), ctx.ClientIP(), ctx.Response.StatusCode(), latency)
	}

}
