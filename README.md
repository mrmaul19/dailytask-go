# Daily Task API — Go Fiber + Clean Architecture

Mini project REST API untuk mencatat daily task: **no task, judul, description, file lampiran**,
plus resource `goals` dan `users`.
Stack: **Go Fiber + database/sql (driver pgx) + PostgreSQL** — tanpa ORM, seluruh query ditulis manual.
Dipakai sebagai bahan latihan Clean Architecture di Go.

**Daftar isi**

1. [Cara menjalankan](#cara-menjalankan)
2. [Dokumentasi API (Swagger)](#dokumentasi-api-swagger)
3. [Alur kerja: menambah endpoint baru (step by step)](#alur-kerja-menambah-endpoint-baru-step-by-step)
4. [Arsitektur yang dipakai](#arsitektur-yang-dipakai)
5. [Peta file & folder, berurutan](#peta-file--folder-berurutan)
6. [Daftar endpoint](#daftar-endpoint)
7. [Contoh request](#contoh-request)
8. [Postman](#postman)
9. [Bentuk response](#bentuk-response)
10. [Catatan implementasi yang perlu dipahami](#catatan-implementasi-yang-perlu-dipahami)
11. [Lanjutan untuk latihan](#lanjutan-untuk-latihan)

## Cara menjalankan

1. **Siapkan database.** Kalau sudah punya PostgreSQL sendiri, cukup buat databasenya:

   ```sql
   CREATE DATABASE daily_task;
   ```

   Kalau belum punya, pakai container yang sudah disiapkan:

   ```bash
   docker compose up -d
   ```

2. **Jalankan aplikasi.**

   ```bash
   cp .env.example .env     # sesuaikan DB_USER / DB_PASSWORD
   go mod tidy
   go run ./cmd/api         # http://localhost:3000
   go test ./...            # unit test usecase (tidak butuh database)
   ```

   Port mengikuti `APP_PORT` di `.env` (default `3000`). Semua contoh URL di README ini
   memakai `3000`; sesuaikan kalau `.env` Anda memakai port lain.
   Setelah hidup, dokumentasi interaktifnya ada di
   **<http://localhost:3000/swagger/index.html>** — lihat [bagian berikutnya](#dokumentasi-api-swagger).

Tabel dibuat otomatis saat aplikasi start oleh runner migrasi di [migrate.go](internal/infrastructure/database/migrate.go):
file `.sql` di [migrations/](migrations/) dijalankan urut nama, masing-masing dalam satu transaksi,
lalu dicatat di tabel `schema_migrations` supaya tidak pernah jalan dua kali.
Kalau koneksi salah, aplikasi berhenti dengan pesan jelas — `database.New` melakukan ping,
jadi error muncul saat start, bukan saat request pertama masuk.

> Port 5432 sudah dipakai instance Postgres lain? Set `DB_PORT=5433` di `.env`,
> lalu `docker compose up -d` — aplikasi dan container otomatis ikut port itu.

Catatan: unit test usecase memakai fake repository & fake storage, jadi `go test ./...`
tetap jalan tanpa PostgreSQL hidup. Itu salah satu keuntungan konkret dari layering ini.

---

## Dokumentasi API (Swagger)

Spec OpenAPI **digenerate dari anotasi komentar di handler**, bukan ditulis sebagai file YAML
terpisah. Jadi dokumentasi tinggal satu tempat dengan kodenya dan tidak gampang basi.

| Komponen | Keterangan |
|---|---|
| Generator | [`swaggo/swag`](https://github.com/swaggo/swag) — CLI yang membaca anotasi `@Summary`, `@Param`, dst. |
| Penyaji UI | [`gofiber/swagger`](https://github.com/gofiber/swagger) — middleware fiber yang menyajikan Swagger UI |
| Spec OpenAPI | **2.0 (Swagger)** — versi yang didukung swag v1 |
| Hasil generate | `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml` |

### Membukanya

```
http://localhost:3000/swagger/index.html     ← Swagger UI (interaktif, ada "Try it out")
http://localhost:3000/swagger/doc.json       ← spec mentah, untuk Postman/Insomnia/codegen
```

URL-nya juga ikut tercetak di log saat aplikasi start.

Bisa dimatikan lewat `.env` — berguna kalau tidak mau spec API terbuka di publik:

```bash
ENABLE_DOCS=false      # default true; /swagger/* jadi 404
```

### Sekali saja: install CLI-nya

`swag` adalah tool terpisah, bukan dependency aplikasi, jadi tidak ikut `go mod tidy`:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

Binary-nya masuk ke `$(go env GOPATH)/bin`. Kalau `swag` tidak dikenali, folder itu belum ada
di `PATH` — panggil lewat path lengkapnya saja:

```bash
# Windows (Git Bash / PowerShell)
"$(go env GOPATH)/bin/swag" --version
```

### Regenerate setelah mengubah anotasi

**Wajib dijalankan setiap kali anotasi atau DTO berubah.** Spec tidak dibaca saat runtime —
ia dikompilasi dari `docs/docs.go`, jadi tanpa regenerate, UI masih menampilkan versi lama:

```bash
swag init -g cmd/api/main.go --output docs --parseInternal
```

Arti flag-nya:

| Flag | Kenapa perlu |
|---|---|
| `-g cmd/api/main.go` | file yang memuat anotasi umum (`@title`, `@version`, `@tag.*`) |
| `--output docs` | tujuan hasil generate |
| `--parseInternal` | **penting** — tanpa ini swag melewati paket di bawah `internal/`, dan semua DTO kita ada di sana sehingga `definitions` keluar kosong |

Hasilnya di-commit ke repo, karena `cmd/api/main.go` meng-import `daily-task-go/docs`;
tanpa `docs/docs.go` ikut ter-commit, `go build` di mesin lain gagal.

### Bagaimana potongannya saling nyambung

```
anotasi di handler/route  ──swag init──►  docs/docs.go  ──import di main.go──►  spec terdaftar
                                                                                      │
                     browser  ◄──Swagger UI──  app.Get("/swagger/*")  ◄───────────────┘
                                               (route.go, kalau EnableDocs)
```

Empat tempat yang terlibat:

1. **Anotasi** — komentar `@Summary` / `@Param` / `@Success` di atas tiap method handler, plus
   anotasi umum di atas `func main()` ([cmd/api/main.go](cmd/api/main.go)).
2. **Envelope DTO** — [swagger_dto.go](internal/delivery/http/dto/swagger_dto.go). `pkg/response`
   memakai `Data any`, jadi generator tidak bisa menebak isi `data`. File ini menyediakan tipe
   konkret per endpoint (`dto.TaskEnvelope`, `dto.TaskListEnvelope`, `dto.ErrorEnvelope`, dst.)
   yang dirujuk anotasi `@Success` / `@Failure`.
3. **Registrasi route** — `app.Get("/swagger/*", fiberSwagger.HandlerDefault)` di
   [route.go](internal/delivery/http/route/route.go), dipasang hanya kalau `EnableDocs` true.
4. **Host dinamis** — `main.go` mengisi `SwaggerInfo.Host` dari `cfg.App.BaseURL`, bukan dari
   anotasi `@host` yang statis. Dengan begini tombol **Try it out** tetap menembak host yang benar
   walau port atau domainnya berubah.

> **Jebakan:** `/health` sengaja dipisah menjadi fungsi bernama `healthCheck` di `route.go`.
> swag hanya membaca komentar di atas **deklarasi func**, jadi closure yang ditulis langsung di
> dalam `app.Get("/health", func(c *fiber.Ctx) error { ... })` tidak akan pernah ikut terdokumentasi.

---

## Alur kerja: menambah endpoint baru (step by step)

**Aturan emasnya: selalu kerjakan dari dalam ke luar.** Mulai dari `entity`, berakhir di `route` +
`main.go`. Alasannya praktis — tiap langkah hanya memakai hal yang **sudah ada** di langkah
sebelumnya, jadi `go build ./...` tetap hijau sepanjang jalan. Kalau mulai dari handler, kamu
menulis pemanggilan ke usecase yang belum ada dan baru ketahuan salah di akhir.

```
 STEP 1       STEP 2        STEP 3       STEP 4         STEP 5      STEP 6
 entity  ──►  migrasi  ──►  port    ──►  repo SQL  ──►  usecase ──► unit test
                            (interface)  (implementasi)                 │
                                                                        ▼
 STEP 10  ◄──  STEP 9  ◄──  STEP 8  ◄──  STEP 7
 wiring        route        handler      dto
 (main.go)                  + anotasi    + envelope
                              swagger      swagger
     │
     └──►  STEP 11 swag init  ──►  STEP 12 verifikasi
```

### Step 0 — tentukan dulu: step mana yang perlu?

Tidak semua endpoint butuh 10 langkah. Cek tabel ini sebelum mulai menulis kode:

| Kasus | Step yang dikerjakan |
|---|---|
| Resource benar-benar baru (mis. `users`) | **1 → 12** (semua) |
| Endpoint baru di resource lama, query SQL-nya **sudah ada** (mis. `GET /tasks/:id/download`) | **7, 8, 9** saja |
| Endpoint baru di resource lama, butuh **query SQL baru** (mis. filter `?status=`) | **3, 4, 5, 6, 7, 8, 9** |
| Tambah kolom baru di tabel yang sudah ada | **1, 2, 4, 5, 6, 7** (+8 kalau request/response berubah) |
| Ganti teknologi penyimpanan (disk → S3) | **4 saja** (ganti implementasi port, usecase tidak disentuh) |

Step **11 (`swag init`) dan 12 (verifikasi) selalu dikerjakan** kalau ada perubahan yang menyentuh
step 7 atau 8 — yaitu setiap kali bentuk request/response atau anotasinya berubah.

Contoh lengkap yang sudah jadi di repo ini: resource **`users`** (`user_*.go` di tiap layer).
Pakai file-file itu sebagai cetakan saat mengerjakan step di bawah.
Versi panjang beserta semua jebakannya ada di [docs/menambah-endpoint.md](docs/menambah-endpoint.md).

### Step 1 — Entity

**File:** `internal/domain/entity/<resource>.go`
**Tujuan:** bentuk data inti + aturan bisnis, bebas dari framework.

```go
package entity

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

Aturan:

- **Dilarang** mengimpor fiber, pgx, atau apa pun dari luar — hanya stdlib.
- Tidak ada tag `db:` / `json:` di sini. Entity bukan baris tabel dan bukan response JSON.
- Punya aturan bisnis (mis. status hanya boleh `aktif`/`nonaktif`)? Taruh sebagai **method di
  entity**, bukan di handler.
- Siapkan satu `Err<Resource>NotFound` di sini; dipakai repository dan dibaca usecase.

### Step 2 — Migrasi SQL

**File:** `migrations/000N_create_<tabel>.sql` (nomor urut berikutnya)
**Tujuan:** skema tabel. Otomatis jalan saat aplikasi start.

```sql
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

Aturan:

- **Jangan pernah mengedit file migrasi yang sudah pernah jalan.** Riwayat dicatat per nama file
  di `schema_migrations`, jadi file yang sudah tercatat tidak akan dieksekusi ulang. Perubahan
  skema = file `000N` baru.
- Kolom teks dibuat `NOT NULL DEFAULT ''` supaya bisa di-`Scan` langsung ke `string`, tanpa
  `sql.NullString` di repository.
- Selalu sertakan `deleted_at TIMESTAMPTZ` — pola delete di project ini soft delete.
- **Jebakan:** file migrasi biasanya dibuat dengan menyalin file sebelumnya. Kalau nama tabel
  lupa diganti, `CREATE TABLE IF NOT EXISTS <tabel_lama>` **diam saja** (tabelnya memang sudah
  ada), migrasi tetap tercatat sukses, dan tabel yang kamu maksud tidak pernah terbuat — tanpa
  satu pun pesan error. Setelah menyalin, periksa **setiap** penyebutan nama tabel.

### Step 3 — Port repository (interface)

**File:** `internal/domain/repository/<resource>_repository.go`
**Tujuan:** kontrak akses data yang dipakai usecase. Implementasinya nanti di `infrastructure`.

```go
type UserFilter struct {
	Search string // dicocokkan ke name / job
	Page   int
	Limit  int
}

func (f UserFilter) Offset() int { ... }

type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	FindAll(ctx context.Context, filter UserFilter) ([]entity.User, int64, error)
	FindByID(ctx context.Context, id uint) (*entity.User, error)
	Update(ctx context.Context, user *entity.User) error
	Delete(ctx context.Context, id uint) error
}
```

Aturan:

- Tambahkan method baru **hanya kalau SQL-nya memang berbeda**. Endpoint baru yang bisa dilayani
  method lama (mis. `/download` memakai `FindByID`) tidak perlu method baru.
- Signature-nya memakai tipe `entity` dan stdlib saja — tidak ada `*sql.Rows` yang bocor ke sini.
- `context.Context` selalu jadi parameter pertama.

### Step 4 — Implementasi SQL

**File:** `internal/infrastructure/persistence/<resource>_repository_sql.go`
**Tujuan:** satu-satunya tempat yang tahu nama kolom dan bentuk SQL.

Salin pola dari [goals_repository_sql.go](internal/infrastructure/persistence/goals_repository_sql.go). Tiga hal yang wajib sinkron:

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

Aturan:

- **Jebakan utama:** kalau urutan (a) dan (b) tidak cocok, kodenya **tetap compile** tapi datanya
  tertukar saat runtime (`job` masuk ke `address`, dst.). ORM dulu menutupi ini lewat tag struct;
  dengan SQL manual, tanggung jawabnya ada di kamu.
- Semua nilai lewat parameter `$1, $2, ...` — **jangan pernah** dirangkai ke string query.
- Setiap query membawa `deleted_at IS NULL` (pakai konstanta `notDeleted` yang sudah ada).
- `sql.ErrNoRows` diterjemahkan ke `entity.Err<Resource>NotFound`, jangan dibocorkan ke atas.
- `Create` dan `Update` memakai `RETURNING` supaya `updated_at` ikut segar tanpa query tambahan.
- Pencarian pakai `ILIKE`, bukan `LIKE` (di PostgreSQL `LIKE` case-sensitive).
- `rowScanner` dan `notDeleted` sudah ada di paket `persistence` — tinggal dipakai, jangan dibuat ulang.

### Step 5 — Usecase

**File:** `internal/usecase/<resource>_usecase.go`
**Tujuan:** aturan bisnis + penerjemah error repository menjadi `apperror` ber-`Kind`.

```go
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

Aturan:

- Input/output usecase pakai struct sendiri (`CreateUserInput`), **bukan** DTO HTTP. Usecase tidak
  boleh tahu soal fiber atau multipart.
- Di sinilah error teknis jadi error berarti: `NotFound`, `Validation`, `Conflict`, `Internal`.
- Untuk `Update`, pakai pointer agar "field tidak dikirim" (pakai nilai lama) bisa dibedakan dari
  "field dikirim kosong" (ditolak). Pola tabel field yang ringkas ada di
  [goals_usecase.go](internal/usecase/goals_usecase.go).
- `normalizePaging`, `defaultPage`, `defaultLimit`, `maxLimit` sudah ada di
  [task_usecase.go](internal/usecase/task_usecase.go). Paketnya sama, jadi **jangan dideklarasikan
  ulang** — itu error `redeclared in this block`.

### Step 6 — Unit test usecase

**File:** `internal/usecase/<resource>_usecase_test.go`
**Tujuan:** menguji aturan bisnis **sebelum** menyentuh HTTP sama sekali.

```bash
go test -count=1 ./internal/usecase/...
```

Aturan:

- Buat fake repository in-memory seperti `fakeGoalsRepo` di
  [goals_usecase_test.go](internal/usecase/goals_usecase_test.go) — tanpa mock library, tanpa database.
- Helper `kindOf` sudah tersedia di paket test, tinggal dipakai.
- Test ini tidak butuh PostgreSQL hidup. Kalau ternyata butuh, berarti ada kebocoran layer.

### Step 7 — DTO

**File:** `internal/delivery/http/dto/<resource>_dto.go`
**Tujuan:** bentuk request & response JSON. Entity tidak pernah dikirim langsung ke client.

```go
type CreateUserRequest struct {
	Name    string `form:"name" json:"name" validate:"required,min=3,max=100"`
	Address string `form:"address" json:"address" validate:"required,max=255"`
	// ...
}

type UpdateUserRequest struct {
	Name *string `form:"name" json:"name" validate:"omitempty,min=3,max=100"`
	// ...sisanya juga pointer
}

func NewUserResponse(user *entity.User) UserResponse { ... }
```

Aturan:

- Validasi **bentuk** (panjang, wajib/tidak, tipe) ada di sini. Validasi **aturan bisnis** tetap di
  entity/usecase. Keduanya sengaja dobel: DTO menjaga request, entity menjaga invariant walau
  dipanggil dari jalur lain.
- Request update memakai pointer **plus** `omitempty`. Kalau pointer diberi `required`, request
  yang tidak mengirim field itu langsung ditolak 400 — partial update jadi tidak jalan meski
  usecase-nya sudah mendukung.
- Sediakan `New<Resource>Response` (satu data) dan `New<Resource>Responses` (list) di sini, supaya
  handler tidak merakit JSON manual.
- Tag `form:` dan `json:` dipasang bersamaan agar endpoint menerima JSON maupun `multipart/form-data`.
- **Untuk Swagger:** tambahkan juga envelope-nya di
  [swagger_dto.go](internal/delivery/http/dto/swagger_dto.go) — `<Resource>Envelope` (satu data) dan
  `<Resource>ListEnvelope` (list + `response.Meta`). Tanpa itu, anotasi `@Success` tidak punya tipe
  konkret untuk dirujuk, karena `response.Body.Data` bertipe `any`.

### Step 8 — Handler

**File:** `internal/delivery/http/handler/<resource>_handler.go`
**Tujuan:** terjemahan HTTP ⇄ usecase. Tanpa logika bisnis, tanpa SQL.

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

Urutan isi setiap handler selalu sama: **parse → validate → panggil usecase → `return err` →
bungkus response**.

Aturan:

- **Jangan** memakai `c.Status(404).JSON(...)`. Pemetaan `Kind` → status code hanya boleh ada di
  [error_handler.go](internal/delivery/http/middleware/error_handler.go).
- Pakai `c.UserContext()`, bukan `context.Background()`, supaya request yang dibatalkan ikut
  membatalkan query.
- Balasan selalu lewat `response.Success` / `response.SuccessWithMeta` agar bentuk JSON konsisten.

**Sekalian tulis anotasi Swagger-nya di sini**, tepat di atas deklarasi method:

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
```

Yang sering salah pada anotasi:

- `@Router` memakai **`{id}`**, bukan `:id` seperti di fiber. Salah tulis = endpoint tidak muncul
  di UI tanpa pesan error apa pun.
- Path di `@Router` harus **lengkap** (`/api/v1/users/{id}`), karena `@BasePath` di project ini `/`.
- Body JSON pakai `@Param request body dto.CreateUserRequest true "..."`;
  form/multipart pakai satu `@Param` per field dengan tipe `formData`.
- Endpoint yang menerima dua-duanya ditulis `@Accept mpfd,json`. Untuk download file:
  `@Produce octet-stream` + `@Success 200 {file} binary`.
- Daftarkan `@Tags` yang namanya sudah ada di `@tag.*` pada `main.go`, supaya grup di UI tidak pecah.

### Step 9 — Route

**File:** `internal/delivery/http/route/route.go`
**Tujuan:** daftar endpoint di satu tempat.

```go
users := v1.Group("/users")
users.Post("/", deps.UserHandler.Create)
users.Get("/", deps.UserHandler.List)
users.Get("/:id", deps.UserHandler.GetByID)
users.Put("/:id", deps.UserHandler.Update)
users.Delete("/:id", deps.UserHandler.Delete)
```

Aturan:

- **Jebakan:** path literal harus didaftarkan **sebelum** path parameter di level yang sama. Kalau
  menambah `GET /users/stats`, ia harus ditulis di atas `users.Get("/:id")` — kalau tidak,
  `/stats` tertangkap `:id` lalu gagal di `parseID` dengan 400.
- Tambahkan handler baru ke struct `route.Deps` di file yang sama — ini yang paling sering terlupa.

### Step 10 — Wiring di composition root

**File:** `cmd/api/main.go` (hanya kalau resource-nya baru)
**Tujuan:** merakit repository → usecase → handler, lalu menyerahkannya ke `route.Register`.

```go
userRepo := persistence.NewUserRepository(db)
userUsecase := usecase.NewUserUsecase(userRepo)
userHandler := handler.NewUserHandler(userUsecase, validate)

route.Register(app, route.Deps{
	TaskHandler:  taskHandler,
	GoalsHandler: goalsHandler,
	UserHandler:  userHandler,           // <-- tambah
})
```

Aturan:

- Hanya di file ini implementasi konkret dipilih. Semua layer lain cuma kenal interface.
- `validate` dibuat sekali lalu dipakai bersama semua handler — jangan buat instance baru per handler.

### Step 11 — Regenerate spec Swagger

**File:** `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml` (semuanya hasil generate — jangan diedit tangan)
**Tujuan:** menyalin anotasi yang baru ditulis ke dalam spec OpenAPI.

```bash
swag init -g cmd/api/main.go --output docs --parseInternal
```

Aturan:

- Spec **tidak** dibaca saat runtime; ia dikompilasi dari `docs/docs.go`. Lupa regenerate = UI
  masih menampilkan endpoint versi lama, padahal kodenya sudah berubah.
- Jangan pernah mengedit `docs/swagger.json` atau `docs/docs.go` secara manual — perubahan itu
  akan hilang pada `swag init` berikutnya. Yang diedit adalah anotasinya.
- `--parseInternal` wajib ada, karena seluruh DTO project ini berada di bawah `internal/`.
- Commit hasil generate-nya. `cmd/api/main.go` meng-import `daily-task-go/docs`, jadi build gagal
  di mesin lain kalau file itu tidak ikut.

### Step 12 — Verifikasi

```bash
go build ./...
go vet ./...
gofmt -l .                 # harus kosong
go test -count=1 ./...

# penjaga arsitektur — keduanya HARUS kosong
go list -deps ./internal/domain/... | grep -E "fiber|pgx"
go list -deps ./internal/usecase    | grep -E "fiber|pgx"

go run ./cmd/api
curl http://localhost:3000/api/v1/users

# endpoint baru benar-benar masuk spec?
curl -s http://localhost:3000/swagger/doc.json | grep -o '"/api/v1/users/{id}"'
```

Lalu buka <http://localhost:3000/swagger/index.html> dan pastikan endpoint barunya muncul di grup
yang benar, lalu coba lewat tombol **Try it out**.

Lalu pastikan migrasi benar-benar membuat tabelnya:

```sql
-- di DBeaver / psql, database daily_task
SELECT version FROM schema_migrations ORDER BY version;
```

Terakhir, tambahkan request-nya ke koleksi di [postman/](postman/) supaya **Run collection** tetap
mencakup semua endpoint.

### Kesalahan yang paling sering terjadi

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
| `required` pada pointer di request update | Pakai `omitempty` supaya partial update jalan |
| Ubah anotasi tapi lupa `swag init` | Regenerate, lalu commit isi `docs/` |
| Edit `docs/swagger.json` langsung | Edit anotasinya, lalu `swag init` |
| `@Router /api/v1/users/:id` (gaya fiber) | `@Router /api/v1/users/{id}` (gaya OpenAPI) |

---

## Arsitektur yang dipakai

**Clean Architecture** (dikenal juga sebagai hexagonal / ports & adapters) dengan empat lapisan.
Inti aturannya satu: **panah ketergantungan selalu mengarah ke dalam.**

```
        cmd/api (composition root — merakit semua)
                      │
            ┌─────────┴──────────┐
            ▼                    ▼
   delivery/http          infrastructure
   (fiber handler)      (database/sql, disk storage)
            │                    │
            └─────────┬──────────┘
                      ▼
                   usecase          ← aturan bisnis
                      │
                      ▼
                   domain           ← entity + port (interface)
```

| Lapisan | Isi | Tahu apa | Tidak tahu apa |
|---|---|---|---|
| **domain** | entity + port (interface) | stdlib saja | fiber, pgx, HTTP, SQL |
| **usecase** | aturan bisnis, orkestrasi | domain (interface) | fiber, pgx, nama kolom |
| **infrastructure** | implementasi port | domain, config | HTTP, usecase |
| **delivery** | handler, dto, middleware, route | usecase, dto, pkg | SQL, nama kolom |
| **cmd/api** | composition root | semuanya | — |

Empat konsekuensi yang membuat layering ini bukan sekadar hiasan:

1. `usecase` hanya tahu interface (`TaskRepository`, `FileStorage`), bukan implementasinya — jadi
   unit test jalan dengan fake repository, **tanpa PostgreSQL**.
2. `infrastructure` **mengimplementasikan** interface milik domain (dependency inversion). Ganti
   disk → S3 = tambah satu adapter, usecase nol perubahan.
3. Ganti database = ganti isi `persistence/`, karena hanya file di situ yang tahu nama kolom.
4. Ganti fiber → net/http = ganti isi `delivery/http/`, entity dan usecase tidak disentuh.

Cek sendiri aturan ini masih berlaku:

```bash
go list -deps ./internal/domain/... | grep -E "fiber|pgx"   # harus kosong
go list -deps ./internal/usecase    | grep -E "fiber|pgx"   # harus kosong
```

### Perjalanan satu request, urut per file

Contoh `GET /api/v1/tasks/1`:

| # | File | Yang terjadi |
|---|---|---|
| 1 | [cmd/api/main.go](cmd/api/main.go) | (saat start) semua dependency dirakit, server menyala |
| 2 | [route/route.go](internal/delivery/http/route/route.go) | path `/api/v1/tasks/:id` dicocokkan → `TaskHandler.GetByID` |
| 3 | [handler/task_handler.go](internal/delivery/http/handler/task_handler.go) | `parseID` baca `:id`, lalu panggil usecase |
| 4 | [usecase/task_usecase.go](internal/usecase/task_usecase.go) | aturan bisnis, panggil port `TaskRepository.FindByID` |
| 5 | [repository/task_repository.go](internal/domain/repository/task_repository.go) | interface-nya — ini yang dikenal usecase |
| 6 | [persistence/task_repository_sql.go](internal/infrastructure/persistence/task_repository_sql.go) | implementasi: `SELECT ... WHERE id=$1 AND deleted_at IS NULL`, hasil di-`Scan` ke entity |
| 7 | [entity/task.go](internal/domain/entity/task.go) | bentuk data yang mengalir balik ke atas |
| 8 | usecase | `sql.ErrNoRows` → `entity.ErrTaskNotFound` → `apperror.NotFound` |
| 9 | [dto/task_dto.go](internal/delivery/http/dto/task_dto.go) | entity diubah jadi `TaskResponse` (`file_url`, bukan `file_path`) |
| 10 | [pkg/response/response.go](pkg/response/response.go) | dibungkus `{success, message, data, meta}` |
| 11 | [middleware/error_handler.go](internal/delivery/http/middleware/error_handler.go) | kalau tadi error: `Kind` → status code (404/400/409/500) |

Perhatikan arahnya: turun lewat delivery → usecase → domain, lalu naik lagi. Tidak ada satu pun
langkah yang melompati layer.

---

## Peta file & folder, berurutan

Urutan di bawah mengikuti arah yang sama dengan step menambah endpoint: **dari lapisan paling
dalam ke paling luar.** Kalau baru membaca codebase ini pertama kali, baca dengan urutan ini juga.

### 1. `internal/domain/` — lapisan paling dalam, tanpa ketergantungan

| File | Guna |
|---|---|
| [entity/task.go](internal/domain/entity/task.go) | struct `Task` murni + `ErrTaskNotFound`. Tidak ada tag `db`/`json`. |
| [entity/goals.go](internal/domain/entity/goals.go) | struct `Goals` + `ErrGoalsNotFound`. |
| [entity/user.go](internal/domain/entity/user.go) | struct `User` + `ErrUserNotFound`. |
| [repository/task_repository.go](internal/domain/repository/task_repository.go) | **port** akses data task: interface `TaskRepository` + `TaskFilter` (search/page/limit + `Offset()`). |
| [repository/goals_repository.go](internal/domain/repository/goals_repository.go) | port akses data goals. |
| [repository/user_repository.go](internal/domain/repository/user_repository.go) | port akses data user. |
| [storage/file_storage.go](internal/domain/storage/file_storage.go) | **port** penyimpanan file: `Save` / `Open` / `Delete`. Sengaja memakai `io.Reader`, bukan `*multipart.FileHeader`, supaya usecase tidak terikat HTTP. |
| [security/password_hasher.go](internal/domain/security/password_hasher.go) | **port** hashing password (`Hash` + `Verify`). Dibuat interface supaya algoritmanya bisa diganti (bcrypt → argon2id) tanpa menyentuh usecase, dan supaya unit test bisa memakai hasher cepat. |
| [security/token_issuer.go](internal/domain/security/token_issuer.go) | **port** token akses: `Claims`, `Token`, `Issue`, `Parse`. Menukar JWT dengan session opaque cukup mengganti adapternya. |

Semua file di folder ini hanya mengimpor stdlib. Itu yang dijaga oleh perintah `go list -deps` di atas.

### 2. `internal/usecase/` — aturan bisnis

| File | Guna |
|---|---|
| [task_usecase.go](internal/usecase/task_usecase.go) | alur task: generate `no_task` (`TASK-YYYYMMDD-XXXX`), urutan simpan file ↔ insert DB, partial update, hapus lampiran saat delete. Juga rumah dari helper paging (`normalizePaging`, `defaultPage`, `defaultLimit`, `maxLimit`) yang dipakai seluruh paket. |
| [goals_usecase.go](internal/usecase/goals_usecase.go) | alur goals; contoh pola partial update pakai tabel field + pointer. |
| [user_usecase.go](internal/usecase/user_usecase.go) | alur user; pola yang sama, tanpa file lampiran. |
| [auth_usecase.go](internal/usecase/auth_usecase.go) | register & login. Register memakai ulang `UserUsecase.Create` (tidak menulis ulang hashing/cek email unik); login menyamakan respons & waktu eksekusi antara "email tidak ada" dan "password salah". |
| [task_usecase_test.go](internal/usecase/task_usecase_test.go) | unit test dengan fake repository **dan** fake storage. |
| [goals_usecase_test.go](internal/usecase/goals_usecase_test.go) | unit test + `fakeGoalsRepo` dan helper `kindOf` yang dipakai test lain. |
| [user_usecase_test.go](internal/usecase/user_usecase_test.go) | unit test user. |

Di layer inilah error repository diterjemahkan menjadi `apperror` ber-`Kind`. Usecase tidak pernah
menyebut status code.

### 3. `internal/infrastructure/` — adapter ke dunia luar

| File | Guna |
|---|---|
| [database/postgres.go](internal/infrastructure/database/postgres.go) | buka koneksi `database/sql` dengan driver pgx, set pool, lalu **ping** — error koneksi muncul saat start, bukan saat request pertama. |
| [database/migrate.go](internal/infrastructure/database/migrate.go) | runner migrasi: baca `.sql` dari `migrations/`, jalankan urut nama, satu transaksi per file, catat di `schema_migrations`. |
| [persistence/task_repository_sql.go](internal/infrastructure/persistence/task_repository_sql.go) | implementasi `TaskRepository`. **Satu-satunya** yang tahu nama kolom tasks. Rumah dari helper `rowScanner` dan konstanta `notDeleted`. |
| [persistence/goals_repository_sql.go](internal/infrastructure/persistence/goals_repository_sql.go) | implementasi `GoalsRepository` — cetakan paling bersih untuk resource baru tanpa file. |
| [persistence/user_repository_sql.go](internal/infrastructure/persistence/user_repository_sql.go) | implementasi `UserRepository`. |
| [storage/local_storage.go](internal/infrastructure/storage/local_storage.go) | implementasi `FileStorage` ke disk: acak nama file, validasi ukuran & ekstensi, cegah path traversal di `resolve`. |
| [security/bcrypt_hasher.go](internal/infrastructure/security/bcrypt_hasher.go) | implementasi `PasswordHasher` memakai bcrypt, lengkap dengan penjagaan batas 72 byte. |
| [security/jwt_issuer.go](internal/infrastructure/security/jwt_issuer.go) | implementasi `TokenIssuer` memakai JWT HS256. Algoritma dikunci saat Parse untuk mencegah serangan *alg confusion*, dan token tanpa `exp` ditolak. |

### 4. `internal/delivery/http/` — pintu masuk HTTP

| File | Guna |
|---|---|
| [dto/task_dto.go](internal/delivery/http/dto/task_dto.go) | `CreateTaskRequest` / `UpdateTaskRequest` (pointer) / `TaskResponse` + konstruktornya. Mengekspos `file_url`, menyembunyikan `file_path`. |
| [dto/goals_dto.go](internal/delivery/http/dto/goals_dto.go) | DTO goals. |
| [dto/user_dto.go](internal/delivery/http/dto/user_dto.go) | DTO user. |
| [dto/auth_dto.go](internal/delivery/http/dto/auth_dto.go) | `RegisterRequest` (tanpa `status` — sengaja), `LoginRequest`, `AuthResponse` (token + `expires_at` + user). |
| [dto/swagger_dto.go](internal/delivery/http/dto/swagger_dto.go) | **khusus dokumentasi** — envelope bertipe konkret (`TaskEnvelope`, `TaskListEnvelope`, `AuthEnvelope`, `ErrorEnvelope`, dst.) yang dirujuk anotasi `@Success`/`@Failure`. Tidak dipakai kode runtime. |
| [handler/task_handler.go](internal/delivery/http/handler/task_handler.go) | handler fiber untuk task, termasuk `Download` (stream file). Rumah dari helper `parseID`. |
| [handler/goals_handler.go](internal/delivery/http/handler/goals_handler.go) | handler fiber untuk goals. |
| [handler/user_handler.go](internal/delivery/http/handler/user_handler.go) | handler fiber untuk user. |
| [handler/auth_handler.go](internal/delivery/http/handler/auth_handler.go) | handler `register` & `login`. |
| [middleware/error_handler.go](internal/delivery/http/middleware/error_handler.go) | **satu-satunya** penerjemah error → HTTP status; juga menangani error bawaan fiber (404 route tidak ada, body terlalu besar). |
| [middleware/auth.go](internal/delivery/http/middleware/auth.go) | `Auth` memeriksa header `Authorization: Bearer <token>` lalu menaruh `userID`/`userEmail`/`userRole` di `c.Locals` (*siapa pemanggilnya*); `RequireAdmin` menolak non-admin dengan 403 (*dia boleh apa*). `RequireAdmin` wajib dipasang **setelah** `Auth`. |
| [middleware/rate_limit.go](internal/delivery/http/middleware/rate_limit.go) | pembatas percobaan per IP untuk `/auth/*`, dengan bentuk body 429 yang konsisten dengan response lain. |
| [route/route.go](internal/delivery/http/route/route.go) | `/health` (sebagai fungsi `healthCheck` agar bisa dianotasi), `/swagger/*` (kalau `EnableDocs`), grup `/api/v1/auth` (tanpa proteksi, tapi ber-rate-limit), grup `/api/v1/users` (`Auth` + `RequireAdmin`), dan struct `Deps`. |

### 5. `pkg/` — shared kernel, boleh dipakai layer mana pun

| File | Guna |
|---|---|
| [apperror/apperror.go](pkg/apperror/apperror.go) | tipe `AppError` + `Kind` (`VALIDATION_ERROR`, `NOT_FOUND`, `CONFLICT`, `INTERNAL_ERROR`) dan konstruktornya. Jembatan antar layer tanpa menyebut HTTP. |
| [response/response.go](pkg/response/response.go) | bentuk JSON standar: `Success`, `SuccessWithMeta`, `Error`, `NewMeta` (menghitung `total_pages`). |
| [validator/validator.go](pkg/validator/validator.go) | bungkus `go-playground/validator`, hasilnya map `field → pesan` yang siap jadi JSON. |

### 6. Akar project

| Path | Guna |
|---|---|
| [cmd/api/main.go](cmd/api/main.go) | composition root: anotasi umum Swagger (`@title`, `@version`, `@tag.*`) → load config → set `SwaggerInfo.Host` → koneksi DB → migrasi → storage → rakit repo/usecase/handler → middleware fiber (recover, logger, cors) → `route.Register` → graceful shutdown (SIGINT/SIGTERM, timeout 10s). |
| [config/config.go](config/config.go) | baca `.env` + env var. **Satu-satunya** pemakai `os.Getenv`. Juga `DSN()` (untuk koneksi), `MaskedDSN()` (untuk log, password disembunyikan), dan `App.EnableDocs`. |
| [migrations/](migrations/) | `0001_create_tasks.sql`, `0002_create_goals.sql`, `0003_create_users.sql`, `0004_add_email_password_to_users.sql`, `0005_add_role_to_users.sql`. |
| [migrations/embed.go](migrations/embed.go) | `//go:embed` semua `.sql` ke dalam binary, jadi deploy tidak perlu menyalin folder migrasi. |
| [.env.example](.env.example) | daftar lengkap env var beserta nilai defaultnya, termasuk `ENABLE_DOCS`. |
| [docker-compose.yml](docker-compose.yml) | container PostgreSQL untuk development. |
| [uploads/](uploads/) | lampiran task tersimpan di sini (nama file diacak). Isinya di-gitignore. |
| [postman/](postman/) | koleksi + environment Postman. |
| [docs/menambah-endpoint.md](docs/menambah-endpoint.md) | versi panjang dari step-by-step di atas, lengkap dengan contoh kode utuh. |
| `docs/docs.go` | **hasil generate `swag init`** — spec yang di-embed ke binary, di-import oleh `main.go`. Jangan diedit tangan. |
| `docs/swagger.json`, `docs/swagger.yaml` | **hasil generate** — spec OpenAPI 2.0 dalam bentuk file, untuk diimpor ke Postman/Insomnia atau dipakai codegen client. |

### Tiga pemisahan yang paling sering ditanya saat review

1. **`entity.Task` vs baris tabel** — entity bebas dari detail penyimpanan. Yang tahu nama kolom
   dan bentuk SQL hanya [task_repository_sql.go](internal/infrastructure/persistence/task_repository_sql.go). Ganti database = ganti file itu saja.
2. **`entity.Task` vs `dto.TaskResponse`** — entity menyimpan `FilePath` (detail internal),
   response hanya mengekspos `file_url`.
3. **Error** — usecase melempar `apperror` ber-`Kind`; yang menerjemahkan ke HTTP status hanya
   [error_handler.go](internal/delivery/http/middleware/error_handler.go). Handler cukup `return err`.

---

## Daftar endpoint

| Method | Path | Keterangan | Token |
|---|---|---|---|
| GET | `/health` | cek service | – |
| GET | `/swagger/index.html` | Swagger UI (kalau `ENABLE_DOCS=true`) | – |
| GET | `/swagger/doc.json` | spec OpenAPI mentah | – |

### Auth

| Method | Path | Keterangan | Token |
|---|---|---|---|
| POST | `/api/v1/auth/register` | daftar akun sendiri, langsung dapat token | – |
| POST | `/api/v1/auth/login` | tukar email+password dengan token | – |

Register mengembalikan token juga, jadi client tidak perlu memanggil login lagi
setelah berhasil daftar. **Register selalu membuat `role=user`** — field `role` dan `status`
di body diabaikan, supaya tidak ada yang bisa mengangkat dirinya sendiri jadi admin.

Grup ini dibatasi **10 percobaan per menit per IP** (atur lewat `AUTH_RATE_LIMIT_MAX` dan
`AUTH_RATE_LIMIT_WINDOW`); kelebihannya dibalas `429`. Tanpa batas ini, password bisa ditebak
terus-menerus — bcrypt memperlambat tiap percobaan, tapi memperlambat bukan menghentikan.

**Login sengaja tidak membedakan "email tidak terdaftar" dari "password salah"** — keduanya
dibalas `401` dengan pesan yang sama persis, dan sama-sama menjalankan satu verifikasi bcrypt
sehingga waktu responsnya pun seragam. Kalau dibedakan, endpoint ini berubah fungsi jadi alat
untuk memeriksa email mana saja yang punya akun.

Memakai tokennya:

```bash
curl http://localhost:3000/api/v1/users \
  -H "Authorization: Bearer <token>"
```

Di Swagger UI, klik tombol **Authorize** lalu isi `Bearer <token>`.

### Tasks

| Method | Path | Keterangan |
|---|---|---|
| POST | `/api/v1/tasks` | buat task (`multipart/form-data` atau JSON) |
| GET | `/api/v1/tasks?search=&page=&limit=` | list + pencarian + paginasi |
| GET | `/api/v1/tasks/:id` | detail |
| GET | `/api/v1/tasks/:id/download` | unduh lampiran |
| PUT | `/api/v1/tasks/:id` | update sebagian field |
| DELETE | `/api/v1/tasks/:id` | hapus task + lampirannya |

Field: `no_task` (opsional — digenerate `TASK-YYYYMMDD-XXXX` kalau kosong, wajib unik),
`judul` (wajib, 3–255), `description` (opsional), `file` (opsional).

### Goals

| Method | Path | Keterangan |
|---|---|---|
| POST | `/api/v1/goals` | buat goals |
| GET | `/api/v1/goals?search=&page=&limit=` | list + pencarian + paginasi |
| GET | `/api/v1/goals/:id` | detail |
| PUT | `/api/v1/goals/:id` | update sebagian field |
| DELETE | `/api/v1/goals/:id` | hapus goals |

Field: `title` (wajib, 3–255), `category` / `priority` / `status` (wajib, maks 100),
`progress` (opsional, maks 100). Pencarian dicocokkan ke `title` dan `category`.
Goals tidak punya lampiran file, jadi request/response-nya JSON murni.

### Users

**Seluruh grup ini butuh token DENGAN role `admin`.** Pendaftaran mandiri lewat
`/api/v1/auth/register`.

| Method | Path | Keterangan | Akses |
|---|---|---|---|
| POST | `/api/v1/users` | buat user (boleh menentukan `status` & `role`) | admin |
| GET | `/api/v1/users?search=&page=&limit=` | list + pencarian + paginasi | admin |
| GET | `/api/v1/users/:id` | detail | admin |
| PUT | `/api/v1/users/:id` | update (termasuk promosi/penurunan role) | admin |
| DELETE | `/api/v1/users/:id` | hapus user | admin |

Beda `POST /users` dan `/auth/register`: register **tidak menerima** `status` maupun `role`
(dipaksa `aktif` + `user` oleh server), sedangkan endpoint ini boleh menentukan keduanya —
karena endpoint ini sendiri sudah dibatasi untuk admin.

Bedanya 401 dan 403 di sini disengaja:

| Status | Artinya |
|---|---|
| `401` | "saya tidak tahu kamu siapa" — token tidak dikirim, rusak, atau kedaluwarsa. Login dulu. |
| `403` | "saya tahu kamu siapa, tapi kamu bukan admin". Login ulang **tidak** akan menolong. |

### Membuat admin pertama

Ayam & telur: `/auth/register` selalu membuat `role=user`, dan `/api/v1/users` sendiri sudah
butuh admin. Jadi admin pertama dipromosikan manual, sekali seumur hidup database:

```sql
UPDATE users SET role = 'admin' WHERE email = 'email-anda@contoh.com';
```

Lalu **login ulang** untuk mendapat token baru. Token lama tetap membawa `role` lama sampai
kedaluwarsa — lihat catatan implementasi di bawah.

Field: `email` (wajib, format email, maks 255, **unik**), `password` (wajib, 8–72 karakter),
`name` (wajib, 3–100), `address` (wajib, maks 255), `job` / `status` (wajib, maks 100),
`role` (opsional, hanya `user` atau `admin`, default `user`).
Pencarian dicocokkan ke `name` dan `job`.

Soal email & password:

- **Password disimpan sebagai hash bcrypt**, tidak pernah dalam bentuk aslinya, dan **tidak pernah
  ikut di response** — `dto.UserResponse` memang tidak punya field itu.
- Batas `max=72` bukan angka sembarangan: bcrypt mengabaikan byte ke-73 dan seterusnya secara
  diam-diam, jadi password yang lebih panjang hanya memberi rasa aman palsu.
- **Email unik tanpa membedakan huruf besar/kecil.** `Budi@Mail.com` dan `budi@mail.com` dianggap
  orang yang sama → `409`. Email disimpan dalam huruf kecil.
- Pada `PUT`, kirim `password` **hanya kalau ingin menggantinya**. Kalau tidak dikirim, password
  lama tetap dipakai.
- Email bekas user yang sudah dihapus (soft delete) boleh dipakai lagi — unique index-nya parsial
  (`WHERE deleted_at IS NULL`).

## Contoh request

```bash
# buat task dengan lampiran
curl -X POST http://localhost:3000/api/v1/tasks \
  -F "judul=Belajar Clean Architecture" \
  -F "description=Bikin layer domain, usecase, delivery" \
  -F "file=@laporan.pdf"

# list + cari
curl "http://localhost:3000/api/v1/tasks?search=clean&page=1&limit=10"

# update hanya judul (field lain tetap)
curl -X PUT http://localhost:3000/api/v1/tasks/1 \
  -H "Content-Type: application/json" -d '{"judul":"Judul baru"}'

# ganti lampiran (file lama otomatis dihapus dari disk)
curl -X PUT http://localhost:3000/api/v1/tasks/1 -F "file=@revisi.pdf"

# lepas lampiran
curl -X PUT http://localhost:3000/api/v1/tasks/1 \
  -H "Content-Type: application/json" -d '{"remove_file":true}'
```

```bash
# buat goals
curl -X POST http://localhost:3000/api/v1/goals \
  -H "Content-Type: application/json" \
  -d '{"title":"Belajar Clean Architecture","category":"belajar","priority":"tinggi","status":"berjalan","progress":"10%"}'

# list + cari (dicocokkan ke title dan category)
curl "http://localhost:3000/api/v1/goals?search=belajar&page=1&limit=10"

# update hanya progress (field lain tetap)
curl -X PUT http://localhost:3000/api/v1/goals/1 \
  -H "Content-Type: application/json" -d '{"progress":"80%"}'

# hapus
curl -X DELETE http://localhost:3000/api/v1/goals/1
```

```bash
# daftar akun (publik) — langsung dapat token
curl -X POST http://localhost:3000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"siti@mail.com","password":"rahasia123","name":"Siti Aminah","address":"Bandung","job":"designer"}'

# login
TOKEN=$(curl -s -X POST http://localhost:3000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"siti@mail.com","password":"rahasia123"}' | jq -r .data.token)

# pakai tokennya
curl http://localhost:3000/api/v1/users -H "Authorization: Bearer $TOKEN"
```

```bash
# buat user lewat endpoint pengelolaan (butuh token, boleh set status)
curl -X POST http://localhost:3000/api/v1/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email":"budi@mail.com","password":"rahasia123","name":"Budi Santoso","address":"Jakarta","job":"backend developer","status":"aktif"}'

# list + cari (dicocokkan ke name dan job)
curl "http://localhost:3000/api/v1/users?search=budi&page=1&limit=10" \
  -H "Authorization: Bearer $TOKEN"

# ganti job saja — password lama tetap, tidak perlu dikirim ulang
curl -X PUT http://localhost:3000/api/v1/users/1 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" -d '{"job":"tech lead"}'

# ganti password
curl -X PUT http://localhost:3000/api/v1/users/1 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" -d '{"password":"passwordbaru123"}'
```

## Postman

Koleksi di folder [postman/](postman/) ditulis manual dan isinya lebih kaya daripada spec Swagger:
ada test script per request dan folder "Contoh error". Kalau cuma butuh daftar endpoint yang
selalu sinkron dengan kode, impor saja spec-nya: **Import → Link →**
`http://localhost:3000/swagger/doc.json`.

Untuk koleksi yang sudah disiapkan, import dua file ini lewat **Import** di Postman:

| File | Isi |
|---|---|
| `Daily-Task-API.postman_collection.json` | 27 request: folder Tasks, Goals, dan "Contoh error" |
| `Daily-Task-API-Local.postman_environment.json` | variable `baseUrl`, `taskId`, `goalsId` |

Pilih environment **Daily Task API - Local**, lalu jalankan request dari atas ke bawah.
Request *Create task (dengan file)* dan *Create goals — JSON* menyimpan `taskId` / `goalsId`
otomatis lewat test script, jadi request Detail / Update / Download / Delete langsung terpakai
tanpa menyalin ID manual.

Khusus request yang mengirim file, buka tab **Body → form-data**, lalu klik **Select Files**
di baris `file` untuk memilih file dari komputer Anda. Postman tidak bisa menyimpan path file
di dalam koleksi, jadi bagian ini memang harus diisi sendiri.

Setiap request sudah punya test script, jadi **Run collection** akan langsung menunjukkan
request mana yang gagal. Folder "Contoh error" sengaja berisi request yang harus gagal:
409 no_task duplikat, 400 validasi judul, 400 ekstensi file, 404 task tidak ada, 400 id bukan angka,
400 title goals terlalu pendek, 400 field goals wajib tidak dikirim, 404 goals tidak ada.

> Jangan menulis `const data = ...` di tab **Scripts**. Sandbox Postman sudah punya variabel global
> bernama `data`, dan deklarasi itu melempar `SyntaxError: Identifier 'data' has already been
> declared` — akibatnya **seluruh test script request tersebut tidak dieksekusi**, tanpa tanda apa
> pun selain hitungan assertion yang lebih sedikit. Pakai nama lain, misalnya `hasil`.

## Bentuk response

```jsonc
// sukses
{ "success": true, "message": "...", "data": { ... }, "meta": { "page":1,"limit":10,"total":2,"total_pages":1 } }

// gagal
{ "success": false, "message": "data tidak valid", "errors": { "judul": "minimal 3 karakter" } }
```

Mapping status: validasi → 400, tidak ditemukan → 404, `no_task` bentrok → 409, sisanya → 500.

## Catatan implementasi yang perlu dipahami

- **Update pakai pointer.** `UpdateTaskRequest` memakai `*string` supaya bisa membedakan
  "field tidak dikirim" (pakai nilai lama) dari "field dikirim kosong" (error validasi).
- **Urutan simpan file.** File disimpan dulu, lalu insert DB. Kalau DB gagal, file yang
  terlanjur ditulis dihapus kembali. Saat ganti file, file lama baru dihapus setelah DB sukses.
- **File tidak diserve lewat `app.Static`.** Nama file di disk diacak dan aksesnya lewat
  endpoint `/download` yang membaca path dari DB, jadi nama file tidak bisa ditebak.
  Alasan teknis tambahan: `app.Static`/`c.Download` menahan file handle di cache fasthttp,
  dan di Windows file yang handle-nya tertahan **gagal dihapus** — jadi file jadi sampah di disk.
  Karena itu `FileStorage.Open` membuka file manual lalu di-stream dengan `c.SendStream`.
- **Validasi file** (ukuran & ekstensi) ada di adapter storage, diatur lewat
  `MAX_UPLOAD_MB` dan `ALLOWED_EXT`.
- **Path traversal** dicegah di `localStorage.resolve`: path yang keluar dari folder upload ditolak.
- **Pencarian pakai `ILIKE`, bukan `LIKE`.** Di PostgreSQL `LIKE` itu case-sensitive, jadi
  `?search=postgres` tidak akan menemukan judul "Setup Database Postgres" kalau pakai `LIKE`.
- **Soft delete.** `DELETE` hanya mengisi kolom `deleted_at`; semua query membawa
  `deleted_at IS NULL`. Lampiran di disk tetap dihapus permanen oleh usecase.
- **Semua nilai lewat parameter `$1, $2, ...`**, tidak pernah dirangkai ke string query,
  jadi input pencarian tidak bisa dipakai untuk SQL injection.
- **Password tidak pernah masuk log.** `Database.DSN()` untuk koneksi, `Database.MaskedDSN()`
  untuk log dan pesan error.
- **Password user di-hash, bukan disimpan apa adanya.** Yang meng-hash adalah usecase lewat port
  `security.PasswordHasher`, bukan handler dan bukan repository — jadi tidak ada satu pun jalur
  masuk yang bisa menyimpan password polos. `entity.User.Password` selalu berisi hash, dan
  `dto.UserResponse` sengaja tidak punya field itu sama sekali.
- **Keunikan email dijaga dua lapis.** Usecase mengecek lewat `FindByEmail` supaya pesan errornya
  ramah (`409 email "x" sudah digunakan`), tetapi penjaga sebenarnya adalah unique index di
  database — tanpa itu, dua request bersamaan bisa lolos berdua.
- **Login tidak membocorkan keberadaan akun.** Email tidak terdaftar dan password salah dibalas
  `401` dengan pesan identik. Tidak cukup sampai di situ: saat email tidak ketemu, usecase tetap
  menjalankan satu verifikasi ke hash dummy, karena kalau langsung `return`, request itu balas
  jauh lebih cepat daripada email yang terdaftar (bcrypt sengaja lambat) — dan selisihnya bisa
  diukur dari luar. Ada unit test yang mengunci kedua perilaku ini.
- **Algoritma JWT dikunci saat verifikasi.** `Parse` hanya menerima `HS256`
  (`jwt.WithValidMethods`) dan mewajibkan klaim `exp`. Tanpa itu, token ber-`alg: none` bisa
  diterima mentah-mentah. Sudah diuji: token `alg=none` ditolak `401`.
- **Token tidak bisa ditarik kembali.** Itu sifat JWT, bukan bug. Konsekuensinya: logout sejati
  dan pencabutan akses butuh refresh token atau blacklist. Sementara ini yang membatasi dampak
  hanyalah `JWT_TTL`.
- **`role` ikut di dalam token, jadi token bisa basi.** Middleware membacanya dari claims supaya
  tidak perlu query database tiap request. Harganya: admin yang baru diturunkan jadi user biasa
  **masih punya hak admin sampai tokennya kedaluwarsa**, dan sebaliknya user yang baru
  dipromosikan harus login ulang dulu. Sudah diuji langsung: promosi di database tidak mengubah
  apa pun sampai login ulang. Ini alasan konkret kenapa `JWT_TTL` sebaiknya pendek. Kalau
  pencabutan harus berlaku seketika, role wajib dibaca dari database di tiap request —
  tukar satu query per request dengan kesegaran data.
- **Role default adalah yang paling sedikit haknya.** `DEFAULT 'user'` di migrasi, dan
  `normalizeRole("")` juga mengembalikan `user`. Kalau suatu saat ada jalur yang lupa mengisi
  role, kegagalannya berupa "haknya kurang", bukan "siapa pun jadi admin".
- **Nilai role dijaga dua lapis**, sama seperti keunikan email: `oneof=user admin` di DTO +
  `normalizeRole` di usecase untuk pesan error yang ramah, dan `CHECK` constraint di database
  sebagai penjaga terakhir kalau ada yang menulis langsung lewat psql.
- **`JWT_SECRET` bawaan menolak jalan di production.** Dicek saat start di `main.go`, bukan saat
  login pertama dicoba.
- **Envelope Swagger memang duplikat, dan itu disengaja.** `response.Body` memakai `Data any`
  supaya satu bentuk body melayani semua endpoint; harganya, generator OpenAPI tidak bisa menebak
  isi `data`. Alternatifnya adalah membuat `response.Body` generic, tetapi swag v1 belum bisa
  mengurai type parameter Go. Jadi `swagger_dto.go` dipilih sebagai duplikasi yang eksplisit —
  konsekuensinya, kalau bentuk body di `pkg/response` berubah, file itu harus ikut disesuaikan manual.

## Lanjutan untuk latihan

1. Tambah field `status` (todo/in-progress/done) + filter `?status=`.
   Step yang kena: 1 → 2 → 4 → 5 → 6 → 7 → 8 (port tidak berubah kalau filter ikut `TaskFilter`).
2. Tambah `deadline` dan endpoint "task hari ini".
3. Ganti `localStorage` dengan adapter MinIO/S3 — kalau arsitekturnya benar, cukup step 4 dan
   `usecase` tidak berubah sama sekali.
4. ~~Tambah JWT auth sebagai middleware~~ ✅ · ~~role/admin~~ ✅ · ~~rate limit~~ ✅.
   Lanjutannya:
   - **Relasi `user_id`** pada task & goals, supaya tiap orang hanya melihat datanya sendiri.
     `middleware.UserIDFromContext(c)` sudah tersedia untuk itu.
   - **Lindungi `/tasks` dan `/goals`** — sekarang masih publik; tinggal tambah
     `deps.AuthMiddleware` pada kedua grup di `route.go`.
   - **Refresh token** kalau butuh logout sejati dan pencabutan role yang berlaku seketika;
     perlu tabel baru.
   - **Rate limit berbasis Redis** kalau nanti jalan lebih dari satu instance — yang sekarang
     menghitung per proses, jadi batas efektifnya berlipat sebanyak jumlah instance.
5. Pakai migration tool (golang-migrate / goose) kalau nanti butuh `down` migration / rollback otomatis.
6. Tambah integration test untuk repository memakai container Postgres (testcontainers-go).
