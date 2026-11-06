package dto

import (
	"time"

	"daily-task-go/internal/domain/entity"
)

type CreateUserRequest struct {
	Name    string `form:"name" json:"name" validate:"required,min=3,max=100"`
	Address string `form:"address" json:"address" validate:"required,max=255"`
	Job     string `form:"job" json:"job" validate:"required,max=100"`
	Status  string `form:"status" json:"status" validate:"required,max=100"`
}

type UpdateUserRequest struct {
	Name    *string `form:"name" json:"name" validate:"required,min=3,max=100"`
	Address *string `form:"address" json:"address" validate:"required,max=255"`
	Job     *string `form:"job" json:"job" validate:"required,max=100"`
	Status  *string `form:"status" json:"status" validate:"required,max=100"`
}

type UserResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	Job       string    `json:"job"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewUserResponse(user *entity.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Address:   user.Address,
		Job:       user.Job,
		Status:    user.Status,
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
