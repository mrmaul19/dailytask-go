# Cara menambah endpoint baru

Panduan urutan kerja di codebase ini. Contoh yang dipakai: menambah resource
`users` (kolom `name`, `address`, `job`, `status`).

## Prinsip: selalu dari dalam ke luar

```
1 entity ──► 2 migrasi ──► 3 port ──► 4 repo SQL ──► 5 usecase ──► 6 test
                                                                     │
                                       9 route ◄── 8 handler ◄── 7 dto
                                          │
                                          └──► 10 main.go (resource baru saja)
                                                   │
                                                   └──► 11 swag init (regenerate docs/)
```

Alasannya praktis: tiap langkah hanya memakai hal yang **sudah ada** di langkah
sebelumnya, jadi `go build` tetap hijau sepanjang jalan. Kalau mulai dari handler,
kamu menulis pemanggilan ke usecase yang belum ada dan baru ketahuan salah di akhir.

## Tabel urutan

| # | Langkah | File | Perlu kapan? |
|---|---|---|---|
| 1 | Entity + aturan bisnis | `internal/domain/entity/` | ada field/invariant baru |
| 2 | Migrasi SQL | `migrations/000N_*.sql` | skema tabel berubah |
| 3 | Port repository | `internal/domain/repository/` | **query-nya beda**, bukan sekadar endpoint beda |
| 4 | Implementasi SQL | `internal/infrastructure/persistence/` | ikut langkah 2 atau 3 |
| 5 | Usecase | `internal/usecase/` | selalu |
| 6 | Unit test usecase | `internal/usecase/*_test.go` | selalu |
| 7 | DTO | `internal/delivery/http/dto/` | selalu |
| 8 | Handler | `internal/delivery/http/handler/` | selalu |
| 9 | Route | `internal/delivery/http/route/route.go` | selalu |
| 10 | Wiring | `cmd/api/main.go` | hanya kalau resource-nya baru |
| 11 | Regenerate spec Swagger | `swag init` → `docs/` | selalu, kalau langkah 7 atau 8 tersentuh |

Langkah 1–4 sering tidak perlu. Contoh: `GET /tasks/:id/download` tidak menambah
tabel maupun method repository — ia memakai `FindByID` yang sudah ada.

---

## 1. Entity

Lapisan paling dalam. Tidak boleh mengimpor fiber, pgx, atau apa pun dari luar.

```go
// internal/domain/entity/user.go
package entity

import (
	"errors"
	"time"
)

var ErrUserNotFound = errors.New("user tidak ditemukan")

type User struct {
	ID        uint
	Name      string
	Address   string
	Job       string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}
```

Kalau ada aturan bisnis, taruh sebagai method di sini — bukan di handler:

```go
const (
	StatusAktif    = "aktif"
	StatusNonaktif = "nonaktif"
)

var ErrStatusTidakValid = errors.New("status tidak valid")

func (u *User) ChangeStatus(status string) error {
	switch status {
	case StatusAktif, StatusNonaktif:
		u.Status = status
		return nil
	default:
		return ErrStatusTidakValid
	}
}
```

## 2. Migrasi

File baru bernomor urut. **Jangan pernah mengedit file yang sudah pernah jalan** —
riwayatnya dicatat per nama file di tabel `schema_migrations`, jadi file yang sudah
tercatat tidak akan dieksekusi ulang.

```sql
-- migrations/0003_create_users.sql
CREATE TABLE IF NOT EXISTS users (
    id         BIGSERIAL    PRIMARY KEY,
    name       VARCHAR(100) NOT NULL DEFAULT '',
    address    VARCHAR(255) NOT NULL DEFAULT '',
    job        VARCHAR(100) NOT NULL DEFAULT '',
    status     VARCHAR(100) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS users_deleted_at_idx ON users (deleted_at);
```

Kolom teks dibuat `NOT NULL DEFAULT ''` supaya bisa di-Scan langsung ke `string`
biasa, tanpa `sql.NullString` di repository.

> **Jebakan:** file migrasi biasanya dibuat dengan menyalin file sebelumnya. Kalau
> nama tabel lupa diganti, `CREATE TABLE IF NOT EXISTS <tabel_lama>` akan diam saja
> karena tabelnya memang sudah ada — lalu migrasi tetap tercatat sukses di
> `schema_migrations`. Tabel yang kamu maksud tidak pernah terbuat, dan tidak ada
> satu pun pesan error. Setelah menyalin, periksa **setiap** penyebutan nama tabel.

