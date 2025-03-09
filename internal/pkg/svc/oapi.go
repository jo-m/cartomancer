package svc

import (
	"context"
	_ "embed"
	"errors"
	"goweb/internal/pkg/db"
	"goweb/internal/pkg/session"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	netmiddleware "github.com/oapi-codegen/nethttp-middleware"
)

//go:generate go tool oapi-codegen -config oapi-cfg.yaml oapi.yaml

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
	q *db.DB
}

// Compile time interface check.
var _ StrictServerInterface = (*Server)(nil)

func (s *Server) authenticationFunc(ctx context.Context, a *openapi3filter.AuthenticationInput) error {
	if a.SecuritySchemeName != "CookieAuth" {
		return errors.New("unknown security scheme")
	}

	sess := session.MustGet(ctx)
	if !sess.UserID.Valid {
		return errors.New("not authenticated")
	}

	return nil
}

func New(q *db.DB, sess session.Store) http.Handler {
	sv := Server{
		q: q,
	}

	filterOptions := openapi3filter.Options{
		AuthenticationFunc:    sv.authenticationFunc,
		ExcludeRequestBody:    false,
		ExcludeResponseBody:   false,
		IncludeResponseStatus: true,
		MultiError:            true,
	}

	middlewareOptions := netmiddleware.Options{
		Options: filterOptions,
	}
	middleware := netmiddleware.OapiRequestValidatorWithOptions(Schema, &middlewareOptions)

	mux := chi.NewRouter()
	mux.Use(middleware)

	h := HandlerFromMux(NewStrictHandler(&sv, nil), mux)

	return h
}
