// Package handler adalah adapter HTTP: terjemahkan request -> input usecase,
// lalu output usecase -> response JSON. Tidak ada logika bisnis di sini.
package handler

import (
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"daily-task-go/internal/delivery/http/dto"
	domainstorage "daily-task-go/internal/domain/storage"
	"daily-task-go/internal/usecase"
	"daily-task-go/pkg/apperror"
	"daily-task-go/pkg/response"
	"daily-task-go/pkg/validator"
)

const fileFieldName = "file"

type TaskHandler struct {
	taskUsecase usecase.TaskUsecase
	validate    *validator.Validator
	baseURL     string
}

func NewTaskHandler(taskUsecase usecase.TaskUsecase, validate *validator.Validator, baseURL string) *TaskHandler {
	return &TaskHandler{taskUsecase: taskUsecase, validate: validate, baseURL: baseURL}
}

// Create POST /api/v1/tasks  (multipart/form-data atau application/json)
//
//	@Summary		Buat task baru
//	@Description	Membuat task baru, dengan atau tanpa lampiran file.
//	@Description	Kirim sebagai `multipart/form-data` kalau ada file, atau `application/json` kalau tanpa file.
//	@Description	`no_task` boleh dikosongkan — server akan membuatnya dengan format `TASK-YYYYMMDD-XXXX`.
//	@Tags			Tasks
//	@Accept			mpfd,json
//	@Produce		json
//	@Param			no_task		formData	string	false	"Nomor task, wajib unik. Digenerate otomatis kalau kosong"	maxlength(50)
//	@Param			judul		formData	string	true	"Judul task"												minlength(3)	maxlength(255)
//	@Param			description	formData	string	false	"Keterangan task"											maxlength(2000)
//	@Param			file		formData	file	false	"Lampiran. Ekstensi & ukuran dibatasi ALLOWED_EXT / MAX_UPLOAD_MB"
//	@Success		201			{object}	dto.TaskEnvelope
//	@Failure		400			{object}	dto.ErrorEnvelope	"validasi gagal atau ekstensi/ukuran file ditolak"
//	@Failure		409			{object}	dto.ErrorEnvelope	"no_task sudah dipakai task lain"
//	@Failure		500			{object}	dto.ErrorEnvelope
//	@Router			/api/v1/tasks [post]
func (h *TaskHandler) Create(c *fiber.Ctx) error {
	var req dto.CreateTaskRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.Validation("format request tidak valid", map[string]string{"body": err.Error()})
	}

	if errs := h.validate.Validate(req); errs != nil {
		return apperror.Validation("data tidak valid", errs)
	}

	upload, closeFile, err := h.readUpload(c)
	if err != nil {
		return err
	}
	defer closeFile()

	task, err := h.taskUsecase.Create(c.UserContext(), usecase.CreateTaskInput{
		NoTask:      req.NoTask,
		Judul:       req.Judul,
		Description: req.Description,
		File:        upload,
	})
	if err != nil {
		return err
	}

	return response.Success(c, http.StatusCreated, "task berhasil dibuat",
		dto.NewTaskResponse(task, h.baseURL))
}

// List GET /api/v1/tasks?search=&page=&limit=
//
//	@Summary		Daftar task
//	@Description	Mengambil daftar task dengan pencarian dan paginasi.
//	@Description	`search` dicocokkan ke `no_task` dan `judul` (case-insensitive, pakai ILIKE).
//	@Tags			Tasks
//	@Produce		json
//	@Param			search	query		string	false	"Kata kunci pencarian"
//	@Param			page	query		int		false	"Halaman, mulai dari 1"	default(1)	minimum(1)
//	@Param			limit	query		int		false	"Jumlah data per halaman"	default(10)	minimum(1)	maximum(100)
//	@Success		200		{object}	dto.TaskListEnvelope
//	@Failure		500		{object}	dto.ErrorEnvelope
//	@Router			/api/v1/tasks [get]
func (h *TaskHandler) List(c *fiber.Ctx) error {
	out, err := h.taskUsecase.List(c.UserContext(), usecase.ListTaskInput{
		Search: c.Query("search"),
		Page:   c.QueryInt("page", 1),
		Limit:  c.QueryInt("limit", 10),
	})
	if err != nil {
		return err
	}

	return response.SuccessWithMeta(c, http.StatusOK, "daftar task berhasil diambil",
		dto.NewTaskResponses(out.Tasks, h.baseURL),
		response.NewMeta(out.Page, out.Limit, out.Total))
}

