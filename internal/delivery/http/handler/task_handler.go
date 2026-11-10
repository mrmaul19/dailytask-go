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

func (h *TaskHandler) Download(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}

	out, err := h.taskUsecase.Download(c.UserContext(), id)
	if err != nil {
		return err
	}

	c.Set(fiber.HeaderContentType, fiber.MIMEOctetStream)
	c.Set(fiber.HeaderContentDisposition,
		`attachment; filename="`+strings.ReplaceAll(out.FileName, `"`, "")+`"`)
	return c.SendStream(out.Content, int(out.Size))
}

func parseID(c *fiber.Ctx) (uint, error) {
	raw := c.Params("id")
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, apperror.Validation("id tidak valid", map[string]string{"id": "harus berupa angka positif"})
	}
	return uint(id), nil
}

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
