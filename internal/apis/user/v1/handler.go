package user

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/softcdata/testudo-server/internal/i18n"
	"github.com/softcdata/testudo-server/internal/transport"
	"github.com/softcdata/testudo-server/internal/userstore"
)

var usernameRegex = regexp.MustCompile(`^[A-Za-z0-9._-]{3,32}$`)

type UserHandler struct {
	Rg    *route.RouterGroup
	Mw    []app.HandlerFunc
	Store userstore.Store
}

func NewUserHandler(store userstore.Store, rg *route.RouterGroup, mw ...app.HandlerFunc) *UserHandler {
	return &UserHandler{
		Rg:    rg,
		Mw:    mw,
		Store: store,
	}
}

func (h *UserHandler) createUser(c context.Context, ctx *app.RequestContext) {
	if h.Store == nil {
		transport.WriteErrorKey(ctx, transport.CodeInternalServerError, i18n.KeyUserStoreNotReady, nil, nil)
		return
	}

	var req CreateUserRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	input, err := normalizeCreateRequest(req)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	input.Actor = actorFromContext(ctx)

	created, err := h.Store.CreateUser(c, input)
	if err != nil {
		writeUserError(ctx, err)
		return
	}

	transport.WriteSuccess(ctx, consts.StatusCreated, toUserDTO(created), nil)
}

func (h *UserHandler) listUsers(c context.Context, ctx *app.RequestContext) {
	if h.Store == nil {
		transport.WriteErrorKey(ctx, transport.CodeInternalServerError, i18n.KeyUserStoreNotReady, nil, nil)
		return
	}

	qParams := transport.ParseOptions(c, ctx)
	users, err := h.Store.ListUsers(c)
	if err != nil {
		writeUserError(ctx, err)
		return
	}

	result := make([]UserDTO, 0, len(users))
	for _, user := range users {
		result = append(result, toUserDTO(user))
	}

	keyword := strings.ToLower(strings.TrimSpace(qParams.Keyword))
	if keyword != "" {
		filtered := make([]UserDTO, 0, len(result))
		for _, item := range result {
			if strings.Contains(strings.ToLower(item.Username), keyword) ||
				strings.Contains(strings.ToLower(item.Email), keyword) ||
				strings.Contains(strings.ToLower(item.Role), keyword) ||
				strings.Contains(strings.ToLower(item.Status), keyword) {
				filtered = append(filtered, item)
			}
		}
		result = filtered
	}

	sortedItems := transport.Sort(result, qParams, compareUserDTO)
	pagedItems, total := transport.Paginate(sortedItems, qParams)

	requestPath := string(ctx.URI().Path())
	data, meta := transport.BuildCollectionResponse(
		requestPath,
		"user",
		pagedItems,
		qParams,
		total,
		nil,
		func(item UserDTO) map[string]string {
			return map[string]string{
				item.Username: fmt.Sprintf("%s/%s", strings.TrimRight(requestPath, "/"), item.Username),
			}
		},
	)

	transport.WriteSuccess(ctx, consts.StatusOK, data, meta)
}

func (h *UserHandler) deleteUser(c context.Context, ctx *app.RequestContext) {
	if h.Store == nil {
		transport.WriteErrorKey(ctx, transport.CodeInternalServerError, i18n.KeyUserStoreNotReady, nil, nil)
		return
	}

	username := strings.TrimSpace(ctx.Param("username"))
	if err := validateUsername(username); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	if err := h.Store.DeleteUser(c, username, actorFromContext(ctx)); err != nil {
		writeUserError(ctx, err)
		return
	}

	transport.WriteSuccess(ctx, consts.StatusOK, DeleteUserResponse{Username: username, Deleted: true}, nil)
}

