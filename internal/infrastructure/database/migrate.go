package database

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"sort"

	"daily-task-go/migrations"
)

// schemaMigrationsDDL mencatat file migrasi mana yang sudah dijalankan,
// supaya migrasi yang sama tidak diterapkan dua kali.
const schemaMigrationsDDL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version    TEXT        PRIMARY KEY,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`

// Migrate menjalankan file .sql di paket migrations yang belum pernah
// dijalankan, urut berdasarkan nama file.
//
// Setiap file dieksekusi dalam satu transaksi bersama pencatatan versinya,
// jadi migrasi yang gagal di tengah tidak meninggalkan skema setengah jadi
// maupun catatan versi yang bohong.
func Migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, schemaMigrationsDDL); err != nil {
		return fmt.Errorf("gagal menyiapkan tabel schema_migrations: %w", err)
	}

	applied, err := appliedVersions(ctx, db)
	if err != nil {
		return err
	}

	files, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil {
		return fmt.Errorf("gagal membaca daftar migrasi: %w", err)
	}
	sort.Strings(files)

	for _, name := range files {
		if applied[name] {
			continue
		}

		body, err := migrations.FS.ReadFile(name)
		if err != nil {
			return fmt.Errorf("gagal membaca migrasi %s: %w", name, err)
		}
		if err := applyMigration(ctx, db, name, string(body)); err != nil {
			return err
		}
		log.Printf("migrasi %s diterapkan", name)
	}

	return nil
}

func appliedVersions(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca riwayat migrasi: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("gagal membaca riwayat migrasi: %w", err)
		}
		applied[version] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("gagal membaca riwayat migrasi: %w", err)
	}

	return applied, nil
}

func applyMigration(ctx context.Context, db *sql.DB, name, body string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("gagal membuka transaksi untuk migrasi %s: %w", name, err)
	}
	// Tidak ada efek apa pun kalau Commit di bawah sudah berhasil.
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, body); err != nil {
		return fmt.Errorf("migrasi %s gagal: %w", name, err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
		return fmt.Errorf("gagal mencatat migrasi %s: %w", name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("gagal commit migrasi %s: %w", name, err)
	}

	return nil
}
