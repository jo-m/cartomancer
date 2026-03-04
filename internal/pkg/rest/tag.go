package rest

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/logg"
	"jo-m.ch/go/detour/internal/pkg/session"
)

func validateTag(tag string) bool {
	n := utf8.RuneCountInString(tag)
	return n >= 2 && n <= 32
}

func (sv *server) handleSetTrackTags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	trackUUID := chi.URLParam(r, "uuid")

	var tags []string
	if err := decodeJSON(r, &tags); err != nil {
		writeDecodeError(w, err)
		return
	}

	for _, tag := range tags {
		if !validateTag(tag) {
			writeError(w, http.StatusBadRequest, "each tag must be 2-32 characters")
			return
		}
	}

	user := session.MustGetUser(ctx)

	track, err := sv.d.QueryRO().GetTrackByUUID(ctx, trackUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "track not found")
			return
		}
		logg.Error(ctx, "failed to get track", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	if track.UserID != user.Uuid {
		writeStatusError(w, http.StatusForbidden)
		return
	}

	err = sv.d.WithTx(ctx, func(q *db.Queries) error {
		if err := q.DeleteTrackTags(ctx, trackUUID); err != nil {
			return err
		}
		for _, tag := range tags {
			t, err := q.UpsertTag(ctx, tag)
			if err != nil {
				return err
			}
			err = q.CreateTrackTag(ctx, db.CreateTrackTagParams{
				TrackID: trackUUID,
				TagID:   t.ID,
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		logg.Error(ctx, "failed to set track tags", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, trackResponseFromDB(track, tags))
}

type tagSuggestionResponse struct {
	Tags []string `json:"tags"`
}

func (sv *server) handleSuggestTags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	prefix := r.URL.Query().Get("prefix")
	if utf8.RuneCountInString(prefix) < 2 {
		writeError(w, http.StatusBadRequest, "prefix must be at least 2 characters")
		return
	}

	escaped := strings.NewReplacer("%", `\%`, "_", `\_`).Replace(prefix)
	tags, err := sv.d.QueryRO().SuggestTags(ctx, escaped+"%")
	if err != nil {
		logg.Error(ctx, "failed to suggest tags", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	if tags == nil {
		tags = []string{}
	}

	writeJSON(w, http.StatusOK, tagSuggestionResponse{Tags: tags})
}
