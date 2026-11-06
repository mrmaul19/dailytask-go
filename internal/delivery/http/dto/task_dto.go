// Package dto adalah bentuk data khusus HTTP (request body & response JSON).
// Entity domain tidak pernah dikirim langsung ke client.
package dto

import (
	"strconv"
	"strings"
	"time"

	"daily-task-go/internal/domain/entity"
)

type CreateTaskRequest struct {
	NoTask      string `form:"no_task" json:"no_task" validate:"omitempty,max=50"`
	Judul       string `form:"judul" json:"judul" validate:"required,min=3,max=255"`
	Description string `form:"description" json:"description" validate:"omitempty,max=2000"`
}

type UpdateTaskRequest struct {
	NoTask      *string `form:"no_task" json:"no_task" validate:"omitempty,max=50"`
	Judul       *string `form:"judul" json:"judul" validate:"omitempty,min=3,max=255"`
	Description *string `form:"description" json:"description" validate:"omitempty,max=2000"`
	RemoveFile  bool    `form:"remove_file" json:"remove_file"`
}

type TaskResponse struct {
	ID          uint      `json:"id"`
	NoTask      string    `json:"no_task"`
	Judul       string    `json:"judul"`
	Description string    `json:"description"`
	FileName    string    `json:"file_name,omitempty"`
	FileURL     string    `json:"file_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func NewTaskResponse(task *entity.Task, baseURL string) TaskResponse {
	res := TaskResponse{
		ID:          task.ID,
		NoTask:      task.NoTask,
		Judul:       task.Judul,
		Description: task.Description,
		FileName:    task.FileName,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
	}
	if task.HasFile() {
		res.FileURL = strings.TrimSuffix(baseURL, "/") + "/api/v1/tasks/" + strconv.FormatUint(uint64(task.ID), 10) + "/download"
	}
	return res
}

func NewTaskResponses(tasks []entity.Task, baseURL string) []TaskResponse {
	out := make([]TaskResponse, 0, len(tasks))
	for i := range tasks {
		out = append(out, NewTaskResponse(&tasks[i], baseURL))
	}
	return out
}
