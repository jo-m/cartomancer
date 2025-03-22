package tmpl

import (
	"context"
	"fmt"
	"goweb/internal/pkg/logg"
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
		defer writer.Close()
		w(writer)
	}()

	return reader
}

const fieldNameBody = "Body"

// RenderPage returns a OpenAPI response of the given type,
// with the body set to the rendered component.
func RenderPage[T any](ctx context.Context, c templ.Component) (T, error) {
	var ret T
	fieldValue := reflect.ValueOf(&ret).Elem().FieldByName(fieldNameBody)
	if !fieldValue.IsValid() {
		panic(fmt.Sprintf("missing %s field in %T", fieldNameBody, ret))
	}

	body := readerFrom(func(w io.Writer) {
		err := c.Render(ctx, w)
		if err != nil {
			logg.Error(ctx, "Faile dto render template", "err", err)
		}
	})
	fieldValue.Set(reflect.ValueOf(body))

	return ret, nil
}

// RenderErrorPage returns a OpenAPI response of the given type,
// rendering the error page with the given status code.
func RenderErrorPage[T any](ctx context.Context, statusCode int) (T, error) {
	p := ErrorPage(session.GetUser(ctx), middleware.GetReqID(ctx), statusCode, http.StatusText(statusCode))
	return RenderPage[T](ctx, p)
}
