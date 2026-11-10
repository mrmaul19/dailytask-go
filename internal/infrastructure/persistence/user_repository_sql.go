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

const userColumns = `id, email, password, name, address, job, status, role, created_at, updated_at`

type userRepositorySQL struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) repository.UserRepository {
	return &userRepositorySQL{db: db}
}

func scanUser(s rowScanner) (*entity.User, error) {
	var user entity.User
	err := s.Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.Name,
		&user.Address,
		&user.Job,
		&user.Status,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepositorySQL) Create(ctx context.Context, user *entity.User) error {
	const query = `
		INSERT INTO users (email, password, name, address, job, status, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now(), now())
		RETURNING ` + userColumns

	row := r.db.QueryRowContext(ctx, query,
		user.Email, user.Password, user.Name, user.Address, user.Job, user.Status, user.Role)

	created, err := scanUser(row)
	if err != nil {
		return err
	}
	*user = *created
	return nil
}

func (r *userRepositorySQL) FindAll(ctx context.Context, filter repository.UserFilter) ([]entity.User, int64, error) {
	conditions := []string{notDeleted}
	args := []any{}

	if search := strings.TrimSpace(filter.Search); search != "" {
		args = append(args, "%"+search+"%")
		conditions = append(conditions, fmt.Sprintf(
			"(name ILIKE $%d OR job ILIKE $%d)", len(args), len(args)))
	}
	where := " WHERE " + strings.Join(conditions, " AND ")

	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT count(*) FROM users`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, filter.Limit, filter.Offset())
	query := `SELECT ` + userColumns + ` FROM users` + where +
		fmt.Sprintf(` ORDER BY id DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := make([]entity.User, 0, min(max(filter.Limit, 0), 100))
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, *user)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (r *userRepositorySQL) FindByID(ctx context.Context, id uint) (*entity.User, error) {
	const query = `SELECT ` + userColumns + ` FROM users WHERE id = $1 AND ` + notDeleted

	user, err := scanUser(r.db.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, entity.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepositorySQL) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	const query = `SELECT ` + userColumns + ` FROM users WHERE lower(email) = lower($1) AND ` + notDeleted

	user, err := scanUser(r.db.QueryRowContext(ctx, query, email))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, entity.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepositorySQL) Update(ctx context.Context, user *entity.User) error {
	const query = `
		UPDATE users SET
			email      = $1,
			password   = $2,
			name       = $3,
			address    = $4,
			job        = $5,
			status     = $6,
			role       = $7,
			updated_at = now()
		WHERE id = $8 AND ` + notDeleted + `
		RETURNING ` + userColumns

	row := r.db.QueryRowContext(ctx, query,
		user.Email, user.Password, user.Name, user.Address, user.Job, user.Status, user.Role, user.ID)

	updated, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return entity.ErrUserNotFound
	}
	if err != nil {
		return err
	}
	*user = *updated
	return nil
}

func (r *userRepositorySQL) Delete(ctx context.Context, id uint) error {
	const query = `UPDATE users SET deleted_at = now() WHERE id = $1 AND ` + notDeleted

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return entity.ErrUserNotFound
	}
	return nil
}
