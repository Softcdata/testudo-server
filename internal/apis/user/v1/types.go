package user

import "github.com/softcdata/testudo-server/internal/userstore"

type UserDTO struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}

type CreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type PatchUserStatusRequest struct {
	Status string `json:"status"`
}

type PatchUserPasswordRequest struct {
	Password string `json:"password"`
}

type DeleteUserResponse struct {
	Username string `json:"username"`
	Deleted  bool   `json:"deleted"`
}

func toUserDTO(user userstore.UserRecord) UserDTO {
	return UserDTO{
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
	}
}
