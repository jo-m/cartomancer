package endpoints

import (
	"context"
	"goweb/internal/pkg/endpoints/tpl"
	"goweb/internal/pkg/session"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

// TODO: rename
func readerFrom(w func(io.Writer)) io.Reader {
	reader, writer := io.Pipe()
	go func() {
		w(writer)
		writer.Close()
	}()

	return reader
}

// TODO: rename
func renderPage(p tpl.Page) io.Reader {
	return readerFrom(func(w io.Writer) { tpl.WritePageTemplate(w, p) })
}

// TODO: rename
func renderError(ctx context.Context, statusCode int) io.Reader {
	p := tpl.ErrorPage{
		RequestID:  middleware.GetReqID(ctx),
		StatusCode: statusCode,
		Error:      http.StatusText(statusCode),
	}
	return readerFrom(func(w io.Writer) { tpl.WritePageTemplate(w, &p) })
}

// TODO: rename
func basePage(ctx context.Context) tpl.BasePage {
	return tpl.BasePage{User: session.GetUser(ctx)}
}
