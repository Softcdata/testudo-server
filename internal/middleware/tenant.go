package middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/softcdata/testudo-server/internal/i18n"
	"github.com/softcdata/testudo-server/internal/transport"
)

func TenantID() app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		tenantID := string(ctx.GetHeader("X-Tenant-ID"))
		if tenantID == "" {
			//若是ws跳过
			// 尝试从 Query 参数获取 (主要用于 WebSocket)
			tenantID = ctx.Query("tenant_id")
			if tenantID == "" {
				// 若是 WebSocket 握手请求，且没有 tenant_id，暂时允许跳过 (或者根据业务需求强制要求 Query 参数)
				// 检查 Upgrade 头是否存在
				if string(ctx.Request.Header.Get("Upgrade")) != "" {
					// 默认使用 default 租户或允许通过
					ctx.Set("tenant_id", "default")
					ctx.Next(c)
					return
				}

				transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyTenantHeaderRequired, nil, nil)
				ctx.Abort()
				return
			}
		}
		ctx.Set("tenant_id", tenantID)
		ctx.Next(c)
	}
}
