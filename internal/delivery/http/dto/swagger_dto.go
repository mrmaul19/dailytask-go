package dto

import "daily-task-go/pkg/response"

type MessageEnvelope struct {
	Success bool   `json:"success" example:"true"`
	Message string `json:"message" example:"task berhasil dihapus"`
}

type ErrorEnvelope struct {
	Success bool              `json:"success" example:"false"`
	Message string            `json:"message" example:"data tidak valid"`
	Errors  map[string]string `json:"errors,omitempty"`
}

type HealthData struct {
	Status string `json:"status" example:"ok"`
}

type HealthEnvelope struct {
	Success bool       `json:"success" example:"true"`
	Message string     `json:"message" example:"service berjalan normal"`
	Data    HealthData `json:"data"`
}

type AuthEnvelope struct {
	Success bool         `json:"success" example:"true"`
	Message string       `json:"message" example:"login berhasil"`
	Data    AuthResponse `json:"data"`
}

type TaskEnvelope struct {
	Success bool         `json:"success" example:"true"`
	Message string       `json:"message" example:"detail task berhasil diambil"`
	Data    TaskResponse `json:"data"`
}

type TaskListEnvelope struct {
	Success bool           `json:"success" example:"true"`
	Message string         `json:"message" example:"daftar task berhasil diambil"`
	Data    []TaskResponse `json:"data"`
	Meta    response.Meta  `json:"meta"`
}

type GoalsEnvelope struct {
	Success bool          `json:"success" example:"true"`
	Message string        `json:"message" example:"detail goals berhasil diambil"`
	Data    GoalsResponse `json:"data"`
}

type GoalsListEnvelope struct {
	Success bool            `json:"success" example:"true"`
	Message string          `json:"message" example:"daftar goals berhasil diambil"`
	Data    []GoalsResponse `json:"data"`
	Meta    response.Meta   `json:"meta"`
}

type TimeEntryEnvelope struct {
	Success bool              `json:"success" example:"true"`
	Message string            `json:"message" example:"timer berhasil dimulai"`
	Data    TimeEntryResponse `json:"data"`
}

type RunningTimeEntryEnvelope struct {
	Success bool               `json:"success" example:"true"`
	Message string             `json:"message" example:"tidak ada timer yang sedang berjalan"`
	Data    *TimeEntryResponse `json:"data"`
}

type TimeEntryListEnvelope struct {
	Success bool                `json:"success" example:"true"`
	Message string              `json:"message" example:"daftar catatan waktu berhasil diambil"`
	Data    []TimeEntryResponse `json:"data"`
	Meta    response.Meta       `json:"meta"`
}

type UserEnvelope struct {
	Success bool         `json:"success" example:"true"`
	Message string       `json:"message" example:"detail user berhasil diambil"`
	Data    UserResponse `json:"data"`
}

type UserListEnvelope struct {
	Success bool           `json:"success" example:"true"`
	Message string         `json:"message" example:"daftar user berhasil diambil"`
	Data    []UserResponse `json:"data"`
	Meta    response.Meta  `json:"meta"`
}
