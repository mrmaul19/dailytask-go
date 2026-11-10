package repository

import (
	"context"
	"time"

	"daily-task-go/internal/domain/entity"
)

type TimeEntryFilter struct {
	UserID uint
	TaskID uint
	From   *time.Time
	To     *time.Time
	Page   int
	Limit  int
}

func (f TimeEntryFilter) Offset() int {
	if f.Page < 1 {
		return 0
	}
	return (f.Page - 1) * f.Limit
}

type TimeEntryRepository interface {
	Create(ctx context.Context, entry *entity.TimeEntry) error
	FindAll(ctx context.Context, filter TimeEntryFilter) ([]entity.TimeEntry, int64, error)
	FindByID(ctx context.Context, id uint) (*entity.TimeEntry, error)
	FindRunningByUserID(ctx context.Context, userID uint) (*entity.TimeEntry, error)
	Update(ctx context.Context, entry *entity.TimeEntry) error
	Delete(ctx context.Context, id uint) error
}
