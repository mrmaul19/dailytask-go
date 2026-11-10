package repository

import (
	"context"

	"daily-task-go/internal/domain/entity"
)

type TaskFilter struct {
	Search string
	Page   int
	Limit  int
}

func (f TaskFilter) Offset() int {
	if f.Page < 1 {
		return 0
	}
	return (f.Page - 1) * f.Limit
}

type TaskRepository interface {
	Create(ctx context.Context, task *entity.Task) error
	FindAll(ctx context.Context, filter TaskFilter) ([]entity.Task, int64, error)
	FindByID(ctx context.Context, id uint) (*entity.Task, error)
	FindByNoTask(ctx context.Context, noTask string) (*entity.Task, error)
	Update(ctx context.Context, task *entity.Task) error
	Delete(ctx context.Context, id uint) error
}
