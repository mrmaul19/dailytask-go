package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"daily-task-go/internal/delivery/http/dto"
	"daily-task-go/internal/delivery/http/middleware"
	"daily-task-go/internal/usecase"
	"daily-task-go/pkg/apperror"
	"daily-task-go/pkg/response"
	"daily-task-go/pkg/validator"
)

type TimeEntryHandler struct {
	timeEntryUsecase usecase.TimeEntryUsecase
	validate         *validator.Validator
}

func NewTimeEntryHandler(timeEntryUsecase usecase.TimeEntryUsecase, validate *validator.Validator) *TimeEntryHandler {
	return &TimeEntryHandler{timeEntryUsecase: timeEntryUsecase, validate: validate}
}

func (h *TimeEntryHandler) Start(c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return err
	}

	var req dto.StartTimerRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.Validation("format request tidak valid", map[string]string{"body": err.Error()})
	}

	if errs := h.validate.Validate(req); errs != nil {
		return apperror.Validation("data tidak valid", errs)
	}

	entry, err := h.timeEntryUsecase.Start(c.UserContext(), usecase.StartTimerInput{
		UserID: userID,
		TaskID: req.TaskID,
		Note:   req.Note,
	})
	if err != nil {
		return err
	}

	return response.Success(c, http.StatusCreated, "timer berhasil dimulai",
		dto.NewTimeEntryResponse(entry))
}

func (h *TimeEntryHandler) Stop(c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return err
	}

	var req dto.StopTimerRequest
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&req); err != nil {
			return apperror.Validation("format request tidak valid", map[string]string{"body": err.Error()})
		}
		if errs := h.validate.Validate(req); errs != nil {
			return apperror.Validation("data tidak valid", errs)
		}
	}

	entry, err := h.timeEntryUsecase.Stop(c.UserContext(), usecase.StopTimerInput{
		UserID: userID,
		Note:   req.Note,
	})
	if err != nil {
		return err
	}

	return response.Success(c, http.StatusOK, "timer berhasil dihentikan",
		dto.NewTimeEntryResponse(entry))
}

func (h *TimeEntryHandler) Running(c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return err
	}

	entry, err := h.timeEntryUsecase.Running(c.UserContext(), userID)
	if err != nil {
		return err
	}

	if entry == nil {
		var kosong *dto.TimeEntryResponse
		return response.Success(c, http.StatusOK, "tidak ada timer yang sedang berjalan", kosong)
	}

	hasil := dto.NewTimeEntryResponse(entry)
	return response.Success(c, http.StatusOK, "timer yang sedang berjalan berhasil diambil", &hasil)
}

func (h *TimeEntryHandler) List(c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return err
	}

	from, err := parseWaktuQuery(c, "from")
	if err != nil {
		return err
	}
	to, err := parseWaktuQuery(c, "to")
	if err != nil {
		return err
	}

	out, err := h.timeEntryUsecase.List(c.UserContext(), usecase.ListTimeEntryInput{
		UserID: userID,
		TaskID: uint(c.QueryInt("task_id", 0)),
		From:   from,
		To:     to,
		Page:   c.QueryInt("page", 1),
		Limit:  c.QueryInt("limit", 10),
	})
	if err != nil {
		return err
	}

	return response.SuccessWithMeta(c, http.StatusOK, "daftar catatan waktu berhasil diambil",
		dto.NewTimeEntryResponses(out.Entries),
		response.NewMeta(out.Page, out.Limit, out.Total))
}

func (h *TimeEntryHandler) GetByID(c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return err
	}

	id, err := parseID(c)
	if err != nil {
		return err
	}

	entry, err := h.timeEntryUsecase.GetByID(c.UserContext(), userID, id)
	if err != nil {
		return err
	}

	return response.Success(c, http.StatusOK, "detail catatan waktu berhasil diambil",
		dto.NewTimeEntryResponse(entry))
}

func (h *TimeEntryHandler) Update(c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return err
	}

	id, err := parseID(c)
	if err != nil {
		return err
	}

	var req dto.UpdateTimeEntryRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.Validation("format request tidak valid", map[string]string{"body": err.Error()})
	}

	if errs := h.validate.Validate(req); errs != nil {
		return apperror.Validation("data tidak valid", errs)
	}

	entry, err := h.timeEntryUsecase.Update(c.UserContext(), userID, id, usecase.UpdateTimeEntryInput{
		TaskID:    req.TaskID,
		StartedAt: req.StartedAt,
		EndedAt:   req.EndedAt,
		Note:      req.Note,
	})
	if err != nil {
		return err
	}

	return response.Success(c, http.StatusOK, "catatan waktu berhasil diperbarui",
		dto.NewTimeEntryResponse(entry))
}

func (h *TimeEntryHandler) Delete(c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return err
	}

	id, err := parseID(c)
	if err != nil {
		return err
	}

	if err := h.timeEntryUsecase.Delete(c.UserContext(), userID, id); err != nil {
		return err
	}

	return response.Success(c, http.StatusOK, "catatan waktu berhasil dihapus", nil)
}

func userIDFromContext(c *fiber.Ctx) (uint, error) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok || userID == 0 {
		return 0, apperror.Unauthorized("token tidak dikirim")
	}
	return userID, nil
}

func parseWaktuQuery(c *fiber.Ctx, nama string) (*time.Time, error) {
	raw := strings.TrimSpace(c.Query(nama))
	if raw == "" {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, apperror.Validation("format waktu tidak valid", map[string]string{
			nama: "harus format RFC3339, contoh 2026-11-10T09:00:00+07:00",
		})
	}
	return &parsed, nil
}
