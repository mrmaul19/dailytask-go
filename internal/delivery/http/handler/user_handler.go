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

type UserHandler struct {
	userUsecase usecase.UserUsecase
	validate    *validator.Validator
}

func NewUserHandler(userUsecase usecase.UserUsecase, validate *validator.Validator) *UserHandler {
	return &UserHandler{userUsecase: userUsecase, validate: validate}
}

func (h *UserHandler) Create(c *fiber.Ctx) error {
	var req dto.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.Validation("format request tidak valid", map[string]string{"body": err.Error()})
	}

	if errs := h.validate.Validate(req); errs != nil {
		return apperror.Validation("data tidak valid", errs)
	}

	user, err := h.userUsecase.Create(c.UserContext(), usecase.CreateUserInput{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
		Address:  req.Address,
		Job:      req.Job,
		Status:   req.Status,
		Role:     req.Role,
	})
	if err != nil {
		return err
	}

	return response.Success(c, http.StatusCreated, "user berhasil dibuat",
		dto.NewUserResponse(user))
}

func (h *UserHandler) List(c *fiber.Ctx) error {
	out, err := h.userUsecase.List(c.UserContext(), usecase.ListUserInput{
		Search: c.Query("search"),
		Page:   c.QueryInt("page", 1),
		Limit:  c.QueryInt("limit", 10),
	})
	if err != nil {
		return err
	}

	return response.SuccessWithMeta(c, http.StatusOK, "daftar user berhasil diambil",
		dto.NewUserResponses(out.Users),
		response.NewMeta(out.Page, out.Limit, out.Total))
}

func (h *UserHandler) GetByID(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}

	user, err := h.userUsecase.GetByID(c.UserContext(), id)
	if err != nil {
		return err
	}

	return response.Success(c, http.StatusOK, "detail user berhasil diambil",
		dto.NewUserResponse(user))
}

func (h *UserHandler) Update(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}

	var req dto.UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.Validation("format request tidak valid", map[string]string{"body": err.Error()})
	}

	if errs := h.validate.Validate(req); errs != nil {
		return apperror.Validation("data tidak valid", errs)
	}

	user, err := h.userUsecase.Update(c.UserContext(), id, usecase.UpdateUserInput{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
		Address:  req.Address,
		Job:      req.Job,
		Status:   req.Status,
		Role:     req.Role,
	})
	if err != nil {
		return err
	}

	return response.Success(c, http.StatusOK, "user berhasil diperbarui",
		dto.NewUserResponse(user))
}

func (h *UserHandler) Delete(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}

	if err := h.userUsecase.Delete(c.UserContext(), id); err != nil {
		return err
	}

	return response.Success(c, http.StatusOK, "user berhasil dihapus", nil)
}
