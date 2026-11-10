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
	infrasecurity "daily-task-go/internal/infrastructure/security"
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
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				Token dari /api/v1/auth/login. Isi dengan: Bearer {token}
//
// @tag.name			Auth
// @tag.description	Register & login — satu-satunya grup yang bisa diakses tanpa token
// @tag.name			Health
// @tag.description	Liveness check
// @tag.name			Tasks
// @tag.description	Daily task, satu-satunya resource yang punya lampiran file
// @tag.name			Goals
// @tag.description	Target/goals, JSON murni
// @tag.name			Time Tracking
// @tag.description	Pencatatan waktu pengerjaan task — satu timer berjalan per user
// @tag.name			Users
// @tag.description	Data user
func main() {
	cfg := config.Load()

	// Secret bawaan hanya boleh dipakai saat development. Diperiksa di sini,
	// saat start — bukan saat login pertama dicoba, yang bisa jadi baru
	// ketahuan setelah aplikasi sudah lama berjalan di production.
	if cfg.Auth.JWTSecret == config.DefaultJWTSecret {
		if cfg.App.Env == "production" {
			log.Fatal("JWT_SECRET masih memakai nilai bawaan — wajib diganti di production")
		}
		log.Println("PERINGATAN: JWT_SECRET memakai nilai bawaan, hanya aman untuk development")
	}

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

	// Time tracking ikut memakai taskRepo: Start memastikan task-nya ada
	// sebelum timer dimulai, supaya task_id ngawur dijawab 404 dan bukan
	// 500 dari pelanggaran foreign key.
	timeEntryRepo := persistence.NewTimeEntryRepository(db)
	timeEntryUsecase := usecase.NewTimeEntryUsecase(timeEntryRepo, taskRepo)
	timeEntryHandler := handler.NewTimeEntryHandler(timeEntryUsecase, validate)

	// cost 0 = pakai bcrypt.DefaultCost.
	passwordHasher := infrasecurity.NewBcryptHasher(0)
	tokenIssuer := infrasecurity.NewJWTIssuer(cfg.Auth.JWTSecret, cfg.Auth.JWTTTL)

	userRepo := persistence.NewUserRepository(db)
	userUsecase := usecase.NewUserUsecase(userRepo, passwordHasher)
	userHandler := handler.NewUserHandler(userUsecase, validate)

	authUsecase := usecase.NewAuthUsecase(userUsecase, userRepo, passwordHasher, tokenIssuer)
	authHandler := handler.NewAuthHandler(authUsecase, validate)

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
		TaskHandler:      taskHandler,
		GoalsHandler:     goalsHandler,
		UserHandler:      userHandler,
		AuthHandler:      authHandler,
		TimeEntryHandler: timeEntryHandler,
		AuthMiddleware:   middleware.Auth(tokenIssuer),
		// Dipasang setelah AuthMiddleware di route.go — ia membaca role
		// yang ditaruh AuthMiddleware di c.Locals.
		AdminMiddleware: middleware.RequireAdmin(),
		AuthRateLimit:   middleware.RateLimit(cfg.Auth.RateLimitMax, cfg.Auth.RateLimitWindow),
		EnableDocs:      cfg.App.EnableDocs,
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
