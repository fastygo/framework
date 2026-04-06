package web

import (
	"log/slog"
	"net/http"

	"github.com/fastygo/framework/pkg/core"
)

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
