package middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
)

// TraceMiddleware generates or propagates X-Trace-Id and stores it in context.
func TraceMiddleware() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		traceID := string(c.Request.Header.Peek("X-Trace-Id"))
		if traceID == "" {
			traceID = uuid.NewString()
		}
		c.Set("trace_id", traceID)
		c.Response.Header.Set("X-Trace-Id", traceID)
		c.Next(ctx)
	}
}
