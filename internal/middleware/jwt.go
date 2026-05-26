package middleware

import (
	"context"
	"errors"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	hlog "github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	golang_jwt "github.com/golang-jwt/jwt/v4"
	"github.com/hertz-contrib/jwt"
	"github.com/softcdata/testudo-server/configs"
	"github.com/softcdata/testudo-server/internal/i18n"
	"github.com/softcdata/testudo-server/internal/transport"
	"github.com/softcdata/testudo-server/internal/userstore"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type UserInfo struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

var (
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrUserDisabled        = errors.New("user is disabled")
)

func NewJWT(store userstore.Store) *jwt.HertzJWTMiddleware {

	secret := configs.Cfg.JWT.Secret
	accessExpire := configs.Cfg.JWT.AccessExpire
	if accessExpire == 0 {
		accessExpire = time.Hour * 24
	}

	jwtMiddleware, err := jwt.New(&jwt.HertzJWTMiddleware{
		Realm:       "disaster",
		Key:         []byte(secret),
		Timeout:     accessExpire,
		MaxRefresh:  time.Hour, // Not used for our custom refresh
		IdentityKey: "id",
		// Support token extraction from:
		// 1. Authorization Header (Bearer schema)
		// 2. Sec-WebSocket-Protocol Header (for WS, if token is the only protocol)
		// 3. Query param "token"
		TokenLookup: "header:Authorization,header:Sec-WebSocket-Protocol,query:token",

		// 登录时验证逻辑
		Authenticator: func(ctx context.Context, c *app.RequestContext) (interface{}, error) {
			var loginVals LoginRequest
			if err := c.BindAndValidate(&loginVals); err != nil {
				return "", jwt.ErrMissingLoginValues
			}

			user, err := authenticateLogin(ctx, store, loginVals)
			if err != nil {
				return nil, err
			}

			// Save user to context for LoginResponse
			c.Set("login_user", user)
			return user, nil
		},

		// 登录成功后生成 Token
		LoginResponse: func(ctx context.Context, c *app.RequestContext, code int, token string, expire time.Time) {
			user, exists := c.Get("login_user")
			var currentUser *UserInfo
			var refreshToken string
			if exists {
				if u, ok := user.(*UserInfo); ok {
					currentUser = u
					rt, err := GenerateRefreshToken(u)
					if err == nil {
						refreshToken = rt
					} else {
						hlog.Errorf("generate refresh token err: %v", err)
					}
				}
			}

			transport.WriteSuccess(c, consts.StatusOK, buildLoginResponseData(token, refreshToken, expire, currentUser), nil)
		},

		// 提取用户身份
		IdentityHandler: func(ctx context.Context, c *app.RequestContext) interface{} {
			claims := jwt.ExtractClaims(ctx, c)
			id := int64(claims["id"].(float64))
			username := claims["username"].(string)

			// 将用户信息注入 Context，供后续 Handler 使用
			c.Set("userID", id)
			c.Set("userName", username)

			return &UserInfo{
				ID:       id,
				Username: username,
			}
		},

		// 设置 JWT 负载
		PayloadFunc: func(data interface{}) jwt.MapClaims {
			if v, ok := data.(*UserInfo); ok {
				return jwt.MapClaims{
					"id":       v.ID,
					"username": v.Username,
				}
			}
			return jwt.MapClaims{}
		},

		Unauthorized: func(ctx context.Context, c *app.RequestContext, code int, message string) {
			statusCode := mapUnauthorizedStatus(code, message)
			transport.WriteErrorKey(c, authBizCode(statusCode), authMessageKey(message), nil, nil)
		},
	})

	if err != nil {
		hlog.Errorf("new jwt middleware err: %v", err)
		return nil
	}

	return jwtMiddleware
}

func mapUnauthorizedStatus(code int, message string) int {
	if message == ErrUserDisabled.Error() {
		return consts.StatusForbidden
	}
	if message == jwt.ErrFailedAuthentication.Error() {
		return consts.StatusBadRequest
	}
	return code
}

func authBizCode(statusCode int) int {
	switch statusCode {
	case consts.StatusBadRequest:
		return transport.CodeBadRequest
	case consts.StatusForbidden:
		return transport.CodeForbidden
	default:
		return transport.CodeUnauthorized
	}
}

func authMessageKey(message string) string {
	switch message {
	case ErrUserDisabled.Error():
		return i18n.KeyAuthUserDisabled
	case jwt.ErrMissingLoginValues.Error():
		return i18n.KeyAuthInvalidRequest
	case jwt.ErrFailedAuthentication.Error():
		return i18n.KeyAuthFailed
	default:
		return i18n.KeyAuthInvalidToken
	}
}

func authenticateLogin(ctx context.Context, store userstore.Store, loginVals LoginRequest) (*UserInfo, error) {
	if store == nil {
		return nil, jwt.ErrFailedAuthentication
	}

	user, err := store.GetUserByUsername(ctx, loginVals.Username)
	if err != nil {
		return nil, jwt.ErrFailedAuthentication
	}
	if user.Status == userstore.StatusDisabled {
		return nil, ErrUserDisabled
	}
	if err := userstore.VerifyPassword(user.PasswordHash, loginVals.Password); err != nil {
		return nil, jwt.ErrFailedAuthentication
	}

	id := user.ID
	if id <= 0 {
		id = 1
	}

	return &UserInfo{
		ID:       id,
		Username: user.Username,
	}, nil
}

func GenerateRefreshToken(user *UserInfo) (string, error) {
	refreshExpire := configs.Cfg.JWT.RefreshExpire
	if refreshExpire == 0 {
		refreshExpire = time.Hour * 24 * 7
	}

	token := golang_jwt.New(golang_jwt.GetSigningMethod("HS256"))
	claims := token.Claims.(golang_jwt.MapClaims)
	claims["id"] = user.ID
	claims["username"] = user.Username
	claims["exp"] = time.Now().Add(refreshExpire).Unix()
	claims["type"] = "refresh" // Optional: distinguish from access token

	return token.SignedString([]byte(configs.Cfg.JWT.Secret))
}

// RefreshTokenHandler handles manual refresh with body param
func RefreshTokenHandler(ctx context.Context, c *app.RequestContext) {
	var req RefreshRequest
	if err := c.BindAndValidate(&req); err != nil {
		transport.WriteErrorKey(c, transport.CodeBadRequest, i18n.KeyAuthInvalidRequest, nil, nil)
		return
	}

	if req.RefreshToken == "" {
		transport.WriteErrorKey(c, transport.CodeBadRequest, i18n.KeyAuthRefreshRequired, nil, nil)
		return
	}

	// Parse and validate Refresh Token
	token, err := golang_jwt.Parse(req.RefreshToken, func(token *golang_jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*golang_jwt.SigningMethodHMAC); !ok {
			return nil, golang_jwt.ErrSignatureInvalid
		}
		return []byte(configs.Cfg.JWT.Secret), nil
	})

	if err != nil || !token.Valid {
		transport.WriteErrorKey(c, transport.CodeUnauthorized, i18n.KeyAuthInvalidRefreshToken, nil, nil)
		return
	}

	claims, ok := token.Claims.(golang_jwt.MapClaims)
	if !ok {
		transport.WriteErrorKey(c, transport.CodeUnauthorized, i18n.KeyAuthInvalidTokenClaims, nil, nil)
		return
	}

	// Verify type if we added it
	if t, ok := claims["type"]; ok && t != "refresh" {
		transport.WriteErrorKey(c, transport.CodeUnauthorized, i18n.KeyAuthNotRefreshToken, nil, nil)
		return
	}

	// Generate new Access Token
	id := int64(claims["id"].(float64))
	username := claims["username"].(string)

	accessToken, expire, err := GenerateAccessToken(id, username)
	if err != nil {
		transport.WriteErrorKey(c, transport.CodeInternalServerError, i18n.KeyAuthGenerateTokenFailed, nil, nil)
		return
	}

	// We do NOT rotate Refresh Token here (unless required, but proposal didn't specify).
	// Simply return new Access Token.

	transport.WriteSuccess(c, consts.StatusOK, map[string]interface{}{
		"accessToken": accessToken,
		"expire":      expire.Format(time.RFC3339),
	}, nil)
}

func GenerateAccessToken(id int64, username string) (string, time.Time, error) {
	accessExpire := configs.Cfg.JWT.AccessExpire
	if accessExpire == 0 {
		accessExpire = time.Hour * 24
	}
	expire := time.Now().Add(accessExpire)

	token := golang_jwt.New(golang_jwt.GetSigningMethod("HS256"))
	claims := token.Claims.(golang_jwt.MapClaims)
	claims["id"] = id
	claims["username"] = username
	claims["exp"] = expire.Unix()
	// "orig_iat" is used by Hertz Middleware? No, it uses standard claims.

	tokenString, err := token.SignedString([]byte(configs.Cfg.JWT.Secret))
	return tokenString, expire, err
}

func buildLoginResponseData(accessToken, refreshToken string, expire time.Time, user *UserInfo) map[string]interface{} {
	data := map[string]interface{}{
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
		"expire":       expire.Format(time.RFC3339),
	}
	if user != nil {
		data["userid"] = user.ID
		data["username"] = user.Username
	}
	return data
}
