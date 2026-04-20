package core

import (
	"fmt"
	"net/http"
)

// ErrorCode classifies a DomainError. Codes are stable strings so
// they can be safely compared, logged, and returned to clients.
type ErrorCode string

// Stable error codes. Use the matching constructors instead of
// constructing DomainError literals so log analytics and HTTP
// status mapping stay consistent.
const (
	// ErrorCodeNotFound — the requested entity does not exist.
	ErrorCodeNotFound ErrorCode = "not_found"
	// ErrorCodeConflict — the request collides with current state
	// (e.g. uniqueness violation, optimistic-lock failure).
	ErrorCodeConflict ErrorCode = "conflict"
	// ErrorCodeValidation — input failed validation.
	ErrorCodeValidation ErrorCode = "validation"
	// ErrorCodeUnauthorized — caller did not authenticate.
	ErrorCodeUnauthorized ErrorCode = "unauthorized"
	// ErrorCodeForbidden — caller authenticated but lacks permission.
	ErrorCodeForbidden ErrorCode = "forbidden"
	// ErrorCodeInternal — unexpected server-side failure.
	ErrorCodeInternal ErrorCode = "internal"
)

// DomainError is the canonical typed error used across feature code.
// It carries a stable Code (for analytics and HTTP status mapping),
// a human-readable Message, and an optional Cause for error chains.
type DomainError struct {
	// Code classifies the error and drives StatusCode mapping.
	Code ErrorCode
	// Message is a human-readable explanation safe to log.
	Message string
	// Cause is the underlying error if any. Nil if the DomainError
	// is itself the root cause.
	Cause error
}

// NewDomainError constructs a DomainError without a cause.
func NewDomainError(code ErrorCode, message string) DomainError {
	return DomainError{
		Code:    code,
		Message: message,
	}
}

// WrapDomainError constructs a DomainError that wraps cause for
// later inspection via errors.As / errors.Unwrap.
func WrapDomainError(code ErrorCode, message string, cause error) DomainError {
	return DomainError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// Error renders the DomainError in the form "<code>: <message>"
// (and "<code>: <message> (<cause>)" when Cause is non-nil).
func (e DomainError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Cause.Error())
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap exposes the wrapped Cause for errors.As / errors.Is /
// errors.Unwrap traversal. Returns nil for DomainError values that
// were constructed without a cause (NewDomainError).
func (e DomainError) Unwrap() error {
	return e.Cause
}

// StatusCode maps the DomainError to an HTTP status code. Unknown
// codes (and ErrorCodeInternal) return 500.
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
