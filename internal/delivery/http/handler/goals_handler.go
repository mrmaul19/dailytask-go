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

// Create POST /api/v1/goals
//
//	@Summary		Buat goals baru
//	@Description	Goals tidak punya lampiran file, jadi request dan response-nya JSON murni.
//	@Tags			Goals
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.CreateGoalsRequest	true	"Data goals"
//	@Success		201		{object}	dto.GoalsEnvelope
//	@Failure		400		{object}	dto.ErrorEnvelope	"validasi gagal"
//	@Failure		500		{object}	dto.ErrorEnvelope
//	@Router			/api/v1/goals [post]
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

// List GET /api/v1/goals?search=&page=&limit=
//
//	@Summary		Daftar goals
//	@Description	`search` dicocokkan ke `title` dan `category` (case-insensitive, pakai ILIKE).
//	@Tags			Goals
//	@Produce		json
//	@Param			search	query		string	false	"Kata kunci pencarian"
//	@Param			page	query		int		false	"Halaman, mulai dari 1"	default(1)	minimum(1)
//	@Param			limit	query		int		false	"Jumlah data per halaman"	default(10)	minimum(1)	maximum(100)
//	@Success		200		{object}	dto.GoalsListEnvelope
//	@Failure		500		{object}	dto.ErrorEnvelope
//	@Router			/api/v1/goals [get]
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

// GetByID GET /api/v1/goals/:id
//
//	@Summary		Detail goals
//	@Description	Goals yang sudah dihapus (soft delete) tidak ikut terambil.
//	@Tags			Goals
//	@Produce		json
//	@Param			id	path		int	true	"ID goals"	minimum(1)
//	@Success		200	{object}	dto.GoalsEnvelope
//	@Failure		400	{object}	dto.ErrorEnvelope	"id bukan angka positif"
//	@Failure		404	{object}	dto.ErrorEnvelope	"goals tidak ditemukan"
//	@Failure		500	{object}	dto.ErrorEnvelope
//	@Router			/api/v1/goals/{id} [get]
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

// Update PUT /api/v1/goals/:id
//
//	@Summary		Perbarui goals
//	@Description	Update sebagian field (partial update). Field yang tidak dikirim tetap memakai nilai lama,
//	@Description	field yang dikirim kosong ditolak dengan 400.
//	@Tags			Goals
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int						true	"ID goals"	minimum(1)
//	@Param			request	body		dto.UpdateGoalsRequest	true	"Field yang mau diubah saja"
//	@Success		200		{object}	dto.GoalsEnvelope
//	@Failure		400		{object}	dto.ErrorEnvelope	"validasi gagal atau id tidak valid"
//	@Failure		404		{object}	dto.ErrorEnvelope	"goals tidak ditemukan"
//	@Failure		500		{object}	dto.ErrorEnvelope
//	@Router			/api/v1/goals/{id} [put]
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

// Delete DELETE /api/v1/goals/:id
//
//	@Summary		Hapus goals
//	@Description	Soft delete: baris hanya diberi `deleted_at`, tidak benar-benar dihapus dari tabel.
//	@Tags			Goals
//	@Produce		json
//	@Param			id	path		int	true	"ID goals"	minimum(1)
//	@Success		200	{object}	dto.MessageEnvelope
//	@Failure		400	{object}	dto.ErrorEnvelope	"id bukan angka positif"
//	@Failure		404	{object}	dto.ErrorEnvelope	"goals tidak ditemukan"
//	@Failure		500	{object}	dto.ErrorEnvelope
//	@Router			/api/v1/goals/{id} [delete]
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
