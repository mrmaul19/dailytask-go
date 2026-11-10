package usecase_test

import (
	"context"
	"testing"
	"time"

	"daily-task-go/internal/domain/entity"
	"daily-task-go/internal/domain/repository"
	"daily-task-go/internal/usecase"
	"daily-task-go/pkg/apperror"
)

type fakeTimeEntryRepo struct {
	entries                    map[uint]entity.TimeEntry
	lastID                     uint
	tolakCreateKarenaIndexUnik bool
}

func newFakeTimeEntryRepo() *fakeTimeEntryRepo {
	return &fakeTimeEntryRepo{entries: map[uint]entity.TimeEntry{}}
}

func (r *fakeTimeEntryRepo) Create(_ context.Context, entry *entity.TimeEntry) error {
	if r.tolakCreateKarenaIndexUnik {
		return entity.ErrTimerSedangBerjalan
	}
	for _, e := range r.entries {
		if e.UserID == entry.UserID && e.EndedAt == nil {
			return entity.ErrTimerSedangBerjalan
		}
	}

	r.lastID++
	entry.ID = r.lastID
	entry.CreatedAt = time.Now()
	entry.UpdatedAt = time.Now()
	r.entries[entry.ID] = *entry
	return nil
}

func (r *fakeTimeEntryRepo) FindAll(_ context.Context, filter repository.TimeEntryFilter) ([]entity.TimeEntry, int64, error) {
	var out []entity.TimeEntry
	for _, e := range r.entries {
		if filter.UserID != 0 && e.UserID != filter.UserID {
			continue
		}
		if filter.TaskID != 0 && e.TaskID != filter.TaskID {
			continue
		}
		if filter.From != nil && e.StartedAt.Before(*filter.From) {
			continue
		}
		if filter.To != nil && !e.StartedAt.Before(*filter.To) {
			continue
		}
		out = append(out, e)
	}
	return out, int64(len(out)), nil
}

func (r *fakeTimeEntryRepo) FindByID(_ context.Context, id uint) (*entity.TimeEntry, error) {
	e, ok := r.entries[id]
	if !ok {
		return nil, entity.ErrTimeEntryNotFound
	}
	return &e, nil
}

func (r *fakeTimeEntryRepo) FindRunningByUserID(_ context.Context, userID uint) (*entity.TimeEntry, error) {
	for _, e := range r.entries {
		if e.UserID == userID && e.EndedAt == nil {
			found := e
			return &found, nil
		}
	}
	return nil, entity.ErrTimeEntryNotFound
}

func (r *fakeTimeEntryRepo) Update(_ context.Context, entry *entity.TimeEntry) error {
	if _, ok := r.entries[entry.ID]; !ok {
		return entity.ErrTimeEntryNotFound
	}
	if entry.EndedAt != nil && !entry.EndedAt.After(entry.StartedAt) {
		return entity.ErrRentangWaktuTidakValid
	}

	entry.UpdatedAt = time.Now()
	r.entries[entry.ID] = *entry
	return nil
}

func (r *fakeTimeEntryRepo) Delete(_ context.Context, id uint) error {
	if _, ok := r.entries[id]; !ok {
		return entity.ErrTimeEntryNotFound
	}
	delete(r.entries, id)
	return nil
}

// ---------- helper ----------

const (
	userPemilik = uint(7)
	userLain    = uint(8)
)

func setupTimeEntry(t *testing.T) (usecase.TimeEntryUsecase, *fakeTimeEntryRepo, uint) {
	t.Helper()

	taskRepo := newFakeTaskRepo()
	task := &entity.Task{NoTask: "TASK-001", Judul: "Setup database"}
	if err := taskRepo.Create(context.Background(), task); err != nil {
		t.Fatalf("setup task gagal: %v", err)
	}

	entryRepo := newFakeTimeEntryRepo()
	return usecase.NewTimeEntryUsecase(entryRepo, taskRepo), entryRepo, task.ID
}

func mulaiTimer(t *testing.T, uc usecase.TimeEntryUsecase, userID, taskID uint) *entity.TimeEntry {
	t.Helper()
	entry, err := uc.Start(context.Background(), usecase.StartTimerInput{
		UserID: userID,
		TaskID: taskID,
		Note:   "ngoding",
	})
	if err != nil {
		t.Fatalf("setup timer gagal: %v", err)
	}
	return entry
}

