package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"daily-task-go/internal/domain/entity"
	"daily-task-go/internal/domain/repository"
	"daily-task-go/internal/domain/storage"
	"daily-task-go/pkg/apperror"
)

type CreateTaskInput struct {
	NoTask      string
	Judul       string
	Description string
	File        *storage.Upload
}

type UpdateTaskInput struct {
	NoTask      *string
	Judul       *string
	Description *string
	File        *storage.Upload
	RemoveFile  bool
}

type ListTaskInput struct {
	Search string
	Page   int
	Limit  int
}

type DownloadOutput struct {
	FileName string
	Size     int64
	Content  io.ReadCloser
}

type ListTaskOutput struct {
	Tasks []entity.Task
	Total int64
	Page  int
	Limit int
}

type TaskUsecase interface {
	Create(ctx context.Context, in CreateTaskInput) (*entity.Task, error)
	List(ctx context.Context, in ListTaskInput) (*ListTaskOutput, error)
	GetByID(ctx context.Context, id uint) (*entity.Task, error)
	Update(ctx context.Context, id uint, in UpdateTaskInput) (*entity.Task, error)
	Delete(ctx context.Context, id uint) error
	Download(ctx context.Context, id uint) (*DownloadOutput, error)
}

type taskUsecase struct {
	taskRepo repository.TaskRepository
	files    storage.FileStorage
}

func NewTaskUsecase(taskRepo repository.TaskRepository, files storage.FileStorage) TaskUsecase {
	return &taskUsecase{taskRepo: taskRepo, files: files}
}

const (
	defaultPage  = 1
	defaultLimit = 10
	maxLimit     = 100
)

func (u *taskUsecase) Create(ctx context.Context, in CreateTaskInput) (*entity.Task, error) {
	noTask := strings.TrimSpace(in.NoTask)
	if noTask == "" {
		generated, err := generateNoTask()
		if err != nil {
			return nil, apperror.Internal("gagal membuat nomor task", err)
		}
		noTask = generated
	}

	if err := u.ensureNoTaskAvailable(ctx, noTask, 0); err != nil {
		return nil, err
	}

	task := &entity.Task{
		NoTask:      noTask,
		Judul:       strings.TrimSpace(in.Judul),
		Description: strings.TrimSpace(in.Description),
	}

	var storedPath string
	if in.File != nil {
		stored, err := u.files.Save(ctx, *in.File)
		if err != nil {
			return nil, err
		}
		storedPath = stored.Path
		task.AttachFile(stored.FileName, stored.Path)
	}

	if err := u.taskRepo.Create(ctx, task); err != nil {
		if storedPath != "" {
			_ = u.files.Delete(ctx, storedPath)
		}
		return nil, apperror.Internal("gagal menyimpan task", err)
	}

	return task, nil
}

func (u *taskUsecase) List(ctx context.Context, in ListTaskInput) (*ListTaskOutput, error) {
	page, limit := normalizePaging(in.Page, in.Limit)

	tasks, total, err := u.taskRepo.FindAll(ctx, repository.TaskFilter{
		Search: strings.TrimSpace(in.Search),
		Page:   page,
		Limit:  limit,
	})
	if err != nil {
		return nil, apperror.Internal("gagal mengambil daftar task", err)
	}

	return &ListTaskOutput{Tasks: tasks, Total: total, Page: page, Limit: limit}, nil
}

func (u *taskUsecase) GetByID(ctx context.Context, id uint) (*entity.Task, error) {
	task, err := u.taskRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, entity.ErrTaskNotFound) {
			return nil, apperror.NotFound(fmt.Sprintf("task dengan id %d tidak ditemukan", id))
		}
		return nil, apperror.Internal("gagal mengambil task", err)
	}
	return task, nil
}

func (u *taskUsecase) Update(ctx context.Context, id uint, in UpdateTaskInput) (*entity.Task, error) {
	task, err := u.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.NoTask != nil {
		noTask := strings.TrimSpace(*in.NoTask)
		if noTask == "" {
			return nil, apperror.Validation("data tidak valid", map[string]string{"no_task": "tidak boleh kosong"})
		}
		if noTask != task.NoTask {
			if err := u.ensureNoTaskAvailable(ctx, noTask, task.ID); err != nil {
				return nil, err
			}
			task.NoTask = noTask
		}
	}

	if in.Judul != nil {
		judul := strings.TrimSpace(*in.Judul)
		if judul == "" {
			return nil, apperror.Validation("data tidak valid", map[string]string{"judul": "tidak boleh kosong"})
		}
		task.Judul = judul
	}

	if in.Description != nil {
		task.Description = strings.TrimSpace(*in.Description)
	}

	oldPath := task.FilePath

	switch {
	case in.File != nil:
		stored, err := u.files.Save(ctx, *in.File)
		if err != nil {
			return nil, err
		}
		task.AttachFile(stored.FileName, stored.Path)
	case in.RemoveFile:
		task.DetachFile()
	}

	if err := u.taskRepo.Update(ctx, task); err != nil {
		if task.FilePath != oldPath && task.HasFile() {
			_ = u.files.Delete(ctx, task.FilePath)
		}
		return nil, apperror.Internal("gagal memperbarui task", err)
	}
	if oldPath != "" && oldPath != task.FilePath {
		_ = u.files.Delete(ctx, oldPath)
	}

	return task, nil
}

func (u *taskUsecase) Delete(ctx context.Context, id uint) error {
	task, err := u.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := u.taskRepo.Delete(ctx, task.ID); err != nil {
		return apperror.Internal("gagal menghapus task", err)
	}

	if task.HasFile() {
		_ = u.files.Delete(ctx, task.FilePath)
	}
	return nil
}

func (u *taskUsecase) ensureNoTaskAvailable(ctx context.Context, noTask string, exceptID uint) error {
	existing, err := u.taskRepo.FindByNoTask(ctx, noTask)
	if err != nil {
		if errors.Is(err, entity.ErrTaskNotFound) {
			return nil
		}
		return apperror.Internal("gagal memeriksa nomor task", err)
	}
	if existing.ID == exceptID {
		return nil
	}
	return apperror.Conflict(fmt.Sprintf("nomor task %q sudah digunakan", noTask))
}

func normalizePaging(page, limit int) (int, int) {
	if page < 1 {
		page = defaultPage
	}
	if limit < 1 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return page, limit
}

func generateNoTask() (string, error) {
	buf := make([]byte, 2)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return fmt.Sprintf("TASK-%s-%s",
		time.Now().Format("20060102"),
		strings.ToUpper(hex.EncodeToString(buf)),
	), nil
}

func (u *taskUsecase) Download(ctx context.Context, id uint) (*DownloadOutput, error) {
	task, err := u.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !task.HasFile() {
		return nil, apperror.NotFound("task ini tidak memiliki file")
	}

	content, err := u.files.Open(ctx, task.FilePath)
	if err != nil {
		return nil, err
	}

	return &DownloadOutput{
		FileName: task.FileName,
		Size:     content.Size,
		Content:  content.Reader,
	}, nil
}
