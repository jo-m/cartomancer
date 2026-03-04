// Package rest contains the HTTP REST API.
package rest

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/jobs"
	"jo-m.ch/go/detour/internal/pkg/session"
)

// requireUser is middleware that allows only authenticated (non-admin) users.
func (sv *server) requireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if session.GetUser(r.Context()) == nil {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

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
		r.Post("/sessions/login", sv.handleLogin)
		r.Post("/sessions/logout", sv.handleLogout)
		r.Get("/sessions/me", sv.handleGetSession)
	})

	mux.Group(func(r chi.Router) {
		r.Use(sv.requireUser)

		r.Get("/tracks", sv.handleListTracks)
		r.Post("/tracks", sv.handleUploadTrack)
		r.Get("/tracks/{uuid}", sv.handleGetTrack)
		r.Patch("/tracks/{uuid}", sv.handleEditTrack)
		r.Put("/tracks/{uuid}/tags", sv.handleSetTrackTags)
		r.Get("/tracks/{uuid}/blob", sv.handleDownloadTrackBlob)

		r.Get("/tags", sv.handleSuggestTags)
	})

	mux.Group(func(r chi.Router) {
		r.Use(sv.requireAdmin)
		r.Get("/admin/users", sv.handleAdminListUsers)
		r.Post("/admin/users", sv.handleAdminCreateUser)
		r.Get("/admin/users/{uuid}", sv.handleAdminGetUser)
		r.Patch("/admin/users/{uuid}", sv.handleAdminUpdateUser)
		r.Delete("/admin/users/{uuid}", sv.handleAdminDeleteUser)
		r.Post("/admin/users/{uuid}/reset-password", sv.handleAdminResetUserPassword)
	})

	return mux
}
