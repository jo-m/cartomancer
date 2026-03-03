// Package rest contains the HTTP REST API.
package rest

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/jobs"
	"jo-m.ch/go/detour/internal/pkg/session"
)

type server struct {
	d            *db.DB
	sessions     *session.Store
	jobSubmitter *jobs.Submitter
}

// New creates a new API handler.
func New(d *db.DB, sessions *session.Store, submitter *jobs.Submitter) http.Handler {
	sv := server{
		d:            d,
		sessions:     sessions,
		jobSubmitter: submitter,
	}

	mux := chi.NewRouter()

	mux.Group(func(r chi.Router) {
		r.Use(sessions.Middleware)
		r.Post("/sessions/login", sv.handleLogin)
		r.Post("/sessions/logout", sv.handleLogout)
		r.Get("/sessions/me", sv.handleGetSession)
	})

	return mux
}
