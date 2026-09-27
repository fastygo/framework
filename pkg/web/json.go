package web

import (
	"encoding/json"
	"net/http"
)

// WriteJSON sets a JSON content type, writes status, and JSON-encodes
// payload. It does not buffer: a marshalling failure mid-stream produces
// a partial body — keep payload simple (no custom MarshalJSON that may panic).
func WriteJSON(w http.ResponseWriter, status int, payload any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	return enc.Encode(payload)
}