// GetByID GET /api/v1/tasks/:id
//
//	@Summary		Detail task
//	@Description	Mengambil satu task berdasarkan id. Task yang sudah dihapus (soft delete) tidak ikut terambil.
//	@Tags			Tasks
//	@Produce		json
//	@Param			id	path		int	true	"ID task"	minimum(1)
//	@Success		200	{object}	dto.TaskEnvelope
//	@Failure		400	{object}	dto.ErrorEnvelope	"id bukan angka positif"
//	@Failure		404	{object}	dto.ErrorEnvelope	"task tidak ditemukan"
//	@Failure		500	{object}	dto.ErrorEnvelope
//	@Router			/api/v1/tasks/{id} [get]
func (h *TaskHandler) GetByID(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}

	task, err := h.taskUsecase.GetByID(c.UserContext(), id)
	if err != nil {
		return err
	}

	return response.Success(c, http.StatusOK, "detail task berhasil diambil",
		dto.NewTaskResponse(task, h.baseURL))
}

// Update PUT /api/v1/tasks/:id
//
//	@Summary		Perbarui task
//	@Description	Update sebagian field (partial update). Field yang tidak dikirim tetap memakai nilai lama,
//	@Description	field yang dikirim kosong ditolak dengan 400.
//	@Description	Mengirim `file` baru akan menghapus lampiran lama dari disk setelah DB sukses;
//	@Description	`remove_file=true` melepas lampiran tanpa menggantinya.
//	@Tags			Tasks
//	@Accept			mpfd,json
//	@Produce		json
//	@Param			id			path		int		true	"ID task"	minimum(1)
//	@Param			no_task		formData	string	false	"Nomor task, wajib unik"	maxlength(50)
//	@Param			judul		formData	string	false	"Judul task"				minlength(3)	maxlength(255)
//	@Param			description	formData	string	false	"Keterangan task"			maxlength(2000)
//	@Param			file		formData	file	false	"Lampiran pengganti"
//	@Param			remove_file	formData	bool	false	"true untuk melepas lampiran yang ada"
//	@Success		200			{object}	dto.TaskEnvelope
//	@Failure		400			{object}	dto.ErrorEnvelope	"validasi gagal, id tidak valid, atau file ditolak"
//	@Failure		404			{object}	dto.ErrorEnvelope	"task tidak ditemukan"
//	@Failure		409			{object}	dto.ErrorEnvelope	"no_task sudah dipakai task lain"
//	@Failure		500			{object}	dto.ErrorEnvelope
//	@Router			/api/v1/tasks/{id} [put]
func (h *TaskHandler) Update(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}

	req, err := h.bindUpdate(c)
	if err != nil {
		return err
	}

	if errs := h.validate.Validate(req); errs != nil {
		return apperror.Validation("data tidak valid", errs)
	}

	upload, closeFile, err := h.readUpload(c)
	if err != nil {
		return err
	}
	defer closeFile()

	task, err := h.taskUsecase.Update(c.UserContext(), id, usecase.UpdateTaskInput{
		NoTask:      req.NoTask,
		Judul:       req.Judul,
		Description: req.Description,
		File:        upload,
		RemoveFile:  req.RemoveFile,
	})
	if err != nil {
		return err
	}

	return response.Success(c, http.StatusOK, "task berhasil diperbarui",
		dto.NewTaskResponse(task, h.baseURL))
}

