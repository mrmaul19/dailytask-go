package entity

import (
	"errors"
	"strings"
	"time"
)

var ErrUserNotFound = errors.New("user tidak ditemukan")

// StatusAktif adalah status bawaan untuk user yang mendaftar sendiri lewat
// endpoint register. Pendaftar tidak boleh menentukan statusnya sendiri.
const StatusAktif = "aktif"

// Role menentukan apa yang boleh dilakukan seorang user.
//
// RoleUser adalah nilai bawaan — baik untuk baris lama (lewat DEFAULT di
// migrasi) maupun untuk pendaftar baru. Default yang aman adalah yang paling
// sedikit haknya: kalau defaultnya admin, semua akun yang sudah ada mendadak
// punya hak penuh.
const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

var ErrRoleTidakValid = errors.New("role tidak valid")

// IsAdmin ditaruh di entity, bukan di middleware, karena "siapa yang dianggap
// admin" adalah aturan bisnis — middleware hanya menegakkannya.
func (u *User) IsAdmin() bool { return u.Role == RoleAdmin }

// IsValidRole memeriksa apakah sebuah nilai role dikenali.
func IsValidRole(role string) bool {
	switch role {
	case RoleUser, RoleAdmin:
		return true
	default:
		return false
	}
}

type User struct {
	ID    uint
	Email string
	// Password menyimpan HASH, bukan password asli. Yang mengubah password
	// polos menjadi hash adalah usecase lewat port security.PasswordHasher;
	// nilai polosnya tidak pernah sampai ke entity maupun database.
	Password  string
	Name      string
	Address   string
	Job       string
	Status    string
	Role      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NormalizeEmail menyeragamkan email sebelum disimpan atau dibandingkan.
// Email bersifat case-insensitive, jadi "Budi@Mail.com" dan "budi@mail.com"
// harus dianggap satu orang yang sama.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
