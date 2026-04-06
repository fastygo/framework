package core

import (
	"fmt"
	"net/http"
)

type ErrorCode string

const (
	ErrorCodeNotFound   ErrorCode = "not_found"
	ErrorCodeConflict   ErrorCode = "conflict"
	ErrorCodeValidation ErrorCode = "validation"
	ErrorCodeUnauthorized ErrorCode = "unauthorized"
	ErrorCodeForbidden  ErrorCode = "forbidden"
	ErrorCodeInternal   ErrorCode = "internal"
)

type DomainError struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func NewDomainError(code ErrorCode, message string) DomainError {
	return DomainError{
		Code:    code,
		Message: message,
	}
}

func WrapDomainError(code ErrorCode, message string, cause error) DomainError {
	return DomainError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

func (e DomainError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Cause.Error())
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e DomainError) StatusCode() int {
	switch e.Code {
	case ErrorCodeNotFound:
		return http.StatusNotFound
	case ErrorCodeConflict:
		return http.StatusConflict
	case ErrorCodeValidation:
		return http.StatusBadRequest
	case ErrorCodeUnauthorized:
		return http.StatusUnauthorized
	case ErrorCodeForbidden:
		return http.StatusForbidden
	case ErrorCodeInternal:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