func mulaiTimerSatuJamLalu(t *testing.T, uc usecase.TimeEntryUsecase, userID, taskID uint) *entity.TimeEntry {
	t.Helper()

	entry := mulaiTimer(t, uc, userID, taskID)
	updated, err := uc.Update(context.Background(), userID, entry.ID,
		usecase.UpdateTimeEntryInput{StartedAt: ptrTime(time.Now().Add(-time.Hour))})
	if err != nil {
		t.Fatalf("setup memundurkan started_at gagal: %v", err)
	}
	return updated
}

func ptrTime(v time.Time) *time.Time { return &v }

func TestTimeEntryStart_MenyimpanDanMengisiID(t *testing.T) {
	uc, repo, taskID := setupTimeEntry(t)

	entry := mulaiTimer(t, uc, userPemilik, taskID)

	if entry.ID == 0 {
		t.Error("id seharusnya terisi setelah start")
	}
	if !entry.IsRunning() {
		t.Error("timer baru seharusnya berjalan (ended_at masih nil)")
	}
	if entry.UserID != userPemilik {
		t.Errorf("user_id = %d, ingin %d", entry.UserID, userPemilik)
	}
	if len(repo.entries) != 1 {
		t.Errorf("jumlah entry tersimpan = %d, ingin 1", len(repo.entries))
	}
}

func TestTimeEntryStart_SpasiNoteDipangkas(t *testing.T) {
	uc, _, taskID := setupTimeEntry(t)

	entry, err := uc.Start(context.Background(), usecase.StartTimerInput{
		UserID: userPemilik,
		TaskID: taskID,
		Note:   "  debug query lambat  ",
	})
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if entry.Note != "debug query lambat" {
		t.Errorf("note = %q, ingin %q", entry.Note, "debug query lambat")
	}
}

func TestTimeEntryStart_TimerKeduaDitolak(t *testing.T) {
	uc, _, taskID := setupTimeEntry(t)
	mulaiTimer(t, uc, userPemilik, taskID)

	_, err := uc.Start(context.Background(), usecase.StartTimerInput{
		UserID: userPemilik,
		TaskID: taskID,
	})
	if err == nil {
		t.Fatal("start kedua seharusnya ditolak")
	}
	if got := kindOf(t, err); got != apperror.KindConflict {
		t.Errorf("kind = %s, ingin %s", got, apperror.KindConflict)
	}
}

func TestTimeEntryStart_TaskTidakDitemukan(t *testing.T) {
	uc, _, _ := setupTimeEntry(t)

	_, err := uc.Start(context.Background(), usecase.StartTimerInput{
		UserID: userPemilik,
		TaskID: 999999,
	})
	if err == nil {
		t.Fatal("task yang tidak ada seharusnya ditolak")
	}
	if got := kindOf(t, err); got != apperror.KindNotFound {
		t.Errorf("kind = %s, ingin %s", got, apperror.KindNotFound)
	}
}

func TestTimeEntryStart_TimerTerpisahPerUser(t *testing.T) {
	uc, _, taskID := setupTimeEntry(t)
	mulaiTimer(t, uc, userPemilik, taskID)

	if _, err := uc.Start(context.Background(), usecase.StartTimerInput{
		UserID: userLain,
		TaskID: taskID,
	}); err != nil {
		t.Fatalf("user lain seharusnya tetap bisa memulai timer: %v", err)
	}
}

func TestTimeEntryStart_BalapanDenganIndexUnikJadiKonflik(t *testing.T) {
	uc, repo, taskID := setupTimeEntry(t)
	repo.tolakCreateKarenaIndexUnik = true

	_, err := uc.Start(context.Background(), usecase.StartTimerInput{
		UserID: userPemilik,
		TaskID: taskID,
	})
	if err == nil {
		t.Fatal("penolakan index unik seharusnya diteruskan sebagai error")
	}
	if got := kindOf(t, err); got != apperror.KindConflict {
		t.Errorf("kind = %s, ingin %s", got, apperror.KindConflict)
	}
}

