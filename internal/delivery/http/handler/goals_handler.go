package handler

import (
	"net/http"

	"github.com/gofiber/fiber/v2"

	"daily-task-go/internal/delivery/http/dto"
	"daily-task-go/internal/usecase"
	"daily-task-go/pkg/apperror"
	"daily-task-go/pkg/response"
	"daily-task-go/pkg/validator"
)

type GoalsHandler struct {
	goalsUsecase usecase.GoalsUsecase
	validate     *validator.Validator
}

func NewGoalsHandler(goalsUsecase usecase.GoalsUsecase, validate *validator.Validator) *GoalsHandler {
	return &GoalsHandler{goalsUsecase: goalsUsecase, validate: validate}
}

func (h *GoalsHandler) Create(c *fiber.Ctx) error {
	var req dto.CreateGoalsRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.Validation("format request tidak valid", map[string]string{"body": err.Error()})
	}

	if errs := h.validate.Validate(req); errs != nil {
		return apperror.Validation("data tidak valid", errs)
	}

	goals, err := h.goalsUsecase.Create(c.UserContext(), usecase.CreateGoalsInput{
		Title:    req.Title,
		Category: req.Category,
		Priority: req.Priority,
		Status:   req.Status,
		Progress: req.Progress,
	})
	if err != nil {
		return err
	}

	return response.Success(c, http.StatusCreated, "goals berhasil dibuat",
		dto.NewGoalsResponse(goals))
}

func (h *GoalsHandler) List(c *fiber.Ctx) error {
	out, err := h.goalsUsecase.List(c.UserContext(), usecase.ListGoalsInput{
		Search: c.Query("search"),
		Page:   c.QueryInt("page", 1),
		Limit:  c.QueryInt("limit", 10),
	})
	if err != nil {
		return err
	}

	return response.SuccessWithMeta(c, http.StatusOK, "daftar goals berhasil diambil",
		dto.NewGoalsResponses(out.Goals),
		response.NewMeta(out.Page, out.Limit, out.Total))
}

func (h *GoalsHandler) GetByID(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}

	goals, err := h.goalsUsecase.GetByID(c.UserContext(), id)
	if err != nil {
		return err
	}

	return response.Success(c, http.StatusOK, "detail goals berhasil diambil",
		dto.NewGoalsResponse(goals))
}

func (h *GoalsHandler) Update(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}

	var req dto.UpdateGoalsRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.Validation("format request tidak valid", map[string]string{"body": err.Error()})
	}

	if errs := h.validate.Validate(req); errs != nil {
		return apperror.Validation("data tidak valid", errs)
	}

	goals, err := h.goalsUsecase.Update(c.UserContext(), id, usecase.UpdateGoalsInput{
		Title:    req.Title,
		Category: req.Category,
		Priority: req.Priority,
		Status:   req.Status,
		Progress: req.Progress,
	})
	if err != nil {
		return err
	}

	return response.Success(c, http.StatusOK, "goals berhasil diperbarui",
		dto.NewGoalsResponse(goals))
}

func (h *GoalsHandler) Delete(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}

	if err := h.goalsUsecase.Delete(c.UserContext(), id); err != nil {
		return err
	}

	return response.Success(c, http.StatusOK, "goals berhasil dihapus", nil)
}
