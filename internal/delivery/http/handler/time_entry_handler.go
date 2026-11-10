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

// Start POST /api/v1/time-entries/start
//
//	@Summary		Mulai timer
//	@Description	Memulai pencatatan waktu untuk sebuah task. Waktu mulai diambil dari jam server
//	@Description	dan pemiliknya dari token — keduanya tidak bisa dikirim dari request.
//	@Description	Hanya boleh ada satu timer berjalan per user. Kalau masih ada yang berjalan,
//	@Description	request ini dibalas 409 dan timer lama TIDAK ikut dihentikan.
//	@Tags			Time Tracking
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.StartTimerRequest	true	"Task yang dikerjakan"
//	@Success		201		{object}	dto.TimeEntryEnvelope
//	@Failure		400		{object}	dto.ErrorEnvelope	"validasi gagal"
//	@Failure		401		{object}	dto.ErrorEnvelope	"token tidak dikirim atau tidak valid"
//	@Failure		404		{object}	dto.ErrorEnvelope	"task tidak ditemukan"
//	@Failure		409		{object}	dto.ErrorEnvelope	"masih ada timer yang berjalan"
//	@Failure		500		{object}	dto.ErrorEnvelope
//	@Router			/api/v1/time-entries/start [post]
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

// Stop POST /api/v1/time-entries/stop
//
//	@Summary		Hentikan timer
//	@Description	Menutup timer milik pemanggil. Tidak perlu mengirim id entry: hanya ada satu
//	@Description	timer yang mungkin berjalan per user, jadi tidak ada yang ambigu — sekaligus
//	@Description	berarti timer milik orang lain tidak bisa dihentikan lewat endpoint ini.
//	@Description	`note` opsional: kalau tidak dikirim, catatan dari saat Start dipertahankan.
//	@Tags			Time Tracking
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.StopTimerRequest	false	"Catatan penutup (opsional)"
//	@Success		200		{object}	dto.TimeEntryEnvelope
//	@Failure		400		{object}	dto.ErrorEnvelope	"validasi gagal"
//	@Failure		401		{object}	dto.ErrorEnvelope	"token tidak dikirim atau tidak valid"
//	@Failure		409		{object}	dto.ErrorEnvelope	"tidak ada timer yang sedang berjalan"
//	@Failure		500		{object}	dto.ErrorEnvelope
//	@Router			/api/v1/time-entries/stop [post]
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

// Running GET /api/v1/time-entries/running
//
//	@Summary		Timer yang sedang berjalan
//	@Description	Dipakai saat halaman dimuat, supaya timer yang sedang berjalan tidak hilang
//	@Description	setelah browser di-refresh.
//	@Description	Tidak ada timer yang berjalan BUKAN error: dibalas 200 dengan `data` null,
//	@Description	karena itu keadaan normal yang akan sering terjadi.
//	@Tags			Time Tracking
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200	{object}	dto.RunningTimeEntryEnvelope
//	@Failure		401	{object}	dto.ErrorEnvelope	"token tidak dikirim atau tidak valid"
//	@Failure		500	{object}	dto.ErrorEnvelope
//	@Router			/api/v1/time-entries/running [get]
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

// List GET /api/v1/time-entries?task_id=&from=&to=&page=&limit=
//
//	@Summary		Daftar catatan waktu
//	@Description	Hanya catatan waktu milik pemanggil yang dikembalikan — `user_id` diambil dari
//	@Description	token, bukan dari query string.
//	@Description	`from` dan `to` memakai format RFC3339, contoh `2026-11-01T00:00:00+07:00`.
//	@Description	Keduanya disaring pada `started_at`, dengan batas bawah inklusif dan batas atas
//	@Description	eksklusif — jadi rentang satu bulan penuh ditulis dari tanggal 1 bulan ini
//	@Description	sampai tanggal 1 bulan berikutnya.
//	@Tags			Time Tracking
//	@Security		BearerAuth
//	@Produce		json
//	@Param			task_id	query		int		false	"Saring per task"	minimum(1)
//	@Param			from	query		string	false	"Mulai dari (RFC3339)"
//	@Param			to		query		string	false	"Sampai sebelum (RFC3339)"
//	@Param			page	query		int		false	"Halaman, mulai dari 1"	default(1)	minimum(1)
//	@Param			limit	query		int		false	"Jumlah data per halaman"	default(10)	minimum(1)	maximum(100)
//	@Success		200		{object}	dto.TimeEntryListEnvelope
//	@Failure		400		{object}	dto.ErrorEnvelope	"format waktu atau rentang tidak valid"
//	@Failure		401		{object}	dto.ErrorEnvelope	"token tidak dikirim atau tidak valid"
//	@Failure		500		{object}	dto.ErrorEnvelope
//	@Router			/api/v1/time-entries [get]
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

