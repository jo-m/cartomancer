package tmpl

import (
	"context"
	"fmt"
	"goweb/internal/pkg/session"
	"io"
	"net/http"
	"reflect"

	"github.com/a-h/templ"
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

const fieldNameBody = "Body"

func RenderPage[T any](ctx context.Context, c templ.Component) (T, error) {
	var ret T
	fieldValue := reflect.ValueOf(&ret).Elem().FieldByName(fieldNameBody)
	if !fieldValue.IsValid() {
		panic(fmt.Sprintf("missing %s field in %T", fieldNameBody, ret))
	}

	body := readerFrom(func(w io.Writer) { c.Render(ctx, w) })
	fieldValue.Set(reflect.ValueOf(body))

	return ret, nil
}

// TODO: offer variant with custom message
func RenderErrorPage[T any](ctx context.Context, statusCode int) (T, error) {
	p := ErrorPage(session.GetUser(ctx), middleware.GetReqID(ctx), statusCode, http.StatusText(statusCode))
	return RenderPage[T](ctx, p)
}
