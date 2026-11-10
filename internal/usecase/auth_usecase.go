package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"

	"daily-task-go/internal/domain/entity"
	"daily-task-go/internal/domain/repository"
	"daily-task-go/internal/domain/security"
	"daily-task-go/pkg/apperror"
)

const pesanKredensialSalah = "email atau password salah"

type RegisterInput struct {
	Email    string
	Password string
	Name     string
	Address  string
	Job      string
}

type LoginInput struct {
	Email    string
	Password string
}

type AuthOutput struct {
	User  *entity.User
	Token security.Token
}

// ---------- Kontrak usecase ----------

type AuthUsecase interface {
	Register(ctx context.Context, in RegisterInput) (*AuthOutput, error)
	Login(ctx context.Context, in LoginInput) (*AuthOutput, error)
}

type authUsecase struct {
	users     UserUsecase
	userRepo  repository.UserRepository
	hasher    security.PasswordHasher
	tokens    security.TokenIssuer
	dummyHash string
}

func NewAuthUsecase(
	users UserUsecase,
	userRepo repository.UserRepository,
	hasher security.PasswordHasher,
	tokens security.TokenIssuer,
) AuthUsecase {
	return &authUsecase{
		users:     users,
		userRepo:  userRepo,
		hasher:    hasher,
		tokens:    tokens,
		dummyHash: buatDummyHash(hasher),
	}
}

func (a *authUsecase) Register(ctx context.Context, in RegisterInput) (*AuthOutput, error) {
	user, err := a.users.Create(ctx, CreateUserInput{
		Email:    in.Email,
		Password: in.Password,
		Name:     in.Name,
		Address:  in.Address,
		Job:      in.Job,
		Status:   entity.StatusAktif,
		Role:     entity.RoleUser,
	})
	if err != nil {
		return nil, err
	}

	return a.terbitkanToken(user)
}

func (a *authUsecase) Login(ctx context.Context, in LoginInput) (*AuthOutput, error) {
	email := entity.NormalizeEmail(in.Email)

	user, err := a.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, entity.ErrUserNotFound) {
			_ = a.hasher.Verify(a.dummyHash, in.Password)
			return nil, apperror.Unauthorized(pesanKredensialSalah)
		}
		return nil, apperror.Internal("gagal memeriksa kredensial", err)
	}

	if err := a.hasher.Verify(user.Password, in.Password); err != nil {
		return nil, apperror.Unauthorized(pesanKredensialSalah)
	}

	return a.terbitkanToken(user)
}

func (a *authUsecase) terbitkanToken(user *entity.User) (*AuthOutput, error) {
	token, err := a.tokens.Issue(security.Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
	})
	if err != nil {
		return nil, apperror.Internal("gagal menerbitkan token", err)
	}
	return &AuthOutput{User: user, Token: token}, nil
}

func buatDummyHash(hasher security.PasswordHasher) string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	hashed, err := hasher.Hash(hex.EncodeToString(buf))
	if err != nil {
		return ""
	}
	return hashed
}
