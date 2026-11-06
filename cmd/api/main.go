// Composition root: satu-satunya tempat semua dependency dirakit.
// Alur ketergantungan selalu dari luar ke dalam:
// main -> delivery -> usecase -> domain, dengan infrastructure
// mengimplementasikan port milik domain.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"daily-task-go/config"
	swaggerdocs "daily-task-go/docs"
	"daily-task-go/internal/delivery/http/handler"
	"daily-task-go/internal/delivery/http/middleware"
	"daily-task-go/internal/delivery/http/route"
	"daily-task-go/internal/infrastructure/database"
	"daily-task-go/internal/infrastructure/persistence"
	infrastorage "daily-task-go/internal/infrastructure/storage"
	"daily-task-go/internal/usecase"
	"daily-task-go/pkg/validator"
)

// @title			Daily Task API
// @version		1.0
// @description	REST API untuk mencatat daily task (dengan lampiran file), goals, dan users.
// @description	Dibangun dengan Go Fiber + database/sql (driver pgx) + PostgreSQL, tanpa ORM,
// @description	mengikuti Clean Architecture. Spec ini digenerate dari anotasi di handler
// @description	oleh `swag init`, jadi ia selalu ikut kode — bukan dokumen terpisah yang gampang basi.
//
// @BasePath	/
// @schemes	http https
//
// @tag.name			Health
// @tag.description	Liveness check
// @tag.name			Tasks
// @tag.description	Daily task, satu-satunya resource yang punya lampiran file
// @tag.name			Goals
// @tag.description	Target/goals, JSON murni
// @tag.name			Users
// @tag.description	Data user
func main() {
	cfg := config.Load()

	// Host pada spec diambil dari config, bukan di-hardcode lewat anotasi
	// @host, supaya tombol "Try it out" tetap benar saat port atau domain ganti.
	swaggerdocs.SwaggerInfo.Host = strings.TrimPrefix(
		strings.TrimPrefix(cfg.App.BaseURL, "http://"), "https://")

	// --- infrastructure ---
	db, err := database.New(cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	migrateCtx, cancelMigrate := context.WithTimeout(context.Background(), 30*time.Second)
	if err := database.Migrate(migrateCtx, db); err != nil {
		cancelMigrate()
		log.Fatalf("migrasi: %v", err)
	}
	cancelMigrate()

	fileStorage, err := infrastorage.NewLocalStorage(
		cfg.Storage.UploadDir,
		cfg.Storage.MaxUploadBytes(),
		cfg.Storage.AllowedExt,
	)
	if err != nil {
		log.Fatalf("storage: %v", err)
	}

	// --- wiring layer ---
	validate := validator.New()

	taskRepo := persistence.NewTaskRepository(db)
	taskUsecase := usecase.NewTaskUsecase(taskRepo, fileStorage)
	taskHandler := handler.NewTaskHandler(taskUsecase, validate, cfg.App.BaseURL)

	goalsRepo := persistence.NewGoalsRepository(db)
	goalsUsecase := usecase.NewGoalsUsecase(goalsRepo)
	goalsHandler := handler.NewGoalsHandler(goalsUsecase, validate)

	userRepo := persistence.NewUserRepository(db)
	userUsecase := usecase.NewUserUsecase(userRepo)
	userHandler := handler.NewUserHandler(userUsecase, validate)

	// --- http server ---
	app := fiber.New(fiber.Config{
		AppName:      cfg.App.Name,
		ErrorHandler: middleware.ErrorHandler(cfg.App.Env == "development"),
		BodyLimit:    int(cfg.Storage.MaxUploadBytes()) + 1024*1024, // sisa untuk field teks
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	})

	app.Use(recover.New())
	app.Use(logger.New(logger.Config{Format: "[${time}] ${status} ${method} ${path} (${latency})\n"}))
	app.Use(cors.New())

	route.Register(app, route.Deps{
		TaskHandler:  taskHandler,
		GoalsHandler: goalsHandler,
		UserHandler:  userHandler,
		EnableDocs:   cfg.App.EnableDocs,
	})

	// --- graceful shutdown ---
	go func() {
		addr := ":" + cfg.App.Port
		log.Printf("%s berjalan di http://localhost%s (env=%s, db=%s)",
			cfg.App.Name, addr, cfg.App.Env, cfg.Database.MaskedDSN())
		if cfg.App.EnableDocs {
			log.Printf("dokumentasi swagger: %s/swagger/index.html", cfg.App.BaseURL)
		}
		if err := app.Listen(addr); err != nil {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("menghentikan server...")
	if err := app.ShutdownWithTimeout(10 * time.Second); err != nil {
		log.Printf("shutdown: %v", err)
	}
	log.Println("server berhenti dengan rapi")
}
