package endpoints

import (
	"io"
	"net/http"
)

var (
	HeaderContentType = http.CanonicalHeaderKey("Content-Type")
)

const (
	ApplicationJSON = "application/json"
	TextHTML        = "text/html"
)

func Body(w func(io.Writer)) io.Reader {
	reader, writer := io.Pipe()
	go func() {
		w(writer)
		writer.Close()
	}()

	return reader
}