// Delete DELETE /api/v1/tasks/:id
//
//	@Summary		Hapus task
//	@Description	Soft delete: baris hanya diberi `deleted_at`, tetapi lampiran di disk dihapus permanen.
//	@Tags			Tasks
//	@Produce		json
//	@Param			id	path		int	true	"ID task"	minimum(1)
//	@Success		200	{object}	dto.MessageEnvelope
//	@Failure		400	{object}	dto.ErrorEnvelope	"id bukan angka positif"
//	@Failure		404	{object}	dto.ErrorEnvelope	"task tidak ditemukan"
//	@Failure		500	{object}	dto.ErrorEnvelope
//	@Router			/api/v1/tasks/{id} [delete]
func (h *TaskHandler) Delete(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}

	if err := h.taskUsecase.Delete(c.UserContext(), id); err != nil {
		return err
	}

	return response.Success(c, http.StatusOK, "task berhasil dihapus", nil)
}

// Download GET /api/v1/tasks/:id/download
//
//	@Summary		Unduh lampiran task
//	@Description	Mengirim file lampiran sebagai stream. File tidak diserve lewat folder static:
//	@Description	nama file di disk diacak dan path-nya dibaca dari DB, jadi URL-nya tidak bisa ditebak.
//	@Tags			Tasks
//	@Produce		octet-stream
//	@Param			id	path		int		true	"ID task"	minimum(1)
//	@Success		200	{file}		binary	"isi file lampiran"
//	@Failure		400	{object}	dto.ErrorEnvelope	"id bukan angka positif"
//	@Failure		404	{object}	dto.ErrorEnvelope	"task tidak ditemukan atau tidak punya lampiran"
//	@Failure		500	{object}	dto.ErrorEnvelope
//	@Router			/api/v1/tasks/{id}/download [get]
func (h *TaskHandler) Download(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}

	out, err := h.taskUsecase.Download(c.UserContext(), id)
	if err != nil {
		return err
	}

	// fasthttp menutup stream ini otomatis setelah response terkirim.
	c.Set(fiber.HeaderContentType, fiber.MIMEOctetStream)
	c.Set(fiber.HeaderContentDisposition,
		`attachment; filename="`+strings.ReplaceAll(out.FileName, `"`, "")+`"`)
	return c.SendStream(out.Content, int(out.Size))
}

// ---------- helper ----------

func parseID(c *fiber.Ctx) (uint, error) {
	raw := c.Params("id")
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, apperror.Validation("id tidak valid", map[string]string{"id": "harus berupa angka positif"})
	}
	return uint(id), nil
}

// readUpload mengambil file dari form (kalau ada) dan mengubahnya menjadi
// domainstorage.Upload. closeFile selalu aman dipanggil.
func (h *TaskHandler) readUpload(c *fiber.Ctx) (*domainstorage.Upload, func(), error) {
	noop := func() {}

	header, err := c.FormFile(fileFieldName)
	if err != nil || header == nil {
		return nil, noop, nil // field file memang opsional
	}

	src, err := header.Open()
	if err != nil {
		return nil, noop, apperror.Validation("file gagal dibaca", map[string]string{"file": err.Error()})
	}

	upload := &domainstorage.Upload{
		FileName: header.Filename,
		Size:     header.Size,
		Content:  src,
	}
	return upload, func() { _ = src.Close() }, nil
}

// bindUpdate memakai pointer agar bisa membedakan "field tidak dikirim"
// (biarkan nilai lama) dari "field dikirim kosong".
func (h *TaskHandler) bindUpdate(c *fiber.Ctx) (dto.UpdateTaskRequest, error) {
	var req dto.UpdateTaskRequest

	if strings.Contains(strings.ToLower(c.Get(fiber.HeaderContentType)), fiber.MIMEApplicationJSON) {
		if err := c.BodyParser(&req); err != nil {
			return req, apperror.Validation("format request tidak valid", map[string]string{"body": err.Error()})
		}
		return req, nil
	}

	form, err := c.MultipartForm()
	if err != nil {
		return req, apperror.Validation("format request tidak valid",
			map[string]string{"body": "gunakan multipart/form-data atau application/json"})
	}

	req.NoTask = formValue(form, "no_task")
	req.Judul = formValue(form, "judul")
	req.Description = formValue(form, "description")
	if v := formValue(form, "remove_file"); v != nil {
		req.RemoveFile = *v == "true" || *v == "1"
	}

	return req, nil
}

func formValue(form *multipart.Form, key string) *string {
	values, ok := form.Value[key]
	if !ok || len(values) == 0 {
		return nil
	}
	return &values[0]
}
