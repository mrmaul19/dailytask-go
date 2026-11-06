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

// Register POST /api/v1/auth/register
//
//	@Summary		Daftar akun baru
//	@Description	Endpoint publik untuk mendaftar sendiri. Langsung mengembalikan token,
//	@Description	jadi client tidak perlu memanggil login lagi setelah berhasil daftar.
//	@Description	Status akun ditentukan server (`aktif`) dan tidak bisa dikirim dari sini —
//	@Description	gunakan `POST /api/v1/users` kalau ingin menentukannya sendiri.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.RegisterRequest	true	"Data pendaftaran"
//	@Success		201		{object}	dto.AuthEnvelope
//	@Failure		400		{object}	dto.ErrorEnvelope	"validasi gagal"
//	@Failure		409		{object}	dto.ErrorEnvelope	"email sudah terdaftar"
//	@Failure		429		{object}	dto.ErrorEnvelope	"terlalu banyak percobaan"
//	@Failure		500		{object}	dto.ErrorEnvelope
//	@Router			/api/v1/auth/register [post]
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

// Login POST /api/v1/auth/login
//
//	@Summary		Login dan ambil token
//	@Description	Mengembalikan token akses yang dipakai sebagai header `Authorization: Bearer {token}`.
//	@Description	Email yang tidak terdaftar dan password yang salah dibalas 401 dengan pesan yang sama persis —
//	@Description	disengaja, supaya endpoint ini tidak bisa dipakai memeriksa email mana saja yang punya akun.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.LoginRequest	true	"Email dan password"
//	@Success		200		{object}	dto.AuthEnvelope
//	@Failure		400		{object}	dto.ErrorEnvelope	"validasi gagal"
//	@Failure		401		{object}	dto.ErrorEnvelope	"email atau password salah"
//	@Failure		429		{object}	dto.ErrorEnvelope	"terlalu banyak percobaan"
//	@Failure		500		{object}	dto.ErrorEnvelope
//	@Router			/api/v1/auth/login [post]
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
