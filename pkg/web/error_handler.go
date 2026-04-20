package web

import (
	"log/slog"
	"net/http"

	"github.com/fastygo/framework/pkg/core"
)

// HandleError writes a uniform error response and logs it. core.DomainError
// is mapped to its StatusCode and human-readable Message; any other error
// produces 500 + "Internal Server Error" while the original error is logged
// in full under "http.error".
//
// A nil err is a no-op so handlers can safely chain `web.HandleError(w, err)`
// after the happy path.
func HandleError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	code := http.StatusInternalServerError
	message := http.StatusText(http.StatusInternalServerError)

	if domainError, ok := err.(core.DomainError); ok {
		code = domainError.StatusCode()
		message = domainError.Message
	}

	slog.Error("http.error", "error", err, "status", code)
	http.Error(w, message, code)
}
