package security

import (
	"errors"
	"time"
)

var ErrTokenTidakValid = errors.New("token tidak valid")

type Claims struct {
	UserID uint
	Email  string
	Role   string
}

type Token struct {
	Value     string
	ExpiresAt time.Time
}

type TokenIssuer interface {
	Issue(claims Claims) (Token, error)
	Parse(raw string) (*Claims, error)
}
