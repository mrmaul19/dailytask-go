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

type AuthHandler struct {
	authUsecase usecase.AuthUsecase
	validate    *validator.Validator
}

func NewAuthHandler(authUsecase usecase.AuthUsecase, validate *validator.Validator) *AuthHandler {
	return &AuthHandler{authUsecase: authUsecase, validate: validate}
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req dto.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.Validation("format request tidak valid", map[string]string{"body": err.Error()})
	}

	if errs := h.validate.Validate(req); errs != nil {
		return apperror.Validation("data tidak valid", errs)
	}

	out, err := h.authUsecase.Register(c.UserContext(), usecase.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
		Address:  req.Address,
		Job:      req.Job,
	})
	if err != nil {
		return err
	}

	return response.Success(c, http.StatusCreated, "registrasi berhasil",
		dto.NewAuthResponse(out.User, out.Token))
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req dto.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return apperror.Validation("format request tidak valid", map[string]string{"body": err.Error()})
	}

	if errs := h.validate.Validate(req); errs != nil {
		return apperror.Validation("data tidak valid", errs)
	}

	out, err := h.authUsecase.Login(c.UserContext(), usecase.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return err
	}

	return response.Success(c, http.StatusOK, "login berhasil",
		dto.NewAuthResponse(out.User, out.Token))
}
