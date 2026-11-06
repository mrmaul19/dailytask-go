package entity

import (
	"errors"
	"time"
)

var ErrUserNotFound = errors.New("user tidak ditemukan")

type User struct {
	ID        uint
	Name      string
	Address   string
	Job       string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}
