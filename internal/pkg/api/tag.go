package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"golang.org/x/text/unicode/norm"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/session"
)

// maxTagsPerTrack is the maximum number of tags that can be assigned to a
// single track in one request.
const maxTagsPerTrack = 50

// normalizeTag returns the NFC-normalized form of the tag so that differently
// encoded but semantically equal inputs (for example "a"+U+0308 vs. the single
// "ä" codepoint) compare and store as the same tag.
func normalizeTag(tag string) string {
	return norm.NFC.String(tag)
}

// validateTag checks that a tag is 2-32 codepoints long and contains only
// unicode letters and digits. The tag is expected to be NFC-normalized.
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

	if len(tags) > maxTagsPerTrack {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("at most %d tags allowed per track", maxTagsPerTrack))
		return
	}

	for i, tag := range tags {
		tags[i] = normalizeTag(tag)
		if !validateTag(tags[i]) {
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

// likePrefix escapes % and _ so they are treated as literals and appends the
// LIKE wildcard so the returned pattern matches "starts with prefix".
func likePrefix(prefix string) string {
	return strings.NewReplacer("%", `\%`, "_", `\_`).Replace(prefix) + "%"
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

	rows, err := sv.d.QueryRO().SuggestTags(ctx, db.SuggestTagsParams{
		UserID: user.Uuid,
		Tag:    likePrefix(r.URL.Query().Get("prefix")),
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

// handleSuggestPublicTags returns tags that appear on public tracks, optionally
// filtered by a prefix. Tags are ordered by the number of distinct public
// tracks carrying them, descending, then alphabetically. This endpoint does
// not require authentication.
func (sv *server) handleSuggestPublicTags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rows, err := sv.d.QueryRO().SuggestPublicTags(ctx, likePrefix(r.URL.Query().Get("prefix")))
	if err != nil {
		logg.Error(ctx, "failed to suggest public tags", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	out := make([]tagSuggestion, len(rows))
	for i, r := range rows {
		out[i] = tagSuggestion{Tag: r.Tag, NTracks: r.NTracks}
	}

	writeJSON(w, http.StatusOK, tagSuggestionResponse{Tags: out})
}
