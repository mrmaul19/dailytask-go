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
	TaskHandler  *handler.TaskHandler
	GoalsHandler *handler.GoalsHandler
	UserHandler  *handler.UserHandler

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

	users := v1.Group("/users")
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
