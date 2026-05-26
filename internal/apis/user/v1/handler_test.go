package user

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route/param"
	"github.com/softcdata/testudo-server/internal/userstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeUserStore struct {
	createFn   func(input userstore.CreateUserInput) (userstore.UserRecord, error)
	listFn     func() ([]userstore.UserRecord, error)
	deleteFn   func(username, actor string) error
	patchPwdFn func(username, password, actor string) (userstore.UserRecord, error)
	patchFn    func(username, status, actor string) (userstore.UserRecord, error)
}

func (f *fakeUserStore) EnsureInitialized(context.Context) error {
	return nil
}

func (f *fakeUserStore) GetUserByUsername(context.Context, string) (userstore.UserRecord, error) {
	return userstore.UserRecord{}, userstore.ErrUserNotFound
}

func (f *fakeUserStore) ListUsers(context.Context) ([]userstore.UserRecord, error) {
	if f.listFn == nil {
		return []userstore.UserRecord{}, nil
	}
	return f.listFn()
}

func (f *fakeUserStore) CreateUser(_ context.Context, input userstore.CreateUserInput) (userstore.UserRecord, error) {
	if f.createFn == nil {
		return userstore.UserRecord{}, nil
	}
	return f.createFn(input)
}

func (f *fakeUserStore) DeleteUser(_ context.Context, username, actor string) error {
	if f.deleteFn == nil {
		return nil
	}
	return f.deleteFn(username, actor)
}

func (f *fakeUserStore) UpdateUserPassword(_ context.Context, username, password, actor string) (userstore.UserRecord, error) {
	if f.patchPwdFn == nil {
		return userstore.UserRecord{}, nil
	}
	return f.patchPwdFn(username, password, actor)
}

func (f *fakeUserStore) UpdateUserStatus(_ context.Context, username, status, actor string) (userstore.UserRecord, error) {
	if f.patchFn == nil {
		return userstore.UserRecord{}, nil
	}
	return f.patchFn(username, status, actor)
}

