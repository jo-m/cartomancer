package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/session"
)

// validateTag checks that a tag is 2-32 characters and contains only
// unicode letters and digits.
func validateTag(tag string) bool {
	n := utf8.RuneCountInString(tag)
	if n < 2 || n > 32 {
		return false
	}
	for _, r := range tag {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
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
			writeError(w, http.StatusBadRequest, "each tag must be 2-32 alphanumeric characters")
			return
		}
	}

	user := session.MustGetUser(ctx)

	var errNotOwner = errors.New("not owner")

	err := sv.d.WithTx(ctx, func(q *db.Queries) error {
		t, txErr := q.GetTrackByUUID(ctx, trackUUID)
		if txErr != nil {
			return txErr
		}
		if t.UserID != user.Uuid {
			return errNotOwner
		}

		if txErr = q.DeleteTrackTags(ctx, trackUUID); txErr != nil {
			return txErr
		}
		for _, tag := range tags {
			tagRow, txErr := q.UpsertTag(ctx, db.UpsertTagParams{Tag: tag, UserID: user.Uuid})
			if txErr != nil {
				return txErr
			}
			if txErr = q.CreateTrackTag(ctx, db.CreateTrackTagParams{
				TrackID: trackUUID,
				TagID:   tagRow.ID,
			}); txErr != nil {
				return txErr
			}
		}
		return nil
	})
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "track not found")
		return
	}
	if errors.Is(err, errNotOwner) {
		writeStatusError(w, http.StatusForbidden)
		return
	}
	if err != nil {
		logg.Error(ctx, "failed to set track tags", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	// Fetch the full track view for the response. Safe to do outside the tx
	// because the current user owns the track and cannot have deleted it.
	track, fetchErr := sv.d.GetTrackByUUIDForViewer(ctx, trackUUID, user.Uuid)
	if fetchErr != nil {
		logg.Error(ctx, "failed to fetch track after tag update", "err", fetchErr)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, trackResponseFromDB(track, tags, nil, true))
}

type tagSuggestion struct {
	Tag     string `json:"tag"`
	NTracks int64  `json:"nTracks"`
}

type tagSuggestionResponse struct {
	Tags []tagSuggestion `json:"tags"`
}

// handleSuggestTags returns the authenticated user's tags, optionally filtered
// by a prefix (matched case-sensitively with LIKE). Tags are ordered by the
// number of the user's tracks that carry them, descending, then alphabetically.
// Anonymous callers receive an empty list.
func (sv *server) handleSuggestTags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := session.GetUser(ctx)
	if user == nil {
		writeJSON(w, http.StatusOK, tagSuggestionResponse{Tags: []tagSuggestion{}})
		return
	}

	prefix := r.URL.Query().Get("prefix")
	escaped := strings.NewReplacer("%", `\%`, "_", `\_`).Replace(prefix)
	rows, err := sv.d.QueryRO().SuggestTags(ctx, db.SuggestTagsParams{
		UserID: user.Uuid,
		Tag:    escaped + "%",
	})
	if err != nil {
		logg.Error(ctx, "failed to suggest tags", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	out := make([]tagSuggestion, len(rows))
	for i, r := range rows {
		out[i] = tagSuggestion{Tag: r.Tag, NTracks: r.NTracks}
	}

	writeJSON(w, http.StatusOK, tagSuggestionResponse{Tags: out})
}
