package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"daily-task-go/internal/domain/entity"
	"daily-task-go/internal/domain/repository"
	"daily-task-go/internal/domain/security"
	"daily-task-go/pkg/apperror"
)

// ---------- Input / Output (bentuk data antar-layer, bebas dari HTTP) ----------

type CreateUserInput struct {
	Email string
	// Password berisi password POLOS dari client. Di-hash di usecase ini,
	// dan tidak pernah disimpan maupun dikembalikan dalam bentuk aslinya.
	Password string
	Name     string
	Address  string
	Job      string
	Status   string
	// Role boleh kosong — diisi entity.RoleUser. Default yang aman adalah
	// yang paling sedikit haknya.
	Role string
}

type UpdateUserInput struct {
	Email *string
	// Password polos. nil = jangan ganti password.
	Password *string
	Name     *string
	Address  *string
	Job      *string
	Status   *string
	Role     *string
}

type ListUserInput struct {
	Search string
	Page   int
	Limit  int
}

type ListUserOutput struct {
	Users []entity.User
	Total int64
	Page  int
	Limit int
}

// ---------- Kontrak usecase ----------

type UserUsecase interface {
	Create(ctx context.Context, in CreateUserInput) (*entity.User, error)
	List(ctx context.Context, in ListUserInput) (*ListUserOutput, error)
	GetByID(ctx context.Context, id uint) (*entity.User, error)
	Update(ctx context.Context, id uint, in UpdateUserInput) (*entity.User, error)
	Delete(ctx context.Context, id uint) error
}

type userUsecase struct {
	userRepo repository.UserRepository
	hasher   security.PasswordHasher
}

// NewUserUsecase menerima port lewat constructor (dependency injection),
// sehingga mudah diganti mock saat testing.
func NewUserUsecase(userRepo repository.UserRepository, hasher security.PasswordHasher) UserUsecase {
	return &userUsecase{userRepo: userRepo, hasher: hasher}
}

func (u *userUsecase) Create(ctx context.Context, in CreateUserInput) (*entity.User, error) {
	email := entity.NormalizeEmail(in.Email)
	if err := u.ensureEmailAvailable(ctx, email, 0); err != nil {
		return nil, err
	}

	hashed, err := u.hashPassword(in.Password)
	if err != nil {
		return nil, err
	}

	role, err := normalizeRole(in.Role)
	if err != nil {
		return nil, err
	}

	user := &entity.User{
		Email:    email,
		Password: hashed,
		Name:     strings.TrimSpace(in.Name),
		Address:  strings.TrimSpace(in.Address),
		Job:      strings.TrimSpace(in.Job),
		Status:   strings.TrimSpace(in.Status),
		Role:     role,
	}

	if err := u.userRepo.Create(ctx, user); err != nil {
		return nil, apperror.Internal("gagal menyimpan user", err)
	}

	return user, nil
}

