package middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
)

func RequestID() app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		id := uuid.New().String()
		ctx.Response.Header.Set("X-Request-ID", id)
		ctx.Set("request_id", id)
		ctx.Next(c)
	}
}
