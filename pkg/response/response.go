// Package response menstandarkan bentuk JSON yang dikirim ke client.
package response

import "github.com/gofiber/fiber/v2"

type Meta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type Body struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Errors  any    `json:"errors,omitempty"`
	Meta    *Meta  `json:"meta,omitempty"`
}

func Success(c *fiber.Ctx, status int, message string, data any) error {
	return c.Status(status).JSON(Body{Success: true, Message: message, Data: data})
}

func SuccessWithMeta(c *fiber.Ctx, status int, message string, data any, meta *Meta) error {
	return c.Status(status).JSON(Body{Success: true, Message: message, Data: data, Meta: meta})
}

func Error(c *fiber.Ctx, status int, message string, errs any) error {
	return c.Status(status).JSON(Body{Success: false, Message: message, Errors: errs})
}

func NewMeta(page, limit int, total int64) *Meta {
	totalPages := 0
	if limit > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}
	return &Meta{Page: page, Limit: limit, Total: total, TotalPages: totalPages}
}
