package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"daily-task-go/internal/domain/entity"
	"daily-task-go/internal/domain/repository"
	"daily-task-go/pkg/apperror"
)

type StartTimerInput struct {
	UserID uint
	TaskID uint
	Note   string
}

type StopTimerInput struct {
	UserID uint
	Note   *string
}

type UpdateTimeEntryInput struct {
	TaskID    *uint
	StartedAt *time.Time
	EndedAt   *time.Time
	Note      *string
}

type ListTimeEntryInput struct {
	UserID uint
	TaskID uint
	From   *time.Time
	To     *time.Time
	Page   int
	Limit  int
}

type ListTimeEntryOutput struct {
	Entries []entity.TimeEntry
	Total   int64
	Page    int
	Limit   int
}

type TimeEntryUsecase interface {
	Start(ctx context.Context, in StartTimerInput) (*entity.TimeEntry, error)
	Stop(ctx context.Context, in StopTimerInput) (*entity.TimeEntry, error)
	Running(ctx context.Context, userID uint) (*entity.TimeEntry, error)
	List(ctx context.Context, in ListTimeEntryInput) (*ListTimeEntryOutput, error)
	GetByID(ctx context.Context, userID, id uint) (*entity.TimeEntry, error)
	Update(ctx context.Context, userID, id uint, in UpdateTimeEntryInput) (*entity.TimeEntry, error)
	Delete(ctx context.Context, userID, id uint) error
}

type timeEntryUsecase struct {
	timeEntryRepo repository.TimeEntryRepository
	taskRepo      repository.TaskRepository
}

func NewTimeEntryUsecase(
	timeEntryRepo repository.TimeEntryRepository,
	taskRepo repository.TaskRepository,
) TimeEntryUsecase {
	return &timeEntryUsecase{timeEntryRepo: timeEntryRepo, taskRepo: taskRepo}
}

func (u *timeEntryUsecase) Start(ctx context.Context, in StartTimerInput) (*entity.TimeEntry, error) {
	if _, err := u.taskRepo.FindByID(ctx, in.TaskID); err != nil {
		if errors.Is(err, entity.ErrTaskNotFound) {
			return nil, apperror.NotFound(fmt.Sprintf("task dengan id %d tidak ditemukan", in.TaskID))
		}
		return nil, apperror.Internal("gagal memeriksa task", err)
	}

	berjalan, err := u.timeEntryRepo.FindRunningByUserID(ctx, in.UserID)
	if err != nil && !errors.Is(err, entity.ErrTimeEntryNotFound) {
		return nil, apperror.Internal("gagal memeriksa timer yang berjalan", err)
	}
	if berjalan != nil {
		return nil, apperror.Conflict(fmt.Sprintf(
			"masih ada timer yang berjalan untuk task %d sejak %s, hentikan dulu",
			berjalan.TaskID, berjalan.StartedAt.Format(time.RFC3339)))
	}

	entry := &entity.TimeEntry{
		UserID:    in.UserID,
		TaskID:    in.TaskID,
		StartedAt: time.Now(),
		Note:      strings.TrimSpace(in.Note),
	}

	if err := u.timeEntryRepo.Create(ctx, entry); err != nil {
		if errors.Is(err, entity.ErrTimerSedangBerjalan) {
			return nil, apperror.Conflict("masih ada timer yang berjalan, hentikan dulu")
		}
		return nil, apperror.Internal("gagal memulai timer", err)
	}

	return entry, nil
}

func (u *timeEntryUsecase) Stop(ctx context.Context, in StopTimerInput) (*entity.TimeEntry, error) {
	entry, err := u.timeEntryRepo.FindRunningByUserID(ctx, in.UserID)
	if err != nil {
		if errors.Is(err, entity.ErrTimeEntryNotFound) {
			return nil, apperror.Conflict("tidak ada timer yang sedang berjalan")
		}
		return nil, apperror.Internal("gagal mengambil timer yang berjalan", err)
	}

	if in.Note != nil {
		entry.Note = strings.TrimSpace(*in.Note)
	}

	if err := entry.Stop(time.Now()); err != nil {
		return nil, terjemahkanErrorWaktu(err)
	}

	if err := u.timeEntryRepo.Update(ctx, entry); err != nil {
		return nil, apperror.Internal("gagal menghentikan timer", err)
	}

	return entry, nil
}

