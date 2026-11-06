package middleware

import (
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"

	"daily-task-go/pkg/response"
)

// RateLimit membatasi jumlah request per IP dalam satu jendela waktu.
//
// Dipakai di /auth/*: tanpa ini, password bisa dicoba terus-menerus tanpa
// batas. Hashing bcrypt memperlambat setiap percobaan, tetapi memperlambat
// bukan berarti menghentikan.
//
// Penyimpanannya di memori proses. Konsekuensi yang perlu disadari:
//   - kalau aplikasi dijalankan beberapa instance, hitungannya terpisah
//     per instance — batas efektifnya jadi berlipat;
//   - hitungan hilang saat aplikasi restart.
//
// Untuk kebutuhan yang lebih serius, ganti Storage-nya dengan Redis.
func RateLimit(max int, window time.Duration) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: window,

		// Pembatasan dihitung per IP.
		//
		// Catatan: di belakang reverse proxy, c.IP() mengembalikan IP proxy —
		// sehingga SEMUA pengguna berbagi satu jatah. Kalau aplikasi ini
		// nanti ditaruh di belakang nginx/cloud load balancer, nyalakan
		// EnableTrustedProxyCheck di konfigurasi fiber supaya header
		// X-Forwarded-For dibaca dengan benar.
		KeyGenerator: func(c *fiber.Ctx) string { return c.IP() },

		// Tanpa ini, limiter membalas body "Too Many Requests" apa adanya —
		// bentuknya berbeda dari seluruh response lain di API ini.
		LimitReached: func(c *fiber.Ctx) error {
			return response.Error(c, http.StatusTooManyRequests,
				"terlalu banyak percobaan, coba lagi nanti", nil)
		},

		// Request yang berhasil tetap dihitung. Kalau hanya yang gagal yang
		// dihitung, penyerang bisa menyelingi tebakan dengan login yang benar
		// di akun miliknya sendiri untuk mengulur batas.
		SkipSuccessfulRequests: false,
	})
}
