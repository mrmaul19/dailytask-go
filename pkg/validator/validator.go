// Package validator membungkus go-playground/validator dan mengubah hasilnya
// menjadi map field -> pesan yang siap dikirim sebagai JSON.
package validator

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

type Validator struct {
	validate *validator.Validate
}

func New() *Validator {
	return &Validator{validate: validator.New()}
}

// Validate mengembalikan nil bila valid, atau map error per field.
func (v *Validator) Validate(payload any) map[string]string {
	if err := v.validate.Struct(payload); err != nil {
		var invalid *validator.InvalidValidationError
		if errors.As(err, &invalid) {
			return map[string]string{"_error": invalid.Error()}
		}

		var validationErrs validator.ValidationErrors
		if errors.As(err, &validationErrs) {
			out := make(map[string]string, len(validationErrs))
			for _, fieldErr := range validationErrs {
				field := strings.ToLower(fieldErr.Field())
				out[field] = message(fieldErr)
			}
			return out
		}
	}
	return nil
}

func message(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "wajib diisi"
	case "min":
		return fmt.Sprintf("minimal %s karakter", fe.Param())
	case "max":
		return fmt.Sprintf("maksimal %s karakter", fe.Param())
	case "gt":
		return fmt.Sprintf("harus lebih besar dari %s", fe.Param())
	default:
		return fmt.Sprintf("tidak valid (aturan: %s)", fe.Tag())
	}
}
