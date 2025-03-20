// Package endpoints contains HTTP endpoints implementations.
package endpoints

import (
	"context"
	"encoding/json"
	"errors"
	"goweb/internal/pkg/db"
	"goweb/internal/pkg/endpoints/tpl"
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
	p := tpl.MainPage{
		BasePage: basePage(ctx),
		Content:  "Hello, world!",
	}
	return oapi.Get200TexthtmlResponse{Body: renderPage(&p)}, nil
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

func errorHandler(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set(headerContentType, applicationJSON)
	w.WriteHeader(statusCode)

	if statusCode == http.StatusUnauthorized {
		toSend := mkErr("unauthorized", "")
		_ = json.NewEncoder(w).Encode(toSend)
		return
	}

	if message == "" {
		message = http.StatusText(statusCode)
	}

	toSend := mkErr(message, "")
	_ = json.NewEncoder(w).Encode(toSend)
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
		ErrorHandler: errorHandler,
	}
	middleware := netmiddleware.OapiRequestValidatorWithOptions(oapi.Schema(), &middlewareOptions)

	mux := chi.NewRouter()
	mux.Use(middleware)

	h := oapi.HandlerFromMux(oapi.NewStrictHandler(&sv, nil), mux)

	return h
}