## 3. Port repository

Kontrak yang dipakai usecase. Implementasinya ada di `infrastructure`.

```go
// internal/domain/repository/user_repository.go
type UserFilter struct {
	Search string // dicocokkan ke name / job
	Page   int
	Limit  int
}

func (f UserFilter) Offset() int {
	if f.Page < 1 {
		return 0
	}
	return (f.Page - 1) * f.Limit
}

type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	FindAll(ctx context.Context, filter UserFilter) ([]entity.User, int64, error)
	FindByID(ctx context.Context, id uint) (*entity.User, error)
	Update(ctx context.Context, user *entity.User) error
	Delete(ctx context.Context, id uint) error
}
```

Tambahkan method baru **hanya kalau SQL-nya memang berbeda**. Endpoint baru yang
bisa dilayani method yang sudah ada tidak perlu method baru.

## 4. Implementasi SQL

Salin pola dari `goals_repository_sql.go`. Tiga hal yang wajib sinkron:

```go
// (a) daftar kolom — urutannya menentukan
const userColumns = `id, name, address, job, status, created_at, updated_at`

// (b) scanUser — urutan Scan HARUS persis sama dengan (a)
func scanUser(s rowScanner) (*entity.User, error) {
	var user entity.User
	err := s.Scan(&user.ID, &user.Name, &user.Address, &user.Job,
		&user.Status, &user.CreatedAt, &user.UpdatedAt)
	...
}

// (c) INSERT dan UPDATE — kolomnya ikut, nomor $N digeser
```

> **Jebakan:** kalau urutan (a) dan (b) tidak cocok, kodenya **tetap compile** tapi
> datanya tertukar saat runtime (`job` masuk ke `address`, dsb). ORM dulu menutupi
> ini lewat tag struct; dengan SQL manual, tanggung jawabnya ada di kamu.

Hal lain yang wajib diikuti:

- Semua nilai lewat parameter `$1, $2, ...` — jangan pernah dirangkai ke string query.
- Setiap query membawa `deleted_at IS NULL` (pakai konstanta `notDeleted`).
- `sql.ErrNoRows` diterjemahkan ke `entity.ErrUserNotFound`, jangan dibocorkan ke atas.
- `Update` dan `Create` memakai `RETURNING` supaya `updated_at` ikut segar tanpa query tambahan.

`rowScanner` dan `notDeleted` sudah ada di paket `persistence`, tinggal dipakai.

## 5. Usecase

Di sinilah error repository diterjemahkan menjadi `apperror` ber-`Kind`.

```go
type CreateUserInput struct {
	Name, Address, Job, Status string
}

type UserUsecase interface {
	Create(ctx context.Context, in CreateUserInput) (*entity.User, error)
	List(ctx context.Context, in ListUserInput) (*ListUserOutput, error)
	GetByID(ctx context.Context, id uint) (*entity.User, error)
	Update(ctx context.Context, id uint, in UpdateUserInput) (*entity.User, error)
	Delete(ctx context.Context, id uint) error
}

func (u *userUsecase) GetByID(ctx context.Context, id uint) (*entity.User, error) {
	user, err := u.userRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, entity.ErrUserNotFound) {
			return nil, apperror.NotFound(fmt.Sprintf("user dengan id %d tidak ditemukan", id))
		}
		return nil, apperror.Internal("gagal mengambil user", err)
	}
	return user, nil
}
```

`normalizePaging`, `defaultPage`, `defaultLimit`, dan `maxLimit` sudah ada di
`task_usecase.go`. Paketnya sama, jadi **jangan dideklarasikan ulang** — itu akan
jadi error `redeclared in this block`.

Untuk `Update`, pakai pointer agar "field tidak dikirim" (pakai nilai lama) bisa
dibedakan dari "field dikirim kosong" (ditolak). Lihat `goals_usecase.go` untuk
pola tabel field yang ringkas.

## 6. Unit test

Sebelum menyentuh HTTP sama sekali. Buat fake repository in-memory seperti
`fakeGoalsRepo` di `goals_usecase_test.go`. Helper `kindOf` sudah tersedia di
paket test, tinggal dipakai.

```bash
go test -count=1 ./internal/usecase/...
```

Test ini tidak butuh PostgreSQL hidup — itu keuntungan konkret dari layering ini.

## 7. DTO

Entity tidak pernah dikirim langsung ke client.

