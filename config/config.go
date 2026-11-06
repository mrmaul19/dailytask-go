// Package config memuat seluruh konfigurasi aplikasi dari environment variable.
// Tidak ada paket lain yang boleh memanggil os.Getenv secara langsung.
package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type App struct {
	Name    string
	Port    string
	Env     string
	BaseURL string
	// EnableDocs menyalakan UI Swagger di /swagger/index.html.
	// Matikan (ENABLE_DOCS=false) kalau tidak ingin spec API terbuka di publik.
	EnableDocs bool
}

type Database struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
	TimeZone string
}

type Storage struct {
	UploadDir   string
	MaxUploadMB int64
	AllowedExt  []string
}

type Config struct {
	App      App
	Database Database
	Storage  Storage
}

// Load membaca file .env (kalau ada) lalu environment variable.
func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		App: App{
			Name:       env("APP_NAME", "daily-task"),
			Port:       env("APP_PORT", "3000"),
			Env:        env("APP_ENV", "development"),
			BaseURL:    strings.TrimSuffix(env("APP_BASE_URL", "http://localhost:3000"), "/"),
			EnableDocs: envBool("ENABLE_DOCS", true),
		},
		Database: Database{
			Host:     env("DB_HOST", "localhost"),
			Port:     env("DB_PORT", "5432"),
			User:     env("DB_USER", "postgres"),
			Password: env("DB_PASSWORD", "postgres"),
			Name:     env("DB_NAME", "daily_task"),
			SSLMode:  env("DB_SSLMODE", "disable"),
			TimeZone: env("DB_TIMEZONE", "Asia/Jakarta"),
		},
		Storage: Storage{
			UploadDir:   env("UPLOAD_DIR", "uploads"),
			MaxUploadMB: envInt("MAX_UPLOAD_MB", 5),
			AllowedExt:  splitAndTrim(env("ALLOWED_EXT", ".pdf,.doc,.docx,.xls,.xlsx,.png,.jpg,.jpeg,.zip")),
		},
	}
}

// DSN merakit connection string Postgres dalam bentuk URL.
// Dirakit lewat net/url supaya karakter khusus pada password (@, /, :, spasi)
// ikut di-escape dan tidak merusak connection string.
func (d Database) DSN() string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(d.User, d.Password),
		Host:   net.JoinHostPort(d.Host, d.Port),
		Path:   "/" + d.Name,
	}

	q := url.Values{}
	q.Set("sslmode", d.SSLMode)
	if d.TimeZone != "" {
		// Dikirim sebagai runtime parameter ke server Postgres, jadi now()
		// dan kolom timestamptz memakai zona waktu ini.
		q.Set("timezone", d.TimeZone)
	}
	u.RawQuery = q.Encode()

	return u.String()
}

// MaskedDSN aman untuk ditulis ke log: password disembunyikan.
func (d Database) MaskedDSN() string {
	return fmt.Sprintf("postgres://%s:***@%s:%s/%s?sslmode=%s",
		d.User, d.Host, d.Port, d.Name, d.SSLMode)
}

func (s Storage) MaxUploadBytes() int64 { return s.MaxUploadMB * 1024 * 1024 }

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			return parsed
		}
	}
	return fallback
}

func envInt(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			return parsed
		}
	}
	return fallback
}

func splitAndTrim(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.ToLower(strings.TrimSpace(p)); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
