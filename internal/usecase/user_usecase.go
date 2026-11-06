package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"daily-task-go/internal/domain/entity"
	"daily-task-go/internal/domain/repository"
	"daily-task-go/pkg/apperror"
)

// ---------- Input / Output (bentuk data antar-layer, bebas dari HTTP) ----------

type CreateUserInput struct {
	Name    string
	Address string
	Job     string
	Status  string
}

type UpdateUserInput struct {
	Name    *string
	Address *string
	Job     *string
	Status  *string
}

type ListUserInput struct {
	Search string
	Page   int
	Limit  int
}

type ListUserOutput struct {
	Users []entity.User
	Total int64
	Page  int
	Limit int
}

// ---------- Kontrak usecase ----------

type UserUsecase interface {
	Create(ctx context.Context, in CreateUserInput) (*entity.User, error)
	List(ctx context.Context, in ListUserInput) (*ListUserOutput, error)
	GetByID(ctx context.Context, id uint) (*entity.User, error)
	Update(ctx context.Context, id uint, in UpdateUserInput) (*entity.User, error)
	Delete(ctx context.Context, id uint) error
}

type userUsecase struct {
	userRepo repository.UserRepository
}

// NewUserUsecase menerima port lewat constructor (dependency injection),
// sehingga mudah diganti mock saat testing.
func NewUserUsecase(userRepo repository.UserRepository) UserUsecase {
	return &userUsecase{userRepo: userRepo}
}

func (u *userUsecase) Create(ctx context.Context, in CreateUserInput) (*entity.User, error) {
	user := &entity.User{
		Name:    strings.TrimSpace(in.Name),
		Address: strings.TrimSpace(in.Address),
		Job:     strings.TrimSpace(in.Job),
		Status:  strings.TrimSpace(in.Status),
	}

	if err := u.userRepo.Create(ctx, user); err != nil {
		return nil, apperror.Internal("gagal menyimpan user", err)
	}

	return user, nil
}

func (u *userUsecase) List(ctx context.Context, in ListUserInput) (*ListUserOutput, error) {
	page, limit := normalizePaging(in.Page, in.Limit)

	users, total, err := u.userRepo.FindAll(ctx, repository.UserFilter{
		Search: strings.TrimSpace(in.Search),
		Page:   page,
		Limit:  limit,
	})
	if err != nil {
		return nil, apperror.Internal("gagal mengambil daftar user", err)
	}

	return &ListUserOutput{Users: users, Total: total, Page: page, Limit: limit}, nil
}

func (u *userUsecase) GetByID(ctx context.Context, id uint) (*entity.User, error) {
	user, err := u.userRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, entity.ErrUserNotFound) {
			return nil, apperror.NotFound(fmt.Sprintf("user dengan id %d tidak ditemukan", id))
		}
		return nil, apperror.Internal("gagal mengambil user", err)
	}
	return user, nil
}

func (u *userUsecase) Update(ctx context.Context, id uint, in UpdateUserInput) (*entity.User, error) {
	user, err := u.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Pointer dipakai supaya "field tidak dikirim" (biarkan nilai lama)
	// bisa dibedakan dari "field dikirim kosong" (ditolak).
	fields := []struct {
		name  string
		input *string
		dest  *string
	}{
		{"name", in.Name, &user.Name},
		{"address", in.Address, &user.Address},
		{"job", in.Job, &user.Job},
		{"status", in.Status, &user.Status},
	}

	for _, f := range fields {
		if f.input == nil {
			continue
		}
		value := strings.TrimSpace(*f.input)
		if value == "" {
			return nil, apperror.Validation("data tidak valid",
				map[string]string{f.name: "tidak boleh kosong"})
		}
		*f.dest = value
	}

	if err := u.userRepo.Update(ctx, user); err != nil {
		return nil, apperror.Internal("gagal memperbarui user", err)
	}

	return user, nil
}

func (u *userUsecase) Delete(ctx context.Context, id uint) error {
	user, err := u.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := u.userRepo.Delete(ctx, user.ID); err != nil {
		return apperror.Internal("gagal menghapus user", err)
	}

	return nil
}
