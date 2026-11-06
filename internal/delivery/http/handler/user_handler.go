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

// Create POST /api/v1/users
//
//	@Summary		Buat user baru
//	@Description	Semua field wajib diisi. `email` harus unik (dibandingkan tanpa membedakan huruf besar/kecil)
//	@Description	dan `password` minimal 8 karakter, maksimal 72.
//	@Description	Password disimpan sebagai hash bcrypt dan tidak pernah ikut dikembalikan di response.
//	@Tags			Users
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.CreateUserRequest	true	"Data user"
//	@Success		201		{object}	dto.UserEnvelope
//	@Failure		400		{object}	dto.ErrorEnvelope	"validasi gagal"
//	@Failure		401		{object}	dto.ErrorEnvelope	"token tidak dikirim atau tidak valid"
//	@Failure		403		{object}	dto.ErrorEnvelope	"butuh hak admin"
//	@Failure		409		{object}	dto.ErrorEnvelope	"email sudah dipakai user lain"
//	@Failure		500		{object}	dto.ErrorEnvelope
//	@Router			/api/v1/users [post]
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

// List GET /api/v1/users?search=&page=&limit=
//
//	@Summary		Daftar user
//	@Description	`search` dicocokkan ke `name` dan `job` (case-insensitive, pakai ILIKE).
//	@Tags			Users
//	@Security		BearerAuth
//	@Produce		json
//	@Param			search	query		string	false	"Kata kunci pencarian"
//	@Param			page	query		int		false	"Halaman, mulai dari 1"	default(1)	minimum(1)
//	@Param			limit	query		int		false	"Jumlah data per halaman"	default(10)	minimum(1)	maximum(100)
//	@Success		200		{object}	dto.UserListEnvelope
//	@Failure		401		{object}	dto.ErrorEnvelope	"token tidak dikirim atau tidak valid"
//	@Failure		403		{object}	dto.ErrorEnvelope	"butuh hak admin"
//	@Failure		500		{object}	dto.ErrorEnvelope
//	@Router			/api/v1/users [get]
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

// GetByID GET /api/v1/users/:id
//
//	@Summary		Detail user
//	@Description	User yang sudah dihapus (soft delete) tidak ikut terambil.
//	@Tags			Users
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path		int	true	"ID user"	minimum(1)
//	@Success		200	{object}	dto.UserEnvelope
//	@Failure		400	{object}	dto.ErrorEnvelope	"id bukan angka positif"
//	@Failure		401	{object}	dto.ErrorEnvelope	"token tidak dikirim atau tidak valid"
//	@Failure		403	{object}	dto.ErrorEnvelope	"butuh hak admin"
//	@Failure		404	{object}	dto.ErrorEnvelope	"user tidak ditemukan"
//	@Failure		500	{object}	dto.ErrorEnvelope
//	@Router			/api/v1/users/{id} [get]
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

// Update PUT /api/v1/users/:id
//
//	@Summary		Perbarui user
//	@Description	Update sebagian field (partial update). Field yang tidak dikirim tetap memakai nilai lama,
//	@Description	field yang dikirim kosong ditolak dengan 400.
//	@Description	Kirim `password` hanya kalau memang ingin menggantinya — password lama tetap dipakai kalau field ini tidak ada.
//	@Tags			Users
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int						true	"ID user"	minimum(1)
//	@Param			request	body		dto.UpdateUserRequest	true	"Field yang mau diubah saja"
//	@Success		200		{object}	dto.UserEnvelope
//	@Failure		400		{object}	dto.ErrorEnvelope	"validasi gagal atau id tidak valid"
//	@Failure		401		{object}	dto.ErrorEnvelope	"token tidak dikirim atau tidak valid"
//	@Failure		403		{object}	dto.ErrorEnvelope	"butuh hak admin"
//	@Failure		404		{object}	dto.ErrorEnvelope	"user tidak ditemukan"
//	@Failure		409		{object}	dto.ErrorEnvelope	"email sudah dipakai user lain"
//	@Failure		500		{object}	dto.ErrorEnvelope
//	@Router			/api/v1/users/{id} [put]
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

// Delete DELETE /api/v1/users/:id
//
//	@Summary		Hapus user
//	@Description	Soft delete: baris hanya diberi `deleted_at`, tidak benar-benar dihapus dari tabel.
//	@Tags			Users
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path		int	true	"ID user"	minimum(1)
//	@Success		200	{object}	dto.MessageEnvelope
//	@Failure		400	{object}	dto.ErrorEnvelope	"id bukan angka positif"
//	@Failure		401	{object}	dto.ErrorEnvelope	"token tidak dikirim atau tidak valid"
//	@Failure		403	{object}	dto.ErrorEnvelope	"butuh hak admin"
//	@Failure		404	{object}	dto.ErrorEnvelope	"user tidak ditemukan"
//	@Failure		500	{object}	dto.ErrorEnvelope
//	@Router			/api/v1/users/{id} [delete]
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
