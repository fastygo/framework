package core

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestNewDomainError_RendersCodeAndMessage(t *testing.T) {
	err := NewDomainError(ErrorCodeNotFound, "user missing")

	if err.Code != ErrorCodeNotFound {
		t.Errorf("Code: got %q, want %q", err.Code, ErrorCodeNotFound)
	}
	if err.Message != "user missing" {
		t.Errorf("Message: got %q, want %q", err.Message, "user missing")
	}
	if err.Cause != nil {
		t.Errorf("Cause: got %v, want nil", err.Cause)
	}

	got := err.Error()
	want := "not_found: user missing"
	if got != want {
		t.Fatalf("Error(): got %q, want %q", got, want)
	}
}

func TestWrapDomainError_PreservesCauseAndRenders(t *testing.T) {
	root := errors.New("rows scanned: invalid syntax")
	err := WrapDomainError(ErrorCodeInternal, "db query failed", root)

	if err.Cause != root {
		t.Errorf("Cause: got %v, want %v", err.Cause, root)
	}

	got := err.Error()
	if !strings.HasPrefix(got, "internal: db query failed") {
		t.Errorf("Error() prefix: got %q", got)
	}
	if !strings.Contains(got, "rows scanned: invalid syntax") {
		t.Errorf("Error() must contain the cause text, got %q", got)
	}
}

func TestDomainError_StatusCode_AllCodes(t *testing.T) {
	tests := []struct {
		code   ErrorCode
		status int
	}{
		{ErrorCodeNotFound, http.StatusNotFound},
		{ErrorCodeConflict, http.StatusConflict},
		{ErrorCodeValidation, http.StatusBadRequest},
		{ErrorCodeUnauthorized, http.StatusUnauthorized},
		{ErrorCodeForbidden, http.StatusForbidden},
		{ErrorCodeInternal, http.StatusInternalServerError},
		{ErrorCode("totally-unknown"), http.StatusInternalServerError},
		{ErrorCode(""), http.StatusInternalServerError},
	}
	for _, tc := range tests {
		t.Run(string(tc.code), func(t *testing.T) {
			got := NewDomainError(tc.code, "x").StatusCode()
			if got != tc.status {
				t.Errorf("StatusCode for %q: got %d, want %d", tc.code, got, tc.status)
			}
		})
	}
}

// sentinelCause is a typed error used to verify errors.As traversal
// through DomainError.
type sentinelCause struct{ tag string }

func (s sentinelCause) Error() string { return "sentinel:" + s.tag }

func TestDomainError_ErrorsAs_TraversesCause(t *testing.T) {
	root := sentinelCause{tag: "boom"}
	wrapped := WrapDomainError(ErrorCodeInternal, "outer", root)

	var got sentinelCause
	if !errors.As(wrapped, &got) {
		t.Fatalf("errors.As must traverse DomainError.Cause via Unwrap")
	}
	if got.tag != "boom" {
		t.Fatalf("traversed cause tag: got %q, want %q", got.tag, "boom")
	}
}

func TestDomainError_ErrorsIs_MatchesCause(t *testing.T) {
	root := errors.New("disk full")
	wrapped := WrapDomainError(ErrorCodeInternal, "persist failed", root)

	if !errors.Is(wrapped, root) {
		t.Fatalf("errors.Is must match the wrapped cause")
	}
}

func TestDomainError_Unwrap_NilWhenNoCause(t *testing.T) {
	err := NewDomainError(ErrorCodeNotFound, "missing")
	if err.Unwrap() != nil {
		t.Fatalf("Unwrap on cause-less DomainError must be nil, got %v", err.Unwrap())
	}
}
