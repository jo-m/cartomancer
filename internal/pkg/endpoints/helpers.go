package endpoints

import (
	"context"
	"goweb/internal/pkg/endpoints/tpl"
	"goweb/internal/pkg/session"
	"io"

	"github.com/go-chi/chi/v5/middleware"
)

func readerFrom(w func(io.Writer)) io.Reader {
	reader, writer := io.Pipe()
	go func() {
		w(writer)
		writer.Close()
	}()

	return reader
}

func renderPage(p tpl.Page) io.Reader {
	return readerFrom(func(w io.Writer) { tpl.WritePageTemplate(w, p) })
}

func renderError500(ctx context.Context) io.Reader {
	p := tpl.Error500Page{
		RequestID: middleware.GetReqID(ctx),
	}
	return readerFrom(func(w io.Writer) { tpl.WritePageTemplate(w, &p) })
}

func basePage(ctx context.Context) tpl.BasePage {
	return tpl.BasePage{User: session.GetUser(ctx)}
}
