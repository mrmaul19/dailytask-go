package usecase_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"daily-task-go/internal/domain/entity"
	"daily-task-go/internal/domain/repository"
	"daily-task-go/internal/domain/storage"
	"daily-task-go/internal/usecase"
	"daily-task-go/pkg/apperror"
)

type fakeTaskRepo struct {
	tasks  map[uint]entity.Task
	lastID uint
}

func newFakeTaskRepo() *fakeTaskRepo {
	return &fakeTaskRepo{tasks: map[uint]entity.Task{}}
}

func (r *fakeTaskRepo) Create(_ context.Context, task *entity.Task) error {
	r.lastID++
	task.ID = r.lastID
	r.tasks[task.ID] = *task
	return nil
}

func (r *fakeTaskRepo) FindAll(_ context.Context, filter repository.TaskFilter) ([]entity.Task, int64, error) {
	var out []entity.Task
	for _, t := range r.tasks {
		if filter.Search == "" || strings.Contains(t.Judul, filter.Search) {
			out = append(out, t)
		}
	}
	return out, int64(len(out)), nil
}

func (r *fakeTaskRepo) FindByID(_ context.Context, id uint) (*entity.Task, error) {
	t, ok := r.tasks[id]
	if !ok {
		return nil, entity.ErrTaskNotFound
	}
	return &t, nil
}

func (r *fakeTaskRepo) FindByNoTask(_ context.Context, noTask string) (*entity.Task, error) {
	for _, t := range r.tasks {
		if t.NoTask == noTask {
			found := t
			return &found, nil
		}
	}
	return nil, entity.ErrTaskNotFound
}

func (r *fakeTaskRepo) Update(_ context.Context, task *entity.Task) error {
	if _, ok := r.tasks[task.ID]; !ok {
		return entity.ErrTaskNotFound
	}
	r.tasks[task.ID] = *task
	return nil
}

func (r *fakeTaskRepo) Delete(_ context.Context, id uint) error {
	if _, ok := r.tasks[id]; !ok {
		return entity.ErrTaskNotFound
	}
	delete(r.tasks, id)
	return nil
}

type fakeStorage struct {
	files map[string][]byte
	seq   int
}

func newFakeStorage() *fakeStorage {
	return &fakeStorage{files: map[string][]byte{}}
}

func (s *fakeStorage) Save(_ context.Context, upload storage.Upload) (*storage.Stored, error) {
	data, err := io.ReadAll(upload.Content)
	if err != nil {
		return nil, err
	}
	s.seq++
	path := fmt.Sprintf("uploads/file-%d", s.seq)
	s.files[path] = data
	return &storage.Stored{FileName: upload.FileName, Path: path}, nil
}

func (s *fakeStorage) Open(_ context.Context, path string) (*storage.Content, error) {
	data, ok := s.files[path]
	if !ok {
		return nil, apperror.NotFound("file tidak ditemukan di penyimpanan")
	}
	return &storage.Content{Reader: io.NopCloser(bytes.NewReader(data)), Size: int64(len(data))}, nil
}

func (s *fakeStorage) Delete(_ context.Context, path string) error {
	delete(s.files, path)
	return nil
}

func setup() (usecase.TaskUsecase, *fakeTaskRepo, *fakeStorage) {
	repo := newFakeTaskRepo()
	files := newFakeStorage()
	return usecase.NewTaskUsecase(repo, files), repo, files
}

func upload(name, content string) *storage.Upload {
	return &storage.Upload{
		FileName: name,
		Size:     int64(len(content)),
		Content:  strings.NewReader(content),
	}
}

func kindOf(t *testing.T, err error) apperror.Kind {
	t.Helper()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error bukan *apperror.AppError: %v", err)
	}
	return appErr.Kind
}

func TestCreate_NoTaskDigenerateOtomatis(t *testing.T) {
	uc, _, _ := setup()

	task, err := uc.Create(context.Background(), usecase.CreateTaskInput{
		Judul:       "Belajar clean architecture",
		Description: "latihan",
	})
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if !strings.HasPrefix(task.NoTask, "TASK-") {
		t.Errorf("no_task = %q, ingin berawalan TASK-", task.NoTask)
	}
	if task.HasFile() {
		t.Error("task tanpa upload seharusnya tidak punya file")
	}
}

func TestCreate_NoTaskDuplikatDitolak(t *testing.T) {
	uc, _, _ := setup()
	ctx := context.Background()

	if _, err := uc.Create(ctx, usecase.CreateTaskInput{NoTask: "TASK-001", Judul: "pertama"}); err != nil {
		t.Fatalf("setup gagal: %v", err)
	}

	_, err := uc.Create(ctx, usecase.CreateTaskInput{NoTask: "TASK-001", Judul: "kedua"})
	if err == nil {
		t.Fatal("ingin error konflik, dapat nil")
	}
	if got := kindOf(t, err); got != apperror.KindConflict {
		t.Errorf("kind = %s, ingin %s", got, apperror.KindConflict)
	}
}

