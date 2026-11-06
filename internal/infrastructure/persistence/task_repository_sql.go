// Package persistence adalah adapter: implementasi port repository memakai
// database/sql dengan SQL yang ditulis manual, tanpa ORM.
//
// Semua nilai dari luar selalu dikirim sebagai parameter ($1, $2, ...),
// tidak pernah dirangkai ke dalam string query, supaya bebas SQL injection.
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

// taskColumns dipakai bersama oleh seluruh SELECT/RETURNING di file ini
// supaya urutan kolomnya selalu sama dengan urutan Scan di scanTask.
//
// COALESCE membuat kolom teks tetap aman di-Scan ke string biasa, termasuk
// pada database lama yang kolomnya masih nullable.
const taskColumns = `id, no_task, judul,
	COALESCE(description, '') AS description,
	COALESCE(file_name, '')   AS file_name,
	COALESCE(file_path, '')   AS file_path,
	created_at, updated_at`

// Baris yang sudah di-soft-delete tidak pernah ikut terbaca.
const notDeleted = `deleted_at IS NULL`

type taskRepositorySQL struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) repository.TaskRepository {
	return &taskRepositorySQL{db: db}
}

// rowScanner menyatukan *sql.Row dan *sql.Rows supaya scanTask bisa dipakai
// baik untuk query satu baris maupun banyak baris.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanTask(s rowScanner) (*entity.Task, error) {
	var task entity.Task
	err := s.Scan(
		&task.ID,
		&task.NoTask,
		&task.Judul,
		&task.Description,
		&task.FileName,
		&task.FilePath,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *taskRepositorySQL) Create(ctx context.Context, task *entity.Task) error {
	const query = `
		INSERT INTO tasks (no_task, judul, description, file_name, file_path, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, now(), now())
		RETURNING ` + taskColumns

	row := r.db.QueryRowContext(ctx, query,
		task.NoTask, task.Judul, task.Description, task.FileName, task.FilePath)

	created, err := scanTask(row)
	if err != nil {
		return err
	}
	*task = *created // balikkan ID & timestamp hasil insert
	return nil
}

func (r *taskRepositorySQL) FindAll(ctx context.Context, filter repository.TaskFilter) ([]entity.Task, int64, error) {
	conditions := []string{notDeleted}
	args := []any{}

	if search := strings.TrimSpace(filter.Search); search != "" {
		args = append(args, "%"+search+"%")
		// Satu keyword dipakai untuk tiga kolom, jadi nomor parameternya sama.
		conditions = append(conditions, fmt.Sprintf(
			"(no_task ILIKE $%d OR judul ILIKE $%d OR description ILIKE $%d)",
			len(args), len(args), len(args)))
	}
	where := " WHERE " + strings.Join(conditions, " AND ")

	// Hitung total dulu dengan kondisi yang sama, untuk metadata pagination.
	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT count(*) FROM tasks`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, filter.Limit, filter.Offset())
	query := `SELECT ` + taskColumns + ` FROM tasks` + where +
		fmt.Sprintf(` ORDER BY id DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	// Kapasitas dibatasi supaya Limit yang tidak wajar tidak langsung
	// mengalokasikan slice raksasa.
	tasks := make([]entity.Task, 0, min(max(filter.Limit, 0), 100))
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, *task)
	}
	// Wajib: error saat iterasi (koneksi putus, dsb) hanya muncul di sini.
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

func (r *taskRepositorySQL) FindByID(ctx context.Context, id uint) (*entity.Task, error) {
	const query = `SELECT ` + taskColumns + ` FROM tasks WHERE id = $1 AND ` + notDeleted

	task, err := scanTask(r.db.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, entity.ErrTaskNotFound
	}
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (r *taskRepositorySQL) FindByNoTask(ctx context.Context, noTask string) (*entity.Task, error) {
	const query = `SELECT ` + taskColumns + ` FROM tasks WHERE no_task = $1 AND ` + notDeleted

	task, err := scanTask(r.db.QueryRowContext(ctx, query, noTask))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, entity.ErrTaskNotFound
	}
	if err != nil {
		return nil, err
	}
	return task, nil
}

// Update menimpa kolom yang boleh diubah, lalu mengembalikan baris hasilnya
// lewat RETURNING -- jadi UpdatedAt ikut ter-refresh tanpa query tambahan.
func (r *taskRepositorySQL) Update(ctx context.Context, task *entity.Task) error {
	const query = `
		UPDATE tasks SET
			no_task     = $1,
			judul       = $2,
			description = $3,
			file_name   = $4,
			file_path   = $5,
			updated_at  = now()
		WHERE id = $6 AND ` + notDeleted + `
		RETURNING ` + taskColumns

	row := r.db.QueryRowContext(ctx, query,
		task.NoTask, task.Judul, task.Description, task.FileName, task.FilePath, task.ID)

	updated, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		// Tidak ada baris yang cocok: id-nya tidak ada atau sudah dihapus.
		return entity.ErrTaskNotFound
	}
	if err != nil {
		return err
	}
	*task = *updated
	return nil
}

// Delete adalah soft delete: barisnya hanya ditandai, bukan dibuang.
func (r *taskRepositorySQL) Delete(ctx context.Context, id uint) error {
	const query = `UPDATE tasks SET deleted_at = now() WHERE id = $1 AND ` + notDeleted

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return entity.ErrTaskNotFound
	}
	return nil
}
