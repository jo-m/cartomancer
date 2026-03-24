// Package api contains the HTTP REST API.
package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"jo-m.ch/go/detour/internal/pkg/app"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/jobs"
	"jo-m.ch/go/detour/internal/pkg/password"
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
	d              *db.DB
	sessions       *session.Store
	jobSubmitter   *jobs.Submitter
	appConfig      app.AppConfig
	emailJWTSecret []byte
}

// New creates a new API handler.
func New(d *db.DB, sessions *session.Store, submitter *jobs.Submitter, appConfig app.AppConfig) (http.Handler, error) {
	emailSecret := []byte(appConfig.EmailJWTSecret)
	if len(emailSecret) == 0 {
		emailSecret = password.GenRandBytes(emailJWTSecretLenBytes)
	}
	if len(emailSecret) != emailJWTSecretLenBytes {
		return nil, fmt.Errorf("email JWT secret must be %d bytes but is %d", emailJWTSecretLenBytes, len(emailSecret))
	}

	sv := server{
		d:              d,
		sessions:       sessions,
		jobSubmitter:   submitter,
		appConfig:      appConfig,
		emailJWTSecret: emailSecret,
	}

	mux := chi.NewRouter()
	mux.Use(rejectCORS)
	mux.Use(csrfProtect)

	mux.Get("/app_config", sv.handleGetAppConfig)
	mux.Get("/version", sv.handleGetVersion)
	mux.Get("/users/{uuid}/avatar", sv.handleGetUserAvatar)
	mux.Get("/users/{uuid}/stars", sv.handleGetUserStars)

	mux.Group(func(r chi.Router) {
		r.Post("/register", sv.handleRegister)
		r.Post("/confirm-email", sv.handleConfirmEmail)
	})

	mux.Group(func(r chi.Router) {
		r.Post("/sessions/login", sv.handleLogin)
		r.Post("/sessions/logout", sv.handleLogout)
		r.Get("/sessions", sv.handleGetSession)
	})

	// Public read endpoints: accessible without authentication.
	mux.Get("/tracks", sv.handleListTracks)
	mux.Get("/tracks/statistics", sv.handleTrackStatistics)
	mux.Get("/tracks/{uuid}", sv.handleGetTrack)
	mux.Get("/tracks/{uuid}/download", sv.handleDownloadTrackBlob)
	mux.Get("/tracks/{uuid}/preview.svg", sv.handleDownloadTrackSVG)
	mux.Get("/tracks/{uuid}/profile.svg", sv.handleDownloadTrackProfileSVG)
	mux.Get("/tracks/{uuid}/points", sv.handleGetTrackPoints)
	mux.Get("/tracks/{uuid}/road-closures", sv.handleGetTrackRoadClosures)
	mux.Post("/tracks/{uuid}/forecast", sv.handleGetTrackForecast)
	mux.Get("/tags", sv.handleSuggestTags)

	mux.Group(func(r chi.Router) {
		r.Use(sv.requireUser)

		r.Get("/tracks/groups", sv.handleListTrackGroups)
		r.Get("/tracks/groups/{uuid}", sv.handleGetTrackGroup)

		r.Post("/tracks", sv.handleUploadTrack)
		r.Post("/tracks/{uuid}/star", sv.handleStarTrack)
		r.Delete("/tracks/{uuid}/star", sv.handleUnstarTrack)
		r.Patch("/tracks", sv.handleBulkEditTracks)
		r.Get("/tracks/editing", sv.handleListTracksForEditing)
		r.Patch("/tracks/{uuid}", sv.handleEditTrack)
		r.Delete("/tracks/{uuid}", sv.handleDeleteTrack)
		r.Put("/tracks/{uuid}/tags", sv.handleSetTrackTags)
		r.Post("/tracks/editing-complete", sv.handleEditingComplete)

		r.Patch("/account", sv.handleUpdateAccount)
		r.Delete("/account", sv.handleDeleteAccount)
		r.Post("/account/change-password", sv.handleChangePassword)
		r.Post("/account/change-email", sv.handleChangeEmail)
		r.Post("/account/rotate-avatar", sv.handleRotateAvatar)
	})

	mux.Group(func(r chi.Router) {
		r.Use(sv.requireAdmin)
		r.Get("/admin/users", sv.handleAdminListUsers)
		r.Post("/admin/users", sv.handleAdminCreateUser)
		r.Get("/admin/users/{uuid}", sv.handleAdminGetUser)
		r.Patch("/admin/users/{uuid}", sv.handleAdminUpdateUser)
		r.Delete("/admin/users/{uuid}", sv.handleAdminDeleteUser)
		r.Post("/admin/users/{uuid}/reset-password", sv.handleAdminResetUserPassword)
		r.Post("/admin/users/{uuid}/confirm-email", sv.handleAdminConfirmEmail)

		r.Get("/admin/forecasts", sv.handleAdminListForecasts)
	})

	return mux, nil
}
