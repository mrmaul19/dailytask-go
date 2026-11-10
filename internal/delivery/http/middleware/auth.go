package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"daily-task-go/internal/domain/entity"
	"daily-task-go/internal/domain/security"
	"daily-task-go/pkg/apperror"
)

const (
	ContextUserID    = "userID"
	ContextUserEmail = "userEmail"
	ContextUserRole  = "userRole"
)

const bearerPrefix = "bearer "

func Auth(tokens security.TokenIssuer) fiber.Handler {
	return func(c *fiber.Ctx) error {
		raw := strings.TrimSpace(c.Get(fiber.HeaderAuthorization))
		if raw == "" {
			return apperror.Unauthorized("token tidak dikirim")
		}

		if len(raw) < len(bearerPrefix) || !strings.EqualFold(raw[:len(bearerPrefix)], bearerPrefix) {
			return apperror.Unauthorized("format Authorization harus: Bearer {token}")
		}

		value := strings.TrimSpace(raw[len(bearerPrefix):])
		if value == "" {
			return apperror.Unauthorized("token tidak dikirim")
		}

		claims, err := tokens.Parse(value)
		if err != nil {
			return apperror.Unauthorized("token tidak valid atau sudah kedaluwarsa")
		}

		c.Locals(ContextUserID, claims.UserID)
		c.Locals(ContextUserEmail, claims.Email)
		c.Locals(ContextUserRole, claims.Role)

		return c.Next()
	}
}

func RequireAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, _ := c.Locals(ContextUserRole).(string)
		if role != entity.RoleAdmin {
			return apperror.Forbidden("butuh hak admin")
		}
		return c.Next()
	}
}

func UserRoleFromContext(c *fiber.Ctx) (string, bool) {
	role, ok := c.Locals(ContextUserRole).(string)
	return role, ok
}

func UserIDFromContext(c *fiber.Ctx) (uint, bool) {
	id, ok := c.Locals(ContextUserID).(uint)
	return id, ok
}
