package svc

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	netmiddleware "github.com/oapi-codegen/nethttp-middleware"
)

//go:generate go tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config oapi-cfg.yaml oapi.yaml

//go:embed oapi.yaml
var schema []byte
var Schema *openapi3.T

func init() {
	var err error
	Schema, err = openapi3.NewLoader().LoadFromData(schema)
	if err != nil {
		panic(err)
	}
}

type Server struct {
	db *sql.DB
}

// Compile time interface check.
var _ StrictServerInterface = (*Server)(nil)

func authenticationFunc(ctx context.Context, a *openapi3filter.AuthenticationInput) error {
	// TODO: proper implementation and error handling
	if a.SecuritySchemeName != "CookieAuth" {
		panic("unknown security scheme")
	}

	cookie, err := a.RequestValidationInput.Request.Cookie(sessionCookieName)
	if err != nil {
		return errors.New("no session")
	}

	if cookie.Value != "asdf" {
		return errors.New("unknown session")
	}

	fmt.Println("authenticated")

	return nil
}

func customSchemaErrorFunc(err *openapi3.SchemaError) string {
	return "TODO: implement schema error func"
}

func New(db *sql.DB) http.Handler {
	filterOptions := openapi3filter.Options{
		AuthenticationFunc:    authenticationFunc,
		ExcludeRequestBody:    false,
		ExcludeResponseBody:   false,
		IncludeResponseStatus: true,
		MultiError:            true,
	}
	filterOptions.WithCustomSchemaErrorFunc(customSchemaErrorFunc)

	middlewareOptions := netmiddleware.Options{
		Options: filterOptions,
	}
	middleware := netmiddleware.OapiRequestValidatorWithOptions(Schema, &middlewareOptions)

	mux := chi.NewRouter()
	mux.Use(middleware)

	sv := Server{db: db}
	h := HandlerFromMux(NewStrictHandler(&sv, nil), mux)

	return h
}
