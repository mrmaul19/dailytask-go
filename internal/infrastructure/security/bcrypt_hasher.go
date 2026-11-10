package security

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	domainsecurity "daily-task-go/internal/domain/security"
)

const MaxPasswordBytes = 72

var ErrPasswordTerlaluPanjang = errors.New("password melebihi 72 byte")

type bcryptHasher struct {
	cost int
}

func NewBcryptHasher(cost int) domainsecurity.PasswordHasher {
	if cost <= 0 {
		cost = bcrypt.DefaultCost
	}
	return &bcryptHasher{cost: cost}
}

func (h *bcryptHasher) Hash(plain string) (string, error) {
	if len(plain) > MaxPasswordBytes {
		return "", ErrPasswordTerlaluPanjang
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), h.cost)
	if err != nil {
		return "", fmt.Errorf("bcrypt: %w", err)
	}
	return string(hashed), nil
}

func (h *bcryptHasher) Verify(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}
