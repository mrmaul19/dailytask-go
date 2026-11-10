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
	infrasecurity "daily-task-go/internal/infrastructure/security"
	infrastorage "daily-task-go/internal/infrastructure/storage"
	"daily-task-go/internal/usecase"
	"daily-task-go/pkg/validator"
)

func main() {
	cfg := config.Load()

	if cfg.Auth.JWTSecret == config.DefaultJWTSecret {
		if cfg.App.Env == "production" {
			log.Fatal("JWT_SECRET masih memakai nilai bawaan — wajib diganti di production")
		}
		log.Println("PERINGATAN: JWT_SECRET memakai nilai bawaan, hanya aman untuk development")
	}

	swaggerdocs.SwaggerInfo.Host = strings.TrimPrefix(
		strings.TrimPrefix(cfg.App.BaseURL, "http://"), "https://")

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

	validate := validator.New()
	taskRepo := persistence.NewTaskRepository(db)
	taskUsecase := usecase.NewTaskUsecase(taskRepo, fileStorage)
	taskHandler := handler.NewTaskHandler(taskUsecase, validate, cfg.App.BaseURL)
	goalsRepo := persistence.NewGoalsRepository(db)
	goalsUsecase := usecase.NewGoalsUsecase(goalsRepo)
	goalsHandler := handler.NewGoalsHandler(goalsUsecase, validate)
	timeEntryRepo := persistence.NewTimeEntryRepository(db)
	timeEntryUsecase := usecase.NewTimeEntryUsecase(timeEntryRepo, taskRepo)
	timeEntryHandler := handler.NewTimeEntryHandler(timeEntryUsecase, validate)
	passwordHasher := infrasecurity.NewBcryptHasher(0)
	tokenIssuer := infrasecurity.NewJWTIssuer(cfg.Auth.JWTSecret, cfg.Auth.JWTTTL)
	userRepo := persistence.NewUserRepository(db)
	userUsecase := usecase.NewUserUsecase(userRepo, passwordHasher)
	userHandler := handler.NewUserHandler(userUsecase, validate)

	authUsecase := usecase.NewAuthUsecase(userUsecase, userRepo, passwordHasher, tokenIssuer)
	authHandler := handler.NewAuthHandler(authUsecase, validate)

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
		TaskHandler:      taskHandler,
		GoalsHandler:     goalsHandler,
		UserHandler:      userHandler,
		AuthHandler:      authHandler,
		TimeEntryHandler: timeEntryHandler,
		AuthMiddleware:   middleware.Auth(tokenIssuer),
		AdminMiddleware:  middleware.RequireAdmin(),
		AuthRateLimit:    middleware.RateLimit(cfg.Auth.RateLimitMax, cfg.Auth.RateLimitWindow),
		EnableDocs:       cfg.App.EnableDocs,
	})

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
