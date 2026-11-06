// Package apperror menyediakan tipe error yang dipakai lintas layer.
// Usecase melempar error dengan Kind, lalu layer delivery menerjemahkannya
// menjadi HTTP status code. Dengan begini usecase tidak perlu tahu soal HTTP.
package apperror

import (
	"errors"
	"fmt"
)

type Kind string

const (
	KindValidation Kind = "VALIDATION_ERROR"
	KindNotFound   Kind = "NOT_FOUND"
	KindConflict   Kind = "CONFLICT"
	KindInternal   Kind = "INTERNAL_ERROR"
)

type AppError struct {
	Kind    Kind
	Message string
	Details any
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Kind, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Kind, e.Message)
}

func (e *AppError) Unwrap() error { return e.Err }

func Validation(message string, details any) *AppError {
	return &AppError{Kind: KindValidation, Message: message, Details: details}
}

func NotFound(message string) *AppError {
	return &AppError{Kind: KindNotFound, Message: message}
}

func Conflict(message string) *AppError {
	return &AppError{Kind: KindConflict, Message: message}
}

func Internal(message string, err error) *AppError {
	return &AppError{Kind: KindInternal, Message: message, Err: err}
}

// From membungkus error apa pun menjadi *AppError supaya error handler
// selalu bekerja dengan satu tipe.
func From(err error) *AppError {
	if err == nil {
		return nil
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return Internal("terjadi kesalahan pada server", err)
}
