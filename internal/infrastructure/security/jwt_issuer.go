package security

import (
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"

	domainsecurity "daily-task-go/internal/domain/security"
)

var signingMethod = jwt.SigningMethodHS256

type jwtClaims struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

type jwtIssuer struct {
	secret []byte
	ttl    time.Duration
}

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
		jwt.WithValidMethods([]string{signingMethod.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
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
