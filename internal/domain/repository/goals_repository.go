package repository

import (
	"context"

	"daily-task-go/internal/domain/entity"
)

type GoalsFilter struct {
	Search string
	Page   int
	Limit  int
}

func (f GoalsFilter) Offset() int {
	if f.Page < 1 {
		return 0
	}
	return (f.Page - 1) * f.Limit
}

type GoalsRepository interface {
	Create(ctx context.Context, goals *entity.Goals) error
	FindAll(ctx context.Context, filter GoalsFilter) ([]entity.Goals, int64, error)
	FindByID(ctx context.Context, id uint) (*entity.Goals, error)
	Update(ctx context.Context, goals *entity.Goals) error
	Delete(ctx context.Context, id uint) error
}
