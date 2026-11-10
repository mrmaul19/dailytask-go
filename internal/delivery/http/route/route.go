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
	AuthMiddleware   fiber.Handler
	AdminMiddleware  fiber.Handler
	AuthRateLimit    fiber.Handler
	EnableDocs       bool
}

func Register(app *fiber.App, deps Deps) {
	app.Get("/health", healthCheck)

	if deps.EnableDocs {
		app.Get("/swagger/*", fiberSwagger.HandlerDefault)
	}

	v1 := app.Group("/api/v1")

	auth := v1.Group("/auth", deps.AuthRateLimit)
	auth.Post("/register", deps.AuthHandler.Register)
	auth.Post("/login", deps.AuthHandler.Login)

	tasks := v1.Group("/tasks")
	tasks.Post("/", deps.TaskHandler.Create)
	tasks.Get("/", deps.TaskHandler.List)
	tasks.Get("/:id", deps.TaskHandler.GetByID)
	tasks.Get("/:id/download", deps.TaskHandler.Download)
	tasks.Put("/:id", deps.TaskHandler.Update)
	tasks.Delete("/:id", deps.TaskHandler.Delete)

	goals := v1.Group("/goals")
	goals.Post("/", deps.GoalsHandler.Create)
	goals.Get("/", deps.GoalsHandler.List)
	goals.Get("/:id", deps.GoalsHandler.GetByID)
	goals.Put("/:id", deps.GoalsHandler.Update)
	goals.Delete("/:id", deps.GoalsHandler.Delete)

	timeEntries := v1.Group("/time-entries", deps.AuthMiddleware)
	timeEntries.Post("/start", deps.TimeEntryHandler.Start)
	timeEntries.Post("/stop", deps.TimeEntryHandler.Stop)
	timeEntries.Get("/running", deps.TimeEntryHandler.Running)
	timeEntries.Get("/", deps.TimeEntryHandler.List)
	timeEntries.Get("/:id", deps.TimeEntryHandler.GetByID)
	timeEntries.Put("/:id", deps.TimeEntryHandler.Update)
	timeEntries.Delete("/:id", deps.TimeEntryHandler.Delete)

	users := v1.Group("/users", deps.AuthMiddleware, deps.AdminMiddleware)
	users.Post("/", deps.UserHandler.Create)
	users.Get("/", deps.UserHandler.List)
	users.Get("/:id", deps.UserHandler.GetByID)
	users.Put("/:id", deps.UserHandler.Update)
	users.Delete("/:id", deps.UserHandler.Delete)
}

func healthCheck(c *fiber.Ctx) error {
	return response.Success(c, http.StatusOK, "service berjalan normal", fiber.Map{"status": "ok"})
}
