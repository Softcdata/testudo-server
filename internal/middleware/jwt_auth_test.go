package middleware

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"github.com/hertz-contrib/jwt"
	"github.com/softcdata/testudo-server/internal/i18n"
	"github.com/softcdata/testudo-server/internal/transport"
	"github.com/softcdata/testudo-server/internal/userstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAuthStore struct {
	user userstore.UserRecord
	err  error
}

func (f *fakeAuthStore) EnsureInitialized(context.Context) error {
	return nil
}

func (f *fakeAuthStore) GetUserByUsername(_ context.Context, username string) (userstore.UserRecord, error) {
	if f.err != nil {
		return userstore.UserRecord{}, f.err
	}
	if f.user.Username != username {
		return userstore.UserRecord{}, userstore.ErrUserNotFound
	}
	return f.user, nil
}

func (f *fakeAuthStore) ListUsers(context.Context) ([]userstore.UserRecord, error) {
	return []userstore.UserRecord{}, nil
}

func (f *fakeAuthStore) CreateUser(context.Context, userstore.CreateUserInput) (userstore.UserRecord, error) {
	return userstore.UserRecord{}, nil
}

func (f *fakeAuthStore) DeleteUser(context.Context, string, string) error {
	return nil
}

func (f *fakeAuthStore) UpdateUserPassword(context.Context, string, string, string) (userstore.UserRecord, error) {
	return userstore.UserRecord{}, nil
}

func (f *fakeAuthStore) UpdateUserStatus(context.Context, string, string, string) (userstore.UserRecord, error) {
	return userstore.UserRecord{}, nil
}

func TestAuthenticateLoginSuccess(t *testing.T) {
	hash, err := userstore.HashPassword("admin123")
	require.NoError(t, err)

	store := &fakeAuthStore{user: userstore.UserRecord{
		ID:           7,
		Username:     "admin",
		Status:       userstore.StatusActive,
		PasswordHash: hash,
	}}

	user, err := authenticateLogin(context.Background(), store, LoginRequest{Username: "admin", Password: "admin123"})
	require.NoError(t, err)
	assert.Equal(t, int64(7), user.ID)
	assert.Equal(t, "admin", user.Username)
}

func TestAuthenticateLoginDisabled(t *testing.T) {
	hash, err := userstore.HashPassword("admin123")
	require.NoError(t, err)

	store := &fakeAuthStore{user: userstore.UserRecord{
		ID:           1,
		Username:     "admin",
		Status:       userstore.StatusDisabled,
		PasswordHash: hash,
	}}

	_, err = authenticateLogin(context.Background(), store, LoginRequest{Username: "admin", Password: "admin123"})
	require.ErrorIs(t, err, ErrUserDisabled)
}

func TestAuthenticateLoginWrongPassword(t *testing.T) {
	hash, err := userstore.HashPassword("admin123")
	require.NoError(t, err)

	store := &fakeAuthStore{user: userstore.UserRecord{
		ID:           1,
		Username:     "admin",
		Status:       userstore.StatusActive,
		PasswordHash: hash,
	}}

	_, err = authenticateLogin(context.Background(), store, LoginRequest{Username: "admin", Password: "wrong"})
	require.Error(t, err)
	assert.Equal(t, jwt.ErrFailedAuthentication, err)
}

func TestMapUnauthorizedStatusFailedAuthentication(t *testing.T) {
	status := mapUnauthorizedStatus(consts.StatusUnauthorized, jwt.ErrFailedAuthentication.Error())
	assert.Equal(t, consts.StatusBadRequest, status)
}

func TestMapUnauthorizedStatusDisabledUser(t *testing.T) {
	status := mapUnauthorizedStatus(consts.StatusUnauthorized, ErrUserDisabled.Error())
	assert.Equal(t, consts.StatusForbidden, status)
}

func TestMapUnauthorizedStatusDefault(t *testing.T) {
	status := mapUnauthorizedStatus(consts.StatusUnauthorized, "token is invalid")
	assert.Equal(t, consts.StatusUnauthorized, status)
}

func TestAuthMessageKey(t *testing.T) {
	assert.Equal(t, i18n.KeyAuthUserDisabled, authMessageKey(ErrUserDisabled.Error()))
	assert.Equal(t, i18n.KeyAuthInvalidRequest, authMessageKey(jwt.ErrMissingLoginValues.Error()))
	assert.Equal(t, i18n.KeyAuthFailed, authMessageKey(jwt.ErrFailedAuthentication.Error()))
	assert.Equal(t, i18n.KeyAuthInvalidToken, authMessageKey("token is invalid"))
}

func TestAuthBizCode(t *testing.T) {
	assert.Equal(t, transport.CodeBadRequest, authBizCode(consts.StatusBadRequest))
	assert.Equal(t, transport.CodeForbidden, authBizCode(consts.StatusForbidden))
	assert.Equal(t, transport.CodeUnauthorized, authBizCode(consts.StatusUnauthorized))
}

func TestRefreshTokenHandlerMissingTokenUsesEnvelope(t *testing.T) {
	ctx := app.NewContext(16)
	ctx.Request.SetBody([]byte(`{"refreshToken":""}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	ctx.Set(i18n.ContextLocaleKey, i18n.LocaleEnUS)

	RefreshTokenHandler(context.Background(), ctx)

	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
	body := string(ctx.Response.Body())
	assert.Contains(t, body, `"message":"refresh token required"`)
	assert.Contains(t, body, `"message_key":"auth.refresh_token_required"`)
	assert.False(t, strings.Contains(body, `"msg"`))
}

func TestBuildLoginResponseDataIncludesUserID(t *testing.T) {
	expire := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	data := buildLoginResponseData("access-token", "refresh-token", expire, &UserInfo{
		ID:       42,
		Username: "admin",
	})

	assert.Equal(t, int64(42), data["userid"])
	assert.Equal(t, "admin", data["username"])
	assert.Equal(t, "access-token", data["accessToken"])
	assert.Equal(t, "refresh-token", data["refreshToken"])
	assert.Equal(t, expire.Format(time.RFC3339), data["expire"])
}

func TestBuildLoginResponseDataWithoutUserID(t *testing.T) {
	expire := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	data := buildLoginResponseData("access-token", "refresh-token", expire, nil)

	_, exists := data["userid"]
	assert.False(t, exists)
	_, exists = data["username"]
	assert.False(t, exists)
	assert.Equal(t, "access-token", data["accessToken"])
	assert.Equal(t, "refresh-token", data["refreshToken"])
	assert.Equal(t, expire.Format(time.RFC3339), data["expire"])
}
