// Package rest contains the HTTP REST API.
package rest

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/jobs"
	"jo-m.ch/go/detour/internal/pkg/session"
)

type server struct {
	d            *db.DB
	jobSubmitter *jobs.Submitter
}

// New creates a new API handler.
func New(d *db.DB, _ *session.Store, submitter *jobs.Submitter) http.Handler {
	sv := server{
		d:            d,
		jobSubmitter: submitter,
	}

	mux := chi.NewRouter()
	mux.Get("/status", sv.handleStatus)

	return mux
}

// TODO: Remove this again.
func (s *server) handleStatus(w http.ResponseWriter, r *http.Request) {
	count, err := s.d.QueryRO().GetSessionsCount(r.Context())
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]int64{
		"active_sessions": count,
	})
}
