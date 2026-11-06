// File ini hanya dipakai untuk dokumentasi Swagger/OpenAPI.
//
// pkg/response sengaja memakai `Data any` supaya satu bentuk body bisa dipakai
// seluruh endpoint. Konsekuensinya, generator OpenAPI tidak bisa menebak isi
// `data`. Envelope di bawah mengisi kekosongan itu: tipe konkret per endpoint
// yang dipakai pada anotasi `@Success` / `@Failure`.
//
// Tidak ada kode runtime yang memakai struct ini — kalau bentuk response di
// pkg/response berubah, file ini harus ikut disesuaikan secara manual.
package dto

import "daily-task-go/pkg/response"

// ---------- envelope umum ----------

// MessageEnvelope dipakai endpoint yang tidak mengembalikan data (mis. DELETE).
type MessageEnvelope struct {
	Success bool   `json:"success" example:"true"`
	Message string `json:"message" example:"task berhasil dihapus"`
}

// ErrorEnvelope adalah bentuk body untuk semua response gagal.
// Isi `errors` berbeda per kasus: map field -> pesan untuk error validasi,
// dan null untuk error yang tidak terkait field tertentu.
type ErrorEnvelope struct {
	Success bool              `json:"success" example:"false"`
	Message string            `json:"message" example:"data tidak valid"`
	Errors  map[string]string `json:"errors,omitempty"`
}

// HealthData adalah isi `data` pada GET /health.
type HealthData struct {
	Status string `json:"status" example:"ok"`
}

// HealthEnvelope adalah response GET /health.
type HealthEnvelope struct {
	Success bool       `json:"success" example:"true"`
	Message string     `json:"message" example:"service berjalan normal"`
	Data    HealthData `json:"data"`
}

// ---------- auth ----------

// AuthEnvelope adalah response register dan login.
type AuthEnvelope struct {
	Success bool         `json:"success" example:"true"`
	Message string       `json:"message" example:"login berhasil"`
	Data    AuthResponse `json:"data"`
}

// ---------- tasks ----------

// TaskEnvelope adalah response satu task.
type TaskEnvelope struct {
	Success bool         `json:"success" example:"true"`
	Message string       `json:"message" example:"detail task berhasil diambil"`
	Data    TaskResponse `json:"data"`
}

// TaskListEnvelope adalah response daftar task beserta metadata paginasi.
type TaskListEnvelope struct {
	Success bool           `json:"success" example:"true"`
	Message string         `json:"message" example:"daftar task berhasil diambil"`
	Data    []TaskResponse `json:"data"`
	Meta    response.Meta  `json:"meta"`
}

// ---------- goals ----------

// GoalsEnvelope adalah response satu goals.
type GoalsEnvelope struct {
	Success bool          `json:"success" example:"true"`
	Message string        `json:"message" example:"detail goals berhasil diambil"`
	Data    GoalsResponse `json:"data"`
}

// GoalsListEnvelope adalah response daftar goals beserta metadata paginasi.
type GoalsListEnvelope struct {
	Success bool            `json:"success" example:"true"`
	Message string          `json:"message" example:"daftar goals berhasil diambil"`
	Data    []GoalsResponse `json:"data"`
	Meta    response.Meta   `json:"meta"`
}

// ---------- users ----------

// UserEnvelope adalah response satu user.
type UserEnvelope struct {
	Success bool         `json:"success" example:"true"`
	Message string       `json:"message" example:"detail user berhasil diambil"`
	Data    UserResponse `json:"data"`
}

// UserListEnvelope adalah response daftar user beserta metadata paginasi.
type UserListEnvelope struct {
	Success bool           `json:"success" example:"true"`
	Message string         `json:"message" example:"daftar user berhasil diambil"`
	Data    []UserResponse `json:"data"`
	Meta    response.Meta  `json:"meta"`
}