func TestTimeEntryStop_MenutupTimerDanMengisiNote(t *testing.T) {
	uc, _, taskID := setupTimeEntry(t)
	mulaiTimerSatuJamLalu(t, uc, userPemilik, taskID)

	stopped, err := uc.Stop(context.Background(), usecase.StopTimerInput{
		UserID: userPemilik,
		Note:   ptr("selesai debug"),
	})
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if stopped.IsRunning() {
		t.Error("setelah stop, ended_at seharusnya sudah terisi")
	}
	if stopped.Note != "selesai debug" {
		t.Errorf("note = %q, ingin %q", stopped.Note, "selesai debug")
	}
}

func TestTimeEntryStop_NoteTidakDikirimTidakBerubah(t *testing.T) {
	uc, _, taskID := setupTimeEntry(t)
	mulaiTimerSatuJamLalu(t, uc, userPemilik, taskID)

	stopped, err := uc.Stop(context.Background(), usecase.StopTimerInput{UserID: userPemilik})
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if stopped.Note != "ngoding" {
		t.Errorf("note = %q, ingin tetap %q", stopped.Note, "ngoding")
	}
}

func TestTimeEntryStop_TanpaTimerBerjalanDitolak(t *testing.T) {
	uc, _, _ := setupTimeEntry(t)

	_, err := uc.Stop(context.Background(), usecase.StopTimerInput{UserID: userPemilik})
	if err == nil {
		t.Fatal("stop tanpa timer berjalan seharusnya ditolak")
	}
	if got := kindOf(t, err); got != apperror.KindConflict {
		t.Errorf("kind = %s, ingin %s", got, apperror.KindConflict)
	}
}

func TestTimeEntryStop_HanyaMenutupTimerMilikPemanggil(t *testing.T) {
	uc, _, taskID := setupTimeEntry(t)
	milikPemilik := mulaiTimerSatuJamLalu(t, uc, userPemilik, taskID)
	mulaiTimer(t, uc, userLain, taskID)

	stopped, err := uc.Stop(context.Background(), usecase.StopTimerInput{UserID: userPemilik})
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if stopped.ID != milikPemilik.ID {
		t.Errorf("id yang dihentikan = %d, ingin %d", stopped.ID, milikPemilik.ID)
	}

	lain, err := uc.Running(context.Background(), userLain)
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if lain == nil {
		t.Error("timer user lain seharusnya tidak ikut berhenti")
	}
}

func TestTimeEntryRunning_MengembalikanTimerYangBerjalan(t *testing.T) {
	uc, _, taskID := setupTimeEntry(t)
	entry := mulaiTimer(t, uc, userPemilik, taskID)

	running, err := uc.Running(context.Background(), userPemilik)
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if running == nil {
		t.Fatal("seharusnya ada timer yang berjalan")
	}
	if running.ID != entry.ID {
		t.Errorf("id = %d, ingin %d", running.ID, entry.ID)
	}
}

func TestTimeEntryRunning_TidakAdaTimerBukanError(t *testing.T) {
	uc, _, _ := setupTimeEntry(t)

	running, err := uc.Running(context.Background(), userPemilik)
	if err != nil {
		t.Fatalf("tidak ada timer berjalan seharusnya bukan error, dapat: %v", err)
	}
	if running != nil {
		t.Errorf("entry = %v, ingin nil", running)
	}
}

func TestTimeEntryGetByID_MilikUserLainDianggapTidakAda(t *testing.T) {
	uc, _, taskID := setupTimeEntry(t)
	entry := mulaiTimer(t, uc, userPemilik, taskID)

	if _, err := uc.GetByID(context.Background(), userPemilik, entry.ID); err != nil {
		t.Fatalf("pemilik seharusnya bisa membaca: %v", err)
	}

	_, err := uc.GetByID(context.Background(), userLain, entry.ID)
	if err == nil {
		t.Fatal("entry milik orang lain seharusnya tidak bisa dibaca")
	}
	if got := kindOf(t, err); got != apperror.KindNotFound {
		t.Errorf("kind = %s, ingin %s", got, apperror.KindNotFound)
	}
}

func TestTimeEntryGetByID_TidakDitemukan(t *testing.T) {
	uc, _, _ := setupTimeEntry(t)

	_, err := uc.GetByID(context.Background(), userPemilik, 999999)
	if err == nil {
		t.Fatal("id yang tidak ada seharusnya ditolak")
	}
	if got := kindOf(t, err); got != apperror.KindNotFound {
		t.Errorf("kind = %s, ingin %s", got, apperror.KindNotFound)
	}
}

