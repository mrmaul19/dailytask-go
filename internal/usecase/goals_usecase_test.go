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

type fakeGoalsRepo struct {
	goals  map[uint]entity.Goals
	lastID uint
}

func newFakeGoalsRepo() *fakeGoalsRepo {
	return &fakeGoalsRepo{goals: map[uint]entity.Goals{}}
}

func (r *fakeGoalsRepo) Create(_ context.Context, goals *entity.Goals) error {
	r.lastID++
	goals.ID = r.lastID
	r.goals[goals.ID] = *goals
	return nil
}

func (r *fakeGoalsRepo) FindAll(_ context.Context, filter repository.GoalsFilter) ([]entity.Goals, int64, error) {
	var out []entity.Goals
	for _, g := range r.goals {
		if filter.Search == "" || strings.Contains(g.Title, filter.Search) {
			out = append(out, g)
		}
	}
	return out, int64(len(out)), nil
}

func (r *fakeGoalsRepo) FindByID(_ context.Context, id uint) (*entity.Goals, error) {
	g, ok := r.goals[id]
	if !ok {
		return nil, entity.ErrGoalsNotFound
	}
	return &g, nil
}

func (r *fakeGoalsRepo) Update(_ context.Context, goals *entity.Goals) error {
	if _, ok := r.goals[goals.ID]; !ok {
		return entity.ErrGoalsNotFound
	}
	r.goals[goals.ID] = *goals
	return nil
}

func (r *fakeGoalsRepo) Delete(_ context.Context, id uint) error {
	if _, ok := r.goals[id]; !ok {
		return entity.ErrGoalsNotFound
	}
	delete(r.goals, id)
	return nil
}

// ---------- helper ----------

func setupGoals() (usecase.GoalsUsecase, *fakeGoalsRepo) {
	repo := newFakeGoalsRepo()
	return usecase.NewGoalsUsecase(repo), repo
}

func seedGoals(t *testing.T, uc usecase.GoalsUsecase) *entity.Goals {
	t.Helper()
	goals, err := uc.Create(context.Background(), usecase.CreateGoalsInput{
		Title:    "Lulus sertifikasi Go",
		Category: "karier",
		Priority: "tinggi",
		Status:   "berjalan",
		Progress: "20%",
	})
	if err != nil {
		t.Fatalf("setup gagal: %v", err)
	}
	return goals
}

func ptr(s string) *string { return &s }

// ---------- test ----------

func TestGoalsCreate_MenyimpanDanMengisiID(t *testing.T) {
	uc, repo := setupGoals()

	goals := seedGoals(t, uc)

	if goals.ID == 0 {
		t.Error("id seharusnya terisi setelah create")
	}
	if len(repo.goals) != 1 {
		t.Errorf("jumlah goals tersimpan = %d, ingin 1", len(repo.goals))
	}
}

func TestGoalsCreate_SpasiDipangkas(t *testing.T) {
	uc, _ := setupGoals()

	goals, err := uc.Create(context.Background(), usecase.CreateGoalsInput{
		Title:    "  Olahraga rutin  ",
		Category: "  kesehatan  ",
	})
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if goals.Title != "Olahraga rutin" {
		t.Errorf("title = %q, ingin %q", goals.Title, "Olahraga rutin")
	}
	if goals.Category != "kesehatan" {
		t.Errorf("category = %q, ingin %q", goals.Category, "kesehatan")
	}
}

func TestGoalsUpdate_HanyaFieldYangDikirimYangBerubah(t *testing.T) {
	uc, _ := setupGoals()
	ctx := context.Background()

	goals := seedGoals(t, uc)

	updated, err := uc.Update(ctx, goals.ID, usecase.UpdateGoalsInput{
		Progress: ptr("75%"),
	})
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if updated.Progress != "75%" {
		t.Errorf("progress = %q, ingin %q", updated.Progress, "75%")
	}
	if updated.Title != goals.Title {
		t.Errorf("title ikut berubah jadi %q, seharusnya tetap %q", updated.Title, goals.Title)
	}
	if updated.Status != goals.Status {
		t.Errorf("status ikut berubah jadi %q, seharusnya tetap %q", updated.Status, goals.Status)
	}
}

func TestGoalsUpdate_FieldDikirimKosongDitolak(t *testing.T) {
	uc, _ := setupGoals()
	ctx := context.Background()

	goals := seedGoals(t, uc)

	_, err := uc.Update(ctx, goals.ID, usecase.UpdateGoalsInput{Title: ptr("   ")})
	if err == nil {
		t.Fatal("ingin error validasi, dapat nil")
	}
	if got := kindOf(t, err); got != apperror.KindValidation {
		t.Errorf("kind = %s, ingin %s", got, apperror.KindValidation)
	}
}

func TestGoalsUpdate_TidakDitemukan(t *testing.T) {
	uc, _ := setupGoals()

	_, err := uc.Update(context.Background(), 404, usecase.UpdateGoalsInput{Title: ptr("apa saja")})
	if err == nil {
		t.Fatal("ingin error not found, dapat nil")
	}
	if got := kindOf(t, err); got != apperror.KindNotFound {
		t.Errorf("kind = %s, ingin %s", got, apperror.KindNotFound)
	}
}

func TestGoalsGetByID_TidakDitemukan(t *testing.T) {
	uc, _ := setupGoals()

	_, err := uc.GetByID(context.Background(), 404)
	if err == nil {
		t.Fatal("ingin error not found, dapat nil")
	}
	if got := kindOf(t, err); got != apperror.KindNotFound {
		t.Errorf("kind = %s, ingin %s", got, apperror.KindNotFound)
	}
}

func TestGoalsDelete_MenghapusDariRepo(t *testing.T) {
	uc, repo := setupGoals()
	ctx := context.Background()

	goals := seedGoals(t, uc)

	if err := uc.Delete(ctx, goals.ID); err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if len(repo.goals) != 0 {
		t.Errorf("jumlah goals tersisa = %d, ingin 0", len(repo.goals))
	}

	if err := uc.Delete(ctx, goals.ID); err == nil {
		t.Fatal("hapus dua kali seharusnya error not found")
	}
}

func TestGoalsList_PagingDinormalisasi(t *testing.T) {
	uc, _ := setupGoals()

	out, err := uc.List(context.Background(), usecase.ListGoalsInput{Page: 0, Limit: 0})
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if out.Page != 1 || out.Limit != 10 {
		t.Errorf("page=%d limit=%d, ingin page=1 limit=10", out.Page, out.Limit)
	}
}

func TestGoalsList_LimitDibatasiMaksimum(t *testing.T) {
	uc, _ := setupGoals()

	out, err := uc.List(context.Background(), usecase.ListGoalsInput{Page: 2, Limit: 9999})
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if out.Limit != 100 {
		t.Errorf("limit = %d, ingin dibatasi ke 100", out.Limit)
	}
}