// GetByID GET /api/v1/time-entries/:id
//
//	@Summary		Detail catatan waktu
//	@Description	Catatan waktu milik user lain dibalas 404, bukan 403 — 403 akan memberi tahu
//	@Description	bahwa id itu ada, cukup untuk memetakan catatan waktu siapa saja dengan
//	@Description	mencoba angka satu per satu.
//	@Tags			Time Tracking
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path		int	true	"ID catatan waktu"	minimum(1)
//	@Success		200	{object}	dto.TimeEntryEnvelope
//	@Failure		400	{object}	dto.ErrorEnvelope	"id bukan angka positif"
//	@Failure		401	{object}	dto.ErrorEnvelope	"token tidak dikirim atau tidak valid"
//	@Failure		404	{object}	dto.ErrorEnvelope	"catatan waktu tidak ditemukan"
//	@Failure		500	{object}	dto.ErrorEnvelope
//	@Router			/api/v1/time-entries/{id} [get]
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

// Update PUT /api/v1/time-entries/:id
//
//	@Summary		Koreksi catatan waktu
//	@Description	Update sebagian field. Dipakai terutama untuk membetulkan durasi saat lupa
//	@Description	menekan Stop: kirim `ended_at` dengan waktu yang semestinya.
//	@Description	Field yang tidak dikirim tetap memakai nilai lama. Mengirim `ended_at` pada
//	@Description	entry yang masih berjalan berarti menutupnya; sebaliknya entry yang sudah
//	@Description	ditutup tidak bisa dijalankan kembali lewat endpoint ini.
//	@Tags			Time Tracking
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int							true	"ID catatan waktu"	minimum(1)
//	@Param			request	body		dto.UpdateTimeEntryRequest	true	"Field yang mau diubah saja"
//	@Success		200		{object}	dto.TimeEntryEnvelope
//	@Failure		400		{object}	dto.ErrorEnvelope	"validasi gagal, id tidak valid, atau rentang waktu terbalik"
//	@Failure		401		{object}	dto.ErrorEnvelope	"token tidak dikirim atau tidak valid"
//	@Failure		404		{object}	dto.ErrorEnvelope	"catatan waktu atau task tidak ditemukan"
//	@Failure		500		{object}	dto.ErrorEnvelope
//	@Router			/api/v1/time-entries/{id} [put]
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

// Delete DELETE /api/v1/time-entries/:id
//
//	@Summary		Hapus catatan waktu
//	@Description	Soft delete: baris hanya diberi `deleted_at`. Menghapus timer yang sedang
//	@Description	berjalan membebaskan user untuk memulai timer baru.
//	@Tags			Time Tracking
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path		int	true	"ID catatan waktu"	minimum(1)
//	@Success		200	{object}	dto.MessageEnvelope
//	@Failure		400	{object}	dto.ErrorEnvelope	"id bukan angka positif"
//	@Failure		401	{object}	dto.ErrorEnvelope	"token tidak dikirim atau tidak valid"
//	@Failure		404	{object}	dto.ErrorEnvelope	"catatan waktu tidak ditemukan"
//	@Failure		500	{object}	dto.ErrorEnvelope
//	@Router			/api/v1/time-entries/{id} [delete]
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