func (u *timeEntryUsecase) Running(ctx context.Context, userID uint) (*entity.TimeEntry, error) {
	entry, err := u.timeEntryRepo.FindRunningByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, entity.ErrTimeEntryNotFound) {
			return nil, nil
		}
		return nil, apperror.Internal("gagal mengambil timer yang berjalan", err)
	}
	return entry, nil
}

func (u *timeEntryUsecase) List(ctx context.Context, in ListTimeEntryInput) (*ListTimeEntryOutput, error) {
	page, limit := normalizePaging(in.Page, in.Limit)

	if in.From != nil && in.To != nil && !in.To.After(*in.From) {
		return nil, apperror.Validation("rentang tanggal tidak valid",
			map[string]string{"to": "harus setelah from"})
	}

	entries, total, err := u.timeEntryRepo.FindAll(ctx, repository.TimeEntryFilter{
		UserID: in.UserID,
		TaskID: in.TaskID,
		From:   in.From,
		To:     in.To,
		Page:   page,
		Limit:  limit,
	})
	if err != nil {
		return nil, apperror.Internal("gagal mengambil daftar catatan waktu", err)
	}

	return &ListTimeEntryOutput{Entries: entries, Total: total, Page: page, Limit: limit}, nil
}

func (u *timeEntryUsecase) GetByID(ctx context.Context, userID, id uint) (*entity.TimeEntry, error) {
	entry, err := u.timeEntryRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, entity.ErrTimeEntryNotFound) {
			return nil, apperror.NotFound(fmt.Sprintf("catatan waktu dengan id %d tidak ditemukan", id))
		}
		return nil, apperror.Internal("gagal mengambil catatan waktu", err)
	}
	if entry.UserID != userID {
		return nil, apperror.NotFound(fmt.Sprintf("catatan waktu dengan id %d tidak ditemukan", id))
	}

	return entry, nil
}

func (u *timeEntryUsecase) Update(ctx context.Context, userID, id uint, in UpdateTimeEntryInput) (*entity.TimeEntry, error) {
	entry, err := u.GetByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	if in.TaskID != nil {
		if _, err := u.taskRepo.FindByID(ctx, *in.TaskID); err != nil {
			if errors.Is(err, entity.ErrTaskNotFound) {
				return nil, apperror.NotFound(fmt.Sprintf("task dengan id %d tidak ditemukan", *in.TaskID))
			}
			return nil, apperror.Internal("gagal memeriksa task", err)
		}
		entry.TaskID = *in.TaskID
	}

	if in.StartedAt != nil {
		entry.StartedAt = *in.StartedAt
	}
	if in.EndedAt != nil {
		selesai := *in.EndedAt
		entry.EndedAt = &selesai
	}
	if in.Note != nil {
		entry.Note = strings.TrimSpace(*in.Note)
	}

	if err := entry.Validate(); err != nil {
		return nil, terjemahkanErrorWaktu(err)
	}

	if err := u.timeEntryRepo.Update(ctx, entry); err != nil {
		if errors.Is(err, entity.ErrRentangWaktuTidakValid) {
			return nil, terjemahkanErrorWaktu(err)
		}
		return nil, apperror.Internal("gagal memperbarui catatan waktu", err)
	}

	return entry, nil
}

func (u *timeEntryUsecase) Delete(ctx context.Context, userID, id uint) error {
	entry, err := u.GetByID(ctx, userID, id)
	if err != nil {
		return err
	}

	if err := u.timeEntryRepo.Delete(ctx, entry.ID); err != nil {
		return apperror.Internal("gagal menghapus catatan waktu", err)
	}

	return nil
}

func terjemahkanErrorWaktu(err error) error {
	switch {
	case errors.Is(err, entity.ErrRentangWaktuTidakValid):
		return apperror.Validation("rentang waktu tidak valid",
			map[string]string{"ended_at": "harus setelah waktu mulai"})
	case errors.Is(err, entity.ErrTimerTidakBerjalan):
		return apperror.Conflict("timer ini sudah dihentikan")
	default:
		return apperror.Internal("gagal memproses catatan waktu", err)
	}
}
