package dto

import (
	"time"

	"daily-task-go/internal/domain/entity"
)

type CreateUserRequest struct {
	Email    string `form:"email" json:"email" validate:"required,email,max=255"`
	Password string `form:"password" json:"password" validate:"required,min=8,max=72"`
	Name     string `form:"name" json:"name" validate:"required,min=3,max=100"`
	Address  string `form:"address" json:"address" validate:"required,max=255"`
	Job      string `form:"job" json:"job" validate:"required,max=100"`
	Status   string `form:"status" json:"status" validate:"required,max=100"`
	Role     string `form:"role" json:"role" validate:"omitempty,oneof=user admin"`
}

type UpdateUserRequest struct {
	Email    *string `form:"email" json:"email" validate:"omitempty,email,max=255"`
	Password *string `form:"password" json:"password" validate:"omitempty,min=8,max=72"`
	Name     *string `form:"name" json:"name" validate:"omitempty,min=3,max=100"`
	Address  *string `form:"address" json:"address" validate:"omitempty,max=255"`
	Job      *string `form:"job" json:"job" validate:"omitempty,max=100"`
	Status   *string `form:"status" json:"status" validate:"omitempty,max=100"`
	Role     *string `form:"role" json:"role" validate:"omitempty,oneof=user admin"`
}

type UserResponse struct {
	ID        uint      `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	Job       string    `json:"job"`
	Status    string    `json:"status"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewUserResponse(user *entity.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Address:   user.Address,
		Job:       user.Job,
		Status:    user.Status,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func NewUserResponses(list []entity.User) []UserResponse {
	out := make([]UserResponse, 0, len(list))
	for i := range list {
		out = append(out, NewUserResponse(&list[i]))
	}
	return out
}
