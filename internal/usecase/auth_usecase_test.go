package usecase_test

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"daily-task-go/internal/domain/entity"
	"daily-task-go/internal/domain/security"
	"daily-task-go/internal/usecase"
	"daily-task-go/pkg/apperror"
)

// ---------- fake token issuer ----------

// fakeTokenIssuer menghindari JWT asli di unit test: yang diuji di lapisan ini
// adalah "apakah token diterbitkan untuk user yang benar", bukan benar-tidaknya
// tanda tangan kriptografis — itu urusan adapter infrastructure.
type fakeTokenIssuer struct {
	issued []security.Claims
	err    error
}

func (f *fakeTokenIssuer) Issue(claims security.Claims) (security.Token, error) {
	if f.err != nil {
		return security.Token{}, f.err
	}
	f.issued = append(f.issued, claims)
	return security.Token{
		Value:     "token-untuk-" + strconv.FormatUint(uint64(claims.UserID), 10),
		ExpiresAt: time.Now().Add(time.Hour),
	}, nil
}

func (f *fakeTokenIssuer) Parse(string) (*security.Claims, error) {
	return nil, security.ErrTokenTidakValid
}

// ---------- helper ----------

func setupAuth() (usecase.AuthUsecase, *fakeUserRepo, *fakeTokenIssuer, *fakeHasher) {
	repo := newFakeUserRepo()
	hasher := &fakeHasher{}
	tokens := &fakeTokenIssuer{}
	userUC := usecase.NewUserUsecase(repo, hasher)
	return usecase.NewAuthUsecase(userUC, repo, hasher, tokens), repo, tokens, hasher
}

func registerContoh(t *testing.T, uc usecase.AuthUsecase) *usecase.AuthOutput {
	t.Helper()
	out, err := uc.Register(context.Background(), usecase.RegisterInput{
		Email:    "budi@mail.com",
		Password: "rahasia123",
		Name:     "Budi Santoso",
		Address:  "Jl. Merdeka 1",
		Job:      "Backend Engineer",
	})
	if err != nil {
		t.Fatalf("setup gagal: %v", err)
	}
	return out
}

// ---------- register ----------

func TestAuthRegister_MengembalikanTokenDanUser(t *testing.T) {
	uc, _, tokens, _ := setupAuth()

	out := registerContoh(t, uc)

	if out.Token.Value == "" {
		t.Error("token kosong setelah register")
	}
	if out.User.ID == 0 {
		t.Error("id user tidak terisi")
	}
	if len(tokens.issued) != 1 {
		t.Fatalf("jumlah token terbit = %d, ingin 1", len(tokens.issued))
	}
	if tokens.issued[0].UserID != out.User.ID {
		t.Errorf("token diterbitkan untuk user %d, ingin %d", tokens.issued[0].UserID, out.User.ID)
	}
}

func TestAuthRegister_StatusDipaksaAktif(t *testing.T) {
	uc, _, _, _ := setupAuth()

	out := registerContoh(t, uc)

	// RegisterInput memang tidak punya field Status — test ini mengunci
	// perilakunya supaya tidak ada yang diam-diam menambahkannya nanti.
	if out.User.Status != "aktif" {
		t.Errorf("status = %q, ingin %q", out.User.Status, "aktif")
	}
}

// Pendaftar tidak boleh bisa menjadikan dirinya admin. RegisterInput memang
// tidak punya field Role — test ini mengunci perilakunya supaya tidak ada
// yang diam-diam menambahkannya nanti.
func TestAuthRegister_RoleDipaksaUserBiasa(t *testing.T) {
	uc, _, tokens, _ := setupAuth()

	out := registerContoh(t, uc)

	if out.User.Role != entity.RoleUser {
		t.Errorf("role = %q, ingin %q", out.User.Role, entity.RoleUser)
	}
	if out.User.IsAdmin() {
		t.Error("pendaftar baru tidak boleh langsung admin")
	}
	if tokens.issued[0].Role != entity.RoleUser {
		t.Errorf("role di token = %q, ingin %q", tokens.issued[0].Role, entity.RoleUser)
	}
}

func TestAuthLogin_RoleIkutMasukToken(t *testing.T) {
	uc, repo, tokens, _ := setupAuth()
	ctx := context.Background()

	terdaftar := registerContoh(t, uc)

	// Promosikan jadi admin langsung di repo, meniru UPDATE manual di database.
	promoted := repo.users[terdaftar.User.ID]
	promoted.Role = entity.RoleAdmin
	repo.users[promoted.ID] = promoted

	if _, err := uc.Login(ctx, usecase.LoginInput{
		Email:    "budi@mail.com",
		Password: "rahasia123",
	}); err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}

	terakhir := tokens.issued[len(tokens.issued)-1]
	if terakhir.Role != entity.RoleAdmin {
		t.Errorf("role di token = %q, ingin %q", terakhir.Role, entity.RoleAdmin)
	}
}

func TestAuthRegister_PasswordDisimpanSebagaiHash(t *testing.T) {
	uc, _, _, _ := setupAuth()

	out := registerContoh(t, uc)

	if out.User.Password == "rahasia123" {
		t.Fatal("password polos tersimpan — wajib di-hash")
	}
}

