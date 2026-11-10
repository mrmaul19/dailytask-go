package dto

import (
	"time"

	"daily-task-go/internal/domain/entity"
)

type CreateGoalsRequest struct {
	Title    string `form:"title" json:"title" validate:"required,min=3,max=255"`
	Category string `form:"category" json:"category" validate:"required,max=100"`
	Priority string `form:"priority" json:"priority" validate:"required,max=100"`
	Status   string `form:"status" json:"status" validate:"required,max=100"`
	Progress string `form:"progress" json:"progress" validate:"omitempty,max=100"`
}

type UpdateGoalsRequest struct {
	Title    *string `form:"title" json:"title" validate:"omitempty,min=3,max=255"`
	Category *string `form:"category" json:"category" validate:"omitempty,max=100"`
	Priority *string `form:"priority" json:"priority" validate:"omitempty,max=100"`
	Status   *string `form:"status" json:"status" validate:"omitempty,max=100"`
	Progress *string `form:"progress" json:"progress" validate:"omitempty,max=100"`
}

type GoalsResponse struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	Category  string    `json:"category"`
	Priority  string    `json:"priority"`
	Status    string    `json:"status"`
	Progress  string    `json:"progress"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewGoalsResponse(goals *entity.Goals) GoalsResponse {
	return GoalsResponse{
		ID:        goals.ID,
		Title:     goals.Title,
		Category:  goals.Category,
		Priority:  goals.Priority,
		Status:    goals.Status,
		Progress:  goals.Progress,
		CreatedAt: goals.CreatedAt,
		UpdatedAt: goals.UpdatedAt,
	}
}

func NewGoalsResponses(list []entity.Goals) []GoalsResponse {
	out := make([]GoalsResponse, 0, len(list))
	for i := range list {
		out = append(out, NewGoalsResponse(&list[i]))
	}
	return out
}
