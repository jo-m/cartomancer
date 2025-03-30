package oapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
)

var (
	headerContentType = http.CanonicalHeaderKey("Content-Type")
)

const (
	applicationJSON = "application/json"
	onRequest       = "request"
)

func makeErrorJSON(msg, on string, details ...ErrorJSON) ErrorJSON {
	scp := &on
	if on == "" {
		scp = nil
	}

	det := &details
	if len(details) == 0 {
		det = nil
	}

	return ErrorJSON{
		Msg:     msg,
		Details: det,
		On:      scp,
	}
}

// Returns an ErrorJSON containing the request ID, if one could be found in the context passed.
// Otherwise nil.
func makeRequestIDErrorJSON(ctx context.Context) []ErrorJSON {
	id := middleware.GetReqID(ctx)
	if id == "" {
		return nil
	}
	return []ErrorJSON{
		makeErrorJSON(id, "reqID"),
	}
}

// Create an ErrorJSON from a HTTP status code.
func makeStatusErrorJSON(ctx context.Context, statusCode int) ErrorJSON {
	text := strings.ToLower(http.StatusText(statusCode))
	return makeErrorJSON(text, onRequest, makeRequestIDErrorJSON(ctx)...)
}

// MakeJSONError returns a OpenAPI response of the given type,
// with the body set to the error JSON given by its HTTP status code.
func MakeJSONError[T any](ctx context.Context) (T, error) {
	var ret T
	typ := reflect.TypeOf(ret)

	if typ.NumField() != 1 {
		panic(fmt.Sprintf("expected exactly one field, got %d", typ.NumField()))
	}
	field := typ.Field(0)

	var embed any
	switch field.Type {
	case reflect.TypeOf(N400BadRequestJSONResponse{}):
		embed = N400BadRequestJSONResponse(makeStatusErrorJSON(ctx, 400))
	case reflect.TypeOf(N401UnauthorizedJSONResponse{}):
		embed = N401UnauthorizedJSONResponse(makeStatusErrorJSON(ctx, 401))
	case reflect.TypeOf(N404NotFoundJSONResponse{}):
		embed = N404NotFoundJSONResponse(makeStatusErrorJSON(ctx, 404))
	case reflect.TypeOf(N409ConflictJSONResponse{}):
		embed = N409ConflictJSONResponse(makeStatusErrorJSON(ctx, 409))
	case reflect.TypeOf(N500InternalServerErrorJSONResponse{}):
		embed = N500InternalServerErrorJSONResponse(makeStatusErrorJSON(ctx, 500))
	default:
		panic(fmt.Sprintf("unexpected field type %s", field.Type))
	}

	ptr := reflect.ValueOf(&ret).Elem().Field(0)
	ptr.Set(reflect.ValueOf(embed))

	return ret, nil
}

// ErrorHandler sends the error as an ErrorJSON response.
func ErrorHandler(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set(headerContentType, applicationJSON)
	w.WriteHeader(statusCode)

	if statusCode == http.StatusUnauthorized {
		toSend := makeStatusErrorJSON(context.TODO(), http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(toSend)
		return
	}

	if message == "" {
		message = strings.ToLower(http.StatusText(statusCode))
	}

	toSend := makeErrorJSON(message, onRequest)
	_ = json.NewEncoder(w).Encode(toSend)
}