func (u *userUsecase) List(ctx context.Context, in ListUserInput) (*ListUserOutput, error) {
	page, limit := normalizePaging(in.Page, in.Limit)

	users, total, err := u.userRepo.FindAll(ctx, repository.UserFilter{
		Search: strings.TrimSpace(in.Search),
		Page:   page,
		Limit:  limit,
	})
	if err != nil {
		return nil, apperror.Internal("gagal mengambil daftar user", err)
	}

	return &ListUserOutput{Users: users, Total: total, Page: page, Limit: limit}, nil
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

func (u *userUsecase) Update(ctx context.Context, id uint, in UpdateUserInput) (*entity.User, error) {
	user, err := u.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Pointer dipakai supaya "field tidak dikirim" (biarkan nilai lama)
	// bisa dibedakan dari "field dikirim kosong" (ditolak).
	fields := []struct {
		name  string
		input *string
		dest  *string
	}{
		{"name", in.Name, &user.Name},
		{"address", in.Address, &user.Address},
		{"job", in.Job, &user.Job},
		{"status", in.Status, &user.Status},
	}

	for _, f := range fields {
		if f.input == nil {
			continue
		}
		value := strings.TrimSpace(*f.input)
		if value == "" {
			return nil, apperror.Validation("data tidak valid",
				map[string]string{f.name: "tidak boleh kosong"})
		}
		*f.dest = value
	}

	// Email diperlakukan terpisah karena perlu normalisasi + cek keunikan.
	if in.Email != nil {
		email := entity.NormalizeEmail(*in.Email)
		if email == "" {
			return nil, apperror.Validation("data tidak valid",
				map[string]string{"email": "tidak boleh kosong"})
		}
		if err := u.ensureEmailAvailable(ctx, email, user.ID); err != nil {
			return nil, err
		}
		user.Email = email
	}

	// Role terpisah karena nilainya terbatas, bukan teks bebas.
	if in.Role != nil {
		role, err := normalizeRole(*in.Role)
		if err != nil {
			return nil, err
		}
		user.Role = role
	}

	// Password terpisah karena harus di-hash, dan sengaja TIDAK di-TrimSpace:
	// spasi di awal/akhir adalah bagian sah dari sebuah password.
	if in.Password != nil {
		hashed, err := u.hashPassword(*in.Password)
		if err != nil {
			return nil, err
		}
		user.Password = hashed
	}

	if err := u.userRepo.Update(ctx, user); err != nil {
		return nil, apperror.Internal("gagal memperbarui user", err)
	}

	return user, nil
}

func (u *userUsecase) Delete(ctx context.Context, id uint) error {
	user, err := u.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := u.userRepo.Delete(ctx, user.ID); err != nil {
		return apperror.Internal("gagal menghapus user", err)
	}

	return nil
}

// ensureEmailAvailable menjaga invariant: email harus unik.
// exceptID dipakai saat update, supaya user tidak dianggap bentrok dengan
// dirinya sendiri ketika mengirim email yang sama dengan miliknya sekarang.
//
// Pemeriksaan di sini bukan jaminan mutlak — dua request bersamaan masih bisa
// lolos keduanya. Penjaga terakhirnya adalah unique index di database
// (users_email_unique_idx); yang kedua lolos akan gagal di repository.
func (u *userUsecase) ensureEmailAvailable(ctx context.Context, email string, exceptID uint) error {
	existing, err := u.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, entity.ErrUserNotFound) {
			return nil
		}
		return apperror.Internal("gagal memeriksa email", err)
	}
	if existing.ID == exceptID {
		return nil
	}
	return apperror.Conflict(fmt.Sprintf("email %q sudah digunakan", email))
}

// normalizeRole memberi nilai bawaan yang aman dan menolak nilai asing.
//
// Role kosong menjadi entity.RoleUser — bukan admin. Kalau suatu saat ada
// jalur yang lupa mengisi role, kegagalannya berupa "haknya terlalu sedikit",
// bukan "siapa pun jadi admin".
func normalizeRole(raw string) (string, error) {
	role := strings.ToLower(strings.TrimSpace(raw))
	if role == "" {
		return entity.RoleUser, nil
	}
	if !entity.IsValidRole(role) {
		return "", apperror.Validation("data tidak valid",
			map[string]string{"role": fmt.Sprintf("hanya boleh %q atau %q", entity.RoleUser, entity.RoleAdmin)})
	}
	return role, nil
}

// hashPassword membungkus port hasher supaya kegagalannya menjadi apperror
// yang seragam. Pesan errornya sengaja tidak memuat password apa pun.
func (u *userUsecase) hashPassword(plain string) (string, error) {
	if plain == "" {
		return "", apperror.Validation("data tidak valid",
			map[string]string{"password": "tidak boleh kosong"})
	}

	hashed, err := u.hasher.Hash(plain)
	if err != nil {
		return "", apperror.Internal("gagal mengamankan password", err)
	}
	return hashed, nil
}
