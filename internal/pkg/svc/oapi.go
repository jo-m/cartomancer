package svc

import (
	"context"
	"crypto/subtle"
	"database/sql"
	_ "embed"
	"errors"
	"goweb/internal/pkg/db"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	db *sql.DB
}

func (s *Server) DB() *db.Queries {
	return db.New(s.db)
}

// Compile time interface check.
var _ StrictServerInterface = (*Server)(nil)

// TODO:: move to middleware
func (s *Server) authenticationFunc(ctx context.Context, a *openapi3filter.AuthenticationInput) error {
	// TODO: proper error handling
	if a.SecuritySchemeName != "CookieAuth" {
		panic("unknown security scheme")
	}

	cookie, err := a.RequestValidationInput.Request.Cookie(sessionCookieName)
	if err != nil {
		return errors.New("no session id")
	}

	parts := strings.SplitN(cookie.Value, ":", 2)
	if len(parts) != 2 {
		return errors.New("invalid session string")
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return errors.New("invalid session id")
	}

	session, err := s.DB().GetUserSession(ctx, id)
	if err != nil {
		return errors.New("no such session")
	}

	if time.Since(session.CreatedAt) > time.Minute*30 {
		return errors.New("session expired")
	}

	if subtle.ConstantTimeCompare([]byte(parts[1]), []byte(session.Secret)) != 1 {
		return errors.New("invalid secret")
	}

	slog.Info("Authenticated", "session_id", session.ID, "user_id", session.UserID)

	return nil
}

func customSchemaErrorFunc(err *openapi3.SchemaError) string {
	return "TODO: implement schema error func"
}

func New(db *sql.DB) http.Handler {
	sv := Server{db: db}

	filterOptions := openapi3filter.Options{
		AuthenticationFunc:    sv.authenticationFunc,
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

	h := HandlerFromMux(NewStrictHandler(&sv, nil), mux)

	return h
}
