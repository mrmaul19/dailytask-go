package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"

	"daily-task-go/internal/domain/entity"
	"daily-task-go/internal/domain/repository"
	"daily-task-go/internal/domain/security"
	"daily-task-go/pkg/apperror"
)

// pesanKredensialSalah sengaja SATU pesan untuk dua sebab berbeda:
// email tidak terdaftar, dan password salah.
//
// Kalau dibedakan, endpoint login berubah fungsi menjadi alat pemeriksa
// "email ini punya akun atau tidak" — siapa pun bisa memetakan daftar
// pengguna tanpa perlu menebak satu password pun.
const pesanKredensialSalah = "email atau password salah"

// ---------- Input / Output ----------

// RegisterInput sengaja TIDAK punya Status maupun Role.
//
// Role terutama: kalau field itu ada di sini, siapa pun cukup mengirim
// "role":"admin" saat mendaftar untuk mendapat hak penuh. Keduanya dipaksa
// oleh server di bawah. Bandingkan dengan CreateUserInput yang dipakai
// endpoint pengelolaan user — di sana keduanya memang boleh ditentukan,
// karena endpoint itu sendiri sudah dibatasi untuk admin.
type RegisterInput struct {
	Email    string
	Password string
	Name     string
	Address  string
	Job      string
}

type LoginInput struct {
	Email    string
	Password string
}

type AuthOutput struct {
	User  *entity.User
	Token security.Token
}

// ---------- Kontrak usecase ----------

type AuthUsecase interface {
	Register(ctx context.Context, in RegisterInput) (*AuthOutput, error)
	Login(ctx context.Context, in LoginInput) (*AuthOutput, error)
}

type authUsecase struct {
	users    UserUsecase
	userRepo repository.UserRepository
	hasher   security.PasswordHasher
	tokens   security.TokenIssuer

	// dummyHash dipakai untuk meratakan waktu respons saat email tidak
	// ditemukan. Lihat penjelasan di Login.
	dummyHash string
}

func NewAuthUsecase(
	users UserUsecase,
	userRepo repository.UserRepository,
	hasher security.PasswordHasher,
	tokens security.TokenIssuer,
) AuthUsecase {
	return &authUsecase{
		users:     users,
		userRepo:  userRepo,
		hasher:    hasher,
		tokens:    tokens,
		dummyHash: buatDummyHash(hasher),
	}
}

// Register memakai ulang UserUsecase.Create, bukan menulis ulang alurnya.
// Dengan begitu normalisasi email, pemeriksaan email unik, dan hashing
// password hanya hidup di satu tempat.
func (a *authUsecase) Register(ctx context.Context, in RegisterInput) (*AuthOutput, error) {
	user, err := a.users.Create(ctx, CreateUserInput{
		Email:    in.Email,
		Password: in.Password,
		Name:     in.Name,
		Address:  in.Address,
		Job:      in.Job,
		// Keduanya dipaksa server, tidak pernah datang dari request.
		Status: entity.StatusAktif,
		Role:   entity.RoleUser,
	})
	if err != nil {
		// Termasuk apperror.Conflict kalau email sudah dipakai -> 409.
		return nil, err
	}

	return a.terbitkanToken(user)
}

func (a *authUsecase) Login(ctx context.Context, in LoginInput) (*AuthOutput, error) {
	email := entity.NormalizeEmail(in.Email)

	user, err := a.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, entity.ErrUserNotFound) {
			// Tetap jalankan satu verifikasi ke hash palsu sebelum menolak.
			//
			// Tanpa ini, email yang tidak terdaftar dibalas seketika, sedangkan
			// email terdaftar harus menunggu bcrypt yang sengaja lambat.
			// Selisih waktunya konsisten dan bisa diukur dari luar, sehingga
			// keberadaan sebuah akun tetap bocor walau pesan errornya sama.
			_ = a.hasher.Verify(a.dummyHash, in.Password)
			return nil, apperror.Unauthorized(pesanKredensialSalah)
		}
		return nil, apperror.Internal("gagal memeriksa kredensial", err)
	}

	if err := a.hasher.Verify(user.Password, in.Password); err != nil {
		// Sebab aslinya tidak diteruskan: password salah dan hash rusak
		// (misalnya baris user lama yang password-nya masih kosong)
		// harus terlihat sama dari luar.
		return nil, apperror.Unauthorized(pesanKredensialSalah)
	}

	return a.terbitkanToken(user)
}

func (a *authUsecase) terbitkanToken(user *entity.User) (*AuthOutput, error) {
	token, err := a.tokens.Issue(security.Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
	})
	if err != nil {
		return nil, apperror.Internal("gagal menerbitkan token", err)
	}
	return &AuthOutput{User: user, Token: token}, nil
}

// buatDummyHash menghasilkan satu hash dari nilai acak saat aplikasi start.
// Nilainya tidak pernah cocok dengan password mana pun, dan dibuat lewat port
// hasher supaya usecase tetap tidak tahu-menahu soal bcrypt.
func buatDummyHash(hasher security.PasswordHasher) string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	hashed, err := hasher.Hash(hex.EncodeToString(buf))
	if err != nil {
		return ""
	}
	return hashed
}
