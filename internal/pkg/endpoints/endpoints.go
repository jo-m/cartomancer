// Package endpoints contains HTTP endpoints implementations.
package endpoints

import (
	"context"
	"errors"
	"goweb/internal/pkg/db"
	"goweb/internal/pkg/endpoints/tmpl"
	"goweb/internal/pkg/oapi"
	"goweb/internal/pkg/session"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	netmiddleware "github.com/oapi-codegen/nethttp-middleware"
)

type Server struct {
	d *db.DB
}

// Compile time interface check.
var _ oapi.StrictServerInterface = (*Server)(nil)

// Get implements oapi.StrictServerInterface.
func (s *Server) Get(ctx context.Context, request oapi.GetRequestObject) (oapi.GetResponseObject, error) {
	c := tmpl.MainPage(session.GetUser(ctx), "Hello, world!")
	return tmpl.RenderPage[oapi.Get200TexthtmlResponse](ctx, c)
}

var (
	errUnknownSecurityScheme = errors.New("unknown security scheme")
	errNotAuthenticated      = errors.New("not authenticated, no user id in session")
)

func (s *Server) authenticationFunc(ctx context.Context, a *openapi3filter.AuthenticationInput) error {
	if a.SecuritySchemeName != "CookieAuth" {
		return errUnknownSecurityScheme
	}

	sess := session.MustGet(ctx)
	if !sess.UserID.Valid {
		return errNotAuthenticated
	}

	return nil
}

func New(d *db.DB, sess *session.Store) http.Handler {
	sv := Server{
		d: d,
	}

	filterOptions := openapi3filter.Options{
		AuthenticationFunc:    sv.authenticationFunc,
		ExcludeRequestBody:    false,
		ExcludeResponseBody:   false,
		IncludeResponseStatus: true,
		MultiError:            false,
	}

	middlewareOptions := netmiddleware.Options{
		Options:      filterOptions,
		ErrorHandler: oapi.ErrorHandler,
	}
	middleware := netmiddleware.OapiRequestValidatorWithOptions(oapi.Schema(), &middlewareOptions)

	mux := chi.NewRouter()
	mux.Use(middleware)

	h := oapi.HandlerFromMux(oapi.NewStrictHandler(&sv, nil), mux)

	return h
}
