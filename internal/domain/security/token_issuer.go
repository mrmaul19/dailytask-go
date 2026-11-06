package security

import (
	"errors"
	"time"
)

// ErrTokenTidakValid dikembalikan Parse untuk SEMUA kegagalan: tanda tangan
// salah, token kedaluwarsa, format rusak, maupun algoritma yang tidak diizinkan.
// Sengaja satu error saja — membedakan "kedaluwarsa" dari "tanda tangan salah"
// di respons HTTP hanya memberi petunjuk gratis kepada penyerang.
var ErrTokenTidakValid = errors.New("token tidak valid")

// Claims adalah isi token yang dipercaya setelah tanda tangannya terverifikasi.
// Sengaja minimal: makin banyak data di dalam token, makin lama ia basi, karena
// token yang sudah terbit tidak bisa ditarik kembali.
type Claims struct {
	UserID uint
	Email  string
	// Role ikut dibawa di dalam token supaya middleware tidak perlu query
	// database pada setiap request.
	//
	// Harganya: token yang sudah terbit membawa role LAMA. Admin yang baru
	// diturunkan jadi user biasa masih punya hak admin sampai tokennya
	// kedaluwarsa. Itu alasan konkret kenapa JWT_TTL sebaiknya pendek.
	Role string
}

// Token adalah hasil penerbitan, beserta kapan ia kedaluwarsa.
// ExpiresAt ikut dikembalikan supaya client tahu kapan harus login lagi
// tanpa perlu membongkar isi token.
type Token struct {
	Value     string
	ExpiresAt time.Time
}

// TokenIssuer menerbitkan dan memeriksa token akses.
//
// Dibuat sebagai port supaya usecase dan middleware tidak terikat ke JWT:
// mengganti JWT dengan session opaque di Redis cukup mengganti adapter-nya.
type TokenIssuer interface {
	Issue(claims Claims) (Token, error)
	Parse(raw string) (*Claims, error)
}