```go
type CreateUserRequest struct {
	Name    string `form:"name" json:"name" validate:"required,min=3,max=100"`
	Address string `form:"address" json:"address" validate:"omitempty,max=255"`
	Job     string `form:"job" json:"job" validate:"omitempty,max=100"`
	Status  string `form:"status" json:"status" validate:"required,max=100"`
}

type UpdateUserRequest struct {
	Name *string `form:"name" json:"name" validate:"omitempty,min=3,max=100"`
	// ...sisanya juga pointer
}
```

Validasi **bentuk** ada di sini. Validasi **aturan bisnis** tetap di entity.
Keduanya sengaja dobel: DTO menjaga request, entity menjaga invariant walau
dipanggil dari jalur lain.

Untuk resource baru, tambahkan juga envelope-nya di `dto/swagger_dto.go`:

```go
type UserEnvelope struct {
	Success bool         `json:"success" example:"true"`
	Message string       `json:"message" example:"detail user berhasil diambil"`
	Data    UserResponse `json:"data"`
}

type UserListEnvelope struct {
	Success bool           `json:"success" example:"true"`
	Message string         `json:"message" example:"daftar user berhasil diambil"`
	Data    []UserResponse `json:"data"`
	Meta    response.Meta  `json:"meta"`
}
```

`pkg/response` memakai `Data any`, jadi generator OpenAPI tidak bisa menebak isi
`data`. Envelope inilah yang nanti dirujuk anotasi `@Success` di langkah 8.

## 8. Handler

Terjemahan HTTP saja, tanpa logika bisnis. Pola yang konsisten di semua handler:

```go
func (h *UserHandler) GetByID(c *fiber.Ctx) error {
	id, err := parseID(c)       // helper paket handler, sudah ada
	if err != nil {
		return err
	}

	user, err := h.userUsecase.GetByID(c.UserContext(), id)
	if err != nil {
		return err              // SELALU return err
	}

	return response.Success(c, http.StatusOK, "detail user berhasil diambil",
		dto.NewUserResponse(user))
}
```

Jangan memakai `c.Status(404).JSON(...)`. Pemetaan `Kind` ke status code hanya
boleh ada di `middleware/error_handler.go`.

Sekalian tulis anotasi Swagger-nya, tepat di atas deklarasi method:

```go
//	@Summary		Detail user
//	@Description	User yang sudah dihapus (soft delete) tidak ikut terambil.
//	@Tags			Users
//	@Produce		json
//	@Param			id	path		int	true	"ID user"	minimum(1)
//	@Success		200	{object}	dto.UserEnvelope
//	@Failure		400	{object}	dto.ErrorEnvelope	"id bukan angka positif"
//	@Failure		404	{object}	dto.ErrorEnvelope	"user tidak ditemukan"
//	@Failure		500	{object}	dto.ErrorEnvelope
//	@Router			/api/v1/users/{id} [get]
func (h *UserHandler) GetByID(c *fiber.Ctx) error {
```

> **Jebakan:** `@Router` memakai `{id}` (gaya OpenAPI), bukan `:id` (gaya fiber).
> Kalau salah, endpoint-nya **tidak muncul** di Swagger UI dan `swag init` tetap
> selesai tanpa pesan error apa pun. Path-nya juga harus lengkap
> (`/api/v1/users/{id}`), karena `@BasePath` project ini `/`.

Untuk body JSON pakai `@Param request body dto.CreateUserRequest true "..."`.
Untuk form/multipart, satu `@Param` per field dengan tipe `formData`, dan tandai
`@Accept mpfd,json` kalau endpoint-nya menerima dua-duanya.

## 9. Route

```go
users := v1.Group("/users")
users.Post("/", deps.UserHandler.Create)
users.Get("/", deps.UserHandler.List)
users.Get("/:id", deps.UserHandler.GetByID)
users.Put("/:id", deps.UserHandler.Update)
users.Delete("/:id", deps.UserHandler.Delete)
```

> **Jebakan:** path literal harus didaftarkan **sebelum** path parameter di level
> yang sama. Kalau nanti menambah `GET /users/stats`, ia harus ditulis di atas
> `users.Get("/:id")` — kalau tidak, `/stats` tertangkap `:id` dan gagal di
> `parseID` dengan 400.

## 10. Wiring

Dua tempat, dan yang pertama sering terlupa:

