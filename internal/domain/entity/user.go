package entity

import (
	"errors"
	"strings"
	"time"
)

var ErrUserNotFound = errors.New("user tidak ditemukan")

const StatusAktif = "aktif"
const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

var ErrRoleTidakValid = errors.New("role tidak valid")

func (u *User) IsAdmin() bool { return u.Role == RoleAdmin }

func IsValidRole(role string) bool {
	switch role {
	case RoleUser, RoleAdmin:
		return true
	default:
		return false
	}
}

type User struct {
	ID        uint
	Email     string
	Password  string
	Name      string
	Address   string
	Job       string
	Status    string
	Role      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
