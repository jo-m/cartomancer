// Package endpoints contains HTTP endpoints implementations.
package endpoints

import (
	"context"
	"errors"
	"goweb/internal/pkg/db"
	"goweb/internal/pkg/oapi"
	"goweb/internal/pkg/session"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	netmiddleware "github.com/oapi-codegen/nethttp-middleware"
)

type Server struct {
	q *db.DB
}

// Compile time interface check.
var _ oapi.StrictServerInterface = (*Server)(nil)

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
	middleware := netmiddleware.OapiRequestValidatorWithOptions(oapi.Schema(), &middlewareOptions)

	mux := chi.NewRouter()
	mux.Use(middleware)

	h := oapi.HandlerFromMux(oapi.NewStrictHandler(&sv, nil), mux)

	return h
}
