package endpoints

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"goweb/internal/pkg/api"
	"goweb/internal/pkg/endpoints/users"
	"net/http"
	"time"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	netmiddleware "github.com/oapi-codegen/nethttp-middleware"
)

type router struct {
	db *sql.DB
}

const sessionCookieName = "session"

func (rt router) hello(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "asdf",
		Quoted:   true,
		MaxAge:   1800,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	fmt.Fprint(w, "Set cookie")
}

func auth(ctx context.Context, a *openapi3filter.AuthenticationInput) error {
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

func New(db *sql.DB) http.Handler {
	rt := &router{db: db}

	mux := chi.NewRouter()
	mux.Use(middleware.RequestID)
	mux.Use(middleware.RealIP)
	mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)
	mux.Use(middleware.Timeout(60 * time.Second))

	mux.Get("/hello", rt.hello)

	users := users.New(db)
	mux.Mount("/api/v0/", users)

	oapimux := chi.NewRouter()
	options := netmiddleware.Options{
		Options: openapi3filter.Options{
			AuthenticationFunc:    auth,
			ExcludeRequestBody:    false,
			ExcludeResponseBody:   false,
			IncludeResponseStatus: true,
			MultiError:            true,
		},
	}
	oapimux.Use(netmiddleware.OapiRequestValidatorWithOptions(api.Schema, &options))
	sv := api.NewServer(db)
	h := api.HandlerFromMux(api.NewStrictHandler(&sv, nil), oapimux)
	mux.Mount("/", h)

	return mux
}
