package endpoints

import (
	"context"
	"goweb/internal/pkg/endpoints/tpl"
	"goweb/internal/pkg/session"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

var (
	HeaderContentType = http.CanonicalHeaderKey("Content-Type")
)

const (
	ApplicationJSON = "application/json"
	TextHTML        = "text/html"
)

func ReaderFrom(w func(io.Writer)) io.Reader {
	reader, writer := io.Pipe()
	go func() {
		w(writer)
		writer.Close()
	}()

	return reader
}

func RenderPage(p tpl.Page) io.Reader {
	return ReaderFrom(func(w io.Writer) { tpl.WritePageTemplate(w, p) })
}

func RenderError500(ctx context.Context) io.Reader {
	p := tpl.Error500Page{
		RequestID: middleware.GetReqID(ctx),
	}
	return ReaderFrom(func(w io.Writer) { tpl.WritePageTemplate(w, &p) })
}

func BasePage(ctx context.Context) tpl.BasePage {
	return tpl.BasePage{User: session.GetUser(ctx)}
}
