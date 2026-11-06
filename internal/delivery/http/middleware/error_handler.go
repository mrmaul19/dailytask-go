// Package middleware berisi lintas-concern HTTP.
package middleware

import (
	"errors"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"

	"daily-task-go/pkg/apperror"
	"daily-task-go/pkg/response"
)

// ErrorHandler adalah satu-satunya tempat error diterjemahkan ke HTTP status.
// Handler cukup `return err`, tidak perlu menulis status code di mana-mana.
func ErrorHandler(debug bool) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		// error bawaan fiber (route tidak ada, body terlalu besar, dst.)
		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) {
			return response.Error(c, fiberErr.Code, fiberErr.Message, nil)
		}

		appErr := apperror.From(err)

		status := http.StatusInternalServerError
		switch appErr.Kind {
		case apperror.KindValidation:
			status = http.StatusBadRequest
		case apperror.KindNotFound:
			status = http.StatusNotFound
		case apperror.KindConflict:
			status = http.StatusConflict
		case apperror.KindUnauthorized:
			status = http.StatusUnauthorized
		case apperror.KindForbidden:
			status = http.StatusForbidden
		}

		if status >= http.StatusInternalServerError {
			log.Printf("[ERROR] %s %s: %v", c.Method(), c.Path(), err)
		}

		details := appErr.Details
		if details == nil && debug && appErr.Err != nil {
			details = map[string]string{"debug": appErr.Err.Error()}
		}

		return response.Error(c, status, appErr.Message, details)
	}
}