func TestTimeEntryUpdate_KoreksiLupaStop(t *testing.T) {
	uc, _, taskID := setupTimeEntry(t)
	entry := mulaiTimer(t, uc, userPemilik, taskID)

	seharusnyaSelesai := entry.StartedAt.Add(2 * time.Hour)
	fixed, err := uc.Update(context.Background(), userPemilik, entry.ID,
		usecase.UpdateTimeEntryInput{EndedAt: ptrTime(seharusnyaSelesai)})
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if fixed.IsRunning() {
		t.Error("setelah dikoreksi, entry tidak boleh berjalan lagi")
	}
	if got := fixed.DurationAt(time.Now()); got != 2*time.Hour {
		t.Errorf("durasi = %v, ingin %v", got, 2*time.Hour)
	}
}

func TestTimeEntryUpdate_HanyaFieldYangDikirimYangBerubah(t *testing.T) {
	uc, _, taskID := setupTimeEntry(t)
	entry := mulaiTimer(t, uc, userPemilik, taskID)
	mulaiSemula := entry.StartedAt

	updated, err := uc.Update(context.Background(), userPemilik, entry.ID,
		usecase.UpdateTimeEntryInput{Note: ptr("review PR")})
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if updated.Note != "review PR" {
		t.Errorf("note = %q, ingin %q", updated.Note, "review PR")
	}
	if !updated.StartedAt.Equal(mulaiSemula) {
		t.Error("started_at seharusnya tidak berubah karena tidak dikirim")
	}
	if !updated.IsRunning() {
		t.Error("entry seharusnya masih berjalan karena ended_at tidak dikirim")
	}
}

func TestTimeEntryUpdate_RentangTerbalikDitolak(t *testing.T) {
	uc, _, taskID := setupTimeEntry(t)
	entry := mulaiTimer(t, uc, userPemilik, taskID)

	_, err := uc.Update(context.Background(), userPemilik, entry.ID,
		usecase.UpdateTimeEntryInput{EndedAt: ptrTime(entry.StartedAt.Add(-time.Hour))})
	if err == nil {
		t.Fatal("waktu selesai sebelum waktu mulai seharusnya ditolak")
	}
	if got := kindOf(t, err); got != apperror.KindValidation {
		t.Errorf("kind = %s, ingin %s", got, apperror.KindValidation)
	}
}

func TestTimeEntryUpdate_TaskBaruTidakDitemukan(t *testing.T) {
	uc, _, taskID := setupTimeEntry(t)
	entry := mulaiTimer(t, uc, userPemilik, taskID)

	taskNgawur := uint(999999)
	_, err := uc.Update(context.Background(), userPemilik, entry.ID,
		usecase.UpdateTimeEntryInput{TaskID: &taskNgawur})
	if err == nil {
		t.Fatal("pindah ke task yang tidak ada seharusnya ditolak")
	}
	if got := kindOf(t, err); got != apperror.KindNotFound {
		t.Errorf("kind = %s, ingin %s", got, apperror.KindNotFound)
	}
}

func TestTimeEntryUpdate_MilikUserLainDitolak(t *testing.T) {
	uc, _, taskID := setupTimeEntry(t)
	entry := mulaiTimer(t, uc, userPemilik, taskID)

	_, err := uc.Update(context.Background(), userLain, entry.ID,
		usecase.UpdateTimeEntryInput{Note: ptr("disabotase")})
	if err == nil {
		t.Fatal("entry milik orang lain seharusnya tidak bisa diubah")
	}
	if got := kindOf(t, err); got != apperror.KindNotFound {
		t.Errorf("kind = %s, ingin %s", got, apperror.KindNotFound)
	}
}

// ---------- List ----------

func TestTimeEntryList_DibatasiPerUser(t *testing.T) {
	uc, _, taskID := setupTimeEntry(t)
	mulaiTimer(t, uc, userPemilik, taskID)
	mulaiTimer(t, uc, userLain, taskID)

	out, err := uc.List(context.Background(), usecase.ListTimeEntryInput{UserID: userPemilik})
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if out.Total != 1 {
		t.Errorf("total = %d, ingin 1", out.Total)
	}
	for _, e := range out.Entries {
		if e.UserID != userPemilik {
			t.Errorf("entry milik user %d ikut terbawa", e.UserID)
		}
	}
}

