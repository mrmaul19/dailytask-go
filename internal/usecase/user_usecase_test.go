package usecase_test

import (
	"context"
	"strings"
	"testing"

	"daily-task-go/internal/domain/entity"
	"daily-task-go/internal/domain/repository"
	"daily-task-go/internal/usecase"
	"daily-task-go/pkg/apperror"
)

// ---------- fake repository (in-memory, tanpa database) ----------

type fakeUserRepo struct {
	users  map[uint]entity.User
	lastID uint
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{users: map[uint]entity.User{}}
}

func (r *fakeUserRepo) Create(_ context.Context, user *entity.User) error {
	r.lastID++
	user.ID = r.lastID
	r.users[user.ID] = *user
	return nil
}

func (r *fakeUserRepo) FindAll(_ context.Context, filter repository.UserFilter) ([]entity.User, int64, error) {
	var out []entity.User
	for _, u := range r.users {
		if filter.Search == "" || strings.Contains(u.Name, filter.Search) {
			out = append(out, u)
		}
	}
	return out, int64(len(out)), nil
}

func (r *fakeUserRepo) FindByID(_ context.Context, id uint) (*entity.User, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, entity.ErrUserNotFound
	}
	return &u, nil
}

func (r *fakeUserRepo) Update(_ context.Context, user *entity.User) error {
	if _, ok := r.users[user.ID]; !ok {
		return entity.ErrUserNotFound
	}
	r.users[user.ID] = *user
	return nil
}

func (r *fakeUserRepo) Delete(_ context.Context, id uint) error {
	if _, ok := r.users[id]; !ok {
		return entity.ErrUserNotFound
	}
	delete(r.users, id)
	return nil
}

// ---------- helper ----------

func setupUser() (usecase.UserUsecase, *fakeUserRepo) {
	repo := newFakeUserRepo()
	return usecase.NewUserUsecase(repo), repo
}

func seedUser(t *testing.T, uc usecase.UserUsecase) *entity.User {
	t.Helper()
	user, err := uc.Create(context.Background(), usecase.CreateUserInput{
		Name:    "Budi Santoso",
		Address: "Jl. Merdeka 1",
		Job:     "Backend Engineer",
		Status:  "aktif",
	})
	if err != nil {
		t.Fatalf("setup gagal: %v", err)
	}
	return user
}

// ---------- test ----------

func TestUserCreate_MenyimpanDanMengisiID(t *testing.T) {
	uc, repo := setupUser()

	user := seedUser(t, uc)

	if user.ID == 0 {
		t.Error("id seharusnya terisi setelah create")
	}
	if len(repo.users) != 1 {
		t.Errorf("jumlah user tersimpan = %d, ingin 1", len(repo.users))
	}
}

func TestUserCreate_SpasiDipangkas(t *testing.T) {
	uc, _ := setupUser()

	user, err := uc.Create(context.Background(), usecase.CreateUserInput{
		Name:    "  Siti Aminah  ",
		Address: "  Jl. Mawar 9  ",
		Job:     "  Designer  ",
		Status:  "  aktif  ",
	})
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if user.Name != "Siti Aminah" {
		t.Errorf("name = %q, ingin %q", user.Name, "Siti Aminah")
	}
	if user.Job != "Designer" {
		t.Errorf("job = %q, ingin %q", user.Job, "Designer")
	}
}

func TestUserUpdate_HanyaFieldYangDikirimYangBerubah(t *testing.T) {
	uc, _ := setupUser()
	ctx := context.Background()

	user := seedUser(t, uc)

	updated, err := uc.Update(ctx, user.ID, usecase.UpdateUserInput{
		Job: ptr("Tech Lead"),
	})
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if updated.Job != "Tech Lead" {
		t.Errorf("job = %q, ingin %q", updated.Job, "Tech Lead")
	}
	if updated.Name != user.Name {
		t.Errorf("name ikut berubah jadi %q, seharusnya tetap %q", updated.Name, user.Name)
	}
	if updated.Address != user.Address {
		t.Errorf("address ikut berubah jadi %q, seharusnya tetap %q", updated.Address, user.Address)
	}
	if updated.Status != user.Status {
		t.Errorf("status ikut berubah jadi %q, seharusnya tetap %q", updated.Status, user.Status)
	}
}

func TestUserUpdate_FieldDikirimKosongDitolak(t *testing.T) {
	uc, _ := setupUser()
	ctx := context.Background()

	user := seedUser(t, uc)

	_, err := uc.Update(ctx, user.ID, usecase.UpdateUserInput{Name: ptr("   ")})
	if err == nil {
		t.Fatal("ingin error validasi, dapat nil")
	}
	if got := kindOf(t, err); got != apperror.KindValidation {
		t.Errorf("kind = %s, ingin %s", got, apperror.KindValidation)
	}
}

func TestUserUpdate_TidakDitemukan(t *testing.T) {
	uc, _ := setupUser()

	_, err := uc.Update(context.Background(), 404, usecase.UpdateUserInput{Name: ptr("apa saja")})
	if err == nil {
		t.Fatal("ingin error not found, dapat nil")
	}
	if got := kindOf(t, err); got != apperror.KindNotFound {
		t.Errorf("kind = %s, ingin %s", got, apperror.KindNotFound)
	}
}

func TestUserGetByID_TidakDitemukan(t *testing.T) {
	uc, _ := setupUser()

	_, err := uc.GetByID(context.Background(), 404)
	if err == nil {
		t.Fatal("ingin error not found, dapat nil")
	}
	if got := kindOf(t, err); got != apperror.KindNotFound {
		t.Errorf("kind = %s, ingin %s", got, apperror.KindNotFound)
	}
}

func TestUserDelete_MenghapusDariRepo(t *testing.T) {
	uc, repo := setupUser()
	ctx := context.Background()

	user := seedUser(t, uc)

	if err := uc.Delete(ctx, user.ID); err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if len(repo.users) != 0 {
		t.Errorf("jumlah user tersisa = %d, ingin 0", len(repo.users))
	}

	if err := uc.Delete(ctx, user.ID); err == nil {
		t.Fatal("hapus dua kali seharusnya error not found")
	}
}

func TestUserList_PagingDinormalisasi(t *testing.T) {
	uc, _ := setupUser()

	out, err := uc.List(context.Background(), usecase.ListUserInput{Page: 0, Limit: 0})
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if out.Page != 1 || out.Limit != 10 {
		t.Errorf("page=%d limit=%d, ingin page=1 limit=10", out.Page, out.Limit)
	}
}

func TestUserList_LimitDibatasiMaksimum(t *testing.T) {
	uc, _ := setupUser()

	out, err := uc.List(context.Background(), usecase.ListUserInput{Page: 2, Limit: 9999})
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if out.Limit != 100 {
		t.Errorf("limit = %d, ingin dibatasi ke 100", out.Limit)
	}
}
