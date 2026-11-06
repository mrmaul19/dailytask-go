// Package database bertanggung jawab membuka koneksi ke Postgres
// dan menjalankan migrasi skema.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	// Driver "pgx" untuk database/sql. Diimpor hanya demi efek samping
	// registrasi driver-nya; querynya tetap ditulis manual sebagai SQL.
	_ "github.com/jackc/pgx/v5/stdlib"

	"daily-task-go/config"
)

// New membuka koneksi Postgres dan memastikan koneksinya benar-benar hidup
// lewat ping, bukan hanya "berhasil dibuat". sql.Open sendiri bersifat lazy:
// ia tidak menyentuh server sampai query pertama dijalankan.
func New(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.Database.DSN())
	if err != nil {
		return nil, fmt.Errorf("gagal menyiapkan koneksi ke %s: %w", cfg.Database.MaskedDSN(), err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(10 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("database %s tidak merespons: %w", cfg.Database.MaskedDSN(), err)
	}

	return db, nil
}
