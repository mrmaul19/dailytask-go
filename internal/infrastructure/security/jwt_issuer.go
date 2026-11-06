package security

import (
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"

	domainsecurity "daily-task-go/internal/domain/security"
)

// signingMethod dikunci ke satu algoritma, dan Parse hanya menerima algoritma
// ini. Tanpa pembatasan tersebut, token bisa dipalsukan lewat "alg confusion":
// penyerang mengirim token ber-alg "none", atau menandatangani dengan HMAC
// memakai public key sebagai kunci saat server mengira dirinya memakai RSA.
var signingMethod = jwt.SigningMethodHS256

// jwtClaims adalah bentuk claim di dalam token.
// Subject (sub) dipakai untuk id user karena itu claim standar JWT.
type jwtClaims struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

type jwtIssuer struct {
	secret []byte
	ttl    time.Duration
}

// NewJWTIssuer membuat penerbit token JWT.
//
// Pemeriksaan kelayakan secret dilakukan di composition root (main.go), supaya
// aplikasi berhenti saat start — bukan saat login pertama kali dicoba.
func NewJWTIssuer(secret string, ttl time.Duration) domainsecurity.TokenIssuer {
	return &jwtIssuer{secret: []byte(secret), ttl: ttl}
}

func (i *jwtIssuer) Issue(claims domainsecurity.Claims) (domainsecurity.Token, error) {
	now := time.Now()
	expiresAt := now.Add(i.ttl)

	token := jwt.NewWithClaims(signingMethod, jwtClaims{
		Email: claims.Email,
		Role:  claims.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatUint(uint64(claims.UserID), 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	})

	signed, err := token.SignedString(i.secret)
	if err != nil {
		return domainsecurity.Token{}, fmt.Errorf("menandatangani token: %w", err)
	}

	return domainsecurity.Token{Value: signed, ExpiresAt: expiresAt}, nil
}

func (i *jwtIssuer) Parse(raw string) (*domainsecurity.Claims, error) {
	var claims jwtClaims

	_, err := jwt.ParseWithClaims(raw, &claims,
		func(*jwt.Token) (any, error) { return i.secret, nil },
		// Hanya algoritma yang kita pakai sendiri yang diterima.
		jwt.WithValidMethods([]string{signingMethod.Alg()}),
		// Kedaluwarsa wajib ada; token tanpa exp ditolak, bukan dianggap abadi.
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		// Alasan detailnya sengaja tidak diteruskan ke pemanggil.
		return nil, domainsecurity.ErrTokenTidakValid
	}

	userID, err := strconv.ParseUint(claims.Subject, 10, 64)
	if err != nil || userID == 0 {
		return nil, domainsecurity.ErrTokenTidakValid
	}

	return &domainsecurity.Claims{
		UserID: uint(userID),
		Email:  claims.Email,
		Role:   claims.Role,
	}, nil
}
