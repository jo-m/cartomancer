package endpoints

import (
	"context"
	"goweb/internal/pkg/endpoints/tpl"
	"goweb/internal/pkg/oapi"
	"goweb/internal/pkg/session"
	"io"
	"net/http"
	"reflect"

	"github.com/go-chi/chi/v5/middleware"
)

var (
	headerContentType = http.CanonicalHeaderKey("Content-Type")
)

const (
	applicationJSON = "application/json"
	textHTML        = "text/html"
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

func mkErr(err, scope string, details ...oapi.ErrorJSON) oapi.ErrorJSON {
	scp := &scope
	if scope == "" {
		scp = nil
	}

	det := &details
	if len(details) == 0 {
		det = nil
	}

	return oapi.ErrorJSON{
		Error:   err,
		Scope:   scp,
		Details: det,
	}
}

func mk500[T any]() T {
	var ret T
	typ := reflect.TypeOf(ret)

	err := mkErr(http.StatusText(500), "")
	embed := oapi.N500InternalServerErrorJSONResponse(err)
	embedTyp := reflect.TypeOf(embed)

	for i := range typ.NumField() {
		field := typ.Field(i)
		if field.Type == embedTyp {
			ptr := reflect.ValueOf(&ret).Elem().Field(i)
			ptr.Set(reflect.ValueOf(embed))
		}
	}

	return ret
}

func mk404[T any]() T {
	var ret T
	typ := reflect.TypeOf(ret)

	err := mkErr(http.StatusText(404), "")
	embed := oapi.N404NotFoundJSONResponse(err)
	embedTyp := reflect.TypeOf(embed)

	for i := range typ.NumField() {
		field := typ.Field(i)
		if field.Type == embedTyp {
			ptr := reflect.ValueOf(&ret).Elem().Field(i)
			ptr.Set(reflect.ValueOf(embed))
		}
	}

	return ret
}
