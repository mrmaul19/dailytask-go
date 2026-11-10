package dto

import (
	"time"

	"daily-task-go/internal/domain/entity"
	"daily-task-go/internal/domain/security"
)

type RegisterRequest struct {
	Email    string `form:"email" json:"email" validate:"required,email,max=255"`
	Password string `form:"password" json:"password" validate:"required,min=8,max=72"`
	Name     string `form:"name" json:"name" validate:"required,min=3,max=100"`
	Address  string `form:"address" json:"address" validate:"required,max=255"`
	Job      string `form:"job" json:"job" validate:"required,max=100"`
}

type LoginRequest struct {
	Email    string `form:"email" json:"email" validate:"required,email,max=255"`
	Password string `form:"password" json:"password" validate:"required,max=72"`
}

type AuthResponse struct {
	Token     string       `json:"token"`
	ExpiresAt time.Time    `json:"expires_at"`
	User      UserResponse `json:"user"`
}

func NewAuthResponse(user *entity.User, token security.Token) AuthResponse {
	return AuthResponse{
		Token:     token.Value,
		ExpiresAt: token.ExpiresAt,
		User:      NewUserResponse(user),
	}
}
