package rest

import (
	"encoding/json"
	"net/http"
	"strings"
)

var (
	headerContentType = http.CanonicalHeaderKey("Content-Type")
)

// ErrorJSON is the standard error response body.
type ErrorJSON struct {
	Msg string `json:"msg"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set(headerContentType, "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorJSON{Msg: msg})
}

func writeStatusError(w http.ResponseWriter, status int) {
	writeError(w, status, strings.ToLower(http.StatusText(status)))
}

func decodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}
