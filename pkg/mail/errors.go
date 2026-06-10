package mail

import (
	"errors"
	"fmt"
)

// ErrorCode classifies a mail operation failure so applications can branch
// on failure class (retry, re-authenticate, surface to the user) without
// matching error strings.
type ErrorCode string

// Error codes shared across transports.
const (
	// CodeAuth means authentication or authorization failed.
	CodeAuth ErrorCode = "auth"
	// CodeNotFound means the mailbox, message, or part does not exist.
	CodeNotFound ErrorCode = "not_found"
	// CodeTooLarge means an upload or message exceeded a server limit.
	CodeTooLarge ErrorCode = "too_large"
	// CodeRateLimit means the server asked the client to slow down.
	CodeRateLimit ErrorCode = "rate_limit"
	// CodeUnavailable means the server is unreachable or returned a
	// transient failure; the operation may be retried.
	CodeUnavailable ErrorCode = "unavailable"
	// CodeProtocol means the server response violated the protocol or
	// the request was rejected as malformed. Not retryable.
	CodeProtocol ErrorCode = "protocol"
	// CodeUnsupported means the server lacks a required capability.
	CodeUnsupported ErrorCode = "unsupported"
)

// Error is the typed error returned by all Client operations.
type Error struct {
	// Op names the failed operation, e.g. "jmap: Email/query".
	Op string
	// Code classifies the failure.
	Code ErrorCode
	// Err is the underlying cause, may be nil.
	Err error
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("mail: %s: %s", e.Op, e.Code)
	}
	return fmt.Sprintf("mail: %s: %s: %v", e.Op, e.Code, e.Err)
}

// Unwrap exposes the underlying cause to errors.Is / errors.As.
func (e *Error) Unwrap() error { return e.Err }

// CodeOf extracts the ErrorCode from err. It returns the code of the
// outermost *Error in the chain, or CodeUnavailable for plain errors so
// transient-looking failures default to the retryable class.
func CodeOf(err error) ErrorCode {
	var me *Error
	if errors.As(err, &me) {
		return me.Code
	}
	return CodeUnavailable
}