func TestTimeEntryList_PagingDinormalisasi(t *testing.T) {
	uc, _, _ := setupTimeEntry(t)

	out, err := uc.List(context.Background(), usecase.ListTimeEntryInput{
		UserID: userPemilik,
		Page:   0,
		Limit:  0,
	})
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if out.Page != 1 || out.Limit != 10 {
		t.Errorf("page/limit = %d/%d, ingin 1/10", out.Page, out.Limit)
	}
}

func TestTimeEntryList_LimitDibatasiMaksimum(t *testing.T) {
	uc, _, _ := setupTimeEntry(t)

	out, err := uc.List(context.Background(), usecase.ListTimeEntryInput{
		UserID: userPemilik,
		Limit:  9999,
	})
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if out.Limit != 100 {
		t.Errorf("limit = %d, ingin dibatasi ke 100", out.Limit)
	}
}

func TestTimeEntryList_RentangTanggalTerbalikDitolak(t *testing.T) {
	uc, _, _ := setupTimeEntry(t)

	from := time.Now()
	_, err := uc.List(context.Background(), usecase.ListTimeEntryInput{
		UserID: userPemilik,
		From:   ptrTime(from),
		To:     ptrTime(from.Add(-time.Hour)),
	})
	if err == nil {
		t.Fatal("rentang tanggal terbalik seharusnya ditolak")
	}
	if got := kindOf(t, err); got != apperror.KindValidation {
		t.Errorf("kind = %s, ingin %s", got, apperror.KindValidation)
	}
}

func TestTimeEntryList_MenyaringPerTask(t *testing.T) {
	uc, repo, taskID := setupTimeEntry(t)
	mulaiTimer(t, uc, userPemilik, taskID)

	out, err := uc.List(context.Background(), usecase.ListTimeEntryInput{
		UserID: userPemilik,
		TaskID: taskID + 999,
	})
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if out.Total != 0 {
		t.Errorf("total = %d, ingin 0 untuk task yang tidak punya catatan waktu", out.Total)
	}
	if len(repo.entries) != 1 {
		t.Errorf("datanya sendiri tidak boleh hilang: %d entry", len(repo.entries))
	}
}

func TestTimeEntryDelete_MenghapusDariRepo(t *testing.T) {
	uc, repo, taskID := setupTimeEntry(t)
	entry := mulaiTimer(t, uc, userPemilik, taskID)

	if err := uc.Delete(context.Background(), userPemilik, entry.ID); err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if len(repo.entries) != 0 {
		t.Errorf("jumlah entry tersisa = %d, ingin 0", len(repo.entries))
	}

	_, err := uc.GetByID(context.Background(), userPemilik, entry.ID)
	if err == nil {
		t.Fatal("entry yang sudah dihapus seharusnya tidak terbaca lagi")
	}
	if got := kindOf(t, err); got != apperror.KindNotFound {
		t.Errorf("kind = %s, ingin %s", got, apperror.KindNotFound)
	}
}

func TestTimeEntryDelete_MilikUserLainDitolak(t *testing.T) {
	uc, repo, taskID := setupTimeEntry(t)
	entry := mulaiTimer(t, uc, userPemilik, taskID)

	err := uc.Delete(context.Background(), userLain, entry.ID)
	if err == nil {
		t.Fatal("entry milik orang lain seharusnya tidak bisa dihapus")
	}
	if got := kindOf(t, err); got != apperror.KindNotFound {
		t.Errorf("kind = %s, ingin %s", got, apperror.KindNotFound)
	}
	if len(repo.entries) != 1 {
		t.Error("entry seharusnya masih utuh setelah penolakan")
	}
}

func TestTimeEntryDelete_TimerBerjalanDihapusMembebaskanSlot(t *testing.T) {
	uc, _, taskID := setupTimeEntry(t)
	entry := mulaiTimer(t, uc, userPemilik, taskID)

	if err := uc.Delete(context.Background(), userPemilik, entry.ID); err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if _, err := uc.Start(context.Background(), usecase.StartTimerInput{
		UserID: userPemilik,
		TaskID: taskID,
	}); err != nil {
		t.Fatalf("seharusnya bisa memulai timer baru setelah yang lama dihapus: %v", err)
	}
}
