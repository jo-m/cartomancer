package api

import (
	"encoding/json"
	"errors"
	"log"
	"mime"
	"net/http"
	"strings"
)

var errUnsupportedMediaType = errors.New("Content-Type must be application/json")
var errForbidden = errors.New("forbidden")

// ErrorJSON is the standard error response body.
type ErrorJSON struct {
	Msg string `json:"msg"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set(headerContentType, "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// No ctx here.
		log.Printf("failed to encode JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorJSON{Msg: msg})
}

func writeStatusError(w http.ResponseWriter, status int) {
	writeError(w, status, strings.ToLower(http.StatusText(status)))
}

func writeDecodeError(w http.ResponseWriter, err error) {
	if errors.Is(err, errUnsupportedMediaType) {
		writeError(w, http.StatusUnsupportedMediaType, err.Error())
		return
	}
	writeError(w, http.StatusBadRequest, "invalid request body")
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func decodeJSON(r *http.Request, v any) error {
	ct := r.Header.Get(headerContentType)
	mediaType, _, _ := mime.ParseMediaType(ct)
	if mediaType != "application/json" {
		return errUnsupportedMediaType
	}
	return json.NewDecoder(r.Body).Decode(v)
}
