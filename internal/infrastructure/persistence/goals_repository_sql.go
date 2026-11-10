package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"daily-task-go/internal/domain/entity"
	"daily-task-go/internal/domain/repository"
)

const goalsColumns = `id, title, category, priority, status, progress, created_at, updated_at`

type goalsRepositorySQL struct {
	db *sql.DB
}

func NewGoalsRepository(db *sql.DB) repository.GoalsRepository {
	return &goalsRepositorySQL{db: db}
}

func scanGoals(s rowScanner) (*entity.Goals, error) {
	var goals entity.Goals
	err := s.Scan(
		&goals.ID,
		&goals.Title,
		&goals.Category,
		&goals.Priority,
		&goals.Status,
		&goals.Progress,
		&goals.CreatedAt,
		&goals.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &goals, nil
}

func (r *goalsRepositorySQL) Create(ctx context.Context, goals *entity.Goals) error {
	const query = `
		INSERT INTO goals (title, category, priority, status, progress, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, now(), now())
		RETURNING ` + goalsColumns

	row := r.db.QueryRowContext(ctx, query,
		goals.Title, goals.Category, goals.Priority, goals.Status, goals.Progress)

	created, err := scanGoals(row)
	if err != nil {
		return err
	}
	*goals = *created
	return nil
}

func (r *goalsRepositorySQL) FindAll(ctx context.Context, filter repository.GoalsFilter) ([]entity.Goals, int64, error) {
	conditions := []string{notDeleted}
	args := []any{}

	if search := strings.TrimSpace(filter.Search); search != "" {
		args = append(args, "%"+search+"%")
		conditions = append(conditions, fmt.Sprintf(
			"(title ILIKE $%d OR category ILIKE $%d)", len(args), len(args)))
	}
	where := " WHERE " + strings.Join(conditions, " AND ")

	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT count(*) FROM goals`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, filter.Limit, filter.Offset())
	query := `SELECT ` + goalsColumns + ` FROM goals` + where +
		fmt.Sprintf(` ORDER BY id DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := make([]entity.Goals, 0, min(max(filter.Limit, 0), 100))
	for rows.Next() {
		goals, err := scanGoals(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, *goals)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (r *goalsRepositorySQL) FindByID(ctx context.Context, id uint) (*entity.Goals, error) {
	const query = `SELECT ` + goalsColumns + ` FROM goals WHERE id = $1 AND ` + notDeleted

	goals, err := scanGoals(r.db.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, entity.ErrGoalsNotFound
	}
	if err != nil {
		return nil, err
	}
	return goals, nil
}

func (r *goalsRepositorySQL) Update(ctx context.Context, goals *entity.Goals) error {
	const query = `
		UPDATE goals SET
			title      = $1,
			category   = $2,
			priority   = $3,
			status     = $4,
			progress   = $5,
			updated_at = now()
		WHERE id = $6 AND ` + notDeleted + `
		RETURNING ` + goalsColumns

	row := r.db.QueryRowContext(ctx, query,
		goals.Title, goals.Category, goals.Priority, goals.Status, goals.Progress, goals.ID)

	updated, err := scanGoals(row)
	if errors.Is(err, sql.ErrNoRows) {
		return entity.ErrGoalsNotFound
	}
	if err != nil {
		return err
	}
	*goals = *updated
	return nil
}

func (r *goalsRepositorySQL) Delete(ctx context.Context, id uint) error {
	const query = `UPDATE goals SET deleted_at = now() WHERE id = $1 AND ` + notDeleted

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return entity.ErrGoalsNotFound
	}
	return nil
}
