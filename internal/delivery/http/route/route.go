// Package route mendaftarkan seluruh endpoint HTTP di satu tempat.
package route

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	fiberSwagger "github.com/gofiber/swagger"

	"daily-task-go/internal/delivery/http/handler"
	"daily-task-go/pkg/response"
)

type Deps struct {
	TaskHandler      *handler.TaskHandler
	GoalsHandler     *handler.GoalsHandler
	UserHandler      *handler.UserHandler
	AuthHandler      *handler.AuthHandler
	TimeEntryHandler *handler.TimeEntryHandler

	// AuthMiddleware sudah jadi fiber.Handler, bukan port TokenIssuer.
	// Dengan begitu paket route tidak perlu tahu-menahu soal JWT — ia hanya
	// memasang handler, persis seperti perlakuannya terhadap handler lain.
	AuthMiddleware fiber.Handler
	// AdminMiddleware WAJIB dipasang setelah AuthMiddleware; ia membaca role
	// yang ditaruh AuthMiddleware di c.Locals.
	AdminMiddleware fiber.Handler
	// AuthRateLimit membatasi percobaan di /auth/*, supaya password tidak
	// bisa ditebak terus-menerus.
	AuthRateLimit fiber.Handler

	// EnableDocs menentukan apakah UI Swagger di /swagger/* didaftarkan.
	// Diisi dari config (ENABLE_DOCS) supaya bisa dimatikan di production.
	EnableDocs bool
}

func Register(app *fiber.App, deps Deps) {
	app.Get("/health", healthCheck)

	// Dokumentasi OpenAPI. Spec-nya digenerate oleh `swag init` ke folder docs/
	// dan diregistrasikan lewat import paket docs di cmd/api/main.go.
	if deps.EnableDocs {
		app.Get("/swagger/*", fiberSwagger.HandlerDefault)
	}

	v1 := app.Group("/api/v1")

	// Grup auth WAJIB tanpa AuthMiddleware — kalau ikut dilindungi, tidak ada
	// satu pun cara untuk memperoleh token pertama kali.
	//
	// Justru karena terbuka, grup ini yang paling butuh rate limit: ia menerima
	// password dan membalas apakah password itu benar.
	auth := v1.Group("/auth", deps.AuthRateLimit)
	auth.Post("/register", deps.AuthHandler.Register)
	auth.Post("/login", deps.AuthHandler.Login)

	tasks := v1.Group("/tasks")
	tasks.Post("/", deps.TaskHandler.Create)
	tasks.Get("/", deps.TaskHandler.List)
	tasks.Get("/:id", deps.TaskHandler.GetByID)
	// File diakses lewat endpoint ini (bukan folder static) supaya nama file
	// di disk tidak bisa ditebak dan akses selalu lewat data task di DB.
	tasks.Get("/:id/download", deps.TaskHandler.Download)
	tasks.Put("/:id", deps.TaskHandler.Update)
	tasks.Delete("/:id", deps.TaskHandler.Delete)

	goals := v1.Group("/goals")
	goals.Post("/", deps.GoalsHandler.Create)
	goals.Get("/", deps.GoalsHandler.List)
	goals.Get("/:id", deps.GoalsHandler.GetByID)
	goals.Put("/:id", deps.GoalsHandler.Update)
	goals.Delete("/:id", deps.GoalsHandler.Delete)

	// Catatan waktu. AuthMiddleware WAJIB di sini: setiap endpoint di grup ini
	// bersandar pada identitas pemanggil (user_id diambil dari token, bukan
	// dari body/query). Tanpa middleware-nya, handler menolak dengan 401 —
	// gagal ke arah yang aman, tapi tetap salah pasang.
	//
	// Berbeda dari /tasks dan /goals yang saat ini terbuka, grup ini memang
	// tidak bisa terbuka: "catatan waktu saya" tidak punya arti tanpa "saya".
	timeEntries := v1.Group("/time-entries", deps.AuthMiddleware)

	// URUTAN PENTING: /start, /stop, dan /running didaftarkan SEBELUM /:id.
	// Fiber mencocokkan rute sesuai urutan pendaftaran, jadi kalau /:id lebih
	// dulu, "running" akan dibaca sebagai nilai :id lalu ditolak 400 oleh
	// parseID — kegagalan yang membingungkan karena rutenya sendiri ada.
	timeEntries.Post("/start", deps.TimeEntryHandler.Start)
	timeEntries.Post("/stop", deps.TimeEntryHandler.Stop)
	timeEntries.Get("/running", deps.TimeEntryHandler.Running)

	timeEntries.Get("/", deps.TimeEntryHandler.List)
	timeEntries.Get("/:id", deps.TimeEntryHandler.GetByID)
	timeEntries.Put("/:id", deps.TimeEntryHandler.Update)
	timeEntries.Delete("/:id", deps.TimeEntryHandler.Delete)

	// Pengelolaan user butuh token DAN role admin. Urutannya penting:
	// AuthMiddleware mengisi role, AdminMiddleware membacanya.
	//
	// Pendaftaran mandiri lewat /auth/register, yang memaksa status dan role
	// default serta tidak menerima field sebebas di sini.
	users := v1.Group("/users", deps.AuthMiddleware, deps.AdminMiddleware)
	users.Post("/", deps.UserHandler.Create)
	users.Get("/", deps.UserHandler.List)
	users.Get("/:id", deps.UserHandler.GetByID)
	users.Put("/:id", deps.UserHandler.Update)
	users.Delete("/:id", deps.UserHandler.Delete)
}

// healthCheck dipisah jadi fungsi bernama (bukan closure) supaya bisa
// dianotasi untuk Swagger — swag hanya membaca komentar di atas deklarasi func.
//
//	@Summary		Cek kesehatan service
//	@Description	Dipakai untuk liveness check. Tidak menyentuh database.
//	@Tags			Health
//	@Produce		json
//	@Success		200	{object}	dto.HealthEnvelope
//	@Router			/health [get]
func healthCheck(c *fiber.Ctx) error {
	return response.Success(c, http.StatusOK, "service berjalan normal", fiber.Map{"status": "ok"})
}
