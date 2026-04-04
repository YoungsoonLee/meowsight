package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Standard sentinel errors.
var (
	ErrNotFound       = errors.New("not found")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrForbidden      = errors.New("forbidden")
	ErrBadRequest     = errors.New("bad request")
	ErrConflict       = errors.New("conflict")
	ErrInternal       = errors.New("internal error")
	ErrBudgetExceeded = errors.New("budget exceeded")
	ErrLockHeld       = errors.New("lock already held")
)

// AppError wraps an error with an HTTP status code and user-facing message.
type AppError struct {
	Code    int
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NotFound(msg string) *AppError {
	return &AppError{Code: http.StatusNotFound, Message: msg, Err: ErrNotFound}
}

func Unauthorized(msg string) *AppError {
	return &AppError{Code: http.StatusUnauthorized, Message: msg, Err: ErrUnauthorized}
}

func BadRequest(msg string) *AppError {
	return &AppError{Code: http.StatusBadRequest, Message: msg, Err: ErrBadRequest}
}

func Internal(msg string, err error) *AppError {
	return &AppError{Code: http.StatusInternalServerError, Message: msg, Err: err}
}

func PlanLimitExceeded(resource string, limit int) *AppError {
	return &AppError{
		Code:    http.StatusForbidden,
		Message: fmt.Sprintf("plan limit exceeded: %s (max %d)", resource, limit),
		Err:     ErrForbidden,
	}
}
