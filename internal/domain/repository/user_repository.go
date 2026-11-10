package repository

import (
	"context"

	"daily-task-go/internal/domain/entity"
)

type UserFilter struct {
	Search string
	Page   int
	Limit  int
}

func (f UserFilter) Offset() int {
	if f.Page < 1 {
		return 0
	}
	return (f.Page - 1) * f.Limit
}

type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	FindAll(ctx context.Context, filter UserFilter) ([]entity.User, int64, error)
	FindByID(ctx context.Context, id uint) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	Update(ctx context.Context, user *entity.User) error
	Delete(ctx context.Context, id uint) error
}
