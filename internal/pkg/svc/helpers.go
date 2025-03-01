package svc

import (
	"io"
	"net/http"
	"strings"
)

var (
	HeaderContentType = http.CanonicalHeaderKey("Content-Type")
)

const (
	ApplicationJSON = "application/json"
	TextHTML        = "text/html"
)

func WantsJSON(acceptHeader string) bool {
	// TODO: make this more RFC compliant
	return strings.Contains(acceptHeader, ApplicationJSON)
}

func Body(w func(io.Writer)) io.Reader {
	reader, writer := io.Pipe()
	go func() {
		w(writer)
		writer.Close()
	}()

	return reader
}