func TestCreate_DenganFileMenyimpanLampiran(t *testing.T) {
	uc, _, files := setup()

	task, err := uc.Create(context.Background(), usecase.CreateTaskInput{
		Judul: "laporan harian",
		File:  upload("laporan.pdf", "isi file"),
	})
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if task.FileName != "laporan.pdf" {
		t.Errorf("file_name = %q", task.FileName)
	}
	if len(files.files) != 1 {
		t.Errorf("jumlah file tersimpan = %d, ingin 1", len(files.files))
	}
}

func TestUpdate_HanyaFieldYangDikirimYangBerubah(t *testing.T) {
	uc, _, _ := setup()
	ctx := context.Background()

	task, _ := uc.Create(ctx, usecase.CreateTaskInput{
		NoTask: "TASK-010", Judul: "judul lama", Description: "deskripsi lama",
	})

	judulBaru := "judul baru"
	updated, err := uc.Update(ctx, task.ID, usecase.UpdateTaskInput{Judul: &judulBaru})
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if updated.Judul != judulBaru {
		t.Errorf("judul = %q, ingin %q", updated.Judul, judulBaru)
	}
	if updated.Description != "deskripsi lama" {
		t.Errorf("description ikut berubah menjadi %q", updated.Description)
	}
	if updated.NoTask != "TASK-010" {
		t.Errorf("no_task ikut berubah menjadi %q", updated.NoTask)
	}
}

func TestUpdate_GantiFileMenghapusFileLama(t *testing.T) {
	uc, _, files := setup()
	ctx := context.Background()

	task, _ := uc.Create(ctx, usecase.CreateTaskInput{
		Judul: "ada lampiran", File: upload("lama.pdf", "versi 1"),
	})
	pathLama := task.FilePath

	updated, err := uc.Update(ctx, task.ID, usecase.UpdateTaskInput{File: upload("baru.pdf", "versi 2")})
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if _, masihAda := files.files[pathLama]; masihAda {
		t.Error("file lama seharusnya sudah dihapus")
	}
	if _, ada := files.files[updated.FilePath]; !ada {
		t.Error("file baru tidak tersimpan")
	}
}

func TestUpdate_RemoveFileMelepasLampiran(t *testing.T) {
	uc, _, files := setup()
	ctx := context.Background()

	task, _ := uc.Create(ctx, usecase.CreateTaskInput{
		Judul: "ada lampiran", File: upload("lama.pdf", "versi 1"),
	})

	updated, err := uc.Update(ctx, task.ID, usecase.UpdateTaskInput{RemoveFile: true})
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if updated.HasFile() {
		t.Error("lampiran seharusnya sudah dilepas")
	}
	if len(files.files) != 0 {
		t.Errorf("file di penyimpanan = %d, ingin 0", len(files.files))
	}
}

func TestDelete_IkutMenghapusFileFisik(t *testing.T) {
	uc, repo, files := setup()
	ctx := context.Background()

	task, _ := uc.Create(ctx, usecase.CreateTaskInput{
		Judul: "akan dihapus", File: upload("bukti.png", "gambar"),
	})

	if err := uc.Delete(ctx, task.ID); err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if len(repo.tasks) != 0 {
		t.Error("task masih ada di repository")
	}
	if len(files.files) != 0 {
		t.Error("file masih ada di penyimpanan")
	}
}

func TestGetByID_TidakDitemukan(t *testing.T) {
	uc, _, _ := setup()

	_, err := uc.GetByID(context.Background(), 404)
	if err == nil {
		t.Fatal("ingin error not found, dapat nil")
	}
	if got := kindOf(t, err); got != apperror.KindNotFound {
		t.Errorf("kind = %s, ingin %s", got, apperror.KindNotFound)
	}
}

func TestList_PagingDinormalisasi(t *testing.T) {
	uc, _, _ := setup()

	out, err := uc.List(context.Background(), usecase.ListTaskInput{Page: 0, Limit: 0})
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if out.Page != 1 || out.Limit != 10 {
		t.Errorf("page=%d limit=%d, ingin page=1 limit=10", out.Page, out.Limit)
	}
}

func TestDownload_TaskTanpaFile(t *testing.T) {
	uc, _, _ := setup()
	ctx := context.Background()

	task, _ := uc.Create(ctx, usecase.CreateTaskInput{Judul: "tanpa lampiran"})

	_, err := uc.Download(ctx, task.ID)
	if err == nil {
		t.Fatal("ingin error not found, dapat nil")
	}
	if got := kindOf(t, err); got != apperror.KindNotFound {
		t.Errorf("kind = %s, ingin %s", got, apperror.KindNotFound)
	}
}
