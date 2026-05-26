package middleware

import (
	"context"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
)

// WebSocketTokenAdapter handles token extraction from Sec-WebSocket-Protocol
// and Query parameters, normalizing them to Authorization header for JWT middleware.
// It also handles Base64 decoding if the token in Protocol is encoded.
func WebSocketTokenAdapter() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		// 1. Try Query param "token"
		token := c.Query("token")
		if token != "" {
			c.Request.Header.Set("Authorization", "Bearer "+token)
			c.Next(ctx)
			return
		}

		// 2. Try Sec-WebSocket-Protocol
		protocolHeader := string(c.GetHeader("Sec-WebSocket-Protocol"))
		if protocolHeader != "" {
			protocols := strings.Split(protocolHeader, ",")
			for _, p := range protocols {
				p = strings.TrimSpace(p)

				// Directly use the token from protocol header
				// The client is expected to send the raw JWT token in the Sec-WebSocket-Protocol header
				// If the token contains special characters that are not allowed in the protocol header,
				// the client should use the query parameter "token" instead.
				c.Request.Header.Set("Authorization", "Bearer "+p)
				// Important: For WebSocket handshake to succeed using subprotocol negotiation,
				// the server SHOULD echo back the selected protocol in response header.
				// Although 'jwt' isn't a standard protocol, if client sends it as one, it expects it back.
				c.Header("Sec-WebSocket-Protocol", p)
				c.Next(ctx)
				return
			}
		}

		c.Next(ctx)
	}
}