func TestCreateUserSuccess(t *testing.T) {
	h := &UserHandler{
		Store: &fakeUserStore{createFn: func(input userstore.CreateUserInput) (userstore.UserRecord, error) {
			assert.Equal(t, "alice", input.Username)
			assert.Equal(t, "alice@example.com", input.Email)
			assert.Equal(t, "admin", input.Actor)
			return userstore.UserRecord{
				Username:  "alice",
				Email:     "alice@example.com",
				Role:      "admin",
				Status:    "active",
				CreatedAt: "2026-04-01T10:00:00Z",
			}, nil
		}},
	}

	ctx := app.NewContext(16)
	ctx.Set("userName", "admin")
	ctx.Request.SetBody([]byte(`{"username":"alice","email":"alice@example.com","password":"alice123"}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createUser(context.Background(), ctx)

	assert.Equal(t, consts.StatusCreated, ctx.Response.StatusCode())
	var resp struct {
		Code int     `json:"code"`
		Data UserDTO `json:"data"`
	}
	require.NoError(t, json.Unmarshal(ctx.Response.Body(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "alice", resp.Data.Username)
	assert.Equal(t, "admin", resp.Data.Role)
}

func TestCreateUserDuplicate(t *testing.T) {
	h := &UserHandler{
		Store: &fakeUserStore{createFn: func(input userstore.CreateUserInput) (userstore.UserRecord, error) {
			return userstore.UserRecord{}, userstore.ErrUserNameExists
		}},
	}

	ctx := app.NewContext(16)
	ctx.Request.SetBody([]byte(`{"username":"alice","email":"alice@example.com","password":"alice123"}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.createUser(context.Background(), ctx)
	assert.Equal(t, consts.StatusConflict, ctx.Response.StatusCode())
}

func TestListUsersSuccess(t *testing.T) {
	h := &UserHandler{
		Store: &fakeUserStore{listFn: func() ([]userstore.UserRecord, error) {
			return []userstore.UserRecord{
				{Username: "admin", Email: "admin@example.com", Role: "admin", Status: "active", CreatedAt: "2026-04-01T10:00:00Z"},
				{Username: "alice", Email: "alice@example.com", Role: "admin", Status: "disabled", CreatedAt: "2026-04-01T11:00:00Z"},
			}, nil
		}},
	}

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/api/v1/users")
	h.listUsers(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Items []UserDTO `json:"items"`
		} `json:"data"`
		Meta struct {
			Type         string            `json:"type"`
			ResourceType string            `json:"resourceType"`
			Links        map[string]string `json:"links"`
			Pagination   struct {
				Limit   int   `json:"limit"`
				Total   int64 `json:"total"`
				Partial bool  `json:"partial"`
			} `json:"pagination"`
		} `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(ctx.Response.Body(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Len(t, resp.Data.Items, 2)
	assert.Equal(t, "admin", resp.Data.Items[0].Username)
	assert.Equal(t, "alice", resp.Data.Items[1].Username)
	assert.Equal(t, "collection", resp.Meta.Type)
	assert.Equal(t, "user", resp.Meta.ResourceType)
	assert.Equal(t, 10, resp.Meta.Pagination.Limit)
	assert.EqualValues(t, 2, resp.Meta.Pagination.Total)
	assert.False(t, resp.Meta.Pagination.Partial)
	assert.Contains(t, resp.Meta.Links["self"], "/api/v1/users")
}

func TestListUsersPaginationSortAndKeyword(t *testing.T) {
	h := &UserHandler{
		Store: &fakeUserStore{listFn: func() ([]userstore.UserRecord, error) {
			return []userstore.UserRecord{
				{Username: "admin", Email: "admin@example.com", Role: "admin", Status: "active", CreatedAt: "2026-04-01T10:00:00Z"},
				{Username: "alice", Email: "alice@example.com", Role: "admin", Status: "disabled", CreatedAt: "2026-04-01T11:00:00Z"},
			}, nil
		}},
	}

	ctx := app.NewContext(16)
	ctx.Request.SetRequestURI("/api/v1/users?page=2&limit=1&sort=username&order=asc&keyword=example.com")
	h.listUsers(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Items []UserDTO `json:"items"`
		} `json:"data"`
		Meta struct {
			Links      map[string]string `json:"links"`
			Pagination struct {
				Limit    int    `json:"limit"`
				Total    int64  `json:"total"`
				Partial  bool   `json:"partial"`
				First    string `json:"first"`
				Previous string `json:"previous"`
				Next     string `json:"next"`
				Last     string `json:"last"`
			} `json:"pagination"`
		} `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(ctx.Response.Body(), &resp))
	assert.Equal(t, 0, resp.Code)
	require.Len(t, resp.Data.Items, 1)
	assert.Equal(t, "alice", resp.Data.Items[0].Username)
	assert.Equal(t, 1, resp.Meta.Pagination.Limit)
	assert.EqualValues(t, 2, resp.Meta.Pagination.Total)
	assert.True(t, resp.Meta.Pagination.Partial)
	assert.Contains(t, resp.Meta.Pagination.First, "page=1")
	assert.Contains(t, resp.Meta.Pagination.Previous, "page=1")
	assert.Empty(t, resp.Meta.Pagination.Next)
	assert.Contains(t, resp.Meta.Pagination.Last, "page=2")
	assert.Contains(t, resp.Meta.Links["self"], "limit=1")
	assert.Contains(t, resp.Meta.Links["self"], "page=2")
}

func TestDeleteUserSuccess(t *testing.T) {
	h := &UserHandler{
		Store: &fakeUserStore{deleteFn: func(username, actor string) error {
			assert.Equal(t, "alice", username)
			assert.Equal(t, "ops-admin", actor)
			return nil
		}},
	}

	ctx := app.NewContext(16)
	ctx.Set("userName", "ops-admin")
	ctx.Params = param.Params{{Key: "username", Value: "alice"}}
	h.deleteUser(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp struct {
		Code int                `json:"code"`
		Data DeleteUserResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(ctx.Response.Body(), &resp))
	assert.True(t, resp.Data.Deleted)
	assert.Equal(t, "alice", resp.Data.Username)
}

func TestDeleteUserBuiltInRejected(t *testing.T) {
	h := &UserHandler{
		Store: &fakeUserStore{deleteFn: func(username, actor string) error {
			return userstore.ErrDeleteBuiltInUser
		}},
	}

	ctx := app.NewContext(16)
	ctx.Params = param.Params{{Key: "username", Value: "admin"}}
	h.deleteUser(context.Background(), ctx)
	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
}

func TestPatchUserPasswordSuccess(t *testing.T) {
	h := &UserHandler{
		Store: &fakeUserStore{patchPwdFn: func(username, password, actor string) (userstore.UserRecord, error) {
			assert.Equal(t, "alice", username)
			assert.Equal(t, "alice789", password)
			assert.Equal(t, "system", actor)
			return userstore.UserRecord{
				Username:  "alice",
				Email:     "alice@example.com",
				Role:      "admin",
				Status:    "active",
				CreatedAt: "2026-04-01T10:00:00Z",
			}, nil
		}},
	}

	ctx := app.NewContext(16)
	ctx.Params = param.Params{{Key: "username", Value: "alice"}}
	ctx.Request.SetBody([]byte(`{"password":"alice789"}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.patchUserPassword(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())
}

func TestPatchUserPasswordBadRequest(t *testing.T) {
	h := &UserHandler{Store: &fakeUserStore{patchPwdFn: func(username, password, actor string) (userstore.UserRecord, error) {
		return userstore.UserRecord{}, errors.New("should not be called")
	}}}

	ctx := app.NewContext(16)
	ctx.Params = param.Params{{Key: "username", Value: "alice"}}
	ctx.Request.SetBody([]byte(`{"password":"123"}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.patchUserPassword(context.Background(), ctx)
	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
}

func TestPatchUserStatusSuccess(t *testing.T) {
	h := &UserHandler{
		Store: &fakeUserStore{patchFn: func(username, status, actor string) (userstore.UserRecord, error) {
			assert.Equal(t, "alice", username)
			assert.Equal(t, "disabled", status)
			assert.Equal(t, "system", actor)
			return userstore.UserRecord{
				Username:  "alice",
				Email:     "alice@example.com",
				Role:      "admin",
				Status:    "disabled",
				CreatedAt: "2026-04-01T10:00:00Z",
			}, nil
		}},
	}

	ctx := app.NewContext(16)
	ctx.Params = param.Params{{Key: "username", Value: "alice"}}
	ctx.Request.SetBody([]byte(`{"status":"disabled"}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.patchUserStatus(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp struct {
		Code int     `json:"code"`
		Data UserDTO `json:"data"`
	}
	require.NoError(t, json.Unmarshal(ctx.Response.Body(), &resp))
	assert.Equal(t, "disabled", resp.Data.Status)
}

func TestPatchUserStatusNotFound(t *testing.T) {
	h := &UserHandler{
		Store: &fakeUserStore{patchFn: func(username, status, actor string) (userstore.UserRecord, error) {
			return userstore.UserRecord{}, userstore.ErrUserNotFound
		}},
	}

	ctx := app.NewContext(16)
	ctx.Params = param.Params{{Key: "username", Value: "alice"}}
	ctx.Request.SetBody([]byte(`{"status":"active"}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.patchUserStatus(context.Background(), ctx)
	assert.Equal(t, consts.StatusNotFound, ctx.Response.StatusCode())
}

func TestPatchUserStatusBadRequest(t *testing.T) {
	h := &UserHandler{Store: &fakeUserStore{patchFn: func(username, status, actor string) (userstore.UserRecord, error) {
		return userstore.UserRecord{}, errors.New("should not be called")
	}}}

	ctx := app.NewContext(16)
	ctx.Params = param.Params{{Key: "username", Value: "alice"}}
	ctx.Request.SetBody([]byte(`{"status":"paused"}`))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.patchUserStatus(context.Background(), ctx)
	assert.Equal(t, consts.StatusBadRequest, ctx.Response.StatusCode())
}
