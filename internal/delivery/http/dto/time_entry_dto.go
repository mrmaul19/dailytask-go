package dto

import (
	"time"

	"daily-task-go/internal/domain/entity"
)

type StartTimerRequest struct {
	TaskID uint   `form:"task_id" json:"task_id" validate:"required,min=1"`
	Note   string `form:"note" json:"note" validate:"omitempty,max=255"`
}

type StopTimerRequest struct {
	Note *string `form:"note" json:"note" validate:"omitempty,max=255"`
}

type UpdateTimeEntryRequest struct {
	TaskID    *uint      `form:"task_id" json:"task_id" validate:"omitempty,min=1"`
	StartedAt *time.Time `form:"started_at" json:"started_at"`
	EndedAt   *time.Time `form:"ended_at" json:"ended_at"`
	Note      *string    `form:"note" json:"note" validate:"omitempty,max=255"`
}

type TimeEntryResponse struct {
	ID     uint `json:"id"`
	UserID uint `json:"user_id"`
	TaskID uint `json:"task_id"`

	StartedAt       time.Time  `json:"started_at"`
	EndedAt         *time.Time `json:"ended_at"`
	Note            string     `json:"note"`
	IsRunning       bool       `json:"is_running"`
	DurationSeconds int64      `json:"duration_seconds"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func NewTimeEntryResponse(entry *entity.TimeEntry) TimeEntryResponse {
	return newTimeEntryResponseAt(entry, time.Now())
}

func newTimeEntryResponseAt(entry *entity.TimeEntry, now time.Time) TimeEntryResponse {
	return TimeEntryResponse{
		ID:              entry.ID,
		UserID:          entry.UserID,
		TaskID:          entry.TaskID,
		StartedAt:       entry.StartedAt,
		EndedAt:         entry.EndedAt,
		Note:            entry.Note,
		IsRunning:       entry.IsRunning(),
		DurationSeconds: int64(entry.DurationAt(now).Seconds()),
		CreatedAt:       entry.CreatedAt,
		UpdatedAt:       entry.UpdatedAt,
	}
}

func NewTimeEntryResponses(list []entity.TimeEntry) []TimeEntryResponse {
	now := time.Now()
	out := make([]TimeEntryResponse, 0, len(list))
	for i := range list {
		out = append(out, newTimeEntryResponseAt(&list[i], now))
	}
	return out
}
