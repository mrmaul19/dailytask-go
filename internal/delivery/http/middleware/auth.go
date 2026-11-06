package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"daily-task-go/internal/domain/entity"
	"daily-task-go/internal/domain/security"
	"daily-task-go/pkg/apperror"
)

// Kunci untuk mengambil identitas pemanggil di handler, lewat c.Locals.
// Dideklarasikan sebagai konstanta supaya tidak ada salah ketik string
// yang baru ketahuan saat runtime.
const (
	ContextUserID    = "userID"
	ContextUserEmail = "userEmail"
	ContextUserRole  = "userRole"
)

const bearerPrefix = "bearer "

// Auth menolak request yang tidak membawa token akses yang sah.
//
// Middleware ini hanya menjawab "siapa pemanggilnya" (authentication),
// bukan "dia boleh apa" (authorization) — pembatasan hak akses adalah
// urusan terpisah.
func Auth(tokens security.TokenIssuer) fiber.Handler {
	return func(c *fiber.Ctx) error {
		raw := strings.TrimSpace(c.Get(fiber.HeaderAuthorization))
		if raw == "" {
			return apperror.Unauthorized("token tidak dikirim")
		}

		// Skema dibandingkan tanpa membedakan huruf besar/kecil, sesuai RFC 7235.
		if len(raw) < len(bearerPrefix) || !strings.EqualFold(raw[:len(bearerPrefix)], bearerPrefix) {
			return apperror.Unauthorized("format Authorization harus: Bearer {token}")
		}

		value := strings.TrimSpace(raw[len(bearerPrefix):])
		if value == "" {
			return apperror.Unauthorized("token tidak dikirim")
		}

		claims, err := tokens.Parse(value)
		if err != nil {
			// Alasan detail (kedaluwarsa / tanda tangan salah / rusak) sengaja
			// tidak dibedakan di respons.
			return apperror.Unauthorized("token tidak valid atau sudah kedaluwarsa")
		}

		c.Locals(ContextUserID, claims.UserID)
		c.Locals(ContextUserEmail, claims.Email)
		c.Locals(ContextUserRole, claims.Role)

		return c.Next()
	}
}

// RequireAdmin hanya meneruskan request dari user ber-role admin.
//
// WAJIB dipasang SETELAH Auth, karena ia membaca role yang ditaruh Auth di
// c.Locals. Kalau dipasang sendirian, role selalu kosong dan semua request
// ditolak 403 — gagal ke arah yang aman, tapi tetap salah pasang.
//
// Token yang terbit sebelum kolom role ada tidak punya klaim role, sehingga
// nilainya kosong dan tetap ditolak di sini. Itu perilaku yang diinginkan:
// token lama tidak otomatis menjadi admin.
func RequireAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, _ := c.Locals(ContextUserRole).(string)
		if role != entity.RoleAdmin {
			// 403, bukan 401: identitasnya sudah diketahui dan tokennya sah,
			// hanya haknya yang kurang. Login ulang tidak akan menolong.
			return apperror.Forbidden("butuh hak admin")
		}
		return c.Next()
	}
}

// UserRoleFromContext mengambil role pemanggil yang sudah diverifikasi Auth.
func UserRoleFromContext(c *fiber.Ctx) (string, bool) {
	role, ok := c.Locals(ContextUserRole).(string)
	return role, ok
}

// UserIDFromContext mengambil id user yang sudah diverifikasi middleware Auth.
// ok bernilai false kalau dipanggil dari handler yang tidak dilindungi Auth.
func UserIDFromContext(c *fiber.Ctx) (uint, bool) {
	id, ok := c.Locals(ContextUserID).(uint)
	return id, ok
}
