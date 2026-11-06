package dto

import (
	"time"

	"daily-task-go/internal/domain/entity"
	"daily-task-go/internal/domain/security"
)

// RegisterRequest sengaja TIDAK punya field `status`.
//
// Inilah bedanya dengan CreateUserRequest: pendaftar umum tidak boleh
// menentukan status akunnya sendiri. Kalau field itu ada di sini, siapa pun
// bisa mengirim status apa pun saat mendaftar. Nilainya dipaksa oleh usecase.
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

// AuthResponse adalah hasil register maupun login.
//
// User memakai UserResponse yang sudah ada, sehingga password otomatis tidak
// ikut terkirim — DTO itu memang tidak punya field tersebut.
type AuthResponse struct {
	Token string `json:"token"`
	// ExpiresAt dikirim supaya client tahu kapan harus login lagi tanpa
	// perlu membongkar isi token sendiri.
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
