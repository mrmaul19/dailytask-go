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

type CreateGoalsInput struct {
	Title    string
	Category string
	Priority string
	Status   string
	Progress string
}

type UpdateGoalsInput struct {
	Title    *string
	Category *string
	Priority *string
	Status   *string
	Progress *string
}

type ListGoalsInput struct {
	Search string
	Page   int
	Limit  int
}

type ListGoalsOutput struct {
	Goals []entity.Goals
	Total int64
	Page  int
	Limit int
}

// ---------- Kontrak usecase ----------

type GoalsUsecase interface {
	Create(ctx context.Context, in CreateGoalsInput) (*entity.Goals, error)
	List(ctx context.Context, in ListGoalsInput) (*ListGoalsOutput, error)
	GetByID(ctx context.Context, id uint) (*entity.Goals, error)
	Update(ctx context.Context, id uint, in UpdateGoalsInput) (*entity.Goals, error)
	Delete(ctx context.Context, id uint) error
}

type goalsUsecase struct {
	goalsRepo repository.GoalsRepository
}

// NewGoalsUsecase menerima port lewat constructor (dependency injection),
// sehingga mudah diganti mock saat testing.
func NewGoalsUsecase(goalsRepo repository.GoalsRepository) GoalsUsecase {
	return &goalsUsecase{goalsRepo: goalsRepo}
}

func (u *goalsUsecase) Create(ctx context.Context, in CreateGoalsInput) (*entity.Goals, error) {
	goals := &entity.Goals{
		Title:    strings.TrimSpace(in.Title),
		Category: strings.TrimSpace(in.Category),
		Priority: strings.TrimSpace(in.Priority),
		Status:   strings.TrimSpace(in.Status),
		Progress: strings.TrimSpace(in.Progress),
	}

	if err := u.goalsRepo.Create(ctx, goals); err != nil {
		return nil, apperror.Internal("gagal menyimpan goals", err)
	}

	return goals, nil
}

func (u *goalsUsecase) List(ctx context.Context, in ListGoalsInput) (*ListGoalsOutput, error) {
	page, limit := normalizePaging(in.Page, in.Limit)

	goals, total, err := u.goalsRepo.FindAll(ctx, repository.GoalsFilter{
		Search: strings.TrimSpace(in.Search),
		Page:   page,
		Limit:  limit,
	})
	if err != nil {
		return nil, apperror.Internal("gagal mengambil daftar goals", err)
	}

	return &ListGoalsOutput{Goals: goals, Total: total, Page: page, Limit: limit}, nil
}

func (u *goalsUsecase) GetByID(ctx context.Context, id uint) (*entity.Goals, error) {
	goals, err := u.goalsRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, entity.ErrGoalsNotFound) {
			return nil, apperror.NotFound(fmt.Sprintf("goals dengan id %d tidak ditemukan", id))
		}
		return nil, apperror.Internal("gagal mengambil goals", err)
	}
	return goals, nil
}

func (u *goalsUsecase) Update(ctx context.Context, id uint, in UpdateGoalsInput) (*entity.Goals, error) {
	goals, err := u.GetByID(ctx, id)
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
		{"title", in.Title, &goals.Title},
		{"category", in.Category, &goals.Category},
		{"priority", in.Priority, &goals.Priority},
		{"status", in.Status, &goals.Status},
		{"progress", in.Progress, &goals.Progress},
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

	if err := u.goalsRepo.Update(ctx, goals); err != nil {
		return nil, apperror.Internal("gagal memperbarui goals", err)
	}

	return goals, nil
}

func (u *goalsUsecase) Delete(ctx context.Context, id uint) error {
	goals, err := u.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := u.goalsRepo.Delete(ctx, goals.ID); err != nil {
		return apperror.Internal("gagal menghapus goals", err)
	}

	return nil
}
