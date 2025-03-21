package oapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

var (
	headerContentType = http.CanonicalHeaderKey("Content-Type")
)

const (
	applicationJSON = "application/json"
	scopeRequest    = "request"
)

func makeErrorJSON(err, scope string, details ...ErrorJSON) ErrorJSON {
	scp := &scope
	if scope == "" {
		scp = nil
	}

	det := &details
	if len(details) == 0 {
		det = nil
	}

	return ErrorJSON{
		Error:   err,
		Details: det,
		Scope:   scp,
	}
}

func makeStatusErrorJSON(status int) ErrorJSON {
	text := strings.ToLower(http.StatusText(status))
	return makeErrorJSON(text, scopeRequest)
}

func MakeJSONError[T any]() T {
	var ret T
	typ := reflect.TypeOf(ret)

	if typ.NumField() != 1 {
		panic(fmt.Sprintf("expected exactly one field, got %d", typ.NumField()))
	}
	field := typ.Field(0)

	var embed any
	switch field.Type {
	case reflect.TypeOf(N400BadRequestJSONResponse{}):
		embed = N400BadRequestJSONResponse(makeStatusErrorJSON(400))
	case reflect.TypeOf(N401UnauthorizedJSONResponse{}):
		embed = N401UnauthorizedJSONResponse(makeStatusErrorJSON(401))
	case reflect.TypeOf(N404NotFoundJSONResponse{}):
		embed = N404NotFoundJSONResponse(makeStatusErrorJSON(404))
	case reflect.TypeOf(N409ConflictJSONResponse{}):
		embed = N409ConflictJSONResponse(makeStatusErrorJSON(409))
	case reflect.TypeOf(N500InternalServerErrorJSONResponse{}):
		embed = N500InternalServerErrorJSONResponse(makeStatusErrorJSON(500))
	default:
		panic(fmt.Sprintf("unexpected field type %s", field.Type))
	}

	ptr := reflect.ValueOf(&ret).Elem().Field(0)
	ptr.Set(reflect.ValueOf(embed))

	return ret
}

func ErrorHandler(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set(headerContentType, applicationJSON)
	w.WriteHeader(statusCode)

	if statusCode == http.StatusUnauthorized {
		toSend := makeStatusErrorJSON(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(toSend)
		return
	}

	if message == "" {
		message = strings.ToLower(http.StatusText(statusCode))
	}

	toSend := makeErrorJSON(message, scopeRequest)
	_ = json.NewEncoder(w).Encode(toSend)
}