```go
// internal/delivery/http/route/route.go
type Deps struct {
	TaskHandler  *handler.TaskHandler
	GoalsHandler *handler.GoalsHandler
	UserHandler  *handler.UserHandler   // <-- tambah
}

// cmd/api/main.go
userRepo := persistence.NewUserRepository(db)
userUsecase := usecase.NewUserUsecase(userRepo)
userHandler := handler.NewUserHandler(userUsecase, validate)

route.Register(app, route.Deps{
	TaskHandler:  taskHandler,
	GoalsHandler: goalsHandler,
	UserHandler:  userHandler,           // <-- tambah
})
```

`validate` sudah dibuat sekali di `main.go` dan dipakai bersama semua handler.

## 11. Regenerate spec Swagger

Anotasi yang baru ditulis belum terpakai sampai spec-nya digenerate ulang:

```bash
swag init -g cmd/api/main.go --output docs --parseInternal
```

Tiga hal yang perlu diingat:

- Spec **tidak** dibaca saat runtime — ia dikompilasi dari `docs/docs.go` yang
  di-import `cmd/api/main.go`. Lupa regenerate = UI masih versi lama.
- `--parseInternal` wajib. Tanpa itu swag melewati paket di bawah `internal/`,
  dan karena semua DTO kita ada di sana, `definitions` keluar kosong.
- `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml` adalah hasil generate.
  Jangan diedit tangan, tapi **harus ikut di-commit** — kalau tidak, `go build`
  gagal di mesin lain.

Kalau `swag` belum ada: `go install github.com/swaggo/swag/cmd/swag@latest`.

---

## Checklist verifikasi

```bash
swag init -g cmd/api/main.go --output docs --parseInternal
go build ./...
go vet ./...
gofmt -l .
go test -count=1 ./...

# penjaga arsitektur — keduanya HARUS kosong
go list -deps ./internal/domain/... | grep -E "fiber|pgx"
go list -deps ./internal/usecase    | grep -E "fiber|pgx"

go run ./cmd/api
# sesuaikan portnya dengan APP_PORT di .env
curl http://localhost:3000/api/v1/users

# endpoint baru benar-benar masuk spec?
curl -s http://localhost:3000/swagger/doc.json | grep -o '"/api/v1/users/{id}"'
```

Lalu buka <http://localhost:3000/swagger/index.html> dan cek endpoint barunya
muncul di grup yang benar.

Lalu pastikan migrasi benar-benar membuat tabelnya:

```sql
-- di DBeaver, database daily_task
SELECT version FROM schema_migrations ORDER BY version;
\dt
```

Terakhir, tambahkan request-nya ke koleksi di `postman/` supaya **Run collection**
tetap mencakup semua endpoint.

## Kesalahan yang paling sering terjadi

| Jangan | Lakukan |
|---|---|
| `c.Status(404).JSON(...)` di handler | `return apperror.NotFound(...)` dari usecase |
| Return `entity.User` langsung | Selalu lewat `dto.NewUserResponse` |
| Aturan bisnis di handler atau DTO | Taruh di entity atau usecase |
| `os.Getenv` di luar `config/` | Tambahkan field di `config/config.go` |
| Deklarasi ulang `normalizePaging` dkk | Paketnya sama, pakai yang sudah ada |
| Lupa `deleted_at IS NULL` di query baru | Pakai konstanta `notDeleted` |
| Rangkai nilai ke string SQL | Selalu `$1, $2, ...` sebagai parameter |
| Edit file migrasi yang sudah jalan | Buat file `000N` baru |
| Salin migrasi tanpa ganti nama tabel | Gagal diam-diam, tidak ada error |
| Ubah anotasi tapi lupa `swag init` | Regenerate, lalu commit isi `docs/` |
| Edit `docs/swagger.json` langsung | Edit anotasinya, lalu `swag init` |
| `@Router .../:id` (gaya fiber) | `@Router .../{id}` (gaya OpenAPI) |
| Anotasi di atas closure `app.Get(...)` | Pisah jadi fungsi bernama dulu |

## Catatan untuk test script Postman

Jangan menulis `const data = ...` di tab Scripts. Sandbox Postman sudah punya
variabel global bernama `data`, dan deklarasi itu melempar
`SyntaxError: Identifier 'data' has already been declared` — akibatnya **seluruh
test script request tersebut tidak dieksekusi**, tanpa tanda apa pun selain
hitungan assertion yang lebih sedikit. Pakai nama lain, misalnya `hasil`.
