// Package security adalah adapter yang mengimplementasikan port
// domain/security memakai bcrypt.
package security

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	domainsecurity "daily-task-go/internal/domain/security"
)

// MaxPasswordBytes adalah batas keras dari bcrypt: input di atas 72 byte
// DIABAIKAN diam-diam oleh algoritmanya. Kalau tidak ditolak di sini,
// "password panjang sekali....A" dan "password panjang sekali....B" bisa
// dianggap password yang sama. DTO memvalidasi batas ini lebih dulu;
// pemeriksaan di sini adalah jaring pengaman kalau dipanggil dari jalur lain.
const MaxPasswordBytes = 72

var ErrPasswordTerlaluPanjang = errors.New("password melebihi 72 byte")

type bcryptHasher struct {
	cost int
}

// NewBcryptHasher membuat hasher bcrypt. cost <= 0 memakai bcrypt.DefaultCost.
// Semakin besar cost, semakin lambat hashing-nya — dan itu memang tujuannya,
// karena yang lambat bagi kita juga lambat bagi penebak password.
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

// Verify membandingkan password polos dengan hash-nya.
//
// bcrypt.CompareHashAndPassword memakai perbandingan waktu-tetap, jadi lama
// eksekusinya tidak bergantung pada seberapa mirip password yang dicoba.
// Hash yang kosong atau rusak (misalnya baris user lama yang password-nya
// masih ”) otomatis gagal di sini — itu memang yang diinginkan.
func (h *bcryptHasher) Verify(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}
