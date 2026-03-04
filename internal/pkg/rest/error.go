package rest

import (
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"strings"
)

var errUnsupportedMediaType = errors.New("Content-Type must be application/json")

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

func writeDecodeError(w http.ResponseWriter, err error) {
	if errors.Is(err, errUnsupportedMediaType) {
		writeError(w, http.StatusUnsupportedMediaType, err.Error())
		return
	}
	writeError(w, http.StatusBadRequest, "invalid request body")
}

func decodeJSON(r *http.Request, v any) error {
	ct := r.Header.Get(headerContentType)
	if ct != "" {
		mediaType, _, _ := mime.ParseMediaType(ct)
		if mediaType != "application/json" {
			return errUnsupportedMediaType
		}
	}
	return json.NewDecoder(r.Body).Decode(v)
}
