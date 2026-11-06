package entity

import (
	"errors"
	"time"
)

var ErrGoalsNotFound = errors.New("goals tidak ditemukan")

// Goals merepresentasikan satu goal milik user.
type Goals struct {
	ID        uint
	Title     string
	Category  string
	Priority  string
	Status    string
	Progress  string
	CreatedAt time.Time
	UpdatedAt time.Time
}