func (h *UserHandler) patchUserPassword(c context.Context, ctx *app.RequestContext) {
	if h.Store == nil {
		transport.WriteErrorKey(ctx, transport.CodeInternalServerError, i18n.KeyUserStoreNotReady, nil, nil)
		return
	}

	username := strings.TrimSpace(ctx.Param("username"))
	if err := validateUsername(username); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	var req PatchUserPasswordRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	password := strings.TrimSpace(req.Password)
	if err := validatePassword(password); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	updated, err := h.Store.UpdateUserPassword(c, username, password, actorFromContext(ctx))
	if err != nil {
		writeUserError(ctx, err)
		return
	}

	transport.WriteSuccess(ctx, consts.StatusOK, toUserDTO(updated), nil)
}

func (h *UserHandler) patchUserStatus(c context.Context, ctx *app.RequestContext) {
	if h.Store == nil {
		transport.WriteErrorKey(ctx, transport.CodeInternalServerError, i18n.KeyUserStoreNotReady, nil, nil)
		return
	}

	username := strings.TrimSpace(ctx.Param("username"))
	if err := validateUsername(username); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	var req PatchUserStatusRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	status := strings.ToLower(strings.TrimSpace(req.Status))
	if err := validateStatus(status); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	updated, err := h.Store.UpdateUserStatus(c, username, status, actorFromContext(ctx))
	if err != nil {
		writeUserError(ctx, err)
		return
	}

	transport.WriteSuccess(ctx, consts.StatusOK, toUserDTO(updated), nil)
}

func normalizeCreateRequest(req CreateUserRequest) (userstore.CreateUserInput, error) {
	input := userstore.CreateUserInput{
		Username: strings.TrimSpace(req.Username),
		Email:    strings.TrimSpace(req.Email),
		Password: strings.TrimSpace(req.Password),
	}

	if err := validateUsername(input.Username); err != nil {
		return userstore.CreateUserInput{}, err
	}
	if err := validateEmail(input.Email); err != nil {
		return userstore.CreateUserInput{}, err
	}
	if err := validatePassword(input.Password); err != nil {
		return userstore.CreateUserInput{}, err
	}

	return input, nil
}

func validateUsername(username string) error {
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("invalid username: %q", username)
	}
	return nil
}

func validateEmail(email string) error {
	if email == "" {
		return errors.New("email is required")
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed == nil || strings.TrimSpace(parsed.Address) == "" {
		return fmt.Errorf("invalid email: %q", email)
	}
	return nil
}

func validatePassword(password string) error {
	if password == "" {
		return errors.New("password is required")
	}
	if len(password) < 6 {
		return errors.New("password length must be at least 6")
	}
	if len(password) > 72 {
		return errors.New("password length must be at most 72")
	}
	return nil
}

func validateStatus(status string) error {
	if status != userstore.StatusActive && status != userstore.StatusDisabled {
		return fmt.Errorf("invalid status: %q", status)
	}
	return nil
}

func actorFromContext(ctx *app.RequestContext) string {
	if ctx == nil {
		return "system"
	}
	if v, ok := ctx.Get("userName"); ok {
		if name, ok := v.(string); ok {
			name = strings.TrimSpace(name)
			if name != "" {
				return name
			}
		}
	}
	return "system"
}

func compareUserDTO(a, b UserDTO, field string) int {
	switch field {
	case "username":
		return strings.Compare(a.Username, b.Username)
	case "email":
		return strings.Compare(a.Email, b.Email)
	case "role":
		return strings.Compare(a.Role, b.Role)
	case "status":
		return strings.Compare(a.Status, b.Status)
	case "createdAt":
		return strings.Compare(a.CreatedAt, b.CreatedAt)
	default:
		return 0
	}
}

func writeUserError(ctx *app.RequestContext, err error) {
	switch {
	case errors.Is(err, userstore.ErrUserNameExists):
		transport.WriteError(ctx, transport.CodeConflict, err.Error(), nil)
	case errors.Is(err, userstore.ErrUserEmailExists):
		transport.WriteError(ctx, transport.CodeConflict, err.Error(), nil)
	case errors.Is(err, userstore.ErrUserNotFound):
		transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
	case errors.Is(err, userstore.ErrInvalidUserStatus):
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
	case errors.Is(err, userstore.ErrDeleteBuiltInUser):
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
	case errors.Is(err, userstore.ErrEmptyPassword):
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
	default:
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
	}
}