func TestAuthRegister_EmailDuplikatDitolak(t *testing.T) {
	uc, _, _, _ := setupAuth()

	registerContoh(t, uc)

	_, err := uc.Register(context.Background(), usecase.RegisterInput{
		Email:    "BUDI@MAIL.COM", // beda huruf besar-kecil, tetap orang yang sama
		Password: "rahasia123",
		Name:     "Budi Kembar",
		Address:  "Jl. Lain 2",
		Job:      "QA",
	})
	if err == nil {
		t.Fatal("ingin error conflict, dapat nil")
	}
	if got := kindOf(t, err); got != apperror.KindConflict {
		t.Errorf("kind = %s, ingin %s", got, apperror.KindConflict)
	}
}

// ---------- login ----------

func TestAuthLogin_KredensialBenar(t *testing.T) {
	uc, _, tokens, _ := setupAuth()
	ctx := context.Background()

	terdaftar := registerContoh(t, uc)

	out, err := uc.Login(ctx, usecase.LoginInput{
		Email:    "budi@mail.com",
		Password: "rahasia123",
	})
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if out.User.ID != terdaftar.User.ID {
		t.Errorf("login sebagai user %d, ingin %d", out.User.ID, terdaftar.User.ID)
	}
	if len(tokens.issued) != 2 { // 1 dari register, 1 dari login
		t.Errorf("jumlah token terbit = %d, ingin 2", len(tokens.issued))
	}
}

func TestAuthLogin_EmailBolehBedaBesarKecil(t *testing.T) {
	uc, _, _, _ := setupAuth()

	registerContoh(t, uc)

	if _, err := uc.Login(context.Background(), usecase.LoginInput{
		Email:    "  BUDI@Mail.COM  ",
		Password: "rahasia123",
	}); err != nil {
		t.Fatalf("email seharusnya dinormalisasi sebelum dicari: %v", err)
	}
}

func TestAuthLogin_PasswordSalah(t *testing.T) {
	uc, _, _, _ := setupAuth()

	registerContoh(t, uc)

	_, err := uc.Login(context.Background(), usecase.LoginInput{
		Email:    "budi@mail.com",
		Password: "passwordsalah",
	})
	if err == nil {
		t.Fatal("ingin error unauthorized, dapat nil")
	}
	if got := kindOf(t, err); got != apperror.KindUnauthorized {
		t.Errorf("kind = %s, ingin %s", got, apperror.KindUnauthorized)
	}
}

// Test paling penting di berkas ini: email tak dikenal dan password salah
// harus menghasilkan respons yang TIDAK bisa dibedakan. Kalau yang satu 404
// dan yang lain 401, endpoint login berubah jadi alat untuk memeriksa
// email mana saja yang punya akun.
func TestAuthLogin_EmailTidakDikenalTidakBisaDibedakanDariPasswordSalah(t *testing.T) {
	uc, _, _, _ := setupAuth()
	ctx := context.Background()

	registerContoh(t, uc)

	_, errPasswordSalah := uc.Login(ctx, usecase.LoginInput{
		Email:    "budi@mail.com",
		Password: "passwordsalah",
	})
	_, errEmailAsing := uc.Login(ctx, usecase.LoginInput{
		Email:    "tidakada@mail.com",
		Password: "rahasia123",
	})

	if errPasswordSalah == nil || errEmailAsing == nil {
		t.Fatal("dua-duanya seharusnya error")
	}
	if kindOf(t, errEmailAsing) != apperror.KindUnauthorized {
		t.Errorf("email asing kind = %s, ingin %s", kindOf(t, errEmailAsing), apperror.KindUnauthorized)
	}
	if errPasswordSalah.Error() != errEmailAsing.Error() {
		t.Errorf("pesan error berbeda:\n  password salah: %q\n  email asing   : %q",
			errPasswordSalah.Error(), errEmailAsing.Error())
	}
}

// Saat email tidak ditemukan, usecase tetap harus menjalankan satu verifikasi
// ke hash dummy. Tanpa itu, request untuk email yang tidak terdaftar dibalas
// jauh lebih cepat, dan selisih waktunya membocorkan keberadaan akun.
func TestAuthLogin_EmailTidakDikenalTetapMenjalankanVerify(t *testing.T) {
	uc, _, _, hasher := setupAuth()

	sebelum := hasher.verifyCalls
	_, _ = uc.Login(context.Background(), usecase.LoginInput{
		Email:    "tidakada@mail.com",
		Password: "apa saja",
	})

	if hasher.verifyCalls != sebelum+1 {
		t.Errorf("Verify dipanggil %d kali, ingin %d — waktu respons jadi membocorkan keberadaan akun",
			hasher.verifyCalls-sebelum, 1)
	}
}

func TestAuthLogin_GagalMenerbitkanTokenJadiInternalError(t *testing.T) {
	repo := newFakeUserRepo()
	hasher := &fakeHasher{}
	tokens := &fakeTokenIssuer{}
	userUC := usecase.NewUserUsecase(repo, hasher)
	uc := usecase.NewAuthUsecase(userUC, repo, hasher, tokens)

	registerContoh(t, uc)

	tokens.err = errors.New("kunci tidak terbaca")

	_, err := uc.Login(context.Background(), usecase.LoginInput{
		Email:    "budi@mail.com",
		Password: "rahasia123",
	})
	if err == nil {
		t.Fatal("ingin error, dapat nil")
	}
	if got := kindOf(t, err); got != apperror.KindInternal {
		t.Errorf("kind = %s, ingin %s", got, apperror.KindInternal)
	}
}
