package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"daily-task-go/internal/domain/entity"
	"daily-task-go/internal/domain/repository"
)

const timeEntryColumns = `id, user_id, task_id, started_at, ended_at,
	COALESCE(note, '') AS note,
	created_at, updated_at`

const (
	constraintSatuTimerBerjalan = "time_entries_one_running_per_user"
	constraintRentangWaktu      = "time_entries_interval_check"
)

type timeEntryRepositorySQL struct {
	db *sql.DB
}

func NewTimeEntryRepository(db *sql.DB) repository.TimeEntryRepository {
	return &timeEntryRepositorySQL{db: db}
}

func terjemahkanErrorTimeEntry(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch {
	case pgErr.Code == "23505" && pgErr.ConstraintName == constraintSatuTimerBerjalan:
		return entity.ErrTimerSedangBerjalan
	case pgErr.Code == "23514" && pgErr.ConstraintName == constraintRentangWaktu:
		return entity.ErrRentangWaktuTidakValid
	default:
		return err
	}
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func scanTimeEntry(s rowScanner) (*entity.TimeEntry, error) {
	var (
		entry   entity.TimeEntry
		endedAt sql.NullTime
	)
	err := s.Scan(
		&entry.ID,
		&entry.UserID,
		&entry.TaskID,
		&entry.StartedAt,
		&endedAt,
		&entry.Note,
		&entry.CreatedAt,
		&entry.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if endedAt.Valid {
		selesai := endedAt.Time
		entry.EndedAt = &selesai
	}

	return &entry, nil
}

func (r *timeEntryRepositorySQL) Create(ctx context.Context, entry *entity.TimeEntry) error {
	const query = `
		INSERT INTO time_entries (user_id, task_id, started_at, ended_at, note, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, now(), now())
		RETURNING ` + timeEntryColumns

	row := r.db.QueryRowContext(ctx, query,
		entry.UserID, entry.TaskID, entry.StartedAt, nullTime(entry.EndedAt), entry.Note)

	created, err := scanTimeEntry(row)
	if err != nil {
		return terjemahkanErrorTimeEntry(err)
	}
	*entry = *created
	return nil
}

func (r *timeEntryRepositorySQL) FindAll(ctx context.Context, filter repository.TimeEntryFilter) ([]entity.TimeEntry, int64, error) {
	conditions := []string{notDeleted}
	args := []any{}

	if filter.UserID != 0 {
		args = append(args, filter.UserID)
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", len(args)))
	}
	if filter.TaskID != 0 {
		args = append(args, filter.TaskID)
		conditions = append(conditions, fmt.Sprintf("task_id = $%d", len(args)))
	}

	if filter.From != nil {
		args = append(args, *filter.From)
		conditions = append(conditions, fmt.Sprintf("started_at >= $%d", len(args)))
	}
	if filter.To != nil {
		args = append(args, *filter.To)
		conditions = append(conditions, fmt.Sprintf("started_at < $%d", len(args)))
	}

	where := " WHERE " + strings.Join(conditions, " AND ")

	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT count(*) FROM time_entries`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, filter.Limit, filter.Offset())

	query := `SELECT ` + timeEntryColumns + ` FROM time_entries` + where +
		fmt.Sprintf(` ORDER BY started_at DESC, id DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	entries := make([]entity.TimeEntry, 0, min(max(filter.Limit, 0), 100))
	for rows.Next() {
		entry, err := scanTimeEntry(rows)
		if err != nil {
			return nil, 0, err
		}
		entries = append(entries, *entry)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return entries, total, nil
}

func (r *timeEntryRepositorySQL) FindByID(ctx context.Context, id uint) (*entity.TimeEntry, error) {
	const query = `SELECT ` + timeEntryColumns + ` FROM time_entries WHERE id = $1 AND ` + notDeleted

	entry, err := scanTimeEntry(r.db.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, entity.ErrTimeEntryNotFound
	}
	if err != nil {
		return nil, err
	}
	return entry, nil
}

func (r *timeEntryRepositorySQL) FindRunningByUserID(ctx context.Context, userID uint) (*entity.TimeEntry, error) {
	const query = `SELECT ` + timeEntryColumns + ` FROM time_entries
		WHERE user_id = $1 AND ended_at IS NULL AND ` + notDeleted

	entry, err := scanTimeEntry(r.db.QueryRowContext(ctx, query, userID))
	if errors.Is(err, sql.ErrNoRows) {
		// Tidak ada timer berjalan adalah keadaan normal, bukan kegagalan.
		return nil, entity.ErrTimeEntryNotFound
	}
	if err != nil {
		return nil, err
	}
	return entry, nil
}

func (r *timeEntryRepositorySQL) Update(ctx context.Context, entry *entity.TimeEntry) error {
	const query = `
		UPDATE time_entries SET
			task_id    = $1,
			started_at = $2,
			ended_at   = $3,
			note       = $4,
			updated_at = now()
		WHERE id = $5 AND ` + notDeleted + `
		RETURNING ` + timeEntryColumns

	row := r.db.QueryRowContext(ctx, query,
		entry.TaskID, entry.StartedAt, nullTime(entry.EndedAt), entry.Note, entry.ID)

	updated, err := scanTimeEntry(row)
	if errors.Is(err, sql.ErrNoRows) {
		return entity.ErrTimeEntryNotFound
	}
	if err != nil {
		return terjemahkanErrorTimeEntry(err)
	}
	*entry = *updated
	return nil
}

func (r *timeEntryRepositorySQL) Delete(ctx context.Context, id uint) error {
	const query = `UPDATE time_entries SET deleted_at = now() WHERE id = $1 AND ` + notDeleted

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return entity.ErrTimeEntryNotFound
	}
	return nil
}
